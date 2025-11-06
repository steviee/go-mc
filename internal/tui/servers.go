package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/state"
)

// loadServers loads server information from state and container runtime
func loadServers(ctx context.Context, client container.Client) ([]ServerInfo, error) {
	// Get list of registered servers from global state
	serverNames, err := state.ListServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	if len(serverNames) == 0 {
		return []ServerInfo{}, nil
	}

	// Collect server information
	servers := make([]ServerInfo, 0, len(serverNames))
	for _, name := range serverNames {
		server, err := collectServerInfo(ctx, name, client)
		if err != nil {
			// Log warning but continue with other servers
			slog.Warn("failed to collect server info", "server", name, "error", err)
			// Add server with error status
			servers = append(servers, ServerInfo{
				Name:    name,
				Status:  "error",
				Version: "unknown",
				Port:    0,
				Uptime:  "-",
			})
			continue
		}

		servers = append(servers, server)
	}

	return servers, nil
}

// collectServerInfo collects information about a single server
func collectServerInfo(ctx context.Context, name string, client container.Client) (ServerInfo, error) {
	server := ServerInfo{
		Name:   name,
		Status: "unknown",
	}

	// Load server state
	serverState, err := state.LoadServerState(ctx, name)
	if err != nil {
		return server, fmt.Errorf("failed to load server state: %w", err)
	}

	// Set basic info from state
	server.Version = serverState.Minecraft.Version
	server.Port = serverState.Minecraft.GamePort
	server.MemoryTotal = serverState.Minecraft.Memory

	// If no container ID, server was never started
	if serverState.ContainerID == "" {
		server.Status = "created"
		return server, nil
	}

	// Inspect container to get current status
	containerInfo, err := client.InspectContainer(ctx, serverState.ContainerID)
	if err != nil {
		// Container no longer exists
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no such container") {
			server.Status = "missing"
			return server, nil
		}
		return server, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Map container state to our status
	server.Status = normalizeContainerState(containerInfo.State)

	// Calculate uptime for running containers
	if server.Status == "running" && !serverState.LastStarted.IsZero() {
		server.StartedAt = serverState.LastStarted
		server.Uptime = formatUptime(serverState.LastStarted)
		// Note: Memory usage stats are not available through current container client
		// We show total allocation instead
		server.MemoryUsed = "-"
	}

	return server, nil
}

// normalizeContainerState maps container states to simplified status strings
func normalizeContainerState(state string) string {
	state = strings.ToLower(state)
	switch state {
	case "running":
		return "running"
	case "created":
		return "created"
	case "stopped", "exited":
		return "stopped"
	case "paused":
		return "paused"
	default:
		return state
	}
}

// formatUptime formats a start time into a human-readable uptime string
func formatUptime(startedAt time.Time) string {
	if startedAt.IsZero() {
		return "-"
	}

	duration := time.Since(startedAt)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
