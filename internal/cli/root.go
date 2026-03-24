// Package cli defines the Cobra command tree and CLI boundary for havn.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// ErrNotImplemented is returned by commands that are wired but not yet backed
// by domain logic.
var ErrNotImplemented = errors.New("not implemented")

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

// rootOpts holds all flag values for the root command.
type rootOpts struct {
	// Persistent (global) flags.
	JSON    bool
	Verbose bool
	Config  string

	// Local container flags (root command only).
	Shell  string
	Env    string
	CPUs   int
	Memory string
	Port   string
	NoDolt bool
	Image  string
}

// NewRoot creates the root cobra command with the given dependencies.
func NewRoot(_ Deps) *cobra.Command {
	var opts rootOpts

	root := &cobra.Command{
		Use:   "havn [flags] [path]",
		Short: "Manage development environment containers",
		Args:  cobra.MaximumNArgs(1),

		SilenceErrors: true,
		SilenceUsage:  true,

		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn: %w", ErrNotImplemented)
		},
	}

	root.PersistentFlags().BoolVar(&opts.JSON, "json", false, "machine-readable JSON output")
	root.PersistentFlags().BoolVar(&opts.Verbose, "verbose", false, "show detailed output")
	root.PersistentFlags().StringVar(&opts.Config, "config", "", "path to config file")

	root.Flags().StringVar(&opts.Shell, "shell", "", "devShell to activate")
	root.Flags().StringVar(&opts.Env, "env", "", "Nix flake ref for dev environment")
	root.Flags().IntVar(&opts.CPUs, "cpus", 0, "CPU limit")
	root.Flags().StringVar(&opts.Memory, "memory", "", "memory limit")
	root.Flags().StringVar(&opts.Port, "port", "", "SSH port mapping")
	root.Flags().BoolVar(&opts.NoDolt, "no-dolt", false, "skip Dolt server")
	root.Flags().StringVar(&opts.Image, "image", "", "override base image")

	return root
}
