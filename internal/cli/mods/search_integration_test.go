//go:build integration

package mods

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchCommand_Integration_RealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		name         string
		query        string
		limit        int
		sort         string
		version      string
		expectHits   bool
		expectInName string
	}{
		{
			name:         "search fabric-api",
			query:        "fabric-api",
			limit:        5,
			sort:         "relevance",
			expectHits:   true,
			expectInName: "Fabric API",
		},
		{
			name:         "search sodium",
			query:        "sodium",
			limit:        10,
			sort:         "downloads",
			expectHits:   true,
			expectInName: "Sodium",
		},
		{
			name:       "search with version filter",
			query:      "optimization",
			limit:      5,
			sort:       "relevance",
			version:    "1.21.1",
			expectHits: true,
		},
		{
			name:       "empty results",
			query:      "xyznonexistentmodthatdoesnotexist",
			limit:      5,
			sort:       "relevance",
			expectHits: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			searchLimit = tt.limit
			searchSort = tt.sort
			searchVersion = tt.version

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := runSearch(context.Background(), &stdout, &stderr, tt.query)

			if !tt.expectHits {
				// Empty results should not error
				require.NoError(t, err)
				output := stdout.String()
				assert.Contains(t, output, "No mods found")
				return
			}

			require.NoError(t, err)

			output := stdout.String()

			// Check table headers
			assert.Contains(t, output, "SLUG")
			assert.Contains(t, output, "NAME")
			assert.Contains(t, output, "DOWNLOADS")
			assert.Contains(t, output, "DESCRIPTION")

			// Check for expected content if specified
			if tt.expectInName != "" {
				assert.Contains(t, output, tt.expectInName)
			}

			// Verify download counts are formatted
			assert.True(t, strings.Contains(output, "K") || strings.Contains(output, "M"),
				"should contain formatted download counts")
		})
	}
}

func TestSearchCommand_Integration_JSONOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Save original state
	originalLimit := searchLimit
	originalSort := searchSort
	originalVersion := searchVersion

	// Reset state after test
	defer func() {
		searchLimit = originalLimit
		searchSort = originalSort
		searchVersion = originalVersion
	}()

	// Set test parameters
	searchLimit = 5
	searchSort = "relevance"
	searchVersion = ""

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Note: We need to mock IsJSONOutput() to return true for this test
	// For now, this test demonstrates the integration but won't actually
	// use JSON mode without the global flag set

	err := runSearch(context.Background(), &stdout, &stderr, "fabric-api")
	require.NoError(t, err)

	// For non-JSON mode, just verify we got output
	output := stdout.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "SLUG")
}

func TestSearchCommand_Integration_SortByDownloads(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	searchLimit = 10
	searchSort = "downloads"
	searchVersion = ""

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runSearch(context.Background(), &stdout, &stderr, "optimization")
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "SLUG")

	// Results should be sorted by downloads (high to low)
	// Most popular optimization mods should appear first
	lines := strings.Split(output, "\n")
	assert.Greater(t, len(lines), 3, "should have header and at least one result")
}

func TestSearchCommand_Integration_VersionFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	searchLimit = 5
	searchSort = "relevance"
	searchVersion = "1.21.1"

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runSearch(context.Background(), &stdout, &stderr, "fabric")
	require.NoError(t, err)

	output := stdout.String()

	// Should return results compatible with 1.21.1
	assert.Contains(t, output, "SLUG")
}

func TestSearchCommand_Integration_LargeLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	searchLimit = 100 // Maximum allowed
	searchSort = "relevance"
	searchVersion = ""

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runSearch(context.Background(), &stdout, &stderr, "fabric")
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "SLUG")

	// Count result lines (exclude header and footer)
	lines := strings.Split(output, "\n")
	resultLines := 0
	for _, line := range lines {
		if strings.Contains(line, "  ") && !strings.Contains(line, "SLUG") &&
			!strings.Contains(line, "Showing") && !strings.Contains(line, "Found") &&
			len(strings.TrimSpace(line)) > 0 && !strings.HasPrefix(line, "-") {
			resultLines++
		}
	}

	// Should get multiple results (exact count depends on API)
	assert.Greater(t, resultLines, 0)
}

func TestSearchCommand_Integration_RateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Make multiple rapid requests to test rate limiting behavior
	searchLimit = 5
	searchSort = "relevance"
	searchVersion = ""

	for i := 0; i < 5; i++ {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		err := runSearch(context.Background(), &stdout, &stderr, "fabric")

		// Should handle rate limiting gracefully
		if err != nil {
			// If we hit rate limit, error should be clear
			assert.Contains(t, err.Error(), "rate")
		} else {
			// Otherwise should succeed
			output := stdout.String()
			assert.Contains(t, output, "SLUG")
		}
	}
}
