package mods

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/steviee/go-mc/internal/modrinth"
	"github.com/steviee/go-mc/internal/state"
)

// Installer handles mod installation from Modrinth.
// It downloads mod files, saves them to the server's mods directory,
// tracks installed mods in the server state, and handles dependencies automatically.
type Installer struct {
	modrinthClient *modrinth.Client
	httpClient     *http.Client
}

// NewInstaller creates a new mod installer.
// The installer uses the Modrinth API to search for mods, resolve dependencies,
// and download mod files.
func NewInstaller() *Installer {
	return &Installer{
		modrinthClient: modrinth.NewClient(nil),
		httpClient:     &http.Client{},
	}
}

// InstallMods installs a list of mods (by slug) to a server.
// It automatically resolves and installs dependencies using the curated mod database.
// Dependencies are installed before the mods that require them.
//
// The method:
//  1. Loads the server state to get Minecraft version and mods directory
//  2. Resolves dependencies from the curated database
//  3. Downloads each mod file from Modrinth
//  4. Saves mod metadata to the server state
//
// Returns a list of installed mod slugs (including dependencies) and any error.
//
// Example:
//
//	installer := NewInstaller()
//	installed, err := installer.InstallMods(ctx, "my-server", []string{"lithium"})
//	// installed will be ["fabric-api", "lithium"] since lithium depends on fabric-api
func (i *Installer) InstallMods(ctx context.Context, serverName string, modSlugs []string) ([]string, error) {
	// Load server state
	serverState, err := state.LoadServerState(ctx, serverName)
	if err != nil {
		return nil, fmt.Errorf("load server state: %w", err)
	}

	// Get server's mods directory
	modsDir, err := getModsDir(serverState)
	if err != nil {
		return nil, fmt.Errorf("get mods directory: %w", err)
	}

	// Ensure mods directory exists
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return nil, fmt.Errorf("create mods directory: %w", err)
	}

	slog.Info("installing mods",
		"server", serverName,
		"mods", modSlugs,
		"mods_dir", modsDir)

	// Resolve dependencies to get full list of mods to install
	allModSlugs, err := ResolveDependencies(modSlugs)
	if err != nil {
		return nil, fmt.Errorf("resolve dependencies: %w", err)
	}

	slog.Debug("dependencies resolved",
		"requested", modSlugs,
		"total", allModSlugs)

	installed := []string{}

	// Install each mod
	for _, slug := range allModSlugs {
		// Check if already installed
		if i.isModInstalled(serverState, slug) {
			slog.Debug("mod already installed, skipping", "slug", slug)
			continue
		}

		// Install the mod
		modInfo, err := i.installSingleMod(ctx, serverState, slug, modsDir)
		if err != nil {
			return installed, fmt.Errorf("install mod %q: %w", slug, err)
		}

		// Add to server state
		serverState.Mods = append(serverState.Mods, modInfo)
		installed = append(installed, slug)

		slog.Info("mod installed",
			"slug", slug,
			"version", modInfo.Version,
			"filename", modInfo.Filename)
	}

	// Save updated server state
	if len(installed) > 0 {
		if err := state.SaveServerState(ctx, serverState); err != nil {
			return installed, fmt.Errorf("save server state: %w", err)
		}
	}

	return installed, nil
}

// installSingleMod installs a single mod and returns its ModInfo.
// It queries the Modrinth API to find a compatible version, downloads the file,
// and returns the mod metadata for storage in the server state.
func (i *Installer) installSingleMod(ctx context.Context, serverState *state.ServerState, slug string, modsDir string) (state.ModInfo, error) {
	// Get mod metadata from database
	dbMod, err := GetMod(slug)
	if err != nil {
		return state.ModInfo{}, err
	}

	// Find compatible version from Modrinth
	version, err := i.modrinthClient.FindCompatibleVersion(ctx, dbMod.ModrinthID, serverState.Minecraft.Version, "")
	if err != nil {
		return state.ModInfo{}, fmt.Errorf("find compatible version: %w", err)
	}

	// Get primary file
	file, err := modrinth.GetPrimaryFile(version)
	if err != nil {
		return state.ModInfo{}, fmt.Errorf("get primary file: %w", err)
	}

	// Download file
	destPath := filepath.Join(modsDir, file.Filename)
	if err := i.downloadFile(ctx, file.URL, destPath); err != nil {
		return state.ModInfo{}, fmt.Errorf("download file: %w", err)
	}

	// Create ModInfo for state
	modInfo := state.ModInfo{
		Name:         dbMod.Name,
		Slug:         slug,
		Version:      version.VersionNumber,
		ProjectID:    dbMod.ModrinthID,
		VersionID:    version.ID,
		URL:          file.URL,
		Filename:     file.Filename,
		SHA512:       "", // Modrinth API doesn't provide SHA512 in the file struct
		SizeBytes:    file.Size,
		Dependencies: dbMod.Dependencies,
	}

	return modInfo, nil
}

// downloadFile downloads a file from URL to destination path.
// It uses an HTTP GET request with context support for cancellation.
func (i *Installer) downloadFile(ctx context.Context, url, destPath string) error {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Execute request
	resp, err := i.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() {
		_ = out.Close()
	}()

	// Copy content
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	slog.Debug("file downloaded",
		"url", url,
		"destination", destPath)

	return nil
}

// isModInstalled checks if a mod is already installed by checking the server state.
func (i *Installer) isModInstalled(serverState *state.ServerState, slug string) bool {
	for _, mod := range serverState.Mods {
		if mod.Slug == slug {
			return true
		}
	}
	return false
}

// getModsDir returns the mods directory path for a server.
// The mods directory is located at <data_volume>/mods.
func getModsDir(serverState *state.ServerState) (string, error) {
	if serverState.Volumes.Data == "" {
		return "", fmt.Errorf("server data volume not configured")
	}

	return filepath.Join(serverState.Volumes.Data, "mods"), nil
}

// EnsureFabricAPI ensures Fabric API is installed on the server.
// This is an opinionated default since most Fabric mods require Fabric API.
// If Fabric API is already installed, this method does nothing.
//
// Example:
//
//	installer := NewInstaller()
//	if err := installer.EnsureFabricAPI(ctx, "my-server"); err != nil {
//	    log.Fatal(err)
//	}
func (i *Installer) EnsureFabricAPI(ctx context.Context, serverName string) error {
	slog.Info("ensuring Fabric API is installed", "server", serverName)

	// Check if already installed
	serverState, err := state.LoadServerState(ctx, serverName)
	if err != nil {
		return fmt.Errorf("load server state: %w", err)
	}

	if i.isModInstalled(serverState, "fabric-api") {
		slog.Debug("Fabric API already installed")
		return nil
	}

	// Install Fabric API
	_, err = i.InstallMods(ctx, serverName, []string{"fabric-api"})
	if err != nil {
		return fmt.Errorf("install Fabric API: %w", err)
	}

	return nil
}
