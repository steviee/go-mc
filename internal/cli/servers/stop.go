package servers

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/state"
)

// StopFlags holds all flags for the stop command
type StopFlags struct {
	All     bool
	Force   bool
	Timeout time.Duration
}

// NewStopCommand creates the servers stop subcommand
func NewStopCommand() *cobra.Command {
	flags := &StopFlags{}

	cmd := &cobra.Command{
		Use:   "stop <name...>",
		Short: "Stop one or more running Minecraft servers",
		Long: `Stop one or more running Minecraft servers by stopping their containers.

The container will be sent a SIGTERM signal and given time to gracefully shut down.
If the timeout expires, SIGKILL will be sent.

You can stop multiple servers by specifying multiple names, or use --all to stop
all running servers.

If a server is already stopped, it will be skipped with a warning.`,
		Example: `  # Stop a single server
  go-mc servers stop myserver

  # Stop multiple servers
  go-mc servers stop survival creative

  # Stop all running servers
  go-mc servers stop --all

  # Force immediate stop (SIGKILL)
  go-mc servers stop myserver --force

  # Stop with custom graceful shutdown timeout
  go-mc servers stop myserver --timeout 60s

  # JSON output for scripting
  go-mc servers stop myserver --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(cmd.Context(), cmd.OutOrStdout(), args, flags)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&flags.All, "all", false, "Stop all running servers")
	cmd.Flags().BoolVar(&flags.Force, "force", false, "Force immediate stop (SIGKILL)")
	cmd.Flags().DurationVar(&flags.Timeout, "timeout", 30*time.Second, "Graceful shutdown timeout")

	return cmd
}

// runStop executes the stop command
func runStop(ctx context.Context, stdout io.Writer, args []string, flags *StopFlags) error {
	jsonMode := isJSONMode()

	// Get list of servers to stop
	serverNames, err := getServerNamesFromArgs(ctx, args, flags.All)
	if err != nil {
		return outputLifecycleError(stdout, jsonMode, err)
	}

	// Create result tracker
	result := NewOperationResult()

	// Get container client
	client, err := createContainerClient(ctx)
	if err != nil {
		return outputLifecycleError(stdout, jsonMode, err)
	}
	defer func() { _ = client.Close() }()

	// Process each server
	for _, name := range serverNames {
		if err := stopServer(ctx, client, name, flags, result); err != nil {
			slog.Debug("error processing server", "name", name, "error", err)
		}
	}

	// Output results
	return outputOperationResult(stdout, "stopped", result)
}

// stopServer stops a single server
func stopServer(ctx context.Context, client container.Client, name string, flags *StopFlags, result *OperationResult) error {
	// Load server state
	serverState, err := loadServerForOperation(ctx, name)
	if err != nil {
		result.Failed[name] = err.Error()
		return err
	}

	// Check if container exists
	info, err := checkContainerExists(ctx, client, serverState.ContainerID)
	if err != nil {
		result.Failed[name] = err.Error()
		return err
	}

	// Check if already stopped
	if isContainerStopped(info.State) {
		result.Skipped = append(result.Skipped, name)
		return nil
	}

	// Determine timeout
	timeout := flags.Timeout
	if flags.Force {
		// Force immediate stop with zero timeout
		timeout = 0
	}

	// Stop container
	if err := client.StopContainer(ctx, serverState.ContainerID, &timeout); err != nil {
		result.Failed[name] = err.Error()
		return err
	}

	// Update server state
	if err := updateServerStatus(ctx, serverState, state.StatusStopped); err != nil {
		result.Failed[name] = err.Error()
		return err
	}

	result.Success = append(result.Success, name)
	return nil
}

// isContainerStopped checks if a container state indicates it's stopped
func isContainerStopped(containerState string) bool {
	state := strings.ToLower(containerState)
	return state == "stopped" || state == "exited" || state == "created"
}
