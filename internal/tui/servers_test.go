package tui

import (
	"testing"
	"time"

	"github.com/steviee/go-mc/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeContainerState(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "running state",
			input:    "running",
			expected: "running",
		},
		{
			name:     "running state uppercase",
			input:    "RUNNING",
			expected: "running",
		},
		{
			name:     "created state",
			input:    "created",
			expected: "created",
		},
		{
			name:     "stopped state",
			input:    "stopped",
			expected: "stopped",
		},
		{
			name:     "exited state",
			input:    "exited",
			expected: "stopped",
		},
		{
			name:     "paused state",
			input:    "paused",
			expected: "paused",
		},
		{
			name:     "unknown state",
			input:    "restarting",
			expected: "restarting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeContainerState(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatUptime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		startedAt time.Time
		expected  string
	}{
		{
			name:      "zero time",
			startedAt: time.Time{},
			expected:  "-",
		},
		{
			name:      "5 minutes ago",
			startedAt: now.Add(-5 * time.Minute),
			expected:  "5m",
		},
		{
			name:      "30 minutes ago",
			startedAt: now.Add(-30 * time.Minute),
			expected:  "30m",
		},
		{
			name:      "1 hour 15 minutes ago",
			startedAt: now.Add(-75 * time.Minute),
			expected:  "1h 15m",
		},
		{
			name:      "2 hours ago",
			startedAt: now.Add(-2 * time.Hour),
			expected:  "2h 0m",
		},
		{
			name:      "1 day 3 hours ago",
			startedAt: now.Add(-27 * time.Hour),
			expected:  "1d 3h",
		},
		{
			name:      "5 days ago",
			startedAt: now.Add(-5 * 24 * time.Hour),
			expected:  "5d 0h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUptime(tt.startedAt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatUptime_EdgeCases(t *testing.T) {
	now := time.Now()

	// Test 23 hours 59 minutes (should show hours)
	startedAt := now.Add(-23*time.Hour - 59*time.Minute)
	result := formatUptime(startedAt)
	assert.Contains(t, result, "h")
	assert.NotContains(t, result, "d")

	// Test 24 hours 1 minute (should show days)
	startedAt = now.Add(-24*time.Hour - 1*time.Minute)
	result = formatUptime(startedAt)
	assert.Contains(t, result, "d")
}

func TestDetectPorts(t *testing.T) {
	tests := []struct {
		name     string
		server   *state.ServerState
		expected []PortInfo
	}{
		{
			name: "basic server with game and RCON ports",
			server: &state.ServerState{
				Name: "test-server",
				Minecraft: state.MinecraftConfig{
					GamePort: 25565,
					RconPort: 25575,
				},
				Mods: []state.ModInfo{},
			},
			expected: []PortInfo{
				{Number: 25565, Protocol: "tcp", Service: "Minecraft", Source: "config"},
				{Number: 25575, Protocol: "tcp", Service: "RCON", Source: "config"},
			},
		},
		{
			name: "server with voice chat mod",
			server: &state.ServerState{
				Name: "voice-server",
				Minecraft: state.MinecraftConfig{
					GamePort: 25565,
					RconPort: 25575,
				},
				Mods: []state.ModInfo{
					{Name: "Simple Voice Chat", Slug: "simple-voice-chat", Version: "2.5.10"},
				},
			},
			expected: []PortInfo{
				{Number: 25565, Protocol: "tcp", Service: "Minecraft", Source: "config"},
				{Number: 25575, Protocol: "tcp", Service: "RCON", Source: "config"},
				{Number: 24454, Protocol: "udp", Service: "Simple Voice Chat", Source: "mod"},
			},
		},
		{
			name: "server with geyser and bluemap mods",
			server: &state.ServerState{
				Name: "crossplay-server",
				Minecraft: state.MinecraftConfig{
					GamePort: 25565,
					RconPort: 25575,
				},
				Mods: []state.ModInfo{
					{Name: "Geyser", Slug: "geyser", Version: "2.1.0"},
					{Name: "BlueMap", Slug: "bluemap", Version: "3.15"},
				},
			},
			expected: []PortInfo{
				{Number: 25565, Protocol: "tcp", Service: "Minecraft", Source: "config"},
				{Number: 25575, Protocol: "tcp", Service: "RCON", Source: "config"},
				{Number: 19132, Protocol: "udp", Service: "Geyser (Bedrock)", Source: "mod"},
				{Number: 8100, Protocol: "tcp", Service: "BlueMap", Source: "mod"},
			},
		},
		{
			name: "server with custom ports",
			server: &state.ServerState{
				Name: "custom-server",
				Minecraft: state.MinecraftConfig{
					GamePort: 30000,
					RconPort: 30001,
				},
				Mods: []state.ModInfo{},
			},
			expected: []PortInfo{
				{Number: 30000, Protocol: "tcp", Service: "Minecraft", Source: "config"},
				{Number: 30001, Protocol: "tcp", Service: "RCON", Source: "config"},
			},
		},
		{
			name: "server with zero ports (edge case)",
			server: &state.ServerState{
				Name: "zero-server",
				Minecraft: state.MinecraftConfig{
					GamePort: 0,
					RconPort: 0,
				},
				Mods: []state.ModInfo{},
			},
			expected: []PortInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ports := detectPorts(tt.server)

			assert.Len(t, ports, len(tt.expected))
			for i, expected := range tt.expected {
				assert.Equal(t, expected.Number, ports[i].Number)
				assert.Equal(t, expected.Protocol, ports[i].Protocol)
				assert.Equal(t, expected.Service, ports[i].Service)
				assert.Equal(t, expected.Source, ports[i].Source)
			}
		})
	}
}

