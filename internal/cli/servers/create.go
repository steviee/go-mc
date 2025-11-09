package servers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/mods"
	"github.com/steviee/go-mc/internal/state"
)

const (
	// Default Minecraft version (hardcoded for now as per requirements)
	defaultMinecraftVersion = "1.21.1"

	// Default memory allocation
	defaultMemory = "2G"

	// Default starting port
	defaultStartPort = 25565

	// RCON port offset from game port
	rconPortOffset = 10000

	// Container image
	containerImage = "itzg/minecraft-server:latest"
)

// CreateFlags holds all flags for the create command
type CreateFlags struct {
	Version       string
	Memory        string
	Port          int
	Mods          []string
	Start         bool
	DryRun        bool
	WithLithium   bool
	WithVoiceChat bool
	WithGeyser    bool
	WithBlueMap   bool
}

// ServerConfig holds the configuration for creating a server
type ServerConfig struct {
	Name        string
	Version     string
	Memory      string
	Port        int
	RCONPort    int
	Mods        []string
	RCONPass    string
	ContainerID string
}

// CreateOutput holds the output for JSON mode
type CreateOutput struct {
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NewCreateCommand creates the servers create subcommand
func NewCreateCommand() *cobra.Command {
	flags := &CreateFlags{}

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new Minecraft Fabric server",
		Long: `Create a new Minecraft Fabric server with smart defaults.

The server will be created in a Podman container with the itzg/minecraft-server image.
All configuration is stored in YAML files under ~/.config/go-mc/.

Smart defaults (Omakase principle):
  - Minecraft version: 1.21.1 (latest stable)
  - Memory: 2G
  - Port: Auto-allocated starting from 25565
  - Fabric: Latest compatible version
  - RCON: Auto-generated secure password

The server is created in a stopped state. Use --start to start it immediately.`,
		Example: `  # Create a server with defaults (includes Fabric API automatically)
  go-mc servers create myserver

  # Create with specific version and memory
  go-mc servers create myserver --version 1.20.4 --memory 4G

  # Create with performance optimization
  go-mc servers create myserver --with-lithium

  # Create with voice chat support (allocates UDP port 24454)
  go-mc servers create myserver --with-voice-chat

  # Create with Bedrock support and web map (allocates ports automatically)
  go-mc servers create myserver --with-geyser --with-bluemap

  # Create with multiple mods and start immediately
  go-mc servers create myserver --with-lithium --with-voice-chat --start

  # Preview configuration without creating
  go-mc servers create myserver --dry-run

  # Create with custom mods via slug
  go-mc servers create myserver --mods sodium,phosphor`,
		Args: requireServerName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0], flags)
		},
	}

	// Add flags
	cmd.Flags().StringVar(&flags.Version, "version", defaultMinecraftVersion, "Minecraft version")
	cmd.Flags().StringVar(&flags.Memory, "memory", defaultMemory, "RAM allocation (e.g., 2G, 4G, 512M)")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Server port (default: auto-allocate from 25565)")
	cmd.Flags().StringSliceVar(&flags.Mods, "mods", []string{}, "Comma-separated mod slugs for initial installation")
	cmd.Flags().BoolVar(&flags.Start, "start", false, "Start server immediately after creation")
	cmd.Flags().BoolVar(&flags.DryRun, "dry-run", false, "Show configuration without creating")
	cmd.Flags().BoolVar(&flags.WithLithium, "with-lithium", false, "Install Lithium (performance optimization)")
	cmd.Flags().BoolVar(&flags.WithVoiceChat, "with-voice-chat", false, "Install Simple Voice Chat (proximity voice)")
	cmd.Flags().BoolVar(&flags.WithGeyser, "with-geyser", false, "Install Geyser (Bedrock client support)")
	cmd.Flags().BoolVar(&flags.WithBlueMap, "with-bluemap", false, "Install BlueMap (3D web map)")

	return cmd
}

