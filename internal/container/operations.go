package container

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	nettypes "github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/docker/go-units"
	spec "github.com/opencontainers/runtime-spec/specs-go"
)

// CreateContainer creates a new container with the given configuration.
//
// It validates the configuration, converts it to a Podman spec, and creates
// the container. The container is created in a stopped state.
//
// Returns the container ID on success, or an error if:
//   - A container with the same name already exists
//   - The image cannot be found
//   - Resource limits are invalid
//   - Container creation fails
func (c *client) CreateContainer(ctx context.Context, config *ContainerConfig) (string, error) {
	slog.Debug("creating container",
		"name", config.Name,
		"image", config.Image)

	// Check if container already exists
	exists, err := containers.Exists(c.conn, config.Name, nil)
	if err != nil {
		return "", fmt.Errorf("failed to check if container exists: %w", err)
	}
	if exists {
		return "", fmt.Errorf("%w: %s", ErrContainerAlreadyExists, config.Name)
	}

	// Convert config to Podman spec
	spec, err := c.buildContainerSpec(config)
	if err != nil {
		return "", fmt.Errorf("failed to build container spec: %w", err)
	}

	// Create container
	// Use c.conn as base context (contains Podman client from bindings.NewConnection)
	timeoutCtx, cancel := context.WithTimeout(c.conn, c.timeout)
	defer cancel()

	response, err := containers.CreateWithSpec(timeoutCtx, spec, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	slog.Info("container created",
		"id", response.ID,
		"name", config.Name)

	return response.ID, nil
}

// buildContainerSpec converts ContainerConfig to Podman SpecGenerator.
func (c *client) buildContainerSpec(config *ContainerConfig) (*specgen.SpecGenerator, error) {
	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name:    config.Name,
			Command: config.Command,
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image: config.Image,
		},
	}

	// Set working directory
	if config.WorkingDir != "" {
		spec.WorkDir = config.WorkingDir
	}

	// Set environment variables
	if len(config.Env) > 0 {
		env := make(map[string]string)
		for k, v := range config.Env {
			env[k] = v
		}
		spec.Env = env
	}

	// Set labels
	if len(config.Labels) > 0 {
		labels := make(map[string]string)
		for k, v := range config.Labels {
			labels[k] = v
		}
		spec.Labels = labels
	}

	// Set port mappings
	if len(config.Ports) > 0 {
		portMappings, err := c.buildPortMappings(config.Ports)
		if err != nil {
			return nil, fmt.Errorf("failed to build port mappings: %w", err)
		}
		spec.PortMappings = portMappings
	}

	// Set volume mounts
	if len(config.Volumes) > 0 {
		mounts, err := c.buildVolumeMounts(config.Volumes)
		if err != nil {
			return nil, fmt.Errorf("failed to build volume mounts: %w", err)
		}
		spec.Mounts = mounts
	}

	// Set resource limits
	if err := c.setResourceLimits(spec, config); err != nil {
		return nil, fmt.Errorf("failed to set resource limits: %w", err)
	}

	return spec, nil
}

// buildPortMappings converts port map to Podman port mappings.
func (c *client) buildPortMappings(ports map[int]int) ([]nettypes.PortMapping, error) {
	mappings := make([]nettypes.PortMapping, 0, len(ports))

	for hostPort, containerPort := range ports {
		if hostPort < 1 || hostPort > 65535 {
			return nil, fmt.Errorf("invalid host port: %d", hostPort)
		}
		if containerPort < 1 || containerPort > 65535 {
			return nil, fmt.Errorf("invalid container port: %d", containerPort)
		}

		mappings = append(mappings, nettypes.PortMapping{
			HostPort:      uint16(hostPort),
			ContainerPort: uint16(containerPort),
			Protocol:      "tcp",
			HostIP:        "0.0.0.0",
		})
	}

	return mappings, nil
}

// buildVolumeMounts converts volume map to Podman mounts.
func (c *client) buildVolumeMounts(volumes map[string]string) ([]spec.Mount, error) {
	mounts := make([]spec.Mount, 0, len(volumes))

	for hostPath, containerPath := range volumes {
		// Parse options if present (e.g., "/host/path:/container/path:ro")
		opts := []string{"rw"}
		if strings.Contains(containerPath, ":") {
			parts := strings.Split(containerPath, ":")
			if len(parts) == 2 {
				containerPath = parts[0]
				opts = []string{parts[1]}
			}
		}

		mounts = append(mounts, spec.Mount{
			Source:      hostPath,
			Destination: containerPath,
			Type:        "bind",
			Options:     opts,
		})
	}

	return mounts, nil
}

