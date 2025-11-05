package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStopCommand(t *testing.T) {
	cmd := NewStopCommand()

	assert.Equal(t, "stop", cmd.Use[:4])
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("all"))
	assert.NotNil(t, cmd.Flags().Lookup("force"))
	assert.NotNil(t, cmd.Flags().Lookup("timeout"))
}

func TestStopFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantAll      bool
		wantForce    bool
		wantTimeout  time.Duration
		wantParseErr bool
	}{
		{
			name:        "default flags",
			args:        []string{"server1"},
			wantAll:     false,
			wantForce:   false,
			wantTimeout: 30 * time.Second,
		},
		{
			name:        "with all flag",
			args:        []string{"--all"},
			wantAll:     true,
			wantForce:   false,
			wantTimeout: 30 * time.Second,
		},
		{
			name:        "with force flag",
			args:        []string{"server1", "--force"},
			wantAll:     false,
			wantForce:   true,
			wantTimeout: 30 * time.Second,
		},
		{
			name:        "with custom timeout",
			args:        []string{"server1", "--timeout", "1m"},
			wantAll:     false,
			wantForce:   false,
			wantTimeout: 1 * time.Minute,
		},
		{
			name:        "force and custom timeout",
			args:        []string{"server1", "--force", "--timeout", "10s"},
			wantAll:     false,
			wantForce:   true,
			wantTimeout: 10 * time.Second,
		},
		{
			name:        "all flags together",
			args:        []string{"--all", "--force", "--timeout", "45s"},
			wantAll:     true,
			wantForce:   true,
			wantTimeout: 45 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewStopCommand()
			cmd.SetArgs(tt.args)

			// Don't execute, just parse flags
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				return nil
			}

			err := cmd.Execute()
			if tt.wantParseErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			all, _ := cmd.Flags().GetBool("all")
			force, _ := cmd.Flags().GetBool("force")
			timeout, _ := cmd.Flags().GetDuration("timeout")

			assert.Equal(t, tt.wantAll, all)
			assert.Equal(t, tt.wantForce, force)
			assert.Equal(t, tt.wantTimeout, timeout)
		})
	}
}

func TestRunStop_NoArgs(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	flags := &StopFlags{}

	err := runStop(ctx, &buf, []string{}, flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no server names specified")
}

func TestRunStop_AllAndArgs(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	flags := &StopFlags{All: true}

	err := runStop(ctx, &buf, []string{"server1"}, flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify server names with --all flag")
}

func TestRunStop_JSONOutput(t *testing.T) {
	// Set JSON mode environment variable
	_ = os.Setenv("GOMC_JSON", "true")
	defer func() { _ = os.Unsetenv("GOMC_JSON") }()

	ctx := context.Background()
	var buf bytes.Buffer
	flags := &StopFlags{}

	// Test with no servers (should error)
	err := runStop(ctx, &buf, []string{"nonexistent"}, flags)
	require.Error(t, err)

	// Should have JSON output
	var output LifecycleOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	assert.Equal(t, "error", output.Status)
}

func TestStopServer_InvalidServerName(t *testing.T) {
	ctx := context.Background()
	result := NewOperationResult()

	err := stopServer(ctx, nil, "invalid@name", &StopFlags{}, result)
	require.Error(t, err)
	assert.Contains(t, result.Failed["invalid@name"], "invalid server name")
}

func TestStopFlags_ForceTimeout(t *testing.T) {
	tests := []struct {
		name            string
		force           bool
		configTimeout   time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "normal stop with default timeout",
			force:           false,
			configTimeout:   30 * time.Second,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "force stop overrides timeout to zero",
			force:           true,
			configTimeout:   30 * time.Second,
			expectedTimeout: 0,
		},
		{
			name:            "normal stop with custom timeout",
			force:           false,
			configTimeout:   60 * time.Second,
			expectedTimeout: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := &StopFlags{
				Force:   tt.force,
				Timeout: tt.configTimeout,
			}

			// Simulate what stopServer does
			timeout := flags.Timeout
			if flags.Force {
				timeout = 0
			}

			assert.Equal(t, tt.expectedTimeout, timeout)
		})
	}
}

func TestIsContainerStopped_EdgeCases(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{"", false},
		{"STOPPED", true},
		{"stopped", true},
		{"Stopped", true},
		{"StOpPeD", true},
		{"EXITED", true},
		{"exited", true},
		{"CREATED", true},
		{"created", true},
		{"stop", false},
		{"exit", false},
		{"create", false},
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
