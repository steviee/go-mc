package whitelist

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/steviee/go-mc/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeleteCommand(t *testing.T) {
	cmd := NewDeleteCommand()

	assert.Equal(t, "delete", cmd.Use[:6])
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.Contains(t, cmd.Aliases, "rm")
	assert.Contains(t, cmd.Aliases, "remove")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestRunDelete_Success(t *testing.T) {
	ctx := context.Background()

	// Setup test state directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config", "go-mc")
	_ = os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	require.NoError(t, os.MkdirAll(configDir, 0750))

	// Initialize global state
	_, err := state.LoadGlobalState(ctx)
	require.NoError(t, err)

	// Create a whitelist
	whitelistState := state.NewWhitelistState("test-delete")
	require.NoError(t, state.SaveWhitelistState(ctx, whitelistState))

	tests := []struct {
		name       string
		jsonOutput bool
		contains   []string
	}{
		{
			name:       "human output",
			jsonOutput: false,
			contains:   []string{"Deleted whitelist", "test-delete"},
		},
		{
			name:       "JSON output",
			jsonOutput: true,
			contains:   []string{"success", "test-delete"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Recreate whitelist for each test
			whitelistState := state.NewWhitelistState("test-delete")
			require.NoError(t, state.SaveWhitelistState(ctx, whitelistState))

			var buf bytes.Buffer
			err := runDelete(ctx, &buf, "test-delete", tt.jsonOutput)
			require.NoError(t, err)

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}

			// Verify whitelist was deleted
			exists, err := state.WhitelistExists(ctx, "test-delete")
			require.NoError(t, err)
			assert.False(t, exists)
		})
	}
}

func TestRunDelete_InvalidName(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	err := runDelete(ctx, &buf, "invalid@name", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid whitelist name")
}

func TestRunDelete_NotExists(t *testing.T) {
	ctx := context.Background()

	// Setup test state directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config", "go-mc")
	_ = os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	require.NoError(t, os.MkdirAll(configDir, 0750))

	// Initialize global state
	_, err := state.LoadGlobalState(ctx)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runDelete(ctx, &buf, "nonexistent", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestRunDelete_JSONError(t *testing.T) {
	ctx := context.Background()

	// Setup test state directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config", "go-mc")
	_ = os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	require.NoError(t, os.MkdirAll(configDir, 0750))

	// Initialize global state
	_, err := state.LoadGlobalState(ctx)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runDelete(ctx, &buf, "nonexistent", true)
	require.Error(t, err)

	// Should have JSON output even on error
	var output Output
	decodeErr := json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, decodeErr)
	assert.Equal(t, "error", output.Status)
}

func TestOutputDeleteJSON(t *testing.T) {
	var buf bytes.Buffer
	err := outputDeleteJSON(&buf, "test-whitelist")
	require.NoError(t, err)

	var output Output
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, "success", output.Status)
	assert.Contains(t, output.Message, "test-whitelist")
}

func TestOutputDeleteHuman(t *testing.T) {
	var buf bytes.Buffer
	err := outputDeleteHuman(&buf, "test-whitelist")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Deleted whitelist")
	assert.Contains(t, output, "test-whitelist")
}
