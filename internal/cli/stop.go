package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/container"
)

type stopOpts struct {
	All bool
}

func newStopCmd(backend container.StopBackend) *cobra.Command {
	var opts stopOpts

	cmd := &cobra.Command{
		Use:   "stop [name|path]",
		Short: "Stop havn container(s)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if backend == nil {
				return fmt.Errorf("havn stop: %w", ErrNotImplemented)
			}

			out := commandOutput(cmd)
			if opts.All {
				result, err := container.StopAll(cmd.Context(), backend)
				if err != nil {
					return fmt.Errorf("havn stop --all: %w", err)
				}

				for _, name := range result.Stopped {
					out.Status("Stopped " + name)
				}
				for _, failure := range result.Failed {
					out.Status(fmt.Sprintf("Failed to stop %s: %s", failure.Name, failure.Err))
				}

				message := fmt.Sprintf("%d stopped, %d failed", len(result.Stopped), len(result.Failed))
				out.Status(message)
				if out.IsJSON() {
					status := "ok"
					if len(result.Failed) > 0 {
						status = "error"
					}
					if err := out.DataJSON(map[string]any{
						"status":  status,
						"message": message,
					}); err != nil {
						return err
					}
					if len(result.Failed) > 0 {
						return &ExitError{Code: 1, Err: fmt.Errorf("%s", message)}
					}
					return nil
				}
				if len(result.Failed) > 0 {
					return &ExitError{Code: 1, Err: fmt.Errorf("%s", message)}
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("havn stop: requires a container name/path or --all")
			}

			target := strings.TrimSpace(args[0])
			if target == "" {
				return fmt.Errorf("havn stop: requires a non-empty container name/path")
			}

			containerName, err := container.Stop(cmd.Context(), backend, target)
			if err != nil {
				return fmt.Errorf("havn stop: %w", err)
			}

			out.Status("Stopped " + containerName)
			if out.IsJSON() {
				return out.DataJSON(map[string]any{
					"status":    "ok",
					"message":   "container stopped",
					"container": containerName,
				})
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "stop all havn containers")

	return cmd
}
