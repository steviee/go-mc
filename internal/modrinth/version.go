package modrinth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
)

// GetVersions fetches all versions for a project with optional filtering.
func (c *Client) GetVersions(ctx context.Context, projectID string, filter *VersionFilter) ([]Version, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	params := url.Values{}

	// Add filters if provided
	if filter != nil {
		if len(filter.Loaders) > 0 {
			loadersJSON, err := json.Marshal(filter.Loaders)
			if err != nil {
				return nil, fmt.Errorf("marshal loaders: %w", err)
			}
			params.Add("loaders", string(loadersJSON))
		}

		if len(filter.GameVersions) > 0 {
			versionsJSON, err := json.Marshal(filter.GameVersions)
			if err != nil {
				return nil, fmt.Errorf("marshal game versions: %w", err)
			}
			params.Add("game_versions", string(versionsJSON))
		}
	}

	path := "/project/" + projectID + "/version"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	slog.Debug("fetching project versions",
		"project_id", projectID,
		"filter", filter)

	// Execute request
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("get versions request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check response
	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	// Decode response
	var versions []Version
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	slog.Debug("versions retrieved",
		"project_id", projectID,
		"count", len(versions))

	return versions, nil
}

// FindCompatibleVersion finds the best compatible version for the given Minecraft and loader versions.
// Returns the latest compatible version if found.
func (c *Client) FindCompatibleVersion(ctx context.Context, projectID, minecraftVersion, loaderVersion string) (*Version, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	if minecraftVersion == "" {
		return nil, fmt.Errorf("minecraft version cannot be empty")
	}

	filter := &VersionFilter{
		Loaders:      []string{"fabric"},
		GameVersions: []string{minecraftVersion},
	}

	slog.Debug("finding compatible version",
		"project_id", projectID,
		"minecraft_version", minecraftVersion,
		"loader_version", loaderVersion)

	versions, err := c.GetVersions(ctx, projectID, filter)
	if err != nil {
		return nil, fmt.Errorf("get versions: %w", err)
	}

	if len(versions) == 0 {
		return nil, ErrNoCompatibleVersion
	}

	// Return the first version (latest compatible)
	slog.Debug("compatible version found",
		"project_id", projectID,
		"version", versions[0].VersionNumber)

	return &versions[0], nil
}

// GetPrimaryFile returns the primary file from a version.
func GetPrimaryFile(version *Version) (*File, error) {
	if version == nil {
		return nil, fmt.Errorf("version cannot be nil")
	}

	if len(version.Files) == 0 {
		return nil, fmt.Errorf("version has no files")
	}

	// Find primary file
	for i := range version.Files {
		if version.Files[i].Primary {
			return &version.Files[i], nil
		}
	}

	// If no primary file marked, return first file
	return &version.Files[0], nil
}

// ResolveDependencies recursively resolves mod dependencies.
func (c *Client) ResolveDependencies(ctx context.Context, version *Version, minecraftVersion string) ([]Version, error) {
	if version == nil {
		return nil, fmt.Errorf("version cannot be nil")
	}

	if minecraftVersion == "" {
		return nil, fmt.Errorf("minecraft version cannot be empty")
	}

	slog.Debug("resolving dependencies",
		"version", version.VersionNumber,
		"minecraft_version", minecraftVersion)

	resolved := make([]Version, 0)
	seen := make(map[string]bool)

	// Mark this version as seen
	seen[version.ProjectID] = true

	var err error
	resolved, err = c.resolveDepsRecursive(ctx, version, minecraftVersion, seen, resolved)
	if err != nil {
		return nil, err
	}

	slog.Debug("dependencies resolved",
		"count", len(resolved))

	return resolved, nil
}

// resolveDepsRecursive is the recursive helper for ResolveDependencies.
func (c *Client) resolveDepsRecursive(ctx context.Context, version *Version, minecraftVersion string, seen map[string]bool, resolved []Version) ([]Version, error) {
	for _, dep := range version.Dependencies {
		// Skip if not required
		if dep.DependencyType != "required" {
			continue
		}

		// Skip if no project ID
		if dep.ProjectID == "" {
			continue
		}

		// Check for circular dependency
		if seen[dep.ProjectID] {
			slog.Debug("circular dependency detected, skipping",
				"project_id", dep.ProjectID)
			continue
		}

		seen[dep.ProjectID] = true

		// Find compatible version for dependency
		depVersion, err := c.FindCompatibleVersion(ctx, dep.ProjectID, minecraftVersion, "")
		if err != nil {
			return nil, fmt.Errorf("resolve dependency %s: %w", dep.ProjectID, err)
		}

		resolved = append(resolved, *depVersion)

		// Recursively resolve dependencies of this dependency
		resolved, err = c.resolveDepsRecursive(ctx, depVersion, minecraftVersion, seen, resolved)
		if err != nil {
			return nil, err
		}
	}

	return resolved, nil
}
