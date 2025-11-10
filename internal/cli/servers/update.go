package servers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/backup"
	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/minecraft"
	"github.com/steviee/go-mc/internal/modrinth"
	"github.com/steviee/go-mc/internal/mods"
	"github.com/steviee/go-mc/internal/state"
)

// UpdateFlags holds flags for the update command.
type UpdateFlags struct {
	Version  string
	Latest   bool
	ModsOnly bool
	Backup   bool
	Restart  bool
	DryRun   bool
}

// UpdateOutput holds the output for JSON mode.
type UpdateOutput struct {
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// ModUpdateResult represents the result of updating a single mod.
type ModUpdateResult struct {
	Slug       string `json:"slug"`
	Status     string `json:"status"` // "success", "failed", "skipped"
	OldVersion string `json:"old_version,omitempty"`
	NewVersion string `json:"new_version,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// UpdateSummary holds the complete update summary.
type UpdateSummary struct {
	ServerName   string            `json:"server"`
	BackupID     string            `json:"backup_id,omitempty"`
	MinecraftOld string            `json:"minecraft_old"`
	MinecraftNew string            `json:"minecraft_new"`
	FabricOld    string            `json:"fabric_old"`
	FabricNew    string            `json:"fabric_new"`
	ModsUpdated  []ModUpdateResult `json:"mods_updated,omitempty"`
	ModsSkipped  []ModUpdateResult `json:"mods_skipped,omitempty"`
	ModsFailed   []ModUpdateResult `json:"mods_failed,omitempty"`
	Restarted    bool              `json:"restarted"`
}

// NewUpdateCommand creates the servers update subcommand.
func NewUpdateCommand() *cobra.Command {
	flags := &UpdateFlags{}

	cmd := &cobra.Command{
		Use:   "update <server-name>",
		Short: "Update Minecraft/Fabric version or mods",
		Long: `Update a server's Minecraft version, Fabric loader, and mods.

The update command handles the complete update workflow:
1. Creates a backup (unless --backup=false)
2. Stops the server if running
3. Updates Minecraft/Fabric versions (unless --mods-only)
4. Updates all installed mods to compatible versions
5. Recreates the container with new configuration
6. Optionally restarts the server

Use --dry-run to preview changes without applying them.`,
		Example: `  # Update to specific Minecraft version
  go-mc servers update myserver --version 1.21.5

  # Update to latest Minecraft + Fabric + mods
  go-mc servers update myserver --latest --restart

  # Update only mods (preserve MC version)
  go-mc servers update myserver --mods-only

  # Preview changes without applying
  go-mc servers update myserver --version 1.21.5 --dry-run

  # Update without backup (not recommended)
  go-mc servers update myserver --version 1.21.5 --backup=false

  # JSON output
  go-mc servers update myserver --latest --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverName := args[0]
			return runUpdate(cmd.Context(), cmd.OutOrStdout(), serverName, flags)
		},
	}

	cmd.Flags().StringVar(&flags.Version, "version", "", "Update to specific Minecraft version")
	cmd.Flags().BoolVar(&flags.Latest, "latest", false, "Update to latest Minecraft + Fabric")
	cmd.Flags().BoolVar(&flags.ModsOnly, "mods-only", false, "Update mods only (preserve MC version)")
	cmd.Flags().BoolVar(&flags.Backup, "backup", true, "Create backup before update")
	cmd.Flags().BoolVar(&flags.Restart, "restart", false, "Restart server after update")
	cmd.Flags().BoolVar(&flags.DryRun, "dry-run", false, "Show what would be updated without applying")

	// Mutual exclusivity validation
	cmd.MarkFlagsMutuallyExclusive("version", "latest", "mods-only")

	return cmd
}

