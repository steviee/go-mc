package whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/state"
)

// Output represents the JSON output format.
type Output struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewCreateCommand creates the whitelist create command.
func NewCreateCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new whitelist",
		Long: `Create a new named whitelist.

Whitelists can be shared across multiple servers by referencing
them in server configuration.`,
		Example: `  # Create a whitelist
  go-mc whitelist create mylist

  # Create with JSON output
  go-mc whitelist create mylist --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.Context(), cmd.OutOrStdout(), args[0], jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func runCreate(ctx context.Context, w io.Writer, name string, jsonOutput bool) error {
	// Validate whitelist name
	if err := state.ValidateWhitelistName(name); err != nil {
		return outputError(w, jsonOutput, fmt.Errorf("invalid whitelist name: %w", err))
	}

	// Initialize directories
	if err := state.InitDirs(); err != nil {
		return outputError(w, jsonOutput, fmt.Errorf("failed to initialize directories: %w", err))
	}

	// Check if whitelist already exists
	exists, err := state.WhitelistExists(ctx, name)
	if err != nil {
		return outputError(w, jsonOutput, err)
	}
	if exists {
		return outputError(w, jsonOutput, fmt.Errorf("whitelist %q already exists", name))
	}

	// Create new whitelist
	whitelistState := state.NewWhitelistState(name)
	if err := state.SaveWhitelistState(ctx, whitelistState); err != nil {
		return outputError(w, jsonOutput, err)
	}

	// Output results
	if jsonOutput {
		return outputCreateJSON(w, whitelistState)
	}

	return outputCreateHuman(w, whitelistState)
}

func outputCreateJSON(w io.Writer, whitelistState *state.WhitelistState) error {
	out := Output{
		Status:  "success",
		Data:    whitelistState,
		Message: fmt.Sprintf("Created whitelist %q", whitelistState.Name),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputCreateHuman(w io.Writer, whitelistState *state.WhitelistState) error {
	_, _ = fmt.Fprintf(w, "Created whitelist %q\n", whitelistState.Name)
	_, _ = fmt.Fprintf(w, "Created at: %s\n", whitelistState.CreatedAt.Format("2006-01-02 15:04:05"))
	return nil
}

func outputError(w io.Writer, jsonOutput bool, err error) error {
	if jsonOutput {
		out := Output{
			Status: "error",
			Error:  err.Error(),
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	}
	return err
}
