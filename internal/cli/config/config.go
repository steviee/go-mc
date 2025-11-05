package config

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the config command group
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long: `View and modify go-mc configuration settings.

Commands in this group allow you to view current configuration,
modify settings, and reset to defaults. Configuration is stored
in ~/.config/go-mc/config.yaml by default.`,
		Example: `  # View current configuration
  go-mc config show

  # Set a configuration value
  go-mc config set default_memory 4G

  # Get a specific value
  go-mc config get default_memory

  # Reset to defaults
  go-mc config reset

  # Edit config in $EDITOR
  go-mc config edit

  # Show configuration file path
  go-mc config path`,
		Aliases: []string{"cfg"},
	}

	// Subcommands will be added in future phases
	// cmd.AddCommand(NewShowCommand())
	// cmd.AddCommand(NewSetCommand())
	// cmd.AddCommand(NewGetCommand())
	// cmd.AddCommand(NewResetCommand())
	// cmd.AddCommand(NewEditCommand())
	// cmd.AddCommand(NewPathCommand())

	return cmd
}
