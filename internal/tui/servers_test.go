package tui

import (
	"testing"
	"time"

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
