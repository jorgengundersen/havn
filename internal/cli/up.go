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
		Long: "Start or reuse a project container without interactive attach.\n\n" +
			"`havn up` defaults to lifecycle-only startup. Use `--validate` to run required startup validation, " +
			"or `--prepare` to run validation plus optional startup preparation.",
		Example: "  havn up .\n" +
			"  havn up --validate .\n" +
			"  havn up --prepare .\n" +
			"  havn enter .",
		Args: cobra.MaximumNArgs(1),
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
			checkMode := startupCheckModeForUp(validate, prepare)
			startupTelemetry := container.NewStartupCheckTelemetry()
			_, err = startService.StartOrAttach(cmd.Context(), cfg, projectCtx.Path, out.Status, container.StartOptions{
				VerboseStartup:        verbose,
				Mode:                  container.StartupModeNoAttach,
				StartupChecks:         checkMode,
				StartupCheckTelemetry: startupTelemetry,
			})
			if err != nil {
				return fmt.Errorf("%s: %w", upCommandScope(checkMode), err)
			}

			containerName, err := projectCtx.ContainerName()
			if err != nil {
				return fmt.Errorf("havn up: %w", err)
			}
			out.Status(fmt.Sprintf("Container %s is running for project %s", containerName, projectCtx.Path))
			if out.IsJSON() {
				phaseSummary := startupCheckJSONPhaseSummary(startupTelemetry.Events())
				return out.DataJSON(map[string]any{
					"status":               "ok",
					"message":              "container running",
					"container":            string(containerName),
					"project_path":         projectCtx.Path,
					"startup_checks":       startupCheckModeLabel(checkMode),
					"startup_check_phases": phaseSummary,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Env, "env", "", "Nix flake ref for dev environment")
	cmd.Flags().IntVar(&opts.CPUs, "cpus", 0, "CPU limit")
	cmd.Flags().StringVar(&opts.Memory, "memory", "", "memory limit")
	cmd.Flags().StringVar(&opts.Port, "port", "", "SSH-only host port mapping (<host>:22); use config ports for app services")
	cmd.Flags().BoolVar(&opts.NoDolt, "no-dolt", false, "skip Dolt server")
	cmd.Flags().StringVar(&opts.Image, "image", "", "override base image")
	cmd.Flags().BoolVar(&validate, "validate", false, "run required startup validation")
	cmd.Flags().BoolVar(&prepare, "prepare", false, "run startup validation and optional preparation (implies --validate)")

	return cmd
}

func startupCheckJSONPhaseSummary(events []container.StartupCheckPhaseEvent) []map[string]any {
	summary := make([]map[string]any, 0, len(events))
	for _, event := range events {
		if event.Outcome == container.StartupCheckPhaseOutcomeStart {
			continue
		}
		summary = append(summary, map[string]any{
			"phase":       string(event.Phase),
			"outcome":     string(event.Outcome),
			"duration_ms": event.Duration.Milliseconds(),
		})
	}
	return summary
}

func upCommandScope(mode container.StartupCheckMode) string {
	switch mode {
	case container.StartupCheckValidate:
		return "havn up --validate"
	case container.StartupCheckPrepare:
		return "havn up --prepare"
	default:
		return "havn up"
	}
}

func startupCheckModeLabel(mode container.StartupCheckMode) string {
	switch mode {
	case container.StartupCheckValidate:
		return "validate"
	case container.StartupCheckPrepare:
		return "prepare"
	default:
		return "default"
	}
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
