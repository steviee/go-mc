package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCommand(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		commit      string
		date        string
		builtBy     string
		wantUse     string
		wantShort   string
		wantAliases []string
	}{
		{
			name:        "creates root command with version info",
			version:     "1.0.0",
			commit:      "abc123",
			date:        "2025-11-05",
			builtBy:     "goreleaser",
			wantUse:     "go-mc",
			wantShort:   "Manage Minecraft Fabric servers with Podman",
			wantAliases: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand(tt.version, tt.commit, tt.date, tt.builtBy)

			assert.Equal(t, tt.wantUse, cmd.Use)
			assert.Equal(t, tt.wantShort, cmd.Short)
			assert.NotEmpty(t, cmd.Long)
			assert.NotEmpty(t, cmd.Example)
		})
	}
}

func TestRootCommand_GlobalFlags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		wantType string
	}{
		{
			name:     "has config flag",
			flagName: "config",
			wantType: "string",
		},
		{
			name:     "has json flag",
			flagName: "json",
			wantType: "bool",
		},
		{
			name:     "has quiet flag",
			flagName: "quiet",
			wantType: "bool",
		},
		{
			name:     "has verbose flag",
			flagName: "verbose",
			wantType: "bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand("dev", "unknown", "unknown", "unknown")

			flag := cmd.PersistentFlags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
			assert.Equal(t, tt.wantType, flag.Value.Type())
		})
	}
}

func TestRootCommand_Subcommands(t *testing.T) {
	tests := []struct {
		name        string
		commandName string
		wantShort   string
	}{
		{
			name:        "has version command",
			commandName: "version",
			wantShort:   "Print version information",
		},
		{
			name:        "has servers command",
			commandName: "servers",
			wantShort:   "Manage Minecraft Fabric servers",
		},
		{
			name:        "has users command",
			commandName: "users",
			wantShort:   "Manage Minecraft users and permissions",
		},
		{
			name:        "has whitelist command",
			commandName: "whitelist",
			wantShort:   "Manage server whitelist",
		},
		{
			name:        "has mods command",
			commandName: "mods",
			wantShort:   "Manage Modrinth mods",
		},
		{
			name:        "has system command",
			commandName: "system",
			wantShort:   "System management and monitoring",
		},
		{
			name:        "has config command",
			commandName: "config",
			wantShort:   "Manage configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand("dev", "unknown", "unknown", "unknown")

			subCmd := findCommand(cmd, tt.commandName)
			require.NotNil(t, subCmd, "command %s should exist", tt.commandName)
			assert.Equal(t, tt.wantShort, subCmd.Short)
		})
	}
}

func TestRootCommand_Execute(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "help flag",
			args:       []string{"--help"},
			wantOutput: "go-mc is a CLI tool for managing Minecraft Fabric servers",
			wantErr:    false,
		},
		{
			name:       "version command",
			args:       []string{"version"},
			wantOutput: "go-mc version dev",
			wantErr:    false,
		},
		{
			name:       "invalid command",
			args:       []string{"invalid"},
			wantOutput: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand("dev", "unknown", "unknown", "unknown")
			cmd.SetArgs(tt.args)

			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			err := cmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.wantOutput != "" {
					assert.Contains(t, out.String(), tt.wantOutput)
				}
			}
		})
	}
}

func TestRootCommand_FlagInteractions(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "json and quiet flags are mutually exclusive",
			args:    []string{"--json", "--quiet", "version"},
			wantErr: true,
			errMsg:  "if any flags in the group [json quiet] are set none of the others can be",
		},
		{
			name:    "verbose and quiet flags are mutually exclusive",
			args:    []string{"--verbose", "--quiet", "version"},
			wantErr: true,
			errMsg:  "if any flags in the group [verbose quiet] are set none of the others can be",
		},
		{
			name:    "json and verbose flags can be used together",
			args:    []string{"--json", "--verbose", "version"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand("dev", "unknown", "unknown", "unknown")
			cmd.SetArgs(tt.args)

			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			err := cmd.Execute()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetLogger(t *testing.T) {
	// Initialize logger by creating a command and running PersistentPreRunE
	cmd := NewRootCommand("dev", "unknown", "unknown", "unknown")
	cmd.SetArgs([]string{"version"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	require.NoError(t, err)

	// GetLogger should return a non-nil logger
	logger := GetLogger()
	assert.NotNil(t, logger)
}

func TestFlagAccessors(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		accessor func() bool
		want     bool
	}{
		{
			name:     "IsJSONOutput returns true when --json is set",
			args:     []string{"--json", "version"},
			accessor: IsJSONOutput,
			want:     true,
		},
		{
			name:     "IsJSONOutput returns false when --json is not set",
			args:     []string{"version"},
			accessor: IsJSONOutput,
			want:     false,
		},
		{
			name:     "IsQuiet returns true when --quiet is set",
			args:     []string{"--quiet", "version"},
			accessor: IsQuiet,
			want:     true,
		},
		{
			name:     "IsQuiet returns false when --quiet is not set",
			args:     []string{"version"},
			accessor: IsQuiet,
			want:     false,
		},
		{
			name:     "IsVerbose returns true when --verbose is set",
			args:     []string{"--verbose", "version"},
			accessor: IsVerbose,
			want:     true,
		},
		{
			name:     "IsVerbose returns false when --verbose is not set",
			args:     []string{"version"},
			accessor: IsVerbose,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			jsonOut = false
			quiet = false
			verbose = false

			cmd := NewRootCommand("dev", "unknown", "unknown", "unknown")
			cmd.SetArgs(tt.args)

			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			err := cmd.Execute()
			require.NoError(t, err)

			got := tt.accessor()
			assert.Equal(t, tt.want, got)
		})
	}
}

// Helper function to find a command by name
func findCommand(rootCmd *cobra.Command, name string) *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}
