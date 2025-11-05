package state

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerState(t *testing.T) {
	state := NewServerState("survival")

	require.NotNil(t, state)
	assert.Equal(t, "survival", state.Name)
	assert.Equal(t, StatusStopped, state.Status)
	assert.Empty(t, state.Mods)
	assert.Empty(t, state.Ops)
	assert.False(t, state.CreatedAt.IsZero())
	assert.False(t, state.UpdatedAt.IsZero())
}

func TestLoadServerState(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server state
	state := NewServerState("survival")
	state.ID = "069a79f4-44e9-4726-a5be-fca90e38aaf5"
	state.Minecraft.Version = "1.20.4"
	state.Minecraft.Memory = "2G"
	state.Minecraft.JavaVersion = 21

	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Load server state
	loadedState, err := LoadServerState(ctx, "survival")
	require.NoError(t, err)
	require.NotNil(t, loadedState)

	assert.Equal(t, "survival", loadedState.Name)
	assert.Equal(t, "1.20.4", loadedState.Minecraft.Version)
	assert.Equal(t, "2G", loadedState.Minecraft.Memory)
	assert.Equal(t, 21, loadedState.Minecraft.JavaVersion)
}

func TestLoadServerState_NotExists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	_, err := LoadServerState(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestLoadServerState_Corrupted(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create corrupted server file
	serverPath, err := GetServerPath("corrupted")
	require.NoError(t, err)

	corruptedData := []byte("this is not valid YAML: {[}]")
	err = os.WriteFile(serverPath, corruptedData, 0644)
	require.NoError(t, err)

	// Load server state (should fail with backup)
	_, err = LoadServerState(ctx, "corrupted")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "corrupted and backed up")

	// Verify backup was created
	backupPath := serverPath + ".corrupted"
	_, err = os.Stat(backupPath)
	require.NoError(t, err)
}

func TestSaveServerState(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	state := NewServerState("survival")
	state.ID = "069a79f4-44e9-4726-a5be-fca90e38aaf5"
	state.Minecraft.Version = "1.20.4"
	state.Minecraft.Memory = "4G"
	state.Minecraft.JavaVersion = 21
	state.Minecraft.GamePort = 25565
	state.Minecraft.RconPort = 25575

	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Verify file was created
	serverPath, err := GetServerPath("survival")
	require.NoError(t, err)
	_, err = os.Stat(serverPath)
	require.NoError(t, err)
}

