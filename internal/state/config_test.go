package state

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, "podman", cfg.Container.Runtime)
	assert.Equal(t, "2G", cfg.Defaults.Memory)
	assert.Equal(t, 21, cfg.Defaults.JavaVersion)
	assert.Equal(t, 25565, cfg.Ports.GamePortStart)
	assert.Equal(t, 25575, cfg.Ports.RconPortStart)
	assert.Equal(t, 16, cfg.Ports.RconPasswordLength)
	assert.Equal(t, true, cfg.Backups.Compress)
	assert.Equal(t, 5, cfg.Backups.KeepCount)
	assert.Equal(t, 1*time.Second, cfg.TUI.RefreshInterval)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, 50, cfg.Limits.MaxServers)
}

func TestLoadConfig_CreatesDefaultIfMissing(t *testing.T) {
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	// Initialize directories
	err := InitDirs()
	require.NoError(t, err)

	ctx := context.Background()
	cfg, err := LoadConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify config was created with defaults
	assert.Equal(t, "podman", cfg.Container.Runtime)
	assert.Equal(t, "2G", cfg.Defaults.Memory)

	// Verify config file was created
	configPath, err := GetConfigPath()
	require.NoError(t, err)
	_, err = os.Stat(configPath)
	require.NoError(t, err)
}

func TestLoadConfig_LoadsExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	// Initialize directories
	err := InitDirs()
	require.NoError(t, err)

	ctx := context.Background()

	// Create custom config
	customCfg := DefaultConfig()
	customCfg.Defaults.Memory = "4G"
	customCfg.Ports.GamePortStart = 30000

	err = SaveConfig(ctx, customCfg)
	require.NoError(t, err)

	// Load config
	loadedCfg, err := LoadConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, loadedCfg)

	// Verify custom values
	assert.Equal(t, "4G", loadedCfg.Defaults.Memory)
	assert.Equal(t, 30000, loadedCfg.Ports.GamePortStart)
}

func TestLoadConfig_RecorversFromCorruption(t *testing.T) {
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	// Initialize directories
	err := InitDirs()
	require.NoError(t, err)

	ctx := context.Background()

	// Create corrupted config file
	configPath, err := GetConfigPath()
	require.NoError(t, err)

	corruptedData := []byte("this is not valid YAML: {[}]")
	err = os.WriteFile(configPath, corruptedData, 0644)
	require.NoError(t, err)

	// Load config (should recover)
	cfg, err := LoadConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify backup was created
	backupPath := configPath + ".corrupted"
	_, err = os.Stat(backupPath)
	require.NoError(t, err)

	// Verify new config has defaults
	assert.Equal(t, "podman", cfg.Container.Runtime)
	assert.Equal(t, "2G", cfg.Defaults.Memory)
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	// Initialize directories
	err := InitDirs()
	require.NoError(t, err)

	ctx := context.Background()

	// Create config
	cfg := DefaultConfig()
	cfg.Defaults.Memory = "8G"

	err = SaveConfig(ctx, cfg)
	require.NoError(t, err)

	// Verify file was created
	configPath, err := GetConfigPath()
	require.NoError(t, err)
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Load and verify
	loadedCfg, err := LoadConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "8G", loadedCfg.Defaults.Memory)
}

func TestSaveConfig_NilConfig(t *testing.T) {
	ctx := context.Background()
	err := SaveConfig(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
			errMsg:  "config cannot be nil",
		},
		{
			name: "invalid runtime",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Container.Runtime = "invalid"
				return cfg
			}(),
			wantErr: true,
			errMsg:  "invalid container runtime",
		},
		{
			name: "empty image",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Container.Image = ""
				return cfg
			}(),
			wantErr: true,
			errMsg:  "container image cannot be empty",
		},
		{
			name: "invalid default memory",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Defaults.Memory = "invalid"
				return cfg
			}(),
			wantErr: true,
			errMsg:  "invalid default memory",
		},
		{
			name: "invalid Java version",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Defaults.JavaVersion = 99
				return cfg
			}(),
			wantErr: true,
			errMsg:  "invalid default Java version",
		},
		{
			name: "invalid game port",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Ports.GamePortStart = 0
				return cfg
			}(),
			wantErr: true,
			errMsg:  "invalid game port start",
		},
		{
			name: "invalid rcon port",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Ports.RconPortStart = 70000
				return cfg
			}(),
			wantErr: true,
			errMsg:  "invalid rcon port start",
		},
		{
			name: "rcon password too short",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Ports.RconPasswordLength = 4
				return cfg
			}(),
			wantErr: true,
			errMsg:  "rcon password length must be between 8 and 32",
		},
		{
			name: "negative backup keep count",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Backups.KeepCount = -1
				return cfg
			}(),
			wantErr: true,
			errMsg:  "backup keep count must be >= 0",
		},
		{
			name: "TUI refresh interval too fast",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.TUI.RefreshInterval = 50 * time.Millisecond
				return cfg
			}(),
			wantErr: true,
			errMsg:  "TUI refresh interval must be >= 100ms",
		},
		{
			name: "invalid log level",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Logging.Level = "invalid"
				return cfg
			}(),
			wantErr: true,
			errMsg:  "invalid log level",
		},
		{
			name: "max servers too low",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Limits.MaxServers = 0
				return cfg
			}(),
			wantErr: true,
			errMsg:  "max servers must be >= 1",
		},
		{
			name: "invalid max memory per server",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Limits.MaxMemoryPerServer = "invalid"
				return cfg
			}(),
			wantErr: true,
			errMsg:  "invalid max memory per server",
		},
		{
			name: "max ports too low",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Limits.MaxPorts = 0
				return cfg
			}(),
			wantErr: true,
			errMsg:  "max ports must be >= 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_AtomicWrites(t *testing.T) {
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	// Initialize directories
	err := InitDirs()
	require.NoError(t, err)

	ctx := context.Background()

	// Create initial config
	cfg := DefaultConfig()
	err = SaveConfig(ctx, cfg)
	require.NoError(t, err)

	// Perform multiple updates
	for i := 0; i < 10; i++ {
		cfg.Ports.GamePortStart = 25565 + i
		err = SaveConfig(ctx, cfg)
		require.NoError(t, err)

		// Verify config is valid after each write
		loadedCfg, err := LoadConfig(ctx)
		require.NoError(t, err)
		assert.Equal(t, 25565+i, loadedCfg.Ports.GamePortStart)
	}

	// Verify no temp files are left
	configDir, err := GetConfigDir()
	require.NoError(t, err)
	entries, err := os.ReadDir(configDir)
	require.NoError(t, err)

	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), ".tmp-", "temp file should not exist")
	}
}

func TestConfig_ConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	// Initialize directories
	err := InitDirs()
	require.NoError(t, err)

	ctx := context.Background()

	// Create initial config
	cfg := DefaultConfig()
	err = SaveConfig(ctx, cfg)
	require.NoError(t, err)

	// Perform concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := LoadConfig(ctx)
			require.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all reads to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
