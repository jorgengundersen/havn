package doctor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/doctor"
)

func TestContainerChecks_AllChecksForSingleContainer(t *testing.T) {
	backend := newFakeBackend()
	cfg := config.Default()
	cfg.Dolt.Enabled = true

	checks := doctor.ContainerChecks(backend, cfg, "havn-user-myproject", "/home/devuser/project", true)

	// Should have all 7 checks: nix_store, nix_devshell, project_mount, config_mounts, ssh_agent, dolt_connectivity, beads_health
	require.Len(t, checks, 7)
	assert.Equal(t, "nix_store", checks[0].ID())
	assert.Equal(t, "nix_devshell", checks[1].ID())
	assert.Equal(t, "project_mount", checks[2].ID())
	assert.Equal(t, "config_mounts", checks[3].ID())
	assert.Equal(t, "ssh_agent", checks[4].ID())
	assert.Equal(t, "dolt_connectivity", checks[5].ID())
	assert.Equal(t, "beads_health", checks[6].ID())

	for _, c := range checks {
		assert.Equal(t, "container", c.Tier())
		assert.Equal(t, "havn-user-myproject", c.Container())
	}
}

func TestContainerChecks_DoltDisabledSkipsConnectivity(t *testing.T) {
	backend := newFakeBackend()
	cfg := config.Default()
	cfg.Dolt.Enabled = false

	checks := doctor.ContainerChecks(backend, cfg, "havn-user-myproject", "/home/devuser/project", false)

	// All 7 checks are still created (dolt_connectivity will skip at runtime)
	require.Len(t, checks, 7)
}
