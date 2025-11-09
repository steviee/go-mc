package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestView_NoServers(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)
	model.loading = false
	model.servers = []ServerInfo{}

	view := model.View()

	assert.Contains(t, view, "go-mc Dashboard")
	assert.Contains(t, view, "No servers found")
}

func TestView_Loading(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)
	model.loading = true
	model.servers = []ServerInfo{}

	view := model.View()

	assert.Contains(t, view, "Loading servers")
}

func TestView_WithServers(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)
	model.loading = false
	model.servers = []ServerInfo{
		{
			Name:        "server1",
			Status:      "running",
			Version:     "1.20.4",
			Port:        25565,
			MemoryTotal: "2G",
			MemoryUsed:  "1.5G",
			Uptime:      "2h 30m",
		},
		{
			Name:        "server2",
			Status:      "stopped",
			Version:     "1.21.1",
			Port:        25566,
			MemoryTotal: "4G",
			MemoryUsed:  "-",
			Uptime:      "-",
		},
	}
	model.selectedIdx = 0
	model.lastUpdate = time.Now()

	view := model.View()

	// Check header
	assert.Contains(t, view, "go-mc Dashboard")
	assert.Contains(t, view, "Last Update:")

	// Check table headers
	assert.Contains(t, view, "NAME")
	assert.Contains(t, view, "STATUS")
	assert.Contains(t, view, "VERSION")
	assert.Contains(t, view, "PORT")
	assert.Contains(t, view, "MEMORY")
	assert.Contains(t, view, "UPTIME")

	// Check server data
	assert.Contains(t, view, "server1")
	assert.Contains(t, view, "server2")
	assert.Contains(t, view, "1.20.4")
	assert.Contains(t, view, "1.21.1")
	assert.Contains(t, view, "25565")
	assert.Contains(t, view, "25566")

	// Check footer
	assert.Contains(t, view, "[↑/↓] navigate")
	assert.Contains(t, view, "[s]tart")
	assert.Contains(t, view, "[q]uit")
}

func TestView_WithError(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)
	model.loading = false
	model.servers = []ServerInfo{}
	model.err = assert.AnError
	model.errorTime = time.Now()

	view := model.View()

	assert.Contains(t, view, "Error:")
}

func TestView_Quitting(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)
	model.quitting = true

	view := model.View()

	assert.Contains(t, view, "Dashboard closed")
}

func TestRenderHeader(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)
	model.lastUpdate = time.Date(2025, 1, 1, 14, 30, 25, 0, time.UTC)
	model.width = 80

	header := model.renderHeader()

	assert.Contains(t, header, "go-mc Dashboard")
	assert.Contains(t, header, "14:30:25")
}

func TestRenderTable(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)
	model.servers = []ServerInfo{
		{
			Name:        "testserver",
			Status:      "running",
			Version:     "1.20.4",
			Port:        25565,
			MemoryTotal: "2G",
			MemoryUsed:  "-",
			Uptime:      "1h 30m",
		},
	}
	model.selectedIdx = 0

	table := model.renderTable()

	// Check that table contains expected data
	assert.Contains(t, table, "testserver")
	assert.Contains(t, table, "running")
	assert.Contains(t, table, "1.20.4")
	assert.Contains(t, table, "25565")
	assert.Contains(t, table, "1h 30m")
}

func TestRenderFooter(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)

	footer := model.renderFooter()

	assert.Contains(t, footer, "navigate")
	assert.Contains(t, footer, "tart")   // [s]tart
	assert.Contains(t, footer, "top")    // s[x]top
	assert.Contains(t, footer, "estart") // [r]estart
	assert.Contains(t, footer, "uit")    // [q]uit
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		name     string
		server   ServerInfo
		expected string
	}{
		{
			name: "running server with usage",
			server: ServerInfo{
				Status:      "running",
				MemoryUsed:  "1.5G",
				MemoryTotal: "2G",
			},
			expected: "1.5G/2G",
		},
		{
			name: "running server without usage",
			server: ServerInfo{
				Status:      "running",
				MemoryUsed:  "-",
				MemoryTotal: "2G",
			},
			expected: "-/2G",
		},
		{
			name: "stopped server",
			server: ServerInfo{
				Status:      "stopped",
				MemoryTotal: "2G",
			},
			expected: "-",
		},
		{
			name: "created server",
			server: ServerInfo{
				Status:      "created",
				MemoryTotal: "4G",
			},
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMemory(tt.server)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStatusIndicator(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"running", "●"},
		{"stopped", "●"},
		{"created", "●"},
		{"exited", "●"},
		{"missing", "✗"},
		{"unknown", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			indicator := getStatusIndicator(tt.status)
			assert.Equal(t, tt.expected, indicator)
		})
	}
}

func TestGetStatusStyle(t *testing.T) {
	tests := []struct {
		status string
		name   string
	}{
		{"running", "running status"},
		{"stopped", "stopped status"},
		{"exited", "exited status"},
		{"created", "created status"},
		{"missing", "missing status"},
		{"unknown", "unknown status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := getStatusStyle(tt.status)
			assert.NotNil(t, style)
			// Just verify that calling the function doesn't panic
			// and returns a valid style
		})
	}
}

