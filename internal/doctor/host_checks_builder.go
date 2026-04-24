package doctor

import (
	"os"
	"path/filepath"

	"github.com/jorgengundersen/havn/internal/config"
)

// HostChecks builds the tier-1 check list from configuration.
func HostChecks(backend Backend, cfg config.Config, globalConfigPath, projectConfigPath string, effectiveValidationErr error, hasEffectiveConfig bool, explicitDolt bool) []Check {
	if globalConfigPath == "" {
		homeDir, _ := os.UserHomeDir()
		globalConfigPath = filepath.Join(homeDir, ".config", "havn", "config.toml")
	}

	checks := []Check{
		NewDockerDaemonCheck(backend),
		NewGlobalConfigCheck(globalConfigPath),
		NewProjectConfigCheck(projectConfigPath, effectiveValidationErr),
	}

	if !hasEffectiveConfig {
		return checks
	}

	volumeNames := []string{cfg.Volumes.Nix, cfg.Volumes.Data, cfg.Volumes.Cache, cfg.Volumes.State}
	doltChecksEnabled := cfg.Dolt.Enabled || explicitDolt
	if doltChecksEnabled {
		volumeNames = append(volumeNames, "havn-dolt-data", "havn-dolt-config")
	}

	doltImage := ""
	if explicitDolt {
		doltImage = cfg.Dolt.Image
	}

	checks = append(checks,
		NewBaseImageCheck(backend, cfg.Image),
		NewNetworkCheck(backend, cfg.Network),
		NewVolumesCheck(backend, volumeNames),
		NewDoltServerCheck(backend, doltChecksEnabled, doltImage),
		NewDoltDatabaseCheck(backend, doltChecksEnabled, cfg.Dolt.Database),
	)

	return checks
}
