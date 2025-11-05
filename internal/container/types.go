package container

import (
	"context"
	"time"
)

// Client interface for container operations.
type Client interface {
	// Ping tests the connection to the container daemon.
	Ping(ctx context.Context) error

	// Info returns daemon information.
	Info(ctx context.Context) (*RuntimeInfo, error)

	// Close closes the client connection.
	Close() error

	// Runtime returns the container runtime being used (podman or docker).
	Runtime() string

	// CreateContainer creates a new container with the given configuration.
	CreateContainer(ctx context.Context, config *ContainerConfig) (string, error)

	// StartContainer starts a stopped container.
	StartContainer(ctx context.Context, containerID string) error

	// WaitForContainer waits for a container to reach the specified condition.
	WaitForContainer(ctx context.Context, containerID string, condition string) error

	// StopContainer stops a running container with optional timeout.
	StopContainer(ctx context.Context, containerID string, timeout *time.Duration) error

	// RestartContainer restarts a container with optional timeout.
	RestartContainer(ctx context.Context, containerID string, timeout *time.Duration) error

	// RemoveContainer removes a container.
	RemoveContainer(ctx context.Context, containerID string, opts *RemoveOptions) error

	// InspectContainer returns detailed information about a container.
	InspectContainer(ctx context.Context, containerID string) (*ContainerInfo, error)

	// ListContainers lists containers based on the provided options.
	ListContainers(ctx context.Context, opts *ListOptions) ([]*ContainerInfo, error)
}

// RuntimeInfo contains information about the container runtime.
type RuntimeInfo struct {
	Runtime    string // "podman" or "docker"
	Version    string // e.g., "4.5.0"
	APIVersion string // e.g., "1.41"
	Rootless   bool   // Whether running rootless
	SocketPath string // Path to Unix socket
	OS         string // Operating system
	Arch       string // Architecture
}

// Config for client initialization.
type Config struct {
	Runtime    string        // "podman", "docker", or "auto"
	SocketPath string        // Optional: explicit socket path
	Timeout    time.Duration // Default: 30s
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Runtime:    "auto",
		SocketPath: "",
		Timeout:    30 * time.Second,
	}
}

// ContainerConfig defines the configuration for creating a container.
type ContainerConfig struct {
	Name       string            // Container name
	Image      string            // Container image (e.g., "docker.io/library/alpine:latest")
	Env        map[string]string // Environment variables
	Ports      map[int]int       // Port mappings: hostPort:containerPort
	Volumes    map[string]string // Volume mounts: hostPath:containerPath
	Memory     string            // Memory limit (e.g., "2G", "512M")
	CPUQuota   int64             // CPU quota in microseconds
	WorkingDir string            // Working directory inside container
	Command    []string          // Command to run
	Labels     map[string]string // Container labels
}

// ContainerInfo contains information about a container.
type ContainerInfo struct {
	ID      string            // Container ID
	Name    string            // Container name
	State   string            // Container state (running, stopped, paused, exited)
	Status  string            // Human-readable status
	Image   string            // Image name
	Ports   map[int]int       // Port mappings
	Created time.Time         // Creation time
	Labels  map[string]string // Container labels
}

// RemoveOptions specifies options for removing a container.
type RemoveOptions struct {
	Force         bool // Force removal even if running
	RemoveVolumes bool // Remove associated volumes
}

// ListOptions specifies options for listing containers.
type ListOptions struct {
	All    bool              // Include stopped containers
	Limit  int               // Maximum number of containers to return
	Filter map[string]string // Filters (e.g., {"label": "go-mc"})
}
