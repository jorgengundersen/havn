package doctor

import (
	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/mount"
)

// ContainerChecks builds the tier-2 check list for a single container.
func ContainerChecks(backend Backend, cfg config.Config, container, projectPath, sshAuthSock string, configMounts []mount.Spec, beadsExists bool) []Check {
	configExpectations := make([]ConfigMountExpectation, 0, len(configMounts))
	for _, m := range configMounts {
		configExpectations = append(configExpectations, ConfigMountExpectation{Target: m.Target, ReadOnly: m.ReadOnly})
	}

	return []Check{
		NewNixStoreCheck(backend, container),
		NewNixDevshellCheck(backend, container, cfg.Env, cfg.Shell),
		NewProjectMountCheck(backend, container, projectPath),
		NewConfigMountsCheck(backend, container, configExpectations),
		NewSSHAgentCheck(backend, container, sshAuthSock),
		NewDoltConnectivityCheck(backend, container, cfg.Network, cfg.Dolt.Enabled),
		NewBeadsHealthCheck(backend, container, beadsExists),
	}
}
