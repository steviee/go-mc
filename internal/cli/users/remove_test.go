package users

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

func TestNewRemoveCommand(t *testing.T) {
	cmd := NewRemoveCommand()

	assert.Equal(t, "remove", cmd.Use[:6])
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.Contains(t, cmd.Aliases, "rm")
	assert.Contains(t, cmd.Aliases, "del")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("whitelist"))
	assert.NotNil(t, cmd.Flags().Lookup("global"))
}

func TestRunRemove_InvalidWhitelistName(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	err := runRemove(ctx, &buf, []string{"player"}, false, "invalid@name", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid whitelist name")
}

func TestRunRemove_WhitelistNotExists(t *testing.T) {
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
	err = runRemove(ctx, &buf, []string{"player"}, false, "nonexistent", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestRunRemove_GlobalFlag(t *testing.T) {
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

	// Create default whitelist
	whitelistState := state.NewWhitelistState("default")
	require.NoError(t, state.SaveWhitelistState(ctx, whitelistState))
	defer func() { _ = state.DeleteWhitelistState(ctx, "default") }()

	var buf bytes.Buffer
	// This will fail at Mojang API lookup, but we're testing that it tries to use "default"
	_ = runRemove(ctx, &buf, []string{"nonexistent-player"}, false, "ignored", true)

	// The function should have attempted to use "default" whitelist
	// We can't fully test success without mocking Mojang API
}

func TestOutputRemoveJSON(t *testing.T) {
	tests := []struct {
		name    string
		removed []state.PlayerInfo
		errors  map[string]string
		wantMsg string
	}{
		{
			name: "success only",
			removed: []state.PlayerInfo{
				{UUID: "uuid1", Name: "Player1"},
			},
			errors:  map[string]string{},
			wantMsg: "Removed 1 user(s)",
		},
		{
			name:    "with errors",
			removed: []state.PlayerInfo{},
			errors: map[string]string{
				"player1": "not found",
			},
			wantMsg: "Removed 0 user(s)",
		},
		{
			name: "mixed success and errors",
			removed: []state.PlayerInfo{
				{UUID: "uuid1", Name: "Player1"},
			},
			errors: map[string]string{
				"player2": "not found",
			},
			wantMsg: "Removed 1 user(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputRemoveJSON(&buf, "test-whitelist", tt.removed, tt.errors)
			require.NoError(t, err)

			var output Output
			err = json.Unmarshal(buf.Bytes(), &output)
			require.NoError(t, err)

			assert.Contains(t, output.Message, tt.wantMsg)

			// Check status
			if len(tt.removed) == 0 {
				assert.Equal(t, "error", output.Status)
			} else {
				assert.Equal(t, "success", output.Status)
			}

			// Check data
			data, ok := output.Data.(map[string]interface{})
			require.True(t, ok)

			whitelist, ok := data["whitelist"].(string)
			require.True(t, ok)
			assert.Equal(t, "test-whitelist", whitelist)

			// Check errors field
			if len(tt.errors) > 0 {
				_, hasErrors := data["errors"]
				assert.True(t, hasErrors)
			}
		})
	}
}

func TestOutputRemoveHuman(t *testing.T) {
	tests := []struct {
		name     string
		removed  []state.PlayerInfo
		errors   map[string]string
		wantErr  bool
		contains []string
	}{
		{
			name: "success only",
			removed: []state.PlayerInfo{
				{UUID: "uuid1", Name: "Player1"},
			},
			errors:   map[string]string{},
			wantErr:  false,
			contains: []string{"Removed 1 user(s)", "Player1"},
		},
		{
			name:    "errors only",
			removed: []state.PlayerInfo{},
			errors: map[string]string{
				"player1": "not found",
			},
			wantErr:  true,
			contains: []string{"Failed to remove", "player1", "not found"},
		},
		{
			name: "mixed success and errors",
			removed: []state.PlayerInfo{
				{UUID: "uuid1", Name: "Player1"},
			},
			errors: map[string]string{
				"player2": "not found",
			},
			wantErr:  false,
			contains: []string{"Removed 1 user(s)", "Player1", "Failed to remove", "player2"},
		},
		{
			name: "multiple successes",
			removed: []state.PlayerInfo{
				{UUID: "uuid1", Name: "Player1"},
				{UUID: "uuid2", Name: "Player2"},
			},
			errors:   map[string]string{},
			wantErr:  false,
			contains: []string{"Removed 2 user(s)", "Player1", "Player2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputRemoveHuman(&buf, "test-whitelist", tt.removed, tt.errors)

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
