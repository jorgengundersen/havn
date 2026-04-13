package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/doctor"
)

var (
	errDoctorWarnings = errors.New("doctor found warnings")
	errDoctorErrors   = errors.New("doctor found errors")
)

type doctorOpts struct {
	All bool
}

func newDoctorCmd(backend doctor.Backend) *cobra.Command {
	var opts doctorOpts

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose environment health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			jsonMode, _ := cmd.Flags().GetBool("json")
			verbose, _ := cmd.Flags().GetBool("verbose")
			out := NewOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, verbose)
			ctx := cmd.Context()

			cfg := config.Default()
			projectConfigPath := ".havn/config.toml"

			checks := doctor.HostChecks(backend, cfg, projectConfigPath)

			containers := resolveContainers(ctx, backend, opts.All)
			cwd, _ := os.Getwd()
			beadsExists := dirExists(filepath.Join(cwd, ".beads"))
			for _, name := range containers {
				cc := doctor.ContainerChecks(backend, cfg, name, cwd, beadsExists)
				checks = append(checks, cc...)
			}

			runner := doctor.NewRunner(checks)
			report := runner.Run(ctx)
			report = addContainerTierSkipIfNeeded(report, containers)

			return outputReport(out, report)
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "check all running havn containers")

	return cmd
}

func resolveContainers(ctx context.Context, backend doctor.Backend, all bool) []string {
	labels := map[string]string{"managed-by": "havn"}
	if all {
		containers, err := backend.ListContainers(ctx, labels)
		if err != nil {
			return nil
		}
		return containers
	}

	// Default: find the container for the current directory.
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	expectedName := deriveContainerName(cwd)
	if expectedName == "" {
		return nil
	}

	// Check if it's running.
	info, found, err := backend.ContainerInspect(ctx, expectedName)
	if err != nil || !found || !info.Running {
		return nil
	}
	return []string{expectedName}
}

// deriveContainerName produces "havn-<parent>-<project>" from an absolute path.
// e.g. ~/Repos/github.com/user/api -> havn-user-api
func deriveContainerName(cwd string) string {
	project := filepath.Base(cwd)
	parent := filepath.Base(filepath.Dir(cwd))
	if project == "" || parent == "" || project == "." || parent == "." {
		return ""
	}
	return "havn-" + parent + "-" + project
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

func addContainerTierSkipIfNeeded(report doctor.Report, containers []string) doctor.Report {
	if len(containers) > 0 {
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
