# Container Package

The `container` package provides a unified client interface for interacting with Podman and Docker container runtimes.

## Features

- **Auto-detection**: Automatically detects and connects to available container runtimes
- **Rootless Podman support**: Prefers rootless Podman over rootful
- **Docker fallback**: Falls back to Docker if Podman is not available
- **Context support**: All operations support context for cancellation and timeouts
- **Comprehensive error handling**: Clear error messages with actionable suggestions
- **Thread-safe**: Safe for concurrent use

## Usage

### Basic Usage

```go
import (
    "context"
    "log"

    "github.com/steviee/go-mc/internal/container"
)

func main() {
    ctx := context.Background()

    // Create client with auto-detection
    client, err := container.NewClient(ctx, &container.Config{
        Runtime: "auto",
        Timeout: 30 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Test connection
    if err := client.Ping(ctx); err != nil {
        log.Fatal(err)
    }

    // Get runtime information
    info, err := client.Info(ctx)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Connected to %s %s (%s)",
        info.Runtime,
        info.Version,
        map[bool]string{true: "rootless", false: "rootful"}[info.Rootless])
}
```

### Configuration Options

```go
cfg := &container.Config{
    Runtime:    "auto",  // "auto", "podman", or "docker"
    SocketPath: "",      // Optional: explicit socket path
    Timeout:    30 * time.Second,
}
```

### Explicit Runtime Selection

```go
// Force Podman
client, err := container.NewClient(ctx, &container.Config{
    Runtime: "podman",
})

// Force Docker
client, err := container.NewClient(ctx, &container.Config{
    Runtime: "docker",
})
```

### Explicit Socket Path

```go
// Use specific Podman socket
client, err := container.NewClient(ctx, &container.Config{
    Runtime:    "podman",
    SocketPath: "/run/user/1000/podman/podman.sock",
})
```

## Socket Auto-Detection

The client automatically detects container runtime sockets in the following order:

### Podman Detection Order
1. Rootless socket: `/run/user/<UID>/podman/podman.sock` (preferred)
2. Rootful socket: `/run/podman/podman.sock` (fallback)

### Docker Detection
- Socket: `/var/run/docker.sock`

## Error Handling

The package provides detailed error messages with actionable suggestions:

```go
client, err := container.NewClient(ctx, cfg)
if err != nil {
    if errors.Is(err, container.ErrNoRuntimeAvailable) {
        // No container runtime found
        log.Printf("Install Podman: %v", err)
    } else if errors.Is(err, container.ErrDaemonNotRunning) {
        // Daemon is not running
        log.Printf("Start daemon: %v", err)
    } else if errors.Is(err, container.ErrPermissionDenied) {
        // Permission error
        log.Printf("Fix permissions: %v", err)
    }
}
```

### Sentinel Errors

- `ErrDaemonNotRunning` - Container daemon is not running
- `ErrSocketNotFound` - Container socket file does not exist
- `ErrAPIVersionMismatch` - API version not supported
- `ErrPermissionDenied` - Permission denied accessing socket
- `ErrNoRuntimeAvailable` - No container runtime available

## Testing

### Unit Tests

```bash
go test ./internal/container/
```

### Integration Tests

Integration tests require a running Podman or Docker daemon:

```bash
go test -tags=integration ./internal/container/
```

Skip in short mode:
```bash
go test -short ./internal/container/
```

### Test Coverage

```bash
go test -cover ./internal/container/
```

### Race Detection

```bash
go test -race ./internal/container/
```

## Mock Client

For testing purposes, the package provides a `MockClient`:

```go
import "github.com/steviee/go-mc/internal/container"

func TestMyFunction(t *testing.T) {
    mock := &container.MockClient{
        PingFunc: func(ctx context.Context) error {
            return nil
        },
        InfoFunc: func(ctx context.Context) (*container.RuntimeInfo, error) {
            return &container.RuntimeInfo{
                Runtime:  "podman",
                Version:  "5.0.0",
                Rootless: true,
            }, nil
        },
    }

    // Use mock in tests
    myFunction(mock)
}
```

## Architecture

### Client Interface

```go
type Client interface {
    Ping(ctx context.Context) error
    Info(ctx context.Context) (*RuntimeInfo, error)
    Close() error
    Runtime() string
}
```

### RuntimeInfo Structure

```go
type RuntimeInfo struct {
    Runtime    string // "podman" or "docker"
    Version    string // e.g., "5.0.0"
    APIVersion string // e.g., "1.41"
    Rootless   bool   // Whether running rootless
    SocketPath string // Path to Unix socket
    OS         string // Operating system
    Arch       string // Architecture
}
```

## Dependencies

- `github.com/containers/podman/v5` - Podman Go bindings
- Standard library packages for networking and context

## Thread Safety

All client operations are thread-safe and can be called concurrently from multiple goroutines.

## Context Support

All operations accept a `context.Context` parameter for:
- Cancellation
- Timeouts
- Deadline propagation

Example with timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := client.Ping(ctx)
```

## Logging

The package uses structured logging via `log/slog` with the following levels:
- `DEBUG`: Socket detection attempts
- `INFO`: Successful connections, runtime selection
- `ERROR`: Connection failures, configuration errors

## Best Practices

1. **Always defer Close()**: Ensure resources are cleaned up
2. **Use contexts**: Pass contexts for cancellation support
3. **Handle errors**: Check for sentinel errors and provide user-friendly messages
4. **Prefer auto-detection**: Let the client detect the best runtime
5. **Set reasonable timeouts**: Default 30s, adjust based on your needs

## Troubleshooting

### No runtime available

```
Error: no container runtime available
```

**Solution**: Install Podman:
```bash
sudo apt install podman
systemctl --user enable --now podman.socket
```

### Daemon not running

```
Error: container daemon is not running
```

**Solution**: Start the daemon:
```bash
systemctl --user start podman.socket
```

### Permission denied (Podman)

```
Error: permission denied accessing socket
```

**Solution**: For rootful Podman:
```bash
sudo usermod -aG podman $USER
# Log out and log back in
```

### Permission denied (Docker)

```
Error: permission denied accessing socket
```

**Solution**:
```bash
sudo usermod -aG docker $USER
# Log out and log back in
```

## License

MIT License - See LICENSE file in project root.
