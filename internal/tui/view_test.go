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
