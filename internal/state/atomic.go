package state

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWrite writes data to a file atomically using a temp file + rename strategy.
// This ensures that the file is never in a partially written state, even if the
// process is interrupted during the write.
//
// The operation works as follows:
//  1. Write data to a temporary file in the same directory
//  2. Sync the temp file to disk (fsync)
//  3. Rename the temp file to the target path (atomic on POSIX systems)
//
// If any step fails, the original file (if it exists) remains unchanged.
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	// Ensure the parent directory exists
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to ensure parent directory: %w", err)
	}

	// Create temp file in the same directory as target
	// This ensures the rename is atomic (same filesystem)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on error
	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	// Write data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Sync to disk to ensure data is persisted
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close temp file before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions on temp file
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	// Atomically rename temp file to target path
	// This is atomic on POSIX systems when both files are on the same filesystem
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file to target: %w", err)
	}

	success = true
	return nil
}

// AtomicWriteWithBackup writes data atomically and creates a backup of the existing file.
// If the target file exists, it is backed up to target.bak before writing.
// This is useful when updating critical state files.
func AtomicWriteWithBackup(path string, data []byte, perm os.FileMode) error {
	// Check if target file exists
	if _, err := os.Stat(path); err == nil {
		// File exists, create backup
		backupPath := path + ".bak"
		if err := os.Rename(path, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Perform atomic write
	if err := AtomicWrite(path, data, perm); err != nil {
		return err
	}

	return nil
}
