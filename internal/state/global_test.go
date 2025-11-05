package state

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnv(t *testing.T) func() {
	tmpDir := t.TempDir()

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	err := InitDirs()
	require.NoError(t, err)

	return func() {
		_ = os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}
}

func TestNewGlobalState(t *testing.T) {
	state := NewGlobalState()

	require.NotNil(t, state)
	assert.Empty(t, state.AllocatedPorts)
	assert.Empty(t, state.Servers)
	assert.True(t, state.LastGCRun.IsZero())
}

func TestLoadGlobalState_CreatesIfMissing(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()
	state, err := LoadGlobalState(ctx)
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Empty(t, state.AllocatedPorts)
	assert.Empty(t, state.Servers)
}

func TestLoadGlobalState_LoadsExisting(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create initial state
	initialState := NewGlobalState()
	initialState.AllocatedPorts = []int{25565, 25566}
	initialState.Servers = []string{"survival", "creative"}

	err := SaveGlobalState(ctx, initialState)
	require.NoError(t, err)

	// Load state
	loadedState, err := LoadGlobalState(ctx)
	require.NoError(t, err)
	require.NotNil(t, loadedState)

	assert.Equal(t, []int{25565, 25566}, loadedState.AllocatedPorts)
	assert.Equal(t, []string{"survival", "creative"}, loadedState.Servers)
}

func TestLoadGlobalState_RecoversFromCorruption(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create corrupted state file
	statePath, err := GetStatePath()
	require.NoError(t, err)

	corruptedData := []byte("this is not valid YAML: {[}]")
	err = os.WriteFile(statePath, corruptedData, 0644)
	require.NoError(t, err)

	// Load state (should recover)
	state, err := LoadGlobalState(ctx)
	require.NoError(t, err)
	require.NotNil(t, state)

	// Verify backup was created
	backupPath := statePath + ".corrupted"
	_, err = os.Stat(backupPath)
	require.NoError(t, err)

	// Verify new state is empty
	assert.Empty(t, state.AllocatedPorts)
	assert.Empty(t, state.Servers)
}

func TestSaveGlobalState(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	state := NewGlobalState()
	state.AllocatedPorts = []int{25565}
	state.Servers = []string{"survival"}

	err := SaveGlobalState(ctx, state)
	require.NoError(t, err)

	// Verify file was created
	statePath, err := GetStatePath()
	require.NoError(t, err)
	_, err = os.Stat(statePath)
	require.NoError(t, err)
}

