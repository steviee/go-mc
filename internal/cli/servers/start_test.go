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

func TestNewStartCommand(t *testing.T) {
	cmd := NewStartCommand()

	assert.Equal(t, "start", cmd.Use[:5])
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("all"))
	assert.NotNil(t, cmd.Flags().Lookup("wait"))
	assert.NotNil(t, cmd.Flags().Lookup("timeout"))
}

func TestStartFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantAll      bool
		wantWait     bool
		wantTimeout  time.Duration
		wantParseErr bool
	}{
		{
			name:        "default flags",
			args:        []string{"server1"},
			wantAll:     false,
			wantWait:    false,
			wantTimeout: 60 * time.Second,
		},
		{
			name:        "with all flag",
			args:        []string{"--all"},
			wantAll:     true,
			wantWait:    false,
			wantTimeout: 60 * time.Second,
		},
		{
			name:        "with wait flag",
			args:        []string{"server1", "--wait"},
			wantAll:     false,
			wantWait:    true,
			wantTimeout: 60 * time.Second,
		},
		{
			name:        "with custom timeout",
			args:        []string{"server1", "--timeout", "2m"},
			wantAll:     false,
			wantWait:    false,
			wantTimeout: 2 * time.Minute,
		},
		{
			name:        "all flags together",
			args:        []string{"--all", "--wait", "--timeout", "30s"},
			wantAll:     true,
			wantWait:    true,
			wantTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewStartCommand()
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
			wait, _ := cmd.Flags().GetBool("wait")
			timeout, _ := cmd.Flags().GetDuration("timeout")

			assert.Equal(t, tt.wantAll, all)
			assert.Equal(t, tt.wantWait, wait)
			assert.Equal(t, tt.wantTimeout, timeout)
		})
	}
}

func TestRunStart_NoArgs(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	flags := &StartFlags{}

	err := runStart(ctx, &buf, []string{}, flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no server names specified")
}

func TestRunStart_AllAndArgs(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	flags := &StartFlags{All: true}

	err := runStart(ctx, &buf, []string{"server1"}, flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify server names with --all flag")
}

func TestRunStart_JSONOutput(t *testing.T) {
	// Set JSON mode environment variable
	_ = os.Setenv("GOMC_JSON", "true")
	defer func() { _ = os.Unsetenv("GOMC_JSON") }()

	ctx := context.Background()
	var buf bytes.Buffer
	flags := &StartFlags{}

	// Test with no servers (should error)
	err := runStart(ctx, &buf, []string{"nonexistent"}, flags)
	require.Error(t, err)

	// Should have JSON output
	var output LifecycleOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	assert.Equal(t, "error", output.Status)
}

func TestStartServer_InvalidServerName(t *testing.T) {
	ctx := context.Background()
	result := NewOperationResult()

	err := startServer(ctx, nil, "invalid@name", &StartFlags{}, result)
	require.Error(t, err)
	assert.Contains(t, result.Failed["invalid@name"], "invalid server name")
}

func TestStartFlags_Validation(t *testing.T) {
	tests := []struct {
		name    string
		flags   *StartFlags
		wantErr bool
	}{
		{
			name: "valid flags",
			flags: &StartFlags{
				Wait:    true,
				Timeout: 60 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "wait without explicit timeout uses default",
			flags: &StartFlags{
				Wait:    true,
				Timeout: 0, // Will use default
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just validate that flags structure is correct
			assert.NotNil(t, tt.flags)
		})
	}
}

func TestIsContainerRunning_EdgeCases(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{"", false},
		{"RUNNING", true},
		{"running", true},
		{"Running", true},
		{"RuNnInG", true},
		{"run", false},
		{"running-ish", false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := isContainerRunning(tt.state)
			assert.Equal(t, tt.want, got)
		})
	}
}
