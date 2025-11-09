package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View renders the TUI
func (m Model) View() string {
	if m.quitting {
		return "Dashboard closed.\n"
	}

	var b strings.Builder

	// Render header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Render table
	if m.loading && len(m.servers) == 0 {
		b.WriteString("\nLoading servers...\n")
	} else if len(m.servers) == 0 {
		b.WriteString("\nNo servers found. Create one with 'go-mc servers create <name>'\n")
	} else {
		b.WriteString(m.renderTable())
	}

	// Render footer
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	// Render error message if any
	if m.err != nil && time.Since(m.errorTime) < 3*time.Second {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s", m.err)))
	}

	return b.String()
}

// renderHeader renders the dashboard header
func (m Model) renderHeader() string {
	title := "go-mc Dashboard"
	lastUpdate := fmt.Sprintf("Last Update: %s", m.lastUpdate.Format("15:04:05"))

	// Calculate spacing to align title and update time
	totalWidth := 80
	if m.width > 0 {
		totalWidth = m.width
	}

	titleLen := len(title)
	updateLen := len(lastUpdate)
	spacing := totalWidth - titleLen - updateLen - 4 // 4 for padding

	if spacing < 1 {
		spacing = 1
	}

	// Build header with box drawing
	var b strings.Builder
	b.WriteString("╭")
	b.WriteString(strings.Repeat("─", totalWidth-2))
	b.WriteString("╮\n")

	headerText := fmt.Sprintf(" %s%s%s ", title, strings.Repeat(" ", spacing), lastUpdate)
	b.WriteString("│")
	b.WriteString(headerStyle.Render(headerText))
	b.WriteString("│\n")

	b.WriteString("╰")
	b.WriteString(strings.Repeat("─", totalWidth-2))
	b.WriteString("╯")

	return b.String()
}

// renderTable renders the server list table
func (m Model) renderTable() string {
	var b strings.Builder

	// Calculate column widths
	nameWidth := len("NAME")
	statusWidth := len("STATUS")
	versionWidth := len("VERSION")
	portWidth := len("PORT")
	rconWidth := len("RCON")
	cpuWidth := len("CPU%")
	memPercentWidth := len("MEM%")
	modsWidth := len("MODS")
	memoryWidth := len("MEMORY")
	uptimeWidth := len("UPTIME")

	for _, server := range m.servers {
		if len(server.Name) > nameWidth {
			nameWidth = len(server.Name)
		}
		if len(server.Status)+2 > statusWidth { // +2 for indicator
			statusWidth = len(server.Status) + 2
		}
		if len(server.Version) > versionWidth {
			versionWidth = len(server.Version)
		}
		portStr := fmt.Sprintf("%d", server.Port)
		if len(portStr) > portWidth {
			portWidth = len(portStr)
		}

		rconStr := fmt.Sprintf("%d", server.RCONPort)
		if len(rconStr) > rconWidth {
			rconWidth = len(rconStr)
		}

		cpuStr := formatCPU(server)
		if len(cpuStr) > cpuWidth {
			cpuWidth = len(cpuStr)
		}

		memPctStr := formatMemoryPercent(server)
		if len(memPctStr) > memPercentWidth {
			memPercentWidth = len(memPctStr)
		}

		modsStr := fmt.Sprintf("%d", len(server.InstalledMods))
		if len(modsStr) > modsWidth {
			modsWidth = len(modsStr)
		}

		memStr := formatMemory(server)
		if len(memStr) > memoryWidth {
			memoryWidth = len(memStr)
		}
		if len(server.Uptime) > uptimeWidth {
			uptimeWidth = len(server.Uptime)
		}
	}

	// Render header row
	headerRow := fmt.Sprintf("%-*s  %-*s  %-*s  %*s  %*s  %*s  %*s  %*s  %*s  %*s",
		nameWidth, "NAME",
		statusWidth, "STATUS",
		versionWidth, "VERSION",
		portWidth, "PORT",
		rconWidth, "RCON",
		cpuWidth, "CPU%",
		memPercentWidth, "MEM%",
		modsWidth, "MODS",
		memoryWidth, "MEMORY",
		uptimeWidth, "UPTIME",
	)
	b.WriteString(tableHeaderStyle.Render(headerRow))
	b.WriteString("\n")

	// Render server rows
	for i, server := range m.servers {
		uptime := server.Uptime
		if uptime == "" {
			uptime = "-"
		}

		// Format status with indicator
		indicator := getStatusIndicator(server.Status)
		statusStyle := getStatusStyle(server.Status)
		statusText := fmt.Sprintf("%s %s", indicator, server.Status)

		// Apply selection style if this is the selected row
		var row string
		if i == m.selectedIdx {
			// For selected row, render with selection background
			// but preserve status color for the status field
			nameCol := fmt.Sprintf("%-*s", nameWidth, server.Name)
			statusCol := fmt.Sprintf("%-*s", statusWidth, statusText)
			versionCol := fmt.Sprintf("%-*s", versionWidth, server.Version)
			portCol := fmt.Sprintf("%*d", portWidth, server.Port)
			rconCol := fmt.Sprintf("%*d", rconWidth, server.RCONPort)
			cpuCol := fmt.Sprintf("%*s", cpuWidth, formatCPU(server))
			memPctCol := fmt.Sprintf("%*s", memPercentWidth, formatMemoryPercent(server))
			modsCol := fmt.Sprintf("%*d", modsWidth, len(server.InstalledMods))
			memCol := fmt.Sprintf("%*s", memoryWidth, formatMemory(server))
			uptimeCol := fmt.Sprintf("%*s", uptimeWidth, uptime)

			// Apply selected style to name
			row = selectedRowStyle.Render("> "+nameCol) + "  "

			// Apply status color to status column (with selection hint)
			row += statusStyle.Background(lipgloss.Color("#FFA500")).Foreground(lipgloss.Color("#000000")).Render(statusCol) + "  "

			// Apply selected style to remaining columns
			row += selectedRowStyle.Render(versionCol) + "  "
			row += selectedRowStyle.Render(portCol) + "  "
			row += selectedRowStyle.Render(rconCol) + "  "
			row += selectedRowStyle.Render(cpuCol) + "  "
			row += selectedRowStyle.Render(memPctCol) + "  "
			row += selectedRowStyle.Render(modsCol) + "  "
			row += selectedRowStyle.Render(memCol) + "  "
			row += selectedRowStyle.Render(uptimeCol)

			b.WriteString(row)
		} else {
			// For non-selected rows, render normally with status color
			nameCol := fmt.Sprintf("  %-*s", nameWidth, server.Name)
			statusCol := fmt.Sprintf("%-*s", statusWidth, statusText)
			versionCol := fmt.Sprintf("%-*s", versionWidth, server.Version)
			portCol := fmt.Sprintf("%*d", portWidth, server.Port)
			rconCol := fmt.Sprintf("%*d", rconWidth, server.RCONPort)
			cpuCol := fmt.Sprintf("%*s", cpuWidth, formatCPU(server))
			memPctCol := fmt.Sprintf("%*s", memPercentWidth, formatMemoryPercent(server))
			modsCol := fmt.Sprintf("%*d", modsWidth, len(server.InstalledMods))
			memCol := fmt.Sprintf("%*s", memoryWidth, formatMemory(server))
			uptimeCol := fmt.Sprintf("%*s", uptimeWidth, uptime)

			row = nameCol + "  " +
				statusStyle.Render(statusCol) + "  " +
				versionCol + "  " +
				portCol + "  " +
				rconCol + "  " +
				cpuCol + "  " +
				memPctCol + "  " +
				modsCol + "  " +
				memCol + "  " +
				uptimeCol

			b.WriteString(row)
		}

		b.WriteString("\n")
	}

	return b.String()
}

