module github.com/steviee/go-mc

go 1.21

require (
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.18.2
	gopkg.in/yaml.v3 v3.0.1
)

// Note: Dependencies will be added as features are implemented
// - Podman client: github.com/containers/podman/v4
// - TUI: github.com/charmbracelet/bubbletea, lipgloss
// - Testing: github.com/stretchr/testify
// - RCON: github.com/gorcon/rcon-cli
