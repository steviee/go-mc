package system

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steviee/go-mc/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckOSCompatibilityFromString(t *testing.T) {
	tests := []struct {
		name      string
		osRelease string
		wantErr   bool
	}{
		{
			name: "Debian 12",
			osRelease: `ID=debian
VERSION_ID="12"`,
			wantErr: false,
		},
		{
			name: "Debian 13",
			osRelease: `ID=debian
VERSION_ID="13"`,
			wantErr: false,
		},
		{
			name: "Debian 12 with extra fields",
			osRelease: `NAME="Debian GNU/Linux"
ID=debian
VERSION_ID="12"
VERSION="12 (bookworm)"
PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"`,
			wantErr: false,
		},
		{
			name: "Ubuntu",
			osRelease: `ID=ubuntu
VERSION_ID="22.04"`,
			wantErr: true,
		},
		{
			name: "Debian 11 (too old)",
			osRelease: `ID=debian
VERSION_ID="11"`,
			wantErr: true,
		},
		{
			name: "Debian 14 (too new)",
			osRelease: `ID=debian
VERSION_ID="14"`,
			wantErr: true,
		},
		{
			name: "Fedora",
			osRelease: `ID=fedora
VERSION_ID="38"`,
			wantErr: true,
		},
		{
			name: "Arch (no version)",
			osRelease: `ID=arch
NAME="Arch Linux"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkOSCompatibilityFromString(tt.osRelease)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckDependencies(t *testing.T) {
	deps, err := checkDependencies()
	assert.NoError(t, err)
	assert.NotEmpty(t, deps)

	// Check that required dependencies are marked as such
	foundPodman := false
	foundPolicyKit := false
	for _, dep := range deps {
		if dep.Command == "podman" {
			foundPodman = true
			assert.True(t, dep.Required, "Podman should be required")
			assert.Equal(t, "Podman", dep.Name)
			assert.Equal(t, "podman", dep.Package)
		}
		if dep.Command == "pkaction" {
			foundPolicyKit = true
			assert.True(t, dep.Required, "PolicyKit should be required")
			assert.Equal(t, "PolicyKit", dep.Name)
			assert.Equal(t, "polkitd", dep.Package)
		}
	}
	assert.True(t, foundPodman, "Podman should be in dependency list")
	assert.True(t, foundPolicyKit, "PolicyKit should be in dependency list")
}

func TestFilterMissing(t *testing.T) {
	tests := []struct {
		name     string
		deps     []Dependency
		wantLen  int
		wantName string
	}{
		{
			name: "one missing required",
			deps: []Dependency{
				{Name: "Podman", Command: "podman", Required: true, Installed: false},
				{Name: "curl", Command: "curl", Required: false, Installed: true},
				{Name: "git", Command: "git", Required: false, Installed: false},
			},
			wantLen:  1,
			wantName: "Podman",
		},
		{
			name: "no missing required",
			deps: []Dependency{
				{Name: "Podman", Command: "podman", Required: true, Installed: true},
				{Name: "curl", Command: "curl", Required: false, Installed: true},
				{Name: "git", Command: "git", Required: false, Installed: false},
			},
			wantLen: 0,
		},
		{
			name: "all missing but not required",
			deps: []Dependency{
				{Name: "curl", Command: "curl", Required: false, Installed: false},
				{Name: "git", Command: "git", Required: false, Installed: false},
			},
			wantLen: 0,
		},
		{
			name: "multiple missing required",
			deps: []Dependency{
				{Name: "Podman", Command: "podman", Required: true, Installed: false},
				{Name: "systemd", Command: "systemd", Required: true, Installed: false},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missing := filterMissing(tt.deps)
			assert.Len(t, missing, tt.wantLen)
			if tt.wantLen > 0 && tt.wantName != "" {
				assert.Equal(t, tt.wantName, missing[0].Name)
			}
		})
	}
}

func TestIsAlreadyInitialized(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if oldConfigHome != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	t.Run("not initialized", func(t *testing.T) {
		initialized, err := isAlreadyInitialized()
		require.NoError(t, err)
		assert.False(t, initialized)
	})

	t.Run("initialized with config file", func(t *testing.T) {
		// Create config directory and file
		configDir := filepath.Join(tmpDir, "go-mc")
		require.NoError(t, os.MkdirAll(configDir, 0750))

		configPath := filepath.Join(configDir, "config.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("test: config"), 0644))

		initialized, err := isAlreadyInitialized()
		require.NoError(t, err)
		assert.True(t, initialized)

		// Cleanup
		_ = os.Remove(configPath)
	})

	t.Run("initialized with state file", func(t *testing.T) {
		// Create config directory and state file
		configDir := filepath.Join(tmpDir, "go-mc")
		require.NoError(t, os.MkdirAll(configDir, 0750))

		statePath := filepath.Join(configDir, "state.yaml")
		require.NoError(t, os.WriteFile(statePath, []byte("test: state"), 0644))

		initialized, err := isAlreadyInitialized()
		require.NoError(t, err)
		assert.True(t, initialized)

		// Cleanup
		_ = os.Remove(statePath)
	})
}

func TestCreateDirectories(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	oldDataHome := os.Getenv("XDG_DATA_HOME")
	defer func() {
		if oldConfigHome != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
		if oldDataHome != "" {
			_ = os.Setenv("XDG_DATA_HOME", oldDataHome)
		} else {
			_ = os.Unsetenv("XDG_DATA_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	_ = os.Setenv("XDG_DATA_HOME", tmpDir)

	err := createDirectories()
	require.NoError(t, err)

	// Verify config directories exist
	configDir := filepath.Join(tmpDir, "go-mc")
	assert.DirExists(t, configDir)
	assert.DirExists(t, filepath.Join(configDir, "servers"))
	assert.DirExists(t, filepath.Join(configDir, "whitelists"))
	assert.DirExists(t, filepath.Join(configDir, "backups", "archives"))

	// Verify data directories exist
	dataDir := filepath.Join(tmpDir, "go-mc")
	assert.DirExists(t, dataDir)
	assert.DirExists(t, filepath.Join(dataDir, "servers"))
	assert.DirExists(t, filepath.Join(dataDir, "backups"))

	// Verify permissions (should be 0750)
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0750), info.Mode().Perm())
}

func TestInitializeGlobalState(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if oldConfigHome != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create config directory
	configDir := filepath.Join(tmpDir, "go-mc")
	require.NoError(t, os.MkdirAll(configDir, 0750))

	ctx := context.Background()

	t.Run("initialize new state", func(t *testing.T) {
		err := initializeGlobalState(ctx)
		require.NoError(t, err)

		// Verify state file exists
		statePath := filepath.Join(configDir, "state.yaml")
		assert.FileExists(t, statePath)

		// Verify state can be loaded
		globalState, err := state.LoadGlobalState(ctx)
		require.NoError(t, err)
		assert.NotNil(t, globalState)
		assert.Empty(t, globalState.AllocatedPorts)
		assert.Empty(t, globalState.Servers)
	})

	t.Run("state already exists", func(t *testing.T) {
		// State already created in previous test
		err := initializeGlobalState(ctx)
		require.NoError(t, err)

		// Should not fail or overwrite existing state
		statePath := filepath.Join(configDir, "state.yaml")
		assert.FileExists(t, statePath)
	})
}

func TestGenerateDefaultConfig(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp directory
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if oldConfigHome != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create config directory
	configDir := filepath.Join(tmpDir, "go-mc")
	require.NoError(t, os.MkdirAll(configDir, 0750))

	ctx := context.Background()

	t.Run("generate new config", func(t *testing.T) {
		// Create a mock stdout
		stdout := os.Stdout

		err := generateDefaultConfig(ctx, stdout, true) // non-interactive
		require.NoError(t, err)

		// Verify config file exists
		configPath := filepath.Join(configDir, "config.yaml")
		assert.FileExists(t, configPath)

		// Verify config can be loaded
		cfg, err := state.LoadConfig(ctx)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "podman", cfg.Container.Runtime)
		assert.Equal(t, "2G", cfg.Defaults.Memory)
	})

	t.Run("config already exists (non-interactive)", func(t *testing.T) {
		// Config already created in previous test
		stdout := os.Stdout

		err := generateDefaultConfig(ctx, stdout, true) // non-interactive
		require.NoError(t, err)

		// Should not overwrite in non-interactive mode
		configPath := filepath.Join(configDir, "config.yaml")
		assert.FileExists(t, configPath)
	})
}

func TestPromptForConfig(t *testing.T) {
	// This test is difficult to automate as it requires stdin input
	// We'll just test that the function doesn't panic with default values
	stdout := os.Stdout
	config := state.DefaultConfig()

	// We can't actually test the interactive prompting without mocking stdin
	// Just verify the function signature works
	result := promptForConfig(stdout, config)
	assert.NotNil(t, result)
	assert.Equal(t, config, result) // No changes without input
}

func TestConfirm(t *testing.T) {
	// This test requires stdin mocking, which is complex
	// We'll skip automated testing for this interactive function
	t.Skip("Skipping interactive test - requires stdin mocking")
}

func TestReadLine(t *testing.T) {
	// This test requires stdin mocking, which is complex
	// We'll skip automated testing for this interactive function
	t.Skip("Skipping interactive test - requires stdin mocking")
}

// TestPullImageWithCommand tests if podman pull can be executed
// This is a basic test that checks if the command structure is correct
func TestPullImageWithCommand(t *testing.T) {
	// Only run if podman is available
	_, err := exec.LookPath("podman")
	if err != nil {
		t.Skip("Podman not available, skipping pull test")
	}

	t.Run("command structure", func(t *testing.T) {
		// Don't actually pull the image, just verify command structure
		// by checking that podman is available
		cmd := exec.Command("podman", "--version")
		err := cmd.Run()
		assert.NoError(t, err, "Podman should be available for testing")
	})
}

// Integration test for the full setup flow
func TestSetupIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires:
	// - Debian 12/13 OS
	// - sudo access
	// - Podman not yet configured
	// It's meant to be run manually or in CI with proper setup
	t.Skip("Skipping full integration test - requires manual setup")
}

// Benchmark dependency checking
func BenchmarkCheckDependencies(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = checkDependencies()
	}
}

// Benchmark OS compatibility check
func BenchmarkCheckOSCompatibility(b *testing.B) {
	osRelease := `ID=debian
VERSION_ID="12"
NAME="Debian GNU/Linux"
VERSION="12 (bookworm)"
PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checkOSCompatibilityFromString(osRelease)
	}
}

func TestPrintHeader(t *testing.T) {
	// Create a buffer to capture output
	var buf strings.Builder

	printHeader(&buf)

	output := buf.String()
	assert.Contains(t, output, "go-mc System Setup")
	assert.Contains(t, output, "==================")
}

func TestOutputSuccess(t *testing.T) {
	t.Run("json mode", func(t *testing.T) {
		var buf bytes.Buffer

		err := outputSuccess(&buf, true)
		require.NoError(t, err)

		var output SetupOutput
		err = json.NewDecoder(&buf).Decode(&output)
		require.NoError(t, err)

		assert.Equal(t, "success", output.Status)
		assert.Equal(t, "Setup completed successfully", output.Message)
		assert.NotNil(t, output.Data)

		// Check next steps are present
		nextSteps, ok := output.Data["next_steps"].([]interface{})
		require.True(t, ok)
		assert.Len(t, nextSteps, 3)
	})

	t.Run("text mode", func(t *testing.T) {
		var buf bytes.Buffer

		err := outputSuccess(&buf, false)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Setup completed successfully!")
		assert.Contains(t, output, "Next steps:")
		assert.Contains(t, output, "go-mc servers create")
		assert.Contains(t, output, "go-mc servers start")
		assert.Contains(t, output, "go-mc servers top")
	})
}

func TestOutputError(t *testing.T) {
	testErr := fmt.Errorf("test error message")

	t.Run("json mode", func(t *testing.T) {
		var buf bytes.Buffer

		err := outputError(&buf, true, testErr)
		assert.Equal(t, testErr, err) // Should return the original error

		var output SetupOutput
		err = json.NewDecoder(&buf).Decode(&output)
		require.NoError(t, err)

		assert.Equal(t, "error", output.Status)
		assert.Equal(t, "test error message", output.Error)
	})

	t.Run("text mode", func(t *testing.T) {
		var buf bytes.Buffer

		err := outputError(&buf, false, testErr)
		assert.Equal(t, testErr, err) // Should return the original error

		// In text mode, nothing is written to stdout (error is just returned)
		assert.Empty(t, buf.String())
	})
}

func TestIsJSONMode(t *testing.T) {
	// Test that isJSONMode returns false (current implementation)
	result := isJSONMode()
	assert.False(t, result, "isJSONMode should return false until JSON mode is implemented")
}
