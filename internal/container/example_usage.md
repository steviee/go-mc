# Container Lifecycle Operations - Usage Examples

This document provides examples of how to use the container lifecycle operations in the go-mc project.

## Table of Contents
- [Basic Setup](#basic-setup)
- [Creating a Container](#creating-a-container)
- [Starting and Stopping](#starting-and-stopping)
- [Container Management](#container-management)
- [Advanced Configuration](#advanced-configuration)
- [Error Handling](#error-handling)

## Basic Setup

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/steviee/go-mc/internal/container"
)

func main() {
    ctx := context.Background()

    // Create a new container client (auto-detects Podman/Docker)
    client, err := container.NewClient(ctx, container.DefaultConfig())
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // Test connection
    if err := client.Ping(ctx); err != nil {
        log.Fatalf("Failed to connect to container daemon: %v", err)
    }
}
```

## Creating a Container

### Minimal Configuration

```go
config := &container.ContainerConfig{
    Name:  "my-minecraft-server",
    Image: "docker.io/itzg/minecraft-server:latest",
}

containerID, err := client.CreateContainer(ctx, config)
if err != nil {
    log.Fatalf("Failed to create container: %v", err)
}
log.Printf("Created container: %s", containerID)
```

### Full Configuration

```go
config := &container.ContainerConfig{
    Name:  "my-minecraft-server",
    Image: "docker.io/itzg/minecraft-server:latest",

    // Environment variables
    Env: map[string]string{
        "EULA":           "TRUE",
        "TYPE":           "FABRIC",
        "VERSION":        "1.20.4",
        "MEMORY":         "4G",
        "MAX_PLAYERS":    "20",
    },

    // Port mappings (host:container)
    Ports: map[int]int{
        25565: 25565, // Minecraft
        25575: 25575, // RCON
    },

    // Volume mounts (host:container)
    Volumes: map[string]string{
        "/opt/minecraft/data": "/data",
    },

    // Resource limits
    Memory:   "4G",
    CPUQuota: 200000, // 2 CPUs

    // Labels for organization
    Labels: map[string]string{
        "managed-by": "go-mc",
        "server-type": "minecraft",
    },
}

containerID, err := client.CreateContainer(ctx, config)
if err != nil {
    log.Fatalf("Failed to create container: %v", err)
}
```

## Starting and Stopping

### Start Container

```go
err := client.StartContainer(ctx, "my-minecraft-server")
if err != nil {
    log.Fatalf("Failed to start container: %v", err)
}

// Wait for container to be running
err = client.WaitForContainer(ctx, "my-minecraft-server", "running")
if err != nil {
    log.Fatalf("Container failed to start: %v", err)
}
```

### Stop Container

```go
// Stop with default timeout
timeout := 30 * time.Second
err := client.StopContainer(ctx, "my-minecraft-server", &timeout)
if err != nil {
    log.Fatalf("Failed to stop container: %v", err)
}
```

### Restart Container

```go
timeout := 30 * time.Second
err := client.RestartContainer(ctx, "my-minecraft-server", &timeout)
if err != nil {
    log.Fatalf("Failed to restart container: %v", err)
}
```

## Container Management

### Inspect Container

```go
info, err := client.InspectContainer(ctx, "my-minecraft-server")
if err != nil {
    log.Fatalf("Failed to inspect container: %v", err)
}

log.Printf("Container: %s", info.Name)
log.Printf("State: %s", info.State)
log.Printf("Image: %s", info.Image)
log.Printf("Created: %s", info.Created)
log.Printf("Ports: %v", info.Ports)
```

### List Containers

```go
// List all containers (including stopped)
containers, err := client.ListContainers(ctx, &container.ListOptions{
    All: true,
})
if err != nil {
    log.Fatalf("Failed to list containers: %v", err)
}

for _, c := range containers {
    log.Printf("Container: %s (%s)", c.Name, c.State)
}
```

### List Running Containers Only

```go
containers, err := client.ListContainers(ctx, &container.ListOptions{
    All: false,
})
```

### List with Filters

```go
containers, err := client.ListContainers(ctx, &container.ListOptions{
    All: true,
    Filter: map[string]string{
        "label": "managed-by=go-mc",
    },
    Limit: 10,
})
```

### Remove Container

```go
// Remove stopped container
err := client.RemoveContainer(ctx, "my-minecraft-server", &container.RemoveOptions{
    Force:         false,
    RemoveVolumes: false,
})

// Force remove (even if running) and remove volumes
err = client.RemoveContainer(ctx, "my-minecraft-server", &container.RemoveOptions{
    Force:         true,
    RemoveVolumes: true,
})
```

## Advanced Configuration

### Container with Port Publishing

```go
config := &container.ContainerConfig{
    Name:  "nginx-server",
    Image: "docker.io/library/nginx:alpine",
    Ports: map[int]int{
        8080: 80,   // HTTP
        8443: 443,  // HTTPS
    },
}
```

### Container with Volume Mounts

```go
config := &container.ContainerConfig{
    Name:  "data-container",
    Image: "docker.io/library/alpine:latest",
    Volumes: map[string]string{
        "/host/data":   "/data",
        "/host/config": "/config:ro", // Read-only mount
    },
}
```

### Container with Resource Limits

```go
config := &container.ContainerConfig{
    Name:     "limited-container",
    Image:    "docker.io/library/alpine:latest",
    Memory:   "2G",      // 2 gigabytes RAM
    CPUQuota: 100000,    // 1 CPU (100000 microseconds per 100ms)
}
```

### Container with Custom Command

```go
config := &container.ContainerConfig{
    Name:       "custom-command",
    Image:      "docker.io/library/alpine:latest",
    Command:    []string{"sh", "-c", "while true; do echo hello; sleep 5; done"},
    WorkingDir: "/app",
}
```

## Error Handling

### Checking for Specific Errors

```go
import "errors"

err := client.StartContainer(ctx, "nonexistent")
if err != nil {
    if errors.Is(err, container.ErrContainerNotFound) {
        log.Printf("Container not found")
    } else {
        log.Printf("Other error: %v", err)
    }
}
```

### Handling Different Error Types

```go
containerID, err := client.CreateContainer(ctx, config)
if err != nil {
    switch {
    case errors.Is(err, container.ErrContainerAlreadyExists):
        log.Printf("Container already exists, using existing one")
        // Continue with existing container
    case errors.Is(err, container.ErrInvalidMemoryFormat):
        log.Printf("Invalid memory format in config")
        return
    default:
        log.Fatalf("Unexpected error: %v", err)
    }
}
```

### Context Cancellation

```go
// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Operation will be canceled if it takes longer than 30 seconds
err := client.StartContainer(ctx, "my-container")
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Printf("Operation timed out")
    } else {
        log.Printf("Error: %v", err)
    }
}
```

## Complete Example: Minecraft Server Lifecycle

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/steviee/go-mc/internal/container"
)

func main() {
    ctx := context.Background()

    // Create client
    client, err := container.NewClient(ctx, container.DefaultConfig())
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // Container configuration
    serverName := "minecraft-server-1"
    config := &container.ContainerConfig{
        Name:  serverName,
        Image: "docker.io/itzg/minecraft-server:latest",
        Env: map[string]string{
            "EULA":    "TRUE",
            "TYPE":    "FABRIC",
            "VERSION": "1.20.4",
            "MEMORY":  "4G",
        },
        Ports: map[int]int{
            25565: 25565,
        },
        Volumes: map[string]string{
            "/opt/minecraft/server1": "/data",
        },
        Memory:   "4G",
        CPUQuota: 200000,
        Labels: map[string]string{
            "managed-by": "go-mc",
        },
    }

    // Create container
    log.Printf("Creating container: %s", serverName)
    containerID, err := client.CreateContainer(ctx, config)
    if err != nil {
        log.Fatalf("Failed to create container: %v", err)
    }
    log.Printf("Created container: %s", containerID)

    // Start container
    log.Printf("Starting container...")
    err = client.StartContainer(ctx, serverName)
    if err != nil {
        log.Fatalf("Failed to start container: %v", err)
    }

    // Wait for running state
    log.Printf("Waiting for container to be running...")
    err = client.WaitForContainer(ctx, serverName, "running")
    if err != nil {
        log.Fatalf("Container failed to start: %v", err)
    }

    // Inspect container
    info, err := client.InspectContainer(ctx, serverName)
    if err != nil {
        log.Fatalf("Failed to inspect container: %v", err)
    }
    log.Printf("Container is %s", info.State)

    // Simulate server running for a while
    log.Printf("Server is running. Press Ctrl+C to stop...")
    time.Sleep(10 * time.Second)

    // Stop container gracefully
    log.Printf("Stopping container...")
    timeout := 30 * time.Second
    err = client.StopContainer(ctx, serverName, &timeout)
    if err != nil {
        log.Fatalf("Failed to stop container: %v", err)
    }

    log.Printf("Container stopped successfully")
}
```

## Best Practices

1. **Always use context**: Pass context to all operations to enable cancellation and timeouts.

2. **Handle errors properly**: Check for specific error types and handle them appropriately.

3. **Clean up resources**: Use `defer client.Close()` to ensure proper cleanup.

4. **Use labels**: Add labels to containers for easy filtering and management.

5. **Set resource limits**: Always set memory and CPU limits to prevent containers from consuming all system resources.

6. **Graceful shutdown**: Use appropriate timeouts when stopping containers to allow for graceful shutdown.

7. **Monitor container state**: Use `InspectContainer` to check container health and state.

8. **Use volume mounts**: Store persistent data in volumes to survive container restarts.
