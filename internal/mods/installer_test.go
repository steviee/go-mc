package mods

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/steviee/go-mc/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstaller(t *testing.T) {
	installer := NewInstaller()

	require.NotNil(t, installer)
	assert.NotNil(t, installer.modrinthClient)
	assert.NotNil(t, installer.httpClient)
}

func TestGetModsDir(t *testing.T) {
	tests := []struct {
		name        string
		serverState *state.ServerState
		want        string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid data volume",
			serverState: &state.ServerState{
				Volumes: state.VolumesConfig{
					Data: "/var/lib/go-mc/data/test-server",
				},
			},
			want:    "/var/lib/go-mc/data/mods",
			wantErr: false,
		},
		{
			name: "empty data volume",
			serverState: &state.ServerState{
				Volumes: state.VolumesConfig{
					Data: "",
				},
			},
			want:    "",
			wantErr: true,
			errMsg:  "server data volume not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getModsDir(tt.serverState)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIsModInstalled(t *testing.T) {
	installer := NewInstaller()

	tests := []struct {
		name        string
		serverState *state.ServerState
		slug        string
		want        bool
	}{
		{
			name: "mod is installed",
			serverState: &state.ServerState{
				Mods: []state.ModInfo{
					{
						Slug: "fabric-api",
						Name: "Fabric API",
					},
					{
						Slug: "lithium",
						Name: "Lithium",
					},
				},
			},
			slug: "fabric-api",
			want: true,
		},
		{
			name: "mod is not installed",
			serverState: &state.ServerState{
				Mods: []state.ModInfo{
					{
						Slug: "fabric-api",
						Name: "Fabric API",
					},
				},
			},
			slug: "lithium",
			want: false,
		},
		{
			name: "empty mods list",
			serverState: &state.ServerState{
				Mods: []state.ModInfo{},
			},
			slug: "fabric-api",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := installer.isModInstalled(tt.serverState, tt.slug)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDownloadFile(t *testing.T) {
	// Create a test HTTP server
	testContent := []byte("test mod file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testContent)
	}))
	defer server.Close()

	// Create temp directory for download
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-mod.jar")

	installer := NewInstaller()
	ctx := context.Background()

	err := installer.DownloadFile(ctx, server.URL, destPath)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, destPath)

	// Verify file content
	content, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestDownloadFile_HTTPError(t *testing.T) {
	// Create a test HTTP server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-mod.jar")

	installer := NewInstaller()
	ctx := context.Background()

	err := installer.DownloadFile(ctx, server.URL, destPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status")
}

func TestDownloadFile_ContextCanceled(t *testing.T) {
	// Create a test HTTP server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never respond
		select {}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-mod.jar")

	installer := NewInstaller()

	// Create context and cancel it immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := installer.DownloadFile(ctx, server.URL, destPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestDownloadFile_InvalidPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	installer := NewInstaller()
	ctx := context.Background()

	// Try to write to a directory that doesn't exist and can't be created
	destPath := "/nonexistent/directory/that/cannot/be/created/test-mod.jar"

	err := installer.DownloadFile(ctx, server.URL, destPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create file")
}

func TestInstallMods_UnknownMod(t *testing.T) {
	// Create temporary state directory
	tmpDir := t.TempDir()
	t.Setenv("GO_MC_STATE_DIR", tmpDir)

	// Create a test server state
	serverState := state.NewServerState("test-server")
	serverState.Minecraft.Version = "1.21.1"
	serverState.Volumes.Data = filepath.Join(tmpDir, "data")

	ctx := context.Background()
	err := state.SaveServerState(ctx, serverState)
	require.NoError(t, err)

	installer := NewInstaller()

	// Try to install a mod that doesn't exist in the database
	_, err = installer.InstallMods(ctx, "test-server", []string{"nonexistent-mod"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown mod")
}

func TestInstallMods_ServerNotFound(t *testing.T) {
	// Create temporary state directory
	tmpDir := t.TempDir()
	t.Setenv("GO_MC_STATE_DIR", tmpDir)

	installer := NewInstaller()
	ctx := context.Background()

	// Try to install mods for a server that doesn't exist
	_, err := installer.InstallMods(ctx, "nonexistent-server", []string{"fabric-api"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

// TestEnsureFabricAPI tests the EnsureFabricAPI method
// Note: This is a unit test that doesn't actually call the Modrinth API
func TestEnsureFabricAPI_AlreadyInstalled(t *testing.T) {
	// Create temporary state directory
	tmpDir := t.TempDir()
	t.Setenv("GO_MC_STATE_DIR", tmpDir)

	// Create a test server state with Fabric API already installed
	serverState := state.NewServerState("test-server")
	serverState.Minecraft.Version = "1.21.1"
	serverState.Volumes.Data = filepath.Join(tmpDir, "data")
	serverState.Mods = []state.ModInfo{
		{
			Slug: "fabric-api",
			Name: "Fabric API",
		},
	}

	ctx := context.Background()
	err := state.SaveServerState(ctx, serverState)
	require.NoError(t, err)

	installer := NewInstaller()

	// Should return without error since it's already installed
	err = installer.EnsureFabricAPI(ctx, "test-server")
	require.NoError(t, err)

	// Verify state wasn't modified
	updatedState, err := state.LoadServerState(ctx, "test-server")
	require.NoError(t, err)
	assert.Len(t, updatedState.Mods, 1)
	assert.Equal(t, "fabric-api", updatedState.Mods[0].Slug)
}

func BenchmarkIsModInstalled(b *testing.B) {
	installer := NewInstaller()

	// Create server state with many mods
	serverState := &state.ServerState{
		Mods: make([]state.ModInfo, 100),
	}

	for i := 0; i < 100; i++ {
		serverState.Mods[i] = state.ModInfo{
			Slug: filepath.Join("mod", string(rune(i))),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = installer.isModInstalled(serverState, "mod-50")
	}
}

func BenchmarkGetModsDir(b *testing.B) {
	serverState := &state.ServerState{
		Volumes: state.VolumesConfig{
			Data: "/var/lib/go-mc/data/test-server",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getModsDir(serverState)
	}
}

// Example_installMods demonstrates how to install mods using the Installer.
func Example_installMods() {
	ctx := context.Background()
	installer := NewInstaller()

	// Install a single mod (will also install dependencies)
	installed, err := installer.InstallMods(ctx, "my-server", []string{"lithium"})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Installed mods: %v\n", installed)
	// Output will include both fabric-api (dependency) and lithium
}

// Example_ensureFabricAPI demonstrates how to ensure Fabric API is installed.
func Example_ensureFabricAPI() {
	ctx := context.Background()
	installer := NewInstaller()

	// Ensure Fabric API is installed (opinionated default)
	if err := installer.EnsureFabricAPI(ctx, "my-server"); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Fabric API is installed")
}

// TestInstallMods_WithPortAllocation tests installing mods that require ports
func TestInstallMods_WithPortAllocation(t *testing.T) {
	// Create temporary state directory
	tmpDir := t.TempDir()
	t.Setenv("GO_MC_STATE_DIR", tmpDir)

	// Create a test server state
	serverState := state.NewServerState("test-server")
	serverState.Minecraft.Version = "1.21.1"
	serverState.Volumes.Data = filepath.Join(tmpDir, "data")

	ctx := context.Background()
	err := state.SaveServerState(ctx, serverState)
	require.NoError(t, err)

	// Initialize global state
	globalState := state.NewGlobalState()
	err = state.SaveGlobalState(ctx, globalState)
	require.NoError(t, err)

	// Note: This test will fail because it tries to actually download from Modrinth
	// and we don't have a mock for the Modrinth API client yet.
	// This is an integration test that should be run separately with network access.
	t.Skip("Skipping integration test that requires Modrinth API access")

	installer := NewInstaller()

	// Try to install simple-voice-chat which requires a port
	installed, err := installer.InstallMods(ctx, "test-server", []string{"simple-voice-chat"})
	require.NoError(t, err)
	assert.Contains(t, installed, "fabric-api")
	assert.Contains(t, installed, "simple-voice-chat")

	// Verify port was allocated
	updatedState, err := state.LoadServerState(ctx, "test-server")
	require.NoError(t, err)

	var voiceChatMod *state.ModInfo
	for i := range updatedState.Mods {
		if updatedState.Mods[i].Slug == "simple-voice-chat" {
			voiceChatMod = &updatedState.Mods[i]
			break
		}
	}

	require.NotNil(t, voiceChatMod)
	assert.Equal(t, 24454, voiceChatMod.Port)
	assert.Equal(t, "udp", voiceChatMod.Protocol)
}
