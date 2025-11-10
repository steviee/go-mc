package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/steviee/go-mc/internal/state"
)

// Service provides backup and restore functionality.
type Service struct {
	// Future: could add options like compression level, etc.
}

// NewService creates a new backup service.
func NewService() *Service {
	return &Service{}
}

// CreateBackupOptions holds options for creating a backup.
type CreateBackupOptions struct {
	ServerName string
	Compress   bool // Default: true
	KeepCount  int  // Retention policy: keep last N backups (default: 5)
}

// CreateBackupResult holds the result of a backup operation.
type CreateBackupResult struct {
	BackupID   string
	BackupInfo state.BackupInfo
	Duration   time.Duration
}

// CreateBackup creates a compressed backup of a server's data and mods directories.
func (s *Service) CreateBackup(ctx context.Context, opts CreateBackupOptions) (*CreateBackupResult, error) {
	startTime := time.Now()

	// Validate options
	if opts.ServerName == "" {
		return nil, fmt.Errorf("server name cannot be empty")
	}
	if opts.KeepCount < 1 {
		opts.KeepCount = 5 // Default
	}

	// Load server state to get paths and metadata
	serverState, err := state.LoadServerState(ctx, opts.ServerName)
	if err != nil {
		return nil, fmt.Errorf("failed to load server state: %w", err)
	}

	// Check if server data directory exists
	if serverState.Volumes.Data == "" {
		return nil, fmt.Errorf("server data volume not configured")
	}
	if _, err := os.Stat(serverState.Volumes.Data); os.IsNotExist(err) {
		return nil, fmt.Errorf("server data directory does not exist: %s", serverState.Volumes.Data)
	}

	// Generate backup ID and paths
	now := time.Now()
	backupID := state.GenerateBackupID(opts.ServerName, now)
	filename := backupID + ".tar.gz"

	archivesDir, err := state.GetArchivesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get archives directory: %w", err)
	}
	if err := state.EnsureDir(archivesDir); err != nil {
		return nil, fmt.Errorf("failed to ensure archives directory: %w", err)
	}

	archivePath := filepath.Join(archivesDir, filename)

	// Check available disk space (require 2x estimated size)
	estimatedSize, err := estimateDirectorySize(serverState.Volumes.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate backup size: %w", err)
	}
	if err := checkDiskSpace(archivesDir, estimatedSize*2); err != nil {
		return nil, fmt.Errorf("insufficient disk space: %w", err)
	}

	// Create tar.gz archive
	archiveSize, err := s.createTarGz(ctx, serverState, archivePath)
	if err != nil {
		// Clean up partial archive on error
		_ = os.Remove(archivePath)
		return nil, fmt.Errorf("failed to create archive: %w", err)
	}

	// Create backup info
	backupInfo := state.BackupInfo{
		ID:               backupID,
		Server:           opts.ServerName,
		MinecraftVersion: serverState.Minecraft.Version,
		FabricVersion:    serverState.Minecraft.FabricLoaderVersion,
		ModsCount:        len(serverState.Mods),
		Filename:         filename,
		FilePath:         archivePath,
		SizeBytes:        archiveSize,
		Compressed:       opts.Compress,
		CreatedAt:        now,
	}

	// Add to registry
	if err := state.AddBackup(ctx, backupInfo); err != nil {
		// Clean up archive if we can't add to registry
		_ = os.Remove(archivePath)
		return nil, fmt.Errorf("failed to add backup to registry: %w", err)
	}

	// Enforce retention policy
	if err := state.EnforceRetentionPolicy(ctx, opts.KeepCount); err != nil {
		// Don't fail the backup, just log the error
		// In production, would use structured logging
		_ = err
	}

	duration := time.Since(startTime)
	return &CreateBackupResult{
		BackupID:   backupID,
		BackupInfo: backupInfo,
		Duration:   duration,
	}, nil
}

// RestoreBackupOptions holds options for restoring a backup.
type RestoreBackupOptions struct {
	BackupID   string
	ServerName string
	Force      bool // Skip confirmation (handled by CLI)
}

