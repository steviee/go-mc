package servers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTopCommand(t *testing.T) {
	cmd := NewTopCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "top", cmd.Use)
	assert.Contains(t, cmd.Aliases, "dashboard")
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.RunE)
}

func TestTopCommand_Aliases(t *testing.T) {
	cmd := NewTopCommand()

	aliases := cmd.Aliases
	assert.Contains(t, aliases, "dashboard")
}

func TestTopCommand_Help(t *testing.T) {
	cmd := NewTopCommand()

	// Check that help text mentions keyboard shortcuts
	assert.Contains(t, cmd.Long, "Keyboard shortcuts")
	assert.Contains(t, cmd.Long, "Start selected server")
	assert.Contains(t, cmd.Long, "Stop selected server")
	assert.Contains(t, cmd.Long, "Quit dashboard")
}