// runUpdate executes the update command.
func runUpdate(ctx context.Context, stdout io.Writer, serverName string, flags *UpdateFlags) error {
	jsonMode := isJSONMode()

	// Validate flags
	if err := validateUpdateFlags(flags); err != nil {
		return outputUpdateError(stdout, jsonMode, err)
	}

	// Validate server name
	if err := state.ValidateServerName(serverName); err != nil {
		return outputUpdateError(stdout, jsonMode, fmt.Errorf("invalid server name: %w", err))
	}

	// Load server state
	serverState, err := state.LoadServerState(ctx, serverName)
	if err != nil {
		return outputUpdateError(stdout, jsonMode, fmt.Errorf("failed to load server state: %w", err))
	}

	// Create API clients
	minecraftClient := minecraft.NewClient(nil)
	modrinthClient := modrinth.NewClient(nil)

	// Determine target versions
	targetMCVersion, targetFabricVersion, err := resolveTargetVersions(
		ctx,
		minecraftClient,
		serverState,
		flags,
	)
	if err != nil {
		return outputUpdateError(stdout, jsonMode, err)
	}

	// Check if update is needed
	if !flags.ModsOnly {
		if targetMCVersion == serverState.Minecraft.Version && targetFabricVersion == serverState.Minecraft.FabricLoaderVersion {
			msg := fmt.Sprintf("Server '%s' is already at version %s (Fabric %s)",
				serverName, targetMCVersion, targetFabricVersion)
			if jsonMode {
				output := UpdateOutput{
					Status:  "success",
					Message: msg,
				}
				return json.NewEncoder(stdout).Encode(output)
			}
			_, _ = fmt.Fprintln(stdout, msg)
			return nil
		}
	}

	// Dry run: show what would be updated
	if flags.DryRun {
		return showDryRunUpdate(ctx, stdout, serverState, targetMCVersion, targetFabricVersion, modrinthClient, jsonMode, flags)
	}

	// Real update: execute the full workflow
	summary, err := executeUpdate(ctx, stdout, serverState, targetMCVersion, targetFabricVersion, minecraftClient, modrinthClient, flags, jsonMode)
	if err != nil {
		return outputUpdateError(stdout, jsonMode, err)
	}

	// Output final summary
	return outputUpdateSummary(stdout, summary, jsonMode)
}

// validateUpdateFlags validates the update flags.
func validateUpdateFlags(flags *UpdateFlags) error {
	// At least one update mode must be specified
	if !flags.Latest && flags.Version == "" && !flags.ModsOnly {
		return fmt.Errorf("must specify --version, --latest, or --mods-only")
	}

	return nil
}

// resolveTargetVersions determines the target Minecraft and Fabric versions.
func resolveTargetVersions(
	ctx context.Context,
	client *minecraft.Client,
	serverState *state.ServerState,
	flags *UpdateFlags,
) (mcVersion, fabricVersion string, err error) {
	// If --mods-only, preserve current versions
	if flags.ModsOnly {
		return serverState.Minecraft.Version, serverState.Minecraft.FabricLoaderVersion, nil
	}

	// Resolve Minecraft version
	if flags.Latest {
		manifest, err := client.GetVersionManifest(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to fetch version manifest: %w", err)
		}
		mcVersion = manifest.Latest.Release
	} else if flags.Version != "" {
		// Validate version exists
		manifest, err := client.GetVersionManifest(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to fetch version manifest: %w", err)
		}
		found := false
		for _, v := range manifest.Versions {
			if v.ID == flags.Version {
				found = true
				break
			}
		}
		if !found {
			return "", "", fmt.Errorf("minecraft version %q not found", flags.Version)
		}
		mcVersion = flags.Version
	} else {
		// Should not reach here due to validation
		return "", "", fmt.Errorf("no version specified")
	}

	// Find compatible Fabric loader
	loaders, err := client.GetFabricLoadersForVersion(ctx, mcVersion)
	if err != nil {
		return "", "", fmt.Errorf("failed to find Fabric loaders for %s: %w", mcVersion, err)
	}

	if len(loaders) == 0 {
		return "", "", fmt.Errorf("no Fabric loaders available for Minecraft %s", mcVersion)
	}

	// Use latest stable, or latest if no stable exists
	for _, loader := range loaders {
		if loader.Stable {
			fabricVersion = loader.Version
			break
		}
	}
	if fabricVersion == "" {
		fabricVersion = loaders[0].Version
	}

	return mcVersion, fabricVersion, nil
}

