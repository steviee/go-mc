package servers

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// isJSONMode checks if JSON output mode is enabled
// This would normally check a global flag, but for now we'll check environment
func isJSONMode() bool {
	// This is a placeholder - in the real implementation, this would check
	// the global --json flag via the root command context or viper
	return os.Getenv("GOMC_JSON") == "true"
}

// requireServerName validates that exactly one server name is provided
func requireServerName(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("server name is required\nUsage: %s <server-name>\n\nRun '%s --help' for more information", cmd.UseLine(), cmd.CommandPath())
	}
	if len(args) > 1 {
		return fmt.Errorf("only one server name allowed, got: %v\nUsage: %s <server-name>\n\nRun '%s --help' for more information", args, cmd.UseLine(), cmd.CommandPath())
	}
	return nil
}