func TestSaveServerState_NilState(t *testing.T) {
	ctx := context.Background()
	err := SaveServerState(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state cannot be nil")
}

func TestDeleteServerState(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server state
	state := NewServerState("survival")
	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Delete server state
	err = DeleteServerState(ctx, "survival")
	require.NoError(t, err)

	// Verify file was deleted
	exists, err := ServerExists(ctx, "survival")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestDeleteServerState_NotExists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	err := DeleteServerState(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestServerExists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Initially doesn't exist
	exists, err := ServerExists(ctx, "survival")
	require.NoError(t, err)
	assert.False(t, exists)

	// Create server
	state := NewServerState("survival")
	err = SaveServerState(ctx, state)
	require.NoError(t, err)

	// Now exists
	exists, err = ServerExists(ctx, "survival")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestListServerStates(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Initially empty
	servers, err := ListServerStates(ctx)
	require.NoError(t, err)
	assert.Empty(t, servers)

	// Create servers
	state1 := NewServerState("survival")
	err = SaveServerState(ctx, state1)
	require.NoError(t, err)

	state2 := NewServerState("creative")
	err = SaveServerState(ctx, state2)
	require.NoError(t, err)

	// List servers
	servers, err = ListServerStates(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 2)
	assert.Contains(t, servers, "survival")
	assert.Contains(t, servers, "creative")
}

func TestValidateServerState(t *testing.T) {
	tests := []struct {
		name    string
		state   *ServerState
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid state",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.ID = "069a79f4-44e9-4726-a5be-fca90e38aaf5"
				s.Minecraft.Version = "1.20.4"
				s.Minecraft.Memory = "2G"
				s.Minecraft.JavaVersion = 21
				s.Minecraft.GamePort = 25565
				s.Minecraft.RconPort = 25575
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
			state: func() *ServerState {
				s := NewServerState("invalid_name")
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid server name",
		},
		{
			name: "invalid ID",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.ID = "not-a-uuid"
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid server ID",
		},
		{
			name: "invalid status",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.Status = "invalid"
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid server status",
		},
		{
			name: "invalid version",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.Minecraft.Version = "1.20 4"
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid Minecraft version",
		},
		{
			name: "invalid memory",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.Minecraft.Memory = "invalid"
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid memory",
		},
		{
			name: "invalid Java version",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.Minecraft.JavaVersion = 99
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid Java version",
		},
		{
			name: "invalid game port",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.Minecraft.GamePort = 70000
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid game port",
		},
		{
			name: "invalid RCON port",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.Minecraft.RconPort = 70000
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid RCON port",
		},
		{
			name: "invalid whitelist name",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.Whitelist.Lists = []string{"invalid_name"}
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid whitelist name",
		},
		{
			name: "invalid op UUID",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.Ops = []OpInfo{
					{UUID: "not-a-uuid", Name: "Notch", Level: 4},
				}
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid op UUID",
		},
		{
			name: "invalid op name",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.Ops = []OpInfo{
					{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Invalid-Name", Level: 4},
				}
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid op name",
		},
		{
			name: "invalid op level",
			state: func() *ServerState {
				s := NewServerState("survival")
				s.Ops = []OpInfo{
					{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", Level: 5},
				}
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid op level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerState(tt.state)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateServerStatus(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server
	state := NewServerState("survival")
	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Update status to running
	err = UpdateServerStatus(ctx, "survival", StatusRunning)
	require.NoError(t, err)

	// Verify status was updated
	loadedState, err := LoadServerState(ctx, "survival")
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, loadedState.Status)
	assert.False(t, loadedState.LastStarted.IsZero())

	// Update status to stopped
	err = UpdateServerStatus(ctx, "survival", StatusStopped)
	require.NoError(t, err)

	// Verify status was updated
	loadedState, err = LoadServerState(ctx, "survival")
	require.NoError(t, err)
	assert.Equal(t, StatusStopped, loadedState.Status)
	assert.False(t, loadedState.LastStopped.IsZero())
}

func TestAddMod(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server
	state := NewServerState("survival")
	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Add mod
	mod := ModInfo{
		Name:    "Fabric API",
		Slug:    "fabric-api",
		Version: "0.92.0+1.20.4",
	}
	err = AddMod(ctx, "survival", mod)
	require.NoError(t, err)

	// Verify mod was added
	loadedState, err := LoadServerState(ctx, "survival")
	require.NoError(t, err)
	assert.Len(t, loadedState.Mods, 1)
	assert.Equal(t, "fabric-api", loadedState.Mods[0].Slug)
}

func TestAddMod_AlreadyExists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server with mod
	state := NewServerState("survival")
	state.Mods = []ModInfo{
		{Name: "Fabric API", Slug: "fabric-api", Version: "0.92.0+1.20.4"},
	}
	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Try to add same mod
	mod := ModInfo{
		Name:    "Fabric API",
		Slug:    "fabric-api",
		Version: "0.92.0+1.20.4",
	}
	err = AddMod(ctx, "survival", mod)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already installed")
}

func TestRemoveMod(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server with mod
	state := NewServerState("survival")
	state.Mods = []ModInfo{
		{Name: "Fabric API", Slug: "fabric-api", Version: "0.92.0+1.20.4"},
	}
	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Remove mod
	err = RemoveMod(ctx, "survival", "fabric-api")
	require.NoError(t, err)

	// Verify mod was removed
	loadedState, err := LoadServerState(ctx, "survival")
	require.NoError(t, err)
	assert.Empty(t, loadedState.Mods)
}

func TestRemoveMod_NotInstalled(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server without mods
	state := NewServerState("survival")
	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Try to remove mod
	err = RemoveMod(ctx, "survival", "fabric-api")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestAddOp(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server
	state := NewServerState("survival")
	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Add op
	op := OpInfo{
		UUID:  "069a79f4-44e9-4726-a5be-fca90e38aaf5",
		Name:  "Notch",
		Level: 4,
	}
	err = AddOp(ctx, "survival", op)
	require.NoError(t, err)

	// Verify op was added
	loadedState, err := LoadServerState(ctx, "survival")
	require.NoError(t, err)
	assert.Len(t, loadedState.Ops, 1)
	assert.Equal(t, "Notch", loadedState.Ops[0].Name)
}

func TestAddOp_AlreadyExists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server with op
	state := NewServerState("survival")
	state.Ops = []OpInfo{
		{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", Level: 4},
	}
	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Try to add same op
	op := OpInfo{
		UUID:  "069a79f4-44e9-4726-a5be-fca90e38aaf5",
		Name:  "Notch",
		Level: 4,
	}
	err = AddOp(ctx, "survival", op)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already added")
}

func TestRemoveOp(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server with op
	state := NewServerState("survival")
	state.Ops = []OpInfo{
		{UUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5", Name: "Notch", Level: 4},
	}
	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Remove op
	err = RemoveOp(ctx, "survival", "069a79f4-44e9-4726-a5be-fca90e38aaf5")
	require.NoError(t, err)

	// Verify op was removed
	loadedState, err := LoadServerState(ctx, "survival")
	require.NoError(t, err)
	assert.Empty(t, loadedState.Ops)
}

func TestRemoveOp_NotFound(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create server without ops
	state := NewServerState("survival")
	err := SaveServerState(ctx, state)
	require.NoError(t, err)

	// Try to remove op
	err = RemoveOp(ctx, "survival", "069a79f4-44e9-4726-a5be-fca90e38aaf5")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
