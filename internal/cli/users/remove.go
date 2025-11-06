package users

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/mojang"
	"github.com/steviee/go-mc/internal/state"
)

// NewRemoveCommand creates the users remove command.
func NewRemoveCommand() *cobra.Command {
	var (
		jsonOutput bool
		whitelist  string
		global     bool
	)

	cmd := &cobra.Command{
		Use:   "remove <username> [username...]",
		Short: "Remove users from whitelist",
		Long: `Remove one or more users from a whitelist.

UUIDs are automatically resolved from usernames via Mojang API.`,
		Example: `  # Remove user from default whitelist
  go-mc users remove notch

  # Remove multiple users
  go-mc users remove notch jeb_

  # Remove from named whitelist
  go-mc users remove --whitelist mylist notch

  # Remove from global whitelist
  go-mc users remove --global notch`,
		Args:    cobra.MinimumNArgs(1),
		Aliases: []string{"rm", "del"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), cmd.OutOrStdout(), args, jsonOutput, whitelist, global)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&whitelist, "whitelist", "w", "default", "Whitelist name")
	cmd.Flags().BoolVar(&global, "global", false, "Use global whitelist (applies to all servers)")

	return cmd
}

func runRemove(ctx context.Context, w io.Writer, usernames []string, jsonOutput bool, whitelistName string, globalFlag bool) error {
	// Use "default" for global flag
	if globalFlag {
		whitelistName = "default"
	}

	// Validate whitelist name
	if err := state.ValidateWhitelistName(whitelistName); err != nil {
		return outputError(w, jsonOutput, fmt.Errorf("invalid whitelist name: %w", err))
	}

	// Check if whitelist exists
	exists, err := state.WhitelistExists(ctx, whitelistName)
	if err != nil {
		return outputError(w, jsonOutput, err)
	}
	if !exists {
		return outputError(w, jsonOutput, fmt.Errorf("whitelist %q does not exist", whitelistName))
	}

	// Create Mojang API client
	mojangClient := mojang.NewClient(nil)

	// Process each username
	removedUsers := make([]state.PlayerInfo, 0, len(usernames))
	errors := make(map[string]string)

	for _, username := range usernames {
		// Lookup UUID
		profile, err := mojangClient.GetUUID(ctx, username)
		if err != nil {
			errors[username] = err.Error()
			continue
		}

		// Get player info before removing
		player, err := state.GetPlayer(ctx, whitelistName, profile.UUID)
		if err != nil {
			errors[username] = err.Error()
			continue
		}

		// Remove player from whitelist
		if err := state.RemovePlayer(ctx, whitelistName, profile.UUID); err != nil {
			errors[username] = err.Error()
			continue
		}

		removedUsers = append(removedUsers, *player)
	}

	// Output results
	if jsonOutput {
		return outputRemoveJSON(w, whitelistName, removedUsers, errors)
	}

	return outputRemoveHuman(w, whitelistName, removedUsers, errors)
}

func outputRemoveJSON(w io.Writer, whitelistName string, removed []state.PlayerInfo, errors map[string]string) error {
	data := map[string]interface{}{
		"whitelist": whitelistName,
		"removed":   removed,
	}

	if len(errors) > 0 {
		data["errors"] = errors
	}

	status := "success"
	if len(removed) == 0 {
		status = "error"
	}

	out := Output{
		Status:  status,
		Data:    data,
		Message: fmt.Sprintf("Removed %d user(s) from whitelist %q", len(removed), whitelistName),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputRemoveHuman(w io.Writer, whitelistName string, removed []state.PlayerInfo, errors map[string]string) error {
	if len(removed) > 0 {
		_, _ = fmt.Fprintf(w, "Removed %d user(s) from whitelist %q:\n", len(removed), whitelistName)
		for _, player := range removed {
			_, _ = fmt.Fprintf(w, "  - %s (%s)\n", player.Name, player.UUID)
		}
	}

	if len(errors) > 0 {
		_, _ = fmt.Fprintf(w, "\nFailed to remove %d user(s):\n", len(errors))
		for username, errMsg := range errors {
			_, _ = fmt.Fprintf(w, "  - %s: %s\n", username, errMsg)
		}
	}

	if len(removed) == 0 && len(errors) > 0 {
		return fmt.Errorf("failed to remove any users")
	}

	return nil
}
