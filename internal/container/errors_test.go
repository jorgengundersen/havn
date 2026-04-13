package container_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/container"
)

func TestNotFoundError_TypedError(t *testing.T) {
	err := &container.NotFoundError{Name: "havn-user-api"}

	assert.Equal(t, "container_not_found", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-user-api"}, err.ErrorDetails())
}

func TestNetworkNotFoundError_TypedError(t *testing.T) {
	err := &container.NetworkNotFoundError{Name: "havn-net"}

	assert.Equal(t, "network_not_found", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-net"}, err.ErrorDetails())
}

func TestImageNotFoundError_TypedError(t *testing.T) {
	err := &container.ImageNotFoundError{Name: "havn-base:latest"}

	assert.Equal(t, "image_not_found", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-base:latest"}, err.ErrorDetails())
}
