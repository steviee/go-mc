package servers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steviee/go-mc/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRCONPassword(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "generate password 1"},
		{name: "generate password 2"},
		{name: "generate password 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password := generateRCONPassword()

			// Check length
			assert.Len(t, password, 16, "password should be 16 characters")

			// Check charset (alphanumeric only)
			for _, ch := range password {
				isAlpha := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
				isDigit := ch >= '0' && ch <= '9'
				assert.True(t, isAlpha || isDigit, "password should only contain alphanumeric characters")
			}
		})
	}

	// Test uniqueness
	passwords := make(map[string]bool)
	for i := 0; i < 100; i++ {
		password := generateRCONPassword()
		assert.False(t, passwords[password], "passwords should be unique")
		passwords[password] = true
	}
}

func TestBuildServerConfig(t *testing.T) {
	ctx := context.Background()

	// Setup test state directory
	tmpDir := t.TempDir()
	setupTestStateDir(t, tmpDir)

	tests := []struct {
		name      string
		flags     *CreateFlags
		wantErr   bool
		errMsg    string
		checkPort bool
	}{
		{
			name: "valid config with defaults",
			flags: &CreateFlags{
				Version: "1.21.1",
				Memory:  "2G",
				Port:    0, // auto-allocate
			},
			wantErr:   false,
			checkPort: true,
		},
		{
			name: "valid config with explicit port",
			flags: &CreateFlags{
				Version: "1.21.1",
				Memory:  "4G",
				Port:    25566,
			},
			wantErr:   false,
			checkPort: true,
		},
		{
			name: "invalid version",
			flags: &CreateFlags{
				Version: "",
				Memory:  "2G",
			},
			wantErr: true,
			errMsg:  "invalid version",
		},
		{
			name: "invalid memory format",
			flags: &CreateFlags{
				Version: "1.21.1",
				Memory:  "invalid",
			},
			wantErr: true,
			errMsg:  "invalid memory format",
		},
		{
			name: "invalid port",
			flags: &CreateFlags{
				Version: "1.21.1",
				Memory:  "2G",
				Port:    99999,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "valid config with mods",
			flags: &CreateFlags{
				Version: "1.21.1",
				Memory:  "2G",
				Port:    0,
				Mods:    []string{"fabric-api", "sodium"},
			},
			wantErr:   false,
			checkPort: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := buildServerConfig(ctx, "testserver", tt.flags)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "testserver", config.Name)
			assert.Equal(t, tt.flags.Version, config.Version)
			assert.Equal(t, tt.flags.Memory, config.Memory)

			if tt.checkPort {
				if tt.flags.Port != 0 {
					assert.Equal(t, tt.flags.Port, config.Port)
				} else {
					assert.GreaterOrEqual(t, config.Port, defaultStartPort)
				}

				// Check RCON port calculation
				assert.Equal(t, config.Port+rconPortOffset, config.RCONPort)
			}

			// Check RCON password is generated
			assert.NotEmpty(t, config.RCONPass)
			assert.Len(t, config.RCONPass, 16)

			// Check mods if provided
			if len(tt.flags.Mods) > 0 {
				assert.Equal(t, tt.flags.Mods, config.Mods)
			}
		})
	}
}

func TestBuildServerConfig_PortAllocation(t *testing.T) {
	ctx := context.Background()

	// Setup test state directory
	tmpDir := t.TempDir()
	setupTestStateDir(t, tmpDir)

	// Allocate some ports
	require.NoError(t, state.AllocatePort(ctx, 25565))
	require.NoError(t, state.AllocatePort(ctx, 25566))

	flags := &CreateFlags{
		Version: "1.21.1",
		Memory:  "2G",
		Port:    0, // auto-allocate
	}

	config, err := buildServerConfig(ctx, "testserver", flags)
	require.NoError(t, err)

	// Should allocate next available port (25567)
	assert.Equal(t, 25567, config.Port)
	assert.Equal(t, 25567+rconPortOffset, config.RCONPort)
}

