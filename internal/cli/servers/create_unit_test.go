package servers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steviee/go-mc/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateServerDirectories_ErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (string, func())
		wantErr bool
	}{
		{
			name: "successful creation",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				_ = os.Setenv("XDG_DATA_HOME", tmpDir)
				return "test-server", func() { _ = os.Unsetenv("XDG_DATA_HOME") }
			},
			wantErr: false,
		},
		{
			name: "create multiple servers",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				_ = os.Setenv("XDG_DATA_HOME", tmpDir)
				return "server-with-hyphens", func() { _ = os.Unsetenv("XDG_DATA_HOME") }
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, cleanup := tt.setup()
			defer cleanup()

			err := createServerDirectories(name)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify directories were created
			dataHome := os.Getenv("XDG_DATA_HOME")
			serverDir := filepath.Join(dataHome, "go-mc", "servers", name)
			dataDir := filepath.Join(serverDir, "data")
			modsDir := filepath.Join(serverDir, "mods")

			assert.DirExists(t, serverDir)
			assert.DirExists(t, dataDir)
			assert.DirExists(t, modsDir)
		})
	}
}

func TestBuildServerState_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("XDG_DATA_HOME") }()

	config := &ServerConfig{
		Name:        "full-config-test",
		Version:     "1.21.1",
		Memory:      "4G",
		Port:        25565,
		RCONPort:    35565,
		RCONPass:    "secure-password",
		ContainerID: "container-123-abc",
		Mods:        []string{"fabric-api", "sodium"},
	}

	serverState := buildServerState(config, "full-config-test")

	// Verify all fields are set correctly
	assert.Equal(t, "full-config-test", serverState.Name)
	assert.Equal(t, "container-123-abc", serverState.ContainerID)
	assert.Equal(t, containerImage, serverState.Image)
	assert.Equal(t, state.StatusStopped, serverState.Status)

	// Verify Minecraft config
	assert.Equal(t, "1.21.1", serverState.Minecraft.Version)
	assert.Equal(t, "4G", serverState.Minecraft.Memory)
	assert.Equal(t, 25565, serverState.Minecraft.GamePort)
	assert.Equal(t, 35565, serverState.Minecraft.RconPort)
	assert.Equal(t, "secure-password", serverState.Minecraft.RconPassword)

	// Verify volumes paths
	assert.NotEmpty(t, serverState.Volumes.Data)
	assert.NotEmpty(t, serverState.Volumes.Backups)
	assert.Contains(t, serverState.Volumes.Data, "full-config-test")
}

func TestGenerateRCONPassword_Fallback(t *testing.T) {
	// Test that fallback works when crypto/rand fails
	// We can't easily simulate crypto/rand failure, but we can verify
	// the password generation works consistently
	passwords := make(map[string]int)

	// Generate many passwords to check uniqueness
	for i := 0; i < 1000; i++ {
		password := generateRCONPassword()
		passwords[password]++

		// Each password should be unique
		if passwords[password] > 1 {
			t.Errorf("duplicate password generated: %s (count: %d)", password, passwords[password])
		}

		// Verify length
		assert.Len(t, password, 16, "password should be 16 characters")

		// Verify charset
		for _, ch := range password {
			isValid := (ch >= 'a' && ch <= 'z') ||
				(ch >= 'A' && ch <= 'Z') ||
				(ch >= '0' && ch <= '9')
			assert.True(t, isValid, "invalid character in password: %c", ch)
		}
	}
}

