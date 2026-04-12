package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/dolt"
)

func newDoltCmd(manager *dolt.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dolt",
		Short: "Manage shared Dolt server",
	}

	cmd.AddCommand(
		newDoltStartCmd(manager),
		newDoltStopCmd(manager),
		newDoltStatusCmd(manager),
		newDoltDatabasesCmd(manager),
		newDoltDropCmd(manager),
		newDoltConnectCmd(manager),
		newDoltImportCmd(),
		newDoltExportCmd(),
	)

	return cmd
}

func newDoltStartCmd(manager *dolt.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start shared Dolt server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if manager == nil {
				return fmt.Errorf("havn dolt start: %w", ErrNotImplemented)
			}

			out := commandOutput(cmd)
			out.Status("Starting shared Dolt server...")
			if err := manager.Start(cmd.Context(), config.Default()); err != nil {
				return fmt.Errorf("havn dolt start: %w", err)
			}

			if out.IsJSON() {
				return out.DataJSON(map[string]string{"status": "ok", "message": "dolt server started"})
			}

			return nil
		},
	}
}

func newDoltStopCmd(manager *dolt.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop shared Dolt server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if manager == nil {
				return fmt.Errorf("havn dolt stop: %w", ErrNotImplemented)
			}

			out := commandOutput(cmd)
			out.Status("Stopping shared Dolt server...")
			if err := manager.Stop(cmd.Context()); err != nil {
				return fmt.Errorf("havn dolt stop: %w", err)
			}

			if out.IsJSON() {
				return out.DataJSON(map[string]string{"status": "ok", "message": "dolt server stopped"})
			}

			return nil
		},
	}
}

func newDoltStatusCmd(manager *dolt.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Dolt server status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if manager == nil {
				return fmt.Errorf("havn dolt status: %w", ErrNotImplemented)
			}

			out := commandOutput(cmd)
			status, err := manager.Status(cmd.Context())
			if err != nil {
				return fmt.Errorf("havn dolt status: %w", err)
			}

			if out.IsJSON() {
				return out.DataJSON(status)
			}

			if !status.Running {
				out.Data("Dolt server is not running")
				return nil
			}

			out.Data(fmt.Sprintf("Dolt server is running (%s)", status.Container))
			out.Data(fmt.Sprintf("Image: %s", status.Image))
			out.Data(fmt.Sprintf("Network: %s", status.Network))
			out.Data(fmt.Sprintf("Port: %d", status.Port))
			return nil
		},
	}
}

func newDoltDatabasesCmd(manager *dolt.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "databases",
		Short: "List databases",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if manager == nil {
				return fmt.Errorf("havn dolt databases: %w", ErrNotImplemented)
			}

			out := commandOutput(cmd)
			databases, err := manager.Databases(cmd.Context())
			if err != nil {
				return fmt.Errorf("havn dolt databases: %w", err)
			}

			if out.IsJSON() {
				return out.DataJSON(databases)
			}

			for _, name := range databases {
				out.Data(name)
			}

			return nil
		},
	}
}

type doltDropOpts struct {
	Yes bool
}

func newDoltDropCmd(manager *dolt.Manager) *cobra.Command {
	var opts doltDropOpts

	cmd := &cobra.Command{
		Use:   "drop <name>",
		Short: "Drop a project database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if manager == nil {
				return fmt.Errorf("havn dolt drop: %w", ErrNotImplemented)
			}
			if !opts.Yes {
				return errors.New("havn dolt drop requires --yes")
			}

			name := args[0]
			out := commandOutput(cmd)
			out.Status(fmt.Sprintf("Dropping database %s...", name))
			if err := manager.Drop(cmd.Context(), name); err != nil {
				return fmt.Errorf("havn dolt drop: %w", err)
			}

			if out.IsJSON() {
				return out.DataJSON(map[string]string{"status": "ok", "message": "database dropped"})
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.Yes, "yes", false, "confirm database drop")

	return cmd
}

func newDoltConnectCmd(manager *dolt.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "connect",
		Short: "Open SQL shell",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if manager == nil {
				return fmt.Errorf("havn dolt connect: %w", ErrNotImplemented)
			}

			out := commandOutput(cmd)
			out.Status("Connecting to shared Dolt SQL shell...")
			if err := manager.Connect(cmd.Context()); err != nil {
				return fmt.Errorf("havn dolt connect: %w", err)
			}

			return nil
		},
	}
}

func commandOutput(cmd *cobra.Command) *Output {
	jsonMode, _ := cmd.Flags().GetBool("json")
	verbose, _ := cmd.Flags().GetBool("verbose")
	return NewOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, verbose)
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
