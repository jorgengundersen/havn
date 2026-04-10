package docker_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/docker"
)

func TestCopyToContainer_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	err = c.CopyToContainer(context.Background(), "nonexistent", "/dst", strings.NewReader(""))

	assert.Error(t, err)
}

func TestCopyToContainer_WrapsErrorWithContext(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	err = c.CopyToContainer(context.Background(), "test-ctr", "/dst", strings.NewReader(""))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker copy to container")
}

func TestCopyFromContainer_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	rc, err := c.CopyFromContainer(context.Background(), "nonexistent", "/src")

	assert.Error(t, err)
	assert.Nil(t, rc)
}

func TestCopyFromContainer_WrapsErrorWithContext(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	rc, err := c.CopyFromContainer(context.Background(), "test-ctr", "/src")

	require.Error(t, err)
	assert.Nil(t, rc)
	assert.Contains(t, err.Error(), "docker copy from container")
}

func TestCopyToContainer_RespectsContextCancellation(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = c.CopyToContainer(ctx, "test-ctr", "/dst", strings.NewReader(""))

	assert.Error(t, err)
}

func TestCopyFromContainer_RespectsContextCancellation(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rc, err := c.CopyFromContainer(ctx, "test-ctr", "/src")

	assert.Error(t, err)
	assert.Nil(t, rc)
}
