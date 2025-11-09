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
