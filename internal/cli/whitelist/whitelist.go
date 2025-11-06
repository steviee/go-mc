package whitelist

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the whitelist command group
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whitelist",
		Short: "Manage server whitelist",
		Long: `Manage Minecraft server whitelist with automatic UUID resolution.

Commands in this group allow you to enable/disable whitelist enforcement,
add/remove users from the whitelist, and manage global whitelist configuration
that applies to all servers.`,
		Example: `  # Enable whitelist on a server
  go-mc whitelist enable myserver

  # Disable whitelist
  go-mc whitelist disable myserver

  # Add users to whitelist
  go-mc whitelist add myserver notch jeb_

  # Remove user from whitelist
  go-mc whitelist remove myserver notch

  # List whitelisted users
  go-mc whitelist list myserver

  # Set global whitelist for all servers
  go-mc whitelist global add notch jeb_`,
		Aliases: []string{"wl"},
	}

	// Add subcommands
	cmd.AddCommand(NewCreateCommand())
	cmd.AddCommand(NewDeleteCommand())
	cmd.AddCommand(NewListCommand())
	// TODO: Add enable/disable/sync commands in future phases
	// cmd.AddCommand(NewEnableCommand())
	// cmd.AddCommand(NewDisableCommand())
	// cmd.AddCommand(NewSyncCommand())

	return cmd
}
