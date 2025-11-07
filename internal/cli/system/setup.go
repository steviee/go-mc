package system

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/state"
)

const (
	// Default container image to pre-pull
	defaultContainerImage = "docker.io/itzg/minecraft-server:latest"

	// Supported Debian versions
	debianMinVersion = 12
	debianMaxVersion = 13
)

var (
	// setupFlags holds all flags for the setup command
	setupNonInteractive bool
	setupSkipDeps       bool
	setupForce          bool
)

// Dependency represents a system dependency
type Dependency struct {
	Name      string
	Command   string
	Package   string
	Required  bool
	Installed bool
}

// SetupOutput holds the output for JSON mode
type SetupOutput struct {
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NewSetupCommand creates the system setup subcommand
func NewSetupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "First-time system setup",
		Long: `Perform first-time setup for go-mc.

This command performs the following setup steps:
  1. Check OS compatibility (Debian 12/13 only)
  2. Check and install dependencies (Podman, curl, git)
  3. Configure Podman for rootless operation
  4. Create directory structure (~/.config/go-mc, ~/.local/share/go-mc)
  5. Generate default configuration
  6. Initialize global state
  7. Pull default container image

Run this command once after installing go-mc to prepare your system.`,
		Example: `  # Interactive setup with prompts
  go-mc system setup

  # Non-interactive setup with defaults
  go-mc system setup --non-interactive

  # Skip dependency installation
  go-mc system setup --skip-deps

  # Force re-setup even if already initialized
  go-mc system setup --force`,
		RunE: runSetup,
	}

	// Add flags
	cmd.Flags().BoolVar(&setupNonInteractive, "non-interactive", false, "Skip prompts, use defaults")
	cmd.Flags().BoolVar(&setupSkipDeps, "skip-deps", false, "Skip dependency installation")
	cmd.Flags().BoolVar(&setupForce, "force", false, "Force setup even if already initialized")

	return cmd
}

// runSetup executes the setup command
func runSetup(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	jsonMode := isJSONMode()

	if !jsonMode {
		printHeader(stdout)
	}

	// 1. Check OS compatibility
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "Checking OS compatibility...")
	}
	if err := checkOSCompatibility(); err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("OS compatibility check failed: %w", err))
	}
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "✓ OS compatible (Debian 12/13)")
		_, _ = fmt.Fprintln(stdout)
	}

	// 2. Check if already initialized (unless --force)
	if !setupForce {
		initialized, err := isAlreadyInitialized()
		if err != nil {
			return outputError(stdout, jsonMode, fmt.Errorf("failed to check initialization status: %w", err))
		}
		if initialized {
			if !jsonMode {
				_, _ = fmt.Fprintln(stdout, "System is already initialized.")
				_, _ = fmt.Fprintln(stdout, "Use --force to re-run setup.")
			} else {
				output := SetupOutput{
					Status:  "already_initialized",
					Message: "System is already initialized",
				}
				return json.NewEncoder(stdout).Encode(output)
			}
			return nil
		}
	}

	// 3. Check dependencies
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "Checking dependencies...")
	}
	deps, err := checkDependencies()
	if err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("failed to check dependencies: %w", err))
	}

	missing := filterMissing(deps)
	if len(missing) > 0 {
		if !jsonMode {
			_, _ = fmt.Fprintln(stdout, "Missing dependencies:")
			for _, dep := range missing {
				_, _ = fmt.Fprintf(stdout, "  ✗ %s\n", dep.Name)
			}
			_, _ = fmt.Fprintln(stdout)
		}

		if !setupSkipDeps {
			if err := installDependencies(stdout, stderr, missing, setupNonInteractive); err != nil {
				return outputError(stdout, jsonMode, fmt.Errorf("failed to install dependencies: %w", err))
			}
			if !jsonMode {
				_, _ = fmt.Fprintln(stdout, "✓ Dependencies installed")
			}
		} else {
			if !jsonMode {
				_, _ = fmt.Fprintln(stdout, "Skipping dependency installation (--skip-deps)")
			}
		}
	} else {
		if !jsonMode {
			_, _ = fmt.Fprintln(stdout, "✓ All dependencies installed")
		}
	}
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout)
	}

	// 4. Configure Podman
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "Configuring Podman for rootless operation...")
	}
	if err := configurePodmanRootless(stdout, stderr); err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("failed to configure Podman: %w", err))
	}
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "✓ Podman configured")
		_, _ = fmt.Fprintln(stdout)
	}

	// 5. Create directories
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "Creating directory structure...")
	}
	if err := createDirectories(); err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("failed to create directories: %w", err))
	}
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "✓ Directories created")
		_, _ = fmt.Fprintln(stdout)
	}

	// 6. Generate config
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "Generating default configuration...")
	}
	if err := generateDefaultConfig(ctx, stdout, setupNonInteractive); err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("failed to generate config: %w", err))
	}
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "✓ Configuration created")
		_, _ = fmt.Fprintln(stdout)
	}

	// 7. Initialize state
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "Initializing global state...")
	}
	if err := initializeGlobalState(ctx); err != nil {
		return outputError(stdout, jsonMode, fmt.Errorf("failed to initialize state: %w", err))
	}
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "✓ State initialized")
		_, _ = fmt.Fprintln(stdout)
	}

	// 8. Pull container image
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "Pulling default container image...")
		_, _ = fmt.Fprintf(stdout, "This may take a few minutes...\n")
	}
	if err := pullContainerImage(ctx, stdout, stderr, defaultContainerImage); err != nil {
		// Don't fail completely on image pull error, just warn
		slog.Warn("failed to pull container image", "error", err)
		if !jsonMode {
			_, _ = fmt.Fprintf(stderr, "Warning: Failed to pull container image: %v\n", err)
			_, _ = fmt.Fprintln(stdout, "⚠ Container image pull failed (will retry on first server creation)")
		}
	} else {
		if !jsonMode {
			_, _ = fmt.Fprintln(stdout, "✓ Container image ready")
		}
	}
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout)
	}

	// Success message
	return outputSuccess(stdout, jsonMode)
}

