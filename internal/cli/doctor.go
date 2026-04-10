package cli

import (
	"errors"

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

			cfg := config.Default()
			projectConfigPath := ".havn/config.toml"

			checks := doctor.HostChecks(backend, cfg, projectConfigPath)
			runner := doctor.NewRunner(checks)
			report := runner.Run(cmd.Context())

			return outputReport(out, report)
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "run all diagnostic checks")

	return cmd
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

func exitCodeFromReport(report doctor.Report) error {
	switch report.Status {
	case doctor.StatusWarn:
		return &ExitError{Code: 1, Err: errDoctorWarnings}
	case doctor.StatusError:
		return &ExitError{Code: 2, Err: errDoctorErrors}
	default:
		return nil
	}
}
