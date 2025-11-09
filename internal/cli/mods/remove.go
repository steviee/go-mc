package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/state"
)

// RemoveOutput holds the output for JSON mode
type RemoveOutput struct {
	Status  string   `json:"status"`
	Removed []string `json:"removed,omitempty"`
	Message string   `json:"message,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// NewRemoveCommand creates the mods remove subcommand
func NewRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <server> <mod-slug...>",
		Short: "Remove mods from a server",
		Long: `Remove one or more mods from a server.

The mod files will be deleted from the server's mods directory and the
mod will be removed from the server state. The server must be stopped
before removing mods.

Note: This command does NOT check for reverse dependencies. If you remove
a mod that other mods depend on, those mods may fail to load.`,
		Example: `  # Remove a single mod
  go-mc mods remove myserver sodium

  # Remove multiple mods at once
  go-mc mods remove myserver sodium phosphor

  # Remove with JSON output
  go-mc mods remove myserver lithium --json`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), cmd.OutOrStdout(), args[0], args[1:])
		},
	}

	return cmd
}

// runRemove executes the remove command
func runRemove(ctx context.Context, stdout io.Writer, serverName string, modSlugs []string) error {
	jsonMode := isJSONMode()

	// Validate server name
	if err := state.ValidateServerName(serverName); err != nil {
		return outputRemoveError(stdout, jsonMode, fmt.Errorf("invalid server name: %w", err))
	}

	// Load server state
	serverState, err := state.LoadServerState(ctx, serverName)
	if err != nil {
		return outputRemoveError(stdout, jsonMode, fmt.Errorf("failed to load server: %w", err))
	}

	// Get mods directory (parallel to data volume, not inside it)
	if serverState.Volumes.Data == "" {
		return outputRemoveError(stdout, jsonMode, fmt.Errorf("server data volume not configured"))
	}
	serverDir := filepath.Dir(serverState.Volumes.Data)
	modsDir := filepath.Join(serverDir, "mods")

	removed := []string{}
	for _, slug := range modSlugs {
		// Find mod in state
		var modInfo *state.ModInfo
		for i := range serverState.Mods {
			if serverState.Mods[i].Slug == slug {
				modInfo = &serverState.Mods[i]
				break
			}
		}

		if modInfo == nil {
			// Skip if not installed
			continue
		}

		// Delete mod file
		modPath := filepath.Join(modsDir, modInfo.Filename)
		if err := os.Remove(modPath); err != nil && !os.IsNotExist(err) {
			return outputRemoveError(stdout, jsonMode, fmt.Errorf("failed to delete %s: %w", modInfo.Filename, err))
		}

		// Remove from state
		if err := state.RemoveMod(ctx, serverName, slug); err != nil {
			return outputRemoveError(stdout, jsonMode, fmt.Errorf("failed to remove mod from state: %w", err))
		}

		// Release port if allocated
		if modInfo.Port > 0 {
			_ = state.ReleasePort(ctx, modInfo.Port)
		}

		removed = append(removed, slug)
	}

	// Output success
	return outputRemoveSuccess(stdout, jsonMode, removed)
}

// outputRemoveSuccess outputs a success message
func outputRemoveSuccess(stdout io.Writer, jsonMode bool, removed []string) error {
	if jsonMode {
		output := RemoveOutput{
			Status:  "success",
			Removed: removed,
			Message: fmt.Sprintf("Removed %d mod(s)", len(removed)),
		}
		return json.NewEncoder(stdout).Encode(output)
	}

	if len(removed) == 0 {
		_, _ = fmt.Fprintf(stdout, "No mods removed (not installed)\n")
		return nil
	}

	_, _ = fmt.Fprintf(stdout, "Removed %d mod(s):\n", len(removed))
	for _, slug := range removed {
		_, _ = fmt.Fprintf(stdout, "  • %s\n", slug)
	}

	return nil
}

// outputRemoveError outputs an error message
func outputRemoveError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := RemoveOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}
