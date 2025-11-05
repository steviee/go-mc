package mods

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/steviee/go-mc/internal/modrinth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDownloads(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{15234, "15.2K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{45234567, "45.2M"},
		{98400000, "98.4M"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.input), func(t *testing.T) {
			got := formatDownloads(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "shorter than max",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "exact length",
			input:  "exactly ten",
			maxLen: 11,
			want:   "exactly ten",
		},
		{
			name:   "longer than max",
			input:  "this is too long for the limit",
			maxLen: 20,
			want:   "this is too long ...",
		},
		{
			name:   "very short max",
			input:  "hello",
			maxLen: 3,
			want:   "hel",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, len(got), tt.maxLen)
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		date time.Time
		want string
	}{
		{"today", now, "today"},
		{"1 day ago", now.AddDate(0, 0, -1), "1 day"},
		{"5 days ago", now.AddDate(0, 0, -5), "5 days"},
		{"1 week ago", now.AddDate(0, 0, -7), "1 week"},
		{"2 weeks ago", now.AddDate(0, 0, -14), "2 weeks"},
		{"1 month ago", now.AddDate(0, 0, -30), "1 month"},
		{"3 months ago", now.AddDate(0, 0, -90), "3 months"},
		{"1 year ago", now.AddDate(0, 0, -365), "1 year"},
		{"2 years ago", now.AddDate(0, 0, -730), "2 years"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dateStr := tt.date.Format(time.RFC3339)
			got := formatTimeAgo(dateStr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatTimeAgo_InvalidDate(t *testing.T) {
	result := formatTimeAgo("invalid-date")
	assert.Equal(t, "unknown", result)
}

func TestSearchCommand_FlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		limit       int
		sort        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid limit",
			limit:       50,
			sort:        "relevance",
			expectError: false,
		},
		{
			name:        "limit too low",
			limit:       0,
			sort:        "relevance",
			expectError: true,
			errorMsg:    "limit must be between 1 and 100",
		},
		{
			name:        "limit too high",
			limit:       101,
			sort:        "relevance",
			expectError: true,
			errorMsg:    "limit must be between 1 and 100",
		},
		{
			name:        "invalid sort",
			limit:       20,
			sort:        "invalid",
			expectError: true,
			errorMsg:    "invalid sort",
		},
		{
			name:        "valid downloads sort",
			limit:       20,
			sort:        "downloads",
			expectError: false,
		},
		{
			name:        "valid updated sort",
			limit:       20,
			sort:        "updated",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags to defaults
			searchLimit = tt.limit
			searchSort = tt.sort
			searchVersion = ""

			var stdout bytes.Buffer

			err := runSearch(context.Background(), &stdout, &bytes.Buffer{}, "test-query")

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				// We expect an error from the actual API call in unit tests,
				// but flag validation should pass
				// The error would be about network/connection, not validation
				if err != nil {
					assert.NotContains(t, err.Error(), "limit must be")
					assert.NotContains(t, err.Error(), "invalid sort")
				}
			}
		})
	}
}

func TestOutputSearchJSON_EmptyResults(t *testing.T) {
	var stdout bytes.Buffer

	results := &modrinth.SearchResult{
		Hits:      []modrinth.Project{},
		TotalHits: 0,
		Limit:     20,
		Offset:    0,
	}

	err := outputSearchJSON(&stdout, results)
	require.NoError(t, err)

	var output SearchOutput
	err = json.NewDecoder(&stdout).Decode(&output)
	require.NoError(t, err)

	assert.Equal(t, "success", output.Status)
	assert.Equal(t, float64(0), output.Data["count"])
	assert.Equal(t, float64(0), output.Data["total"])
}

func TestOutputSearchJSON_WithResults(t *testing.T) {
	var stdout bytes.Buffer

	results := &modrinth.SearchResult{
		Hits: []modrinth.Project{
			{
				Slug:        "sodium",
				Title:       "Sodium",
				Description: "Modern rendering engine",
				Downloads:   45000000,
				ProjectID:   "AANobbMI",
				Author:      "JellySquid",
				Categories:  []string{"optimization", "fabric"},
				IconURL:     "https://cdn.modrinth.com/sodium.png",
			},
			{
				Slug:        "lithium",
				Title:       "Lithium",
				Description: "General-purpose optimization",
				Downloads:   32000000,
				ProjectID:   "gvQqBUqZ",
				Author:      "JellySquid",
				Categories:  []string{"optimization", "fabric"},
				IconURL:     "https://cdn.modrinth.com/lithium.png",
			},
		},
		TotalHits: 156,
		Limit:     20,
		Offset:    0,
	}

	err := outputSearchJSON(&stdout, results)
	require.NoError(t, err)

	var output SearchOutput
	err = json.NewDecoder(&stdout).Decode(&output)
	require.NoError(t, err)

	assert.Equal(t, "success", output.Status)
	assert.Equal(t, float64(2), output.Data["count"])
	assert.Equal(t, float64(156), output.Data["total"])
	assert.Equal(t, float64(20), output.Data["limit"])
	assert.Equal(t, float64(0), output.Data["offset"])

	// Check results array
	resultsData := output.Data["results"].([]interface{})
	assert.Len(t, resultsData, 2)

	// Verify first result
	firstResult := resultsData[0].(map[string]interface{})
	assert.Equal(t, "sodium", firstResult["slug"])
	assert.Equal(t, "Sodium", firstResult["name"])
	assert.Equal(t, "Modern rendering engine", firstResult["description"])
}

