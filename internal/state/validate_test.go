package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateServerName(t *testing.T) {
	tests := []struct {
		name    string
		srvName string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple name",
			srvName: "survival",
			wantErr: false,
		},
		{
			name:    "valid name with hyphen",
			srvName: "creative-world",
			wantErr: false,
		},
		{
			name:    "valid name with numbers",
			srvName: "server1",
			wantErr: false,
		},
		{
			name:    "valid single character",
			srvName: "a",
			wantErr: false,
		},
		{
			name:    "empty name",
			srvName: "",
			wantErr: true,
			errMsg:  "server name cannot be empty",
		},
		{
			name:    "name with underscore",
			srvName: "my_server",
			wantErr: true,
			errMsg:  "must contain only alphanumeric",
		},
		{
			name:    "name starting with hyphen",
			srvName: "-server",
			wantErr: true,
			errMsg:  "must contain only alphanumeric",
		},
		{
			name:    "name ending with hyphen",
			srvName: "server-",
			wantErr: true,
			errMsg:  "must contain only alphanumeric",
		},
		{
			name:    "name too long",
			srvName: "this-is-a-very-long-server-name-that-exceeds-the-maximum-length-limit",
			wantErr: true,
			errMsg:  "must be 63 characters or less",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerName(tt.srvName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name    string
		uuid    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid UUID",
			uuid:    "069a79f4-44e9-4726-a5be-fca90e38aaf5",
			wantErr: false,
		},
		{
			name:    "valid UUID uppercase",
			uuid:    "069A79F4-44E9-4726-A5BE-FCA90E38AAF5",
			wantErr: false,
		},
		{
			name:    "empty UUID",
			uuid:    "",
			wantErr: true,
			errMsg:  "UUID cannot be empty",
		},
		{
			name:    "invalid format",
			uuid:    "not-a-uuid",
			wantErr: true,
			errMsg:  "invalid UUID format",
		},
		{
			name:    "missing hyphens",
			uuid:    "069a79f444e94726a5befca90e38aaf5",
			wantErr: true,
			errMsg:  "invalid UUID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUID(tt.uuid)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid port",
			port:    25565,
			wantErr: false,
		},
		{
			name:    "minimum port",
			port:    1,
			wantErr: false,
		},
		{
			name:    "maximum port",
			port:    65535,
			wantErr: false,
		},
		{
			name:    "port too low",
			port:    0,
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name:    "port too high",
			port:    65536,
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name:    "negative port",
			port:    -1,
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateMemory(t *testing.T) {
	tests := []struct {
		name    string
		memory  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid megabytes",
			memory:  "512M",
			wantErr: false,
		},
		{
			name:    "valid gigabytes",
			memory:  "2G",
			wantErr: false,
		},
		{
			name:    "valid terabytes",
			memory:  "1T",
			wantErr: false,
		},
		{
			name:    "empty memory",
			memory:  "",
			wantErr: true,
			errMsg:  "memory cannot be empty",
		},
		{
			name:    "missing unit",
			memory:  "512",
			wantErr: true,
			errMsg:  "invalid memory format",
		},
		{
			name:    "lowercase unit",
			memory:  "512m",
			wantErr: true,
			errMsg:  "invalid memory format",
		},
		{
			name:    "invalid unit",
			memory:  "512K",
			wantErr: true,
			errMsg:  "invalid memory format",
		},
		{
			name:    "space between number and unit",
			memory:  "512 M",
			wantErr: true,
			errMsg:  "invalid memory format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMemory(tt.memory)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid version",
			version: "1.20.4",
			wantErr: false,
		},
		{
			name:    "valid snapshot version",
			version: "23w51a",
			wantErr: false,
		},
		{
			name:    "empty version",
			version: "",
			wantErr: true,
			errMsg:  "version cannot be empty",
		},
		{
			name:    "version with spaces",
			version: "1.20 4",
			wantErr: true,
			errMsg:  "version cannot contain spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVersion(tt.version)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateWhitelistName(t *testing.T) {
	tests := []struct {
		name     string
		listName string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid name",
			listName: "default",
			wantErr:  false,
		},
		{
			name:     "valid name with hyphen",
			listName: "vip-players",
			wantErr:  false,
		},
		{
			name:     "empty name",
			listName: "",
			wantErr:  true,
			errMsg:   "whitelist name cannot be empty",
		},
		{
			name:     "name with underscore",
			listName: "vip_players",
			wantErr:  true,
			errMsg:   "must contain only alphanumeric",
		},
		{
			name:     "name too long",
			listName: "this-is-a-very-long-whitelist-name-that-exceeds-the-maximum-allowed-length-limit",
			wantErr:  true,
			errMsg:   "must be 63 characters or less",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWhitelistName(tt.listName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePlayerName(t *testing.T) {
	tests := []struct {
		name       string
		playerName string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid name",
			playerName: "Notch",
			wantErr:    false,
		},
		{
			name:       "valid name with underscore",
			playerName: "Player_1",
			wantErr:    false,
		},
		{
			name:       "valid name with numbers",
			playerName: "Steve123",
			wantErr:    false,
		},
		{
			name:       "empty name",
			playerName: "",
			wantErr:    true,
			errMsg:     "player name cannot be empty",
		},
		{
			name:       "name too long",
			playerName: "ThisNameIsTooLongForMinecraft",
			wantErr:    true,
			errMsg:     "player name must be 16 characters or less",
		},
		{
			name:       "name with hyphen",
			playerName: "Player-1",
			wantErr:    true,
			errMsg:     "must contain only alphanumeric characters and underscores",
		},
		{
			name:       "name with space",
			playerName: "Player 1",
			wantErr:    true,
			errMsg:     "must contain only alphanumeric characters and underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePlayerName(tt.playerName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateJavaVersion(t *testing.T) {
	tests := []struct {
		name    string
		version int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Java 8",
			version: 8,
			wantErr: false,
		},
		{
			name:    "Java 11",
			version: 11,
			wantErr: false,
		},
		{
			name:    "Java 17",
			version: 17,
			wantErr: false,
		},
		{
			name:    "Java 21",
			version: 21,
			wantErr: false,
		},
		{
			name:    "invalid version",
			version: 16,
			wantErr: true,
			errMsg:  "invalid Java version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJavaVersion(tt.version)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateOpLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "level 1",
			level:   1,
			wantErr: false,
		},
		{
			name:    "level 4",
			level:   4,
			wantErr: false,
		},
		{
			name:    "level 0",
			level:   0,
			wantErr: true,
			errMsg:  "op level must be between 1 and 4",
		},
		{
			name:    "level 5",
			level:   5,
			wantErr: true,
			errMsg:  "op level must be between 1 and 4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOpLevel(tt.level)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid path",
			path:    "/home/user/.config/go-mc",
			wantErr: false,
		},
		{
			name:    "valid relative path",
			path:    "config/servers",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "path cannot be empty",
		},
		{
			name:    "path with directory traversal",
			path:    "/home/user/../../../etc/passwd",
			wantErr: true,
			errMsg:  "path cannot contain '..'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
