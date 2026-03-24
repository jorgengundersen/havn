package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDoltCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dolt",
		Short: "Manage shared Dolt server",
	}

	cmd.AddCommand(
		newDoltStartCmd(),
		newDoltStopCmd(),
		newDoltStatusCmd(),
		newDoltDatabasesCmd(),
		newDoltDropCmd(),
		newDoltConnectCmd(),
		newDoltImportCmd(),
		newDoltExportCmd(),
	)

	return cmd
}

func newDoltStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start shared Dolt server",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn dolt start: %w", ErrNotImplemented)
		},
	}
}

func newDoltStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop shared Dolt server",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn dolt stop: %w", ErrNotImplemented)
		},
	}
}

func newDoltStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Dolt server status",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn dolt status: %w", ErrNotImplemented)
		},
	}
}

func newDoltDatabasesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "databases",
		Short: "List databases",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn dolt databases: %w", ErrNotImplemented)
		},
	}
}

type doltDropOpts struct {
	Yes bool
}

func newDoltDropCmd() *cobra.Command {
	var opts doltDropOpts

	cmd := &cobra.Command{
		Use:   "drop <name>",
		Short: "Drop a project database",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn dolt drop: %w", ErrNotImplemented)
		},
	}

	cmd.Flags().BoolVar(&opts.Yes, "yes", false, "confirm database drop")

	return cmd
}

func newDoltConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect",
		Short: "Open SQL shell",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn dolt connect: %w", ErrNotImplemented)
		},
	}
}

func newDoltImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Import local database",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn dolt import: %w", ErrNotImplemented)
		},
	}
}

func newDoltExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export <name>",
		Short: "Export database",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("havn dolt export: %w", ErrNotImplemented)
		},
	}
}