func TestOutputSearchTable_EmptyResults(t *testing.T) {
	var stdout bytes.Buffer

	results := &modrinth.SearchResult{
		Hits:      []modrinth.Project{},
		TotalHits: 0,
		Limit:     20,
		Offset:    0,
	}

	err := outputSearchTable(&stdout, results)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "No mods found")
}

func TestOutputSearchTable_WithResults(t *testing.T) {
	var stdout bytes.Buffer

	results := &modrinth.SearchResult{
		Hits: []modrinth.Project{
			{
				Slug:        "sodium",
				Title:       "Sodium",
				Description: "Modern rendering engine and optimization mod for Minecraft",
				Downloads:   45234567,
			},
			{
				Slug:        "fabric-api",
				Title:       "Fabric API",
				Description: "Essential hooks for Fabric mods",
				Downloads:   98400000,
			},
		},
		TotalHits: 2,
		Limit:     20,
		Offset:    0,
	}

	err := outputSearchTable(&stdout, results)
	require.NoError(t, err)

	output := stdout.String()

	// Check header
	assert.Contains(t, output, "SLUG")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "DOWNLOADS")
	assert.Contains(t, output, "DESCRIPTION")

	// Check content
	assert.Contains(t, output, "sodium")
	assert.Contains(t, output, "Sodium")
	assert.Contains(t, output, "45.2M")
	assert.Contains(t, output, "fabric-api")
	assert.Contains(t, output, "Fabric API")
	assert.Contains(t, output, "98.4M")

	// Check footer
	assert.Contains(t, output, "Found 2 result(s)")
}

func TestOutputSearchTable_PartialResults(t *testing.T) {
	var stdout bytes.Buffer

	results := &modrinth.SearchResult{
		Hits: []modrinth.Project{
			{Slug: "mod1", Title: "Mod 1", Description: "Description 1", Downloads: 1000},
			{Slug: "mod2", Title: "Mod 2", Description: "Description 2", Downloads: 2000},
		},
		TotalHits: 156, // More results available
		Limit:     2,
		Offset:    0,
	}

	err := outputSearchTable(&stdout, results)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Showing 2 of 156 results")
	assert.Contains(t, output, "Use --limit to see more")
}

func TestOutputSearchError_JSON(t *testing.T) {
	var stdout bytes.Buffer

	returnedErr := outputSearchError(&stdout, true, fmt.Errorf("test error"))

	var output SearchOutput
	decodeErr := json.NewDecoder(&stdout).Decode(&output)
	require.NoError(t, decodeErr)

	assert.Equal(t, "error", output.Status)
	assert.Equal(t, "test error", output.Error)
	require.Error(t, returnedErr)
}

func TestOutputSearchError_Text(t *testing.T) {
	var stdout bytes.Buffer

	err := outputSearchError(&stdout, false, fmt.Errorf("test error"))

	require.Error(t, err)
	assert.Equal(t, "test error", err.Error())
}

func TestNewSearchCommand(t *testing.T) {
	cmd := NewSearchCommand()

	assert.Equal(t, "search", cmd.Use[:6])
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)

	// Check flags exist
	versionFlag := cmd.Flags().Lookup("version")
	require.NotNil(t, versionFlag)
	assert.Equal(t, "v", versionFlag.Shorthand)

	limitFlag := cmd.Flags().Lookup("limit")
	require.NotNil(t, limitFlag)
	assert.Equal(t, "l", limitFlag.Shorthand)

	sortFlag := cmd.Flags().Lookup("sort")
	require.NotNil(t, sortFlag)
}

func TestTruncate_EdgeCases(t *testing.T) {
	// Test with multibyte UTF-8 characters
	input := "Hello 世界 World"
	result := truncate(input, 10)
	assert.LessOrEqual(t, len(result), 10)

	// Test with exactly maxLen
	exact := "exact"
	result = truncate(exact, 5)
	assert.Equal(t, "exact", result)

	// Test with maxLen = 0
	result = truncate("test", 0)
	assert.Equal(t, "", result)
}
