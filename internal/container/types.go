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
