package tui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/steviee/go-mc/internal/state"
)

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.quitting {
			return m, nil
		}
		// Auto-refresh server list
		return m, tea.Batch(
			tickCmd(),
			loadServersCmd(m.ctx, m.containerClient),
		)

	case serversLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.errorTime = time.Now()
			slog.Error("failed to load servers", "error", msg.err)
			return m, clearErrorCmd()
		}

		m.servers = msg.servers
		m.lastUpdate = time.Now()

		// Adjust selected index if needed
		if len(m.servers) == 0 {
			m.selectedIdx = 0
		} else if m.selectedIdx >= len(m.servers) {
			m.selectedIdx = len(m.servers) - 1
		}

		return m, nil

	case serverActionMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("%s failed for %s: %w", msg.action, msg.server, msg.err)
			m.errorTime = time.Now()
			slog.Error("server action failed", "action", msg.action, "server", msg.server, "error", msg.err)
			return m, clearErrorCmd()
		}

		// Success - reload servers immediately
		slog.Info("server action succeeded", "action", msg.action, "server", msg.server)
		return m, loadServersCmd(m.ctx, m.containerClient)

	case clearErrorMsg:
		// Only clear if error is older than 3 seconds
		if time.Since(m.errorTime) >= 3*time.Second {
			m.err = nil
		}
		return m, nil

	case errorMsg:
		m.err = msg.err
		m.errorTime = time.Now()
		return m, clearErrorCmd()
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	}

	// If no servers, ignore navigation and action keys
	if len(m.servers) == 0 {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil

	case "down", "j":
		if m.selectedIdx < len(m.servers)-1 {
			m.selectedIdx++
		}
		return m, nil

	case "s":
		// Start selected server
		if m.selectedIdx < len(m.servers) {
			server := m.servers[m.selectedIdx]
			if server.Status != "running" {
				containerID, err := getServerContainerID(m.ctx, server.Name)
				if err != nil {
					m.err = fmt.Errorf("failed to get container ID: %w", err)
					m.errorTime = time.Now()
					return m, clearErrorCmd()
				}
				return m, startServerCmd(m.ctx, m.containerClient, server.Name, containerID)
			}
		}
		return m, nil

	case "x":
		// Stop selected server
		if m.selectedIdx < len(m.servers) {
			server := m.servers[m.selectedIdx]
			if server.Status == "running" {
				containerID, err := getServerContainerID(m.ctx, server.Name)
				if err != nil {
					m.err = fmt.Errorf("failed to get container ID: %w", err)
					m.errorTime = time.Now()
					return m, clearErrorCmd()
				}
				return m, stopServerCmd(m.ctx, m.containerClient, server.Name, containerID)
			}
		}
		return m, nil

	case "r":
		// Restart selected server
		if m.selectedIdx < len(m.servers) {
			server := m.servers[m.selectedIdx]
			if server.Status == "running" {
				containerID, err := getServerContainerID(m.ctx, server.Name)
				if err != nil {
					m.err = fmt.Errorf("failed to get container ID: %w", err)
					m.errorTime = time.Now()
					return m, clearErrorCmd()
				}
				return m, restartServerCmd(m.ctx, m.containerClient, server.Name, containerID)
			}
		}
		return m, nil

	case "l":
		// Show logs - not implemented yet
		m.err = fmt.Errorf("logs view not implemented yet")
		m.errorTime = time.Now()
		return m, clearErrorCmd()

	case "d":
		// Delete server - not implemented yet (requires confirmation)
		m.err = fmt.Errorf("delete not implemented yet")
		m.errorTime = time.Now()
		return m, clearErrorCmd()
	}

	return m, nil
}

// getServerContainerID retrieves the container ID for a server
func getServerContainerID(ctx context.Context, name string) (string, error) {
	serverState, err := state.LoadServerState(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to load server state: %w", err)
	}

	if serverState.ContainerID == "" {
		return "", fmt.Errorf("server has no container")
	}

	return serverState.ContainerID, nil
}
