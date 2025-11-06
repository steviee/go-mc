package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Header styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#00ADD8")).
			Padding(0, 1)

	// Table header styles
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Underline(true).
				Foreground(lipgloss.Color("#FFFFFF"))

	// Selected row style
	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#FFA500")).
				Foreground(lipgloss.Color("#000000"))

	// Status color styles
	statusRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00"))

	statusStoppedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000"))

	statusCreatedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFF00"))

	statusMissingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000"))

	// Footer style
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080"))

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)
)

// getStatusStyle returns the style for a given status
func getStatusStyle(status string) lipgloss.Style {
	switch status {
	case "running":
		return statusRunningStyle
	case "stopped", "exited":
		return statusStoppedStyle
	case "created":
		return statusCreatedStyle
	case "missing":
		return statusMissingStyle
	default:
		return lipgloss.NewStyle()
	}
}

// getStatusIndicator returns the status indicator symbol
func getStatusIndicator(status string) string {
	switch status {
	case "running":
		return "●"
	case "stopped", "exited":
		return "●"
	case "created":
		return "●"
	case "missing":
		return "✗"
	default:
		return "?"
	}
}
