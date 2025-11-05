package state

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// ConfigDirName is the root directory name for go-mc configuration
	ConfigDirName = "go-mc"

	// SubdirectoryNames
	ServersSubdir    = "servers"
	WhitelistsSubdir = "whitelists"
	BackupsSubdir    = "backups"
	ArchivesSubdir   = "archives"

	// File names
	ConfigFileName = "config.yaml"
	StateFileName  = "state.yaml"
)

// GetConfigDir returns the path to the go-mc configuration directory.
// It defaults to ~/.config/go-mc/ and creates the directory if it doesn't exist.
func GetConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configHome = filepath.Join(homeDir, ".config")
	}

	configDir := filepath.Join(configHome, ConfigDirName)
	return configDir, nil
}

// GetServersDir returns the path to the servers directory.
func GetServersDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, ServersSubdir), nil
}

// GetWhitelistsDir returns the path to the whitelists directory.
func GetWhitelistsDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, WhitelistsSubdir), nil
}

// GetBackupsDir returns the path to the backups directory.
func GetBackupsDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, BackupsSubdir), nil
}

// GetArchivesDir returns the path to the backup archives directory.
func GetArchivesDir() (string, error) {
	backupsDir, err := GetBackupsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(backupsDir, ArchivesSubdir), nil
}

// GetConfigPath returns the path to the main configuration file.
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, ConfigFileName), nil
}

// GetStatePath returns the path to the global state file.
func GetStatePath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, StateFileName), nil
}

// GetServerPath returns the path to a specific server's state file.
func GetServerPath(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("server name cannot be empty")
	}
	serversDir, err := GetServersDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(serversDir, name+".yaml"), nil
}

// GetWhitelistPath returns the path to a specific whitelist file.
func GetWhitelistPath(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("whitelist name cannot be empty")
	}
	whitelistsDir, err := GetWhitelistsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(whitelistsDir, name+".yaml"), nil
}

// GetServerBackupDir returns the path to a specific server's backup directory.
func GetServerBackupDir(serverName string) (string, error) {
	if serverName == "" {
		return "", fmt.Errorf("server name cannot be empty")
	}
	backupsDir, err := GetBackupsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(backupsDir, serverName), nil
}

// InitDirs initializes the complete directory structure for go-mc.
// It creates all necessary directories with proper permissions.
func InitDirs() error {
	dirs := []string{
		// Get all required directories
	}

	// Collect all directory paths
	configDir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config dir: %w", err)
	}
	dirs = append(dirs, configDir)

	serversDir, err := GetServersDir()
	if err != nil {
		return fmt.Errorf("failed to get servers dir: %w", err)
	}
	dirs = append(dirs, serversDir)

	whitelistsDir, err := GetWhitelistsDir()
	if err != nil {
		return fmt.Errorf("failed to get whitelists dir: %w", err)
	}
	dirs = append(dirs, whitelistsDir)

	archivesDir, err := GetArchivesDir()
	if err != nil {
		return fmt.Errorf("failed to get archives dir: %w", err)
	}
	dirs = append(dirs, archivesDir)

	// Create all directories
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// EnsureDir ensures that a directory exists, creating it if necessary.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to ensure directory %s: %w", path, err)
	}
	return nil
}
