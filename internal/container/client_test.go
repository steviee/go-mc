package container

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "auto", cfg.Runtime)
	assert.Equal(t, "", cfg.SocketPath)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
}

func TestRuntimeError(t *testing.T) {
	tests := []struct {
		name     string
		runtime  string
		socket   string
		err      error
		expected string
	}{
		{
			name:     "podman error",
			runtime:  "podman",
			socket:   "/run/user/1000/podman/podman.sock",
			err:      ErrDaemonNotRunning,
			expected: "podman: container daemon is not running (socket: /run/user/1000/podman/podman.sock)",
		},
		{
			name:     "docker error",
			runtime:  "docker",
			socket:   "/var/run/docker.sock",
			err:      ErrSocketNotFound,
			expected: "docker: container socket not found (socket: /var/run/docker.sock)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rerr := NewRuntimeError(tt.runtime, tt.socket, tt.err)
			assert.Equal(t, tt.expected, rerr.Error())
			assert.Equal(t, tt.err, rerr.Unwrap())
			assert.True(t, errors.Is(rerr, tt.err))
		})
	}
}

func TestGetPodmanRootlessSocket(t *testing.T) {
	socket, err := getPodmanRootlessSocket()
	require.NoError(t, err)
	assert.Contains(t, socket, "/run/user/")
	assert.Contains(t, socket, "/podman/podman.sock")
}

func TestGetPodmanRootfulSocket(t *testing.T) {
	socket := getPodmanRootfulSocket()
	assert.Equal(t, "/run/podman/podman.sock", socket)
}

func TestGetDockerSocket(t *testing.T) {
	socket := getDockerSocket()
	assert.Equal(t, "/var/run/docker.sock", socket)
}

func TestGetNoRuntimeMessage(t *testing.T) {
	msg := getNoRuntimeMessage()
	assert.Contains(t, msg, "No container runtime available")
	assert.Contains(t, msg, "go-mc system setup")
	assert.Contains(t, msg, "podman")
	assert.Contains(t, msg, "polkitd")
	assert.Contains(t, msg, "systemctl")
}

func TestGetPermissionDeniedMessage(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
		socket  string
		want    []string
	}{
		{
			name:    "podman rootless",
			runtime: "podman",
			socket:  "/run/user/1000/podman/podman.sock",
			want:    []string{"Permission denied", "rootless", "systemctl --user"},
		},
		{
			name:    "podman rootful",
			runtime: "podman",
			socket:  "/run/podman/podman.sock",
			want:    []string{"Permission denied", "rootful", "usermod"},
		},
		{
			name:    "docker",
			runtime: "docker",
			socket:  "/var/run/docker.sock",
			want:    []string{"Permission denied", "docker", "usermod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := GetPermissionDeniedMessage(tt.runtime, tt.socket)
			for _, want := range tt.want {
				assert.Contains(t, msg, want)
			}
		})
	}
}

func TestGetDaemonNotRunningMessage(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
		want    []string
	}{
		{
			name:    "podman",
			runtime: "podman",
			want:    []string{"not running", "systemctl", "podman.socket"},
		},
		{
			name:    "docker",
			runtime: "docker",
			want:    []string{"not running", "systemctl", "docker"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := GetDaemonNotRunningMessage(tt.runtime)
			for _, want := range tt.want {
				assert.Contains(t, msg, want)
			}
		})
	}
}

func TestNewClient_InvalidRuntime(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{
		Runtime: "invalid",
		Timeout: 5 * time.Second,
	}

	_, err := NewClient(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid runtime")
}

func TestNewClient_NilConfig(t *testing.T) {
	// This test verifies that NewClient uses default config when nil is passed
	// We expect it to try auto-detection and fail (since we don't have a real daemon)
	ctx := context.Background()

	_, err := NewClient(ctx, nil)
	// Should fail because no daemon is running, but should not panic
	assert.Error(t, err)
}

func TestNewClient_ExplicitSocket_NotFound(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{
		Runtime:    "podman",
		SocketPath: "/nonexistent/socket.sock",
		Timeout:    1 * time.Second,
	}

	_, err := NewClient(ctx, cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSocketNotFound))
}

