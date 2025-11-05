//go:build integration

package container

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContainerLifecycle tests the full container lifecycle with a real Podman instance.
func TestContainerLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create client
	client, err := NewClient(ctx, DefaultConfig())
	require.NoError(t, err, "failed to create client")
	defer client.Close()

	// Test connection
	err = client.Ping(ctx)
	require.NoError(t, err, "failed to ping daemon")

	// Container name for this test
	containerName := "go-mc-test-lifecycle"

	// Cleanup any existing container
	_ = client.RemoveContainer(ctx, containerName, &RemoveOptions{Force: true})

	t.Run("create container", func(t *testing.T) {
		config := &ContainerConfig{
			Name:    containerName,
			Image:   "docker.io/library/alpine:latest",
			Command: []string{"sleep", "300"},
			Env: map[string]string{
				"TEST_VAR": "test_value",
			},
			Labels: map[string]string{
				"test": "go-mc-integration",
			},
		}

		containerID, err := client.CreateContainer(ctx, config)
		require.NoError(t, err)
		assert.NotEmpty(t, containerID)
	})

	t.Run("inspect created container", func(t *testing.T) {
		info, err := client.InspectContainer(ctx, containerName)
		require.NoError(t, err)
		assert.Equal(t, containerName, info.Name)
		assert.Contains(t, info.Image, "alpine")
		assert.Equal(t, "test_value", info.Labels["TEST_VAR"])
	})

	t.Run("start container", func(t *testing.T) {
		err := client.StartContainer(ctx, containerName)
		require.NoError(t, err)
	})

	t.Run("wait for running state", func(t *testing.T) {
		err := client.WaitForContainer(ctx, containerName, "running")
		require.NoError(t, err)

		// Verify it's actually running
		info, err := client.InspectContainer(ctx, containerName)
		require.NoError(t, err)
		assert.Equal(t, "running", info.State)
	})

	t.Run("list containers", func(t *testing.T) {
		containers, err := client.ListContainers(ctx, &ListOptions{All: true})
		require.NoError(t, err)
		assert.NotEmpty(t, containers)

		// Find our container
		found := false
		for _, c := range containers {
			if c.Name == containerName {
				found = true
				assert.Equal(t, "running", c.State)
				break
			}
		}
		assert.True(t, found, "container not found in list")
	})

	t.Run("restart container", func(t *testing.T) {
		timeout := 10 * time.Second
		err := client.RestartContainer(ctx, containerName, &timeout)
		require.NoError(t, err)

		// Wait a bit for restart to complete
		time.Sleep(1 * time.Second)

		// Verify it's running again
		info, err := client.InspectContainer(ctx, containerName)
		require.NoError(t, err)
		assert.Equal(t, "running", info.State)
	})

	t.Run("stop container", func(t *testing.T) {
		timeout := 10 * time.Second
		err := client.StopContainer(ctx, containerName, &timeout)
		require.NoError(t, err)

		// Verify it's stopped
		info, err := client.InspectContainer(ctx, containerName)
		require.NoError(t, err)
		assert.NotEqual(t, "running", info.State)
	})

	t.Run("remove container", func(t *testing.T) {
		err := client.RemoveContainer(ctx, containerName, &RemoveOptions{Force: true})
		require.NoError(t, err)

		// Verify it's gone
		_, err = client.InspectContainer(ctx, containerName)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})
}

// TestContainerWithPorts tests container creation with port mappings.
func TestContainerWithPorts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := NewClient(ctx, DefaultConfig())
	require.NoError(t, err)
	defer client.Close()

	containerName := "go-mc-test-ports"
	defer client.RemoveContainer(ctx, containerName, &RemoveOptions{Force: true})

	config := &ContainerConfig{
		Name:  containerName,
		Image: "docker.io/library/nginx:alpine",
		Ports: map[int]int{
			18080: 80,
		},
	}

	containerID, err := client.CreateContainer(ctx, config)
	require.NoError(t, err)
	assert.NotEmpty(t, containerID)

	// Start and verify ports
	err = client.StartContainer(ctx, containerName)
	require.NoError(t, err)

	info, err := client.InspectContainer(ctx, containerName)
	require.NoError(t, err)
	assert.Contains(t, info.Ports, 18080)
}

// TestContainerWithVolumes tests container creation with volume mounts.
func TestContainerWithVolumes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := NewClient(ctx, DefaultConfig())
	require.NoError(t, err)
	defer client.Close()

	containerName := "go-mc-test-volumes"
	defer client.RemoveContainer(ctx, containerName, &RemoveOptions{Force: true})

	// Create a temp directory for testing
	tmpDir := t.TempDir()

	config := &ContainerConfig{
		Name:    containerName,
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"sleep", "60"},
		Volumes: map[string]string{
			tmpDir: "/data",
		},
	}

	containerID, err := client.CreateContainer(ctx, config)
	require.NoError(t, err)
	assert.NotEmpty(t, containerID)
}