// checkOSCompatibility checks if the OS is Debian 12 or 13
func checkOSCompatibility() error {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return fmt.Errorf("failed to read /etc/os-release: %w", err)
	}

	return checkOSCompatibilityFromString(string(data))
}

// checkOSCompatibilityFromString checks OS compatibility from os-release content
func checkOSCompatibilityFromString(osRelease string) error {
	lines := strings.Split(osRelease, "\n")
	isDebian := false
	version := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID=") {
			id := strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			if id == "debian" {
				isDebian = true
			}
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}

	if !isDebian {
		return fmt.Errorf("unsupported OS (requires Debian 12 or 13)")
	}

	// Parse version
	var versionNum int
	if _, err := fmt.Sscanf(version, "%d", &versionNum); err != nil {
		return fmt.Errorf("failed to parse Debian version: %w", err)
	}

	if versionNum < debianMinVersion || versionNum > debianMaxVersion {
		return fmt.Errorf("unsupported Debian version %d (requires 12 or 13)", versionNum)
	}

	return nil
}

// isAlreadyInitialized checks if the system is already initialized
func isAlreadyInitialized() (bool, error) {
	// Check if config file exists
	configPath, err := state.GetConfigPath()
	if err != nil {
		return false, fmt.Errorf("failed to get config path: %w", err)
	}

	if _, err := os.Stat(configPath); err == nil {
		return true, nil
	}

	// Check if state file exists
	statePath, err := state.GetStatePath()
	if err != nil {
		return false, fmt.Errorf("failed to get state path: %w", err)
	}

	if _, err := os.Stat(statePath); err == nil {
		return true, nil
	}

	return false, nil
}

// checkDependencies checks which dependencies are installed
func checkDependencies() ([]Dependency, error) {
	deps := []Dependency{
		{Name: "Podman", Command: "podman", Package: "podman", Required: true, Installed: false},
		{Name: "PolicyKit", Command: "pkaction", Package: "polkitd", Required: true, Installed: false},
		{Name: "curl", Command: "curl", Package: "curl", Required: false, Installed: false},
		{Name: "git", Command: "git", Package: "git", Required: false, Installed: false},
	}

	for i := range deps {
		_, err := exec.LookPath(deps[i].Command)
		deps[i].Installed = (err == nil)
	}

	return deps, nil
}

