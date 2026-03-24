package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List running havn containers",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn list: %w", ErrNotImplemented)
		},
	}
}
