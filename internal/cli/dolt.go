package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/dolt"
)

const migrationOwnershipBoundaryCode = "beads_migration_workflow"

const migrationOwnershipBoundaryMessage = "Ownership boundary [beads_migration_workflow]: migration semantics are owned by beads/Dolt workflows; havn manages shared Dolt infrastructure only."

const doltWarningPrefix = "WARNING: "

func newDoltCmd(manager *dolt.Manager, setup *dolt.Setup) *cobra.Command {
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
		newDoltImportCmd(manager, setup),
		newDoltExportCmd(manager),
	)

	return cmd
}

func newDoltStartCmd(manager *dolt.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start shared Dolt server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := ensureDoltManager("start", manager); err != nil {
				return err
			}

			cfg, err := resolveDoltConfigFromWorkingDir("start")
			if err != nil {
				return err
			}

			out := commandOutput(cmd)
			if err := startSharedDoltServerWithProgress(cmd.Context(), out, manager, cfg); err != nil {
				return doltCommandError("start", err)
			}

			return doltOKResponse(out, "Shared Dolt server started", "dolt server started")
		},
	}
}

func newDoltStopCmd(manager *dolt.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop shared Dolt server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := ensureDoltManager("stop", manager); err != nil {
				return err
			}

			out := commandOutput(cmd)
			out.Status("Stopping shared Dolt server...")
			if err := manager.Stop(cmd.Context()); err != nil {
				return doltCommandError("stop", err)
			}

			return doltOKResponse(out, "Shared Dolt server stopped", "dolt server stopped")
		},
	}
}

func newDoltStatusCmd(manager *dolt.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Dolt server status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := ensureDoltManager("status", manager); err != nil {
				return err
			}

			cfg, err := resolveDoltConfigFromWorkingDir("status")
			if err != nil {
				return err
			}

			out := commandOutput(cmd)
			status, err := manager.Status(cmd.Context())
			if err != nil {
				return doltCommandError("status", err)
			}

			if out.IsJSON() {
				return out.DataJSON(doltStatusPayload(status, cfg.Dolt.Port))
			}

			if !status.Running {
				out.Data("Dolt server is not running")
				out.Data(fmt.Sprintf("Configured SQL port: %d", cfg.Dolt.Port))
				out.Data("Runtime port verification is external (use docker inspect/docker port when needed)")
				return nil
			}

			out.Data(fmt.Sprintf("Dolt server is running (%s)", status.Container))
			out.Data(fmt.Sprintf("Configured SQL port: %d", cfg.Dolt.Port))
			out.Data("Runtime port verification is external (use docker inspect/docker port when needed)")
			out.Data(fmt.Sprintf("Image: %s", status.Image))
			out.Data(fmt.Sprintf("Network: %s", status.Network))
			return nil
		},
	}
}

func doltStatusPayload(status dolt.Status, configuredPort int) map[string]any {
	payload := map[string]any{"running": status.Running, "configured_sql_port": configuredPort}
	if !status.Running {
		return payload
	}

	payload["container"] = status.Container
	payload["image"] = status.Image
	payload["network"] = status.Network
	payload["managed_by_havn"] = status.ManagedByHavn

	return payload
}

