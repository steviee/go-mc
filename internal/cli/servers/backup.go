package servers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/backup"
	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/state"
)

// BackupFlags holds flags for the backup command.
type BackupFlags struct {
	All    bool
	List   bool
	Output string
	Keep   int
}

// BackupOutput holds the output for JSON mode.
type BackupOutput struct {
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NewBackupCommand creates the servers backup subcommand.
func NewBackupCommand() *cobra.Command {
	flags := &BackupFlags{}

	cmd := &cobra.Command{
		Use:   "backup <server-name>",
		Short: "Create a backup of a server",
		Long: `Create a compressed backup of a server's data and mods directories.

Backups are stored in ~/.config/go-mc/backups/archives/ as compressed tar.gz files.
A backup registry tracks all backups with metadata (version, size, date, etc.).

Automatic retention policy keeps only the last N backups (default: 5).`,
		Example: `  # Backup a single server
  go-mc servers backup myserver

  # Backup all servers
  go-mc servers backup --all

  # List available backups for a server
  go-mc servers backup myserver --list

  # Custom retention policy (keep last 10 backups)
  go-mc servers backup myserver --keep 10

  # JSON output
  go-mc servers backup myserver --json`,
		Args: func(cmd *cobra.Command, args []string) error {
			// --list or --all can be used without args
			if flags.List || flags.All {
				if len(args) > 1 {
					return fmt.Errorf("too many arguments")
				}
				return nil
			}
			// Otherwise, exactly one server name is required
			if len(args) != 1 {
				return fmt.Errorf("requires server name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var serverName string
			if len(args) > 0 {
				serverName = args[0]
			}
			return runBackup(cmd.Context(), cmd.OutOrStdout(), serverName, flags)
		},
	}

	cmd.Flags().BoolVarP(&flags.All, "all", "a", false, "Backup all servers")
	cmd.Flags().BoolVar(&flags.List, "list", false, "List available backups")
	cmd.Flags().StringVarP(&flags.Output, "output", "o", "", "Output directory (default: ~/.config/go-mc/backups/archives/)")
	cmd.Flags().IntVar(&flags.Keep, "keep", 5, "Keep last N backups per server")

	return cmd
}

// runBackup executes the backup command.
func runBackup(ctx context.Context, stdout io.Writer, serverName string, flags *BackupFlags) error {
	jsonMode := isJSONMode()

	// Handle --list flag
	if flags.List {
		return runBackupList(ctx, stdout, serverName, jsonMode)
	}

	// Get list of servers to backup
	var serverNames []string
	if flags.All {
		names, err := state.ListServers(ctx)
		if err != nil {
			return outputBackupError(stdout, jsonMode, fmt.Errorf("failed to list servers: %w", err))
		}
		serverNames = names
	} else {
		serverNames = []string{serverName}
	}

	if len(serverNames) == 0 {
		return outputBackupError(stdout, jsonMode, fmt.Errorf("no servers found"))
	}

	// Check if server is running and warn
	if !flags.All && len(serverNames) == 1 {
		if err := warnIfServerRunning(ctx, serverNames[0], stdout, jsonMode); err != nil {
			// Non-fatal, just log warning
			_ = err
		}
	}

	// Create backup service
	backupService := backup.NewService()

	// Backup each server
	results := make(map[string]*backup.CreateBackupResult)
	var errors []string

	for _, name := range serverNames {
		result, err := backupService.CreateBackup(ctx, backup.CreateBackupOptions{
			ServerName: name,
			Compress:   true,
			KeepCount:  flags.Keep,
		})
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		results[name] = result
	}

	// Output results
	return outputBackupSuccess(stdout, jsonMode, results, errors)
}

// runBackupList lists available backups for a server.
func runBackupList(ctx context.Context, stdout io.Writer, serverName string, jsonMode bool) error {
	// List backups
	backups, err := state.ListBackups(ctx, serverName)
	if err != nil {
		return outputBackupError(stdout, jsonMode, fmt.Errorf("failed to list backups: %w", err))
	}

	// JSON output
	if jsonMode {
		output := BackupOutput{
			Status: "success",
			Data: map[string]interface{}{
				"backups": backups,
				"count":   len(backups),
			},
		}
		return json.NewEncoder(stdout).Encode(output)
	}

	// Human-readable output
	if len(backups) == 0 {
		if serverName == "" {
			_, _ = fmt.Fprintln(stdout, "No backups found")
		} else {
			_, _ = fmt.Fprintf(stdout, "No backups found for server %q\n", serverName)
		}
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSERVER\tVERSION\tSIZE\tCREATED")

	for _, b := range backups {
		size := formatBytes(b.SizeBytes)
		age := formatAge(b.CreatedAt)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			b.ID, b.Server, b.MinecraftVersion, size, age)
	}

	_ = w.Flush()
	return nil
}

// warnIfServerRunning warns if a server is currently running.
func warnIfServerRunning(ctx context.Context, serverName string, stdout io.Writer, jsonMode bool) error {
	serverState, err := state.LoadServerState(ctx, serverName)
	if err != nil {
		return err
	}

	if serverState.Status != state.StatusRunning {
		return nil
	}

	// Check container status
	containerClient, err := container.NewClient(ctx, container.DefaultConfig())
	if err != nil {
		return err
	}
	defer containerClient.Close()

	info, err := containerClient.InspectContainer(ctx, serverName)
	if err != nil || info.Status != "running" {
		return nil
	}

	// Server is running, print warning
	if !jsonMode {
		_, _ = fmt.Fprintf(stdout, "Warning: Server %q is running. Consider stopping it before backup for data consistency.\n\n", serverName)
	}

	return nil
}

// outputBackupSuccess outputs backup success result.
func outputBackupSuccess(stdout io.Writer, jsonMode bool, results map[string]*backup.CreateBackupResult, errors []string) error {
	if jsonMode {
		data := make(map[string]interface{})
		for name, result := range results {
			data[name] = map[string]interface{}{
				"backup_id": result.BackupID,
				"size":      result.BackupInfo.SizeBytes,
				"duration":  result.Duration.String(),
			}
		}

		output := BackupOutput{
			Status: "success",
			Data:   data,
		}

		if len(errors) > 0 {
			output.Data["errors"] = errors
		}

		return json.NewEncoder(stdout).Encode(output)
	}

	// Human-readable output
	if len(results) == 0 {
		_, _ = fmt.Fprintln(stdout, "No backups created")
		for _, errMsg := range errors {
			_, _ = fmt.Fprintf(stdout, "  ✗ %s\n", errMsg)
		}
		return fmt.Errorf("all backups failed")
	}

	_, _ = fmt.Fprintf(stdout, "Successfully backed up %d server(s):\n\n", len(results))

	// Sort by server name for consistent output
	names := make([]string, 0, len(results))
	for name := range results {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		result := results[name]
		size := formatBytes(result.BackupInfo.SizeBytes)
		_, _ = fmt.Fprintf(stdout, "  ✓ %s\n", name)
		_, _ = fmt.Fprintf(stdout, "    Backup ID: %s\n", result.BackupID)
		_, _ = fmt.Fprintf(stdout, "    Size:      %s\n", size)
		_, _ = fmt.Fprintf(stdout, "    Duration:  %s\n", result.Duration.Round(time.Millisecond))
		_, _ = fmt.Fprintln(stdout)
	}

	if len(errors) > 0 {
		_, _ = fmt.Fprintf(stdout, "Failed to backup %d server(s):\n", len(errors))
		for _, errMsg := range errors {
			_, _ = fmt.Fprintf(stdout, "  ✗ %s\n", errMsg)
		}
	}

	return nil
}

// outputBackupError outputs an error message.
func outputBackupError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := BackupOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}

// formatBytes formats bytes into human-readable size.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatAge formats a timestamp into relative age (e.g., "2h ago").
func formatAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1d ago"
	}
	return fmt.Sprintf("%dd ago", days)
}
