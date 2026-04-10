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