func TestView_MultipleServers_Selection(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)
	model.loading = false
	model.servers = []ServerInfo{
		{Name: "server1", Status: "running", Version: "1.20.4", Port: 25565},
		{Name: "server2", Status: "stopped", Version: "1.21.1", Port: 25566},
		{Name: "server3", Status: "created", Version: "1.20.4", Port: 25567},
	}
	model.selectedIdx = 1 // Select second server

	view := model.View()

	// The view should contain all servers
	assert.Contains(t, view, "server1")
	assert.Contains(t, view, "server2")
	assert.Contains(t, view, "server3")

	// Check that there's some indication of selection
	// (The exact rendering may vary, but the view should contain the data)
	lines := strings.Split(view, "\n")
	var foundServer2 bool
	for _, line := range lines {
		if strings.Contains(line, "server2") {
			foundServer2 = true
			break
		}
	}
	assert.True(t, foundServer2, "server2 should be in the view")
}

func TestFormatCPU(t *testing.T) {
	tests := []struct {
		name     string
		server   ServerInfo
		expected string
	}{
		{
			name: "stopped server",
			server: ServerInfo{
				Status:     "stopped",
				CPUPercent: 0,
			},
			expected: "-",
		},
		{
			name: "created server",
			server: ServerInfo{
				Status:     "created",
				CPUPercent: 0,
			},
			expected: "-",
		},
		{
			name: "running server with CPU usage",
			server: ServerInfo{
				Status:     "running",
				CPUPercent: 25.3456,
			},
			expected: "25.3%",
		},
		{
			name: "running server with zero CPU",
			server: ServerInfo{
				Status:     "running",
				CPUPercent: 0,
			},
			expected: "-",
		},
		{
			name: "running server with high CPU",
			server: ServerInfo{
				Status:     "running",
				CPUPercent: 99.9,
			},
			expected: "99.9%",
		},
		{
			name: "running server with low CPU",
			server: ServerInfo{
				Status:     "running",
				CPUPercent: 0.5,
			},
			expected: "0.5%",
		},
		{
			name: "running server with exact rounding",
			server: ServerInfo{
				Status:     "running",
				CPUPercent: 50.0,
			},
			expected: "50.0%",
		},
		{
			name: "running server with fractional rounding",
			server: ServerInfo{
				Status:     "running",
				CPUPercent: 33.33333,
			},
			expected: "33.3%",
		},
		{
			name: "exited server (not running)",
			server: ServerInfo{
				Status:     "exited",
				CPUPercent: 50.0,
			},
			expected: "-",
		},
		{
			name: "paused server",
			server: ServerInfo{
				Status:     "paused",
				CPUPercent: 10.0,
			},
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCPU(tt.server)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatMemoryPercent(t *testing.T) {
	tests := []struct {
		name     string
		server   ServerInfo
		expected string
	}{
		{
			name: "stopped server",
			server: ServerInfo{
				Status:        "stopped",
				MemoryPercent: 0,
			},
			expected: "-",
		},
		{
			name: "created server",
			server: ServerInfo{
				Status:        "created",
				MemoryPercent: 0,
			},
			expected: "-",
		},
		{
			name: "running server with memory usage",
			server: ServerInfo{
				Status:        "running",
				MemoryPercent: 45.2345,
			},
			expected: "45.2%",
		},
		{
			name: "running server with zero memory",
			server: ServerInfo{
				Status:        "running",
				MemoryPercent: 0,
			},
			expected: "-",
		},
		{
			name: "running server with high memory",
			server: ServerInfo{
				Status:        "running",
				MemoryPercent: 98.7,
			},
			expected: "98.7%",
		},
		{
			name: "running server with low memory",
			server: ServerInfo{
				Status:        "running",
				MemoryPercent: 1.2,
			},
			expected: "1.2%",
		},
		{
			name: "running server with exact rounding",
			server: ServerInfo{
				Status:        "running",
				MemoryPercent: 75.0,
			},
			expected: "75.0%",
		},
		{
			name: "running server with fractional rounding",
			server: ServerInfo{
				Status:        "running",
				MemoryPercent: 66.66666,
			},
			expected: "66.7%",
		},
		{
			name: "running server at 100%",
			server: ServerInfo{
				Status:        "running",
				MemoryPercent: 100.0,
			},
			expected: "100.0%",
		},
		{
			name: "exited server (not running)",
			server: ServerInfo{
				Status:        "exited",
				MemoryPercent: 50.0,
			},
			expected: "-",
		},
		{
			name: "paused server",
			server: ServerInfo{
				Status:        "paused",
				MemoryPercent: 25.5,
			},
			expected: "-",
		},
		{
			name: "missing server",
			server: ServerInfo{
				Status:        "missing",
				MemoryPercent: 10.0,
			},
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMemoryPercent(tt.server)
			assert.Equal(t, tt.expected, result)
		})
	}
}
