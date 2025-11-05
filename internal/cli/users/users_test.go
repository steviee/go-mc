package users

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()

	assert.Equal(t, "users", cmd.Use)
	assert.Equal(t, "Manage Minecraft users and permissions", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.Contains(t, cmd.Aliases, "user")
}

func TestNewCommand_HasHelp(t *testing.T) {
	cmd := NewCommand()

	assert.NotEmpty(t, cmd.Short, "command should have short description")
	assert.NotEmpty(t, cmd.Long, "command should have long description")
	assert.NotEmpty(t, cmd.Example, "command should have examples")
}
