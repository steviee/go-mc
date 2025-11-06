package whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/state"
)

// NewListCommand creates the whitelist list command.
func NewListCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all whitelists",
		Long:  `List all available whitelists.`,
		Example: `  # List whitelists
  go-mc whitelist list

  # List with JSON output
  go-mc whitelist list --json`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListWhitelists(cmd.Context(), cmd.OutOrStdout(), jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func runListWhitelists(ctx context.Context, w io.Writer, jsonOutput bool) error {
	// List all whitelists
	whitelists, err := state.ListWhitelistStates(ctx)
	if err != nil {
		return outputError(w, jsonOutput, err)
	}

	// Load details for each whitelist
	infos := make([]WhitelistInfo, 0, len(whitelists))
	for _, name := range whitelists {
		players, err := state.ListPlayers(ctx, name)
		if err != nil {
			continue
		}

		infos = append(infos, WhitelistInfo{
			Name:        name,
			PlayerCount: len(players),
		})
	}

	// Output results
	if jsonOutput {
		return outputListWhitelistsJSON(w, infos)
	}

	return outputListWhitelistsHuman(w, infos)
}

// WhitelistInfo represents whitelist information for listing.
type WhitelistInfo struct {
	Name        string `json:"name"`
	PlayerCount int    `json:"player_count"`
}

func outputListWhitelistsJSON(w io.Writer, infos []WhitelistInfo) error {
	out := Output{
		Status: "success",
		Data: map[string]interface{}{
			"whitelists": infos,
			"count":      len(infos),
		},
		Message: fmt.Sprintf("Found %d whitelist(s)", len(infos)),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputListWhitelistsHuman(w io.Writer, infos []WhitelistInfo) error {
	if len(infos) == 0 {
		_, _ = fmt.Fprintln(w, "No whitelists found")
		return nil
	}

	_, _ = fmt.Fprintf(w, "Whitelists (%d):\n", len(infos))
	for i, info := range infos {
		_, _ = fmt.Fprintf(w, "%3d. %-20s (%d player%s)\n",
			i+1,
			info.Name,
			info.PlayerCount,
			pluralS(info.PlayerCount),
		)
	}

	return nil
}

func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
