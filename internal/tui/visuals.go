package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Sparkline characters from low to high
var sparklineChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// renderSparkline creates a sparkline graph from a slice of values
// Returns a string of Unicode sparkline characters representing the trend
func renderSparkline(values []float64, width int) string {
	if len(values) == 0 {
		return strings.Repeat("▁", width)
	}

	// If we have fewer values than width, pad with zeros at the start
	if len(values) < width {
		padding := make([]float64, width-len(values))
		values = append(padding, values...)
	}

	// If we have more values than width, take the last 'width' values
	if len(values) > width {
		values = values[len(values)-width:]
	}

	// Find min and max for normalization
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Avoid division by zero
	rangeVal := max - min
	if rangeVal == 0 {
		rangeVal = 1
	}

	// Generate sparkline
	var result strings.Builder
	for _, v := range values {
		// Normalize value to 0-1 range
		normalized := (v - min) / rangeVal

		// Map to sparkline character index (0-7)
		index := int(normalized * float64(len(sparklineChars)-1))
		if index < 0 {
			index = 0
		}
		if index >= len(sparklineChars) {
			index = len(sparklineChars) - 1
		}

		result.WriteRune(sparklineChars[index])
	}

	return result.String()
}

// renderProgressBar creates a progress bar with color coding
// value: percentage (0-100)
// width: total width of the bar
func renderProgressBar(value float64, width int) string {
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}

	// Calculate filled and empty portions
	filledWidth := int(math.Round(value / 100.0 * float64(width)))
	emptyWidth := width - filledWidth

	// Choose color based on value
	color := getUsageColor(value)

	// Build the bar
	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", emptyWidth)

	bar := lipgloss.NewStyle().
		Foreground(color).
		Render(filled + empty)

	return bar
}

// renderProgressBarWithPercentage renders a progress bar with percentage text
func renderProgressBarWithPercentage(value float64, barWidth int) string {
	bar := renderProgressBar(value, barWidth)
	percentage := fmt.Sprintf("% 3.0f%%", value)
	return fmt.Sprintf("%s %s", bar, percentage)
}

// getUsageColor returns a color based on usage percentage
func getUsageColor(percentage float64) lipgloss.Color {
	switch {
	case percentage >= 90:
		return lipgloss.Color("#FF0000") // Red
	case percentage >= 70:
		return lipgloss.Color("#FFA500") // Orange
	case percentage >= 50:
		return lipgloss.Color("#FFFF00") // Yellow
	case percentage >= 30:
		return lipgloss.Color("#90EE90") // Light Green
	default:
		return lipgloss.Color("#00FF00") // Green
	}
}

// renderSparklineWithColor renders a sparkline with color coding based on recent value
func renderSparklineWithColor(values []float64, width int) string {
	if len(values) == 0 {
		return renderSparkline(values, width)
	}

	// Get the last value for color coding
	lastValue := values[len(values)-1]
	color := getUsageColor(lastValue)

	sparkline := renderSparkline(values, width)

	return lipgloss.NewStyle().
		Foreground(color).
		Render(sparkline)
}

// renderBox creates a box with the given content using box-drawing characters
func renderBox(title string, content string, width int) string {
	if width < 4 {
		width = 4
	}

	var b strings.Builder

	// Top border
	b.WriteString("╭─")
	if title != "" {
		b.WriteString(" ")
		b.WriteString(title)
		b.WriteString(" ")
		remaining := width - len(title) - 6 // 6 for "╭─  ─╮"
		if remaining > 0 {
			b.WriteString(strings.Repeat("─", remaining))
		}
	} else {
		b.WriteString(strings.Repeat("─", width-4)) // 4 for "╭──╮"
	}
	b.WriteString("─╮\n")

	// Content
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		b.WriteString("│ ")
		b.WriteString(line)
		// Pad to width
		padding := width - len(line) - 4 // 4 for "│  │"
		if padding > 0 {
			b.WriteString(strings.Repeat(" ", padding))
		}
		b.WriteString(" │\n")
	}

	// Bottom border
	b.WriteString("╰")
	b.WriteString(strings.Repeat("─", width-2))
	b.WriteString("╯")

	return b.String()
}

// renderSeparator creates a horizontal separator
func renderSeparator(width int, title string) string {
	if title == "" {
		return strings.Repeat("─", width)
	}

	var b strings.Builder
	b.WriteString("─ ")
	b.WriteString(title)
	b.WriteString(" ")
	remaining := width - len(title) - 4 // 4 for "─  ─"
	if remaining > 0 {
		b.WriteString(strings.Repeat("─", remaining))
	}
	return b.String()
}
