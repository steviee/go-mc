package servers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/backup"
	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/state"
)

// RestoreFlags holds flags for the restore command.
type RestoreFlags struct {
	Force bool
	Start bool
}

// RestoreOutput holds the output for JSON mode.
type RestoreOutput struct {
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NewRestoreCommand creates the servers restore subcommand.
func NewRestoreCommand() *cobra.Command {
	flags := &RestoreFlags{}

	cmd := &cobra.Command{
		Use:   "restore <server-name> <backup-id>",
		Short: "Restore a server from a backup",
		Long: `Restore a server's data and mods from a previously created backup.

This operation:
  1. Stops the server if it's running
  2. Backs up current data (for rollback on failure)
  3. Extracts and restores the backup
  4. Optionally starts the server after restore

If any step fails, the server is rolled back to its previous state.

IMPORTANT: This operation will overwrite the server's current data!`,
		Example: `  # Restore from a specific backup
  go-mc servers restore myserver backup-myserver-2025-01-20-15-30-00

  # Force restore without confirmation
  go-mc servers restore myserver backup-myserver-2025-01-20-15-30-00 --force

  # Restore and start server
  go-mc servers restore myserver backup-myserver-2025-01-20-15-30-00 --start

  # List available backups first
  go-mc servers backup myserver --list`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverName := args[0]
			backupID := args[1]
			return runRestore(cmd.Context(), cmd.OutOrStdout(), os.Stdin, serverName, backupID, flags)
		},
	}

	cmd.Flags().BoolVarP(&flags.Force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&flags.Start, "start", false, "Start server after restore")

	return cmd
}

// runRestore executes the restore command.
func runRestore(ctx context.Context, stdout io.Writer, stdin io.Reader, serverName, backupID string, flags *RestoreFlags) error {
	jsonMode := isJSONMode()

	// Validate server name
	if err := state.ValidateServerName(serverName); err != nil {
		return outputRestoreError(stdout, jsonMode, fmt.Errorf("invalid server name: %w", err))
	}

	// Load server state
	serverState, err := state.LoadServerState(ctx, serverName)
	if err != nil {
		return outputRestoreError(stdout, jsonMode, fmt.Errorf("failed to load server: %w", err))
	}

	// Get backup info
	backupInfo, err := state.GetBackup(ctx, backupID)
	if err != nil {
		return outputRestoreError(stdout, jsonMode, fmt.Errorf("failed to get backup: %w", err))
	}

	// Verify backup is for the correct server
	if backupInfo.Server != serverName {
		return outputRestoreError(stdout, jsonMode,
			fmt.Errorf("backup is for server %q, not %q", backupInfo.Server, serverName))
	}

	// Show backup info and confirm (unless --force)
	if !flags.Force && !jsonMode {
		_, _ = fmt.Fprintf(stdout, "Restore server %q from backup:\n", serverName)
		_, _ = fmt.Fprintf(stdout, "  Backup ID:        %s\n", backupInfo.ID)
		_, _ = fmt.Fprintf(stdout, "  Minecraft Version: %s\n", backupInfo.MinecraftVersion)
		_, _ = fmt.Fprintf(stdout, "  Created:          %s\n", backupInfo.CreatedAt.Format("2006-01-02 15:04:05"))
		_, _ = fmt.Fprintf(stdout, "  Size:             %s\n", formatBytes(backupInfo.SizeBytes))
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "WARNING: This will overwrite the server's current data!")
		_, _ = fmt.Fprint(stdout, "Continue? (y/N): ")

		reader := bufio.NewReader(stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return outputRestoreError(stdout, jsonMode, fmt.Errorf("failed to read confirmation: %w", err))
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			_, _ = fmt.Fprintln(stdout, "Restore cancelled")
			return nil
		}
		_, _ = fmt.Fprintln(stdout)
	}

	// Create container client
	containerClient, err := container.NewClient(ctx, container.DefaultConfig())
	if err != nil {
		return outputRestoreError(stdout, jsonMode, fmt.Errorf("failed to create container client: %w", err))
	}
	defer containerClient.Close()

	// Stop server if running
	wasRunning := false
	if serverState.Status == state.StatusRunning {
		if !jsonMode {
			_, _ = fmt.Fprintf(stdout, "Stopping server %q...\n", serverName)
		}

		timeout := 30 * time.Second
		if err := containerClient.StopContainer(ctx, serverName, &timeout); err != nil {
			return outputRestoreError(stdout, jsonMode, fmt.Errorf("failed to stop server: %w", err))
		}

		// Update server state
		serverState.Status = state.StatusStopped
		if err := state.SaveServerState(ctx, serverState); err != nil {
			return outputRestoreError(stdout, jsonMode, fmt.Errorf("failed to update server state: %w", err))
		}

		wasRunning = true
	}

	// Perform restore
	if !jsonMode {
		_, _ = fmt.Fprintf(stdout, "Restoring from backup %s...\n", backupID)
	}

	backupService := backup.NewService()
	if err := backupService.RestoreBackup(ctx, backup.RestoreBackupOptions{
		BackupID:   backupID,
		ServerName: serverName,
		Force:      flags.Force,
	}); err != nil {
		return outputRestoreError(stdout, jsonMode, fmt.Errorf("restore failed: %w", err))
	}

	// Start server if requested or if it was running before
	shouldStart := flags.Start || wasRunning
	if shouldStart {
		if !jsonMode {
			_, _ = fmt.Fprintf(stdout, "Starting server %q...\n", serverName)
		}

		if err := containerClient.StartContainer(ctx, serverName); err != nil {
			// Don't fail the restore, just warn
			if !jsonMode {
				_, _ = fmt.Fprintf(stdout, "Warning: Failed to start server: %v\n", err)
			}
		} else {
			// Update server state
			serverState.Status = state.StatusRunning
			if err := state.SaveServerState(ctx, serverState); err != nil {
				// Non-fatal
				_ = err
			}
		}
	}

	// Output success
	return outputRestoreSuccess(stdout, jsonMode, serverName, backupID, shouldStart)
}

// outputRestoreSuccess outputs restore success result.
func outputRestoreSuccess(stdout io.Writer, jsonMode bool, serverName, backupID string, started bool) error {
	if jsonMode {
		output := RestoreOutput{
			Status: "success",
			Data: map[string]interface{}{
				"server":    serverName,
				"backup_id": backupID,
				"started":   started,
			},
			Message: fmt.Sprintf("Server %q restored from backup %q", serverName, backupID),
		}
		return json.NewEncoder(stdout).Encode(output)
	}

	// Human-readable output
	_, _ = fmt.Fprintf(stdout, "\n✓ Server %q successfully restored from backup %s\n", serverName, backupID)
	if started {
		_, _ = fmt.Fprintln(stdout, "✓ Server started")
	}
	_, _ = fmt.Fprintln(stdout)

	return nil
}

// outputRestoreError outputs an error message.
func outputRestoreError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := RestoreOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}
