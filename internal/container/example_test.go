package container_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/steviee/go-mc/internal/container"
)

// Example demonstrates basic usage of the container client.
func Example() {
	ctx := context.Background()

	// Create client with auto-detection
	client, err := container.NewClient(ctx, &container.Config{
		Runtime: "auto",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("failed to close client: %v", err)
		}
	}()

	// Test connection
	if err := client.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	// Get runtime information
	info, err := client.Info(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Runtime: %s\n", info.Runtime)
	fmt.Printf("Version: %s\n", info.Version)
	fmt.Printf("Rootless: %v\n", info.Rootless)
}

// ExampleNewClient_auto demonstrates auto-detection of container runtime.
func ExampleNewClient_auto() {
	ctx := context.Background()

	client, err := container.NewClient(ctx, &container.Config{
		Runtime: "auto",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = client.Close()
	}()

	fmt.Printf("Connected to: %s\n", client.Runtime())
}

// ExampleNewClient_podman demonstrates explicit Podman connection.
func ExampleNewClient_podman() {
	ctx := context.Background()

	client, err := container.NewClient(ctx, &container.Config{
		Runtime: "podman",
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = client.Close()
	}()

	info, err := client.Info(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Podman Version: %s\n", info.Version)
	fmt.Printf("Rootless: %v\n", info.Rootless)
}

// ExampleNewClient_explicitSocket demonstrates using an explicit socket path.
func ExampleNewClient_explicitSocket() {
	ctx := context.Background()

	client, err := container.NewClient(ctx, &container.Config{
		Runtime:    "podman",
		SocketPath: "/run/user/1000/podman/podman.sock",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = client.Close()
	}()

	fmt.Println("Connected to explicit socket")
}

// ExampleClient_Ping demonstrates testing the connection.
func ExampleClient_Ping() {
	ctx := context.Background()

	client, err := container.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = client.Close()
	}()

	// Test connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connection successful")
}

// ExampleClient_Info demonstrates getting runtime information.
func ExampleClient_Info() {
	ctx := context.Background()

	client, err := container.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = client.Close()
	}()

	info, err := client.Info(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Runtime: %s %s\n", info.Runtime, info.Version)
	fmt.Printf("API Version: %s\n", info.APIVersion)
	fmt.Printf("OS/Arch: %s/%s\n", info.OS, info.Arch)
	fmt.Printf("Socket: %s\n", info.SocketPath)
	fmt.Printf("Rootless: %v\n", info.Rootless)
}

// ExampleMockClient demonstrates using the mock client for testing.
func ExampleMockClient() {
	// Create a mock client for testing
	mock := &container.MockClient{
		PingFunc: func(ctx context.Context) error {
			return nil
		},
		InfoFunc: func(ctx context.Context) (*container.RuntimeInfo, error) {
			return &container.RuntimeInfo{
				Runtime:    "podman",
				Version:    "5.0.0",
				APIVersion: "1.41",
				Rootless:   true,
				SocketPath: "/run/user/1000/podman/podman.sock",
				OS:         "linux",
				Arch:       "amd64",
			}, nil
		},
		RuntimeFunc: func() string {
			return "podman"
		},
	}

	// Use mock in tests
	ctx := context.Background()
	if err := mock.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	info, err := mock.Info(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Mock Runtime: %s\n", info.Runtime)
	// Output: Mock Runtime: podman
}
