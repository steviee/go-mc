package state

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	// PIDFileName is the name of the PID lock file
	PIDFileName = "go-mc.pid"
)

// PIDLock represents an active PID lock that prevents concurrent execution.
// It holds a file lock on the PID file and contains the process ID.
type PIDLock struct {
	path string
	file *os.File
	pid  int
}

// AcquirePIDLock creates a PID file and locks it to prevent concurrent execution.
// It automatically cleans up stale PID files from processes that are no longer running.
//
// Returns an error if:
//   - Another instance is already running
//   - Failed to create or lock the PID file
//   - Failed to write the PID to the file
func AcquirePIDLock() (*PIDLock, error) {
	pidPath, err := GetPIDPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get PID file path: %w", err)
	}

	return acquirePIDLockAtPath(pidPath)
}

// acquirePIDLockAtPath is the internal implementation that accepts a custom path.
// This allows testing without modifying global state.
func acquirePIDLockAtPath(pidPath string) (*PIDLock, error) {
	// Ensure the config directory exists
	configDir := filepath.Dir(pidPath)
	if err := EnsureDir(configDir); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if PID file exists and contains a running process
	if _, err := os.Stat(pidPath); err == nil {
		// PID file exists, check if it's stale
		if err := checkAndCleanupStalePID(pidPath); err != nil {
			return nil, err
		}
	}

	// Open/create PID file
	f, err := os.OpenFile(pidPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open PID file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if err == syscall.EWOULDBLOCK {
			// Read the PID from the file to provide better error message
			existingPID, _ := readPIDFromFile(pidPath)
			if existingPID > 0 {
				return nil, fmt.Errorf("another instance of go-mc is already running (PID: %d)", existingPID)
			}
			return nil, fmt.Errorf("another instance of go-mc is already running")
		}
		return nil, fmt.Errorf("failed to acquire PID lock: %w", err)
	}

	// Truncate file and write current PID
	if err := f.Truncate(0); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		return nil, fmt.Errorf("failed to truncate PID file: %w", err)
	}

	currentPID := os.Getpid()
	if _, err := fmt.Fprintf(f, "%d\n", currentPID); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		return nil, fmt.Errorf("failed to write PID to file: %w", err)
	}

	// Sync to ensure PID is written to disk
	if err := f.Sync(); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		return nil, fmt.Errorf("failed to sync PID file: %w", err)
	}

	return &PIDLock{
		path: pidPath,
		file: f,
		pid:  currentPID,
	}, nil
}

// Release releases the PID lock and removes the PID file.
// It's safe to call Release multiple times.
func (pl *PIDLock) Release() error {
	if pl.file == nil {
		return nil
	}

	// Release the lock
	if err := syscall.Flock(int(pl.file.Fd()), syscall.LOCK_UN); err != nil {
		_ = pl.file.Close()
		pl.file = nil
		return fmt.Errorf("failed to release PID lock: %w", err)
	}

	// Close the file
	if err := pl.file.Close(); err != nil {
		pl.file = nil
		return fmt.Errorf("failed to close PID file: %w", err)
	}

	pl.file = nil

	// Remove the PID file
	if err := os.Remove(pl.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	return nil
}

// GetPIDPath returns the path to the PID file (~/.config/go-mc/go-mc.pid).
func GetPIDPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, PIDFileName), nil
}

// IsProcessRunning checks if a process with the given PID is running.
// It uses os.FindProcess and signal 0 to test if the process exists.
//
// Returns:
//   - true if the process exists and is accessible
//   - false if the process doesn't exist or has terminated
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		// On Unix systems, FindProcess always succeeds
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		// Process exists and we have permission to signal it
		return true
	}

	// Check the specific error
	if err == syscall.ESRCH {
		// Process doesn't exist
		return false
	}

	// EPERM means process exists but we don't have permission
	// This counts as "running" from our perspective
	if err == syscall.EPERM {
		return true
	}

	// Any other error, assume not running
	return false
}

// CleanupStalePID removes a PID file if the process is no longer running.
// Returns an error if the process is still running or if cleanup fails.
func CleanupStalePID(path string) error {
	pid, err := readPIDFromFile(path)
	if err != nil {
		return fmt.Errorf("failed to read PID from file: %w", err)
	}

	if pid <= 0 {
		// Invalid PID, remove the file
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove stale PID file: %w", err)
		}
		return nil
	}

	if IsProcessRunning(pid) {
		return fmt.Errorf("process %d is still running, cannot cleanup PID file", pid)
	}

	// Process is not running, safe to remove
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove stale PID file: %w", err)
	}

	return nil
}

// SetupSignalHandler sets up signal handling for graceful shutdown.
// It registers handlers for SIGINT and SIGTERM.
//
// Returns a channel that will receive a signal when shutdown is requested.
// The caller should wait on this channel and perform cleanup when a signal is received.
func SetupSignalHandler() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	return sigChan
}

// checkAndCleanupStalePID checks if a PID file contains a stale PID and cleans it up.
// Returns an error if the process is still running.
func checkAndCleanupStalePID(path string) error {
	pid, err := readPIDFromFile(path)
	if err != nil {
		// If we can't read the PID, try to remove the file
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("failed to remove invalid PID file: %w", removeErr)
		}
		return nil
	}

	if pid <= 0 {
		// Invalid PID, remove the file
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove invalid PID file: %w", err)
		}
		return nil
	}

	// Check if process is running
	if IsProcessRunning(pid) {
		return fmt.Errorf("another instance of go-mc is already running (PID: %d)", pid)
	}

	// Process not running, cleanup stale PID file
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove stale PID file: %w", err)
	}

	return nil
}

// readPIDFromFile reads a PID from the specified file.
// Returns 0 if the file is empty or contains invalid data.
func readPIDFromFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	// Parse PID from file content
	pidStr := strings.TrimSpace(string(data))
	if pidStr == "" {
		return 0, nil
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID format: %w", err)
	}

	return pid, nil
}
