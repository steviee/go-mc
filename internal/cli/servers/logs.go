package servers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/state"
)

// LogsFlags holds all flags for the logs command
type LogsFlags struct {
	Follow     bool
	Tail       int
	Since      string
	Timestamps bool
}

// NewLogsCommand creates the servers logs subcommand
func NewLogsCommand() *cobra.Command {
	flags := &LogsFlags{}

	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "View container logs for a Minecraft server",
		Long: `View container logs for a Minecraft server to debug startup issues and monitor server output.

Logs are fetched directly from the container runtime (Podman/Docker).`,
		Example: `  # View last 100 lines of logs
  go-mc servers logs myserver

  # Follow logs in real-time
  go-mc servers logs myserver --follow

  # Show last 50 lines
  go-mc servers logs myserver --tail 50

  # Show logs from last 5 minutes
  go-mc servers logs myserver --since 5m

  # Include timestamps
  go-mc servers logs myserver --timestamps

  # Combine flags
  go-mc servers logs myserver --tail 20 --follow --timestamps`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(cmd.Context(), args[0], flags)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&flags.Follow, "follow", "f", false, "Follow log output (stream in real-time)")
	cmd.Flags().IntVarP(&flags.Tail, "tail", "n", 100, "Number of lines to show from the end of the logs")
	cmd.Flags().StringVar(&flags.Since, "since", "", "Show logs since timestamp (e.g. 5m, 1h, 2025-01-01)")
	cmd.Flags().BoolVarP(&flags.Timestamps, "timestamps", "t", false, "Show timestamps")

	return cmd
}

// runLogs executes the logs command
func runLogs(ctx context.Context, serverName string, flags *LogsFlags) error {
	// Load server state
	serverState, err := state.LoadServerState(ctx, serverName)
	if err != nil {
		return fmt.Errorf("failed to load server state: %w", err)
	}

	// Check if container exists
	if serverState.ContainerID == "" {
		return fmt.Errorf("server '%s' has no container (never started)", serverName)
	}

	// Try podman first, then docker
	runtime := detectContainerRuntime()

	// Build logs command
	args := []string{"logs"}

	if flags.Follow {
		args = append(args, "--follow")
	}

	if flags.Tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", flags.Tail))
	}

	if flags.Since != "" {
		args = append(args, "--since", flags.Since)
	}

	if flags.Timestamps {
		args = append(args, "--timestamps")
	}

	args = append(args, serverState.ContainerID)

	// Execute container runtime logs command
	cmd := exec.CommandContext(ctx, runtime, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Handle Ctrl+C gracefully for follow mode
	if flags.Follow {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigChan)

		// Start command
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start logs: %w", err)
		}

		// Wait for either command completion or interrupt signal
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-sigChan:
			// Interrupt received - kill process
			if err := cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill logs process: %w", err)
			}
			<-done // Wait for process to finish
			return nil
		case err := <-done:
			if err != nil {
				return fmt.Errorf("logs command failed: %w", err)
			}
			return nil
		}
	}

	// Non-follow mode - just run and wait
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}

	return nil
}

// detectContainerRuntime detects which container runtime is available
func detectContainerRuntime() string {
	// Try podman first
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman"
	}
	// Fall back to docker
	return "docker"
}
