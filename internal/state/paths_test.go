package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigDir(t *testing.T) {
	tests := []struct {
		name        string
		envSetup    func(t *testing.T) func()
		wantContain string
	}{
		{
			name: "uses XDG_CONFIG_HOME when set",
			envSetup: func(t *testing.T) func() {
				old := os.Getenv("XDG_CONFIG_HOME")
				_ = os.Setenv("XDG_CONFIG_HOME", "/tmp/test-config")
				return func() { _ = os.Setenv("XDG_CONFIG_HOME", old) }
			},
			wantContain: "/tmp/test-config/go-mc",
		},
		{
			name: "uses ~/.config when XDG_CONFIG_HOME not set",
			envSetup: func(t *testing.T) func() {
				old := os.Getenv("XDG_CONFIG_HOME")
				_ = os.Unsetenv("XDG_CONFIG_HOME")
				return func() { _ = os.Setenv("XDG_CONFIG_HOME", old) }
			},
			wantContain: ".config/go-mc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.envSetup(t)
			defer cleanup()

			dir, err := GetConfigDir()
			require.NoError(t, err)
			assert.Contains(t, dir, tt.wantContain)
		})
	}
}

func TestGetServersDir(t *testing.T) {
	dir, err := GetServersDir()
	require.NoError(t, err)
	assert.Contains(t, dir, "go-mc/servers")
}

func TestGetWhitelistsDir(t *testing.T) {
	dir, err := GetWhitelistsDir()
	require.NoError(t, err)
	assert.Contains(t, dir, "go-mc/whitelists")
}

func TestGetBackupsDir(t *testing.T) {
	dir, err := GetBackupsDir()
	require.NoError(t, err)
	assert.Contains(t, dir, "go-mc/backups")
}

func TestGetArchivesDir(t *testing.T) {
	dir, err := GetArchivesDir()
	require.NoError(t, err)
	assert.Contains(t, dir, "go-mc/backups/archives")
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	require.NoError(t, err)
	assert.Contains(t, path, "go-mc/config.yaml")
}

func TestGetStatePath(t *testing.T) {
	path, err := GetStatePath()
	require.NoError(t, err)
	assert.Contains(t, path, "go-mc/state.yaml")
}

func TestGetServerPath(t *testing.T) {
	tests := []struct {
		name    string
		srvName string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid server name",
			srvName: "survival",
			wantErr: false,
		},
		{
			name:    "empty server name",
			srvName: "",
			wantErr: true,
			errMsg:  "server name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GetServerPath(tt.srvName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Contains(t, path, "go-mc/servers/"+tt.srvName+".yaml")
			}
		})
	}
}

func TestGetWhitelistPath(t *testing.T) {
	tests := []struct {
		name     string
		listName string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid whitelist name",
			listName: "default",
			wantErr:  false,
		},
		{
			name:     "empty whitelist name",
			listName: "",
			wantErr:  true,
			errMsg:   "whitelist name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GetWhitelistPath(tt.listName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Contains(t, path, "go-mc/whitelists/"+tt.listName+".yaml")
			}
		})
	}
}

func TestGetServerBackupDir(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid server name",
			serverName: "survival",
			wantErr:    false,
		},
		{
			name:       "empty server name",
			serverName: "",
			wantErr:    true,
			errMsg:     "server name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GetServerBackupDir(tt.serverName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Contains(t, path, "go-mc/backups/"+tt.serverName)
			}
		})
	}
}

func TestInitDirs(t *testing.T) {
	// Use temp directory for testing
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	err := InitDirs()
	require.NoError(t, err)

	// Check that all directories were created
	expectedDirs := []string{
		filepath.Join(tmpDir, "go-mc"),
		filepath.Join(tmpDir, "go-mc", "servers"),
		filepath.Join(tmpDir, "go-mc", "whitelists"),
		filepath.Join(tmpDir, "go-mc", "backups", "archives"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		require.NoError(t, err, "directory should exist: %s", dir)
		assert.True(t, info.IsDir(), "path should be a directory: %s", dir)
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test", "nested", "dir")

	err := EnsureDir(testPath)
	require.NoError(t, err)

	info, err := os.Stat(testPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Call again should not error
	err = EnsureDir(testPath)
	require.NoError(t, err)
}