// TestContainerWithResourceLimits tests container creation with CPU and memory limits.
func TestContainerWithResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := NewClient(ctx, DefaultConfig())
	require.NoError(t, err)
	defer client.Close()

	containerName := "go-mc-test-limits"
	defer client.RemoveContainer(ctx, containerName, &RemoveOptions{Force: true})

	config := &ContainerConfig{
		Name:     containerName,
		Image:    "docker.io/library/alpine:latest",
		Command:  []string{"sleep", "60"},
		Memory:   "512M",
		CPUQuota: 50000,
	}

	containerID, err := client.CreateContainer(ctx, config)
	require.NoError(t, err)
	assert.NotEmpty(t, containerID)

	// Start container to verify limits work
	err = client.StartContainer(ctx, containerName)
	require.NoError(t, err)
}

// TestContainerErrors tests error conditions.
func TestContainerErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := NewClient(ctx, DefaultConfig())
	require.NoError(t, err)
	defer client.Close()

	t.Run("create duplicate container", func(t *testing.T) {
		containerName := "go-mc-test-duplicate"
		defer client.RemoveContainer(ctx, containerName, &RemoveOptions{Force: true})

		config := &ContainerConfig{
			Name:    containerName,
			Image:   "docker.io/library/alpine:latest",
			Command: []string{"sleep", "60"},
		}

		// Create first time
		_, err := client.CreateContainer(ctx, config)
		require.NoError(t, err)

		// Try to create again
		_, err = client.CreateContainer(ctx, config)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContainerAlreadyExists)
	})

	t.Run("start nonexistent container", func(t *testing.T) {
		err := client.StartContainer(ctx, "nonexistent-container")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})

	t.Run("stop nonexistent container", func(t *testing.T) {
		timeout := 5 * time.Second
		err := client.StopContainer(ctx, "nonexistent-container", &timeout)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})

	t.Run("remove nonexistent container", func(t *testing.T) {
		err := client.RemoveContainer(ctx, "nonexistent-container", &RemoveOptions{})
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})

	t.Run("inspect nonexistent container", func(t *testing.T) {
		_, err := client.InspectContainer(ctx, "nonexistent-container")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})

	t.Run("invalid wait condition", func(t *testing.T) {
		containerName := "go-mc-test-wait"
		defer client.RemoveContainer(ctx, containerName, &RemoveOptions{Force: true})

		config := &ContainerConfig{
			Name:    containerName,
			Image:   "docker.io/library/alpine:latest",
			Command: []string{"sleep", "60"},
		}

		_, err := client.CreateContainer(ctx, config)
		require.NoError(t, err)

		err = client.WaitForContainer(ctx, containerName, "invalid-condition")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCondition)
	})
}

// TestListContainersFiltering tests container listing with filters.
func TestListContainersFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := NewClient(ctx, DefaultConfig())
	require.NoError(t, err)
	defer client.Close()

	// Create test containers with labels
	containerName1 := "go-mc-test-list-1"
	containerName2 := "go-mc-test-list-2"
	defer client.RemoveContainer(ctx, containerName1, &RemoveOptions{Force: true})
	defer client.RemoveContainer(ctx, containerName2, &RemoveOptions{Force: true})

	config1 := &ContainerConfig{
		Name:    containerName1,
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"sleep", "60"},
		Labels: map[string]string{
			"test": "go-mc-integration",
		},
	}

	config2 := &ContainerConfig{
		Name:    containerName2,
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"sleep", "60"},
		Labels: map[string]string{
			"test": "go-mc-integration",
		},
	}

	_, err = client.CreateContainer(ctx, config1)
	require.NoError(t, err)

	_, err = client.CreateContainer(ctx, config2)
	require.NoError(t, err)

	// Start both containers
	err = client.StartContainer(ctx, containerName1)
	require.NoError(t, err)

	err = client.StartContainer(ctx, containerName2)
	require.NoError(t, err)

	t.Run("list all containers", func(t *testing.T) {
		containers, err := client.ListContainers(ctx, &ListOptions{All: true})
		require.NoError(t, err)
		assert.NotEmpty(t, containers)
	})

	t.Run("list only running containers", func(t *testing.T) {
		containers, err := client.ListContainers(ctx, &ListOptions{All: false})
		require.NoError(t, err)
		assert.NotEmpty(t, containers)

		// All listed containers should be running
		for _, c := range containers {
			assert.Equal(t, "running", c.State)
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		containers, err := client.ListContainers(ctx, &ListOptions{
			All:   true,
			Limit: 1,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(containers), 1)
	})
}