// filterMissing filters dependencies that are not installed and required
func filterMissing(deps []Dependency) []Dependency {
	var missing []Dependency
	for _, dep := range deps {
		if !dep.Installed && dep.Required {
			missing = append(missing, dep)
		}
	}
	return missing
}

// installDependencies installs missing dependencies using apt
func installDependencies(stdout, stderr io.Writer, deps []Dependency, nonInteractive bool) error {
	if len(deps) == 0 {
		return nil
	}

	// Confirm installation if interactive
	if !nonInteractive {
		_, _ = fmt.Fprintln(stdout, "The following packages will be installed:")
		for _, dep := range deps {
			_, _ = fmt.Fprintf(stdout, "  - %s\n", dep.Name)
		}
		if !confirm(stdout, "Continue?") {
			return fmt.Errorf("installation cancelled by user")
		}
	}

	// Update apt cache
	_, _ = fmt.Fprintln(stdout, "Updating package cache...")
	cmd := exec.Command("sudo", "apt-get", "update")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apt-get update failed: %w", err)
	}

	// Install packages
	_, _ = fmt.Fprintln(stdout, "Installing packages...")
	packages := []string{"apt-get", "install", "-y"}
	for _, dep := range deps {
		packages = append(packages, dep.Package)
	}

	cmd = exec.Command("sudo", packages...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("package installation failed: %w", err)
	}

	return nil
}

// configurePodmanRootless configures Podman for rootless operation
func configurePodmanRootless(stdout, stderr io.Writer) error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("USER environment variable not set")
	}

	// Enable user lingering (allows services to run when user is not logged in)
	_, _ = fmt.Fprintln(stdout, "  Enabling user lingering...")
	cmd := exec.Command("loginctl", "enable-linger", currentUser)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		// Non-fatal - lingering is nice-to-have but not required
		slog.Warn("failed to enable user lingering", "error", err)
		_, _ = fmt.Fprintf(stderr, "  Warning: Failed to enable lingering (non-fatal): %v\n", err)
	}

	// Start and enable podman.socket
	_, _ = fmt.Fprintln(stdout, "  Starting Podman socket...")
	cmd = exec.Command("systemctl", "--user", "enable", "--now", "podman.socket")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable podman.socket: %w", err)
	}

	// Wait a moment for socket to be ready
	// time.Sleep(1 * time.Second)

	// Verify Podman works
	_, _ = fmt.Fprintln(stdout, "  Verifying Podman installation...")
	cmd = exec.Command("podman", "version")
	cmd.Stdout = io.Discard // Don't clutter output
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("podman verification failed: %w", err)
	}

	return nil
}

