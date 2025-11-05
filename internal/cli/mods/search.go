package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/modrinth"
)

var (
	searchVersion string
	searchLimit   int
	searchSort    string
)

// SearchOutput holds the output structure for JSON mode
type SearchOutput struct {
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// SearchResultData holds the search results for JSON output
type SearchResultData struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Downloads   int      `json:"downloads"`
	IconURL     string   `json:"icon_url"`
	Author      string   `json:"author"`
	Categories  []string `json:"categories"`
	ProjectID   string   `json:"project_id"`
}

// NewSearchCommand creates the mods search subcommand
func NewSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for Fabric mods on Modrinth",
		Long: `Search for Fabric mods on Modrinth with optional filtering.

The search defaults to showing only Fabric mods. Results can be filtered
by Minecraft version and sorted by different criteria.

Sort options:
  - relevance: Best match for search query (default)
  - downloads: Most downloaded mods first
  - updated:   Most recently updated mods first`,
		Example: `  # Search for a mod
  go-mc mods search sodium

  # Search with version filter
  go-mc mods search "fabric api" --version 1.21.1

  # Search with custom limit and sort
  go-mc mods search optimization --limit 50 --sort downloads

  # Get JSON output for scripting
  go-mc mods search lithium --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0])
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&searchVersion, "version", "v", "", "Filter by Minecraft version (e.g., 1.21.1)")
	cmd.Flags().IntVarP(&searchLimit, "limit", "l", 20, "Maximum results to show (1-100)")
	cmd.Flags().StringVar(&searchSort, "sort", "relevance", "Sort by: relevance, downloads, updated")

	return cmd
}

// runSearch executes the search command
func runSearch(ctx context.Context, stdout, stderr io.Writer, query string) error {
	jsonMode := isJSONMode()

	// Validate flags
	if searchLimit < 1 || searchLimit > 100 {
		return outputSearchError(stdout, jsonMode, fmt.Errorf("limit must be between 1 and 100"))
	}

	validSorts := map[string]bool{"relevance": true, "downloads": true, "updated": true}
	if !validSorts[searchSort] {
		return outputSearchError(stdout, jsonMode, fmt.Errorf("invalid sort: must be relevance, downloads, or updated"))
	}

	// Create Modrinth client
	client := modrinth.NewClient(nil)

	// Build search options
	opts := &modrinth.SearchOptions{
		Query: query,
		Limit: searchLimit,
		Facets: [][]string{
			{"project_type:mod"},
			{"categories:fabric"},
		},
	}

	// Add version filter if specified
	if searchVersion != "" {
		opts.Facets = append(opts.Facets, []string{
			fmt.Sprintf("versions:%s", searchVersion),
		})
	}

	// Perform search
	results, err := client.Search(ctx, opts)
	if err != nil {
		return outputSearchError(stdout, jsonMode, fmt.Errorf("search failed: %w", err))
	}

	// Sort results if needed (API defaults to relevance)
	if searchSort != "relevance" {
		sortResults(results.Hits, searchSort)
	}

	// Output results
	if jsonMode {
		return outputSearchJSON(stdout, results)
	}
	return outputSearchTable(stdout, results)
}

// sortResults sorts the search results by the specified field
func sortResults(hits []modrinth.Project, sortBy string) {
	switch sortBy {
	case "downloads":
		sort.Slice(hits, func(i, j int) bool {
			return hits[i].Downloads > hits[j].Downloads
		})
	case "updated":
		// Note: The Project struct doesn't currently have a DateModified field
		// This is a limitation of the current API response
		// For now, we keep the original order for "updated" sort
		// TODO: Update Project struct when DateModified is added
	}
}

// outputSearchTable outputs results in table format
func outputSearchTable(stdout io.Writer, results *modrinth.SearchResult) error {
	if len(results.Hits) == 0 {
		_, _ = fmt.Fprintln(stdout, "No mods found. Try a different search query.")
		return nil
	}

	// Table header
	_, _ = fmt.Fprintf(stdout, "%-20s %-25s %-10s %s\n",
		"SLUG", "NAME", "DOWNLOADS", "DESCRIPTION")
	_, _ = fmt.Fprintf(stdout, "%s\n", strings.Repeat("-", 100))

	// Table rows
	for _, mod := range results.Hits {
		slug := truncate(mod.Slug, 20)
		name := truncate(mod.Title, 25)
		downloads := formatDownloads(mod.Downloads)
		description := truncate(mod.Description, 40)

		_, _ = fmt.Fprintf(stdout, "%-20s %-25s %-10s %s\n",
			slug, name, downloads, description)
	}

	// Footer with result count
	if results.TotalHits > len(results.Hits) {
		_, _ = fmt.Fprintf(stdout, "\nShowing %d of %d results. Use --limit to see more.\n",
			len(results.Hits), results.TotalHits)
	} else {
		_, _ = fmt.Fprintf(stdout, "\nFound %d result(s).\n", results.TotalHits)
	}

	return nil
}

// outputSearchJSON outputs results in JSON format
func outputSearchJSON(stdout io.Writer, results *modrinth.SearchResult) error {
	// Convert to simplified format
	searchResults := make([]SearchResultData, len(results.Hits))
	for i, hit := range results.Hits {
		searchResults[i] = SearchResultData{
			Slug:        hit.Slug,
			Name:        hit.Title,
			Description: hit.Description,
			Downloads:   hit.Downloads,
			IconURL:     hit.IconURL,
			Author:      hit.Author,
			Categories:  hit.Categories,
			ProjectID:   hit.ProjectID,
		}
	}

	output := SearchOutput{
		Status: "success",
		Data: map[string]interface{}{
			"results": searchResults,
			"count":   len(results.Hits),
			"total":   results.TotalHits,
			"limit":   results.Limit,
			"offset":  results.Offset,
		},
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// outputSearchError outputs an error message
func outputSearchError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := SearchOutput{
			Status: "error",
			Error:  err.Error(),
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(output)
	}
	return err
}

// truncate truncates a string to the specified maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatDownloads formats download counts in human-readable format
func formatDownloads(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// formatTimeAgo formats a timestamp as a human-readable "time ago" string
func formatTimeAgo(dateStr string) string {
	// Parse ISO 8601 date
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return "unknown"
	}

	duration := time.Since(t)

	days := int(duration.Hours() / 24)
	if days == 0 {
		return "today"
	}
	if days == 1 {
		return "1 day"
	}
	if days < 7 {
		return fmt.Sprintf("%d days", days)
	}
	if days < 30 {
		weeks := days / 7
		if weeks == 1 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	}

	months := days / 30
	if months == 1 {
		return "1 month"
	}
	if months < 12 {
		return fmt.Sprintf("%d months", months)
	}

	years := months / 12
	if years == 1 {
		return "1 year"
	}
	return fmt.Sprintf("%d years", years)
}
