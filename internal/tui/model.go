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
}

// NewModel creates a new TUI model
func NewModel(client container.Client) *Model {
	return &Model{
		servers:         []ServerInfo{},
		selectedIdx:     0,
		lastUpdate:      time.Now(),
		loading:         true,
		containerClient: client,
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
