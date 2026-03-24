// Package cli defines the Cobra command tree and CLI boundary for havn.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute wires production dependencies, runs the command tree, and returns
// an exit code suitable for os.Exit.
func Execute() int {
	deps := Deps{}
	root := NewRoot(deps)
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 1
	}
	return 0
}

// Deps holds dependencies injected into the command tree.
// Starts empty during skeleton phase; fields added as domain packages land.
type Deps struct{}

// NewRoot creates the root cobra command with the given dependencies.
func NewRoot(_ Deps) *cobra.Command {
	root := &cobra.Command{
		Use:   "havn",
		Short: "Manage development environment containers",

		SilenceErrors: true,
		SilenceUsage:  true,

		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("havn")
			return nil
		},
	}

	return root
}
