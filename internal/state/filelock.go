package state

import (
	"fmt"
	"os"
	"syscall"
)

// FileLock represents a file lock using flock(2).
// It prevents concurrent access to critical files.
type FileLock struct {
	file *os.File
	path string
}

// LockFile acquires an exclusive lock on the specified file.
// It creates the file if it doesn't exist.
// The caller must call Unlock() to release the lock.
//
// This uses syscall.Flock() with LOCK_EX, which provides
// advisory locking (processes must cooperate by using locks).
func LockFile(path string) (*FileLock, error) {
	// Open file for read/write, create if doesn't exist with secure permissions (0600)
	//nolint:gosec // G304: File path is controlled by application, not user input
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for locking: %w", err)
	}

	// Acquire exclusive lock (blocks until available)
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return &FileLock{
		file: f,
		path: path,
	}, nil
}

// TryLockFile attempts to acquire an exclusive lock on the specified file.
// Unlike LockFile, it returns immediately if the lock cannot be acquired.
// Returns nil, ErrLockHeld if the lock is already held by another process.
func TryLockFile(path string) (*FileLock, error) {
	// Open file for read/write, create if doesn't exist with secure permissions (0600)
	//nolint:gosec // G304: File path is controlled by application, not user input
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for locking: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, ErrLockHeld
		}
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return &FileLock{
		file: f,
		path: path,
	}, nil
}

// Unlock releases the file lock and closes the file.
func (fl *FileLock) Unlock() error {
	if fl.file == nil {
		return nil
	}

	// Release lock
	if err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN); err != nil {
		_ = fl.file.Close()
		return fmt.Errorf("failed to release lock: %w", err)
	}

	// Close file
	if err := fl.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	fl.file = nil
	return nil
}

// File returns the underlying file descriptor.
// This can be used to read/write the locked file.
func (fl *FileLock) File() *os.File {
	return fl.file
}

// Path returns the path to the locked file.
func (fl *FileLock) Path() string {
	return fl.path
}

// ErrLockHeld is returned when TryLockFile cannot acquire a lock
// because it is already held by another process.
var ErrLockHeld = fmt.Errorf("lock is held by another process")
