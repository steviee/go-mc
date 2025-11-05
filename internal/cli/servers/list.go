package servers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/state"
)

// ListFlags holds all flags for the list command
type ListFlags struct {
	All      bool
	Filter   string
	Sort     string
	NoHeader bool
}

// ServerListItem represents a server in the list output
type ServerListItem struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Version     string    `json:"version"`
	Port        int       `json:"port"`
	MemoryUsed  string    `json:"memory_used,omitempty"`
	MemoryTotal string    `json:"memory_total"`
	Uptime      string    `json:"uptime,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
}

// ListOutput holds the output for JSON mode
type ListOutput struct {
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NewListCommand creates the servers list subcommand
func NewListCommand() *cobra.Command {
	flags := &ListFlags{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Minecraft Fabric servers",
		Long: `List all Minecraft Fabric servers with their status and resource usage.

By default, only running servers are shown. Use --all to show all servers
including stopped ones.

The output can be filtered by status and sorted by various fields.`,
		Example: `  # List all running servers
  go-mc servers list

  # List all servers including stopped ones
  go-mc servers list --all

  # Filter by status
  go-mc servers list --filter=stopped

  # Sort by memory usage
  go-mc servers list --sort=memory

  # JSON output for scripting
  go-mc servers list --json

  # Omit table header
  go-mc servers list --no-header`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), cmd.OutOrStdout(), flags)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&flags.All, "all", "a", false, "Show all servers including stopped ones")
	cmd.Flags().StringVar(&flags.Filter, "filter", "", "Filter by status (running, created, stopped, all)")
	cmd.Flags().StringVar(&flags.Sort, "sort", "name", "Sort by field (name, status, port, memory, uptime)")
	cmd.Flags().BoolVar(&flags.NoHeader, "no-header", false, "Omit table header")

	return cmd
}

// runList executes the list command
func runList(ctx context.Context, stdout io.Writer, flags *ListFlags) error {
	jsonMode := isJSONMode()

	// Get list of registered servers from global state
	serverNames, err := state.ListServers(ctx)
	if err != nil {
		return outputListError(stdout, jsonMode, fmt.Errorf("failed to load servers: %w", err))
	}

	// If no servers exist, show helpful message
	if len(serverNames) == 0 {
		if jsonMode {
			output := ListOutput{
				Status: "success",
				Data: map[string]interface{}{
					"servers": []ServerListItem{},
					"count":   0,
				},
				Message: "No servers found",
			}
			return json.NewEncoder(stdout).Encode(output)
		}

		_, _ = fmt.Fprintln(stdout, "No servers found. Create one with 'go-mc servers create <name>'")
		return nil
	}

	// Create container client
	containerClient, err := container.NewClient(ctx, container.DefaultConfig())
	if err != nil {
		return outputListError(stdout, jsonMode, fmt.Errorf("failed to create container client: %w", err))
	}
	defer func() { _ = containerClient.Close() }()

	// Collect server information
	items := make([]ServerListItem, 0, len(serverNames))
	for _, name := range serverNames {
		item, err := collectServerInfo(ctx, name, containerClient)
		if err != nil {
			// Log warning but continue with other servers
			slog.Warn("failed to collect server info", "server", name, "error", err)
			continue
		}

		items = append(items, item)
	}

	// Apply filtering
	items = filterServers(items, flags)

	// Check if filter excluded all servers
	if len(items) == 0 {
		if jsonMode {
			output := ListOutput{
				Status: "success",
				Data: map[string]interface{}{
					"servers": []ServerListItem{},
					"count":   0,
				},
				Message: "No servers match the filter",
			}
			return json.NewEncoder(stdout).Encode(output)
		}

		_, _ = fmt.Fprintln(stdout, "No servers match the filter.")
		return nil
	}

	// Apply sorting
	sortServers(items, flags.Sort)

	// Output results
	if jsonMode {
		return outputListJSON(stdout, items)
	}

	return outputListTable(stdout, items, flags.NoHeader)
}

// collectServerInfo collects information about a single server
func collectServerInfo(ctx context.Context, name string, client container.Client) (ServerListItem, error) {
	item := ServerListItem{
		Name:   name,
		Status: "unknown",
	}

	// Load server state
	serverState, err := state.LoadServerState(ctx, name)
	if err != nil {
		return item, fmt.Errorf("failed to load server state: %w", err)
	}

	// Set basic info from state
	item.Version = serverState.Minecraft.Version
	item.Port = serverState.Minecraft.GamePort
	item.MemoryTotal = serverState.Minecraft.Memory

	// If no container ID, server was never started
	if serverState.ContainerID == "" {
		item.Status = "created"
		return item, nil
	}

	// Inspect container to get current status
	containerInfo, err := client.InspectContainer(ctx, serverState.ContainerID)
	if err != nil {
		// Container no longer exists
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no such container") {
			item.Status = "missing"
			return item, nil
		}
		return item, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Map container state to our status
	item.Status = normalizeContainerState(containerInfo.State)

	// Calculate uptime for running containers
	if item.Status == "running" && !serverState.LastStarted.IsZero() {
		item.StartedAt = serverState.LastStarted
		item.Uptime = formatUptime(serverState.LastStarted)
		// Note: Memory usage stats are not available through current container client
		// We show total allocation instead
		item.MemoryUsed = "-"
	}

	return item, nil
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

// filterServers applies filtering based on flags
func filterServers(items []ServerListItem, flags *ListFlags) []ServerListItem {
	// If --all or --filter=all, show everything
	if flags.All || flags.Filter == "all" {
		return items
	}

	// If no filter and not --all, show only running
	filter := flags.Filter
	if filter == "" {
		filter = "running"
	}

	filtered := make([]ServerListItem, 0, len(items))
	for _, item := range items {
		switch filter {
		case "running":
			if item.Status == "running" {
				filtered = append(filtered, item)
			}
		case "stopped":
			if item.Status == "stopped" || item.Status == "created" || item.Status == "exited" {
				filtered = append(filtered, item)
			}
		case "created":
			if item.Status == "created" {
				filtered = append(filtered, item)
			}
		default:
			// Unknown filter, show everything
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// sortServers sorts the server list by the specified field
func sortServers(items []ServerListItem, sortBy string) {
	switch sortBy {
	case "name":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})
	case "status":
		sort.Slice(items, func(i, j int) bool {
			if items[i].Status != items[j].Status {
				return items[i].Status < items[j].Status
			}
			return items[i].Name < items[j].Name
		})
	case "port":
		sort.Slice(items, func(i, j int) bool {
			if items[i].Port != items[j].Port {
				return items[i].Port < items[j].Port
			}
			return items[i].Name < items[j].Name
		})
	case "memory":
		sort.Slice(items, func(i, j int) bool {
			// Running containers first, then by name
			if items[i].Status == "running" && items[j].Status != "running" {
				return true
			}
			if items[i].Status != "running" && items[j].Status == "running" {
				return false
			}
			return items[i].Name < items[j].Name
		})
	case "uptime":
		sort.Slice(items, func(i, j int) bool {
			// Sort by started time (earliest first = longest uptime)
			if !items[i].StartedAt.IsZero() && !items[j].StartedAt.IsZero() {
				return items[i].StartedAt.Before(items[j].StartedAt)
			}
			if !items[i].StartedAt.IsZero() {
				return true
			}
			if !items[j].StartedAt.IsZero() {
				return false
			}
			return items[i].Name < items[j].Name
		})
	default:
		// Default to name sorting
		sort.Slice(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})
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

// outputListTable outputs servers in table format
func outputListTable(stdout io.Writer, items []ServerListItem, noHeader bool) error {
	// Calculate column widths
	nameWidth := len("NAME")
	statusWidth := len("STATUS")
	versionWidth := len("VERSION")
	portWidth := len("PORT")
	memoryWidth := len("MEMORY")
	uptimeWidth := len("UPTIME")

	for _, item := range items {
		if len(item.Name) > nameWidth {
			nameWidth = len(item.Name)
		}
		if len(item.Status) > statusWidth {
			statusWidth = len(item.Status)
		}
		if len(item.Version) > versionWidth {
			versionWidth = len(item.Version)
		}
		portStr := fmt.Sprintf("%d", item.Port)
		if len(portStr) > portWidth {
			portWidth = len(portStr)
		}

		memoryStr := formatMemoryDisplay(item)
		if len(memoryStr) > memoryWidth {
			memoryWidth = len(memoryStr)
		}

		if len(item.Uptime) > uptimeWidth {
			uptimeWidth = len(item.Uptime)
		}
	}

	// Print header
	if !noHeader {
		_, _ = fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %*s  %*s  %*s\n",
			nameWidth, "NAME",
			statusWidth, "STATUS",
			versionWidth, "VERSION",
			portWidth, "PORT",
			memoryWidth, "MEMORY",
			uptimeWidth, "UPTIME",
		)
	}

	// Print rows
	for _, item := range items {
		uptime := item.Uptime
		if uptime == "" {
			uptime = "-"
		}

		_, _ = fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %*d  %*s  %*s\n",
			nameWidth, item.Name,
			statusWidth, item.Status,
			versionWidth, item.Version,
			portWidth, item.Port,
			memoryWidth, formatMemoryDisplay(item),
			uptimeWidth, uptime,
		)
	}

	return nil
}

// formatMemoryDisplay formats memory info for table display
func formatMemoryDisplay(item ServerListItem) string {
	if item.Status == "running" {
		// For running containers, show usage/total
		// Since we don't have real-time stats, show "-/total"
		return fmt.Sprintf("-/%s", item.MemoryTotal)
	}
	// For non-running containers, just show dash
	return "-"
}

// outputListJSON outputs servers in JSON format
func outputListJSON(stdout io.Writer, items []ServerListItem) error {
	output := ListOutput{
		Status: "success",
		Data: map[string]interface{}{
			"servers": items,
			"count":   len(items),
		},
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// outputListError outputs an error message
func outputListError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := ListOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}
