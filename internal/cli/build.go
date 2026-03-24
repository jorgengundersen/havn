package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build base image",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn build: %w", ErrNotImplemented)
		},
	}
}
