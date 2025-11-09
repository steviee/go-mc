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

func TestOutputFabricLoadersTable(t *testing.T) {
	tests := []struct {
		name             string
		items            []FabricLoaderItem
		minecraftVersion string
		wantContains     []string
	}{
		{
			name: "multiple loaders with minecraft version",
			items: []FabricLoaderItem{
				{Version: "0.16.9", Build: 301, Stable: true},
				{Version: "0.16.8", Build: 300, Stable: true},
				{Version: "0.16.7", Build: 299, Stable: false},
			},
			minecraftVersion: "1.21.1",
			wantContains: []string{
				"MINECRAFT VERSION: 1.21.1",
				"LOADER VERSION",
				"BUILD",
				"STABLE",
				"0.16.9",
				"0.16.8",
				"0.16.7",
				"301",
				"300",
				"299",
				"yes",
				"no",
			},
		},
		{
			name: "multiple loaders without minecraft version",
			items: []FabricLoaderItem{
				{Version: "0.16.9", Build: 301, Stable: true},
				{Version: "0.16.8", Build: 300, Stable: true},
			},
			minecraftVersion: "",
			wantContains: []string{
				"LOADER VERSION",
				"BUILD",
				"STABLE",
				"0.16.9",
				"0.16.8",
				"301",
				"300",
				"yes",
			},
		},
		{
			name:             "empty results",
			items:            []FabricLoaderItem{},
			minecraftVersion: "",
			wantContains: []string{
				"LOADER VERSION",
				"BUILD",
				"STABLE",
			},
		},
		{
			name: "single loader",
			items: []FabricLoaderItem{
				{Version: "0.16.9", Build: 301, Stable: true},
			},
			minecraftVersion: "1.20.4",
			wantContains: []string{
				"MINECRAFT VERSION: 1.20.4",
				"0.16.9",
				"301",
				"yes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputFabricLoadersTable(&buf, tt.items, tt.minecraftVersion)

			require.NoError(t, err)

			output := buf.String()
			for _, want := range tt.wantContains {
				assert.Contains(t, output, want, "output should contain %q", want)
			}

			// Verify table structure
			lines := strings.Split(strings.TrimSpace(output), "\n")
			if tt.minecraftVersion != "" {
				// Should have: minecraft version line, empty line, header, rows
				assert.GreaterOrEqual(t, len(lines), 3+len(tt.items))
			} else {
				// Should have: header, rows
				assert.GreaterOrEqual(t, len(lines), 1+len(tt.items))
			}

			// Verify "yes"/"no" formatting for stable field
			if len(tt.items) > 0 {
				for _, item := range tt.items {
					if item.Stable {
						assert.Contains(t, output, "yes")
					} else {
						assert.Contains(t, output, "no")
					}
				}
			}
		})
	}
}

func TestOutputFabricLoadersJSON(t *testing.T) {
	tests := []struct {
		name             string
		items            []FabricLoaderItem
		minecraftVersion string
		validateJSON     func(*testing.T, ListRemoteOutput)
	}{
		{
			name: "multiple loaders with minecraft version",
			items: []FabricLoaderItem{
				{Version: "0.16.9", Build: 301, Stable: true},
				{Version: "0.16.8", Build: 300, Stable: true},
				{Version: "0.16.7", Build: 299, Stable: false},
			},
			minecraftVersion: "1.21.1",
			validateJSON: func(t *testing.T, output ListRemoteOutput) {
				assert.Equal(t, "success", output.Status)
				assert.NotNil(t, output.Data)

				// Check minecraft_version
				mcVersion, ok := output.Data["minecraft_version"].(string)
				require.True(t, ok)
				assert.Equal(t, "1.21.1", mcVersion)

				// Check loaders array
				loaders, ok := output.Data["loaders"].([]interface{})
				require.True(t, ok)
				assert.Len(t, loaders, 3)

				// Check latest_stable
				latestStable, ok := output.Data["latest_stable"].(string)
				require.True(t, ok)
				assert.Equal(t, "0.16.9", latestStable)

				// Check count
				count, ok := output.Data["count"].(float64)
				require.True(t, ok)
				assert.Equal(t, float64(3), count)
			},
		},
		{
			name: "multiple loaders without minecraft version",
			items: []FabricLoaderItem{
				{Version: "0.16.9", Build: 301, Stable: true},
				{Version: "0.16.8", Build: 300, Stable: false},
			},
			minecraftVersion: "",
			validateJSON: func(t *testing.T, output ListRemoteOutput) {
				assert.Equal(t, "success", output.Status)
				assert.NotNil(t, output.Data)

				// Check minecraft_version is null
				mcVersion := output.Data["minecraft_version"]
				assert.Nil(t, mcVersion)

				// Check loaders array
				loaders, ok := output.Data["loaders"].([]interface{})
				require.True(t, ok)
				assert.Len(t, loaders, 2)

				// Check latest_stable
				latestStable, ok := output.Data["latest_stable"].(string)
				require.True(t, ok)
				assert.Equal(t, "0.16.9", latestStable)

				// Check count
				count, ok := output.Data["count"].(float64)
				require.True(t, ok)
				assert.Equal(t, float64(2), count)
			},
		},
		{
			name:             "empty results",
			items:            []FabricLoaderItem{},
			minecraftVersion: "",
			validateJSON: func(t *testing.T, output ListRemoteOutput) {
				assert.Equal(t, "success", output.Status)
				assert.NotNil(t, output.Data)

				// Check count
				count, ok := output.Data["count"].(float64)
				require.True(t, ok)
				assert.Equal(t, float64(0), count)

				// Check loaders array
				loaders, ok := output.Data["loaders"].([]interface{})
				require.True(t, ok)
				assert.Empty(t, loaders)

				// Check latest_stable is empty
				latestStable, ok := output.Data["latest_stable"].(string)
				require.True(t, ok)
				assert.Equal(t, "", latestStable)
			},
		},
		{
			name: "no stable loaders",
			items: []FabricLoaderItem{
				{Version: "0.16.7", Build: 299, Stable: false},
				{Version: "0.16.6", Build: 298, Stable: false},
			},
			minecraftVersion: "1.20.4",
			validateJSON: func(t *testing.T, output ListRemoteOutput) {
				assert.Equal(t, "success", output.Status)

				// Check latest_stable is empty when no stable loaders
				latestStable, ok := output.Data["latest_stable"].(string)
				require.True(t, ok)
				assert.Equal(t, "", latestStable)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputFabricLoadersJSON(&buf, tt.items, tt.minecraftVersion)

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

func TestRunListRemote_VersionWithoutLoaders(t *testing.T) {
	var buf bytes.Buffer
	flags := &ListRemoteFlags{
		Type:    "release",
		Limit:   20,
		Loaders: false,
		Version: "1.21.1",
	}

	err := runListRemote(context.Background(), &buf, flags)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--version flag requires --loaders flag")
}

func TestNewListRemoteCommand_LoadersFlags(t *testing.T) {
	cmd := NewListRemoteCommand()

	require.NotNil(t, cmd)

	// Check --loaders flag exists with correct default
	loadersFlag := cmd.Flags().Lookup("loaders")
	require.NotNil(t, loadersFlag)
	assert.Equal(t, "false", loadersFlag.DefValue)
	assert.Equal(t, "bool", loadersFlag.Value.Type())

	// Check --version flag exists with correct default
	versionFlag := cmd.Flags().Lookup("version")
	require.NotNil(t, versionFlag)
	assert.Equal(t, "", versionFlag.DefValue)
	assert.Equal(t, "string", versionFlag.Value.Type())
}
