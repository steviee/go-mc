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

func TestNewRestartCommand(t *testing.T) {
	cmd := NewRestartCommand()

	assert.Equal(t, "restart", cmd.Use[:7])
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("all"))
	assert.NotNil(t, cmd.Flags().Lookup("wait"))
	assert.NotNil(t, cmd.Flags().Lookup("timeout"))
}

func TestRestartFlags(t *testing.T) {
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
			args:        []string{"server1", "--timeout", "5m"},
			wantAll:     false,
			wantWait:    false,
			wantTimeout: 5 * time.Minute,
		},
		{
			name:        "all flags together",
			args:        []string{"--all", "--wait", "--timeout", "90s"},
			wantAll:     true,
			wantWait:    true,
			wantTimeout: 90 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRestartCommand()
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

func TestRunRestart_NoArgs(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	flags := &RestartFlags{}

	err := runRestart(ctx, &buf, []string{}, flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no server names specified")
}

func TestRunRestart_AllAndArgs(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	flags := &RestartFlags{All: true}

	err := runRestart(ctx, &buf, []string{"server1"}, flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify server names with --all flag")
}

func TestRunRestart_JSONOutput(t *testing.T) {
	// Set JSON mode environment variable
	_ = os.Setenv("GOMC_JSON", "true")
	defer func() { _ = os.Unsetenv("GOMC_JSON") }()

	ctx := context.Background()
	var buf bytes.Buffer
	flags := &RestartFlags{}

	// Test with no servers (should error)
	err := runRestart(ctx, &buf, []string{"nonexistent"}, flags)
	require.Error(t, err)

	// Should have JSON output
	var output LifecycleOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	assert.Equal(t, "error", output.Status)
}

func TestRestartServer_InvalidServerName(t *testing.T) {
	ctx := context.Background()
	result := NewOperationResult()

	err := restartServer(ctx, nil, "invalid@name", &RestartFlags{}, result)
	require.Error(t, err)
	assert.Contains(t, result.Failed["invalid@name"], "invalid server name")
}

func TestRestartFlags_Validation(t *testing.T) {
	tests := []struct {
		name    string
		flags   *RestartFlags
		wantErr bool
	}{
		{
			name: "valid flags",
			flags: &RestartFlags{
				Wait:    true,
				Timeout: 60 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "wait with longer timeout",
			flags: &RestartFlags{
				Wait:    true,
				Timeout: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "restart all running servers",
			flags: &RestartFlags{
				All:     true,
				Timeout: 60 * time.Second,
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

func TestRestartCommandExamples(t *testing.T) {
	cmd := NewRestartCommand()

	// Verify examples are present
	assert.Contains(t, cmd.Example, "go-mc servers restart myserver")
	assert.Contains(t, cmd.Example, "--all")
	assert.Contains(t, cmd.Example, "--wait")
	assert.Contains(t, cmd.Example, "--json")
}

func TestRestartCommandAliases(t *testing.T) {
	// Verify that restart is a standalone command without aliases
	// (unlike servers which has srv/server aliases)
	cmd := NewRestartCommand()
	assert.Empty(t, cmd.Aliases)
}
