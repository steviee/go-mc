package state

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWhitelistState(t *testing.T) {
	state := NewWhitelistState("default")

	require.NotNil(t, state)
	assert.Equal(t, "default", state.Name)
	assert.Empty(t, state.Players)
	assert.False(t, state.CreatedAt.IsZero())
	assert.False(t, state.UpdatedAt.IsZero())
}

func TestLoadWhitelistState(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create whitelist state
	state := NewWhitelistState("default")
	state.Players = []PlayerInfo{
		{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", AddedAt: time.Now()},
	}

	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Load whitelist state
	loadedState, err := LoadWhitelistState(ctx, "default")
	require.NoError(t, err)
	require.NotNil(t, loadedState)

	assert.Equal(t, "default", loadedState.Name)
	assert.Len(t, loadedState.Players, 1)
	assert.Equal(t, "Notch", loadedState.Players[0].Name)
}

func TestLoadWhitelistState_NotExists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	_, err := LoadWhitelistState(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestLoadWhitelistState_Corrupted(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create corrupted whitelist file
	whitelistPath, err := GetWhitelistPath("corrupted")
	require.NoError(t, err)

	corruptedData := []byte("this is not valid YAML: {[}]")
	err = os.WriteFile(whitelistPath, corruptedData, 0644)
	require.NoError(t, err)

	// Load whitelist state (should fail with backup)
	_, err = LoadWhitelistState(ctx, "corrupted")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "corrupted and backed up")

	// Verify backup was created
	backupPath := whitelistPath + ".corrupted"
	_, err = os.Stat(backupPath)
	require.NoError(t, err)
}

func TestSaveWhitelistState(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	state := NewWhitelistState("default")
	state.Players = []PlayerInfo{
		{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", AddedAt: time.Now()},
	}

	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Verify file was created
	whitelistPath, err := GetWhitelistPath("default")
	require.NoError(t, err)
	_, err = os.Stat(whitelistPath)
	require.NoError(t, err)
}

func TestSaveWhitelistState_NilState(t *testing.T) {
	ctx := context.Background()
	err := SaveWhitelistState(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state cannot be nil")
}

func TestDeleteWhitelistState(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create whitelist state
	state := NewWhitelistState("default")
	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Delete whitelist state
	err = DeleteWhitelistState(ctx, "default")
	require.NoError(t, err)

	// Verify file was deleted
	exists, err := WhitelistExists(ctx, "default")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestDeleteWhitelistState_NotExists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	err := DeleteWhitelistState(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestWhitelistExists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Initially doesn't exist
	exists, err := WhitelistExists(ctx, "default")
	require.NoError(t, err)
	assert.False(t, exists)

	// Create whitelist
	state := NewWhitelistState("default")
	err = SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Now exists
	exists, err = WhitelistExists(ctx, "default")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestListWhitelistStates(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Initially empty
	whitelists, err := ListWhitelistStates(ctx)
	require.NoError(t, err)
	assert.Empty(t, whitelists)

	// Create whitelists
	state1 := NewWhitelistState("default")
	err = SaveWhitelistState(ctx, state1)
	require.NoError(t, err)

	state2 := NewWhitelistState("vip")
	err = SaveWhitelistState(ctx, state2)
	require.NoError(t, err)

	// List whitelists
	whitelists, err = ListWhitelistStates(ctx)
	require.NoError(t, err)
	assert.Len(t, whitelists, 2)
	assert.Contains(t, whitelists, "default")
	assert.Contains(t, whitelists, "vip")
}

func TestValidateWhitelistState(t *testing.T) {
	tests := []struct {
		name    string
		state   *WhitelistState
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid state",
			state: func() *WhitelistState {
				s := NewWhitelistState("default")
				s.Players = []PlayerInfo{
					{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", AddedAt: time.Now()},
				}
				return s
			}(),
			wantErr: false,
		},
		{
			name:    "nil state",
			state:   nil,
			wantErr: true,
			errMsg:  "state cannot be nil",
		},
		{
			name: "invalid name",
			state: func() *WhitelistState {
				s := NewWhitelistState("invalid_name")
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid whitelist name",
		},
		{
			name: "invalid player UUID",
			state: func() *WhitelistState {
				s := NewWhitelistState("default")
				s.Players = []PlayerInfo{
					{UUID: "not-a-uuid", Name: "Notch", AddedAt: time.Now()},
				}
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid player UUID",
		},
		{
			name: "invalid player name",
			state: func() *WhitelistState {
				s := NewWhitelistState("default")
				s.Players = []PlayerInfo{
					{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Invalid-Name", AddedAt: time.Now()},
				}
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid player name",
		},
		{
			name: "duplicate player UUID",
			state: func() *WhitelistState {
				s := NewWhitelistState("default")
				s.Players = []PlayerInfo{
					{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", AddedAt: time.Now()},
					{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch2", AddedAt: time.Now()},
				}
				return s
			}(),
			wantErr: true,
			errMsg:  "duplicate player UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWhitelistState(tt.state)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAddPlayer(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create whitelist
	state := NewWhitelistState("default")
	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Add player
	player := PlayerInfo{
		UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5",
		Name: "Notch",
	}
	err = AddPlayer(ctx, "default", player)
	require.NoError(t, err)

	// Verify player was added
	loadedState, err := LoadWhitelistState(ctx, "default")
	require.NoError(t, err)
	assert.Len(t, loadedState.Players, 1)
	assert.Equal(t, "Notch", loadedState.Players[0].Name)
	assert.False(t, loadedState.Players[0].AddedAt.IsZero())
}

func TestAddPlayer_CreatesWhitelistIfNotExists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Add player to non-existing whitelist
	player := PlayerInfo{
		UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5",
		Name: "Notch",
	}
	err := AddPlayer(ctx, "newlist", player)
	require.NoError(t, err)

	// Verify whitelist was created
	exists, err := WhitelistExists(ctx, "newlist")
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify player was added
	loadedState, err := LoadWhitelistState(ctx, "newlist")
	require.NoError(t, err)
	assert.Len(t, loadedState.Players, 1)
	assert.Equal(t, "Notch", loadedState.Players[0].Name)
}

func TestAddPlayer_AlreadyExists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create whitelist with player
	state := NewWhitelistState("default")
	state.Players = []PlayerInfo{
		{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", AddedAt: time.Now()},
	}
	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Try to add same player
	player := PlayerInfo{
		UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5",
		Name: "Notch",
	}
	err = AddPlayer(ctx, "default", player)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already in whitelist")
}

func TestRemovePlayer(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create whitelist with player
	state := NewWhitelistState("default")
	state.Players = []PlayerInfo{
		{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", AddedAt: time.Now()},
	}
	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Remove player
	err = RemovePlayer(ctx, "default", "069a79f4-44e9-4726-a5be-fca90e38aaf5")
	require.NoError(t, err)

	// Verify player was removed
	loadedState, err := LoadWhitelistState(ctx, "default")
	require.NoError(t, err)
	assert.Empty(t, loadedState.Players)
}

func TestRemovePlayer_NotInWhitelist(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create empty whitelist
	state := NewWhitelistState("default")
	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Try to remove player
	err = RemovePlayer(ctx, "default", "069a79f4-44e9-4726-a5be-fca90e38aaf5")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in whitelist")
}

func TestIsPlayerInWhitelist(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create whitelist with player
	state := NewWhitelistState("default")
	state.Players = []PlayerInfo{
		{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", AddedAt: time.Now()},
	}
	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Check if player is in whitelist
	inList, err := IsPlayerInWhitelist(ctx, "default", "069a79f4-44e9-4726-a5be-fca90e38aaf5")
	require.NoError(t, err)
	assert.True(t, inList)

	// Check if another player is in whitelist
	inList, err = IsPlayerInWhitelist(ctx, "default", "61699b2e-d327-4a01-9f1e-0ea8c3f06bc6")
	require.NoError(t, err)
	assert.False(t, inList)
}

func TestGetPlayer(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create whitelist with player
	state := NewWhitelistState("default")
	state.Players = []PlayerInfo{
		{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", AddedAt: time.Now()},
	}
	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Get player
	player, err := GetPlayer(ctx, "default", "069a79f4-44e9-4726-a5be-fca90e38aaf5")
	require.NoError(t, err)
	require.NotNil(t, player)
	assert.Equal(t, "Notch", player.Name)
}

func TestGetPlayer_NotInWhitelist(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create empty whitelist
	state := NewWhitelistState("default")
	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// Try to get player
	_, err = GetPlayer(ctx, "default", "069a79f4-44e9-4726-a5be-fca90e38aaf5")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in whitelist")
}

func TestListPlayers(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create whitelist with players
	state := NewWhitelistState("default")
	state.Players = []PlayerInfo{
		{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", AddedAt: time.Now()},
		{UUID: "61699b2e-d327-4a01-9f1e-0ea8c3f06bc6", Name: "Dinnerbone", AddedAt: time.Now()},
	}
	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// List players
	players, err := ListPlayers(ctx, "default")
	require.NoError(t, err)
	assert.Len(t, players, 2)

	// Verify player names
	names := []string{players[0].Name, players[1].Name}
	assert.Contains(t, names, "Notch")
	assert.Contains(t, names, "Dinnerbone")
}

func TestListPlayers_EmptyWhitelist(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create empty whitelist
	state := NewWhitelistState("default")
	err := SaveWhitelistState(ctx, state)
	require.NoError(t, err)

	// List players
	players, err := ListPlayers(ctx, "default")
	require.NoError(t, err)
	assert.Empty(t, players)
}
