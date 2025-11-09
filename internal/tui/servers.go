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
		Name:          name,
		Status:        "unknown",
		InstalledMods: []ModInfo{},
		Ports:         []PortInfo{},
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
	server.FabricVersion = serverState.Minecraft.FabricLoaderVersion
	server.RCONPort = serverState.Minecraft.RconPort

	// Convert installed mods
	if len(serverState.Mods) > 0 {
		installedMods := make([]ModInfo, len(serverState.Mods))
		for i, mod := range serverState.Mods {
			installedMods[i] = ModInfo{
				Name:    mod.Name,
				Slug:    mod.Slug,
				Version: mod.Version,
			}
		}
		server.InstalledMods = installedMods
	}

	// Detect ports
	server.Ports = detectPorts(serverState)

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

	// Calculate uptime and get stats for running containers
	if server.Status == "running" && !serverState.LastStarted.IsZero() {
		server.StartedAt = serverState.LastStarted
		server.Uptime = formatUptime(serverState.LastStarted)

		// Get container stats (CPU and memory usage)
		stats, err := client.GetContainerStats(ctx, serverState.ContainerID)
		if err == nil {
			server.CPUPercent = stats.CPUPercent
			server.MemoryUsedBytes = stats.MemoryUsed
			server.MemoryLimitBytes = stats.MemoryLimit
			server.MemoryPercent = stats.MemoryPercent

			// Update MemoryUsed string with actual usage
			server.MemoryUsed = formatBytes(stats.MemoryUsed)
		} else {
			// Log warning but continue (stats might not be available)
			slog.Warn("failed to get container stats", "server", name, "error", err)
			server.MemoryUsed = "-"
		}
	} else {
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

// detectPorts detects all ports used by a server
func detectPorts(serverState *state.ServerState) []PortInfo {
	ports := []PortInfo{}

	// Game port (always present)
	if serverState.Minecraft.GamePort != 0 {
		ports = append(ports, PortInfo{
			Number:   serverState.Minecraft.GamePort,
			Protocol: "tcp",
			Service:  "Minecraft",
			Source:   "config",
		})
	}

	// RCON port (always present)
	if serverState.Minecraft.RconPort != 0 {
		ports = append(ports, PortInfo{
			Number:   serverState.Minecraft.RconPort,
			Protocol: "tcp",
			Service:  "RCON",
			Source:   "config",
		})
	}

	// Mod-specific ports (detect from known mods)
	modPorts := detectModPorts(serverState.Mods)
	ports = append(ports, modPorts...)

	return ports
}

// detectModPorts detects ports required by installed mods
func detectModPorts(mods []state.ModInfo) []PortInfo {
	var ports []PortInfo

	// Known mod port mappings
	knownModPorts := map[string]PortInfo{
		"simple-voice-chat": {
			Number:   24454,
			Protocol: "udp",
			Service:  "Simple Voice Chat",
			Source:   "mod",
		},
		"geyser": {
			Number:   19132,
			Protocol: "udp",
			Service:  "Geyser (Bedrock)",
			Source:   "mod",
		},
		"bluemap": {
			Number:   8100,
			Protocol: "tcp",
			Service:  "BlueMap",
			Source:   "mod",
		},
	}

	// Check each installed mod
	for _, mod := range mods {
		if portInfo, found := knownModPorts[mod.Slug]; found {
			ports = append(ports, portInfo)
		}
	}

	return ports
}

// formatBytes formats bytes into human-readable format (KB, MB, GB)
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	if bytes >= GB {
		return fmt.Sprintf("%.1fG", float64(bytes)/float64(GB))
	}
	if bytes >= MB {
		return fmt.Sprintf("%.1fM", float64(bytes)/float64(MB))
	}
	if bytes >= KB {
		return fmt.Sprintf("%.1fK", float64(bytes)/float64(KB))
	}
	return fmt.Sprintf("%dB", bytes)
}
