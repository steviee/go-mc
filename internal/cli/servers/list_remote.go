package servers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/minecraft"
)

// ListRemoteFlags holds all flags for the list-remote command
type ListRemoteFlags struct {
	Type  string
	Limit int
}

// ListRemoteOutput holds the output for JSON mode
type ListRemoteOutput struct {
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// RemoteVersionItem represents a version in the output
type RemoteVersionItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	ReleaseTime string `json:"releaseTime"`
}

// NewListRemoteCommand creates the servers list-remote subcommand
func NewListRemoteCommand() *cobra.Command {
	flags := &ListRemoteFlags{}

	cmd := &cobra.Command{
		Use:   "list-remote",
		Short: "List available Minecraft versions from Mojang",
		Long: `List available Minecraft Java Edition versions from Mojang's official version manifest.

By default, only release versions are shown. Use --type to filter by release, snapshot, or show all versions.

This command helps you discover which Minecraft versions are available before creating a server.`,
		Example: `  # List latest 20 releases
  go-mc servers list-remote

  # List all snapshots
  go-mc servers list-remote --type snapshot --limit 50

  # List all versions (releases and snapshots)
  go-mc servers list-remote --type all --limit 100

  # JSON output for scripting
  go-mc servers list-remote --json

  # List only the 5 most recent releases
  go-mc servers list-remote --limit 5`,
		Aliases: []string{"versions", "list-versions"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListRemote(cmd.Context(), cmd.OutOrStdout(), flags)
		},
	}

	// Add flags
	cmd.Flags().StringVar(&flags.Type, "type", "release", "Filter by type: release, snapshot, all")
	cmd.Flags().IntVar(&flags.Limit, "limit", 20, "Limit number of results (0 for unlimited)")

	return cmd
}

// runListRemote executes the list-remote command
func runListRemote(ctx context.Context, stdout io.Writer, flags *ListRemoteFlags) error {
	jsonMode := isJSONMode()

	// Validate type flag
	if flags.Type != "release" && flags.Type != "snapshot" && flags.Type != "all" {
		return outputListRemoteError(stdout, jsonMode, fmt.Errorf("invalid type %q: must be release, snapshot, or all", flags.Type))
	}

	// Create Minecraft API client
	client := minecraft.NewClient(nil)

	// Fetch version manifest
	manifest, err := client.GetVersionManifest(ctx)
	if err != nil {
		return outputListRemoteError(stdout, jsonMode, fmt.Errorf("failed to fetch version manifest: %w", err))
	}

	// Filter versions by type and limit
	filtered := minecraft.FilterVersions(manifest.Versions, flags.Type, flags.Limit)

	// Convert to output format
	items := make([]RemoteVersionItem, len(filtered))
	for i, v := range filtered {
		items[i] = RemoteVersionItem{
			ID:          v.ID,
			Type:        v.Type,
			ReleaseTime: v.ReleaseTime,
		}
	}

	// Output results
	if jsonMode {
		return outputListRemoteJSON(stdout, items, manifest.Latest.Release, manifest.Latest.Snapshot, len(manifest.Versions))
	}

	return outputListRemoteTable(stdout, items, manifest.Latest.Release, manifest.Latest.Snapshot)
}

// outputListRemoteTable outputs versions in table format
func outputListRemoteTable(stdout io.Writer, items []RemoteVersionItem, latestRelease, latestSnapshot string) error {
	// Print latest versions info
	_, _ = fmt.Fprintf(stdout, "Latest Release:  %s\n", latestRelease)
	_, _ = fmt.Fprintf(stdout, "Latest Snapshot: %s\n\n", latestSnapshot)

	// Calculate column widths
	versionWidth := len("VERSION")
	typeWidth := len("TYPE")
	releasedWidth := len("RELEASED")

	for _, item := range items {
		if len(item.ID) > versionWidth {
			versionWidth = len(item.ID)
		}
		if len(item.Type) > typeWidth {
			typeWidth = len(item.Type)
		}

		// Format release time for width calculation
		releaseDate := formatReleaseDate(item.ReleaseTime)
		if len(releaseDate) > releasedWidth {
			releasedWidth = len(releaseDate)
		}
	}

	// Print header
	_, _ = fmt.Fprintf(stdout, "%-*s  %-*s  %*s\n",
		versionWidth, "VERSION",
		typeWidth, "TYPE",
		releasedWidth, "RELEASED",
	)

	// Print rows
	for _, item := range items {
		releaseDate := formatReleaseDate(item.ReleaseTime)
		_, _ = fmt.Fprintf(stdout, "%-*s  %-*s  %*s\n",
			versionWidth, item.ID,
			typeWidth, item.Type,
			releasedWidth, releaseDate,
		)
	}

	return nil
}

// outputListRemoteJSON outputs versions in JSON format
func outputListRemoteJSON(stdout io.Writer, items []RemoteVersionItem, latestRelease, latestSnapshot string, totalCount int) error {
	output := ListRemoteOutput{
		Status: "success",
		Data: map[string]interface{}{
			"latest": map[string]string{
				"release":  latestRelease,
				"snapshot": latestSnapshot,
			},
			"versions": items,
			"count":    len(items),
			"total":    totalCount,
		},
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// outputListRemoteError outputs an error message
func outputListRemoteError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := ListRemoteOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}

// formatReleaseDate formats an ISO 8601 timestamp to a simple date
func formatReleaseDate(releaseTime string) string {
	// Parse ISO 8601 timestamp
	t, err := time.Parse(time.RFC3339, releaseTime)
	if err != nil {
		return releaseTime // Return as-is if parsing fails
	}

	// Format as YYYY-MM-DD
	return t.Format("2006-01-02")
}
