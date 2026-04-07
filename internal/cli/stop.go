package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type stopOpts struct {
	All bool
}

func newStopCmd() *cobra.Command {
	var opts stopOpts

	cmd := &cobra.Command{
		Use:   "stop [name|path]",
		Short: "Stop havn container(s)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn stop: %w", ErrNotImplemented)
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "stop all havn containers")

	return cmd
}
