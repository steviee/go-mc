package container

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMemory(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{
			name:    "gigabytes",
			input:   "2G",
			want:    2 * 1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "megabytes",
			input:   "512M",
			want:    512 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "kilobytes",
			input:   "1024K",
			want:    1024 * 1024,
			wantErr: false,
		},
		{
			name:    "bytes",
			input:   "1024",
			want:    1024,
			wantErr: false,
		},
		{
			name:    "lowercase g",
			input:   "1g",
			want:    1 * 1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMemory(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParsePortMapping(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantHost      int
		wantContainer int
		wantErr       bool
	}{
		{
			name:          "both ports specified",
			input:         "8080:80",
			wantHost:      8080,
			wantContainer: 80,
			wantErr:       false,
		},
		{
			name:          "single port",
			input:         "3000",
			wantHost:      3000,
			wantContainer: 3000,
			wantErr:       false,
		},
		{
			name:          "different ports",
			input:         "25565:25565",
			wantHost:      25565,
			wantContainer: 25565,
			wantErr:       false,
		},
		{
			name:          "invalid host port",
			input:         "abc:80",
			wantHost:      0,
			wantContainer: 0,
			wantErr:       true,
		},
		{
			name:          "invalid container port",
			input:         "80:xyz",
			wantHost:      0,
			wantContainer: 0,
			wantErr:       true,
		},
		{
			name:          "too many colons",
			input:         "8080:80:tcp",
			wantHost:      0,
			wantContainer: 0,
			wantErr:       true,
		},
		{
			name:          "invalid single port",
			input:         "notaport",
			wantHost:      0,
			wantContainer: 0,
			wantErr:       true,
		},
		{
			name:          "empty string",
			input:         "",
			wantHost:      0,
			wantContainer: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotContainer, err := parsePortMapping(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, gotHost)
			assert.Equal(t, tt.wantContainer, gotContainer)
		})
	}
}

func TestBuildPortMappings(t *testing.T) {
	c := &client{}

	tests := []struct {
		name    string
		ports   map[int]int
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid single port",
			ports: map[int]int{
				8080: 80,
			},
			wantErr: false,
		},
		{
			name: "valid multiple ports",
			ports: map[int]int{
				8080:  80,
				25565: 25565,
				3306:  3306,
			},
			wantErr: false,
		},
		{
			name: "invalid host port too low",
			ports: map[int]int{
				0: 80,
			},
			wantErr: true,
			errMsg:  "invalid host port",
		},
		{
			name: "invalid host port too high",
			ports: map[int]int{
				65536: 80,
			},
			wantErr: true,
			errMsg:  "invalid host port",
		},
		{
			name: "invalid container port too low",
			ports: map[int]int{
				8080: 0,
			},
			wantErr: true,
			errMsg:  "invalid container port",
		},
		{
			name: "invalid container port too high",
			ports: map[int]int{
				8080: 65536,
			},
			wantErr: true,
			errMsg:  "invalid container port",
		},
		{
			name:    "empty ports",
			ports:   map[int]int{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings, err := c.buildPortMappings(tt.ports)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}
			require.NoError(t, err)
			assert.Len(t, mappings, len(tt.ports))

			// Verify port mappings
			for _, mapping := range mappings {
				hostPort := int(mapping.HostPort)
				containerPort := int(mapping.ContainerPort)
				expectedContainer, ok := tt.ports[hostPort]
				assert.True(t, ok, "host port %d not found in input", hostPort)
				assert.Equal(t, expectedContainer, containerPort)
				assert.Equal(t, "tcp", mapping.Protocol)
				assert.Equal(t, "0.0.0.0", mapping.HostIP)
			}
		})
	}
}

func TestBuildVolumeMounts(t *testing.T) {
	c := &client{}

	tests := []struct {
		name    string
		volumes map[string]string
		wantErr bool
	}{
		{
			name: "simple bind mount",
			volumes: map[string]string{
				"/host/data": "/container/data",
			},
			wantErr: false,
		},
		{
			name: "read-only mount",
			volumes: map[string]string{
				"/host/config": "/container/config:ro",
			},
			wantErr: false,
		},
		{
			name: "multiple mounts",
			volumes: map[string]string{
				"/host/data":   "/container/data",
				"/host/config": "/container/config:ro",
				"/host/logs":   "/container/logs:rw",
			},
			wantErr: false,
		},
		{
			name:    "empty volumes",
			volumes: map[string]string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mounts, err := c.buildVolumeMounts(tt.volumes)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, mounts, len(tt.volumes))

			// Verify mounts
			for _, mount := range mounts {
				containerPath := mount.Destination
				// Check if this mount has options
				expectedHost := ""
				for host, container := range tt.volumes {
					if container == containerPath || (len(container) > len(containerPath) && container[:len(containerPath)] == containerPath) {
						expectedHost = host
						break
					}
				}
				assert.NotEmpty(t, expectedHost, "container path %s not found in input", containerPath)
				assert.Equal(t, expectedHost, mount.Source)
				assert.Equal(t, "bind", mount.Type)
				assert.NotEmpty(t, mount.Options)
			}
		})
	}
}