// runCreate executes the create command
func runCreate(ctx context.Context, stdout, stderr io.Writer, name string, flags *CreateFlags) error {
	jsonMode := isJSONMode()

	// Validate server name
	if err := state.ValidateServerName(name); err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("invalid server name: %w", err))
	}

	// Check if server already exists
	exists, err := state.ServerExists(ctx, name)
	if err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("failed to check if server exists: %w", err))
	}
	if exists {
		return outputError(stdout, jsonMode, fmt.Errorf("server %q already exists", name))
	}

	// Validate and build configuration
	config, err := buildServerConfig(ctx, name, flags)
	if err != nil {
		return outputError(stdout, jsonMode, err)
	}

	// If dry-run, just show configuration
	if flags.DryRun {
		return showDryRun(stdout, jsonMode, config)
	}

	// Initialize directories
	if err := state.InitDirs(); err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("failed to initialize directories: %w", err))
	}

	// Create data directories for server
	if err := createServerDirectories(name); err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("failed to create server directories: %w", err))
	}

	// Create container
	containerClient, err := container.NewClient(ctx, container.DefaultConfig())
	if err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("failed to create container client: %w", err))
	}
	defer func() { _ = containerClient.Close() }()

	containerID, err := createContainer(ctx, containerClient, config, name)
	if err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("failed to create container: %w", err))
	}

	config.ContainerID = containerID

	// Allocate ports in global state
	if err := allocatePorts(ctx, config.Port, config.RCONPort); err != nil {
		// Cleanup container on failure
		_ = containerClient.RemoveContainer(ctx, containerID, &container.RemoveOptions{Force: true})
		return outputError(stdout, jsonMode, fmt.Errorf("failed to allocate ports: %w", err))
	}

	// Register server in global state
	if err := state.RegisterServer(ctx, name); err != nil {
		// Cleanup on failure
		_ = state.ReleasePort(ctx, config.Port)
		_ = state.ReleasePort(ctx, config.RCONPort)
		_ = containerClient.RemoveContainer(ctx, containerID, &container.RemoveOptions{Force: true})
		return outputError(stdout, jsonMode, fmt.Errorf("failed to register server: %w", err))
	}

	// Save server state
	serverState := buildServerState(config, name)
	if err := state.SaveServerState(ctx, serverState); err != nil {
		// Cleanup on failure
		_ = state.UnregisterServer(ctx, name)
		_ = state.ReleasePort(ctx, config.Port)
		_ = state.ReleasePort(ctx, config.RCONPort)
		_ = containerClient.RemoveContainer(ctx, containerID, &container.RemoveOptions{Force: true})
		return outputError(stdout, jsonMode, fmt.Errorf("failed to save server state: %w", err))
	}

	// Install mods if requested
	if err := installModsIfRequested(ctx, name, flags, stdout, stderr, jsonMode); err != nil {
		// Don't fail completely, just log the error
		slog.Warn("failed to install mods", "error", err)
		if !jsonMode {
			_, _ = fmt.Fprintf(stderr, "Warning: Failed to install mods: %v\n", err)
		}
	}

	// Start container if requested
	if flags.Start {
		if err := containerClient.StartContainer(ctx, containerID); err != nil {
			// Don't fail completely, just log the error
			slog.Warn("failed to start container", "error", err)
			if !jsonMode {
				_, _ = fmt.Fprintf(stderr, "Warning: Failed to start container: %v\n", err)
			}
		} else {
			// Update server status
			if err := state.UpdateServerStatus(ctx, name, state.StatusRunning); err != nil {
				slog.Warn("failed to update server status", "error", err)
			}
			serverState.Status = state.StatusRunning
		}
	}

	// Output success
	return outputSuccess(stdout, jsonMode, config, flags.Start)
}

// buildServerConfig builds and validates the server configuration
func buildServerConfig(ctx context.Context, name string, flags *CreateFlags) (*ServerConfig, error) {
	config := &ServerConfig{
		Name:    name,
		Version: flags.Version,
		Memory:  flags.Memory,
		Mods:    flags.Mods,
	}

	// Validate version
	if err := state.ValidateVersion(config.Version); err != nil {
		return nil, fmt.Errorf("invalid version: %w", err)
	}

	// Validate memory
	if err := state.ValidateMemory(config.Memory); err != nil {
		return nil, fmt.Errorf("invalid memory format: %w", err)
	}

	// Allocate port
	if flags.Port != 0 {
		// Use specified port
		if err := state.ValidatePort(flags.Port); err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}

		// Check if port is already allocated
		allocated, err := state.IsPortAllocated(ctx, flags.Port)
		if err != nil {
			return nil, fmt.Errorf("failed to check port allocation: %w", err)
		}
		if allocated {
			return nil, fmt.Errorf("port %d is already allocated", flags.Port)
		}

		config.Port = flags.Port
	} else {
		// Auto-allocate port
		port, err := state.GetNextAvailablePort(ctx, defaultStartPort)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate port: %w", err)
		}
		config.Port = port
	}

	// Calculate RCON port
	config.RCONPort = config.Port + rconPortOffset

	// Validate RCON port
	if err := state.ValidatePort(config.RCONPort); err != nil {
		return nil, fmt.Errorf("invalid RCON port %d (calculated from game port): %w", config.RCONPort, err)
	}

	// Check if RCON port is already allocated
	allocated, err := state.IsPortAllocated(ctx, config.RCONPort)
	if err != nil {
		return nil, fmt.Errorf("failed to check RCON port allocation: %w", err)
	}
	if allocated {
		return nil, fmt.Errorf("RCON port %d is already allocated (calculated from game port %d)", config.RCONPort, config.Port)
	}

	// Generate RCON password
	config.RCONPass = generateRCONPassword()

	return config, nil
}

