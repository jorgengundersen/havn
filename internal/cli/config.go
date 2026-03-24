package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage havn configuration",
	}

	cmd.AddCommand(newConfigShowCmd())

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show effective merged config",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn config show: %w", ErrNotImplemented)
		},
	}
}
