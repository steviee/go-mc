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
	Type    string
	Limit   int
	Loaders bool
	Version string
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

// FabricLoaderItem represents a Fabric loader in the output
type FabricLoaderItem struct {
	Version string `json:"version"`
	Build   int    `json:"build"`
	Stable  bool   `json:"stable"`
}

// NewListRemoteCommand creates the servers list-remote subcommand
func NewListRemoteCommand() *cobra.Command {
	flags := &ListRemoteFlags{}

	cmd := &cobra.Command{
		Use:   "list-remote",
		Short: "List available Minecraft versions or Fabric loaders",
		Long: `List available Minecraft Java Edition versions from Mojang's official version manifest,
or Fabric loader versions from Fabric Meta API.

By default, only Minecraft release versions are shown. Use --type to filter by release, snapshot, or show all versions.
Use --loaders to show Fabric loader versions instead of Minecraft versions.

This command helps you discover which Minecraft versions and Fabric loaders are available before creating a server.`,
		Example: `  # List latest 20 Minecraft releases
  go-mc servers list-remote

  # List all snapshots
  go-mc servers list-remote --type snapshot --limit 50

  # List all versions (releases and snapshots)
  go-mc servers list-remote --type all --limit 100

  # List all Fabric loader versions
  go-mc servers list-remote --loaders

  # List Fabric loaders for specific Minecraft version
  go-mc servers list-remote --loaders --version 1.21.1

  # List latest 10 Fabric loaders
  go-mc servers list-remote --loaders --limit 10

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
	cmd.Flags().BoolVar(&flags.Loaders, "loaders", false, "Show Fabric loader versions instead of Minecraft versions")
	cmd.Flags().StringVar(&flags.Version, "version", "", "Filter by Minecraft version (requires --loaders)")

	return cmd
}

// runListRemote executes the list-remote command
func runListRemote(ctx context.Context, stdout io.Writer, flags *ListRemoteFlags) error {
	jsonMode := isJSONMode()

	// Validate flags
	if flags.Version != "" && !flags.Loaders {
		return outputListRemoteError(stdout, jsonMode, fmt.Errorf("--version flag requires --loaders flag"))
	}

	// If loaders mode, delegate to runListRemoteLoaders
	if flags.Loaders {
		return runListRemoteLoaders(ctx, stdout, flags)
	}

	// Validate type flag for Minecraft versions
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

// runListRemoteLoaders executes the list-remote command in loaders mode
func runListRemoteLoaders(ctx context.Context, stdout io.Writer, flags *ListRemoteFlags) error {
	jsonMode := isJSONMode()

	// Create Minecraft API client
	client := minecraft.NewClient(nil)

	var loaders []minecraft.FabricLoader
	var err error

	// Fetch loaders based on whether version is specified
	if flags.Version != "" {
		loaders, err = client.GetFabricLoadersForVersion(ctx, flags.Version)
		if err != nil {
			return outputListRemoteError(stdout, jsonMode, fmt.Errorf("failed to fetch Fabric loaders for version %s: %w", flags.Version, err))
		}
	} else {
		loaders, err = client.GetFabricLoaders(ctx)
		if err != nil {
			return outputListRemoteError(stdout, jsonMode, fmt.Errorf("failed to fetch Fabric loaders: %w", err))
		}
	}

	// Apply limit if set
	if flags.Limit > 0 && len(loaders) > flags.Limit {
		loaders = loaders[:flags.Limit]
	}

	// Convert to output format
	items := make([]FabricLoaderItem, len(loaders))
	for i, loader := range loaders {
		items[i] = FabricLoaderItem{
			Version: loader.Version,
			Build:   loader.Build,
			Stable:  loader.Stable,
		}
	}

	// Output results
	if jsonMode {
		return outputFabricLoadersJSON(stdout, items, flags.Version)
	}

	return outputFabricLoadersTable(stdout, items, flags.Version)
}

// outputFabricLoadersTable outputs Fabric loaders in table format
func outputFabricLoadersTable(stdout io.Writer, items []FabricLoaderItem, minecraftVersion string) error {
	// Print Minecraft version if specified
	if minecraftVersion != "" {
		_, _ = fmt.Fprintf(stdout, "MINECRAFT VERSION: %s\n\n", minecraftVersion)
	}

	// Calculate column widths
	versionWidth := len("LOADER VERSION")
	buildWidth := len("BUILD")
	stableWidth := len("STABLE")

	for _, item := range items {
		if len(item.Version) > versionWidth {
			versionWidth = len(item.Version)
		}
		buildStr := fmt.Sprintf("%d", item.Build)
		if len(buildStr) > buildWidth {
			buildWidth = len(buildStr)
		}
	}

	// Print header
	_, _ = fmt.Fprintf(stdout, "%-*s  %*s  %*s\n",
		versionWidth, "LOADER VERSION",
		buildWidth, "BUILD",
		stableWidth, "STABLE",
	)

	// Print rows
	for _, item := range items {
		stable := "no"
		if item.Stable {
			stable = "yes"
		}
		_, _ = fmt.Fprintf(stdout, "%-*s  %*d  %*s\n",
			versionWidth, item.Version,
			buildWidth, item.Build,
			stableWidth, stable,
		)
	}

	return nil
}

// outputFabricLoadersJSON outputs Fabric loaders in JSON format
func outputFabricLoadersJSON(stdout io.Writer, items []FabricLoaderItem, minecraftVersion string) error {
	// Find latest stable loader
	latestStable := ""
	for _, item := range items {
		if item.Stable {
			latestStable = item.Version
			break
		}
	}

	// Build data map
	data := map[string]interface{}{
		"loaders":       items,
		"latest_stable": latestStable,
		"count":         len(items),
	}

	// Add minecraft_version if specified
	if minecraftVersion != "" {
		data["minecraft_version"] = minecraftVersion
	} else {
		data["minecraft_version"] = nil
	}

	output := ListRemoteOutput{
		Status: "success",
		Data:   data,
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
