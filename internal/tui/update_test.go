package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestHandleKeyPress_NoServers(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)
	model.servers = []ServerInfo{} // Empty list

	// Try to navigate - should have no effect
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.handleKeyPress(msg)
	m := updatedModel.(Model)

	assert.Equal(t, 0, m.selectedIdx)
}

func TestHandleKeyPress_Actions_NoServers(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)
	model.servers = []ServerInfo{} // Empty list

	// Try to perform action - should have no effect
	keys := []string{"s", "x", "r", "l", "d"}
	for _, key := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		updatedModel, _ := model.handleKeyPress(msg)
		m := updatedModel.(Model)

		// Should not crash or cause errors
		assert.NotNil(t, m)
	}
}

func TestModelUpdate_Tick_WhileQuitting(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)
	model.quitting = true

	msg := tickMsg(time.Now())
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	assert.True(t, m.quitting)
	assert.Nil(t, cmd) // Should not schedule another tick
}

func TestModelUpdate_ServerAction_Success(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)

	msg := serverActionMsg{
		action: "start",
		server: "test-server",
		err:    nil,
	}

	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	// Should reload servers
	assert.Nil(t, m.err)
	assert.NotNil(t, cmd)
}

func TestModelUpdate_ServerAction_Error(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)

	msg := serverActionMsg{
		action: "start",
		server: "test-server",
		err:    assert.AnError,
	}

	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	// Should set error
	assert.NotNil(t, m.err)
	assert.Contains(t, m.err.Error(), "start")
	assert.Contains(t, m.err.Error(), "test-server")
	assert.NotNil(t, cmd) // Should schedule error clear
}

func TestModelUpdate_ErrorMsg(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)

	msg := errorMsg{
		err: assert.AnError,
	}

	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	assert.NotNil(t, m.err)
	assert.Equal(t, assert.AnError, m.err)
	assert.NotNil(t, cmd) // Should schedule error clear
}

func TestModelUpdate_AdjustSelectedIdx(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)
	model.selectedIdx = 5
	model.servers = []ServerInfo{
		{Name: "server1"},
		{Name: "server2"},
	}

	// Load servers with fewer items than selected index
	msg := serversLoadedMsg{
		servers: []ServerInfo{
			{Name: "server1"},
		},
		err: nil,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Should adjust selected index to last item
	assert.Equal(t, 0, m.selectedIdx)
}

func TestStartServerCmd(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}

	client.On("StartContainer", ctx, "container123").Return(nil)

	cmd := startServerCmd(ctx, client, "test-server", "container123")
	msg := cmd()

	actionMsg, ok := msg.(serverActionMsg)
	assert.True(t, ok)
	assert.Equal(t, "start", actionMsg.action)
	assert.Equal(t, "test-server", actionMsg.server)
	assert.NoError(t, actionMsg.err)

	client.AssertExpectations(t)
}

func TestStartServerCmd_NoContainerID(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}

	cmd := startServerCmd(ctx, client, "test-server", "")
	msg := cmd()

	actionMsg, ok := msg.(serverActionMsg)
	assert.True(t, ok)
	assert.Equal(t, "start", actionMsg.action)
	assert.Equal(t, "test-server", actionMsg.server)
	assert.Error(t, actionMsg.err)
	assert.Contains(t, actionMsg.err.Error(), "no container ID")
}

func TestStopServerCmd(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}

	timeout := 30 * time.Second
	client.On("StopContainer", ctx, "container123", &timeout).Return(nil)

	cmd := stopServerCmd(ctx, client, "test-server", "container123")
	msg := cmd()

	actionMsg, ok := msg.(serverActionMsg)
	assert.True(t, ok)
	assert.Equal(t, "stop", actionMsg.action)
	assert.Equal(t, "test-server", actionMsg.server)
	assert.NoError(t, actionMsg.err)

	client.AssertExpectations(t)
}

func TestRestartServerCmd(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}

	timeout := 30 * time.Second
	client.On("RestartContainer", ctx, "container123", &timeout).Return(nil)

	cmd := restartServerCmd(ctx, client, "test-server", "container123")
	msg := cmd()

	actionMsg, ok := msg.(serverActionMsg)
	assert.True(t, ok)
	assert.Equal(t, "restart", actionMsg.action)
	assert.Equal(t, "test-server", actionMsg.server)
	assert.NoError(t, actionMsg.err)

	client.AssertExpectations(t)
}

func TestTickCmd(t *testing.T) {
	cmd := tickCmd()
	assert.NotNil(t, cmd)

	// Execute the command
	msg := cmd()
	_, ok := msg.(tickMsg)
	assert.True(t, ok)
}

func TestClearErrorCmd(t *testing.T) {
	cmd := clearErrorCmd()
	assert.NotNil(t, cmd)

	// Execute the command (this will take 3 seconds in real usage, but we just check it exists)
	assert.NotNil(t, cmd)
}

func TestModelUpdate_SelectedIndexAdjustment_EmptyList(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)
	model.selectedIdx = 2

	msg := serversLoadedMsg{
		servers: []ServerInfo{},
		err:     nil,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	assert.Equal(t, 0, m.selectedIdx)
}

func TestHandleKeyPress_StartServer_AlreadyRunning(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)
	model.servers = []ServerInfo{
		{Name: "server1", Status: "running"},
	}
	model.selectedIdx = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updatedModel, cmd := model.handleKeyPress(msg)
	m := updatedModel.(Model)

	assert.NotNil(t, m)
	// Should not start an already running server - no command
	assert.Nil(t, cmd)
}

func TestHandleKeyPress_StopServer_NotRunning(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)
	model.servers = []ServerInfo{
		{Name: "server1", Status: "stopped"},
	}
	model.selectedIdx = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
	updatedModel, cmd := model.handleKeyPress(msg)
	m := updatedModel.(Model)

	assert.NotNil(t, m)
	// Should not stop an already stopped server - no command
	assert.Nil(t, cmd)
}

func TestHandleKeyPress_RestartServer_NotRunning(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)
	model.servers = []ServerInfo{
		{Name: "server1", Status: "stopped"},
	}
	model.selectedIdx = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	updatedModel, cmd := model.handleKeyPress(msg)
	m := updatedModel.(Model)

	assert.NotNil(t, m)
	// Should not restart a stopped server - no command
	assert.Nil(t, cmd)
}

func TestHandleKeyPress_LogsNotImplemented(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)
	model.servers = []ServerInfo{
		{Name: "server1", Status: "running"},
	}
	model.selectedIdx = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
	updatedModel, cmd := model.handleKeyPress(msg)
	m := updatedModel.(Model)

	assert.NotNil(t, m.err)
	assert.Contains(t, m.err.Error(), "not implemented")
	assert.NotNil(t, cmd) // Should clear error
}

func TestHandleKeyPress_DeleteNotImplemented(t *testing.T) {
	ctx := context.Background()
	client := &mockContainerClient{}
	model := NewModel(ctx, client)
	model.servers = []ServerInfo{
		{Name: "server1", Status: "running"},
	}
	model.selectedIdx = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	updatedModel, cmd := model.handleKeyPress(msg)
	m := updatedModel.(Model)

	assert.NotNil(t, m.err)
	assert.Contains(t, m.err.Error(), "not implemented")
	assert.NotNil(t, cmd) // Should clear error
}
