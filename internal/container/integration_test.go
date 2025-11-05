// go:build integration
//go:build integration

package container

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPodmanClient_RealConnection tests connecting to a real Podman daemon.
// This test is skipped in short mode and requires Podman to be running.
func TestPodmanClient_RealConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("auto detection", func(t *testing.T) {
		client, err := NewClient(ctx, &Config{
			Runtime: "auto",
			Timeout: 5 * time.Second,
		})
		if err != nil {
			t.Skipf("no container runtime available: %v", err)
			return
		}
		defer client.Close()

		// Test ping
		err = client.Ping(ctx)
		require.NoError(t, err)

		// Test info
		info, err := client.Info(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, info.Runtime)
		assert.NotEmpty(t, info.Version)
		assert.NotEmpty(t, info.SocketPath)

		t.Logf("Connected to %s %s (%s)", info.Runtime, info.Version,
			map[bool]string{true: "rootless", false: "rootful"}[info.Rootless])
	})

	t.Run("explicit podman", func(t *testing.T) {
		client, err := NewClient(ctx, &Config{
			Runtime: "podman",
			Timeout: 5 * time.Second,
		})
		if err != nil {
			t.Skipf("Podman not available: %v", err)
			return
		}
		defer client.Close()

		assert.Equal(t, "podman", client.Runtime())

		info, err := client.Info(ctx)
		require.NoError(t, err)
		assert.Equal(t, "podman", info.Runtime)
		assert.NotEmpty(t, info.Version)
		assert.NotEmpty(t, info.APIVersion)
	})

	t.Run("multiple pings", func(t *testing.T) {
		client, err := NewClient(ctx, &Config{
			Runtime: "auto",
			Timeout: 5 * time.Second,
		})
		if err != nil {
			t.Skipf("no container runtime available: %v", err)
			return
		}
		defer client.Close()

		// Multiple pings should work
		for i := 0; i < 5; i++ {
			err = client.Ping(ctx)
			require.NoError(t, err)
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		client, err := NewClient(ctx, &Config{
			Runtime: "auto",
			Timeout: 5 * time.Second,
		})
		if err != nil {
			t.Skipf("no container runtime available: %v", err)
			return
		}
		defer client.Close()

		// Create a context with a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait a bit to ensure context expires
		time.Sleep(1 * time.Millisecond)

		// This should fail with context deadline exceeded
		err = client.Ping(ctx)
		assert.Error(t, err)
	})

	t.Run("concurrent operations", func(t *testing.T) {
		client, err := NewClient(ctx, &Config{
			Runtime: "auto",
			Timeout: 5 * time.Second,
		})
		if err != nil {
			t.Skipf("no container runtime available: %v", err)
			return
		}
		defer client.Close()

		// Run multiple concurrent operations
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				err := client.Ping(ctx)
				assert.NoError(t, err)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("runtime info details", func(t *testing.T) {
		client, err := NewClient(ctx, &Config{
			Runtime: "auto",
			Timeout: 5 * time.Second,
		})
		if err != nil {
			t.Skipf("no container runtime available: %v", err)
			return
		}
		defer client.Close()

		info, err := client.Info(ctx)
		require.NoError(t, err)

		// Verify all fields are populated
		assert.NotEmpty(t, info.Runtime, "Runtime should not be empty")
		assert.NotEmpty(t, info.Version, "Version should not be empty")
		assert.NotEmpty(t, info.APIVersion, "APIVersion should not be empty")
		assert.NotEmpty(t, info.SocketPath, "SocketPath should not be empty")
		assert.NotEmpty(t, info.OS, "OS should not be empty")
		assert.NotEmpty(t, info.Arch, "Arch should not be empty")

		// Log details for debugging
		t.Logf("Runtime Details:")
		t.Logf("  Runtime:    %s", info.Runtime)
		t.Logf("  Version:    %s", info.Version)
		t.Logf("  APIVersion: %s", info.APIVersion)
		t.Logf("  Rootless:   %v", info.Rootless)
		t.Logf("  SocketPath: %s", info.SocketPath)
		t.Logf("  OS:         %s", info.OS)
		t.Logf("  Arch:       %s", info.Arch)
	})
}

// TestDockerClient_RealConnection tests connecting to a real Docker daemon.
// This test is skipped in short mode and requires Docker to be running.
func TestDockerClient_RealConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("explicit docker", func(t *testing.T) {
		client, err := NewClient(ctx, &Config{
			Runtime: "docker",
			Timeout: 5 * time.Second,
		})
		if err != nil {
			t.Skipf("Docker not available: %v", err)
			return
		}
		defer client.Close()

		assert.Equal(t, "docker", client.Runtime())

		info, err := client.Info(ctx)
		require.NoError(t, err)
		assert.Equal(t, "docker", info.Runtime)
		assert.NotEmpty(t, info.Version)
	})
}

// TestExplicitSocket tests connecting with an explicit socket path.
func TestExplicitSocket(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("rootless socket", func(t *testing.T) {
		socket, err := getPodmanRootlessSocket()
		require.NoError(t, err)

		client, err := NewClient(ctx, &Config{
			Runtime:    "podman",
			SocketPath: socket,
			Timeout:    5 * time.Second,
		})
		if err != nil {
			t.Skipf("Podman rootless socket not available: %v", err)
			return
		}
		defer client.Close()

		info, err := client.Info(ctx)
		require.NoError(t, err)
		assert.True(t, info.Rootless, "Should be running rootless")
		assert.Equal(t, socket, info.SocketPath)
	})
}
