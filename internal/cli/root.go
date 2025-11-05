package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/steviee/go-mc/internal/cli/config"
	"github.com/steviee/go-mc/internal/cli/mods"
	"github.com/steviee/go-mc/internal/cli/servers"
	"github.com/steviee/go-mc/internal/cli/system"
	"github.com/steviee/go-mc/internal/cli/users"
	"github.com/steviee/go-mc/internal/cli/whitelist"
)

var (
	// Global flags
	cfgFile string
	jsonOut bool
	quiet   bool
	verbose bool

	// Global logger
	logger *slog.Logger
)

// NewRootCommand creates and returns the root cobra command
func NewRootCommand(version, commit, date, builtBy string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "go-mc",
		Short: "Manage Minecraft Fabric servers with Podman",
		Long: `go-mc is a CLI tool for managing Minecraft Fabric servers using Podman containers.

It provides a simple, opinionated interface for:
  - Creating and managing Minecraft Fabric servers
  - Installing mods from Modrinth with dependency resolution
  - Managing whitelists and user permissions
  - Monitoring server status and logs
  - Backing up and restoring server data

All servers run in rootless Podman containers for security and isolation.`,
		Example: `  # Create a new server
  go-mc servers create myserver --version 1.20.4 --memory 4G

  # Start a server
  go-mc servers start myserver

  # Install a mod
  go-mc mods install myserver fabric-api

  # View all servers
  go-mc servers list

  # Open TUI dashboard
  go-mc system dashboard`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize logger based on flags
			if err := initLogger(cmd.OutOrStdout()); err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}

			// Initialize config
			if err := initConfig(); err != nil {
				logger.Error("failed to initialize config", "error", err)
				return fmt.Errorf("failed to initialize config: %w", err)
			}

			return nil
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/go-mc/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose logging")

	// Mark json and quiet as mutually exclusive
	rootCmd.MarkFlagsMutuallyExclusive("json", "quiet")
	rootCmd.MarkFlagsMutuallyExclusive("verbose", "quiet")

	// Add version command
	rootCmd.AddCommand(NewVersionCommand(version, commit, date, builtBy))

	// Add command groups (placeholders for now)
	rootCmd.AddCommand(NewServersCommand())
	rootCmd.AddCommand(NewUsersCommand())
	rootCmd.AddCommand(NewWhitelistCommand())
	rootCmd.AddCommand(NewModsCommand())
	rootCmd.AddCommand(NewSystemCommand())
	rootCmd.AddCommand(NewConfigCommand())

	return rootCmd
}

// NewServersCommand creates the servers command group
func NewServersCommand() *cobra.Command {
	return servers.NewCommand()
}

// NewUsersCommand creates the users command group
func NewUsersCommand() *cobra.Command {
	return users.NewCommand()
}

// NewWhitelistCommand creates the whitelist command group
func NewWhitelistCommand() *cobra.Command {
	return whitelist.NewCommand()
}

// NewModsCommand creates the mods command group
func NewModsCommand() *cobra.Command {
	return mods.NewCommand()
}

// NewSystemCommand creates the system command group
func NewSystemCommand() *cobra.Command {
	return system.NewCommand()
}

// NewConfigCommand creates the config command group
func NewConfigCommand() *cobra.Command {
	return config.NewCommand()
}

// initLogger initializes the global logger based on flags
func initLogger(out io.Writer) error {
	var level slog.Level
	var handler slog.Handler

	// Determine log level
	switch {
	case quiet:
		level = slog.LevelError
	case verbose:
		level = slog.LevelDebug
	default:
		level = slog.LevelInfo
	}

	// Create handler based on output format
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if jsonOut {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	logger = slog.New(handler)
	slog.SetDefault(logger)

	return nil
}

// initConfig reads in config file and ENV variables if set
func initConfig() error {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get user home directory: %w", err)
		}

		// Search config in ~/.config/go-mc directory
		configDir := filepath.Join(home, ".config", "go-mc")
		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("GOMC")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("read config file: %w", err)
		}
	} else {
		logger.Debug("using config file", "path", viper.ConfigFileUsed())
	}

	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *slog.Logger {
	return logger
}

// IsJSONOutput returns true if JSON output is enabled
func IsJSONOutput() bool {
	return jsonOut
}

// IsQuiet returns true if quiet mode is enabled
func IsQuiet() bool {
	return quiet
}

// IsVerbose returns true if verbose mode is enabled
func IsVerbose() bool {
	return verbose
}
