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

// Output represents the JSON output format.
type Output struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewAddCommand creates the users add command.
func NewAddCommand() *cobra.Command {
	var (
		jsonOutput bool
		whitelist  string
		global     bool
	)

	cmd := &cobra.Command{
		Use:   "add <username> [username...]",
		Short: "Add users to whitelist",
		Long: `Add one or more users to a whitelist with automatic UUID lookup.

UUIDs are automatically resolved from usernames via Mojang API.
Results are cached to avoid repeated API calls.`,
		Example: `  # Add user to default whitelist
  go-mc users add notch

  # Add multiple users
  go-mc users add notch jeb_ dinnerbone

  # Add to named whitelist
  go-mc users add --whitelist mylist notch

  # Add to global whitelist (applies to all servers)
  go-mc users add --global notch`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd.Context(), cmd.OutOrStdout(), args, jsonOutput, whitelist, global)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&whitelist, "whitelist", "w", "default", "Whitelist name")
	cmd.Flags().BoolVar(&global, "global", false, "Use global whitelist (applies to all servers)")

	return cmd
}

func runAdd(ctx context.Context, w io.Writer, usernames []string, jsonOutput bool, whitelistName string, globalFlag bool) error {
	// Use "default" for global flag
	if globalFlag {
		whitelistName = "default"
	}

	// Validate whitelist name
	if err := state.ValidateWhitelistName(whitelistName); err != nil {
		return outputError(w, jsonOutput, fmt.Errorf("invalid whitelist name: %w", err))
	}

	// Initialize directories
	if err := state.InitDirs(); err != nil {
		return outputError(w, jsonOutput, fmt.Errorf("failed to initialize directories: %w", err))
	}

	// Create Mojang API client
	mojangClient := mojang.NewClient(nil)

	// Process each username
	addedUsers := make([]state.PlayerInfo, 0, len(usernames))
	errors := make(map[string]string)

	for _, username := range usernames {
		// Lookup UUID
		profile, err := mojangClient.GetUUID(ctx, username)
		if err != nil {
			errors[username] = err.Error()
			continue
		}

		// Add player to whitelist
		player := state.PlayerInfo{
			UUID: profile.UUID,
			Name: profile.Username,
		}

		if err := state.AddPlayer(ctx, whitelistName, player); err != nil {
			errors[username] = err.Error()
			continue
		}

		addedUsers = append(addedUsers, player)
	}

	// Output results
	if jsonOutput {
		return outputAddJSON(w, whitelistName, addedUsers, errors)
	}

	return outputAddHuman(w, whitelistName, addedUsers, errors)
}

func outputAddJSON(w io.Writer, whitelistName string, added []state.PlayerInfo, errors map[string]string) error {
	data := map[string]interface{}{
		"whitelist": whitelistName,
		"added":     added,
	}

	if len(errors) > 0 {
		data["errors"] = errors
	}

	status := "success"
	if len(added) == 0 {
		status = "error"
	}

	out := Output{
		Status:  status,
		Data:    data,
		Message: fmt.Sprintf("Added %d user(s) to whitelist %q", len(added), whitelistName),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputAddHuman(w io.Writer, whitelistName string, added []state.PlayerInfo, errors map[string]string) error {
	if len(added) > 0 {
		_, _ = fmt.Fprintf(w, "Added %d user(s) to whitelist %q:\n", len(added), whitelistName)
		for _, player := range added {
			_, _ = fmt.Fprintf(w, "  - %s (%s)\n", player.Name, player.UUID)
		}
	}

	if len(errors) > 0 {
		_, _ = fmt.Fprintf(w, "\nFailed to add %d user(s):\n", len(errors))
		for username, errMsg := range errors {
			_, _ = fmt.Fprintf(w, "  - %s: %s\n", username, errMsg)
		}
	}

	if len(added) == 0 && len(errors) > 0 {
		return fmt.Errorf("failed to add any users")
	}

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