// setResourceLimits sets CPU and memory limits on the spec.
func (c *client) setResourceLimits(s *specgen.SpecGenerator, config *ContainerConfig) error {
	// Set memory limit
	if config.Memory != "" {
		memBytes, err := parseMemory(config.Memory)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidMemoryFormat, config.Memory)
		}
		if s.ResourceLimits == nil {
			s.ResourceLimits = &spec.LinuxResources{}
		}
		if s.ResourceLimits.Memory == nil {
			s.ResourceLimits.Memory = &spec.LinuxMemory{}
		}
		s.ResourceLimits.Memory.Limit = &memBytes
	}

	// Set CPU quota
	if config.CPUQuota > 0 {
		if s.ResourceLimits == nil {
			s.ResourceLimits = &spec.LinuxResources{}
		}
		if s.ResourceLimits.CPU == nil {
			s.ResourceLimits.CPU = &spec.LinuxCPU{}
		}
		s.ResourceLimits.CPU.Quota = &config.CPUQuota
	}

	return nil
}

// parseMemory parses memory string (e.g., "2G", "512M") to bytes.
func parseMemory(mem string) (int64, error) {
	// Use docker/go-units for parsing
	bytes, err := units.RAMInBytes(mem)
	if err != nil {
		return 0, err
	}
	return bytes, nil
}

// StartContainer starts a stopped container.
//
// Returns an error if:
//   - Container does not exist
//   - Container is already running
//   - Start operation fails
func (c *client) StartContainer(ctx context.Context, containerID string) error {
	slog.Debug("starting container", "id", containerID)

	// Use c.conn as base context (contains Podman client from bindings.NewConnection)
	timeoutCtx, cancel := context.WithTimeout(c.conn, c.timeout)
	defer cancel()

	if err := containers.Start(timeoutCtx, containerID, nil); err != nil {
		if strings.Contains(err.Error(), "no such container") {
			return fmt.Errorf("%w: %s", ErrContainerNotFound, containerID)
		}
		return fmt.Errorf("failed to start container: %w", err)
	}

	slog.Info("container started", "id", containerID)
	return nil
}

// WaitForContainer waits for a container to reach the specified condition.
//
// Supported conditions:
//   - "running": Wait until container is running
//   - "not-running": Wait until container stops
//   - "removed": Wait until container is removed
//
// Returns an error if:
//   - Container does not exist
//   - Invalid condition specified
//   - Context is canceled
func (c *client) WaitForContainer(ctx context.Context, containerID string, condition string) error {
	slog.Debug("waiting for container condition",
		"id", containerID,
		"condition", condition)

	// Validate condition
	validConditions := []string{"running", "not-running", "removed"}
	valid := false
	for _, c := range validConditions {
		if condition == c {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("%w: %s (must be one of: %s)",
			ErrInvalidCondition, condition, strings.Join(validConditions, ", "))
	}

	// For "running" condition, poll container state
	if condition == "running" {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				info, err := c.InspectContainer(ctx, containerID)
				if err != nil {
					return fmt.Errorf("failed to inspect container: %w", err)
				}
				if info.State == "running" {
					return nil
				}
			}
		}
	}

	// For other conditions, use Podman's Wait API
	// Use c.conn as base context (contains Podman client from bindings.NewConnection)
	timeoutCtx, cancel := context.WithTimeout(c.conn, c.timeout)
	defer cancel()

	opts := new(containers.WaitOptions)
	opts.Conditions = []string{condition}

	_, err := containers.Wait(timeoutCtx, containerID, opts)
	if err != nil {
		if strings.Contains(err.Error(), "no such container") {
			return fmt.Errorf("%w: %s", ErrContainerNotFound, containerID)
		}
		return fmt.Errorf("failed to wait for container: %w", err)
	}

	return nil
}

