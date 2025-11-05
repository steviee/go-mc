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

// StartFlags holds all flags for the start command
type StartFlags struct {
	All     bool
	Wait    bool
	Timeout time.Duration
}

// NewStartCommand creates the servers start subcommand
func NewStartCommand() *cobra.Command {
	flags := &StartFlags{}

	cmd := &cobra.Command{
		Use:   "start <name...>",
		Short: "Start one or more stopped Minecraft servers",
		Long: `Start one or more stopped Minecraft servers by starting their containers.

You can start multiple servers by specifying multiple names, or use --all to start
all stopped servers.

If a server is already running, it will be skipped with a warning.`,
		Example: `  # Start a single server
  go-mc servers start myserver

  # Start multiple servers
  go-mc servers start survival creative

  # Start all stopped servers
  go-mc servers start --all

  # Start and wait for container to be fully running
  go-mc servers start myserver --wait

  # Start with custom timeout
  go-mc servers start myserver --wait --timeout 2m

  # JSON output for scripting
  go-mc servers start myserver --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd.Context(), cmd.OutOrStdout(), args, flags)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&flags.All, "all", false, "Start all stopped servers")
	cmd.Flags().BoolVar(&flags.Wait, "wait", false, "Wait until containers are fully started")
	cmd.Flags().DurationVar(&flags.Timeout, "timeout", 60*time.Second, "Timeout for --wait")

	return cmd
}

// runStart executes the start command
func runStart(ctx context.Context, stdout io.Writer, args []string, flags *StartFlags) error {
	jsonMode := isJSONMode()

	// Get list of servers to start
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
		if err := startServer(ctx, client, name, flags, result); err != nil {
			slog.Debug("error processing server", "name", name, "error", err)
		}
	}

	// Output results
	return outputOperationResult(stdout, "started", result)
}

// startServer starts a single server
func startServer(ctx context.Context, client container.Client, name string, flags *StartFlags, result *OperationResult) error {
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

	// Check if already running
	if isContainerRunning(info.State) {
		result.Skipped = append(result.Skipped, name)
		return nil
	}

	// Start container
	if err := client.StartContainer(ctx, serverState.ContainerID); err != nil {
		result.Failed[name] = err.Error()
		return err
	}

	// Wait if requested
	if flags.Wait {
		waitCtx, cancel := context.WithTimeout(ctx, flags.Timeout)
		defer cancel()

		if err := client.WaitForContainer(waitCtx, serverState.ContainerID, "running"); err != nil {
			// Container started but wait failed
			result.Failed[name] = "started but wait timed out: " + err.Error()
			// Still update state since container did start
			_ = updateServerStatus(ctx, serverState, state.StatusRunning)
			return err
		}
	}

	// Update server state
	if err := updateServerStatus(ctx, serverState, state.StatusRunning); err != nil {
		result.Failed[name] = err.Error()
		return err
	}

	result.Success = append(result.Success, name)
	return nil
}

// isContainerRunning checks if a container state indicates it's running
func isContainerRunning(containerState string) bool {
	state := strings.ToLower(containerState)
	return state == "running"
}
