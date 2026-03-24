package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type doctorOpts struct {
	All bool
}

func newDoctorCmd() *cobra.Command {
	var opts doctorOpts

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose environment health",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn doctor: %w", ErrNotImplemented)
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "run all diagnostic checks")

	return cmd
}
