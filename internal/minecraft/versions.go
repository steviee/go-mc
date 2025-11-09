package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	// VersionManifestURL is the Mojang API endpoint for the version manifest.
	VersionManifestURL = "https://launchermeta.mojang.com/mc/game/version_manifest.json"

	// FabricMetaAllLoadersURL is the Fabric Meta API endpoint for all loader versions.
	FabricMetaAllLoadersURL = "https://meta.fabricmc.net/v2/versions/loader"

	// FabricMetaLoaderForVersionURL is the Fabric Meta API endpoint template for loaders compatible with a specific Minecraft version.
	// Use with fmt.Sprintf(FabricMetaLoaderForVersionURL, minecraftVersion).
	FabricMetaLoaderForVersionURL = "https://meta.fabricmc.net/v2/versions/loader/%s"

	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second

	// UserAgent is the user agent string sent with API requests.
	UserAgent = "go-mc/dev (https://github.com/steviee/go-mc)"
)

// VersionManifest represents the Mojang version manifest response.
type VersionManifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []VersionInfo `json:"versions"`
}

// VersionInfo represents a single Minecraft version entry.
type VersionInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // "release" or "snapshot"
	URL         string `json:"url"`
	Time        string `json:"time"`
	ReleaseTime string `json:"releaseTime"`
}

// FabricLoader represents a Fabric loader version from the Fabric Meta API.
type FabricLoader struct {
	Version   string `json:"version"`   // Version string (e.g., "0.16.9")
	Build     int    `json:"build"`     // Build number
	Stable    bool   `json:"stable"`    // Whether this is a stable release
	Maven     string `json:"maven"`     // Maven coordinates
	Separator string `json:"separator"` // Version separator character
}

// FabricLoaderWrapper wraps FabricLoader for version-specific endpoint responses.
// The /v2/versions/loader/{version} endpoint returns objects with nested loader data.
type FabricLoaderWrapper struct {
	Loader FabricLoader `json:"loader"`
}

// Client is a Minecraft version API client.
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// Config holds client configuration.
type Config struct {
	Timeout   time.Duration
	UserAgent string
}

// NewClient creates a new Minecraft version API client.
func NewClient(config *Config) *Client {
	if config == nil {
		config = &Config{}
	}

	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	if config.UserAgent == "" {
		config.UserAgent = UserAgent
	}

	slog.Debug("creating Minecraft version API client",
		"timeout", config.Timeout)

	return &Client{
		httpClient: &http.Client{Timeout: config.Timeout},
		userAgent:  config.UserAgent,
	}
}

// GetVersionManifest fetches the version manifest from Mojang API.
// It returns the manifest containing all available Minecraft versions.
func (c *Client) GetVersionManifest(ctx context.Context) (*VersionManifest, error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, VersionManifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	slog.Debug("fetching Minecraft version manifest",
		"url", VersionManifestURL)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var manifest VersionManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	slog.Debug("fetched version manifest",
		"total_versions", len(manifest.Versions),
		"latest_release", manifest.Latest.Release,
		"latest_snapshot", manifest.Latest.Snapshot)

	return &manifest, nil
}

// FilterVersions filters versions by type and applies a limit.
// Valid types are "release", "snapshot", or "all".
// If limit is 0 or negative, all matching versions are returned.
func FilterVersions(versions []VersionInfo, versionType string, limit int) []VersionInfo {
	filtered := make([]VersionInfo, 0)

	for _, v := range versions {
		// Apply type filter
		if versionType != "all" && v.Type != versionType {
			continue
		}

		filtered = append(filtered, v)

		// Apply limit
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered
}

// GetFabricLoaders fetches all Fabric loader versions from Fabric Meta API.
// It returns a list of all available Fabric loader versions.
func (c *Client) GetFabricLoaders(ctx context.Context) ([]FabricLoader, error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, FabricMetaAllLoadersURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	slog.Debug("fetching all Fabric loader versions",
		"url", FabricMetaAllLoadersURL)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var loaders []FabricLoader
	if err := json.NewDecoder(resp.Body).Decode(&loaders); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	slog.Debug("fetched Fabric loader versions",
		"total_loaders", len(loaders))

	return loaders, nil
}

// GetFabricLoadersForVersion fetches Fabric loader versions compatible with a specific Minecraft version.
// It returns a list of Fabric loaders that support the given Minecraft version.
func (c *Client) GetFabricLoadersForVersion(ctx context.Context, minecraftVersion string) ([]FabricLoader, error) {
	// Build URL for specific Minecraft version
	url := fmt.Sprintf(FabricMetaLoaderForVersionURL, minecraftVersion)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	slog.Debug("fetching Fabric loaders for Minecraft version",
		"minecraft_version", minecraftVersion,
		"url", url)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response - this endpoint returns wrapper objects
	var wrappers []FabricLoaderWrapper
	if err := json.NewDecoder(resp.Body).Decode(&wrappers); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Extract loaders from wrappers
	loaders := make([]FabricLoader, len(wrappers))
	for i, wrapper := range wrappers {
		loaders[i] = wrapper.Loader
	}

	slog.Debug("fetched Fabric loaders for Minecraft version",
		"minecraft_version", minecraftVersion,
		"total_loaders", len(loaders))

	return loaders, nil
}

// GetLatestStableFabricLoader fetches the latest stable Fabric loader version.
// It returns the most recent stable Fabric loader, or an error if none are found.
func (c *Client) GetLatestStableFabricLoader(ctx context.Context) (*FabricLoader, error) {
	// Fetch all loaders
	loaders, err := c.GetFabricLoaders(ctx)
	if err != nil {
		return nil, fmt.Errorf("get fabric loaders: %w", err)
	}

	// Find latest stable loader
	for _, loader := range loaders {
		if loader.Stable {
			slog.Debug("found latest stable Fabric loader",
				"version", loader.Version,
				"build", loader.Build)
			return &loader, nil
		}
	}

	return nil, fmt.Errorf("no stable Fabric loader found")
}
