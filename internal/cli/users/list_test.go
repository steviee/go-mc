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

func TestNewListCommand(t *testing.T) {
	cmd := NewListCommand()

	assert.Equal(t, "list", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.Contains(t, cmd.Aliases, "ls")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("whitelist"))
	assert.NotNil(t, cmd.Flags().Lookup("global"))
}

func TestRunList_Success(t *testing.T) {
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

	// Create a whitelist with players
	whitelistState := state.NewWhitelistState("test-list")
	require.NoError(t, state.SaveWhitelistState(ctx, whitelistState))
	require.NoError(t, state.AddPlayer(ctx, "test-list", state.PlayerInfo{
		UUID: "550e8400-e29b-41d4-a716-446655440000",
		Name: "Player1",
	}))
	require.NoError(t, state.AddPlayer(ctx, "test-list", state.PlayerInfo{
		UUID: "550e8400-e29b-41d4-a716-446655440001",
		Name: "Player2",
	}))

	defer func() { _ = state.DeleteWhitelistState(ctx, "test-list") }()

	tests := []struct {
		name       string
		jsonOutput bool
		contains   []string
	}{
		{
			name:       "human output",
			jsonOutput: false,
			contains:   []string{"Users in whitelist", "test-list", "Player1", "Player2"},
		},
		{
			name:       "JSON output",
			jsonOutput: true,
			contains:   []string{"success", "Player1", "Player2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := runList(ctx, &buf, tt.jsonOutput, "test-list", false)
			require.NoError(t, err)

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}

func TestRunList_Empty(t *testing.T) {
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

	// Create empty whitelist
	whitelistState := state.NewWhitelistState("empty-list")
	require.NoError(t, state.SaveWhitelistState(ctx, whitelistState))
	defer func() { _ = state.DeleteWhitelistState(ctx, "empty-list") }()

	var buf bytes.Buffer
	err = runList(ctx, &buf, false, "empty-list", false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "empty")
}

func TestRunList_GlobalFlag(t *testing.T) {
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
	err = runList(ctx, &buf, false, "ignored", true) // global=true should use "default"
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "default")
}

func TestRunList_InvalidName(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	err := runList(ctx, &buf, false, "invalid@name", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid whitelist name")
}

func TestRunList_NotExists(t *testing.T) {
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
	err = runList(ctx, &buf, false, "nonexistent", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestOutputListJSON(t *testing.T) {
	players := []state.PlayerInfo{
		{UUID: "uuid1", Name: "Player1"},
		{UUID: "uuid2", Name: "Player2"},
	}

	var buf bytes.Buffer
	err := outputListJSON(&buf, "test-whitelist", players)
	require.NoError(t, err)

	var output Output
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, "success", output.Status)
	assert.Contains(t, output.Message, "2 user(s)")

	data, ok := output.Data.(map[string]interface{})
	require.True(t, ok)

	whitelist, ok := data["whitelist"].(string)
	require.True(t, ok)
	assert.Equal(t, "test-whitelist", whitelist)

	count, ok := data["count"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(2), count)
}

func TestOutputListHuman(t *testing.T) {
	tests := []struct {
		name     string
		players  []state.PlayerInfo
		contains []string
	}{
		{
			name:     "empty list",
			players:  []state.PlayerInfo{},
			contains: []string{"empty"},
		},
		{
			name: "with players",
			players: []state.PlayerInfo{
				{UUID: "uuid1", Name: "Player1"},
				{UUID: "uuid2", Name: "Player2"},
			},
			contains: []string{"Users in whitelist", "Player1", "Player2", "uuid1", "uuid2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputListHuman(&buf, "test-whitelist", tt.players)
			require.NoError(t, err)

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}
