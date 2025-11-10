package state

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// BackupInfo represents metadata for a single backup.
type BackupInfo struct {
	ID               string    `yaml:"id"`
	Server           string    `yaml:"server"`
	MinecraftVersion string    `yaml:"minecraft_version"`
	FabricVersion    string    `yaml:"fabric_version"`
	ModsCount        int       `yaml:"mods_count"`
	Filename         string    `yaml:"filename"`
	FilePath         string    `yaml:"file_path"`
	SizeBytes        int64     `yaml:"size_bytes"`
	Compressed       bool      `yaml:"compressed"`
	CreatedAt        time.Time `yaml:"created_at"`
}

// BackupRegistry holds all backup metadata.
type BackupRegistry struct {
	Backups []BackupInfo `yaml:"backups"`
}

// GetBackupRegistryPath returns the path to the backup registry file.
func GetBackupRegistryPath() (string, error) {
	backupsDir, err := GetBackupsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(backupsDir, "registry.yaml"), nil
}

// LoadBackupRegistry loads the backup registry from disk.
// If the file doesn't exist, returns an empty registry (not an error).
func LoadBackupRegistry(ctx context.Context) (*BackupRegistry, error) {
	registryPath, err := GetBackupRegistryPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get registry path: %w", err)
	}

	// If registry doesn't exist, return empty registry
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		return &BackupRegistry{Backups: []BackupInfo{}}, nil
	}

	// Read registry file
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	// Parse YAML
	var registry BackupRegistry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	// Ensure Backups slice is never nil
	if registry.Backups == nil {
		registry.Backups = []BackupInfo{}
	}

	return &registry, nil
}

// SaveBackupRegistry saves the backup registry to disk atomically.
func SaveBackupRegistry(ctx context.Context, registry *BackupRegistry) error {
	if registry == nil {
		return fmt.Errorf("registry cannot be nil")
	}

	registryPath, err := GetBackupRegistryPath()
	if err != nil {
		return fmt.Errorf("failed to get registry path: %w", err)
	}

	// Ensure backups directory exists
	backupsDir, err := GetBackupsDir()
	if err != nil {
		return fmt.Errorf("failed to get backups dir: %w", err)
	}
	if err := EnsureDir(backupsDir); err != nil {
		return fmt.Errorf("failed to ensure backups dir: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(registry)
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write atomically
	if err := AtomicWrite(registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// AddBackup adds a new backup to the registry and saves it.
func AddBackup(ctx context.Context, backup BackupInfo) error {
	// Validate backup info
	if backup.ID == "" {
		return fmt.Errorf("backup ID cannot be empty")
	}
	if backup.Server == "" {
		return fmt.Errorf("backup server name cannot be empty")
	}
	if backup.FilePath == "" {
		return fmt.Errorf("backup file path cannot be empty")
	}

	// Load current registry
	registry, err := LoadBackupRegistry(ctx)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Check if backup ID already exists
	for _, b := range registry.Backups {
		if b.ID == backup.ID {
			return fmt.Errorf("backup with ID %q already exists", backup.ID)
		}
	}

	// Add new backup
	registry.Backups = append(registry.Backups, backup)

	// Save registry
	if err := SaveBackupRegistry(ctx, registry); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	return nil
}

// RemoveBackup removes a backup from the registry by ID.
// It does not delete the backup file itself.
func RemoveBackup(ctx context.Context, backupID string) error {
	if backupID == "" {
		return fmt.Errorf("backup ID cannot be empty")
	}

	// Load current registry
	registry, err := LoadBackupRegistry(ctx)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Find and remove backup
	found := false
	newBackups := make([]BackupInfo, 0, len(registry.Backups))
	for _, b := range registry.Backups {
		if b.ID == backupID {
			found = true
			continue
		}
		newBackups = append(newBackups, b)
	}

	if !found {
		return fmt.Errorf("backup with ID %q not found", backupID)
	}

	registry.Backups = newBackups

	// Save registry
	if err := SaveBackupRegistry(ctx, registry); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	return nil
}

// GetBackup retrieves a backup by ID.
func GetBackup(ctx context.Context, backupID string) (*BackupInfo, error) {
	if backupID == "" {
		return nil, fmt.Errorf("backup ID cannot be empty")
	}

	registry, err := LoadBackupRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	for _, b := range registry.Backups {
		if b.ID == backupID {
			return &b, nil
		}
	}

	return nil, fmt.Errorf("backup with ID %q not found", backupID)
}

// ListBackups returns all backups for a specific server, sorted by creation time (newest first).
// If serverName is empty, returns all backups across all servers.
func ListBackups(ctx context.Context, serverName string) ([]BackupInfo, error) {
	registry, err := LoadBackupRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Filter by server if specified
	var backups []BackupInfo
	if serverName == "" {
		backups = registry.Backups
	} else {
		for _, b := range registry.Backups {
			if b.Server == serverName {
				backups = append(backups, b)
			}
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// EnforceRetentionPolicy removes old backups beyond the keep count for each server.
// It deletes both the registry entries and the backup files themselves.
func EnforceRetentionPolicy(ctx context.Context, keepCount int) error {
	if keepCount < 1 {
		return fmt.Errorf("keep count must be at least 1, got %d", keepCount)
	}

	registry, err := LoadBackupRegistry(ctx)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Group backups by server
	serverBackups := make(map[string][]BackupInfo)
	for _, b := range registry.Backups {
		serverBackups[b.Server] = append(serverBackups[b.Server], b)
	}

	// For each server, keep only the newest N backups
	toRemove := make(map[string]bool) // Set of backup IDs to remove
	for server, backups := range serverBackups {
		// Sort by creation time (newest first)
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].CreatedAt.After(backups[j].CreatedAt)
		})

		// Mark old backups for removal
		if len(backups) > keepCount {
			for i := keepCount; i < len(backups); i++ {
				toRemove[backups[i].ID] = true
			}
		}

		_ = server // Mark as used (for linter)
	}

	// Remove old backups (both files and registry entries)
	for backupID := range toRemove {
		// Find backup in registry
		var backup *BackupInfo
		for i := range registry.Backups {
			if registry.Backups[i].ID == backupID {
				backup = &registry.Backups[i]
				break
			}
		}

		if backup == nil {
			continue
		}

		// Delete backup file
		if err := os.Remove(backup.FilePath); err != nil && !os.IsNotExist(err) {
			// Log error but continue (don't fail entire cleanup)
			// In production, might want to use structured logging here
			_ = err
		}

		// Remove from registry
		if err := RemoveBackup(ctx, backupID); err != nil {
			return fmt.Errorf("failed to remove backup %q from registry: %w", backupID, err)
		}
	}

	return nil
}

// GenerateBackupID generates a unique backup ID from server name and timestamp.
func GenerateBackupID(serverName string, timestamp time.Time) string {
	return fmt.Sprintf("backup-%s-%s", serverName, timestamp.Format("2006-01-02-15-04-05"))
}