func TestDetectModPorts(t *testing.T) {
	tests := []struct {
		name     string
		mods     []state.ModInfo
		expected []PortInfo
	}{
		{
			name:     "no mods",
			mods:     []state.ModInfo{},
			expected: []PortInfo{},
		},
		{
			name:     "nil mods slice",
			mods:     nil,
			expected: []PortInfo{},
		},
		{
			name: "unknown mod",
			mods: []state.ModInfo{
				{Name: "Unknown Mod", Slug: "unknown-mod", Version: "1.0.0"},
			},
			expected: []PortInfo{},
		},
		{
			name: "simple voice chat",
			mods: []state.ModInfo{
				{Name: "Simple Voice Chat", Slug: "simple-voice-chat", Version: "2.5.10"},
			},
			expected: []PortInfo{
				{Number: 24454, Protocol: "udp", Service: "Simple Voice Chat", Source: "mod"},
			},
		},
		{
			name: "geyser mod",
			mods: []state.ModInfo{
				{Name: "Geyser", Slug: "geyser", Version: "2.1.0"},
			},
			expected: []PortInfo{
				{Number: 19132, Protocol: "udp", Service: "Geyser (Bedrock)", Source: "mod"},
			},
		},
		{
			name: "bluemap mod",
			mods: []state.ModInfo{
				{Name: "BlueMap", Slug: "bluemap", Version: "3.15"},
			},
			expected: []PortInfo{
				{Number: 8100, Protocol: "tcp", Service: "BlueMap", Source: "mod"},
			},
		},
		{
			name: "multiple known mods",
			mods: []state.ModInfo{
				{Name: "Geyser", Slug: "geyser", Version: "2.1.0"},
				{Name: "BlueMap", Slug: "bluemap", Version: "3.15"},
				{Name: "Some Other Mod", Slug: "other-mod", Version: "1.0.0"},
			},
			expected: []PortInfo{
				{Number: 19132, Protocol: "udp", Service: "Geyser (Bedrock)", Source: "mod"},
				{Number: 8100, Protocol: "tcp", Service: "BlueMap", Source: "mod"},
			},
		},
		{
			name: "all known mods",
			mods: []state.ModInfo{
				{Name: "Simple Voice Chat", Slug: "simple-voice-chat", Version: "2.5.10"},
				{Name: "Geyser", Slug: "geyser", Version: "2.1.0"},
				{Name: "BlueMap", Slug: "bluemap", Version: "3.15"},
			},
			expected: []PortInfo{
				{Number: 24454, Protocol: "udp", Service: "Simple Voice Chat", Source: "mod"},
				{Number: 19132, Protocol: "udp", Service: "Geyser (Bedrock)", Source: "mod"},
				{Number: 8100, Protocol: "tcp", Service: "BlueMap", Source: "mod"},
			},
		},
		{
			name: "mix of known and unknown mods",
			mods: []state.ModInfo{
				{Name: "Unknown Mod 1", Slug: "unknown-1", Version: "1.0.0"},
				{Name: "Simple Voice Chat", Slug: "simple-voice-chat", Version: "2.5.10"},
				{Name: "Unknown Mod 2", Slug: "unknown-2", Version: "1.0.0"},
				{Name: "Geyser", Slug: "geyser", Version: "2.1.0"},
			},
			expected: []PortInfo{
				{Number: 24454, Protocol: "udp", Service: "Simple Voice Chat", Source: "mod"},
				{Number: 19132, Protocol: "udp", Service: "Geyser (Bedrock)", Source: "mod"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ports := detectModPorts(tt.mods)

			if len(tt.expected) == 0 {
				assert.Empty(t, ports)
			} else {
				assert.Len(t, ports, len(tt.expected))
				for i, expected := range tt.expected {
					assert.Equal(t, expected.Number, ports[i].Number)
					assert.Equal(t, expected.Protocol, ports[i].Protocol)
					assert.Equal(t, expected.Service, ports[i].Service)
					assert.Equal(t, expected.Source, ports[i].Source)
				}
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0B",
		},
		{
			name:     "bytes only",
			bytes:    512,
			expected: "512B",
		},
		{
			name:     "exactly 1 KB",
			bytes:    1024,
			expected: "1.0K",
		},
		{
			name:     "kilobytes",
			bytes:    2048,
			expected: "2.0K",
		},
		{
			name:     "fractional kilobytes",
			bytes:    1536, // 1.5 KB
			expected: "1.5K",
		},
		{
			name:     "exactly 1 MB",
			bytes:    1024 * 1024,
			expected: "1.0M",
		},
		{
			name:     "megabytes",
			bytes:    5 * 1024 * 1024,
			expected: "5.0M",
		},
		{
			name:     "fractional megabytes",
			bytes:    (2*1024 + 512) * 1024, // 2.5 MB
			expected: "2.5M",
		},
		{
			name:     "exactly 1 GB",
			bytes:    1024 * 1024 * 1024,
			expected: "1.0G",
		},
		{
			name:     "gigabytes",
			bytes:    3 * 1024 * 1024 * 1024,
			expected: "3.0G",
		},
		{
			name:     "fractional gigabytes",
			bytes:    (3*1024 + 512) * 1024 * 1024,
			expected: "3.5G",
		},
		{
			name:     "large gigabytes",
			bytes:    128 * 1024 * 1024 * 1024,
			expected: "128.0G",
		},
		{
			name:     "rounding edge case",
			bytes:    (1024 + 51) * 1024, // 1.05 MB, should round to 1.0M
			expected: "1.0M",
		},
		{
			name:     "small bytes under threshold",
			bytes:    1023,
			expected: "1023B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
