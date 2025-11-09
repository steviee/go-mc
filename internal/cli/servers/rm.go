package servers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/state"
)

// RmFlags holds all flags for the rm command
type RmFlags struct {
	Force   bool
	Volumes bool
	All     bool
}

// RmOutput holds the output for JSON mode
type RmOutput struct {
	Status         string   `json:"status"`
	Removed        []string `json:"removed,omitempty"`
	Failed         []string `json:"failed,omitempty"`
	VolumesDeleted bool     `json:"volumes_deleted"`
	PortsReleased  []int    `json:"ports_released,omitempty"`
	Message        string   `json:"message,omitempty"`
	Error          string   `json:"error,omitempty"`
}

// NewRmCommand creates the servers rm subcommand
func NewRmCommand() *cobra.Command {
	flags := &RmFlags{}

	cmd := &cobra.Command{
		Use:   "rm <name...>",
		Short: "Remove one or more Minecraft servers",
		Long: `Remove one or more Minecraft servers, including their containers and optionally their data.

By default, server data is preserved in volumes. Use --volumes to permanently delete all data.

If a server is running, it will be stopped first before removal.`,
		Example: `  # Remove a server (preserves data)
  go-mc servers rm old-server

  # Remove server including all data
  go-mc servers rm old-server --volumes

  # Force remove without confirmation
  go-mc servers rm old-server --force --volumes

  # Remove multiple servers
  go-mc servers rm server1 server2 server3

  # Remove all stopped servers
  go-mc servers rm --all

  # JSON output
  go-mc servers rm old-server --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(cmd.Context(), cmd.OutOrStdout(), cmd.InOrStdin(), args, flags)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&flags.Force, "force", "f", false, "Skip confirmation prompts")
	cmd.Flags().BoolVarP(&flags.Volumes, "volumes", "v", false, "Also remove server data volumes")
	cmd.Flags().BoolVar(&flags.All, "all", false, "Remove all stopped servers")

	return cmd
}

// runRm executes the rm command
func runRm(ctx context.Context, stdout io.Writer, stdin io.Reader, args []string, flags *RmFlags) error {
	jsonMode := isJSONMode()

	// Get list of servers to remove
	serverNames, err := getServerNamesFromArgs(ctx, args, flags.All)
	if err != nil {
		return outputRmError(stdout, jsonMode, err)
	}

	// Confirmation prompt (unless --force or --json)
	if !flags.Force && !jsonMode {
		confirmed, err := confirmRemoval(stdin, stdout, serverNames, flags.Volumes)
		if err != nil {
			return outputRmError(stdout, jsonMode, err)
		}
		if !confirmed {
			_, _ = fmt.Fprintln(stdout, "Removal cancelled")
			return nil
		}
	}

	// Get container client
	client, err := createContainerClient(ctx)
	if err != nil {
		return outputRmError(stdout, jsonMode, err)
	}
	defer func() { _ = client.Close() }()

	// Track results
	removed := []string{}
	failed := []string{}
	allPortsReleased := []int{}

	// Process each server
	for _, name := range serverNames {
		ports, err := removeServer(ctx, client, name, flags)
		if err != nil {
			slog.Error("failed to remove server", "name", name, "error", err)
			failed = append(failed, name)
		} else {
			removed = append(removed, name)
			allPortsReleased = append(allPortsReleased, ports...)
		}
	}

	// Output results
	return outputRmSuccess(stdout, jsonMode, removed, failed, flags.Volumes, allPortsReleased)
}

// confirmRemoval prompts the user for confirmation
func confirmRemoval(stdin io.Reader, stdout io.Writer, serverNames []string, deleteVolumes bool) (bool, error) {
	if deleteVolumes {
		// Extra warning for data deletion
		_, _ = fmt.Fprintf(stdout, "\n⚠️  WARNING: This will permanently delete ALL data for %d server(s)!\n", len(serverNames))
		_, _ = fmt.Fprintf(stdout, "   Servers: %s\n", strings.Join(serverNames, ", "))
		_, _ = fmt.Fprintln(stdout, "\n   This action CANNOT be undone!")
		_, _ = fmt.Fprintln(stdout)

		// Require typing "yes" for volume deletion
		_, _ = fmt.Fprint(stdout, "Type 'yes' to confirm: ")
		scanner := bufio.NewScanner(stdin)
		if !scanner.Scan() {
			return false, fmt.Errorf("failed to read confirmation")
		}
		response := strings.TrimSpace(scanner.Text())
		return strings.ToLower(response) == "yes", nil
	}

	// Standard confirmation
	_, _ = fmt.Fprintf(stdout, "\n⚠️  This will remove %d server(s):\n", len(serverNames))
	_, _ = fmt.Fprintf(stdout, "   %s\n", strings.Join(serverNames, ", "))
	_, _ = fmt.Fprintln(stdout, "   Data will be preserved (use --volumes to delete)")
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprint(stdout, "Remove server(s)? [y/N]: ")

	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		return false, fmt.Errorf("failed to read confirmation")
	}
	response := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return response == "y" || response == "yes", nil
}

// removeServer removes a single server
func removeServer(ctx context.Context, client container.Client, name string, flags *RmFlags) ([]int, error) {
	releasedPorts := []int{}

	// Load server state
	serverState, err := state.LoadServerState(ctx, name)
	if err != nil {
		return releasedPorts, fmt.Errorf("failed to load server state: %w", err)
	}

	// Collect ports to release
	if serverState.Minecraft.GamePort > 0 {
		releasedPorts = append(releasedPorts, serverState.Minecraft.GamePort)
	}
	if serverState.Minecraft.RconPort > 0 {
		releasedPorts = append(releasedPorts, serverState.Minecraft.RconPort)
	}
	for _, mod := range serverState.Mods {
		if mod.Port > 0 {
			releasedPorts = append(releasedPorts, mod.Port)
		}
	}

	// If container exists, handle it
	if serverState.ContainerID != "" {
		// Check container status
		info, err := client.InspectContainer(ctx, serverState.ContainerID)
		if err == nil {
			// Container exists - stop if running
			if isContainerRunning(info.Status) {
				slog.Info("stopping running container", "server", name, "container_id", serverState.ContainerID)
				stopTimeout := 30 * time.Second
				stopCtx, cancel := context.WithTimeout(ctx, stopTimeout+10*time.Second)
				defer cancel()
				if err := client.StopContainer(stopCtx, serverState.ContainerID, &stopTimeout); err != nil {
					slog.Warn("failed to stop container gracefully", "error", err)
				}
			}

			// Remove container
			slog.Info("removing container", "server", name, "container_id", serverState.ContainerID)
			removeOpts := &container.RemoveOptions{
				Force:         true,
				RemoveVolumes: false, // We handle volumes separately
			}
			if err := client.RemoveContainer(ctx, serverState.ContainerID, removeOpts); err != nil {
				slog.Warn("failed to remove container", "error", err)
			}
		}
	}

	// Remove volumes if requested
	if flags.Volumes {
		slog.Info("removing server directories", "server", name)
		if err := removeServerDirectories(serverState); err != nil {
			slog.Warn("failed to remove server directories", "error", err)
		}
	}

	// Release ports
	for _, port := range releasedPorts {
		if err := state.ReleasePort(ctx, port); err != nil {
			slog.Warn("failed to release port", "port", port, "error", err)
		} else {
			slog.Info("released port", "port", port)
		}
	}

	// Remove server state file
	if err := state.DeleteServerState(ctx, name); err != nil {
		return releasedPorts, fmt.Errorf("failed to delete server state: %w", err)
	}

	// Remove server from global registry
	globalState, err := state.LoadGlobalState(ctx)
	if err != nil {
		return releasedPorts, fmt.Errorf("failed to load global state: %w", err)
	}

	// Remove server from registry
	newServers := []string{}
	for _, s := range globalState.Servers {
		if s != name {
			newServers = append(newServers, s)
		}
	}
	globalState.Servers = newServers

	if err := state.SaveGlobalState(ctx, globalState); err != nil {
		return releasedPorts, fmt.Errorf("failed to update global state: %w", err)
	}

	slog.Info("server removed", "name", name)
	return releasedPorts, nil
}

// removeServerDirectories removes all server directories and data
func removeServerDirectories(serverState *state.ServerState) error {
	// Get XDG data home
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dataHome = homeDir + "/.local/share"
	}

	// Server directory is the parent of the data volume
	serverDir := dataHome + "/go-mc/servers/" + serverState.Name

	slog.Info("deleting server directory", "path", serverDir)
	if err := os.RemoveAll(serverDir); err != nil {
		return fmt.Errorf("failed to remove server directory: %w", err)
	}

	return nil
}

// outputRmSuccess outputs the removal results
func outputRmSuccess(stdout io.Writer, jsonMode bool, removed, failed []string, volumesDeleted bool, portsReleased []int) error {
	if jsonMode {
		status := "success"
		if len(removed) == 0 {
			status = "error"
		} else if len(failed) > 0 {
			status = "partial"
		}

		output := RmOutput{
			Status:         status,
			Removed:        removed,
			Failed:         failed,
			VolumesDeleted: volumesDeleted,
			PortsReleased:  portsReleased,
			Message:        fmt.Sprintf("Removed %d server(s)", len(removed)),
		}
		return json.NewEncoder(stdout).Encode(output)
	}

	// Human-readable output
	if len(removed) == 0 && len(failed) > 0 {
		_, _ = fmt.Fprintln(stdout, "\n❌ Failed to remove any servers")
		for _, name := range failed {
			_, _ = fmt.Fprintf(stdout, "  • %s\n", name)
		}
		return fmt.Errorf("failed to remove %d server(s)", len(failed))
	}

	_, _ = fmt.Fprintln(stdout, "\n✅ Server removal complete:")
	_, _ = fmt.Fprintln(stdout)

	if len(removed) > 0 {
		_, _ = fmt.Fprintf(stdout, "Removed %d server(s):\n", len(removed))
		for _, name := range removed {
			_, _ = fmt.Fprintf(stdout, "  • %s\n", name)
		}
		_, _ = fmt.Fprintln(stdout)
	}

	if len(portsReleased) > 0 {
		_, _ = fmt.Fprintf(stdout, "Released %d port(s): %v\n", len(portsReleased), portsReleased)
		_, _ = fmt.Fprintln(stdout)
	}

	if volumesDeleted {
		_, _ = fmt.Fprintln(stdout, "⚠️  All server data has been permanently deleted")
	} else {
		_, _ = fmt.Fprintln(stdout, "📁 Server data preserved in ~/.local/share/go-mc/servers/")
		_, _ = fmt.Fprintln(stdout, "   Use --volumes flag to delete data permanently")
	}

	if len(failed) > 0 {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintf(stdout, "⚠️  Failed to remove %d server(s):\n", len(failed))
		for _, name := range failed {
			_, _ = fmt.Fprintf(stdout, "  • %s\n", name)
		}
	}

	return nil
}

// outputRmError outputs an error message
func outputRmError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := RmOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}
