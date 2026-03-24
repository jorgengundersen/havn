package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type stopOpts struct {
	All bool
	Yes bool
}

func newStopCmd() *cobra.Command {
	var opts stopOpts

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop havn container(s)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn stop: %w", ErrNotImplemented)
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "stop all havn containers")
	cmd.Flags().BoolVar(&opts.Yes, "yes", false, "skip confirmation prompt")

	return cmd
}
