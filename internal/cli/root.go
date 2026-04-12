// Package cli defines the Cobra command tree and CLI boundary for havn.
package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/doctor"
	"github.com/jorgengundersen/havn/internal/dolt"
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
		var shellExit *ShellExitError
		if errors.As(err, &shellExit) {
			return shellExit.Code
		}
		out.Error(err)
		return ExitCode(err)
	}
	return 0
}

// Deps holds dependencies injected into the command tree.
// Starts empty during skeleton phase; fields added as domain packages land.
type Deps struct {
	Docker        *docker.Client
	ContainerList container.Backend
	ContainerStop container.StopBackend
	DoctorBackend doctor.Backend
	VolumeManager *volume.Manager
	DoltManager   *dolt.Manager
	DoltSetup     *dolt.Setup
	BuildService  BuildService
	StartService  StartService
	Logger        *slog.Logger
}

// StartService is the CLI-facing start-or-attach dependency for the root
// command.
type StartService interface {
	StartOrAttach(ctx context.Context, cfg config.Config, projectPath string, status func(msg string)) (int, error)
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

	if deps.ContainerList == nil && deps.Docker != nil {
		deps.ContainerList = dockerContainerBackend{docker: deps.Docker}
	}
	if deps.ContainerStop == nil && deps.Docker != nil {
		deps.ContainerStop = dockerContainerBackend{docker: deps.Docker}
	}
	if deps.DoctorBackend == nil && deps.Docker != nil {
		deps.DoctorBackend = dockerDoctorBackend{docker: deps.Docker}
	}
	if deps.VolumeManager == nil && deps.Docker != nil {
		deps.VolumeManager = volume.NewManager(dockerVolumeBackend{docker: deps.Docker})
	}

	if deps.DoltManager == nil && deps.Docker != nil {
		doltBackend := dockerDoltBackend{docker: deps.Docker}
		deps.DoltManager = dolt.NewManager(doltBackend)
	}
	if deps.DoltSetup == nil && deps.DoltManager != nil && deps.Docker != nil {
		doltBackend := dockerDoltBackend{docker: deps.Docker}
		deps.DoltSetup = dolt.NewSetup(deps.DoltManager, doltBackend)
	}
	if deps.StartService == nil && deps.Docker != nil {
		deps.StartService = dockerStartService{docker: deps.Docker}
	}

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
			if deps.Docker != nil {
				deps.Docker.SetLogger(deps.Logger)
			}
			return nil
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.StartService == nil {
				return fmt.Errorf("havn: %w", ErrNotImplemented)
			}

			projectArg := "."
			if len(args) == 1 {
				projectArg = args[0]
			}

			projectPath, err := resolveProjectPath(projectArg)
			if err != nil {
				return err
			}

			cfg, err := resolveStartConfig(cmd, opts, projectPath)
			if err != nil {
				return err
			}

			out := commandOutput(cmd)
			exitCode, err := deps.StartService.StartOrAttach(cmd.Context(), cfg, projectPath, out.Status)
			if err != nil {
				return err
			}
			if exitCode != 0 {
				return &ShellExitError{Code: exitCode}
			}

			return nil
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

	root.AddCommand(newListCmd(deps.ContainerList))
	root.AddCommand(newStopCmd(deps.ContainerStop))

	buildService := deps.BuildService
	if buildService == nil && deps.Docker != nil {
		buildService = dockerBuildService{docker: deps.Docker}
	}
	root.AddCommand(newBuildCmd(buildService))
	root.AddCommand(newConfigCmd())
	root.AddCommand(newVolumeCmd(deps.VolumeManager))
	root.AddCommand(newDoctorCmd(deps.DoctorBackend))
	root.AddCommand(newDoltCmd(deps.DoltManager, deps.DoltSetup))

	return root
}

func resolveProjectPath(target string) (string, error) {
	absPath, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	absPath = filepath.Clean(absPath)

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("directory not found: %s", absPath)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(homeDir, absPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("project path must be under your home directory")
	}

	return absPath, nil
}

func resolveStartConfig(cmd *cobra.Command, opts rootOpts, projectPath string) (config.Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config.Config{}, err
	}

	globalPath := opts.Config
	if globalPath == "" {
		globalPath = filepath.Join(homeDir, ".config", "havn", "config.toml")
	}
	projectConfigPath := filepath.Join(projectPath, ".havn", "config.toml")

	global, err := config.LoadFile(globalPath)
	if err != nil {
		return config.Config{}, err
	}
	project, err := config.LoadFile(projectConfigPath)
	if err != nil {
		return config.Config{}, err
	}

	flagOv := config.Overrides{}
	if cmd.Flags().Changed("shell") {
		flagOv.Shell = &opts.Shell
	}
	if cmd.Flags().Changed("env") {
		flagOv.Env = &opts.Env
	}
	if cmd.Flags().Changed("cpus") {
		flagOv.CPUs = &opts.CPUs
	}
	if cmd.Flags().Changed("memory") {
		flagOv.Memory = &opts.Memory
	}
	if cmd.Flags().Changed("image") {
		flagOv.Image = &opts.Image
	}
	if cmd.Flags().Changed("port") {
		sshPort := opts.Port + ":22"
		flagOv.SSHPort = &sshPort
	}

	cfg, src := config.Resolve(global, project, config.EnvOverrides(), flagOv)

	flakePath := filepath.Join(projectPath, ".havn", "flake.nix")
	if _, err := os.Stat(flakePath); err == nil {
		cfg.Env = config.ResolveFlake(cfg, src, true)
	} else {
		cfg.Env = config.ResolveFlake(cfg, src, false)
	}

	if opts.NoDolt {
		cfg.Dolt.Enabled = false
	}

	if err := config.Validate(cfg); err != nil {
		return config.Config{}, err
	}

	return cfg, nil
}
