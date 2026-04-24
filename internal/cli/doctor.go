package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/doctor"
	"github.com/jorgengundersen/havn/internal/mount"
)

var (
	errDoctorWarnings = errors.New("doctor found warnings")
	errDoctorErrors   = errors.New("doctor found errors")
)

type doctorOpts struct {
	All  bool
	Dolt bool
}

type doctorContainerTarget struct {
	Name       string
	Project    string
	BeadsExist bool
}

func newDoctorCmd(backend doctor.Backend) *cobra.Command {
	var opts doctorOpts

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose environment health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			jsonMode, _ := cmd.Flags().GetBool("json")
			verbose, _ := cmd.Flags().GetBool("verbose")
			globalConfigPath, _ := cmd.Flags().GetString("config")
			out := NewOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, verbose)
			ctx := cmd.Context()
			projectCtx, err := projectContextFromWorkingDirForStartup()
			if err != nil {
				return err
			}

			projectPath := projectCtx.Path
			var effectiveValidationErr error
			hasEffectiveConfig := true
			orchestrator := newEffectiveConfigOrchestrator(globalConfigPath)
			cfg, err := orchestrator.Resolve(projectContext{Path: projectPath}, config.Overrides{})
			if err != nil {
				hasEffectiveConfig = false
				var validationErr *config.ValidationError
				if errors.As(err, &validationErr) {
					effectiveValidationErr = err
					hasEffectiveConfig = true
					cfg = config.Default()
				}
			}
			if opts.Dolt {
				cfg.Dolt.Enabled = true
				if cfg.Dolt.Database == "" {
					cfg.Dolt.Database = projectCtx.DefaultDoltDatabase()
				}
			}
			projectConfigPath := projectCtx.ProjectConfigPath()

			checks := doctor.HostChecks(backend, cfg, globalConfigPath, projectConfigPath, effectiveValidationErr, hasEffectiveConfig, opts.Dolt)
			targetConfigResolutionFailures := make([]doctor.ReportCheck, 0)

			targets := resolveContainerTargets(ctx, backend, opts.All, projectPath)
			for _, target := range targets {
				targetCfg, err := orchestrator.Resolve(projectContext{Path: target.Project}, config.Overrides{})
				if err != nil {
					targetConfigResolutionFailures = append(targetConfigResolutionFailures, doctor.ReportCheck{
						Tier:      "container",
						Container: target.Name,
						Name:      "project_config",
						Status:    doctor.StatusError,
						Message:   "Target project config resolution failed",
						Detail:    err.Error(),
					})
					continue
				}

				mountResult, err := resolveMountsForDoctor(targetCfg, target.Project)
				if err != nil {
					mountResult = mount.ResolveResult{}
				}

				cc := doctor.ContainerChecks(
					backend,
					targetCfg,
					target.Name,
					target.Project,
					mountResult.Env["SSH_AUTH_SOCK"],
					mountResult.ConfigMounts,
					target.BeadsExist,
					opts.Dolt,
				)
				checks = append(checks, cc...)
			}

			runner := doctor.NewRunner(checks)
			report := runner.Run(ctx)
			report = addContainerTierSkipIfNeeded(report, targets)
			report = addTargetConfigResolutionFailures(report, targetConfigResolutionFailures)

			return outputReport(out, report)
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "check all running havn containers")
	cmd.Flags().BoolVar(&opts.Dolt, "dolt", false, "run shared Dolt diagnostics regardless of project dolt.enabled (container scope still controlled by --all)")

	return cmd
}

func resolveMountsForDoctor(cfg config.Config, projectPath string) (mount.ResolveResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return mount.ResolveResult{}, err
	}

	return mount.Resolve(cfg, projectPath, homeDir, mount.ResolveOpts{
		Glob:        filepath.Glob,
		Exists:      pathExists,
		SSHAuthSock: os.Getenv("SSH_AUTH_SOCK"),
	})
}

func resolveContainerTargets(ctx context.Context, backend doctor.Backend, all bool, currentProjectPath string) []doctorContainerTarget {
	labels := map[string]string{"managed-by": "havn"}
	if all {
		containers, err := backend.ListContainers(ctx, labels)
		if err != nil {
			return nil
		}

		targets := make([]doctorContainerTarget, 0, len(containers))
		for _, containerName := range containers {
			info, found, err := backend.ContainerInspect(ctx, containerName)
			if err != nil || !found || !info.Running {
				continue
			}

			projectPath := strings.TrimSpace(info.Labels[container.LabelPath])
			if projectPath == "" {
				continue
			}

			cleanProjectPath := filepath.Clean(projectPath)
			targets = append(targets, doctorContainerTarget{
				Name:       containerName,
				Project:    cleanProjectPath,
				BeadsExist: dirExists(filepath.Join(cleanProjectPath, ".beads")),
			})
		}

		return targets
	}

	expectedName, err := (projectContext{Path: currentProjectPath}).ContainerName()
	if err != nil {
		return nil
	}

	info, found, err := backend.ContainerInspect(ctx, expectedName)
	if err != nil || !found || !info.Running {
		return nil
	}

	projectPath := strings.TrimSpace(info.Labels[container.LabelPath])
	if projectPath == "" {
		projectPath = currentProjectPath
	}
	cleanProjectPath := filepath.Clean(projectPath)

	return []doctorContainerTarget{{
		Name:       expectedName,
		Project:    cleanProjectPath,
		BeadsExist: dirExists(filepath.Join(cleanProjectPath, ".beads")),
	}}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func outputReport(out *Output, report doctor.Report) error {
	if out.IsJSON() {
		out.Data(doctor.FormatJSON(report))
	} else if out.verbose {
		out.Data(doctor.FormatVerbose(report))
	} else {
		out.Data(doctor.FormatHuman(report))
	}

	return exitCodeFromReport(report)
}

func addContainerTierSkipIfNeeded(report doctor.Report, targets []doctorContainerTarget) doctor.Report {
	if len(targets) > 0 {
		return report
	}

	report.Checks = append(report.Checks, doctor.ReportCheck{
		Tier:    "container",
		Name:    "container_tier",
		Status:  doctor.StatusSkip,
		Message: "No relevant running havn-managed project containers; tier 2 skipped",
	})

	return report
}

func addTargetConfigResolutionFailures(report doctor.Report, failures []doctor.ReportCheck) doctor.Report {
	if len(failures) == 0 {
		return report
	}

	report.Checks = append(report.Checks, failures...)
	report.Summary = doctor.Summary{}
	report.Status = doctor.StatusPass

	for _, check := range report.Checks {
		switch check.Status {
		case doctor.StatusPass:
			report.Summary.Passed++
		case doctor.StatusWarn:
			report.Summary.Warnings++
			if report.Status == doctor.StatusPass {
				report.Status = doctor.StatusWarn
			}
		case doctor.StatusError:
			report.Summary.Errors++
			report.Status = doctor.StatusError
		}
	}

	return report
}

func exitCodeFromReport(report doctor.Report) error {
	switch report.Status {
	case doctor.StatusWarn:
		return &ExitError{Code: 1, Err: errDoctorWarnings, SuppressOutput: true}
	case doctor.StatusError:
		return &ExitError{Code: 2, Err: errDoctorErrors, SuppressOutput: true}
	default:
		return nil
	}
}
