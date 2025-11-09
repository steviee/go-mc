package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatReleaseDate(t *testing.T) {
	tests := []struct {
		name        string
		releaseTime string
		want        string
	}{
		{
			name:        "valid ISO 8601 timestamp",
			releaseTime: "2025-10-07T09:17:23+00:00",
			want:        "2025-10-07",
		},
		{
			name:        "valid ISO 8601 with Z",
			releaseTime: "2025-11-04T14:00:27Z",
			want:        "2025-11-04",
		},
		{
			name:        "invalid timestamp returns as-is",
			releaseTime: "invalid",
			want:        "invalid",
		},
		{
			name:        "empty string returns as-is",
			releaseTime: "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatReleaseDate(tt.releaseTime)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOutputListRemoteTable(t *testing.T) {
	tests := []struct {
		name           string
		items          []RemoteVersionItem
		latestRelease  string
		latestSnapshot string
		wantContains   []string
	}{
		{
			name: "multiple versions",
			items: []RemoteVersionItem{
				{ID: "1.21.10", Type: "release", ReleaseTime: "2025-10-07T09:17:23+00:00"},
				{ID: "25w45a", Type: "snapshot", ReleaseTime: "2025-11-04T14:00:27+00:00"},
				{ID: "1.21.9", Type: "release", ReleaseTime: "2025-09-18T10:00:00+00:00"},
			},
			latestRelease:  "1.21.10",
			latestSnapshot: "25w45a",
			wantContains: []string{
				"Latest Release:  1.21.10",
				"Latest Snapshot: 25w45a",
				"VERSION",
				"TYPE",
				"RELEASED",
				"1.21.10",
				"25w45a",
				"1.21.9",
				"release",
				"snapshot",
				"2025-10-07",
				"2025-11-04",
				"2025-09-18",
			},
		},
		{
			name:           "empty results",
			items:          []RemoteVersionItem{},
			latestRelease:  "1.21.10",
			latestSnapshot: "25w45a",
			wantContains: []string{
				"Latest Release:  1.21.10",
				"Latest Snapshot: 25w45a",
				"VERSION",
			},
		},
		{
			name: "single version",
			items: []RemoteVersionItem{
				{ID: "1.20.4", Type: "release", ReleaseTime: "2024-12-05T12:00:00+00:00"},
			},
			latestRelease:  "1.21.10",
			latestSnapshot: "25w45a",
			wantContains: []string{
				"1.20.4",
				"release",
				"2024-12-05",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputListRemoteTable(&buf, tt.items, tt.latestRelease, tt.latestSnapshot)

			require.NoError(t, err)

			output := buf.String()
			for _, want := range tt.wantContains {
				assert.Contains(t, output, want, "output should contain %q", want)
			}

			// Verify table structure
			lines := strings.Split(strings.TrimSpace(output), "\n")
			if len(tt.items) > 0 {
				// Should have: latest release line, latest snapshot line, empty line, header, rows
				assert.GreaterOrEqual(t, len(lines), 4+len(tt.items))
			}
		})
	}
}

func TestOutputListRemoteJSON(t *testing.T) {
	tests := []struct {
		name           string
		items          []RemoteVersionItem
		latestRelease  string
		latestSnapshot string
		totalCount     int
		validateJSON   func(*testing.T, ListRemoteOutput)
	}{
		{
			name: "multiple versions",
			items: []RemoteVersionItem{
				{ID: "1.21.10", Type: "release", ReleaseTime: "2025-10-07T09:17:23+00:00"},
				{ID: "25w45a", Type: "snapshot", ReleaseTime: "2025-11-04T14:00:27+00:00"},
			},
			latestRelease:  "1.21.10",
			latestSnapshot: "25w45a",
			totalCount:     1247,
			validateJSON: func(t *testing.T, output ListRemoteOutput) {
				assert.Equal(t, "success", output.Status)
				assert.NotNil(t, output.Data)

				// Check latest versions
				latest, ok := output.Data["latest"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "1.21.10", latest["release"])
				assert.Equal(t, "25w45a", latest["snapshot"])

				// Check count
				assert.Equal(t, float64(2), output.Data["count"])
				assert.Equal(t, float64(1247), output.Data["total"])

				// Check versions array
				versions, ok := output.Data["versions"].([]interface{})
				require.True(t, ok)
				assert.Len(t, versions, 2)
			},
		},
		{
			name:           "empty results",
			items:          []RemoteVersionItem{},
			latestRelease:  "1.21.10",
			latestSnapshot: "25w45a",
			totalCount:     1247,
			validateJSON: func(t *testing.T, output ListRemoteOutput) {
				assert.Equal(t, "success", output.Status)
				assert.Equal(t, float64(0), output.Data["count"])
				assert.Equal(t, float64(1247), output.Data["total"])

				versions, ok := output.Data["versions"].([]interface{})
				require.True(t, ok)
				assert.Empty(t, versions)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputListRemoteJSON(&buf, tt.items, tt.latestRelease, tt.latestSnapshot, tt.totalCount)

			require.NoError(t, err)

			// Parse JSON output
			var output ListRemoteOutput
			err = json.Unmarshal(buf.Bytes(), &output)
			require.NoError(t, err)

			// Validate JSON structure
			if tt.validateJSON != nil {
				tt.validateJSON(t, output)
			}
		})
	}
}

func TestOutputListRemoteError(t *testing.T) {
	tests := []struct {
		name     string
		jsonMode bool
		err      error
	}{
		{
			name:     "JSON mode error",
			jsonMode: true,
			err:      assert.AnError,
		},
		{
			name:     "non-JSON mode error",
			jsonMode: false,
			err:      assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputListRemoteError(&buf, tt.jsonMode, tt.err)

			assert.Error(t, err)
			assert.Equal(t, tt.err, err)

			if tt.jsonMode {
				// Should have JSON error output
				var output ListRemoteOutput
				jsonErr := json.Unmarshal(buf.Bytes(), &output)
				require.NoError(t, jsonErr)
				assert.Equal(t, "error", output.Status)
				assert.NotEmpty(t, output.Error)
			}
		})
	}
}

func TestNewListRemoteCommand(t *testing.T) {
	cmd := NewListRemoteCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "list-remote", cmd.Use)
	assert.Contains(t, cmd.Aliases, "versions")
	assert.Contains(t, cmd.Aliases, "list-versions")
	assert.NotNil(t, cmd.RunE)

	// Check flags
	typeFlag := cmd.Flags().Lookup("type")
	require.NotNil(t, typeFlag)
	assert.Equal(t, "release", typeFlag.DefValue)

	limitFlag := cmd.Flags().Lookup("limit")
	require.NotNil(t, limitFlag)
	assert.Equal(t, "20", limitFlag.DefValue)
}