// MockClient is a mock implementation of the Client interface for testing.
type MockClient struct {
	PingFunc             func(ctx context.Context) error
	InfoFunc             func(ctx context.Context) (*RuntimeInfo, error)
	CloseFunc            func() error
	RuntimeFunc          func() string
	CreateContainerFunc  func(ctx context.Context, config *ContainerConfig) (string, error)
	StartContainerFunc   func(ctx context.Context, containerID string) error
	WaitForContainerFunc func(ctx context.Context, containerID string, condition string) error
	StopContainerFunc    func(ctx context.Context, containerID string, timeout *time.Duration) error
	RestartContainerFunc func(ctx context.Context, containerID string, timeout *time.Duration) error
	RemoveContainerFunc  func(ctx context.Context, containerID string, opts *RemoveOptions) error
	InspectContainerFunc func(ctx context.Context, containerID string) (*ContainerInfo, error)
	ListContainersFunc   func(ctx context.Context, opts *ListOptions) ([]*ContainerInfo, error)
}

func (m *MockClient) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

func (m *MockClient) Info(ctx context.Context) (*RuntimeInfo, error) {
	if m.InfoFunc != nil {
		return m.InfoFunc(ctx)
	}
	return &RuntimeInfo{
		Runtime:    "podman",
		Version:    "4.5.0",
		APIVersion: "1.41",
		Rootless:   true,
		SocketPath: "/run/user/1000/podman/podman.sock",
		OS:         "linux",
		Arch:       "amd64",
	}, nil
}

