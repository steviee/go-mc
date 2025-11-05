package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVersionCommand(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		commit    string
		date      string
		builtBy   string
		wantUse   string
		wantShort string
	}{
		{
			name:      "creates version command",
			version:   "1.0.0",
			commit:    "abc123",
			date:      "2025-11-05",
			builtBy:   "goreleaser",
			wantUse:   "version",
			wantShort: "Print version information",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewVersionCommand(tt.version, tt.commit, tt.date, tt.builtBy)

			assert.Equal(t, tt.wantUse, cmd.Use)
			assert.Equal(t, tt.wantShort, cmd.Short)
			assert.NotEmpty(t, cmd.Long)
			assert.NotEmpty(t, cmd.Example)
		})
	}
}

func TestPrintVersion_TextFormat(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		builtBy string
		want    []string
	}{
		{
			name:    "prints all version info",
			version: "1.0.0",
			commit:  "abc123",
			date:    "2025-11-05",
			builtBy: "goreleaser",
			want: []string{
				"go-mc version 1.0.0",
				"Commit: abc123",
				"Built: 2025-11-05",
				"Built by: goreleaser",
			},
		},
		{
			name:    "prints dev version",
			version: "dev",
			commit:  "unknown",
			date:    "unknown",
			builtBy: "unknown",
			want: []string{
				"go-mc version dev",
				"Commit: unknown",
				"Built: unknown",
				"Built by: unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := printVersion(&buf, tt.version, tt.commit, tt.date, tt.builtBy)
			require.NoError(t, err)

			output := buf.String()
			for _, want := range tt.want {
				assert.Contains(t, output, want)
			}
		})
	}
}

func TestPrintVersion_JSONFormat(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		builtBy string
	}{
		{
			name:    "prints JSON output",
			version: "1.0.0",
			commit:  "abc123",
			date:    "2025-11-05",
			builtBy: "goreleaser",
		},
		{
			name:    "prints JSON with dev version",
			version: "dev",
			commit:  "unknown",
			date:    "unknown",
			builtBy: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set JSON output mode
			jsonOut = true
			defer func() { jsonOut = false }()

			var buf bytes.Buffer

			err := printVersion(&buf, tt.version, tt.commit, tt.date, tt.builtBy)
			require.NoError(t, err)

			// Verify valid JSON
			var result struct {
				Status string      `json:"status"`
				Data   VersionInfo `json:"data"`
			}
			err = json.Unmarshal(buf.Bytes(), &result)
			require.NoError(t, err)

			// Verify content
			assert.Equal(t, "success", result.Status)
			assert.Equal(t, tt.version, result.Data.Version)
			assert.Equal(t, tt.commit, result.Data.Commit)
			assert.Equal(t, tt.date, result.Data.Date)
			assert.Equal(t, tt.builtBy, result.Data.BuiltBy)
		})
	}
}

func TestVersionCommand_Execute(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		version    string
		commit     string
		date       string
		builtBy    string
		jsonMode   bool
		wantOutput []string
		wantErr    bool
	}{
		{
			name:     "execute version command text output",
			args:     []string{},
			version:  "1.0.0",
			commit:   "abc123",
			date:     "2025-11-05",
			builtBy:  "goreleaser",
			jsonMode: false,
			wantOutput: []string{
				"go-mc version 1.0.0",
				"Commit: abc123",
			},
			wantErr: false,
		},
		{
			name:     "execute version command json output",
			args:     []string{},
			version:  "1.0.0",
			commit:   "abc123",
			date:     "2025-11-05",
			builtBy:  "goreleaser",
			jsonMode: true,
			wantOutput: []string{
				`"status": "success"`,
				`"version": "1.0.0"`,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonOut = tt.jsonMode
			defer func() { jsonOut = false }()

			cmd := NewVersionCommand(tt.version, tt.commit, tt.date, tt.builtBy)
			cmd.SetArgs(tt.args)

			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			err := cmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				output := out.String()
				for _, want := range tt.wantOutput {
					assert.Contains(t, output, want)
				}
			}
		})
	}
}

func TestVersionInfo_JSONMarshal(t *testing.T) {
	info := VersionInfo{
		Version: "1.0.0",
		Commit:  "abc123",
		Date:    "2025-11-05",
		BuiltBy: "goreleaser",
	}

	data, err := json.Marshal(info)
	require.NoError(t, err)

	var result VersionInfo
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, info.Version, result.Version)
	assert.Equal(t, info.Commit, result.Commit)
	assert.Equal(t, info.Date, result.Date)
	assert.Equal(t, info.BuiltBy, result.BuiltBy)
}

func TestPrintVersionJSON_ValidJSON(t *testing.T) {
	info := VersionInfo{
		Version: "1.0.0",
		Commit:  "abc123",
		Date:    "2025-11-05",
		BuiltBy: "goreleaser",
	}

	var buf bytes.Buffer
	err := printVersionJSON(&buf, info)
	require.NoError(t, err)

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, result, "status")
	assert.Contains(t, result, "data")
}

func TestPrintVersionText_Format(t *testing.T) {
	info := VersionInfo{
		Version: "1.0.0",
		Commit:  "abc123",
		Date:    "2025-11-05",
		BuiltBy: "goreleaser",
	}

	var buf bytes.Buffer
	err := printVersionText(&buf, info)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 4)

	assert.Contains(t, lines[0], "go-mc version 1.0.0")
	assert.Contains(t, lines[1], "Commit: abc123")
	assert.Contains(t, lines[2], "Built: 2025-11-05")
	assert.Contains(t, lines[3], "Built by: goreleaser")
}