// StopContainer stops a running container with optional timeout.
//
// The container is sent a SIGTERM signal and given the timeout duration
// to gracefully shut down. If the timeout expires, SIGKILL is sent.
//
// If timeout is nil, the default timeout (10 seconds) is used.
//
// Returns an error if:
//   - Container does not exist
//   - Stop operation fails
func (c *client) StopContainer(ctx context.Context, containerID string, timeout *time.Duration) error {
	slog.Debug("stopping container",
		"id", containerID,
		"timeout", timeout)

	// Calculate API timeout: container shutdown timeout + overhead buffer
	// This ensures the HTTP request doesn't timeout before the container finishes stopping
	apiTimeout := c.timeout
	if timeout != nil {
		// API timeout = container shutdown timeout + 15s buffer for API overhead
		apiTimeout = *timeout + 15*time.Second
		slog.Debug("adjusted API timeout for container shutdown",
			"container_timeout", *timeout,
			"api_timeout", apiTimeout)
	}

	// Use c.conn as base context (contains Podman client from bindings.NewConnection)
	timeoutCtx, cancel := context.WithTimeout(c.conn, apiTimeout)
	defer cancel()

	opts := new(containers.StopOptions)
	if timeout != nil {
		seconds := uint(*timeout / time.Second)
		opts = opts.WithTimeout(seconds)
	}

	if err := containers.Stop(timeoutCtx, containerID, opts); err != nil {
		if strings.Contains(err.Error(), "no such container") {
			return fmt.Errorf("%w: %s", ErrContainerNotFound, containerID)
		}
		return fmt.Errorf("failed to stop container: %w", err)
	}

	slog.Info("container stopped", "id", containerID)
	return nil
}

// RestartContainer restarts a container with optional timeout.
//
// This is equivalent to stopping and then starting the container.
// If timeout is nil, the default timeout (10 seconds) is used.
//
// Returns an error if:
//   - Container does not exist
//   - Restart operation fails
func (c *client) RestartContainer(ctx context.Context, containerID string, timeout *time.Duration) error {
	slog.Debug("restarting container",
		"id", containerID,
		"timeout", timeout)

	// Calculate API timeout: container shutdown timeout + overhead buffer
	// Restart includes stop + start, so we need extra time
	apiTimeout := c.timeout
	if timeout != nil {
		// API timeout = container shutdown timeout + 20s buffer (stop + start overhead)
		apiTimeout = *timeout + 20*time.Second
		slog.Debug("adjusted API timeout for container restart",
			"container_timeout", *timeout,
			"api_timeout", apiTimeout)
	}

	// Use c.conn as base context (contains Podman client from bindings.NewConnection)
	timeoutCtx, cancel := context.WithTimeout(c.conn, apiTimeout)
	defer cancel()

	opts := new(containers.RestartOptions)
	if timeout != nil {
		seconds := int(*timeout / time.Second)
		opts = opts.WithTimeout(seconds)
	}

	if err := containers.Restart(timeoutCtx, containerID, opts); err != nil {
		if strings.Contains(err.Error(), "no such container") {
			return fmt.Errorf("%w: %s", ErrContainerNotFound, containerID)
		}
		return fmt.Errorf("failed to restart container: %w", err)
	}

	slog.Info("container restarted", "id", containerID)
	return nil
}

// RemoveContainer removes a container.
//
// Options:
//   - Force: Remove the container even if it's running
//   - RemoveVolumes: Remove associated volumes
//
// Returns an error if:
//   - Container does not exist
//   - Container is running and Force is false
//   - Remove operation fails
func (c *client) RemoveContainer(ctx context.Context, containerID string, opts *RemoveOptions) error {
	slog.Debug("removing container",
		"id", containerID,
		"force", opts != nil && opts.Force,
		"volumes", opts != nil && opts.RemoveVolumes)

	// Use c.conn as base context (contains Podman client from bindings.NewConnection)
	timeoutCtx, cancel := context.WithTimeout(c.conn, c.timeout)
	defer cancel()

	removeOpts := new(containers.RemoveOptions)
	if opts != nil {
		removeOpts.Force = &opts.Force
		removeOpts.Volumes = &opts.RemoveVolumes
	}

	_, err := containers.Remove(timeoutCtx, containerID, removeOpts)
	if err != nil {
		if strings.Contains(err.Error(), "no such container") {
			return fmt.Errorf("%w: %s", ErrContainerNotFound, containerID)
		}
		return fmt.Errorf("failed to remove container: %w", err)
	}

	slog.Info("container removed", "id", containerID)
	return nil
}

