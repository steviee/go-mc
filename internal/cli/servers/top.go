package servers

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/tui"
)

// NewTopCommand creates the servers top subcommand
func NewTopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Interactive dashboard with real-time server monitoring",
		Long: `Launch an interactive TUI dashboard that displays all servers with real-time
status updates, resource usage, and uptime information.

The dashboard auto-refreshes every second and provides quick actions via
keyboard shortcuts.

Keyboard shortcuts:
  ↑/k         Move selection up
  ↓/j         Move selection down
  s           Start selected server
  x           Stop selected server
  r           Restart selected server
  l           Show logs (coming soon)
  d           Delete server (coming soon)
  q/Ctrl+C    Quit dashboard`,
		Example: `  # Launch the dashboard
  go-mc servers top

  # Alternative using alias
  go-mc servers dashboard`,
		Aliases: []string{"dashboard"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTop(cmd.Context())
		},
	}

	return cmd
}

// runTop executes the top command
func runTop(ctx context.Context) error {
	// Create container client
	containerClient, err := container.NewClient(ctx, container.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to create container client: %w", err)
	}
	defer func() { _ = containerClient.Close() }()

	// Create TUI model
	model := tui.NewModel(ctx, containerClient)

	// Start bubbletea program
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run dashboard: %w", err)
	}

	return nil
}
