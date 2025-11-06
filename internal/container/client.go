package container

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/system"
)

// client implements the Client interface.
type client struct {
	conn       context.Context
	runtime    string
	socketPath string
	timeout    time.Duration
}

// NewClient creates a new container client with auto-detection.
//
// Runtime detection order:
//  1. If config.Runtime == "podman" or "docker", use that
//  2. If config.Runtime == "auto" (default):
//     a. Try Podman rootless socket
//     b. Try Podman rootful socket
//     c. Try Docker socket (if allowed by global config)
func NewClient(ctx context.Context, cfg *Config) (Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Set default timeout if not specified
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	slog.Info("initializing container client",
		"runtime", cfg.Runtime,
		"socket", cfg.SocketPath,
		"timeout", cfg.Timeout)

	// If explicit socket path provided, use it
	if cfg.SocketPath != "" {
		return connectToSocket(ctx, cfg.Runtime, cfg.SocketPath, cfg.Timeout)
	}

	// Auto-detect socket based on runtime preference
	switch cfg.Runtime {
	case "podman":
		return detectPodmanSocket(ctx, cfg.Timeout)
	case "docker":
		return detectDockerSocket(ctx, cfg.Timeout)
	case "auto":
		return autoDetectRuntime(ctx, cfg.Timeout)
	default:
		return nil, fmt.Errorf("invalid runtime: %q (must be podman, docker, or auto)", cfg.Runtime)
	}
}

// autoDetectRuntime tries to detect and connect to any available runtime.
func autoDetectRuntime(ctx context.Context, timeout time.Duration) (Client, error) {
	slog.Debug("auto-detecting container runtime")

	// Try Podman first (preferred)
	client, err := detectPodmanSocket(ctx, timeout)
	if err == nil {
		slog.Info("detected Podman runtime")
		return client, nil
	}

	slog.Debug("Podman not available, trying Docker", "error", err)

	// Try Docker as fallback
	client, err = detectDockerSocket(ctx, timeout)
	if err == nil {
		slog.Info("detected Docker runtime")
		return client, nil
	}

	// No runtime available
	return nil, fmt.Errorf("%w: %s", ErrNoRuntimeAvailable, getNoRuntimeMessage())
}

// detectPodmanSocket tries to find and connect to a Podman socket.
func detectPodmanSocket(ctx context.Context, timeout time.Duration) (Client, error) {
	// Try rootless socket first (preferred)
	if socket, err := getPodmanRootlessSocket(); err == nil {
		slog.Debug("trying Podman rootless socket", "socket", socket)
		client, err := connectToSocket(ctx, "podman", socket, timeout)
		if err == nil {
			return client, nil
		}
		slog.Debug("rootless socket connection failed", "error", err)
	}

	// Try rootful socket as fallback
	socket := getPodmanRootfulSocket()
	slog.Debug("trying Podman rootful socket", "socket", socket)
	client, err := connectToSocket(ctx, "podman", socket, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Podman: %w", err)
	}

	return client, nil
}

// detectDockerSocket tries to find and connect to a Docker socket.
func detectDockerSocket(ctx context.Context, timeout time.Duration) (Client, error) {
	socket := getDockerSocket()
	slog.Debug("trying Docker socket", "socket", socket)

	client, err := connectToSocket(ctx, "docker", socket, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker: %w", err)
	}

	return client, nil
}

// connectToSocket attempts to connect to a specific socket.
func connectToSocket(ctx context.Context, runtime, socketPath string, timeout time.Duration) (Client, error) {
	// Check if socket file exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil, NewRuntimeError(runtime, socketPath, ErrSocketNotFound)
	}

	// Check socket permissions
	if err := checkSocketPermissions(socketPath); err != nil {
		return nil, NewRuntimeError(runtime, socketPath, fmt.Errorf("%w: %v", ErrPermissionDenied, err))
	}

	// Create connection context with timeout
	connCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Connect to Podman API
	// The socket path needs to be in the format: unix:///path/to/socket
	socketURI := fmt.Sprintf("unix://%s", socketPath)
	conn, err := bindings.NewConnection(connCtx, socketURI)
	if err != nil {
		// Check if it's a connection timeout or daemon not running
		if os.IsTimeout(err) || strings.Contains(err.Error(), "connection refused") {
			return nil, NewRuntimeError(runtime, socketPath, fmt.Errorf("%w: %v", ErrDaemonNotRunning, err))
		}
		return nil, NewRuntimeError(runtime, socketPath, fmt.Errorf("failed to connect: %w", err))
	}

	// Create client instance
	c := &client{
		conn:       conn,
		runtime:    runtime,
		socketPath: socketPath,
		timeout:    timeout,
	}

	// Test connection with ping
	if err := c.Ping(ctx); err != nil {
		return nil, NewRuntimeError(runtime, socketPath, fmt.Errorf("connection test failed: %w", err))
	}

	slog.Info("connected to container runtime",
		"runtime", runtime,
		"socket", socketPath)

	return c, nil
}

