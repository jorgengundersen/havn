package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVolumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Manage havn volumes",
	}

	cmd.AddCommand(newVolumeListCmd())

	return cmd
}

func newVolumeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List havn volumes",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn volume list: %w", ErrNotImplemented)
		},
	}
}