// createDirectories creates all necessary directories
func createDirectories() error {
	// Get all directory paths
	configDir, err := state.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config dir: %w", err)
	}

	serversDir, err := state.GetServersDir()
	if err != nil {
		return fmt.Errorf("failed to get servers dir: %w", err)
	}

	whitelistsDir, err := state.GetWhitelistsDir()
	if err != nil {
		return fmt.Errorf("failed to get whitelists dir: %w", err)
	}

	archivesDir, err := state.GetArchivesDir()
	if err != nil {
		return fmt.Errorf("failed to get archives dir: %w", err)
	}

	// Also create XDG_DATA_HOME directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	dataDir := filepath.Join(dataHome, "go-mc")
	serverDataDir := filepath.Join(dataDir, "servers")
	backupsDataDir := filepath.Join(dataDir, "backups")

	// Create all directories
	dirs := []string{
		configDir,
		serversDir,
		whitelistsDir,
		archivesDir,
		dataDir,
		serverDataDir,
		backupsDataDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// generateDefaultConfig generates and saves the default configuration
func generateDefaultConfig(ctx context.Context, stdout io.Writer, nonInteractive bool) error {
	configPath, err := state.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		if !nonInteractive {
			if !confirm(stdout, "Config already exists. Overwrite?") {
				_, _ = fmt.Fprintln(stdout, "  Keeping existing configuration")
				return nil
			}
		} else {
			// Don't overwrite in non-interactive mode
			_, _ = fmt.Fprintln(stdout, "  Config already exists, keeping it")
			return nil
		}
	}

	// Create default config
	config := state.DefaultConfig()

	// Interactive prompts (if not non-interactive)
	if !nonInteractive {
		config = promptForConfig(stdout, config)
	}

	// Save config
	if err := state.SaveConfig(ctx, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// promptForConfig prompts user for configuration values
func promptForConfig(stdout io.Writer, config *state.Config) *state.Config {
	_, _ = fmt.Fprintln(stdout, "  Configuration setup (press Enter for defaults):")

	// Prompt for default memory
	_, _ = fmt.Fprintf(stdout, "  Default server memory [%s]: ", config.Defaults.Memory)
	if input := readLine(); input != "" {
		// Validate memory format
		if err := state.ValidateMemory(input); err != nil {
			_, _ = fmt.Fprintf(stdout, "    Invalid format, using default: %s\n", config.Defaults.Memory)
		} else {
			config.Defaults.Memory = input
		}
	}

	// Prompt for default Minecraft version
	_, _ = fmt.Fprintf(stdout, "  Default Minecraft version [%s]: ", config.Defaults.MinecraftVersion)
	if input := readLine(); input != "" {
		config.Defaults.MinecraftVersion = input
	}

	return config
}

// initializeGlobalState initializes the global state file
func initializeGlobalState(ctx context.Context) error {
	statePath, err := state.GetStatePath()
	if err != nil {
		return fmt.Errorf("failed to get state path: %w", err)
	}

	// Check if state already exists
	if _, err := os.Stat(statePath); err == nil {
		// Already initialized
		return nil
	}

	// Create new global state
	globalState := state.NewGlobalState()

	// Save state
	if err := state.SaveGlobalState(ctx, globalState); err != nil {
		return fmt.Errorf("failed to save global state: %w", err)
	}

	return nil
}

// pullContainerImage pulls the default container image
func pullContainerImage(ctx context.Context, stdout, stderr io.Writer, image string) error {
	// Try using the container client first (more reliable)
	client, err := container.NewClient(ctx, container.DefaultConfig())
	if err != nil {
		// Fallback to podman command if client fails
		slog.Debug("container client not available, using podman command", "error", err)
		return pullImageWithCommand(stdout, stderr, image)
	}
	defer func() { _ = client.Close() }()

	// Verify connection
	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to container runtime: %w", err)
	}

	// Pull image using podman command (for progress output)
	return pullImageWithCommand(stdout, stderr, image)
}

// pullImageWithCommand pulls an image using the podman command
func pullImageWithCommand(stdout, stderr io.Writer, image string) error {
	cmd := exec.Command("podman", "pull", image)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	return nil
}

// confirm prompts the user for confirmation
func confirm(stdout io.Writer, prompt string) bool {
	_, _ = fmt.Fprintf(stdout, "%s [y/N]: ", prompt)
	var response string
	_, _ = fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// readLine reads a line from stdin
func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// printHeader prints the setup header
func printHeader(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "go-mc System Setup")
	_, _ = fmt.Fprintln(stdout, "==================")
	_, _ = fmt.Fprintln(stdout)
}

// outputSuccess outputs a success message
func outputSuccess(stdout io.Writer, jsonMode bool) error {
	if jsonMode {
		output := SetupOutput{
			Status:  "success",
			Message: "Setup completed successfully",
			Data: map[string]interface{}{
				"next_steps": []string{
					"Create a server:  go-mc servers create myserver",
					"Start the server: go-mc servers start myserver",
					"View dashboard:   go-mc servers top",
				},
			},
		}
		return json.NewEncoder(stdout).Encode(output)
	}

	_, _ = fmt.Fprintln(stdout, "Setup completed successfully!")
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Next steps:")
	_, _ = fmt.Fprintln(stdout, "  Create a server:  go-mc servers create myserver")
	_, _ = fmt.Fprintln(stdout, "  Start the server: go-mc servers start myserver")
	_, _ = fmt.Fprintln(stdout, "  View dashboard:   go-mc servers top")

	return nil
}

// outputError outputs an error message
func outputError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := SetupOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}

// isJSONMode checks if --json flag is set (inherited from root command)
func isJSONMode() bool {
	// This would be set by the root command's persistent flag
	// For now, return false (JSON mode not implemented yet)
	return false
}
