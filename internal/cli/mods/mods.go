package mods

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the mods command group
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mods",
		Short: "Manage Modrinth mods",
		Long: `Search, install, update, and remove mods from Modrinth with automatic dependency resolution.

Commands in this group integrate with the Modrinth API to provide seamless
mod management. Dependencies are automatically resolved and installed based
on the server's Minecraft and Fabric versions.`,
		Example: `  # Search for mods
  go-mc mods search fabric-api

  # Install a mod
  go-mc mods install myserver fabric-api

  # Install with specific version
  go-mc mods install myserver sodium --version 0.5.5

  # List installed mods
  go-mc mods list myserver

  # Update a mod
  go-mc mods update myserver fabric-api

  # Update all mods
  go-mc mods update myserver --all

  # Remove a mod
  go-mc mods remove myserver sodium`,
		Aliases: []string{"mod"},
	}

	// Subcommands will be added in future phases
	// cmd.AddCommand(NewSearchCommand())
	// cmd.AddCommand(NewInstallCommand())
	// cmd.AddCommand(NewListCommand())
	// cmd.AddCommand(NewUpdateCommand())
	// cmd.AddCommand(NewRemoveCommand())

	return cmd
}
