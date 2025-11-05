//go:build integration

package servers

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup test environment
	tmpDir := t.TempDir()
	setupTestEnvironment(t, tmpDir)

	ctx := context.Background()

	// Test container client availability
	client, err := container.NewClient(ctx, container.DefaultConfig())
	if err != nil {
		t.Skipf("container runtime not available: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name       string
		serverName string
		flags      *CreateFlags
		wantErr    bool
		cleanup    bool
	}{
		{
			name:       "create basic server",
			serverName: "test-basic",
			flags: &CreateFlags{
				Version: "1.21.1",
				Memory:  "1G",
				Port:    0,
				Start:   false,
				DryRun:  false,
			},
			wantErr: false,
			cleanup: true,
		},
		{
			name:       "create with explicit port",
			serverName: "test-port",
			flags: &CreateFlags{
				Version: "1.21.1",
				Memory:  "1G",
				Port:    25570,
				Start:   false,
				DryRun:  false,
			},
			wantErr: false,
			cleanup: true,
		},
		{
			name:       "dry run does not create",
			serverName: "test-dryrun",
			flags: &CreateFlags{
				Version: "1.21.1",
				Memory:  "1G",
				Port:    0,
				Start:   false,
				DryRun:  true,
			},
			wantErr: false,
			cleanup: false,
		},
		{
			name:       "duplicate name fails",
			serverName: "test-duplicate",
			flags: &CreateFlags{
				Version: "1.21.1",
				Memory:  "1G",
				Port:    0,
				Start:   false,
				DryRun:  false,
			},
			wantErr: false,
			cleanup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			// Run create command
			err := runCreate(ctx, &stdout, &stderr, tt.serverName, tt.flags)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err, "stdout: %s\nstderr: %s", stdout.String(), stderr.String())

			// For dry-run, verify nothing was created
			if tt.flags.DryRun {
				exists, err := state.ServerExists(ctx, tt.serverName)
				require.NoError(t, err)
				assert.False(t, exists, "dry-run should not create server")
				return
			}

			// Verify server state was created
			exists, err := state.ServerExists(ctx, tt.serverName)
			require.NoError(t, err)
			assert.True(t, exists, "server state should exist")

			// Load and verify server state
			serverState, err := state.LoadServerState(ctx, tt.serverName)
			require.NoError(t, err)
			assert.Equal(t, tt.serverName, serverState.Name)
			assert.Equal(t, tt.flags.Version, serverState.Minecraft.Version)
			assert.Equal(t, tt.flags.Memory, serverState.Minecraft.Memory)
			assert.NotEmpty(t, serverState.ContainerID)
			assert.NotEmpty(t, serverState.Minecraft.RconPassword)

			// Verify port allocation
			if tt.flags.Port != 0 {
				assert.Equal(t, tt.flags.Port, serverState.Minecraft.GamePort)
			} else {
				assert.GreaterOrEqual(t, serverState.Minecraft.GamePort, defaultStartPort)
			}

			allocated, err := state.IsPortAllocated(ctx, serverState.Minecraft.GamePort)
			require.NoError(t, err)
			assert.True(t, allocated, "game port should be allocated")

			allocated, err = state.IsPortAllocated(ctx, serverState.Minecraft.RconPort)
			require.NoError(t, err)
			assert.True(t, allocated, "RCON port should be allocated")

			// Verify container exists
			info, err := client.InspectContainer(ctx, serverState.ContainerID)
			require.NoError(t, err)
			assert.Equal(t, tt.serverName, info.Name)
			assert.Contains(t, info.Image, "minecraft-server")

			// Verify directories exist
			dataHome := os.Getenv("XDG_DATA_HOME")
			if dataHome == "" {
				homeDir, _ := os.UserHomeDir()
				dataHome = filepath.Join(homeDir, ".local", "share")
			}
			dataDir := filepath.Join(dataHome, "go-mc", "servers", tt.serverName, "data")
			assert.DirExists(t, dataDir)

			// Cleanup
			if tt.cleanup {
				// Remove container
				err = client.RemoveContainer(ctx, serverState.ContainerID, &container.RemoveOptions{
					Force:         true,
					RemoveVolumes: true,
				})
				require.NoError(t, err)

				// Release ports
				_ = state.ReleasePort(ctx, serverState.Minecraft.GamePort)
				_ = state.ReleasePort(ctx, serverState.Minecraft.RconPort)

				// Unregister server
				_ = state.UnregisterServer(ctx, tt.serverName)

				// Delete state
				_ = state.DeleteServerState(ctx, tt.serverName)
			}
		})
	}

	// Test duplicate server creation
	t.Run("duplicate server fails", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		serverName := "test-dup-check"

		flags := &CreateFlags{
			Version: "1.21.1",
			Memory:  "1G",
			Port:    0,
			Start:   false,
			DryRun:  false,
		}

		// Create first server
		err := runCreate(ctx, &stdout, &stderr, serverName, flags)
		require.NoError(t, err)

		// Load state for cleanup
		serverState, err := state.LoadServerState(ctx, serverName)
		require.NoError(t, err)

		// Try to create duplicate
		stdout.Reset()
		stderr.Reset()
		err = runCreate(ctx, &stdout, &stderr, serverName, flags)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")

		// Cleanup
		_ = client.RemoveContainer(ctx, serverState.ContainerID, &container.RemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		})
		_ = state.ReleasePort(ctx, serverState.Minecraft.GamePort)
		_ = state.ReleasePort(ctx, serverState.Minecraft.RconPort)
		_ = state.UnregisterServer(ctx, serverName)
		_ = state.DeleteServerState(ctx, serverName)
	})
}

