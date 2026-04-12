package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/container"
)

func newListCmd(backend container.Backend) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List running havn containers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if backend == nil {
				return fmt.Errorf("havn list: %w", ErrNotImplemented)
			}

			containers, err := container.List(cmd.Context(), backend)
			if err != nil {
				return fmt.Errorf("havn list: %w", err)
			}

			out := commandOutput(cmd)
			if out.IsJSON() {
				return out.DataJSON(containers)
			}

			for _, c := range containers {
				out.Data(strings.Join([]string{
					string(c.Name),
					c.Path,
					c.Image,
					c.Status,
					c.Shell,
					fmt.Sprintf("%d", c.CPUs),
					c.Memory,
					fmt.Sprintf("%t", c.Dolt),
				}, "\t"))
			}

			return nil
		},
	}
}
