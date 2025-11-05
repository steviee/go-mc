package modrinth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
)

// Search searches for mods on Modrinth.
func (c *Client) Search(ctx context.Context, opts *SearchOptions) (*SearchResult, error) {
	if opts == nil {
		opts = &SearchOptions{}
	}

	// Set default limit
	if opts.Limit == 0 {
		opts.Limit = 20
	}

	// Validate limit
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Build query parameters
	params := url.Values{}
	if opts.Query != "" {
		params.Add("query", opts.Query)
	}

	// Add facets (filters)
	if len(opts.Facets) > 0 {
		facetsJSON, err := json.Marshal(opts.Facets)
		if err != nil {
			return nil, fmt.Errorf("marshal facets: %w", err)
		}
		params.Add("facets", string(facetsJSON))
	}

	params.Add("limit", strconv.Itoa(opts.Limit))
	params.Add("offset", strconv.Itoa(opts.Offset))

	path := "/search?" + params.Encode()

	slog.Debug("searching Modrinth",
		"query", opts.Query,
		"limit", opts.Limit,
		"offset", opts.Offset)

	// Execute request
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check response
	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	// Decode response
	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	slog.Debug("search completed",
		"hits", len(result.Hits),
		"total", result.TotalHits)

	return &result, nil
}

// SearchMods is a convenience function for searching mods with Fabric loader filter.
func (c *Client) SearchMods(ctx context.Context, query string, limit int) (*SearchResult, error) {
	if query == "" {
		return nil, ErrInvalidSearchQuery
	}

	opts := &SearchOptions{
		Query: query,
		Facets: [][]string{
			{"project_type:mod"},
			{"categories:fabric"},
		},
		Limit: limit,
	}

	return c.Search(ctx, opts)
}