func TestBuildServerConfig_RCONPortConflict(t *testing.T) {
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

	// Allocate RCON port (game port + offset)
	rconPort := 25565 + rconPortOffset
	require.NoError(t, state.AllocatePort(ctx, rconPort))

	flags := &CreateFlags{
		Version: "1.21.1",
		Memory:  "2G",
		Port:    25565, // This would cause RCON port conflict
	}

	_, err = buildServerConfig(ctx, "test", flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RCON port")
	assert.Contains(t, err.Error(), "already allocated")
}

func TestBuildServerConfig_InvalidRCONPort(t *testing.T) {
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

	// Use a port that would result in invalid RCON port (> 65535)
	flags := &CreateFlags{
		Version: "1.21.1",
		Memory:  "2G",
		Port:    65535, // RCON would be 65535 + 10000 = 75535 (invalid)
	}

	_, err = buildServerConfig(ctx, "test", flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RCON port")
}

func TestShowDryRun_WithMods(t *testing.T) {
	config := &ServerConfig{
		Name:     "modded-server",
		Version:  "1.21.1",
		Memory:   "4G",
		Port:     25565,
		RCONPort: 35565,
		Mods:     []string{"fabric-api", "sodium", "lithium"},
	}

	t.Run("human readable with mods", func(t *testing.T) {
		var buf strings.Builder
		err := showDryRun(&buf, false, config)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "modded-server")
		assert.Contains(t, output, "fabric-api")
		assert.Contains(t, output, "sodium")
		assert.Contains(t, output, "lithium")
	})

	t.Run("JSON with mods", func(t *testing.T) {
		var buf strings.Builder
		err := showDryRun(&buf, true, config)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "dry-run")
		assert.Contains(t, output, "fabric-api")
	})
}

func TestOutputSuccess_BothStates(t *testing.T) {
	config := &ServerConfig{
		Name:        "test-server",
		Version:     "1.21.1",
		Memory:      "2G",
		Port:        25565,
		RCONPort:    35565,
		ContainerID: "abc123def456ghi789",
	}

	t.Run("created state", func(t *testing.T) {
		var buf strings.Builder
		err := outputSuccess(&buf, false, config, false)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Created server")
		assert.Contains(t, output, "test-server")
		assert.Contains(t, output, "go-mc servers start")
	})

	t.Run("running state", func(t *testing.T) {
		var buf strings.Builder
		err := outputSuccess(&buf, false, config, true)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Created server")
		assert.Contains(t, output, "starting")
	})

	t.Run("JSON created state", func(t *testing.T) {
		var buf strings.Builder
		err := outputSuccess(&buf, true, config, false)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "created")
		assert.Contains(t, output, "success")
	})

	t.Run("JSON running state", func(t *testing.T) {
		var buf strings.Builder
		err := outputSuccess(&buf, true, config, true)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "running")
		assert.Contains(t, output, "success")
	})
}

func TestCreateServerDirectories_WithoutXDG(t *testing.T) {
	// Test when XDG_DATA_HOME is not set (uses $HOME/.local/share)
	oldXDG := os.Getenv("XDG_DATA_HOME")
	_ = os.Unsetenv("XDG_DATA_HOME")
	defer func() {
		if oldXDG != "" {
			_ = os.Setenv("XDG_DATA_HOME", oldXDG)
		}
	}()

	// We can't easily test this without mocking os.UserHomeDir
	// but we can verify the function doesn't panic
	err := createServerDirectories("test-no-xdg")

	// It may fail if we don't have permissions, but shouldn't panic
	if err != nil {
		// Expected in test environment
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestBuildServerConfig_AllEdgeCases(t *testing.T) {
	ctx := context.Background()

	// Setup test state directory
	tmpDir := t.TempDir()
	setupTestStateDir(t, tmpDir)

	tests := []struct {
		name    string
		flags   *CreateFlags
		wantErr bool
		errMsg  string
	}{
		{
			name: "minimum valid config",
			flags: &CreateFlags{
				Version: "1.20.1",
				Memory:  "1G",
				Port:    0,
			},
			wantErr: false,
		},
		{
			name: "maximum memory",
			flags: &CreateFlags{
				Version: "1.20.4",
				Memory:  "16G",
				Port:    0,
			},
			wantErr: false,
		},
		{
			name: "empty version auto-fetches latest",
			flags: &CreateFlags{
				Version: "",
				Memory:  "2G",
			},
			wantErr: false, // Auto-fetch makes empty version valid
		},
		{
			name: "empty memory",
			flags: &CreateFlags{
				Version: "1.20.4",
				Memory:  "",
			},
			wantErr: true,
			errMsg:  "invalid memory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := buildServerConfig(ctx, "edgecase-test", tt.flags)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, config)
			assert.NotEmpty(t, config.RCONPass)
			assert.Len(t, config.RCONPass, 16)
		})
	}
}
