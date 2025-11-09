package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/state"
)

// ListOutput holds the output for JSON mode
type ListOutput struct {
	Status  string          `json:"status"`
	Mods    []state.ModInfo `json:"mods,omitempty"`
	Count   int             `json:"count"`
	Message string          `json:"message,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// NewListCommand creates the mods list subcommand
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <server>",
		Short: "List installed mods on a server",
		Long: `List all mods installed on a server.

Shows mod name, slug, version, and port information (if applicable).`,
		Example: `  # List all installed mods
  go-mc mods list myserver

  # List with JSON output
  go-mc mods list myserver --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

// runList executes the list command
func runList(ctx context.Context, stdout io.Writer, serverName string) error {
	jsonMode := isJSONMode()

	// Validate server name
	if err := state.ValidateServerName(serverName); err != nil {
		return outputListError(stdout, jsonMode, fmt.Errorf("invalid server name: %w", err))
	}

	// Load server state
	serverState, err := state.LoadServerState(ctx, serverName)
	if err != nil {
		return outputListError(stdout, jsonMode, fmt.Errorf("failed to load server: %w", err))
	}

	// Output mods
	return outputListSuccess(stdout, jsonMode, serverState.Mods)
}

// outputListSuccess outputs the list of mods
func outputListSuccess(stdout io.Writer, jsonMode bool, modList []state.ModInfo) error {
	if jsonMode {
		output := ListOutput{
			Status: "success",
			Mods:   modList,
			Count:  len(modList),
		}
		return json.NewEncoder(stdout).Encode(output)
	}

	if len(modList) == 0 {
		_, _ = fmt.Fprintf(stdout, "No mods installed\n")
		return nil
	}

	// Calculate column widths
	maxName := len("NAME")
	maxSlug := len("SLUG")
	maxVersion := len("VERSION")
	for _, mod := range modList {
		if len(mod.Name) > maxName {
			maxName = len(mod.Name)
		}
		if len(mod.Slug) > maxSlug {
			maxSlug = len(mod.Slug)
		}
		if len(mod.Version) > maxVersion {
			maxVersion = len(mod.Version)
		}
	}

	// Print header
	_, _ = fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %s\n",
		maxName, "NAME",
		maxSlug, "SLUG",
		maxVersion, "VERSION",
		"PORT/PROTOCOL")

	// Print separator
	_, _ = fmt.Fprintf(stdout, "%s  %s  %s  %s\n",
		strings.Repeat("-", maxName),
		strings.Repeat("-", maxSlug),
		strings.Repeat("-", maxVersion),
		strings.Repeat("-", 13))

	// Print mods
	for _, mod := range modList {
		portInfo := "-"
		if mod.Port > 0 {
			portInfo = fmt.Sprintf("%d/%s", mod.Port, mod.Protocol)
		}

		_, _ = fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %s\n",
			maxName, mod.Name,
			maxSlug, mod.Slug,
			maxVersion, mod.Version,
			portInfo)
	}

	_, _ = fmt.Fprintf(stdout, "\nTotal: %d mod(s)\n", len(modList))

	return nil
}

// outputListError outputs an error message
func outputListError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := ListOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}
