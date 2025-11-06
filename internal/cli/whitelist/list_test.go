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

func TestNewListCommand(t *testing.T) {
	cmd := NewListCommand()

	assert.Equal(t, "list", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.Contains(t, cmd.Aliases, "ls")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestRunListWhitelists_Empty(t *testing.T) {
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

	tests := []struct {
		name       string
		jsonOutput bool
		contains   []string
	}{
		{
			name:       "human output - empty",
			jsonOutput: false,
			contains:   []string{"No whitelists found"},
		},
		{
			name:       "JSON output - empty",
			jsonOutput: true,
			contains:   []string{"success", "0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := runListWhitelists(ctx, &buf, tt.jsonOutput)
			require.NoError(t, err)

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}

func TestRunListWhitelists_WithData(t *testing.T) {
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

	// Create whitelists
	whitelist1 := state.NewWhitelistState("server1")
	require.NoError(t, state.SaveWhitelistState(ctx, whitelist1))
	require.NoError(t, state.AddPlayer(ctx, "server1", state.PlayerInfo{
		UUID: "550e8400-e29b-41d4-a716-446655440000",
		Name: "TestPlayer1",
	}))

	whitelist2 := state.NewWhitelistState("server2")
	require.NoError(t, state.SaveWhitelistState(ctx, whitelist2))
	require.NoError(t, state.AddPlayer(ctx, "server2", state.PlayerInfo{
		UUID: "550e8400-e29b-41d4-a716-446655440001",
		Name: "TestPlayer2",
	}))
	require.NoError(t, state.AddPlayer(ctx, "server2", state.PlayerInfo{
		UUID: "550e8400-e29b-41d4-a716-446655440002",
		Name: "TestPlayer3",
	}))

	defer func() {
		_ = state.DeleteWhitelistState(ctx, "server1")
		_ = state.DeleteWhitelistState(ctx, "server2")
	}()

	tests := []struct {
		name       string
		jsonOutput bool
		contains   []string
	}{
		{
			name:       "human output with data",
			jsonOutput: false,
			contains:   []string{"Whitelists", "server1", "server2", "1 player", "2 players"},
		},
		{
			name:       "JSON output with data",
			jsonOutput: true,
			contains:   []string{"success", "server1", "server2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := runListWhitelists(ctx, &buf, tt.jsonOutput)
			require.NoError(t, err)

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}

func TestOutputListWhitelistsJSON(t *testing.T) {
	tests := []struct {
		name     string
		infos    []WhitelistInfo
		wantErr  bool
		checkMsg string
	}{
		{
			name:     "empty list",
			infos:    []WhitelistInfo{},
			wantErr:  false,
			checkMsg: "Found 0 whitelist",
		},
		{
			name: "single whitelist",
			infos: []WhitelistInfo{
				{Name: "test1", PlayerCount: 5},
			},
			wantErr:  false,
			checkMsg: "Found 1 whitelist",
		},
		{
			name: "multiple whitelists",
			infos: []WhitelistInfo{
				{Name: "test1", PlayerCount: 3},
				{Name: "test2", PlayerCount: 7},
			},
			wantErr:  false,
			checkMsg: "Found 2 whitelist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputListWhitelistsJSON(&buf, tt.infos)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			var output Output
			err = json.Unmarshal(buf.Bytes(), &output)
			require.NoError(t, err)

			assert.Equal(t, "success", output.Status)
			assert.Contains(t, output.Message, tt.checkMsg)

			// Check data
			data, ok := output.Data.(map[string]interface{})
			require.True(t, ok)
			require.NotNil(t, data)

			whitelists, ok := data["whitelists"].([]interface{})
			require.True(t, ok)
			assert.Len(t, whitelists, len(tt.infos))

			count, ok := data["count"].(float64)
			require.True(t, ok)
			assert.Equal(t, float64(len(tt.infos)), count)
		})
	}
}

func TestOutputListWhitelistsHuman(t *testing.T) {
	tests := []struct {
		name     string
		infos    []WhitelistInfo
		contains []string
	}{
		{
			name:     "empty list",
			infos:    []WhitelistInfo{},
			contains: []string{"No whitelists found"},
		},
		{
			name: "single whitelist with one player",
			infos: []WhitelistInfo{
				{Name: "test1", PlayerCount: 1},
			},
			contains: []string{"Whitelists (1)", "test1", "1 player"},
		},
		{
			name: "single whitelist with multiple players",
			infos: []WhitelistInfo{
				{Name: "test2", PlayerCount: 5},
			},
			contains: []string{"Whitelists (1)", "test2", "5 players"},
		},
		{
			name: "multiple whitelists",
			infos: []WhitelistInfo{
				{Name: "server1", PlayerCount: 2},
				{Name: "server2", PlayerCount: 10},
			},
			contains: []string{"Whitelists (2)", "server1", "server2", "2 players", "10 players"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputListWhitelistsHuman(&buf, tt.infos)
			require.NoError(t, err)

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}

func TestPluralS(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{0, "s"},
		{1, ""},
		{2, "s"},
		{10, "s"},
		{100, "s"},
		{-1, "s"},
	}

	for _, tt := range tests {
		t.Run("count="+string(rune(tt.count+'0')), func(t *testing.T) {
			got := pluralS(tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}
