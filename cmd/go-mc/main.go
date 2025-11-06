package main

import (
	"context"
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

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handler for graceful shutdown
	sigChan := state.SetupSignalHandler()
	go func() {
		<-sigChan
		cancel() // Signal shutdown via context cancellation
	}()

	// Execute CLI commands with context
	rootCmd := cli.NewRootCommand(Version, Commit, Date, BuiltBy)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
