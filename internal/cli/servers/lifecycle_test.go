package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/steviee/go-mc/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationResult(t *testing.T) {
	t.Run("new result", func(t *testing.T) {
		result := NewOperationResult()
		assert.NotNil(t, result)
		assert.Empty(t, result.Success)
		assert.Empty(t, result.Failed)
		assert.Empty(t, result.Skipped)
		assert.False(t, result.HasFailures())
	})

	t.Run("has failures", func(t *testing.T) {
		result := NewOperationResult()
		result.Failed["server1"] = "some error"
		assert.True(t, result.HasFailures())
	})

	t.Run("total processed", func(t *testing.T) {
		result := NewOperationResult()
		result.Success = []string{"server1", "server2"}
		result.Failed = map[string]string{"server3": "error"}
		result.Skipped = []string{"server4"}
		assert.Equal(t, 4, result.TotalProcessed())
	})
}

func TestGetServerNamesFromArgs(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		args      []string
		all       bool
		wantErr   bool
		errMsg    string
		setupFunc func() error
		want      []string
	}{
		{
			name:    "explicit server names",
			args:    []string{"server1", "server2"},
			all:     false,
			wantErr: false,
			want:    []string{"server1", "server2"},
		},
		{
			name:    "no args and no all flag",
			args:    []string{},
			all:     false,
			wantErr: true,
			errMsg:  "no server names specified",
		},
		{
			name:    "both args and all flag",
			args:    []string{"server1"},
			all:     true,
			wantErr: true,
			errMsg:  "cannot specify server names with --all flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				require.NoError(t, tt.setupFunc())
			}

			got, err := getServerNamesFromArgs(ctx, tt.args, tt.all)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadServerForOperation(t *testing.T) {
	ctx := context.Background()

	// Initialize test directories
	require.NoError(t, state.InitDirs())

	tests := []struct {
		name      string
		server    string
		setupFunc func() error
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "invalid server name",
			server:  "invalid@name",
			wantErr: true,
			errMsg:  "invalid server name",
		},
		{
			name:    "server not found",
			server:  "nonexistent",
			wantErr: true,
			errMsg:  "does not exist",
		},
		{
			name:   "server without container ID",
			server: "test-no-container",
			setupFunc: func() error {
				serverState := state.NewServerState("test-no-container")
				serverState.ContainerID = "" // No container
				return state.SaveServerState(ctx, serverState)
			},
			wantErr: true,
			errMsg:  "has no container",
		},
		{
			name:   "valid server with container",
			server: "test-valid",
			setupFunc: func() error {
				serverState := state.NewServerState("test-valid")
				serverState.ContainerID = "abc123"
				return state.SaveServerState(ctx, serverState)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				require.NoError(t, tt.setupFunc())
				defer func() { _ = state.DeleteServerState(ctx, tt.server) }()
			}

			result, err := loadServerForOperation(ctx, tt.server)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.server, result.Name)
		})
	}
}

func TestOutputOperationHuman(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		result    *OperationResult
		wantErr   bool
		contains  []string
	}{
		{
			name:      "success only",
			operation: "started",
			result: &OperationResult{
				Success: []string{"server1", "server2"},
				Failed:  make(map[string]string),
				Skipped: []string{},
			},
			wantErr:  false,
			contains: []string{"Started server 'server1'", "Started server 'server2'"},
		},
		{
			name:      "with skipped",
			operation: "started",
			result: &OperationResult{
				Success: []string{"server1"},
				Failed:  make(map[string]string),
				Skipped: []string{"server2"},
			},
			wantErr:  false,
			contains: []string{"Started server 'server1'", "already running", "skipped"},
		},
		{
			name:      "with failures",
			operation: "stopped",
			result: &OperationResult{
				Success: []string{"server1"},
				Failed:  map[string]string{"server2": "container not found"},
				Skipped: []string{},
			},
			wantErr:  true,
			contains: []string{"Stopped server 'server1'", "Failed to stop", "container not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputOperationHuman(&buf, tt.operation, tt.result)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}

func TestOutputOperationJSON(t *testing.T) {
	tests := []struct {
		name       string
		operation  string
		result     *OperationResult
		wantStatus string
	}{
		{
			name:      "success only",
			operation: "started",
			result: &OperationResult{
				Success: []string{"server1"},
				Failed:  make(map[string]string),
				Skipped: []string{},
			},
			wantStatus: "success",
		},
		{
			name:      "partial success",
			operation: "stopped",
			result: &OperationResult{
				Success: []string{"server1"},
				Failed:  map[string]string{"server2": "error"},
				Skipped: []string{},
			},
			wantStatus: "partial",
		},
		{
			name:      "all failures",
			operation: "restarted",
			result: &OperationResult{
				Success: []string{},
				Failed:  map[string]string{"server1": "error"},
				Skipped: []string{},
			},
			wantStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputOperationJSON(&buf, tt.operation, tt.result)
			require.NoError(t, err)

			var output LifecycleOutput
			err = json.Unmarshal(buf.Bytes(), &output)
			require.NoError(t, err)

			assert.Equal(t, tt.wantStatus, output.Status)
			assert.NotNil(t, output.Data)
		})
	}
}

func TestGetOperationVerb(t *testing.T) {
	tests := []struct {
		operation string
		failed    bool
		want      string
	}{
		{"started", false, "Started"},
		{"started", true, "Failed to start"},
		{"stopped", false, "Stopped"},
		{"stopped", true, "Failed to stop"},
		{"restarted", false, "Restarted"},
		{"restarted", true, "Failed to restart"},
		{"unknown", false, "Processed"},
		{"unknown", true, "Failed to process"},
	}

	for _, tt := range tests {
		t.Run(tt.operation+"_"+boolStr(tt.failed), func(t *testing.T) {
			got := getOperationVerb(tt.operation, tt.failed)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetSkipReason(t *testing.T) {
	tests := []struct {
		operation string
		want      string
	}{
		{"started", "already running"},
		{"stopped", "already stopped"},
		{"restarted", "not running"},
		{"unknown", "skipped"},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			got := getSkipReason(tt.operation)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsContainerRunning(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{"running", true},
		{"Running", true},
		{"RUNNING", true},
		{"stopped", false},
		{"exited", false},
		{"created", false},
		{"paused", false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := isContainerRunning(tt.state)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsContainerStopped(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{"stopped", true},
		{"Stopped", true},
		{"exited", true},
		{"created", true},
		{"running", false},
		{"paused", false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := isContainerStopped(tt.state)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Helper function to convert bool to string for test names
func boolStr(b bool) string {
	if b {
		return "failed"
	}
	return "success"
}
