package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/container"
)

func newUpCmd(startService StartService) *cobra.Command {
	var opts rootOpts
	var validate bool
	var prepare bool

	cmd := &cobra.Command{
		Use:   "up [path]",
		Short: "Start or reuse container without attaching",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if startService == nil {
				return fmt.Errorf("havn up: %w", ErrNotImplemented)
			}

			target := "."
			if len(args) == 1 {
				target = args[0]
			}

			projectCtx, err := projectContextFromStartupTarget(target)
			if err != nil {
				return err
			}

			cfg, err := resolveStartConfig(cmd, opts, projectCtx)
			if err != nil {
				return err
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			out := commandOutput(cmd)
			_, err = startService.StartOrAttach(cmd.Context(), cfg, projectCtx.Path, out.Status, container.StartOptions{
				VerboseStartup: verbose,
				Mode:           container.StartupModeNoAttach,
				StartupChecks:  startupCheckModeForUp(validate, prepare),
			})
			if err != nil {
				return fmt.Errorf("havn up: %w", err)
			}

			containerName, err := projectCtx.ContainerName()
			if err != nil {
				return fmt.Errorf("havn up: %w", err)
			}
			out.Status(fmt.Sprintf("Container %s is running for project %s", containerName, projectCtx.Path))

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Env, "env", "", "Nix flake ref for dev environment")
	cmd.Flags().IntVar(&opts.CPUs, "cpus", 0, "CPU limit")
	cmd.Flags().StringVar(&opts.Memory, "memory", "", "memory limit")
	cmd.Flags().StringVar(&opts.Port, "port", "", "SSH port mapping")
	cmd.Flags().BoolVar(&opts.NoDolt, "no-dolt", false, "skip Dolt server")
	cmd.Flags().StringVar(&opts.Image, "image", "", "override base image")
	cmd.Flags().BoolVar(&validate, "validate", false, "run required startup validation")
	cmd.Flags().BoolVar(&prepare, "prepare", false, "run startup validation and optional preparation")

	return cmd
}

func startupCheckModeForUp(validate, prepare bool) container.StartupCheckMode {
	if prepare {
		return container.StartupCheckPrepare
	}
	if validate {
		return container.StartupCheckValidate
	}
	return container.StartupCheckDefault
}
