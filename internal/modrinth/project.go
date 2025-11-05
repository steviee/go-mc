package modrinth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// GetProject fetches project details by ID or slug.
func (c *Client) GetProject(ctx context.Context, idOrSlug string) (*ProjectDetails, error) {
	if idOrSlug == "" {
		return nil, fmt.Errorf("project ID or slug cannot be empty")
	}

	path := "/project/" + idOrSlug

	slog.Debug("fetching project details",
		"id_or_slug", idOrSlug)

	// Execute request
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("get project request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check response
	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	// Decode response
	var project ProjectDetails
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	slog.Debug("project details retrieved",
		"id", project.ID,
		"title", project.Title,
		"versions", len(project.Versions))

	return &project, nil
}