// InspectContainer returns detailed information about a container.
//
// Returns an error if:
//   - Container does not exist
//   - Inspect operation fails
func (c *client) InspectContainer(ctx context.Context, containerID string) (*ContainerInfo, error) {
	slog.Debug("inspecting container", "id", containerID)

	// Use c.conn as base context (contains Podman client from bindings.NewConnection)
	timeoutCtx, cancel := context.WithTimeout(c.conn, c.timeout)
	defer cancel()

	data, err := containers.Inspect(timeoutCtx, containerID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "no such container") {
			return nil, fmt.Errorf("%w: %s", ErrContainerNotFound, containerID)
		}
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return c.convertInspectData(data), nil
}

// convertInspectData converts Podman inspect data to ContainerInfo.
func (c *client) convertInspectData(data *define.InspectContainerData) *ContainerInfo {
	info := &ContainerInfo{
		ID:      data.ID,
		Name:    data.Name,
		State:   data.State.Status,
		Status:  data.State.Status,
		Image:   data.ImageName,
		Labels:  data.Config.Labels,
		Ports:   make(map[int]int),
		Created: data.Created,
	}

	// Convert port mappings from NetworkSettings
	if data.NetworkSettings != nil && data.NetworkSettings.Ports != nil {
		for portProto, bindings := range data.NetworkSettings.Ports {
			if len(bindings) > 0 && bindings[0].HostPort != "" {
				// Extract port number from "port/protocol" format
				portStr := strings.Split(string(portProto), "/")[0]
				containerPort, err := strconv.Atoi(portStr)
				if err != nil {
					continue
				}

				hostPort, err := strconv.Atoi(bindings[0].HostPort)
				if err != nil {
					continue
				}

				info.Ports[hostPort] = containerPort
			}
		}
	}

	return info
}

// ListContainers lists containers based on the provided options.
//
// Options:
//   - All: Include stopped containers (default: only running)
//   - Limit: Maximum number of containers to return (0 = no limit)
//   - Filter: Filter containers by labels, names, etc.
//
// Returns an empty slice if no containers match the criteria.
func (c *client) ListContainers(ctx context.Context, opts *ListOptions) ([]*ContainerInfo, error) {
	slog.Debug("listing containers",
		"all", opts != nil && opts.All,
		"limit", opts)

	// Use c.conn as base context (contains Podman client from bindings.NewConnection)
	timeoutCtx, cancel := context.WithTimeout(c.conn, c.timeout)
	defer cancel()

	listOpts := new(containers.ListOptions)
	if opts != nil {
		listOpts.All = &opts.All
		if opts.Limit > 0 {
			limit := opts.Limit
			listOpts.Last = &limit
		}
		if len(opts.Filter) > 0 {
			filters := make(map[string][]string)
			for k, v := range opts.Filter {
				filters[k] = []string{v}
			}
			listOpts.Filters = filters
		}
	}

	containers, err := containers.List(timeoutCtx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]*ContainerInfo, 0, len(containers))
	for _, container := range containers {
		info := &ContainerInfo{
			ID:      container.ID,
			State:   container.State,
			Status:  container.Status,
			Image:   container.Image,
			Labels:  container.Labels,
			Created: container.Created,
			Ports:   make(map[int]int),
		}

		// Extract name (remove leading slash if present)
		if len(container.Names) > 0 {
			info.Name = strings.TrimPrefix(container.Names[0], "/")
		}

		// Convert port mappings
		for _, port := range container.Ports {
			if port.HostPort > 0 && port.ContainerPort > 0 {
				info.Ports[int(port.HostPort)] = int(port.ContainerPort)
			}
		}

		result = append(result, info)
	}

	slog.Debug("listed containers", "count", len(result))
	return result, nil
}

// parsePortMapping parses a port mapping string like "8080:80" or "80".
func parsePortMapping(mapping string) (hostPort, containerPort int, err error) {
	parts := strings.Split(mapping, ":")
	switch len(parts) {
	case 1:
		// Only container port specified
		port, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid port: %s", parts[0])
		}
		return port, port, nil
	case 2:
		// Host and container ports specified
		host, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid host port: %s", parts[0])
		}
		container, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid container port: %s", parts[1])
		}
		return host, container, nil
	default:
		return 0, 0, fmt.Errorf("invalid port mapping format: %s", mapping)
	}
}
