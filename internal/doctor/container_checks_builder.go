package doctor

import (
	"github.com/jorgengundersen/havn/internal/config"
)

// ContainerChecks builds the tier-2 check list for a single container.
func ContainerChecks(backend Backend, cfg config.Config, container, projectPath string, beadsExists bool) []Check {
	return []Check{
		NewNixStoreCheck(backend, container),
		NewNixDevshellCheck(backend, container, cfg.Env, cfg.Shell),
		NewProjectMountCheck(backend, container, projectPath),
		NewConfigMountsCheck(backend, container, cfg.Mounts.Config),
		NewSSHAgentCheck(backend, container),
		NewDoltConnectivityCheck(backend, container, cfg.Network, cfg.Dolt.Enabled),
		NewBeadsHealthCheck(backend, container, beadsExists),
	}
}
