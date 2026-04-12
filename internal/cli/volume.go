package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/volume"
)

func newVolumeCmd(manager *volume.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Manage havn volumes",
	}

	cmd.AddCommand(newVolumeListCmd(manager))

	return cmd
}

func newVolumeListCmd(manager *volume.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List havn volumes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if manager == nil {
				return fmt.Errorf("havn volume list: %w", ErrNotImplemented)
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("havn volume list: %w", err)
			}

			cfg, err := loadEffectiveConfig(filepath.Clean(cwd))
			if err != nil {
				return fmt.Errorf("havn volume list: %w", err)
			}

			entries, err := manager.List(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("havn volume list: %w", err)
			}

			out := commandOutput(cmd)
			if out.IsJSON() {
				return out.DataJSON(entries)
			}

			for _, entry := range entries {
				state := "missing"
				if entry.Exists {
					state = "exists"
				}
				out.Data(strings.Join([]string{entry.Name, entry.Mount, state}, "\t"))
			}

			return nil
		},
	}
}