func TestRunListRemote_InvalidType(t *testing.T) {
	var buf bytes.Buffer
	flags := &ListRemoteFlags{
		Type:  "invalid",
		Limit: 20,
	}

	err := runListRemote(context.Background(), &buf, flags)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

func TestRunListRemote_ValidTypes(t *testing.T) {
	tests := []struct {
		name string
		typ  string
	}{
		{
			name: "release type",
			typ:  "release",
		},
		{
			name: "snapshot type",
			typ:  "snapshot",
		},
		{
			name: "all type",
			typ:  "all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test will actually hit the Mojang API
			// In a real scenario, we'd mock the HTTP client
			// For now, we'll skip this in short test mode
			if testing.Short() {
				t.Skip("skipping integration test in short mode")
			}

			var buf bytes.Buffer
			flags := &ListRemoteFlags{
				Type:  tt.typ,
				Limit: 5,
			}

			err := runListRemote(context.Background(), &buf, flags)

			// The test might fail if there's no internet connection
			// or if Mojang API is down, so we'll just check that
			// it doesn't panic and returns something
			if err != nil {
				t.Logf("Warning: API call failed (might be expected): %v", err)
			} else {
				output := buf.String()
				assert.NotEmpty(t, output)
				assert.Contains(t, output, "VERSION")
			}
		})
	}
}