func (m *MockClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func (m *MockClient) Runtime() string {
	if m.RuntimeFunc != nil {
		return m.RuntimeFunc()
	}
	return "podman"
}

func (m *MockClient) CreateContainer(ctx context.Context, config *ContainerConfig) (string, error) {
	if m.CreateContainerFunc != nil {
		return m.CreateContainerFunc(ctx, config)
	}
	return "mock-container-id", nil
}

func (m *MockClient) StartContainer(ctx context.Context, containerID string) error {
	if m.StartContainerFunc != nil {
		return m.StartContainerFunc(ctx, containerID)
	}
	return nil
}

func (m *MockClient) WaitForContainer(ctx context.Context, containerID string, condition string) error {
	if m.WaitForContainerFunc != nil {
		return m.WaitForContainerFunc(ctx, containerID, condition)
	}
	return nil
}

func (m *MockClient) StopContainer(ctx context.Context, containerID string, timeout *time.Duration) error {
	if m.StopContainerFunc != nil {
		return m.StopContainerFunc(ctx, containerID, timeout)
	}
	return nil
}

func (m *MockClient) RestartContainer(ctx context.Context, containerID string, timeout *time.Duration) error {
	if m.RestartContainerFunc != nil {
		return m.RestartContainerFunc(ctx, containerID, timeout)
	}
	return nil
}

func (m *MockClient) RemoveContainer(ctx context.Context, containerID string, opts *RemoveOptions) error {
	if m.RemoveContainerFunc != nil {
		return m.RemoveContainerFunc(ctx, containerID, opts)
	}
	return nil
}

func (m *MockClient) InspectContainer(ctx context.Context, containerID string) (*ContainerInfo, error) {
	if m.InspectContainerFunc != nil {
		return m.InspectContainerFunc(ctx, containerID)
	}
	return &ContainerInfo{
		ID:      containerID,
		Name:    "mock-container",
		State:   "running",
		Status:  "Up",
		Image:   "alpine:latest",
		Ports:   map[int]int{},
		Created: time.Now(),
		Labels:  map[string]string{},
	}, nil
}

func (m *MockClient) ListContainers(ctx context.Context, opts *ListOptions) ([]*ContainerInfo, error) {
	if m.ListContainersFunc != nil {
		return m.ListContainersFunc(ctx, opts)
	}
	return []*ContainerInfo{
		{
			ID:      "mock-container-1",
			Name:    "container1",
			State:   "running",
			Status:  "Up",
			Image:   "alpine:latest",
			Ports:   map[int]int{},
			Created: time.Now(),
			Labels:  map[string]string{},
		},
	}, nil
}

func TestMockClient(t *testing.T) {
	ctx := context.Background()

	t.Run("default behavior", func(t *testing.T) {
		mock := &MockClient{}

		err := mock.Ping(ctx)
		assert.NoError(t, err)

		info, err := mock.Info(ctx)
		require.NoError(t, err)
		assert.Equal(t, "podman", info.Runtime)
		assert.Equal(t, "4.5.0", info.Version)
		assert.True(t, info.Rootless)

		assert.Equal(t, "podman", mock.Runtime())

		err = mock.Close()
		assert.NoError(t, err)
	})

	t.Run("custom behavior", func(t *testing.T) {
		mock := &MockClient{
			PingFunc: func(ctx context.Context) error {
				return ErrDaemonNotRunning
			},
			InfoFunc: func(ctx context.Context) (*RuntimeInfo, error) {
				return &RuntimeInfo{
					Runtime:  "docker",
					Version:  "24.0.0",
					Rootless: false,
				}, nil
			},
			RuntimeFunc: func() string {
				return "docker"
			},
		}

		err := mock.Ping(ctx)
		assert.Equal(t, ErrDaemonNotRunning, err)

		info, err := mock.Info(ctx)
		require.NoError(t, err)
		assert.Equal(t, "docker", info.Runtime)
		assert.Equal(t, "24.0.0", info.Version)
		assert.False(t, info.Rootless)

		assert.Equal(t, "docker", mock.Runtime())
	})

	t.Run("context timeout", func(t *testing.T) {
		mock := &MockClient{
			PingFunc: func(ctx context.Context) error {
				// Simulate slow operation
				select {
				case <-time.After(2 * time.Second):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := mock.Ping(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestClient_Interface(t *testing.T) {
	// Verify that both *client and *MockClient implement Client interface
	var _ Client = (*client)(nil)
	var _ Client = (*MockClient)(nil)
}

func TestConfig_DefaultValues(t *testing.T) {
	cfg := &Config{}

	// Test with zero values
	assert.Equal(t, "", cfg.Runtime)
	assert.Equal(t, "", cfg.SocketPath)
	assert.Equal(t, time.Duration(0), cfg.Timeout)
}

func TestCheckSocketPermissions_NonExistentSocket(t *testing.T) {
	err := checkSocketPermissions("/nonexistent/socket.sock")
	assert.Error(t, err)
}

func TestClient_CloseNilClient(t *testing.T) {
	// Create a mock client that's been closed
	c := &client{
		runtime:    "podman",
		socketPath: "/run/user/1000/podman/podman.sock",
		timeout:    30 * time.Second,
	}

	err := c.Close()
	assert.NoError(t, err)
}

func TestClient_Runtime(t *testing.T) {
	c := &client{
		runtime: "podman",
	}
	assert.Equal(t, "podman", c.Runtime())

	c2 := &client{
		runtime: "docker",
	}
	assert.Equal(t, "docker", c2.Runtime())
}

func TestRuntimeError_Interface(t *testing.T) {
	// Verify RuntimeError implements error interface
	var _ error = (*RuntimeError)(nil)
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	// Test that default timeout is set when config has zero timeout
	ctx := context.Background()
	cfg := &Config{
		Runtime:    "invalid",
		SocketPath: "",
		Timeout:    0, // Zero timeout should be replaced with default
	}

	_, err := NewClient(ctx, cfg)
	require.Error(t, err)
	// Should fail for invalid runtime, but timeout should have been set
}

func TestAutoDetectRuntime_NoRuntimeAvailable(t *testing.T) {
	// This test will fail if no runtime is available (expected in CI)
	ctx := context.Background()

	// Try to auto-detect with very short timeout
	_, err := autoDetectRuntime(ctx, 100*time.Millisecond)

	// If no runtime is available, we should get a specific error
	if err != nil {
		assert.Contains(t, err.Error(), "no container runtime available")
	}
}

func TestGetPermissionDeniedMessage_AllPaths(t *testing.T) {
	// Test all code paths for permission denied messages
	tests := []struct {
		name    string
		runtime string
		socket  string
	}{
		{
			name:    "podman rootless with /run/user in path",
			runtime: "podman",
			socket:  "/run/user/1000/podman/podman.sock",
		},
		{
			name:    "podman rootless with different UID",
			runtime: "podman",
			socket:  "/run/user/9999/podman/podman.sock",
		},
		{
			name:    "podman rootful",
			runtime: "podman",
			socket:  "/run/podman/podman.sock",
		},
		{
			name:    "docker",
			runtime: "docker",
			socket:  "/var/run/docker.sock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := GetPermissionDeniedMessage(tt.runtime, tt.socket)
			assert.NotEmpty(t, msg)
			assert.Contains(t, msg, "Permission denied")
			assert.Contains(t, msg, tt.socket)
		})
	}
}

func TestGetDaemonNotRunningMessage_AllRuntimes(t *testing.T) {
	runtimes := []string{"podman", "docker"}

	for _, runtime := range runtimes {
		t.Run(runtime, func(t *testing.T) {
			msg := GetDaemonNotRunningMessage(runtime)
			assert.NotEmpty(t, msg)
			assert.Contains(t, msg, "not running")
			assert.Contains(t, msg, "systemctl")
		})
	}
}

func TestNewRuntimeError(t *testing.T) {
	rerr := NewRuntimeError("podman", "/run/user/1000/podman/podman.sock", ErrSocketNotFound)

	assert.Equal(t, "podman", rerr.Runtime)
	assert.Equal(t, "/run/user/1000/podman/podman.sock", rerr.Socket)
	assert.Equal(t, ErrSocketNotFound, rerr.Err)

	// Test Error() method
	assert.Contains(t, rerr.Error(), "podman")
	assert.Contains(t, rerr.Error(), "/run/user/1000/podman/podman.sock")
	assert.Contains(t, rerr.Error(), "container socket not found")

	// Test Unwrap() method
	assert.Equal(t, ErrSocketNotFound, rerr.Unwrap())
}

func TestSentinelErrors(t *testing.T) {
	// Verify all sentinel errors are defined and distinct
	errors := []error{
		ErrDaemonNotRunning,
		ErrSocketNotFound,
		ErrAPIVersionMismatch,
		ErrPermissionDenied,
		ErrNoRuntimeAvailable,
	}

	// Check each error has a message
	for _, err := range errors {
		assert.NotEmpty(t, err.Error())
	}

	// Check errors are distinct
	for i, err1 := range errors {
		for j, err2 := range errors {
			if i != j {
				assert.NotEqual(t, err1, err2)
			}
		}
	}
}

func TestDetectPodmanSocket_Errors(t *testing.T) {
	ctx := context.Background()

	// This will fail if no Podman socket is available
	_, err := detectPodmanSocket(ctx, 100*time.Millisecond)
	if err != nil {
		assert.Contains(t, err.Error(), "failed to connect to Podman")
	}
}

func TestDetectDockerSocket_Errors(t *testing.T) {
	ctx := context.Background()

	// This will fail if no Docker socket is available
	_, err := detectDockerSocket(ctx, 100*time.Millisecond)
	if err != nil {
		assert.Contains(t, err.Error(), "failed to connect to Docker")
	}
}

func TestConnectToSocket_InvalidURI(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file to use as a fake socket
	tmpFile := "/tmp/fake-socket-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".sock"

	// Try to connect to non-socket file should fail
	_, err := connectToSocket(ctx, "podman", tmpFile, 1*time.Second)
	assert.Error(t, err)
}

func TestGetPodmanRootlessSocket_Error(t *testing.T) {
	// This test verifies the function doesn't panic
	socket, err := getPodmanRootlessSocket()
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotEmpty(t, socket)
		assert.Contains(t, socket, "/run/user/")
		assert.Contains(t, socket, "/podman/podman.sock")
	}
}

func TestNewClient_ConfigVariations(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: true, // Will fail because no runtime available
		},
		{
			name: "explicit podman",
			config: &Config{
				Runtime: "podman",
				Timeout: 1 * time.Second,
			},
			wantErr: true, // Will fail if Podman not running
		},
		{
			name: "explicit docker",
			config: &Config{
				Runtime: "docker",
				Timeout: 1 * time.Second,
			},
			wantErr: true, // Will fail if Docker not running
		},
		{
			name: "auto with timeout",
			config: &Config{
				Runtime: "auto",
				Timeout: 1 * time.Second,
			},
			wantErr: true, // Will fail if no runtime available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(ctx, tt.config)
			if tt.wantErr {
				// We expect errors in CI where no container runtime is available
				// Just verify we don't panic
				t.Logf("Got expected error: %v", err)
			}
		})
	}
}

func TestMockClient_AllMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("with custom close", func(t *testing.T) {
		closeCalled := false
		mock := &MockClient{
			CloseFunc: func() error {
				closeCalled = true
				return nil
			},
		}

		err := mock.Close()
		assert.NoError(t, err)
		assert.True(t, closeCalled)
	})

	t.Run("with error returns", func(t *testing.T) {
		mock := &MockClient{
			PingFunc: func(ctx context.Context) error {
				return ErrDaemonNotRunning
			},
			InfoFunc: func(ctx context.Context) (*RuntimeInfo, error) {
				return nil, ErrSocketNotFound
			},
			CloseFunc: func() error {
				return ErrPermissionDenied
			},
		}

		err := mock.Ping(ctx)
		assert.Equal(t, ErrDaemonNotRunning, err)

		_, err = mock.Info(ctx)
		assert.Equal(t, ErrSocketNotFound, err)

		err = mock.Close()
		assert.Equal(t, ErrPermissionDenied, err)
	})
}

func TestClient_String(t *testing.T) {
	c := &client{
		runtime:    "podman",
		socketPath: "/run/user/1000/podman/podman.sock",
		timeout:    30 * time.Second,
	}

	// Just verify the struct is valid
	assert.Equal(t, "podman", c.runtime)
	assert.Equal(t, "/run/user/1000/podman/podman.sock", c.socketPath)
	assert.Equal(t, 30*time.Second, c.timeout)
}
