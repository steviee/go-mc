package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock, err := LockFile(lockPath)
	require.NoError(t, err)
	require.NotNil(t, lock)
	defer func() { _ = lock.Unlock() }()

	// Verify file was created
	_, err = os.Stat(lockPath)
	require.NoError(t, err)

	// Verify lock is held
	assert.NotNil(t, lock.File())
	assert.Equal(t, lockPath, lock.Path())
}

func TestLockFile_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "new.lock")

	// Verify file doesn't exist
	_, err := os.Stat(lockPath)
	require.True(t, os.IsNotExist(err))

	lock, err := LockFile(lockPath)
	require.NoError(t, err)
	defer func() { _ = lock.Unlock() }()

	// Verify file was created
	_, err = os.Stat(lockPath)
	require.NoError(t, err)
}

func TestTryLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock, err := TryLockFile(lockPath)
	require.NoError(t, err)
	require.NotNil(t, lock)
	defer func() { _ = lock.Unlock() }()

	// Verify file was created
	_, err = os.Stat(lockPath)
	require.NoError(t, err)
}

func TestTryLockFile_AlreadyLocked(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	// Acquire first lock
	lock1, err := TryLockFile(lockPath)
	require.NoError(t, err)
	defer func() { _ = lock1.Unlock() }()

	// Try to acquire second lock (should fail immediately)
	lock2, err := TryLockFile(lockPath)
	assert.Error(t, err)
	assert.Equal(t, ErrLockHeld, err)
	assert.Nil(t, lock2)
}

func TestFileLock_Unlock(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock, err := LockFile(lockPath)
	require.NoError(t, err)

	err = lock.Unlock()
	require.NoError(t, err)

	// Verify we can acquire lock again after unlock
	lock2, err := TryLockFile(lockPath)
	require.NoError(t, err)
	defer func() { _ = lock2.Unlock() }()
}

func TestFileLock_UnlockTwice(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock, err := LockFile(lockPath)
	require.NoError(t, err)

	err = lock.Unlock()
	require.NoError(t, err)

	// Second unlock should not error
	err = lock.Unlock()
	require.NoError(t, err)
}

func TestFileLock_ReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock, err := LockFile(lockPath)
	require.NoError(t, err)
	defer func() { _ = lock.Unlock() }()

	// Write data through lock
	testData := []byte("test data")
	n, err := lock.File().Write(testData)
	require.NoError(t, err)
	assert.Equal(t, len(testData), n)

	// Seek back to beginning
	_, err = lock.File().Seek(0, 0)
	require.NoError(t, err)

	// Read data back
	readData := make([]byte, len(testData))
	n, err = lock.File().Read(readData)
	require.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, testData, readData)
}

func TestFileLock_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	// Acquire first lock
	lock1, err := LockFile(lockPath)
	require.NoError(t, err)

	// Try to acquire lock in goroutine (should block)
	lockAcquired := make(chan bool, 1)
	go func() {
		lock2, err := LockFile(lockPath)
		if err != nil {
			lockAcquired <- false
			return
		}
		defer func() { _ = lock2.Unlock() }()
		lockAcquired <- true
	}()

	// Wait a bit to ensure goroutine is blocked
	time.Sleep(100 * time.Millisecond)

	// Verify lock wasn't acquired
	select {
	case acquired := <-lockAcquired:
		t.Fatalf("lock should not have been acquired: %v", acquired)
	default:
		// Expected: lock is blocked
	}

	// Release first lock
	err = lock1.Unlock()
	require.NoError(t, err)

	// Wait for second lock to be acquired
	select {
	case acquired := <-lockAcquired:
		assert.True(t, acquired, "lock should have been acquired after unlock")
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for lock acquisition")
	}
}

func TestFileLock_MultipleLocks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple lock files
	paths := []string{
		filepath.Join(tmpDir, "lock1.lock"),
		filepath.Join(tmpDir, "lock2.lock"),
		filepath.Join(tmpDir, "lock3.lock"),
	}

	// Acquire all locks
	locks := make([]*FileLock, len(paths))
	for i, path := range paths {
		lock, err := LockFile(path)
		require.NoError(t, err)
		locks[i] = lock
	}

	// Verify all locks are held
	for _, lock := range locks {
		assert.NotNil(t, lock.File())
	}

	// Release all locks
	for _, lock := range locks {
		err := lock.Unlock()
		require.NoError(t, err)
	}
}

func TestFileLock_Accessors(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "accessor.lock")

	lock, err := LockFile(lockPath)
	require.NoError(t, err)
	defer func() { _ = lock.Unlock() }()

	// Test File() accessor
	f := lock.File()
	require.NotNil(t, f)
	assert.NotNil(t, f.Fd(), "file descriptor should be valid")

	// Test Path() accessor
	path := lock.Path()
	assert.Equal(t, lockPath, path)

	// After unlock, File() should return nil
	err = lock.Unlock()
	require.NoError(t, err)
	assert.Nil(t, lock.File(), "File() should return nil after Unlock")
}
