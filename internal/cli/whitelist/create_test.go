package whitelist

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/steviee/go-mc/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreateCommand(t *testing.T) {
	cmd := NewCreateCommand()

	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "create")
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestRunCreate(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "go-mc-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Set XDG_CONFIG_HOME to temp directory
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldConfigHome) }()

	tests := []struct {
		name       string
		wlName     string
		jsonOutput bool
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid whitelist name",
			wlName:     "testlist",
			jsonOutput: false,
			wantErr:    false,
		},
		{
			name:       "valid with json output",
			wlName:     "testlist2",
			jsonOutput: true,
			wantErr:    false,
		},
		{
			name:       "invalid whitelist name - special chars",
			wlName:     "invalid name!",
			jsonOutput: false,
			wantErr:    true,
			errMsg:     "invalid whitelist name",
		},
		{
			name:       "invalid whitelist name - empty",
			wlName:     "",
			jsonOutput: false,
			wantErr:    true,
			errMsg:     "invalid whitelist name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := runCreate(context.Background(), &buf, tt.wlName, tt.jsonOutput)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)

				// Verify whitelist was created
				exists, err := state.WhitelistExists(context.Background(), tt.wlName)
				require.NoError(t, err)
				assert.True(t, exists)

				// Verify output
				if tt.jsonOutput {
					var out Output
					err := json.Unmarshal(buf.Bytes(), &out)
					require.NoError(t, err)
					assert.Equal(t, "success", out.Status)
				} else {
					assert.Contains(t, buf.String(), "Created whitelist")
					assert.Contains(t, buf.String(), tt.wlName)
				}
			}
		})
	}
}

func TestRunCreate_AlreadyExists(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "go-mc-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Set XDG_CONFIG_HOME to temp directory
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldConfigHome) }()

	// Create whitelist first
	err = state.InitDirs()
	require.NoError(t, err)

	wl := state.NewWhitelistState("existing")
	err = state.SaveWhitelistState(context.Background(), wl)
	require.NoError(t, err)

	// Try to create again
	var buf bytes.Buffer
	err = runCreate(context.Background(), &buf, "existing", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestOutputCreateJSON(t *testing.T) {
	wl := state.NewWhitelistState("test")

	var buf bytes.Buffer
	err := outputCreateJSON(&buf, wl)
	require.NoError(t, err)

	var out Output
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	assert.Equal(t, "success", out.Status)
	assert.NotEmpty(t, out.Message)
	assert.NotNil(t, out.Data)
}

func TestOutputCreateHuman(t *testing.T) {
	wl := state.NewWhitelistState("test")

	var buf bytes.Buffer
	err := outputCreateHuman(&buf, wl)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Created whitelist")
	assert.Contains(t, output, "test")
	assert.Contains(t, output, "Created at")
}
