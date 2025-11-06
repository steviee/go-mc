package mods

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

// requireSearchQuery validates that a search query is provided
func requireSearchQuery(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("search query is required\nUsage: %s <query>\n\nExample: go-mc mods search sodium\n\nRun '%s --help' for more information", cmd.UseLine(), cmd.CommandPath())
	}
	if len(args) > 1 {
		return fmt.Errorf("only one search query allowed (use quotes for multi-word queries)\nGot: %v\n\nExample: go-mc mods search \"fabric api\"\n\nRun '%s --help' for more information", args, cmd.CommandPath())
	}
	return nil
}
