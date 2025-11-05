package main

import (
	"fmt"
	"os"
)

// Version information (set by ldflags during build)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	fmt.Printf("go-mc version %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
	fmt.Println()
	fmt.Println("This is a placeholder. The CLI will be implemented in Phase 1.")
	fmt.Println("See https://github.com/steviee/go-mc for development progress.")
	os.Exit(0)
}
