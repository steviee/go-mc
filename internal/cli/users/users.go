package users

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the users command group
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage Minecraft users and permissions",
		Long: `Manage Minecraft users with automatic UUID resolution via Mojang API.

Commands in this group allow you to add users to servers, remove them,
and manage their operator (OP) permissions. UUIDs are automatically
resolved from usernames.`,
		Example: `  # Add a user to a server
  go-mc users add myserver notch

  # Add multiple users
  go-mc users add myserver notch jeb_ dinnerbone

  # Remove a user
  go-mc users remove myserver notch

  # Grant operator permissions
  go-mc users op myserver notch

  # Revoke operator permissions
  go-mc users deop myserver notch

  # List users on a server
  go-mc users list myserver`,
		Aliases: []string{"user"},
	}

	// Add subcommands
	cmd.AddCommand(NewAddCommand())
	cmd.AddCommand(NewRemoveCommand())
	cmd.AddCommand(NewListCommand())
	// TODO: Add OP commands in future phases
	// cmd.AddCommand(NewOpCommand())
	// cmd.AddCommand(NewDeopCommand())

	return cmd
}