// RestoreBackup restores a server from a backup.
func (s *Service) RestoreBackup(ctx context.Context, opts RestoreBackupOptions) error {
	// Validate options
	if opts.BackupID == "" {
		return fmt.Errorf("backup ID cannot be empty")
	}
	if opts.ServerName == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	// Get backup info
	backupInfo, err := state.GetBackup(ctx, opts.BackupID)
	if err != nil {
		return fmt.Errorf("failed to get backup: %w", err)
	}

	// Verify backup is for the correct server
	if backupInfo.Server != opts.ServerName {
		return fmt.Errorf("backup is for server %q, not %q", backupInfo.Server, opts.ServerName)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupInfo.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupInfo.FilePath)
	}

	// Load server state
	serverState, err := state.LoadServerState(ctx, opts.ServerName)
	if err != nil {
		return fmt.Errorf("failed to load server state: %w", err)
	}

	if serverState.Volumes.Data == "" {
		return fmt.Errorf("server data volume not configured")
	}

	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "go-mc-restore-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract archive to temp directory
	if err := s.extractTarGz(ctx, backupInfo.FilePath, tempDir); err != nil {
		return fmt.Errorf("failed to extract backup: %w", err)
	}

	// Backup current data (for rollback on failure)
	serverDir := filepath.Dir(serverState.Volumes.Data)
	rollbackDir, err := os.MkdirTemp("", "go-mc-rollback-*")
	if err != nil {
		return fmt.Errorf("failed to create rollback directory: %w", err)
	}
	defer os.RemoveAll(rollbackDir)

	// Move current data to rollback location
	if err := os.Rename(serverState.Volumes.Data, filepath.Join(rollbackDir, "data")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to backup current data: %w", err)
	}
	modsDir := filepath.Join(serverDir, "mods")
	if err := os.Rename(modsDir, filepath.Join(rollbackDir, "mods")); err != nil && !os.IsNotExist(err) {
		// Rollback data move
		_ = os.Rename(filepath.Join(rollbackDir, "data"), serverState.Volumes.Data)
		return fmt.Errorf("failed to backup current mods: %w", err)
	}

	// Move restored data to server location
	success := false
	defer func() {
		if !success {
			// Rollback on failure
			_ = os.RemoveAll(serverState.Volumes.Data)
			_ = os.RemoveAll(modsDir)
			_ = os.Rename(filepath.Join(rollbackDir, "data"), serverState.Volumes.Data)
			_ = os.Rename(filepath.Join(rollbackDir, "mods"), modsDir)
		}
	}()

	// Move data directory
	if err := os.Rename(filepath.Join(tempDir, "data"), serverState.Volumes.Data); err != nil {
		return fmt.Errorf("failed to restore data directory: %w", err)
	}

	// Move mods directory
	if err := os.Rename(filepath.Join(tempDir, "mods"), modsDir); err != nil {
		return fmt.Errorf("failed to restore mods directory: %w", err)
	}

	success = true
	return nil
}

// createTarGz creates a compressed tar.gz archive of the server's data and mods.
// Returns the size of the created archive.
func (s *Service) createTarGz(ctx context.Context, serverState *state.ServerState, archivePath string) (int64, error) {
	// Create output file
	outFile, err := os.Create(archivePath)
	if err != nil {
		return 0, fmt.Errorf("failed to create archive file: %w", err)
	}
	defer outFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add data directory to archive
	if err := addDirToTar(tarWriter, serverState.Volumes.Data, "data"); err != nil {
		return 0, fmt.Errorf("failed to add data directory to archive: %w", err)
	}

	// Add mods directory to archive (if it exists)
	serverDir := filepath.Dir(serverState.Volumes.Data)
	modsDir := filepath.Join(serverDir, "mods")
	if _, err := os.Stat(modsDir); err == nil {
		if err := addDirToTar(tarWriter, modsDir, "mods"); err != nil {
			return 0, fmt.Errorf("failed to add mods directory to archive: %w", err)
		}
	}

	// Close writers to flush data
	if err := tarWriter.Close(); err != nil {
		return 0, fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return 0, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	if err := outFile.Close(); err != nil {
		return 0, fmt.Errorf("failed to close output file: %w", err)
	}

	// Get archive size
	stat, err := os.Stat(archivePath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat archive: %w", err)
	}

	return stat.Size(), nil
}

// extractTarGz extracts a tar.gz archive to the specified directory.
func (s *Service) extractTarGz(ctx context.Context, archivePath, destDir string) error {
	// Open archive file
	inFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer inFile.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(inFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct target path
		target := filepath.Join(destDir, header.Name)

		// Ensure target is within destDir (security: prevent path traversal)
		if !filepath.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		// Handle different file types
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}

		case tar.TypeReg:
			// Create regular file
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", target, err)
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}

			if err := outFile.Close(); err != nil {
				return fmt.Errorf("failed to close file %s: %w", target, err)
			}

		default:
			// Skip unsupported file types (symlinks, devices, etc.)
			continue
		}
	}

	return nil
}

// addDirToTar recursively adds a directory to a tar archive.
func addDirToTar(tw *tar.Writer, sourceDir, archivePrefix string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header from file info
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", path, err)
		}

		// Update name to be relative to archive prefix
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		header.Name = filepath.Join(archivePrefix, relPath)

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// If not a regular file, we're done
		if !info.Mode().IsRegular() {
			return nil
		}

		// Write file contents
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("failed to write file %s to archive: %w", path, err)
		}

		return nil
	})
}

// estimateDirectorySize estimates the total size of a directory and its contents.
func estimateDirectorySize(dir string) (int64, error) {
	var size int64
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// checkDiskSpace checks if there is enough free disk space in the target directory.
func checkDiskSpace(dir string, required int64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return fmt.Errorf("failed to stat filesystem: %w", err)
	}

	// Available space = block size * available blocks
	available := int64(stat.Bavail) * int64(stat.Bsize)

	if available < required {
		return fmt.Errorf("insufficient disk space: required %d bytes, available %d bytes",
			required, available)
	}

	return nil
}
