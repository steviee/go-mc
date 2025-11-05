package servers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name      string
		startedAt time.Time
		want      string
	}{
		{
			name:      "zero time",
			startedAt: time.Time{},
			want:      "-",
		},
		{
			name:      "5 minutes ago",
			startedAt: time.Now().Add(-5 * time.Minute),
			want:      "5m",
		},
		{
			name:      "1 hour 30 minutes ago",
			startedAt: time.Now().Add(-90 * time.Minute),
			want:      "1h 30m",
		},
		{
			name:      "2 days 3 hours ago",
			startedAt: time.Now().Add(-51 * time.Hour),
			want:      "2d 3h",
		},
		{
			name:      "1 day ago",
			startedAt: time.Now().Add(-24 * time.Hour),
			want:      "1d 0h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUptime(tt.startedAt)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeContainerState(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  string
	}{
		{
			name:  "running",
			state: "running",
			want:  "running",
		},
		{
			name:  "Running (capital)",
			state: "Running",
			want:  "running",
		},
		{
			name:  "created",
			state: "created",
			want:  "created",
		},
		{
			name:  "stopped",
			state: "stopped",
			want:  "stopped",
		},
		{
			name:  "exited",
			state: "exited",
			want:  "stopped",
		},
		{
			name:  "paused",
			state: "paused",
			want:  "paused",
		},
		{
			name:  "unknown state",
			state: "restarting",
			want:  "restarting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeContainerState(tt.state)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterServers(t *testing.T) {
	items := []ServerListItem{
		{Name: "server1", Status: "running"},
		{Name: "server2", Status: "stopped"},
		{Name: "server3", Status: "created"},
		{Name: "server4", Status: "running"},
		{Name: "server5", Status: "exited"},
	}

	tests := []struct {
		name     string
		flags    *ListFlags
		wantLen  int
		wantFunc func(*testing.T, []ServerListItem)
	}{
		{
			name: "default - only running",
			flags: &ListFlags{
				All:    false,
				Filter: "",
			},
			wantLen: 2,
			wantFunc: func(t *testing.T, items []ServerListItem) {
				for _, item := range items {
					assert.Equal(t, "running", item.Status)
				}
			},
		},
		{
			name: "all flag",
			flags: &ListFlags{
				All:    true,
				Filter: "",
			},
			wantLen: 5,
		},
		{
			name: "filter all",
			flags: &ListFlags{
				All:    false,
				Filter: "all",
			},
			wantLen: 5,
		},
		{
			name: "filter running",
			flags: &ListFlags{
				All:    false,
				Filter: "running",
			},
			wantLen: 2,
			wantFunc: func(t *testing.T, items []ServerListItem) {
				for _, item := range items {
					assert.Equal(t, "running", item.Status)
				}
			},
		},
		{
			name: "filter stopped",
			flags: &ListFlags{
				All:    false,
				Filter: "stopped",
			},
			wantLen: 3,
			wantFunc: func(t *testing.T, items []ServerListItem) {
				for _, item := range items {
					assert.Contains(t, []string{"stopped", "created", "exited"}, item.Status)
				}
			},
		},
		{
			name: "filter created",
			flags: &ListFlags{
				All:    false,
				Filter: "created",
			},
			wantLen: 1,
			wantFunc: func(t *testing.T, items []ServerListItem) {
				assert.Equal(t, "created", items[0].Status)
			},
		},
		{
			name: "unknown filter shows all",
			flags: &ListFlags{
				All:    false,
				Filter: "invalid",
			},
			wantLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterServers(items, tt.flags)
			assert.Len(t, got, tt.wantLen)
			if tt.wantFunc != nil {
				tt.wantFunc(t, got)
			}
		})
	}
}

func TestSortServers(t *testing.T) {
	now := time.Now()
	items := []ServerListItem{
		{Name: "server-c", Status: "running", Port: 25567, StartedAt: now.Add(-2 * time.Hour)},
		{Name: "server-a", Status: "stopped", Port: 25565},
		{Name: "server-b", Status: "running", Port: 25566, StartedAt: now.Add(-1 * time.Hour)},
	}

	tests := []struct {
		name     string
		sortBy   string
		wantFunc func(*testing.T, []ServerListItem)
	}{
		{
			name:   "sort by name",
			sortBy: "name",
			wantFunc: func(t *testing.T, items []ServerListItem) {
				assert.Equal(t, "server-a", items[0].Name)
				assert.Equal(t, "server-b", items[1].Name)
				assert.Equal(t, "server-c", items[2].Name)
			},
		},
		{
			name:   "sort by status",
			sortBy: "status",
			wantFunc: func(t *testing.T, items []ServerListItem) {
				assert.Equal(t, "running", items[0].Status)
				assert.Equal(t, "running", items[1].Status)
				assert.Equal(t, "stopped", items[2].Status)
			},
		},
		{
			name:   "sort by port",
			sortBy: "port",
			wantFunc: func(t *testing.T, items []ServerListItem) {
				assert.Equal(t, 25565, items[0].Port)
				assert.Equal(t, 25566, items[1].Port)
				assert.Equal(t, 25567, items[2].Port)
			},
		},
		{
			name:   "sort by memory (running first)",
			sortBy: "memory",
			wantFunc: func(t *testing.T, items []ServerListItem) {
				assert.Equal(t, "running", items[0].Status)
				assert.Equal(t, "running", items[1].Status)
				assert.Equal(t, "stopped", items[2].Status)
			},
		},
		{
			name:   "sort by uptime (longest first)",
			sortBy: "uptime",
			wantFunc: func(t *testing.T, items []ServerListItem) {
				// server-c started 2 hours ago (earliest start = longest uptime)
				assert.Equal(t, "server-c", items[0].Name)
				// server-b started 1 hour ago
				assert.Equal(t, "server-b", items[1].Name)
				// server-a is stopped (no uptime)
				assert.Equal(t, "server-a", items[2].Name)
			},
		},
		{
			name:   "default sort (name)",
			sortBy: "invalid",
			wantFunc: func(t *testing.T, items []ServerListItem) {
				assert.Equal(t, "server-a", items[0].Name)
				assert.Equal(t, "server-b", items[1].Name)
				assert.Equal(t, "server-c", items[2].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying original
			itemsCopy := make([]ServerListItem, len(items))
			copy(itemsCopy, items)

			sortServers(itemsCopy, tt.sortBy)
			tt.wantFunc(t, itemsCopy)
		})
	}
}

func TestFormatMemoryDisplay(t *testing.T) {
	tests := []struct {
		name string
		item ServerListItem
		want string
	}{
		{
			name: "running server",
			item: ServerListItem{
				Status:      "running",
				MemoryTotal: "2G",
			},
			want: "-/2G",
		},
		{
			name: "stopped server",
			item: ServerListItem{
				Status:      "stopped",
				MemoryTotal: "2G",
			},
			want: "-",
		},
		{
			name: "created server",
			item: ServerListItem{
				Status:      "created",
				MemoryTotal: "4G",
			},
			want: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMemoryDisplay(tt.item)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOutputListJSON(t *testing.T) {
	items := []ServerListItem{
		{
			Name:        "server1",
			Status:      "running",
			Version:     "1.21.1",
			Port:        25565,
			MemoryTotal: "2G",
			Uptime:      "2h 30m",
		},
		{
			Name:        "server2",
			Status:      "stopped",
			Version:     "1.20.4",
			Port:        25566,
			MemoryTotal: "4G",
		},
	}

	var buf strings.Builder
	err := outputListJSON(&buf, items)
	require.NoError(t, err)

	// Parse JSON output
	var output ListOutput
	err = json.Unmarshal([]byte(buf.String()), &output)
	require.NoError(t, err)

	// Verify structure
	assert.Equal(t, "success", output.Status)
	assert.Contains(t, output.Data, "servers")
	assert.Contains(t, output.Data, "count")
	assert.Equal(t, float64(2), output.Data["count"])

	// Verify servers data
	servers := output.Data["servers"].([]interface{})
	assert.Len(t, servers, 2)

	// Verify first server
	server1 := servers[0].(map[string]interface{})
	assert.Equal(t, "server1", server1["name"])
	assert.Equal(t, "running", server1["status"])
	assert.Equal(t, "1.21.1", server1["version"])
	assert.Equal(t, float64(25565), server1["port"])
}

func TestOutputListTable(t *testing.T) {
	items := []ServerListItem{
		{
			Name:        "survival",
			Status:      "running",
			Version:     "1.21.1",
			Port:        25565,
			MemoryTotal: "2G",
			Uptime:      "2h 15m",
		},
		{
			Name:        "creative",
			Status:      "stopped",
			Version:     "1.20.4",
			Port:        25566,
			MemoryTotal: "1G",
		},
	}

	t.Run("with header", func(t *testing.T) {
		var buf strings.Builder
		err := outputListTable(&buf, items, false)
		require.NoError(t, err)

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have header + 2 rows
		assert.Len(t, lines, 3)

		// Verify header
		assert.Contains(t, lines[0], "NAME")
		assert.Contains(t, lines[0], "STATUS")
		assert.Contains(t, lines[0], "VERSION")
		assert.Contains(t, lines[0], "PORT")
		assert.Contains(t, lines[0], "MEMORY")
		assert.Contains(t, lines[0], "UPTIME")

		// Verify first row
		assert.Contains(t, lines[1], "survival")
		assert.Contains(t, lines[1], "running")
		assert.Contains(t, lines[1], "1.21.1")
		assert.Contains(t, lines[1], "25565")
		assert.Contains(t, lines[1], "2h 15m")

		// Verify second row
		assert.Contains(t, lines[2], "creative")
		assert.Contains(t, lines[2], "stopped")
		assert.Contains(t, lines[2], "1.20.4")
		assert.Contains(t, lines[2], "25566")
		assert.Contains(t, lines[2], "-")
	})

	t.Run("without header", func(t *testing.T) {
		var buf strings.Builder
		err := outputListTable(&buf, items, true)
		require.NoError(t, err)

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have only 2 rows (no header)
		assert.Len(t, lines, 2)

		// Should not contain header
		assert.NotContains(t, lines[0], "NAME")
		assert.Contains(t, lines[0], "survival")
	})
}

func TestOutputListError(t *testing.T) {
	t.Run("json mode", func(t *testing.T) {
		var buf strings.Builder
		err := outputListError(&buf, true, assert.AnError)

		// Should return the error
		assert.Error(t, err)

		// Should output JSON
		var output ListOutput
		parseErr := json.Unmarshal([]byte(buf.String()), &output)
		require.NoError(t, parseErr)

		assert.Equal(t, "error", output.Status)
		assert.NotEmpty(t, output.Error)
	})

	t.Run("text mode", func(t *testing.T) {
		var buf strings.Builder
		err := outputListError(&buf, false, assert.AnError)

		// Should return the error
		assert.Error(t, err)

		// Should not output anything in text mode
		assert.Empty(t, buf.String())
	})
}

func TestListFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    *ListFlags
		wantAll  bool
		wantSort string
	}{
		{
			name: "default flags",
			flags: &ListFlags{
				All:    false,
				Sort:   "name",
				Filter: "",
			},
			wantAll:  false,
			wantSort: "name",
		},
		{
			name: "all flag set",
			flags: &ListFlags{
				All:    true,
				Sort:   "status",
				Filter: "",
			},
			wantAll:  true,
			wantSort: "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantAll, tt.flags.All)
			assert.Equal(t, tt.wantSort, tt.flags.Sort)
		})
	}
}

func TestServerListItem(t *testing.T) {
	now := time.Now()
	item := ServerListItem{
		Name:        "test-server",
		Status:      "running",
		Version:     "1.21.1",
		Port:        25565,
		MemoryUsed:  "1.5G",
		MemoryTotal: "2G",
		Uptime:      "3h 45m",
		StartedAt:   now,
	}

	assert.Equal(t, "test-server", item.Name)
	assert.Equal(t, "running", item.Status)
	assert.Equal(t, "1.21.1", item.Version)
	assert.Equal(t, 25565, item.Port)
	assert.Equal(t, "1.5G", item.MemoryUsed)
	assert.Equal(t, "2G", item.MemoryTotal)
	assert.Equal(t, "3h 45m", item.Uptime)
	assert.Equal(t, now, item.StartedAt)
}

func TestListOutput(t *testing.T) {
	output := ListOutput{
		Status: "success",
		Data: map[string]interface{}{
			"servers": []ServerListItem{},
			"count":   0,
		},
		Message: "No servers found",
	}

	assert.Equal(t, "success", output.Status)
	assert.Equal(t, "No servers found", output.Message)
	assert.Empty(t, output.Error)
	assert.Contains(t, output.Data, "servers")
	assert.Contains(t, output.Data, "count")
}
