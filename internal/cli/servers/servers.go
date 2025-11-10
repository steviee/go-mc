package servers

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the servers command group
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "servers",
		Short: "Manage Minecraft Fabric servers",
		Long: `Manage the lifecycle of Minecraft Fabric servers running in Podman containers.

Commands in this group allow you to create, start, stop, restart, and remove servers.
You can also view server status, logs, and configuration.`,
		Example: `  # Create a new server
  go-mc servers create myserver --version 1.20.4 --memory 4G

  # List all servers
  go-mc servers list

  # Start a server
  go-mc servers start myserver

  # Stop a server
  go-mc servers stop myserver

  # View server status
  go-mc servers status myserver

  # Remove a server
  go-mc servers rm myserver`,
		Aliases: []string{"server", "srv"},
	}

	// Add subcommands
	cmd.AddCommand(NewCreateCommand())
	cmd.AddCommand(NewListCommand())
	cmd.AddCommand(NewListRemoteCommand())
	cmd.AddCommand(NewStartCommand())
	cmd.AddCommand(NewStopCommand())
	cmd.AddCommand(NewRestartCommand())
	cmd.AddCommand(NewRmCommand())
	cmd.AddCommand(NewLogsCommand())
	cmd.AddCommand(NewTopCommand())
	cmd.AddCommand(NewBackupCommand())
	cmd.AddCommand(NewRestoreCommand())
	cmd.AddCommand(NewUpdateCommand())

	// Future subcommands
	// cmd.AddCommand(NewStatusCommand())
	// cmd.AddCommand(NewInspectCommand())

	return cmd
}
