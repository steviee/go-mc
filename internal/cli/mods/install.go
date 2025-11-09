package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/mods"
	"github.com/steviee/go-mc/internal/state"
)

// InstallOutput holds the output for JSON mode
type InstallOutput struct {
	Status    string   `json:"status"`
	Installed []string `json:"installed,omitempty"`
	Message   string   `json:"message,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// NewInstallCommand creates the mods install subcommand
func NewInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <server> <mod-slug...>",
		Short: "Install mods on a server",
		Long: `Install one or more mods on an existing server from Modrinth.

Dependencies are automatically resolved and installed. If a mod is already
installed, it will be skipped. The server must be stopped before installing mods.`,
		Example: `  # Install a single mod
  go-mc mods install myserver fabric-api

  # Install multiple mods at once
  go-mc mods install myserver lithium sodium phosphor

  # Install with JSON output
  go-mc mods install myserver fabric-api --json`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(cmd.Context(), cmd.OutOrStdout(), args[0], args[1:])
		},
	}

	return cmd
}

// runInstall executes the install command
func runInstall(ctx context.Context, stdout io.Writer, serverName string, modSlugs []string) error {
	jsonMode := isJSONMode()

	// Validate server name
	if err := state.ValidateServerName(serverName); err != nil {
		return outputInstallError(stdout, jsonMode, fmt.Errorf("invalid server name: %w", err))
	}

	// Check if server exists
	exists, err := state.ServerExists(ctx, serverName)
	if err != nil {
		return outputInstallError(stdout, jsonMode, fmt.Errorf("failed to check server: %w", err))
	}
	if !exists {
		return outputInstallError(stdout, jsonMode, fmt.Errorf("server %q does not exist", serverName))
	}

	// Install mods
	installer := mods.NewInstaller()
	installed, err := installer.InstallMods(ctx, serverName, modSlugs)
	if err != nil {
		return outputInstallError(stdout, jsonMode, fmt.Errorf("failed to install mods: %w", err))
	}

	// Output success
	return outputInstallSuccess(stdout, jsonMode, installed)
}

// outputInstallSuccess outputs a success message
func outputInstallSuccess(stdout io.Writer, jsonMode bool, installed []string) error {
	if jsonMode {
		output := InstallOutput{
			Status:    "success",
			Installed: installed,
			Message:   fmt.Sprintf("Installed %d mod(s)", len(installed)),
		}
		return json.NewEncoder(stdout).Encode(output)
	}

	if len(installed) == 0 {
		_, _ = fmt.Fprintf(stdout, "No new mods installed (all mods already present)\n")
		return nil
	}

	_, _ = fmt.Fprintf(stdout, "Installed %d mod(s):\n", len(installed))
	for _, slug := range installed {
		_, _ = fmt.Fprintf(stdout, "  • %s\n", slug)
	}

	return nil
}

// outputInstallError outputs an error message
func outputInstallError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := InstallOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}
