package doctor

import (
	"os"
	"path/filepath"

	"github.com/jorgengundersen/havn/internal/config"
)

// HostChecks builds the tier-1 check list from configuration.
func HostChecks(backend Backend, cfg config.Config, projectConfigPath string) []Check {
	homeDir, _ := os.UserHomeDir()
	globalConfigPath := filepath.Join(homeDir, ".config", "havn", "config.toml")

	volumeNames := []string{cfg.Volumes.Nix, cfg.Volumes.Data, cfg.Volumes.Cache, cfg.Volumes.State}
	if cfg.Dolt.Enabled {
		volumeNames = append(volumeNames, "havn-dolt-data", "havn-dolt-config")
	}

	checks := []Check{
		NewDockerDaemonCheck(backend),
		NewBaseImageCheck(backend, cfg.Image),
		NewNetworkCheck(backend, cfg.Network),
		NewVolumesCheck(backend, volumeNames),
		NewGlobalConfigCheck(globalConfigPath),
		NewProjectConfigCheck(projectConfigPath, cfg),
		NewDoltServerCheck(backend, cfg.Dolt.Enabled),
		NewDoltDatabaseCheck(backend, cfg.Dolt.Enabled, cfg.Dolt.Database),
	}

	return checks
}
