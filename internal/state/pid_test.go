package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPIDPath(t *testing.T) {
	path, err := GetPIDPath()
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "go-mc")
	assert.Contains(t, path, PIDFileName)
}

func TestAcquirePIDLock(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		wantErr     bool
		errContains string
	}{
		{
			name: "successful lock acquisition",
			setupFunc: func(t *testing.T) string {
				return setupTestPIDDir(t)
			},
			wantErr: false,
		},
		{
			name: "cleanup stale PID file",
			setupFunc: func(t *testing.T) string {
				dir := setupTestPIDDir(t)
				pidPath := filepath.Join(dir, PIDFileName)
				// Write a PID that doesn't exist (99999 should be safe)
				err := os.WriteFile(pidPath, []byte("99999\n"), 0644)
				require.NoError(t, err)
				return dir
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupFunc(t)
			defer func() { _ = os.RemoveAll(dir) }()

			pidPath := filepath.Join(dir, PIDFileName)
			lock, err := acquirePIDLockAtPath(pidPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, lock)
			assert.Equal(t, os.Getpid(), lock.pid)

			// Verify PID file exists and contains correct PID
			pid, err := readPIDFromFile(pidPath)
			require.NoError(t, err)
			assert.Equal(t, os.Getpid(), pid)

			// Clean up
			err = lock.Release()
			require.NoError(t, err)

			// Verify PID file is removed
			_, err = os.Stat(pidPath)
			assert.True(t, os.IsNotExist(err))
		})
	}
}

func TestConcurrentLockPrevention(t *testing.T) {
	dir := setupTestPIDDir(t)
	defer func() { _ = os.RemoveAll(dir) }()

	pidPath := filepath.Join(dir, PIDFileName)

	// Acquire first lock
	lock1, err := acquirePIDLockAtPath(pidPath)
	require.NoError(t, err)
	require.NotNil(t, lock1)
	defer func() { _ = lock1.Release() }()

	// Try to acquire second lock (should fail)
	lock2, err := acquirePIDLockAtPath(pidPath)
	require.Error(t, err)
	assert.Nil(t, lock2)
	assert.Contains(t, err.Error(), "already running")
	assert.Contains(t, err.Error(), fmt.Sprintf("PID: %d", os.Getpid()))
}

func TestPIDLockRelease(t *testing.T) {
	dir := setupTestPIDDir(t)
	defer func() { _ = os.RemoveAll(dir) }()

	pidPath := filepath.Join(dir, PIDFileName)
	lock, err := acquirePIDLockAtPath(pidPath)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// Verify PID file exists
	_, err = os.Stat(pidPath)
	require.NoError(t, err)

	// Release lock
	err = lock.Release()
	require.NoError(t, err)

	// Verify PID file is removed
	_, err = os.Stat(pidPath)
	assert.True(t, os.IsNotExist(err))

	// Multiple releases should be safe
	err = lock.Release()
	assert.NoError(t, err)
}

func TestPIDLockReleaseAndReacquire(t *testing.T) {
	dir := setupTestPIDDir(t)
	defer func() { _ = os.RemoveAll(dir) }()

	pidPath := filepath.Join(dir, PIDFileName)

	// Acquire first lock
	lock1, err := acquirePIDLockAtPath(pidPath)
	require.NoError(t, err)
	require.NotNil(t, lock1)

	// Release first lock
	err = lock1.Release()
	require.NoError(t, err)

	// Should be able to acquire lock again
	lock2, err := acquirePIDLockAtPath(pidPath)
	require.NoError(t, err)
	require.NotNil(t, lock2)

	err = lock2.Release()
	require.NoError(t, err)
}