func TestBuildServerConfig_PortConflict(t *testing.T) {
	ctx := context.Background()

	// Setup test state directory
	tmpDir := t.TempDir()
	setupTestStateDir(t, tmpDir)

	// Allocate a port
	require.NoError(t, state.AllocatePort(ctx, 25565))

	flags := &CreateFlags{
		Version: "1.21.1",
		Memory:  "2G",
		Port:    25565, // Try to use already allocated port
	}

	_, err := buildServerConfig(ctx, "testserver", flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already allocated")
}

func TestCreateServerDirectories(t *testing.T) {
	// Use temp directory for XDG_DATA_HOME
	tmpDir := t.TempDir()
	oldDataHome := os.Getenv("XDG_DATA_HOME")
	_ = os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_DATA_HOME", oldDataHome) }()

	name := "testserver"
	err := createServerDirectories(name)
	require.NoError(t, err)

	// Check directories exist
	serverDir := filepath.Join(tmpDir, "go-mc", "servers", name)
	dataDir := filepath.Join(serverDir, "data")
	modsDir := filepath.Join(serverDir, "mods")

	assert.DirExists(t, serverDir)
	assert.DirExists(t, dataDir)
	assert.DirExists(t, modsDir)
}

func TestBuildServerState(t *testing.T) {
	// Use temp directory for XDG_DATA_HOME
	tmpDir := t.TempDir()
	oldDataHome := os.Getenv("XDG_DATA_HOME")
	_ = os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_DATA_HOME", oldDataHome) }()

	config := &ServerConfig{
		Name:        "testserver",
		Version:     "1.21.1",
		Memory:      "2G",
		Port:        25565,
		RCONPort:    35565,
		RCONPass:    "testpassword123",
		ContainerID: "abc123def456",
	}

	serverState := buildServerState(config, "testserver")

	assert.Equal(t, "testserver", serverState.Name)
	assert.Equal(t, config.ContainerID, serverState.ContainerID)
	assert.Equal(t, containerImage, serverState.Image)
	assert.Equal(t, state.StatusStopped, serverState.Status)

	// Check Minecraft config
	assert.Equal(t, config.Version, serverState.Minecraft.Version)
	assert.Equal(t, config.Memory, serverState.Minecraft.Memory)
	assert.Equal(t, config.Port, serverState.Minecraft.GamePort)
	assert.Equal(t, config.RCONPort, serverState.Minecraft.RconPort)
	assert.Equal(t, config.RCONPass, serverState.Minecraft.RconPassword)

	// Check volumes
	expectedDataDir := filepath.Join(tmpDir, "go-mc", "servers", "testserver", "data")
	assert.Equal(t, expectedDataDir, serverState.Volumes.Data)
}

func TestShowDryRun(t *testing.T) {
	tests := []struct {
		name     string
		jsonMode bool
		config   *ServerConfig
		checkMsg string
	}{
		{
			name:     "human readable output",
			jsonMode: false,
			config: &ServerConfig{
				Name:     "testserver",
				Version:  "1.21.1",
				Memory:   "2G",
				Port:     25565,
				RCONPort: 35565,
			},
			checkMsg: "Dry run",
		},
		{
			name:     "json output",
			jsonMode: true,
			config: &ServerConfig{
				Name:     "testserver",
				Version:  "1.21.1",
				Memory:   "4G",
				Port:     25566,
				RCONPort: 35566,
				Mods:     []string{"fabric-api"},
			},
			checkMsg: "dry-run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			err := showDryRun(&buf, tt.jsonMode, tt.config)
			require.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output)
			assert.Contains(t, output, tt.checkMsg)

			if tt.jsonMode {
				// Check it's valid JSON
				assert.True(t, strings.HasPrefix(output, "{"))
			} else {
				// Check human-readable format
				assert.Contains(t, output, tt.config.Name)
				assert.Contains(t, output, tt.config.Version)
			}
		})
	}
}

func TestOutputSuccess(t *testing.T) {
	tests := []struct {
		name     string
		jsonMode bool
		config   *ServerConfig
		started  bool
		checkMsg string
	}{
		{
			name:     "human readable - not started",
			jsonMode: false,
			config: &ServerConfig{
				Name:        "testserver",
				Version:     "1.21.1",
				Memory:      "2G",
				Port:        25565,
				RCONPort:    35565,
				ContainerID: "abc123def456",
			},
			started:  false,
			checkMsg: "Created server",
		},
		{
			name:     "human readable - started",
			jsonMode: false,
			config: &ServerConfig{
				Name:        "testserver",
				Version:     "1.21.1",
				Memory:      "2G",
				Port:        25565,
				RCONPort:    35565,
				ContainerID: "abc123def456",
			},
			started:  true,
			checkMsg: "starting",
		},
		{
			name:     "json output",
			jsonMode: true,
			config: &ServerConfig{
				Name:        "testserver",
				Version:     "1.21.1",
				Memory:      "4G",
				Port:        25566,
				RCONPort:    35566,
				ContainerID: "xyz789abc123",
			},
			started:  false,
			checkMsg: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			err := outputSuccess(&buf, tt.jsonMode, tt.config, tt.started)
			require.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output)
			assert.Contains(t, output, tt.checkMsg)

			if tt.jsonMode {
				// Check it's valid JSON
				assert.True(t, strings.HasPrefix(output, "{"))
			}
		})
	}
}

func TestOutputError(t *testing.T) {
	tests := []struct {
		name     string
		jsonMode bool
		err      error
	}{
		{
			name:     "human readable error",
			jsonMode: false,
			err:      assert.AnError,
		},
		{
			name:     "json error",
			jsonMode: true,
			err:      assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			err := outputError(&buf, tt.jsonMode, tt.err)
			assert.Error(t, err)
			assert.Equal(t, tt.err, err)

			if tt.jsonMode {
				output := buf.String()
				assert.NotEmpty(t, output)
				assert.Contains(t, output, "error")
			}
		})
	}
}

func TestValidateServerName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid simple name",
			input:   "myserver",
			wantErr: false,
		},
		{
			name:    "valid name with hyphen",
			input:   "my-server",
			wantErr: false,
		},
		{
			name:    "valid name with numbers",
			input:   "server123",
			wantErr: false,
		},
		{
			name:    "valid mixed",
			input:   "my-server-123",
			wantErr: false,
		},
		{
			name:    "invalid - empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid - underscore",
			input:   "my_server",
			wantErr: true,
		},
		{
			name:    "invalid - space",
			input:   "my server",
			wantErr: true,
		},
		{
			name:    "invalid - special char",
			input:   "my@server",
			wantErr: true,
		},
		{
			name:    "invalid - starts with hyphen",
			input:   "-myserver",
			wantErr: true,
		},
		{
			name:    "invalid - ends with hyphen",
			input:   "myserver-",
			wantErr: true,
		},
		{
			name:    "invalid - too long",
			input:   strings.Repeat("a", 64),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := state.ValidateServerName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// setupTestStateDir sets up a temporary state directory for testing
func setupTestStateDir(t *testing.T, tmpDir string) {
	t.Helper()

	// Set environment variables to use temp directory
	configDir := filepath.Join(tmpDir, "config", "go-mc")
	_ = os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))

	// Create directories
	require.NoError(t, os.MkdirAll(configDir, 0750))
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "servers"), 0750))

	// Initialize global state
	ctx := context.Background()
	_, err := state.LoadGlobalState(ctx)
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		_ = os.Unsetenv("XDG_CONFIG_HOME")
	})
}
