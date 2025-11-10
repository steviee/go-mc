package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/steviee/go-mc/internal/container"
)

// PortInfo represents a network port used by the server
type PortInfo struct {
	Number   int    // Port number
	Protocol string // "tcp" or "udp"
	Service  string // "Game", "RCON", "Voice Chat", etc.
	Source   string // "config", "detected", "mod"
}

// ModInfo represents basic mod information for display
type ModInfo struct {
	Name    string
	Slug    string
	Version string
}

// MetricsHistory holds historical metrics for sparkline graphs
type MetricsHistory struct {
	CPUHistory    []float64 // Last N CPU percentage values
	MemoryHistory []float64 // Last N memory percentage values
	MaxDataPoints int       // Maximum number of data points to keep
}

// NewMetricsHistory creates a new metrics history with the specified capacity
func NewMetricsHistory(maxPoints int) *MetricsHistory {
	return &MetricsHistory{
		CPUHistory:    make([]float64, 0, maxPoints),
		MemoryHistory: make([]float64, 0, maxPoints),
		MaxDataPoints: maxPoints,
	}
}

// AddCPU adds a CPU usage data point
func (m *MetricsHistory) AddCPU(value float64) {
	m.CPUHistory = append(m.CPUHistory, value)
	if len(m.CPUHistory) > m.MaxDataPoints {
		m.CPUHistory = m.CPUHistory[1:]
	}
}

// AddMemory adds a memory usage data point
func (m *MetricsHistory) AddMemory(value float64) {
	m.MemoryHistory = append(m.MemoryHistory, value)
	if len(m.MemoryHistory) > m.MaxDataPoints {
		m.MemoryHistory = m.MemoryHistory[1:]
	}
}

// ServerInfo represents a server in the TUI
type ServerInfo struct {
	Name        string
	Status      string
	Version     string
	Port        int
	MemoryUsed  string
	MemoryTotal string
	Uptime      string
	StartedAt   time.Time
	// Fabric and RCON information
	FabricVersion string // Fabric loader version
	RCONPort      int    // RCON port
	// Resource usage metrics
	CPUPercent       float64 // CPU usage percentage (0-100)
	MemoryUsedBytes  int64   // Actual memory used in bytes
	MemoryLimitBytes int64   // Memory limit in bytes
	MemoryPercent    float64 // Memory usage percentage (0-100)
	// Historical metrics for graphs
	Metrics *MetricsHistory // Historical CPU/Memory metrics
	// Player information (for future use)
	PlayerCount int // Current player count
	PlayerMax   int // Maximum players
	// Mod and port information
	InstalledMods []ModInfo  // List of installed mods
	Ports         []PortInfo // List of network ports
}

// Model is the bubbletea model for the TUI dashboard
type Model struct {
	servers         []ServerInfo
	selectedIdx     int
	lastUpdate      time.Time
	err             error
	errorTime       time.Time
	loading         bool
	width           int
	height          int
	containerClient container.Client
	quitting        bool
	// Metrics history map (keyed by server name)
	metricsHistory map[string]*MetricsHistory
}

// NewModel creates a new TUI model
func NewModel(client container.Client) *Model {
	return &Model{
		servers:         []ServerInfo{},
		selectedIdx:     0,
		lastUpdate:      time.Now(),
		loading:         true,
		containerClient: client,
		metricsHistory:  make(map[string]*MetricsHistory),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		loadServersCmd(context.Background(), m.containerClient),
	)
}

// tickCmd returns a command that sends a tick message every second
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// loadServersCmd returns a command that loads the server list
func loadServersCmd(ctx context.Context, client container.Client) tea.Cmd {
	return func() tea.Msg {
		servers, err := loadServers(ctx, client)
		return serversLoadedMsg{
			servers: servers,
			err:     err,
		}
	}
}

// startServerCmd returns a command that starts a server
func startServerCmd(ctx context.Context, client container.Client, name string, containerID string) tea.Cmd {
	return func() tea.Msg {
		var err error
		if containerID != "" {
			err = client.StartContainer(ctx, containerID)
		} else {
			err = fmt.Errorf("server has no container ID")
		}

		return serverActionMsg{
			action: "start",
			server: name,
			err:    err,
		}
	}
}

// stopServerCmd returns a command that stops a server
func stopServerCmd(ctx context.Context, client container.Client, name string, containerID string) tea.Cmd {
	return func() tea.Msg {
		var err error
		if containerID != "" {
			timeout := 30 * time.Second
			err = client.StopContainer(ctx, containerID, &timeout)
		} else {
			err = fmt.Errorf("server has no container ID")
		}

		return serverActionMsg{
			action: "stop",
			server: name,
			err:    err,
		}
	}
}

// restartServerCmd returns a command that restarts a server
func restartServerCmd(ctx context.Context, client container.Client, name string, containerID string) tea.Cmd {
	return func() tea.Msg {
		var err error
		if containerID != "" {
			timeout := 30 * time.Second
			err = client.RestartContainer(ctx, containerID, &timeout)
		} else {
			err = fmt.Errorf("server has no container ID")
		}

		return serverActionMsg{
			action: "restart",
			server: name,
			err:    err,
		}
	}
}

// clearErrorCmd returns a command that clears the error message after a delay
func clearErrorCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}