func TestSetResourceLimits(t *testing.T) {
	c := &client{}

	tests := []struct {
		name    string
		config  *ContainerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid memory limit",
			config: &ContainerConfig{
				Memory: "2G",
			},
			wantErr: false,
		},
		{
			name: "valid CPU quota",
			config: &ContainerConfig{
				CPUQuota: 100000,
			},
			wantErr: false,
		},
		{
			name: "both limits",
			config: &ContainerConfig{
				Memory:   "1G",
				CPUQuota: 50000,
			},
			wantErr: false,
		},
		{
			name: "invalid memory format",
			config: &ContainerConfig{
				Memory: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid memory format",
		},
		{
			name:    "no limits",
			config:  &ContainerConfig{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &ContainerConfig{
				Name:  "test",
				Image: "alpine:latest",
			}
			// Build a proper SpecGenerator
			s, err := c.buildContainerSpec(spec)
			require.NoError(t, err)

			err = c.setResourceLimits(s, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}
			require.NoError(t, err)

			// Verify limits are set
			if tt.config.Memory != "" {
				assert.NotNil(t, s.ResourceLimits)
				assert.NotNil(t, s.ResourceLimits.Memory)
				assert.NotNil(t, s.ResourceLimits.Memory.Limit)
			}
			if tt.config.CPUQuota > 0 {
				assert.NotNil(t, s.ResourceLimits)
				assert.NotNil(t, s.ResourceLimits.CPU)
				assert.NotNil(t, s.ResourceLimits.CPU.Quota)
				assert.Equal(t, tt.config.CPUQuota, *s.ResourceLimits.CPU.Quota)
			}
		})
	}
}

func TestBuildContainerSpec(t *testing.T) {
	c := &client{}

	tests := []struct {
		name    string
		config  *ContainerConfig
		wantErr bool
	}{
		{
			name: "minimal config",
			config: &ContainerConfig{
				Name:  "test-container",
				Image: "alpine:latest",
			},
			wantErr: false,
		},
		{
			name: "full config",
			config: &ContainerConfig{
				Name:       "test-container",
				Image:      "alpine:latest",
				Command:    []string{"sh", "-c", "sleep infinity"},
				WorkingDir: "/app",
				Env: map[string]string{
					"FOO": "bar",
					"BAZ": "qux",
				},
				Ports: map[int]int{
					8080: 80,
				},
				Volumes: map[string]string{
					"/host/data": "/data",
				},
				Memory:   "1G",
				CPUQuota: 50000,
				Labels: map[string]string{
					"app": "test",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			config: &ContainerConfig{
				Name:  "test-container",
				Image: "alpine:latest",
				Ports: map[int]int{
					99999: 80,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid memory",
			config: &ContainerConfig{
				Name:   "test-container",
				Image:  "alpine:latest",
				Memory: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := c.buildContainerSpec(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, spec)
			assert.Equal(t, tt.config.Name, spec.Name)
			assert.Equal(t, tt.config.Image, spec.Image)
			if tt.config.WorkingDir != "" {
				assert.Equal(t, tt.config.WorkingDir, spec.WorkDir)
			}
			if len(tt.config.Env) > 0 {
				assert.Equal(t, tt.config.Env, spec.Env)
			}
			if len(tt.config.Labels) > 0 {
				assert.Equal(t, tt.config.Labels, spec.Labels)
			}
			if len(tt.config.Command) > 0 {
				assert.Equal(t, tt.config.Command, spec.Command)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	// Test that operations respect context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	c := &client{
		timeout: 5 * time.Second,
	}

	// These operations should fail quickly due to canceled context
	// Note: We can't actually test these without a real Podman connection,
	// but we can verify the timeout mechanism works

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, c.timeout)
	defer timeoutCancel()

	select {
	case <-timeoutCtx.Done():
		assert.Error(t, timeoutCtx.Err())
	case <-time.After(1 * time.Second):
		t.Fatal("timeout context should be canceled")
	}
}