func TestSaveGlobalState_NilState(t *testing.T) {
	ctx := context.Background()
	err := SaveGlobalState(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state cannot be nil")
}

func TestAllocatePort(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Allocate port
	err := AllocatePort(ctx, 25565)
	require.NoError(t, err)

	// Verify port is allocated
	state, err := LoadGlobalState(ctx)
	require.NoError(t, err)
	assert.Contains(t, state.AllocatedPorts, 25565)
}

func TestAllocatePort_AlreadyAllocated(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Allocate port
	err := AllocatePort(ctx, 25565)
	require.NoError(t, err)

	// Try to allocate same port
	err = AllocatePort(ctx, 25565)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port 25565 is already allocated")
}

func TestAllocatePort_Invalid(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	err := AllocatePort(ctx, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
}

func TestReleasePort(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Allocate port
	err := AllocatePort(ctx, 25565)
	require.NoError(t, err)

	// Release port
	err = ReleasePort(ctx, 25565)
	require.NoError(t, err)

	// Verify port is released
	state, err := LoadGlobalState(ctx)
	require.NoError(t, err)
	assert.NotContains(t, state.AllocatedPorts, 25565)
}

func TestReleasePort_NotAllocated(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	err := ReleasePort(ctx, 25565)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port 25565 is not allocated")
}

func TestIsPortAllocated(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Initially not allocated
	allocated, err := IsPortAllocated(ctx, 25565)
	require.NoError(t, err)
	assert.False(t, allocated)

	// Allocate port
	err = AllocatePort(ctx, 25565)
	require.NoError(t, err)

	// Now allocated
	allocated, err = IsPortAllocated(ctx, 25565)
	require.NoError(t, err)
	assert.True(t, allocated)
}

func TestGetNextAvailablePort(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Allocate some ports
	err := AllocatePort(ctx, 25565)
	require.NoError(t, err)
	err = AllocatePort(ctx, 25566)
	require.NoError(t, err)

	// Get next available port
	port, err := GetNextAvailablePort(ctx, 25565)
	require.NoError(t, err)
	assert.Equal(t, 25567, port)
}

func TestGetNextAvailablePort_StartFromMiddle(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Get next available port starting from 30000
	port, err := GetNextAvailablePort(ctx, 30000)
	require.NoError(t, err)
	assert.Equal(t, 30000, port)
}

func TestRegisterServer(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	err := RegisterServer(ctx, "survival")
	require.NoError(t, err)

	// Verify server is registered
	state, err := LoadGlobalState(ctx)
	require.NoError(t, err)
	assert.Contains(t, state.Servers, "survival")
}

func TestRegisterServer_AlreadyRegistered(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	err := RegisterServer(ctx, "survival")
	require.NoError(t, err)

	// Try to register again
	err = RegisterServer(ctx, "survival")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server \"survival\" is already registered")
}

func TestRegisterServer_Invalid(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	err := RegisterServer(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server name cannot be empty")
}

func TestUnregisterServer(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Register server
	err := RegisterServer(ctx, "survival")
	require.NoError(t, err)

	// Unregister server
	err = UnregisterServer(ctx, "survival")
	require.NoError(t, err)

	// Verify server is unregistered
	state, err := LoadGlobalState(ctx)
	require.NoError(t, err)
	assert.NotContains(t, state.Servers, "survival")
}

func TestUnregisterServer_NotRegistered(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	err := UnregisterServer(ctx, "survival")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server \"survival\" is not registered")
}

func TestIsServerRegistered(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Initially not registered
	registered, err := IsServerRegistered(ctx, "survival")
	require.NoError(t, err)
	assert.False(t, registered)

	// Register server
	err = RegisterServer(ctx, "survival")
	require.NoError(t, err)

	// Now registered
	registered, err = IsServerRegistered(ctx, "survival")
	require.NoError(t, err)
	assert.True(t, registered)
}

func TestListServers(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Initially empty
	servers, err := ListServers(ctx)
	require.NoError(t, err)
	assert.Empty(t, servers)

	// Register servers
	err = RegisterServer(ctx, "survival")
	require.NoError(t, err)
	err = RegisterServer(ctx, "creative")
	require.NoError(t, err)

	// List servers
	servers, err = ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 2)
	assert.Contains(t, servers, "survival")
	assert.Contains(t, servers, "creative")
}

func TestUpdateGCTimestamp(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Load initial state
	state, err := LoadGlobalState(ctx)
	require.NoError(t, err)
	assert.True(t, state.LastGCRun.IsZero())

	// Update GC timestamp
	err = UpdateGCTimestamp(ctx)
	require.NoError(t, err)

	// Verify timestamp was updated
	state, err = LoadGlobalState(ctx)
	require.NoError(t, err)
	assert.False(t, state.LastGCRun.IsZero())
}

func TestGlobalState_ConcurrentPortAllocation(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Allocate ports concurrently
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		port := 25565 + i
		go func(p int) {
			done <- AllocatePort(ctx, p)
		}(port)
	}

	// Wait for all allocations
	for i := 0; i < 10; i++ {
		err := <-done
		require.NoError(t, err)
	}

	// Verify all ports are allocated
	state, err := LoadGlobalState(ctx)
	require.NoError(t, err)
	assert.Len(t, state.AllocatedPorts, 10)
}

func TestGlobalState_ConcurrentServerRegistration(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Register servers concurrently
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("server%d", i)
		go func(n string) {
			done <- RegisterServer(ctx, n)
		}(name)
	}

	// Wait for all registrations
	for i := 0; i < 10; i++ {
		err := <-done
		require.NoError(t, err)
	}

	// Verify all servers are registered
	state, err := LoadGlobalState(ctx)
	require.NoError(t, err)
	assert.Len(t, state.Servers, 10)
}
