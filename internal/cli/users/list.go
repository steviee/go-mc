package users

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/state"
)

// NewListCommand creates the users list command.
func NewListCommand() *cobra.Command {
	var (
		jsonOutput bool
		whitelist  string
		global     bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users in whitelist",
		Long:  `List all users in a whitelist.`,
		Example: `  # List users in default whitelist
  go-mc users list

  # List users in named whitelist
  go-mc users list --whitelist mylist

  # List users in global whitelist
  go-mc users list --global`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), cmd.OutOrStdout(), jsonOutput, whitelist, global)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&whitelist, "whitelist", "w", "default", "Whitelist name")
	cmd.Flags().BoolVar(&global, "global", false, "Use global whitelist (applies to all servers)")

	return cmd
}

func runList(ctx context.Context, w io.Writer, jsonOutput bool, whitelistName string, globalFlag bool) error {
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

	// Load whitelist
	players, err := state.ListPlayers(ctx, whitelistName)
	if err != nil {
		return outputError(w, jsonOutput, err)
	}

	// Output results
	if jsonOutput {
		return outputListJSON(w, whitelistName, players)
	}

	return outputListHuman(w, whitelistName, players)
}

func outputListJSON(w io.Writer, whitelistName string, players []state.PlayerInfo) error {
	out := Output{
		Status: "success",
		Data: map[string]interface{}{
			"whitelist": whitelistName,
			"players":   players,
			"count":     len(players),
		},
		Message: fmt.Sprintf("Found %d user(s) in whitelist %q", len(players), whitelistName),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputListHuman(w io.Writer, whitelistName string, players []state.PlayerInfo) error {
	if len(players) == 0 {
		_, _ = fmt.Fprintf(w, "Whitelist %q is empty\n", whitelistName)
		return nil
	}

	_, _ = fmt.Fprintf(w, "Users in whitelist %q (%d):\n", whitelistName, len(players))
	for i, player := range players {
		_, _ = fmt.Fprintf(w, "%3d. %-16s %s\n", i+1, player.Name, player.UUID)
	}

	return nil
}
