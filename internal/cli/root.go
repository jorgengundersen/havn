// Package cli defines the Cobra command tree and CLI boundary for havn.
package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/doctor"
	"github.com/jorgengundersen/havn/internal/volume"
)

// version is set at build time via ldflags.
var version = "dev"

// ErrNotImplemented is returned by commands that are wired but not yet backed
// by domain logic.
var ErrNotImplemented = errors.New("not implemented")

// Execute wires production dependencies, runs the command tree, and returns
// an exit code suitable for os.Exit.
func Execute() int {
	dockerClient, err := docker.NewClient()
	if err != nil {
		out := NewOutput(os.Stdout, os.Stderr, false, false)
		out.Error(fmt.Errorf("docker client: %w", err))
		return 1
	}
	deps := Deps{Docker: dockerClient}
	root := NewRoot(deps)
	// Logger is wired via PersistentPreRunE after flags are parsed.
	if err := root.Execute(); err != nil {
		jsonMode, _ := root.PersistentFlags().GetBool("json")
		verboseMode, _ := root.PersistentFlags().GetBool("verbose")
		out := NewOutput(os.Stdout, os.Stderr, jsonMode, verboseMode)
		out.Error(err)
		return ExitCode(err)
	}
	return 0
}

// Deps holds dependencies injected into the command tree.
// Starts empty during skeleton phase; fields added as domain packages land.
type Deps struct {
	Docker        *docker.Client
	DoctorBackend doctor.Backend
	VolumeManager *volume.Manager
	Logger        *slog.Logger
}

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
func NewRoot(deps Deps) *cobra.Command {
	var opts rootOpts

	root := &cobra.Command{
		Use:   "havn [flags] [path]",
		Short: "Manage development environment containers",
		Args:  cobra.MaximumNArgs(1),

		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,

		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if deps.Logger == nil {
				deps.Logger = SetupLogger(opts.Verbose, opts.JSON)
			}
			return nil
		},

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

	root.AddCommand(newListCmd())
	root.AddCommand(newStopCmd())
	root.AddCommand(newBuildCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newVolumeCmd())
	root.AddCommand(newDoctorCmd(deps.DoctorBackend))
	root.AddCommand(newDoltCmd())

	return root
}
