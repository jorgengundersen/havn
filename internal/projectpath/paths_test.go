package projectpath_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/projectpath"
)

func TestResolve_MapsHostProjectPathUnderContainerHome(t *testing.T) {
	paths, err := projectpath.Resolve("/home/alice/work/api", "/home/alice")

	assert.NoError(t, err)
	assert.Equal(t, "/home/alice/work/api", paths.HostPath)
	assert.Equal(t, "/home/devuser/work/api", paths.ContainerPath)
}

func TestResolve_HostProjectPathEqualToHomeMapsToContainerHome(t *testing.T) {
	paths, err := projectpath.Resolve("/home/alice", "/home/alice")

	assert.NoError(t, err)
	assert.Equal(t, "/home/alice", paths.HostPath)
	assert.Equal(t, "/home/devuser", paths.ContainerPath)
}

func TestResolve_ProjectPathOutsideHomeReturnsActionableError(t *testing.T) {
	_, err := projectpath.Resolve("/srv/work/api", "/home/alice")

	assert.Error(t, err)
	assert.ErrorContains(t, err, "/srv/work/api")
	assert.ErrorContains(t, err, "/home/alice")
	var outsideErr *projectpath.OutsideHomeError
	assert.ErrorAs(t, err, &outsideErr)
}

func TestResolve_CanonicalizesSymlinkedHomeForBoundaryCheck(t *testing.T) {
	tmp := t.TempDir()
	realHome := filepath.Join(tmp, "real-home")
	linkHome := filepath.Join(tmp, "home-link")
	projectPath := filepath.Join(realHome, "work", "api")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))
	if err := os.Symlink(realHome, linkHome); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	paths, err := projectpath.Resolve(projectPath, linkHome)

	require.NoError(t, err)
	assert.Equal(t, projectPath, paths.HostPath)
	assert.Equal(t, "/home/devuser/work/api", paths.ContainerPath)
}
