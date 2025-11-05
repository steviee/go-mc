package servers

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/state"
)

// RestartFlags holds all flags for the restart command
type RestartFlags struct {
	All     bool
	Wait    bool
	Timeout time.Duration
}

// NewRestartCommand creates the servers restart subcommand
func NewRestartCommand() *cobra.Command {
	flags := &RestartFlags{}

	cmd := &cobra.Command{
		Use:   "restart <name...>",
		Short: "Restart one or more Minecraft servers",
		Long: `Restart one or more Minecraft servers by restarting their containers.

This is equivalent to stopping and then starting the server. The container will
be sent a SIGTERM signal, given time to gracefully shut down, and then started again.

You can restart multiple servers by specifying multiple names, or use --all to restart
all running servers.

If a server is already stopped, it will be skipped with a warning.`,
		Example: `  # Restart a single server
  go-mc servers restart myserver

  # Restart multiple servers
  go-mc servers restart survival creative

  # Restart all running servers
  go-mc servers restart --all

  # Restart and wait for container to be fully running
  go-mc servers restart myserver --wait

  # Restart with custom timeout
  go-mc servers restart myserver --timeout 2m

  # JSON output for scripting
  go-mc servers restart myserver --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestart(cmd.Context(), cmd.OutOrStdout(), args, flags)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&flags.All, "all", false, "Restart all running servers")
	cmd.Flags().BoolVar(&flags.Wait, "wait", false, "Wait until containers are fully restarted")
	cmd.Flags().DurationVar(&flags.Timeout, "timeout", 60*time.Second, "Timeout for restart operation")

	return cmd
}

// runRestart executes the restart command
func runRestart(ctx context.Context, stdout io.Writer, args []string, flags *RestartFlags) error {
	jsonMode := isJSONMode()

	// Get list of servers to restart
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
		if err := restartServer(ctx, client, name, flags, result); err != nil {
			slog.Debug("error processing server", "name", name, "error", err)
		}
	}

	// Output results
	return outputOperationResult(stdout, "restarted", result)
}

// restartServer restarts a single server
func restartServer(ctx context.Context, client container.Client, name string, flags *RestartFlags, result *OperationResult) error {
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

	// Check if container is stopped (can't restart a stopped container)
	if isContainerStopped(info.State) {
		result.Skipped = append(result.Skipped, name)
		return nil
	}

	// Restart container
	timeout := flags.Timeout
	if err := client.RestartContainer(ctx, serverState.ContainerID, &timeout); err != nil {
		result.Failed[name] = err.Error()
		return err
	}

	// Wait if requested
	if flags.Wait {
		waitCtx, cancel := context.WithTimeout(ctx, flags.Timeout)
		defer cancel()

		if err := client.WaitForContainer(waitCtx, serverState.ContainerID, "running"); err != nil {
			// Container restarted but wait failed
			result.Failed[name] = "restarted but wait timed out: " + err.Error()
			// Still update state since container did restart
			_ = updateServerStatus(ctx, serverState, state.StatusRunning)
			return err
		}
	}

	// Update server state (status should remain running, but update timestamp)
	if err := updateServerStatus(ctx, serverState, state.StatusRunning); err != nil {
		result.Failed[name] = err.Error()
		return err
	}

	result.Success = append(result.Success, name)
	return nil
}
