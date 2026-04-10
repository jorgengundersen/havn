package docker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/docker"
)

func TestDaemonUnreachableError_ImplementsError(t *testing.T) {
	err := &docker.DaemonUnreachableError{Host: "unix:///var/run/docker.sock"}

	var target error = err
	assert.Equal(t, `daemon unreachable at "unix:///var/run/docker.sock"`, target.Error())
}

func TestDaemonUnreachableError_TypedError(t *testing.T) {
	err := &docker.DaemonUnreachableError{Host: "unix:///var/run/docker.sock"}

	assert.Equal(t, "daemon_unreachable", err.ErrorType())
	assert.Equal(t, map[string]any{"host": "unix:///var/run/docker.sock"}, err.ErrorDetails())
}

func TestContainerNotFoundError_ImplementsError(t *testing.T) {
	err := &docker.ContainerNotFoundError{Name: "havn-user-api"}

	var target error = err
	assert.Equal(t, `container "havn-user-api" not found`, target.Error())
}

func TestContainerNotFoundError_TypedError(t *testing.T) {
	err := &docker.ContainerNotFoundError{Name: "havn-user-api"}

	assert.Equal(t, "container_not_found", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-user-api"}, err.ErrorDetails())
}

func TestImageNotFoundError_ImplementsError(t *testing.T) {
	err := &docker.ImageNotFoundError{Name: "havn-base:latest"}

	var target error = err
	assert.Equal(t, `image "havn-base:latest" not found`, target.Error())
}

func TestImageNotFoundError_TypedError(t *testing.T) {
	err := &docker.ImageNotFoundError{Name: "havn-base:latest"}

	assert.Equal(t, "image_not_found", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-base:latest"}, err.ErrorDetails())
}

func TestNetworkNotFoundError_ImplementsError(t *testing.T) {
	err := &docker.NetworkNotFoundError{Name: "havn-net"}

	var target error = err
	assert.Equal(t, `network "havn-net" not found`, target.Error())
}

func TestNetworkNotFoundError_TypedError(t *testing.T) {
	err := &docker.NetworkNotFoundError{Name: "havn-net"}

	assert.Equal(t, "network_not_found", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-net"}, err.ErrorDetails())
}

func TestVolumeNotFoundError_ImplementsError(t *testing.T) {
	err := &docker.VolumeNotFoundError{Name: "havn-dolt-data"}

	var target error = err
	assert.Equal(t, `volume "havn-dolt-data" not found`, target.Error())
}

func TestVolumeNotFoundError_TypedError(t *testing.T) {
	err := &docker.VolumeNotFoundError{Name: "havn-dolt-data"}

	assert.Equal(t, "volume_not_found", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-dolt-data"}, err.ErrorDetails())
}
