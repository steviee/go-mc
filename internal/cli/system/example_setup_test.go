package system_test

import (
	"fmt"

	"github.com/steviee/go-mc/internal/cli/system"
)

// ExampleNewSetupCommand demonstrates the basic usage of the setup command.
func ExampleNewSetupCommand() {
	cmd := system.NewSetupCommand()
	fmt.Println(cmd.Use)
	fmt.Println(cmd.Short)
	// Output:
	// setup
	// First-time system setup
}
