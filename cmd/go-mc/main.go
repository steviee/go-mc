package main

import (
	"fmt"
	"os"

	"github.com/steviee/go-mc/internal/cli"
)

// Version information (set by ldflags during build)
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
	BuiltBy = "unknown"
)

func main() {
	rootCmd := cli.NewRootCommand(Version, Commit, Date, BuiltBy)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
