package mods

import "os"

// isJSONMode checks if JSON output mode is enabled
// This would normally check a global flag, but for now we'll check environment
func isJSONMode() bool {
	// This is a placeholder - in the real implementation, this would check
	// the global --json flag via the root command context or viper
	return os.Getenv("GOMC_JSON") == "true"
}
