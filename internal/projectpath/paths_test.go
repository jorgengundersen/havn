package projectpath_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
