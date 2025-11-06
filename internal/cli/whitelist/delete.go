package whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/state"
)

// NewDeleteCommand creates the whitelist delete command.
func NewDeleteCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a whitelist",
		Long:  `Delete a named whitelist and all its entries.`,
		Example: `  # Delete a whitelist
  go-mc whitelist delete mylist

  # Delete with JSON output
  go-mc whitelist delete mylist --json`,
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"rm", "remove"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), cmd.OutOrStdout(), args[0], jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func runDelete(ctx context.Context, w io.Writer, name string, jsonOutput bool) error {
	// Validate whitelist name
	if err := state.ValidateWhitelistName(name); err != nil {
		return outputError(w, jsonOutput, fmt.Errorf("invalid whitelist name: %w", err))
	}

	// Check if whitelist exists
	exists, err := state.WhitelistExists(ctx, name)
	if err != nil {
		return outputError(w, jsonOutput, err)
	}
	if !exists {
		return outputError(w, jsonOutput, fmt.Errorf("whitelist %q does not exist", name))
	}

	// Delete whitelist
	if err := state.DeleteWhitelistState(ctx, name); err != nil {
		return outputError(w, jsonOutput, err)
	}

	// Output results
	if jsonOutput {
		return outputDeleteJSON(w, name)
	}

	return outputDeleteHuman(w, name)
}

func outputDeleteJSON(w io.Writer, name string) error {
	out := Output{
		Status:  "success",
		Message: fmt.Sprintf("Deleted whitelist %q", name),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputDeleteHuman(w io.Writer, name string) error {
	_, _ = fmt.Fprintf(w, "Deleted whitelist %q\n", name)
	return nil
}
