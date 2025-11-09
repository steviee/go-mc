package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/steviee/go-mc/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockContainerClient is a mock implementation of container.Client
type mockContainerClient struct {
	mock.Mock
}

func (m *mockContainerClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockContainerClient) Info(ctx context.Context) (*container.RuntimeInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.RuntimeInfo), args.Error(1)
}

func (m *mockContainerClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockContainerClient) Runtime() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockContainerClient) CreateContainer(ctx context.Context, config *container.ContainerConfig) (string, error) {
	args := m.Called(ctx, config)
	return args.String(0), args.Error(1)
}

func (m *mockContainerClient) StartContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *mockContainerClient) WaitForContainer(ctx context.Context, containerID string, condition string) error {
	args := m.Called(ctx, containerID, condition)
	return args.Error(0)
}

func (m *mockContainerClient) StopContainer(ctx context.Context, containerID string, timeout *time.Duration) error {
	args := m.Called(ctx, containerID, timeout)
	return args.Error(0)
}

func (m *mockContainerClient) RestartContainer(ctx context.Context, containerID string, timeout *time.Duration) error {
	args := m.Called(ctx, containerID, timeout)
	return args.Error(0)
}

func (m *mockContainerClient) RemoveContainer(ctx context.Context, containerID string, opts *container.RemoveOptions) error {
	args := m.Called(ctx, containerID, opts)
	return args.Error(0)
}

func (m *mockContainerClient) InspectContainer(ctx context.Context, containerID string) (*container.ContainerInfo, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.ContainerInfo), args.Error(1)
}

func (m *mockContainerClient) ListContainers(ctx context.Context, opts *container.ListOptions) ([]*container.ContainerInfo, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*container.ContainerInfo), args.Error(1)
}

func (m *mockContainerClient) GetContainerStats(ctx context.Context, containerID string) (*container.ContainerStats, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.ContainerStats), args.Error(1)
}

func TestNewModel(t *testing.T) {
	client := &mockContainerClient{}

	model := NewModel(client)

	assert.NotNil(t, model)
	assert.Equal(t, 0, model.selectedIdx)
	assert.True(t, model.loading)
	assert.Empty(t, model.servers)
	assert.Equal(t, client, model.containerClient)
}

func TestModelUpdate_WindowSize(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)

	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 50,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestModelUpdate_ServersLoaded(t *testing.T) {
	tests := []struct {
		name          string
		servers       []ServerInfo
		err           error
		expectLoading bool
		expectError   bool
		expectServers int
	}{
		{
			name: "successful load",
			servers: []ServerInfo{
				{Name: "server1", Status: "running"},
				{Name: "server2", Status: "stopped"},
			},
			err:           nil,
			expectLoading: false,
			expectError:   false,
			expectServers: 2,
		},
		{
			name:          "load error",
			servers:       nil,
			err:           assert.AnError,
			expectLoading: false,
			expectError:   true,
			expectServers: 0,
		},
		{
			name:          "empty server list",
			servers:       []ServerInfo{},
			err:           nil,
			expectLoading: false,
			expectError:   false,
			expectServers: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockContainerClient{}
			model := NewModel(client)
			model.loading = true

			msg := serversLoadedMsg{
				servers: tt.servers,
				err:     tt.err,
			}

			updatedModel, _ := model.Update(msg)
			m := updatedModel.(Model)

			assert.Equal(t, tt.expectLoading, m.loading)
			assert.Equal(t, tt.expectServers, len(m.servers))
			if tt.expectError {
				assert.NotNil(t, m.err)
			} else {
				assert.Nil(t, m.err)
			}
		})
	}
}

func TestModelUpdate_KeyNavigation(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		initialIdx   int
		serversCount int
		expectedIdx  int
	}{
		{
			name:         "move down",
			key:          "down",
			initialIdx:   0,
			serversCount: 3,
			expectedIdx:  1,
		},
		{
			name:         "move up",
			key:          "up",
			initialIdx:   2,
			serversCount: 3,
			expectedIdx:  1,
		},
		{
			name:         "j key (vim style down)",
			key:          "j",
			initialIdx:   0,
			serversCount: 3,
			expectedIdx:  1,
		},
		{
			name:         "k key (vim style up)",
			key:          "k",
			initialIdx:   1,
			serversCount: 3,
			expectedIdx:  0,
		},
		{
			name:         "up at top boundary",
			key:          "up",
			initialIdx:   0,
			serversCount: 3,
			expectedIdx:  0,
		},
		{
			name:         "down at bottom boundary",
			key:          "down",
			initialIdx:   2,
			serversCount: 3,
			expectedIdx:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockContainerClient{}
			model := NewModel(client)
			model.selectedIdx = tt.initialIdx
			model.servers = make([]ServerInfo, tt.serversCount)
			for i := 0; i < tt.serversCount; i++ {
				model.servers[i] = ServerInfo{Name: "server" + string(rune(i))}
			}

			var msg tea.KeyMsg
			switch tt.key {
			case "up":
				msg = tea.KeyMsg{Type: tea.KeyUp}
			case "down":
				msg = tea.KeyMsg{Type: tea.KeyDown}
			default:
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			updatedModel, _ := model.Update(msg)
			m := updatedModel.(Model)

			assert.Equal(t, tt.expectedIdx, m.selectedIdx)
		})
	}
}

func TestModelUpdate_QuitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"q key", "q"},
		{"ctrl+c", "ctrl+c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockContainerClient{}
			model := NewModel(client)

			var msg tea.Msg
			if tt.key == "ctrl+c" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			updatedModel, cmd := model.Update(msg)
			m := updatedModel.(Model)

			assert.True(t, m.quitting)
			assert.NotNil(t, cmd)
		})
	}
}

func TestModelUpdate_ClearError(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)
	model.err = assert.AnError
	model.errorTime = time.Now().Add(-4 * time.Second) // Error from 4 seconds ago

	msg := clearErrorMsg{}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	assert.Nil(t, m.err)
}

func TestModelUpdate_ClearError_Recent(t *testing.T) {
	client := &mockContainerClient{}
	model := NewModel(client)
	model.err = assert.AnError
	model.errorTime = time.Now().Add(-1 * time.Second) // Error from 1 second ago

	msg := clearErrorMsg{}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Error should still be there since it's less than 3 seconds old
	assert.NotNil(t, m.err)
}

func TestLoadServersCmd(t *testing.T) {
	client := &mockContainerClient{}
	ctx := context.Background()

	// Test that loadServersCmd returns a function
	cmd := loadServersCmd(ctx, client)
	assert.NotNil(t, cmd)

	// Execute the command and verify it returns a message
	msg := cmd()
	assert.NotNil(t, msg)

	// Verify the message is of the correct type
	_, ok := msg.(serversLoadedMsg)
	assert.True(t, ok, "expected serversLoadedMsg type")
}