// showDryRunUpdate shows what would be updated without applying changes.
func showDryRunUpdate(
	ctx context.Context,
	stdout io.Writer,
	serverState *state.ServerState,
	targetMCVersion, targetFabricVersion string,
	modrinthClient *modrinth.Client,
	jsonMode bool,
	flags *UpdateFlags,
) error {
	changes := map[string]interface{}{}

	if !flags.ModsOnly {
		changes["minecraft"] = map[string]string{
			"old": serverState.Minecraft.Version,
			"new": targetMCVersion,
		}
		changes["fabric"] = map[string]string{
			"old": serverState.Minecraft.FabricLoaderVersion,
			"new": targetFabricVersion,
		}
	}

	// Check mod updates
	modResults := []ModUpdateResult{}
	for _, mod := range serverState.Mods {
		result := ModUpdateResult{
			Slug:       mod.Slug,
			OldVersion: mod.Version,
		}

		// Find compatible version
		versions, err := modrinthClient.GetVersions(ctx, mod.ProjectID, &modrinth.VersionFilter{
			GameVersions: []string{targetMCVersion},
			Loaders:      []string{"fabric"},
		})

		if err != nil || len(versions) == 0 {
			result.Status = "skipped"
			result.Reason = "no compatible version found"
		} else {
			result.Status = "success"
			result.NewVersion = versions[0].VersionNumber
		}

		modResults = append(modResults, result)
	}

	changes["mods"] = modResults

	if jsonMode {
		output := UpdateOutput{
			Status:  "success",
			Message: "Dry run - no changes applied",
			Data:    changes,
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	// Human-readable output
	_, _ = fmt.Fprintln(stdout, "Dry run - no changes would be applied:")
	_, _ = fmt.Fprintln(stdout, "")

	if !flags.ModsOnly {
		_, _ = fmt.Fprintf(stdout, "Minecraft version: %s → %s\n",
			serverState.Minecraft.Version, targetMCVersion)
		_, _ = fmt.Fprintf(stdout, "Fabric loader: %s → %s\n",
			serverState.Minecraft.FabricLoaderVersion, targetFabricVersion)
		_, _ = fmt.Fprintln(stdout, "")
	}

	_, _ = fmt.Fprintln(stdout, "Mod updates:")
	for _, result := range modResults {
		switch result.Status {
		case "success":
			_, _ = fmt.Fprintf(stdout, "  ✓ %s: %s → %s\n",
				result.Slug, result.OldVersion, result.NewVersion)
		case "skipped":
			_, _ = fmt.Fprintf(stdout, "  ⚠ %s: %s (skipped - %s)\n",
				result.Slug, result.OldVersion, result.Reason)
		}
	}

	return nil
}

// executeUpdate performs the actual update workflow.
func executeUpdate(
	ctx context.Context,
	stdout io.Writer,
	serverState *state.ServerState,
	targetMCVersion, targetFabricVersion string,
	minecraftClient *minecraft.Client,
	modrinthClient *modrinth.Client,
	flags *UpdateFlags,
	jsonMode bool,
) (*UpdateSummary, error) {
	summary := &UpdateSummary{
		ServerName:   serverState.Name,
		MinecraftOld: serverState.Minecraft.Version,
		FabricOld:    serverState.Minecraft.FabricLoaderVersion,
		MinecraftNew: targetMCVersion,
		FabricNew:    targetFabricVersion,
	}

	// Step 1: Create backup
	if flags.Backup {
		if !jsonMode {
			_, _ = fmt.Fprintln(stdout, "Creating backup...")
		}

		backupService := backup.NewService()
		result, err := backupService.CreateBackup(ctx, backup.CreateBackupOptions{
			ServerName: serverState.Name,
			Compress:   true,
			KeepCount:  5,
		})
		if err != nil {
			return nil, fmt.Errorf("backup failed: %w", err)
		}

		summary.BackupID = result.BackupID

		if !jsonMode {
			_, _ = fmt.Fprintf(stdout, "✓ Backup created: %s\n", result.BackupID)
		}
	}

	// Step 2: Stop server if running
	containerClient, err := container.NewClient(ctx, container.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to container runtime: %w", err)
	}
	defer func() {
		_ = containerClient.Close()
	}()

	serverWasRunning := false
	if serverState.ContainerID != "" {
		info, err := containerClient.InspectContainer(ctx, serverState.ContainerID)
		if err == nil && isContainerRunning(info.State) {
			serverWasRunning = true

			if !jsonMode {
				_, _ = fmt.Fprintln(stdout, "Stopping server...")
			}

			timeout := 30 * time.Second
			if err := containerClient.StopContainer(ctx, serverState.ContainerID, &timeout); err != nil {
				return nil, fmt.Errorf("failed to stop server: %w", err)
			}

			if !jsonMode {
				_, _ = fmt.Fprintln(stdout, "✓ Server stopped")
			}
		}
	}

	// Step 3: Update Minecraft/Fabric versions
	if !flags.ModsOnly {
		serverState.Minecraft.Version = targetMCVersion
		serverState.Minecraft.FabricLoaderVersion = targetFabricVersion

		if !jsonMode {
			_, _ = fmt.Fprintf(stdout, "✓ Minecraft version: %s → %s\n",
				summary.MinecraftOld, summary.MinecraftNew)
			_, _ = fmt.Fprintf(stdout, "✓ Fabric loader: %s → %s\n",
				summary.FabricOld, summary.FabricNew)
		}
	}

	// Step 4: Update mods
	if len(serverState.Mods) > 0 {
		if !jsonMode {
			_, _ = fmt.Fprintln(stdout, "")
			_, _ = fmt.Fprintln(stdout, "Updating mods...")
		}

		modResults, err := updateMods(ctx, serverState, targetMCVersion, modrinthClient, jsonMode, stdout)
		if err != nil {
			return nil, fmt.Errorf("mod update failed: %w", err)
		}

		// Categorize results
		for _, result := range modResults {
			switch result.Status {
			case "success":
				summary.ModsUpdated = append(summary.ModsUpdated, result)
			case "skipped":
				summary.ModsSkipped = append(summary.ModsSkipped, result)
			case "failed":
				summary.ModsFailed = append(summary.ModsFailed, result)
			}
		}
	}

	// Step 5: Save updated server state
	if err := state.SaveServerState(ctx, serverState); err != nil {
		return nil, fmt.Errorf("failed to save server state: %w", err)
	}

	// Step 6: Recreate container
	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintln(stdout, "Recreating container...")
	}

	if err := recreateContainer(ctx, serverState, containerClient); err != nil {
		return nil, fmt.Errorf("failed to recreate container: %w", err)
	}

	if !jsonMode {
		_, _ = fmt.Fprintln(stdout, "✓ Container recreated")
	}

	// Step 7: Restart server if requested or was running
	if flags.Restart || serverWasRunning {
		if !jsonMode {
			_, _ = fmt.Fprintln(stdout, "Starting server...")
		}

		if err := containerClient.StartContainer(ctx, serverState.ContainerID); err != nil {
			return nil, fmt.Errorf("failed to start server: %w", err)
		}

		summary.Restarted = true

		if !jsonMode {
			_, _ = fmt.Fprintln(stdout, "✓ Server started")
		}
	}

	return summary, nil
}

// updateMods updates all mods to versions compatible with the target Minecraft version.
func updateMods(
	ctx context.Context,
	serverState *state.ServerState,
	targetMCVersion string,
	modrinthClient *modrinth.Client,
	jsonMode bool,
	stdout io.Writer,
) ([]ModUpdateResult, error) {
	results := []ModUpdateResult{}
	modInstaller := mods.NewInstaller()

	for i := range serverState.Mods {
		mod := &serverState.Mods[i]
		result := ModUpdateResult{
			Slug:       mod.Slug,
			OldVersion: mod.Version,
		}

		// Find compatible version
		versions, err := modrinthClient.GetVersions(ctx, mod.ProjectID, &modrinth.VersionFilter{
			GameVersions: []string{targetMCVersion},
			Loaders:      []string{"fabric"},
		})

		if err != nil || len(versions) == 0 {
			result.Status = "skipped"
			result.Reason = "no compatible version found"
			slog.Warn("no compatible version found for mod",
				"slug", mod.Slug,
				"minecraft_version", targetMCVersion,
			)
			results = append(results, result)

			if !jsonMode {
				_, _ = fmt.Fprintf(stdout, "  ⚠ %s: no compatible version (staying at %s)\n",
					mod.Slug, mod.Version)
			}
			continue
		}

		latestVersion := versions[0]

		// Skip if already at this version
		if latestVersion.VersionNumber == mod.Version {
			result.Status = "skipped"
			result.Reason = "already at latest compatible version"
			results = append(results, result)
			continue
		}

		// Download and replace mod file
		serverDir := filepath.Dir(serverState.Volumes.Data)
		modsDir := filepath.Join(serverDir, "mods")

		// Remove old mod file
		oldPath := filepath.Join(modsDir, mod.Filename)
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			result.Status = "failed"
			result.Reason = fmt.Sprintf("failed to remove old file: %v", err)
			results = append(results, result)

			if !jsonMode {
				_, _ = fmt.Fprintf(stdout, "  ✗ %s: %s\n", mod.Slug, result.Reason)
			}
			continue
		}

		// Download new version
		if err := modInstaller.DownloadFile(ctx, latestVersion.Files[0].URL, filepath.Join(modsDir, latestVersion.Files[0].Filename)); err != nil {
			result.Status = "failed"
			result.Reason = fmt.Sprintf("download failed: %v", err)
			results = append(results, result)

			if !jsonMode {
				_, _ = fmt.Fprintf(stdout, "  ✗ %s: %s\n", mod.Slug, result.Reason)
			}
			continue
		}

		// Update mod metadata
		mod.Version = latestVersion.VersionNumber
		mod.VersionID = latestVersion.ID
		mod.Filename = latestVersion.Files[0].Filename

		result.Status = "success"
		result.NewVersion = latestVersion.VersionNumber
		results = append(results, result)

		if !jsonMode {
			_, _ = fmt.Fprintf(stdout, "  ✓ %s: %s → %s\n",
				mod.Slug, result.OldVersion, result.NewVersion)
		}
	}

	return results, nil
}