func TestCreateCommand_StartFlag_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup test environment
	tmpDir := t.TempDir()
	setupTestEnvironment(t, tmpDir)

	ctx := context.Background()

	// Test container client availability
	client, err := container.NewClient(ctx, container.DefaultConfig())
	if err != nil {
		t.Skipf("container runtime not available: %v", err)
	}
	defer client.Close()

	serverName := "test-start-flag"
	var stdout, stderr bytes.Buffer

	flags := &CreateFlags{
		Version: "1.21.1",
		Memory:  "1G",
		Port:    0,
		Start:   true, // Start immediately
		DryRun:  false,
	}

	// Create and start server
	err = runCreate(ctx, &stdout, &stderr, serverName, flags)
	require.NoError(t, err, "stdout: %s\nstderr: %s", stdout.String(), stderr.String())

	// Load server state
	serverState, err := state.LoadServerState(ctx, serverName)
	require.NoError(t, err)

	// Note: The container might not be fully running yet, but it should be starting
	// We'll just verify the container exists and was told to start
	info, err := client.InspectContainer(ctx, serverState.ContainerID)
	require.NoError(t, err)
	// State should be "running", "starting", or similar (not "created" or "stopped")
	assert.NotEqual(t, "created", info.State)

	// Cleanup
	_ = client.StopContainer(ctx, serverState.ContainerID, nil)
	_ = client.RemoveContainer(ctx, serverState.ContainerID, &container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	_ = state.ReleasePort(ctx, serverState.Minecraft.GamePort)
	_ = state.ReleasePort(ctx, serverState.Minecraft.RconPort)
	_ = state.UnregisterServer(ctx, serverName)
	_ = state.DeleteServerState(ctx, serverName)
}

func setupTestEnvironment(t *testing.T, tmpDir string) {
	t.Helper()

	// Set environment variables
	configDir := filepath.Join(tmpDir, "config")
	dataDir := filepath.Join(tmpDir, "data")

	os.Setenv("XDG_CONFIG_HOME", configDir)
	os.Setenv("XDG_DATA_HOME", dataDir)

	// Create directories
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "go-mc", "servers"), 0750))
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "go-mc", "servers"), 0750))

	// Initialize state
	ctx := context.Background()
	_, err := state.LoadGlobalState(ctx)
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		os.Unsetenv("XDG_CONFIG_HOME")
		os.Unsetenv("XDG_DATA_HOME")
	})
}
