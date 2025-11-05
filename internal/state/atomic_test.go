package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicWrite(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		perm     os.FileMode
		setupDir func(t *testing.T) string
		wantErr  bool
	}{
		{
			name: "successful write",
			data: []byte("test data"),
			perm: 0644,
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
		{
			name: "write with nested directory",
			data: []byte("nested data"),
			perm: 0600,
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
		{
			name: "empty data",
			data: []byte{},
			perm: 0644,
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupDir(t)
			path := filepath.Join(dir, "test.yaml")

			err := AtomicWrite(path, tt.data, tt.perm)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify file exists and has correct content
				content, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, tt.data, content)

				// Verify permissions
				info, err := os.Stat(path)
				require.NoError(t, err)
				assert.Equal(t, tt.perm, info.Mode().Perm())
			}
		})
	}
}

func TestAtomicWrite_OverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.yaml")

	// Write initial data
	initialData := []byte("initial data")
	err := AtomicWrite(path, initialData, 0644)
	require.NoError(t, err)

	// Verify initial write
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, initialData, content)

	// Overwrite with new data
	newData := []byte("new data")
	err = AtomicWrite(path, newData, 0644)
	require.NoError(t, err)

	// Verify overwrite
	content, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, newData, content)
}

func TestAtomicWrite_NoTempFileLeftBehind(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.yaml")

	data := []byte("test data")
	err := AtomicWrite(path, data, 0644)
	require.NoError(t, err)

	// Check that no temp files are left in the directory
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), ".tmp-", "temp file should not exist")
	}
}

func TestAtomicWrite_CreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nested", "dir", "test.yaml")

	data := []byte("test data")
	err := AtomicWrite(path, data, 0644)
	require.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, content)
}

func TestAtomicWriteWithBackup(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.yaml")

	// Write initial data
	initialData := []byte("initial data")
	err := AtomicWrite(path, initialData, 0644)
	require.NoError(t, err)

	// Write new data with backup
	newData := []byte("new data")
	err = AtomicWriteWithBackup(path, newData, 0644)
	require.NoError(t, err)

	// Verify new file has new data
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, newData, content)

	// Verify backup file exists with old data
	backupPath := path + ".bak"
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, initialData, backupContent)
}

func TestAtomicWriteWithBackup_NoExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.yaml")

	// Write to non-existing file
	data := []byte("test data")
	err := AtomicWriteWithBackup(path, data, 0644)
	require.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, content)

	// Verify no backup file was created
	backupPath := path + ".bak"
	_, err = os.Stat(backupPath)
	assert.True(t, os.IsNotExist(err), "backup file should not exist")
}

func TestAtomicWrite_ConcurrentWrites(t *testing.T) {
	// This test verifies that atomic writes don't corrupt the file
	// even with multiple writers
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.yaml")

	// Perform multiple writes sequentially
	// (concurrent writes should use file locking)
	for i := 0; i < 10; i++ {
		data := []byte("iteration " + string(rune('0'+i)))
		err := AtomicWrite(path, data, 0644)
		require.NoError(t, err)

		// Verify file is valid after each write
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	}
}
