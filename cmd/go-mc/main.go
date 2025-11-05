package main

import (
	"fmt"
	"os"

	"github.com/steviee/go-mc/internal/cli"
	"github.com/steviee/go-mc/internal/state"
)

// Version information (set by ldflags during build)
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
	BuiltBy = "unknown"
)

func main() {
	// Acquire PID lock first to prevent concurrent execution
	pidLock, err := state.AcquirePIDLock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = pidLock.Release() }()

	// Setup signal handler for graceful shutdown
	sigChan := state.SetupSignalHandler()
	go func() {
		<-sigChan
		// Release lock on signal
		_ = pidLock.Release()
		os.Exit(0)
	}()

	// Execute CLI commands
	rootCmd := cli.NewRootCommand(Version, Commit, Date, BuiltBy)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		_ = pidLock.Release()
		os.Exit(1)
	}
}