// renderFooter renders the dashboard footer with action help
func (m Model) renderFooter() string {
	actions := "[↑/↓] navigate  [s]tart  [x]top  [r]estart  [l]ogs  [d]elete  [q]uit"
	return footerStyle.Render(actions)
}

// formatMemory formats memory display for a server
func formatMemory(server ServerInfo) string {
	if server.Status == "running" {
		// For running containers, show usage/total
		if server.MemoryUsed != "" && server.MemoryUsed != "-" {
			return fmt.Sprintf("%s/%s", server.MemoryUsed, server.MemoryTotal)
		}
		return fmt.Sprintf("-/%s", server.MemoryTotal)
	}
	// For non-running containers, just show dash
	return "-"
}

// formatCPU formats CPU usage percentage
func formatCPU(server ServerInfo) string {
	if server.Status != "running" || server.CPUPercent == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", server.CPUPercent)
}

// formatMemoryPercent formats memory usage percentage
func formatMemoryPercent(server ServerInfo) string {
	if server.Status != "running" || server.MemoryPercent == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", server.MemoryPercent)
}

// formatCPUSparkline formats CPU usage as a sparkline
func formatCPUSparkline(server ServerInfo, width int) string {
	if server.Status != "running" || server.Metrics == nil || len(server.Metrics.CPUHistory) == 0 {
		return strings.Repeat("▁", width)
	}
	return renderSparklineWithColor(server.Metrics.CPUHistory, width)
}

// formatMemSparkline formats memory usage as a sparkline
func formatMemSparkline(server ServerInfo, width int) string {
	if server.Status != "running" || server.Metrics == nil || len(server.Metrics.MemoryHistory) == 0 {
		return strings.Repeat("▁", width)
	}
	return renderSparklineWithColor(server.Metrics.MemoryHistory, width)
}

// formatCPUBar formats CPU usage as a progress bar
func formatCPUBar(server ServerInfo) string {
	if server.Status != "running" {
		return " - "
	}
	return renderProgressBarWithPercentage(server.CPUPercent, 6)
}

// formatMemBar formats memory usage as a progress bar
func formatMemBar(server ServerInfo) string {
	if server.Status != "running" {
		return " - "
	}
	return renderProgressBarWithPercentage(server.MemoryPercent, 6)
}