// recreateContainer removes and recreates the container with updated configuration.
func recreateContainer(ctx context.Context, serverState *state.ServerState, client container.Client) error {
	// Remove old container (keep volumes!)
	if serverState.ContainerID != "" {
		opts := &container.RemoveOptions{
			Force: true,
		}
		if err := client.RemoveContainer(ctx, serverState.ContainerID, opts); err != nil {
			slog.Warn("failed to remove old container",
				"container_id", serverState.ContainerID,
				"error", err,
			)
		}
	}

	// Create new container with updated environment
	config := &container.ContainerConfig{
		Name:  serverState.Name,
		Image: "itzg/minecraft-server:latest",
		Env: map[string]string{
			"EULA":                  "TRUE",
			"TYPE":                  "FABRIC",
			"VERSION":               serverState.Minecraft.Version,
			"FABRIC_LOADER_VERSION": serverState.Minecraft.FabricLoaderVersion,
			"MEMORY":                serverState.Minecraft.Memory,
			"RCON_PASSWORD":         serverState.Minecraft.RconPassword,
			"ENABLE_RCON":           "true",
			"RCON_PORT":             fmt.Sprintf("%d", serverState.Minecraft.RconPort),
		},
		Ports: map[int]int{
			serverState.Minecraft.GamePort: 25565,
			serverState.Minecraft.RconPort: 25575,
		},
		Volumes: map[string]string{
			serverState.Volumes.Data: "/data",
		},
		Labels: map[string]string{
			"go-mc.managed":        "true",
			"go-mc.server.name":    serverState.Name,
			"go-mc.server.version": serverState.Minecraft.Version,
			"go-mc.fabric.version": serverState.Minecraft.FabricLoaderVersion,
		},
	}

	containerID, err := client.CreateContainer(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	serverState.ContainerID = containerID
	return nil
}

// outputUpdateSummary outputs the update summary.
func outputUpdateSummary(stdout io.Writer, summary *UpdateSummary, jsonMode bool) error {
	if jsonMode {
		data := map[string]interface{}{
			"server":    summary.ServerName,
			"backup_id": summary.BackupID,
			"minecraft": map[string]string{
				"old": summary.MinecraftOld,
				"new": summary.MinecraftNew,
			},
			"fabric": map[string]string{
				"old": summary.FabricOld,
				"new": summary.FabricNew,
			},
			"mods_updated": summary.ModsUpdated,
			"mods_skipped": summary.ModsSkipped,
			"mods_failed":  summary.ModsFailed,
			"restarted":    summary.Restarted,
		}

		output := UpdateOutput{
			Status: "success",
			Data:   data,
		}

		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	// Human-readable summary
	_, _ = fmt.Fprintln(stdout, "")
	_, _ = fmt.Fprintln(stdout, "Update complete!")
	_, _ = fmt.Fprintf(stdout, "- %d mods updated\n", len(summary.ModsUpdated))
	if len(summary.ModsSkipped) > 0 {
		_, _ = fmt.Fprintf(stdout, "- %d mods skipped (no compatible version)\n", len(summary.ModsSkipped))
	}
	if len(summary.ModsFailed) > 0 {
		_, _ = fmt.Fprintf(stdout, "- %d mods failed\n", len(summary.ModsFailed))
	}

	if summary.BackupID != "" {
		_, _ = fmt.Fprintf(stdout, "- Backup ID: %s\n", summary.BackupID)
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintln(stdout, "If you encounter issues, rollback with:")
		_, _ = fmt.Fprintf(stdout, "  go-mc servers restore %s %s\n", summary.ServerName, summary.BackupID)
	}

	return nil
}

// outputUpdateError outputs an error message.
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