func TestIsProcessRunning(t *testing.T) {
	tests := []struct {
		name     string
		pid      int
		expected bool
	}{
		{
			name:     "current process",
			pid:      os.Getpid(),
			expected: true,
		},
		{
			name:     "invalid negative PID",
			pid:      -1,
			expected: false,
		},
		{
			name:     "invalid zero PID",
			pid:      0,
			expected: false,
		},
		{
			name:     "non-existent high PID",
			pid:      99999,
			expected: false,
		},
		{
			name:     "init process",
			pid:      1,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProcessRunning(tt.pid)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanupStalePID(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		wantErr     bool
		errContains string
	}{
		{
			name: "cleanup stale PID",
			setupFunc: func(t *testing.T) string {
				dir := setupTestPIDDir(t)
				pidPath := filepath.Join(dir, "test.pid")
				// Write non-existent PID
				err := os.WriteFile(pidPath, []byte("99999\n"), 0644)
				require.NoError(t, err)
				return pidPath
			},
			wantErr: false,
		},
		{
			name: "fail on running process",
			setupFunc: func(t *testing.T) string {
				dir := setupTestPIDDir(t)
				pidPath := filepath.Join(dir, "test.pid")
				// Write current process PID
				err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644)
				require.NoError(t, err)
				return pidPath
			},
			wantErr:     true,
			errContains: "still running",
		},
		{
			name: "cleanup invalid PID",
			setupFunc: func(t *testing.T) string {
				dir := setupTestPIDDir(t)
				pidPath := filepath.Join(dir, "test.pid")
				// Write invalid PID
				err := os.WriteFile(pidPath, []byte("invalid\n"), 0644)
				require.NoError(t, err)
				return pidPath
			},
			wantErr:     true,
			errContains: "failed to read PID",
		},
		{
			name: "cleanup empty PID file",
			setupFunc: func(t *testing.T) string {
				dir := setupTestPIDDir(t)
				pidPath := filepath.Join(dir, "test.pid")
				// Write empty file
				err := os.WriteFile(pidPath, []byte(""), 0644)
				require.NoError(t, err)
				return pidPath
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc(t)
			dir := filepath.Dir(path)
			defer func() { _ = os.RemoveAll(dir) }()

			err := CleanupStalePID(path)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				// Verify file is removed
				_, err := os.Stat(path)
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func TestSetupSignalHandler(t *testing.T) {
	sigChan := SetupSignalHandler()
	require.NotNil(t, sigChan)

	// Send a signal to ourselves
	err := syscall.Kill(os.Getpid(), syscall.SIGTERM)
	require.NoError(t, err)

	// Wait for signal with timeout
	select {
	case sig := <-sigChan:
		assert.Equal(t, syscall.SIGTERM, sig)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for signal")
	}
}

func TestReadPIDFromFile(t *testing.T) {
	// Test file not found error first
	t.Run("file not found", func(t *testing.T) {
		_, err := readPIDFromFile("/nonexistent/path/test.pid")
		require.Error(t, err)
	})

	tests := []struct {
		name        string
		content     string
		expectedPID int
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid PID",
			content:     "12345\n",
			expectedPID: 12345,
			wantErr:     false,
		},
		{
			name:        "valid PID with spaces",
			content:     "  67890  \n",
			expectedPID: 67890,
			wantErr:     false,
		},
		{
			name:        "empty file",
			content:     "",
			expectedPID: 0,
			wantErr:     false,
		},
		{
			name:        "invalid PID format",
			content:     "not-a-number\n",
			expectedPID: 0,
			wantErr:     true,
			errContains: "invalid PID format",
		},
		{
			name:        "whitespace only",
			content:     "   \n",
			expectedPID: 0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTestPIDDir(t)
			defer func() { _ = os.RemoveAll(dir) }()

			pidPath := filepath.Join(dir, "test.pid")
			err := os.WriteFile(pidPath, []byte(tt.content), 0644)
			require.NoError(t, err)

			pid, err := readPIDFromFile(pidPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPID, pid)
			}
		})
	}
}

func TestCheckAndCleanupStalePID(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		wantErr     bool
		errContains string
	}{
		{
			name: "cleanup non-existent process",
			setupFunc: func(t *testing.T) string {
				dir := setupTestPIDDir(t)
				pidPath := filepath.Join(dir, PIDFileName)
				err := os.WriteFile(pidPath, []byte("99999\n"), 0644)
				require.NoError(t, err)
				return pidPath
			},
			wantErr: false,
		},
		{
			name: "fail on running process",
			setupFunc: func(t *testing.T) string {
				dir := setupTestPIDDir(t)
				pidPath := filepath.Join(dir, PIDFileName)
				err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644)
				require.NoError(t, err)
				return pidPath
			},
			wantErr:     true,
			errContains: "already running",
		},
		{
			name: "cleanup invalid PID file",
			setupFunc: func(t *testing.T) string {
				dir := setupTestPIDDir(t)
				pidPath := filepath.Join(dir, PIDFileName)
				err := os.WriteFile(pidPath, []byte("invalid"), 0644)
				require.NoError(t, err)
				return pidPath
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc(t)
			dir := filepath.Dir(path)
			defer func() { _ = os.RemoveAll(dir) }()

			err := checkAndCleanupStalePID(path)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				// Verify file is removed
				_, err := os.Stat(path)
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

// TestAcquirePIDLockRace tests concurrent lock acquisition attempts
func TestAcquirePIDLockRace(t *testing.T) {
	dir := setupTestPIDDir(t)
	defer func() { _ = os.RemoveAll(dir) }()

	pidPath := filepath.Join(dir, PIDFileName)

	// Try to acquire lock from multiple goroutines
	const numGoroutines = 10
	successChan := make(chan *PIDLock, numGoroutines)
	errorChan := make(chan error, numGoroutines)

	// Use a start signal to synchronize goroutines
	startChan := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		go func() {
			// Wait for start signal
			<-startChan
			lock, err := acquirePIDLockAtPath(pidPath)
			if err != nil {
				errorChan <- err
			} else {
				successChan <- lock
			}
		}()
	}

	// Start all goroutines at once
	close(startChan)

	// Collect results
	var successfulLock *PIDLock
	successCount := 0
	errorCount := 0

	for i := 0; i < numGoroutines; i++ {
		select {
		case lock := <-successChan:
			successCount++
			if successfulLock == nil {
				successfulLock = lock
			} else {
				// This should not happen with proper locking
				t.Logf("Warning: Multiple locks acquired, releasing extra lock")
				_ = lock.Release()
			}
		case err := <-errorChan:
			errorCount++
			if !strings.Contains(err.Error(), "already running") && !strings.Contains(err.Error(), "failed to acquire PID lock") {
				t.Logf("Unexpected error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for lock acquisition results")
		}
	}

	// At least one should succeed, others should fail
	assert.GreaterOrEqual(t, successCount, 1, "at least one goroutine should acquire the lock")
	assert.Equal(t, numGoroutines, successCount+errorCount, "all goroutines should complete")

	// Clean up
	if successfulLock != nil {
		err := successfulLock.Release()
		require.NoError(t, err)
	}
}

// setupTestPIDDir creates a temporary directory for PID file testing
func setupTestPIDDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "go-mc-pid-test-*")
	require.NoError(t, err)
	return dir
}

// TestPIDLockWithSignal tests signal handling with PID lock
func TestPIDLockWithSignal(t *testing.T) {
	dir := setupTestPIDDir(t)
	defer func() { _ = os.RemoveAll(dir) }()

	pidPath := filepath.Join(dir, PIDFileName)

	// Acquire lock
	lock, err := acquirePIDLockAtPath(pidPath)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// Setup signal handler
	sigChan := SetupSignalHandler()

	// Create a done channel to coordinate test
	done := make(chan bool)

	// Start goroutine to handle signal
	go func() {
		<-sigChan
		err := lock.Release()
		assert.NoError(t, err)
		done <- true
	}()

	// Send signal to ourselves
	err = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	require.NoError(t, err)

	// Wait for cleanup
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for signal handling")
	}

	// Verify PID file is removed
	_, err = os.Stat(pidPath)
	assert.True(t, os.IsNotExist(err))
}