// generateRCONPassword generates a secure random password for RCON
func generateRCONPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 16

	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a simpler method if crypto/rand fails
		slog.Warn("failed to generate random password with crypto/rand", "error", err)
		// This should rarely happen, but we provide a fallback
		return "gomc" + fmt.Sprintf("%d", os.Getpid())
	}

	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}

	return string(b)
}

// createServerDirectories creates the data directories for a server
func createServerDirectories(name string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Use XDG_DATA_HOME or default to ~/.local/share
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	// Create server directories
	serverDir := filepath.Join(dataHome, "go-mc", "servers", name)
	dataDir := filepath.Join(serverDir, "data")
	modsDir := filepath.Join(serverDir, "mods")

	for _, dir := range []string{dataDir, modsDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// createContainer creates the container with the given configuration
func createContainer(ctx context.Context, client container.Client, config *ServerConfig, name string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Use XDG_DATA_HOME or default to ~/.local/share
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	dataDir := filepath.Join(dataHome, "go-mc", "servers", name, "data")

	containerConfig := &container.ContainerConfig{
		Name:  name,
		Image: containerImage,
		Env: map[string]string{
			"TYPE":          "FABRIC",
			"EULA":          "TRUE",
			"VERSION":       config.Version,
			"MEMORY":        config.Memory,
			"RCON_PASSWORD": config.RCONPass,
			"ENABLE_RCON":   "true",
		},
		Ports: map[int]int{
			config.Port:     25565, // Game port
			config.RCONPort: 25575, // RCON port
		},
		Volumes: map[string]string{
			dataDir: "/data",
		},
		Labels: map[string]string{
			"go-mc.server":  name,
			"go-mc.version": config.Version,
			"go-mc.managed": "true",
		},
	}

	containerID, err := client.CreateContainer(ctx, containerConfig)
	if err != nil {
		return "", err
	}

	slog.Info("container created",
		"name", name,
		"id", containerID,
		"port", config.Port,
		"rcon_port", config.RCONPort)

	return containerID, nil
}

// allocatePorts allocates the game port and RCON port in global state
func allocatePorts(ctx context.Context, gamePort, rconPort int) error {
	if err := state.AllocatePort(ctx, gamePort); err != nil {
		return fmt.Errorf("failed to allocate game port: %w", err)
	}

	if err := state.AllocatePort(ctx, rconPort); err != nil {
		// Cleanup game port allocation
		_ = state.ReleasePort(ctx, gamePort)
		return fmt.Errorf("failed to allocate RCON port: %w", err)
	}

	return nil
}

// buildServerState builds the server state from configuration
func buildServerState(config *ServerConfig, name string) *state.ServerState {
	serverState := state.NewServerState(name)

	serverState.ContainerID = config.ContainerID
	serverState.Image = containerImage
	serverState.Status = state.StatusStopped

	serverState.Minecraft = state.MinecraftConfig{
		Version:      config.Version,
		Memory:       config.Memory,
		GamePort:     config.Port,
		RconPort:     config.RCONPort,
		RconPassword: config.RCONPass,
	}

	homeDir, _ := os.UserHomeDir()
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	serverState.Volumes = state.VolumesConfig{
		Data:    filepath.Join(dataHome, "go-mc", "servers", name, "data"),
		Backups: filepath.Join(dataHome, "go-mc", "servers", name, "backups"),
	}

	return serverState
}

// showDryRun displays the configuration without creating the server
func showDryRun(stdout io.Writer, jsonMode bool, config *ServerConfig) error {
	if jsonMode {
		output := CreateOutput{
			Status: "dry-run",
			Data: map[string]interface{}{
				"name":      config.Name,
				"version":   config.Version,
				"port":      config.Port,
				"rcon_port": config.RCONPort,
				"memory":    config.Memory,
				"mods":      config.Mods,
			},
			Message: "Dry run - no changes made",
		}
		return json.NewEncoder(stdout).Encode(output)
	}

	_, _ = fmt.Fprintf(stdout, "Dry run - Configuration preview:\n\n")
	_, _ = fmt.Fprintf(stdout, "  Name:        %s\n", config.Name)
	_, _ = fmt.Fprintf(stdout, "  Version:     %s (Fabric)\n", config.Version)
	_, _ = fmt.Fprintf(stdout, "  Port:        %d\n", config.Port)
	_, _ = fmt.Fprintf(stdout, "  RCON Port:   %d\n", config.RCONPort)
	_, _ = fmt.Fprintf(stdout, "  Memory:      %s\n", config.Memory)
	_, _ = fmt.Fprintf(stdout, "  Container:   %s\n", containerImage)

	if len(config.Mods) > 0 {
		_, _ = fmt.Fprintf(stdout, "  Mods:        %s\n", strings.Join(config.Mods, ", "))
	}

	_, _ = fmt.Fprintf(stdout, "\nNo changes made. Remove --dry-run to create the server.\n")

	return nil
}

// outputSuccess outputs a success message
func outputSuccess(stdout io.Writer, jsonMode bool, config *ServerConfig, started bool) error {
	status := "created"
	if started {
		status = "running"
	}

	if jsonMode {
		output := CreateOutput{
			Status: "success",
			Data: map[string]interface{}{
				"name":         config.Name,
				"version":      config.Version,
				"port":         config.Port,
				"rcon_port":    config.RCONPort,
				"memory":       config.Memory,
				"container_id": config.ContainerID,
				"state":        status,
			},
		}
		return json.NewEncoder(stdout).Encode(output)
	}

	_, _ = fmt.Fprintf(stdout, "Created server '%s'\n", config.Name)
	_, _ = fmt.Fprintf(stdout, "  Minecraft: %s (Fabric)\n", config.Version)
	_, _ = fmt.Fprintf(stdout, "  Port:      %d\n", config.Port)
	_, _ = fmt.Fprintf(stdout, "  RCON:      %d\n", config.RCONPort)
	_, _ = fmt.Fprintf(stdout, "  Memory:    %s\n", config.Memory)
	_, _ = fmt.Fprintf(stdout, "  Container: %s\n", config.ContainerID[:12])

	if started {
		_, _ = fmt.Fprintf(stdout, "\nServer is starting...\n")
	} else {
		_, _ = fmt.Fprintf(stdout, "\nRun 'go-mc servers start %s' to start the server.\n", config.Name)
	}

	return nil
}

// outputError outputs an error message
func outputError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := CreateOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}

// installModsIfRequested installs mods based on flags
func installModsIfRequested(ctx context.Context, serverName string, flags *CreateFlags, stdout, stderr io.Writer, jsonMode bool) error {
	installer := mods.NewInstaller()

	// Always ensure Fabric API is installed (Omakase principle)
	if err := installer.EnsureFabricAPI(ctx, serverName); err != nil {
		return fmt.Errorf("ensure Fabric API: %w", err)
	}

	// Build list of mods to install
	modsToInstall := make([]string, 0)

	// Add mods from --with-* flags
	if flags.WithLithium {
		modsToInstall = append(modsToInstall, "lithium")
	}
	if flags.WithVoiceChat {
		modsToInstall = append(modsToInstall, "simple-voice-chat")
	}
	if flags.WithGeyser {
		modsToInstall = append(modsToInstall, "geyser")
	}
	if flags.WithBlueMap {
		modsToInstall = append(modsToInstall, "bluemap")
	}

	// Add mods from --mods flag
	modsToInstall = append(modsToInstall, flags.Mods...)

	// Install mods if any requested
	if len(modsToInstall) > 0 {
		if !jsonMode {
			_, _ = fmt.Fprintf(stdout, "\nInstalling mods...\n")
		}

		installed, err := installer.InstallMods(ctx, serverName, modsToInstall)
		if err != nil {
			return fmt.Errorf("install mods: %w", err)
		}

		if !jsonMode && len(installed) > 0 {
			_, _ = fmt.Fprintf(stdout, "Installed %d mod(s): %s\n", len(installed), strings.Join(installed, ", "))
		}
	}

	return nil
}
