package servers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsJSONMode(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		want    bool
		cleanup bool
	}{
		{
			name:    "json mode disabled by default",
			envVal:  "",
			want:    false,
			cleanup: false,
		},
		{
			name:    "json mode enabled",
			envVal:  "true",
			want:    true,
			cleanup: true,
		},
		{
			name:    "json mode disabled explicitly",
			envVal:  "false",
			want:    false,
			cleanup: true,
		},
		{
			name:    "invalid value treated as false",
			envVal:  "invalid",
			want:    false,
			cleanup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env value
			original := os.Getenv("GOMC_JSON")
			defer func() {
				if tt.cleanup {
					if original != "" {
						_ = os.Setenv("GOMC_JSON", original)
					} else {
						_ = os.Unsetenv("GOMC_JSON")
					}
				}
			}()

			// Set test env value
			if tt.envVal != "" {
				_ = os.Setenv("GOMC_JSON", tt.envVal)
			} else {
				_ = os.Unsetenv("GOMC_JSON")
			}

			// Test function
			got := isJSONMode()
			assert.Equal(t, tt.want, got)
		})
	}
}
