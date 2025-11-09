package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/modrinth"
	"github.com/steviee/go-mc/internal/mods"
	"github.com/steviee/go-mc/internal/state"
)

// UpdateOutput holds the output for JSON mode
type UpdateOutput struct {
	Status  string   `json:"status"`
	Updated []string `json:"updated,omitempty"`
	Message string   `json:"message,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// UpdateFlags holds flags for the update command
type UpdateFlags struct {
	All bool
}

// NewUpdateCommand creates the mods update subcommand
func NewUpdateCommand() *cobra.Command {
	flags := &UpdateFlags{}

	cmd := &cobra.Command{
		Use:   "update <server> [mod-slug]",
		Short: "Update mods on a server",
		Long: `Update one or all mods on a server to the latest compatible version.

If a mod slug is provided, only that mod will be updated. Use --all to
update all installed mods. The server must be stopped before updating mods.`,
		Example: `  # Update a single mod
  go-mc mods update myserver fabric-api

  # Update all mods
  go-mc mods update myserver --all

  # Update with JSON output
  go-mc mods update myserver lithium --json`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires server name")
			}
			if len(args) == 1 && !flags.All {
				return fmt.Errorf("requires mod slug or --all flag")
			}
			if len(args) > 1 && flags.All {
				return fmt.Errorf("cannot specify mod slug with --all flag")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			serverName := args[0]
			modSlug := ""
			if len(args) > 1 {
				modSlug = args[1]
			}
			return runUpdate(cmd.Context(), cmd.OutOrStdout(), serverName, modSlug, flags)
		},
	}

	cmd.Flags().BoolVar(&flags.All, "all", false, "Update all installed mods")

	return cmd
}

// runUpdate executes the update command
func runUpdate(ctx context.Context, stdout io.Writer, serverName string, modSlug string, flags *UpdateFlags) error {
	jsonMode := isJSONMode()

	// Validate server name
	if err := state.ValidateServerName(serverName); err != nil {
		return outputUpdateError(stdout, jsonMode, fmt.Errorf("invalid server name: %w", err))
	}

	// Load server state
	serverState, err := state.LoadServerState(ctx, serverName)
	if err != nil {
		return outputUpdateError(stdout, jsonMode, fmt.Errorf("failed to load server: %w", err))
	}

	// Get mods directory (parallel to data volume, not inside it)
	if serverState.Volumes.Data == "" {
		return outputUpdateError(stdout, jsonMode, fmt.Errorf("server data volume not configured"))
	}
	serverDir := filepath.Dir(serverState.Volumes.Data)
	modsDir := filepath.Join(serverDir, "mods")

	// Determine which mods to update
	modsToUpdate := []state.ModInfo{}
	if flags.All {
		modsToUpdate = serverState.Mods
	} else {
		// Find specific mod
		for _, mod := range serverState.Mods {
			if mod.Slug == modSlug {
				modsToUpdate = append(modsToUpdate, mod)
				break
			}
		}
		if len(modsToUpdate) == 0 {
			return outputUpdateError(stdout, jsonMode, fmt.Errorf("mod %q is not installed", modSlug))
		}
	}

	// Update each mod
	installer := mods.NewInstaller()
	modrinthClient := modrinth.NewClient(nil)
	updated := []string{}

	for _, currentMod := range modsToUpdate {
		// Find latest compatible version
		latestVersion, err := modrinthClient.FindCompatibleVersion(ctx, currentMod.ProjectID, serverState.Minecraft.Version, "")
		if err != nil {
			// Skip mods we can't update
			continue
		}

		// Check if update is needed
		if latestVersion.VersionNumber == currentMod.Version {
			// Already up-to-date
			continue
		}

		// Get primary file
		file, err := modrinth.GetPrimaryFile(latestVersion)
		if err != nil {
			continue
		}

		// Delete old mod file
		oldPath := filepath.Join(modsDir, currentMod.Filename)
		_ = os.Remove(oldPath)

		// Download new version
		newPath := filepath.Join(modsDir, file.Filename)
		if err := installer.DownloadFile(ctx, file.URL, newPath); err != nil {
			return outputUpdateError(stdout, jsonMode, fmt.Errorf("failed to download %s: %w", file.Filename, err))
		}

		// Update state
		updatedMod := currentMod
		updatedMod.Version = latestVersion.VersionNumber
		updatedMod.VersionID = latestVersion.ID
		updatedMod.URL = file.URL
		updatedMod.Filename = file.Filename
		updatedMod.SizeBytes = file.Size

		// Remove old mod from state
		if err := state.RemoveMod(ctx, serverName, currentMod.Slug); err != nil {
			return outputUpdateError(stdout, jsonMode, fmt.Errorf("failed to update state: %w", err))
		}

		// Add updated mod to state
		if err := state.AddMod(ctx, serverName, updatedMod); err != nil {
			return outputUpdateError(stdout, jsonMode, fmt.Errorf("failed to update state: %w", err))
		}

		updated = append(updated, currentMod.Slug)
	}

	// Output success
	return outputUpdateSuccess(stdout, jsonMode, updated)
}

// outputUpdateSuccess outputs a success message
func outputUpdateSuccess(stdout io.Writer, jsonMode bool, updated []string) error {
	if jsonMode {
		output := UpdateOutput{
			Status:  "success",
			Updated: updated,
			Message: fmt.Sprintf("Updated %d mod(s)", len(updated)),
		}
		return json.NewEncoder(stdout).Encode(output)
	}

	if len(updated) == 0 {
		_, _ = fmt.Fprintf(stdout, "All mods are up-to-date\n")
		return nil
	}

	_, _ = fmt.Fprintf(stdout, "Updated %d mod(s):\n", len(updated))
	for _, slug := range updated {
		_, _ = fmt.Fprintf(stdout, "  • %s\n", slug)
	}

	return nil
}

// outputUpdateError outputs an error message
func outputUpdateError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := UpdateOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}
