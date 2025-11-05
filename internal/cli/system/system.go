package system

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the system command group
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "System management and monitoring",
		Long: `Manage system-wide operations including backups, container cleanup, and TUI dashboard.

Commands in this group provide system-level operations that affect all servers
or the go-mc installation itself.`,
		Example: `  # Open TUI dashboard
  go-mc system dashboard

  # Backup a server
  go-mc system backup myserver

  # Restore from backup
  go-mc system restore myserver backup-2025-11-05.tar.gz

  # List backups
  go-mc system backups list

  # Clean up unused containers and images
  go-mc system prune

  # Check system requirements
  go-mc system check`,
		Aliases: []string{"sys"},
	}

	// Subcommands will be added in future phases
	// cmd.AddCommand(NewDashboardCommand())
	// cmd.AddCommand(NewBackupCommand())
	// cmd.AddCommand(NewRestoreCommand())
	// cmd.AddCommand(NewBackupsCommand())
	// cmd.AddCommand(NewPruneCommand())
	// cmd.AddCommand(NewCheckCommand())

	return cmd
}