// Ping tests the connection to the container daemon.
func (c *client) Ping(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Use Info as a ping - it's a lightweight operation to test connectivity
	_, err := system.Info(timeoutCtx, nil)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// Info returns daemon information.
func (c *client) Info(ctx context.Context) (*RuntimeInfo, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	info, err := system.Info(timeoutCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get info: %w", err)
	}

	// Determine if running rootless
	rootless := false
	if info.Host != nil {
		rootless = info.Host.Security.Rootless
	}

	// Get version information
	version := info.Version.Version
	apiVersion := info.Version.APIVersion

	// Get OS and architecture
	osInfo := ""
	arch := ""
	if info.Host != nil {
		osInfo = info.Host.OS
		arch = info.Host.Arch
	}

	return &RuntimeInfo{
		Runtime:    c.runtime,
		Version:    version,
		APIVersion: apiVersion,
		Rootless:   rootless,
		SocketPath: c.socketPath,
		OS:         osInfo,
		Arch:       arch,
	}, nil
}

// Close closes the client connection.
func (c *client) Close() error {
	// Podman bindings don't require explicit cleanup
	// The context handles the connection lifecycle
	slog.Debug("closing container client", "runtime", c.runtime)
	return nil
}

// Runtime returns the container runtime being used.
func (c *client) Runtime() string {
	return c.runtime
}

// getPodmanRootlessSocket returns the path to the Podman rootless socket.
func getPodmanRootlessSocket() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	uid, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		return "", fmt.Errorf("failed to parse UID: %w", err)
	}

	socket := fmt.Sprintf("/run/user/%d/podman/podman.sock", uid)
	return socket, nil
}

// getPodmanRootfulSocket returns the path to the Podman rootful socket.
func getPodmanRootfulSocket() string {
	return "/run/podman/podman.sock"
}

// getDockerSocket returns the path to the Docker socket.
func getDockerSocket() string {
	return "/var/run/docker.sock"
}

// checkSocketPermissions checks if the socket is accessible.
func checkSocketPermissions(socketPath string) error {
	// Try to connect to the socket
	conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
	if err != nil {
		// Check if it's a permission error
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied")
		}
		return err
	}

	// Close the connection (we only needed to test connectivity)
	if err := conn.Close(); err != nil {
		return fmt.Errorf("failed to close test connection: %w", err)
	}

	return nil
}

// getNoRuntimeMessage returns a user-friendly message when no runtime is available.
func getNoRuntimeMessage() string {
	return `No container runtime available.

RECOMMENDED: Run the automated setup:
  go-mc system setup

This will install and configure Podman, polkitd, and all required dependencies.

MANUAL INSTALLATION (if needed):
  Debian/Ubuntu: sudo apt install podman policykit-1
  For other systems: https://podman.io/getting-started/installation

After manual installation, enable and start the Podman socket:
  systemctl --user enable --now podman.socket`
}

// GetPermissionDeniedMessage returns a user-friendly message for permission errors.
func GetPermissionDeniedMessage(runtime, socket string) string {
	if runtime == "podman" {
		if strings.Contains(socket, "/run/user/") {
			return fmt.Sprintf(`Permission denied accessing Podman rootless socket at %s.
This should not happen for rootless Podman. Please check:
  1. Podman is installed: podman --version
  2. Socket is running: systemctl --user status podman.socket
  3. Try restarting: systemctl --user restart podman.socket

If the issue persists, please report a bug.`, socket)
		}
		return fmt.Sprintf(`Permission denied accessing Podman rootful socket at %s.
To use rootful Podman, add your user to the podman group:
  sudo usermod -aG podman $USER

Then log out and log back in for the changes to take effect.

Alternatively, use rootless Podman (recommended):
  systemctl --user enable --now podman.socket`, socket)
	}

	// Docker
	return fmt.Sprintf(`Permission denied accessing Docker socket at %s.
To use Docker, add your user to the docker group:
  sudo usermod -aG docker $USER

Then log out and log back in for the changes to take effect.

Note: Rootless Podman is recommended instead of Docker.`, socket)
}

// GetDaemonNotRunningMessage returns a user-friendly message when the daemon is not running.
func GetDaemonNotRunningMessage(runtime string) string {
	if runtime == "podman" {
		return `Container daemon is not running. Start it with:
  Podman (rootless): systemctl --user start podman.socket
  Podman (rootful):  sudo systemctl start podman.socket

To enable automatic startup:
  Podman (rootless): systemctl --user enable podman.socket
  Podman (rootful):  sudo systemctl enable podman.socket`
	}

	// Docker
	return `Docker daemon is not running. Start it with:
  sudo systemctl start docker

To enable automatic startup:
  sudo systemctl enable docker`
}