func newDoltDatabasesCmd(manager *dolt.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "databases",
		Short: "List databases",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := ensureDoltManager("databases", manager); err != nil {
				return err
			}

			out := commandOutput(cmd)
			databases, err := manager.Databases(cmd.Context())
			if err != nil {
				return doltCommandError("databases", err)
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
			if err := ensureDoltManager("drop", manager); err != nil {
				return err
			}
			if !opts.Yes {
				return errors.New("havn dolt drop requires --yes")
			}

			name := args[0]
			out := commandOutput(cmd)
			out.Status(fmt.Sprintf("Dropping database %s...", name))
			if err := manager.Drop(cmd.Context(), name); err != nil {
				return doltCommandError("drop", err)
			}

			return doltOKResponse(out, fmt.Sprintf("Database %s dropped", name), "database dropped")
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
			if err := ensureDoltManager("connect", manager); err != nil {
				return err
			}

			out := commandOutput(cmd)
			out.Status("Connecting to shared Dolt SQL shell...")
			if err := manager.Connect(cmd.Context()); err != nil {
				return doltCommandError("connect", err)
			}

			if !out.IsJSON() {
				out.Status("Shared Dolt SQL shell session ended")
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

func ensureDoltManager(command string, manager *dolt.Manager) error {
	if manager == nil {
		return doltCommandError(command, ErrNotImplemented)
	}

	return nil
}

func doltCommandError(command string, err error) error {
	return fmt.Errorf("havn dolt %s: %w", command, err)
}

func resolveDoltConfigFromWorkingDir(command string) (config.Config, error) {
	projectCtx, err := projectContextFromWorkingDir()
	if err != nil {
		return config.Config{}, doltCommandError(command, err)
	}

	orchestrator := newEffectiveConfigOrchestrator("")
	cfg, err := orchestrator.Resolve(projectCtx, config.Overrides{})
	if err != nil {
		return config.Config{}, doltCommandError(command, err)
	}

	return cfg, nil
}

func resolveDoltConfigFromTarget(command, target string) (string, config.Config, error) {
	projectCtx, err := projectContextFromTarget(target)
	if err != nil {
		return "", config.Config{}, doltCommandError(command, err)
	}

	orchestrator := newEffectiveConfigOrchestrator("")
	cfg, err := orchestrator.Resolve(projectCtx, config.Overrides{})
	if err != nil {
		return "", config.Config{}, doltCommandError(command, err)
	}

	return projectCtx.Path, cfg, nil
}

func doltOKResponse(out *Output, humanMessage, jsonMessage string) error {
	if !out.IsJSON() {
		out.Status(humanMessage)
		return nil
	}

	return out.DataJSON(map[string]string{"status": "ok", "message": jsonMessage})
}

func newDoltImportCmd(manager *dolt.Manager, _ *dolt.Setup) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "import <path>",
		Short: "Import local database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ensureDoltManager("import", manager); err != nil {
				return err
			}

			projectPath, cfg, err := resolveDoltConfigFromTarget("import", args[0])
			if err != nil {
				return err
			}

			out := commandOutput(cmd)
			out.Status(fmt.Sprintf("Importing Dolt database from %s...", projectPath))

			if err := startSharedDoltServerWithProgress(cmd.Context(), out, manager, cfg); err != nil {
				return doltCommandError("import", err)
			}

			result, err := manager.Import(cmd.Context(), projectPath, cfg, force)
			if err != nil {
				return doltCommandError("import", err)
			}

			if result.Overwrote {
				out.Status(fmt.Sprintf("Overwriting existing database %s on shared server", result.DatabaseName))
			}

			for _, warning := range result.Warnings {
				out.Status(doltWarningMessage(warning))
			}

			out.Status(doltWarningMessage(migrationOwnershipBoundaryMessage))

			warnings := result.Warnings
			if warnings == nil {
				warnings = []string{}
			}

			if out.IsJSON() {
				return out.DataJSON(map[string]any{
					"status":             "ok",
					"message":            "database imported",
					"database":           result.DatabaseName,
					"path":               projectPath,
					"overwrote":          result.Overwrote,
					"warnings":           warnings,
					"ownership_boundary": migrationOwnershipBoundaryCode,
				})
			}

			out.Status(fmt.Sprintf("Database %s imported", result.DatabaseName))

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing database on shared server")

	return cmd
}

func startSharedDoltServerWithProgress(ctx context.Context, out *Output, manager *dolt.Manager, cfg config.Config) error {
	out.Status("Starting shared Dolt server...")

	imageAcquired := false
	if err := manager.StartWithProgress(ctx, cfg, func(event dolt.StartProgressEvent) {
		if out.IsJSON() {
			return
		}

		switch event.Stage {
		case dolt.StartProgressImageAcquisitionStarted:
			imageAcquired = true
			out.Status(fmt.Sprintf("Dolt image %s not present locally; acquiring image...", event.Image))
		case dolt.StartProgressStartupResumed:
			out.Status("Image acquisition complete; resuming shared Dolt startup...")
		}
	}); err != nil {
		if imageAcquired && !out.IsJSON() {
			out.Status("Shared Dolt startup failed during image acquisition path")
		}
		return err
	}

	if imageAcquired && !out.IsJSON() {
		out.Status("Shared Dolt startup completed after image acquisition")
	}

	return nil
}

type doltExportOpts struct {
	Dest string
}

func newDoltExportCmd(manager *dolt.Manager) *cobra.Command {
	var opts doltExportOpts

	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ensureDoltManager("export", manager); err != nil {
				return err
			}

			destPath, err := filepath.Abs(opts.Dest)
			if err != nil {
				return doltCommandError("export", err)
			}

			name := args[0]
			out := commandOutput(cmd)
			out.Status(fmt.Sprintf("Exporting Dolt database %s to %s...", name, destPath))

			if err := manager.Export(cmd.Context(), name, destPath); err != nil {
				return doltCommandError("export", err)
			}

			out.Status(doltWarningMessage(migrationOwnershipBoundaryMessage))

			if out.IsJSON() {
				return out.DataJSON(map[string]any{
					"status":             "ok",
					"message":            "database exported",
					"database":           name,
					"dest":               destPath,
					"ownership_boundary": migrationOwnershipBoundaryCode,
				})
			}

			out.Status(fmt.Sprintf("Database %s exported to %s", name, destPath))

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Dest, "dest", ".", "destination project directory")

	return cmd
}

func doltWarningMessage(message string) string {
	return doltWarningPrefix + message
}
