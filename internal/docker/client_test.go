package docker_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/docker"
)

func TestNewClient_ReturnsNonNilClient(t *testing.T) {
	c, err := docker.NewClient()
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestPing_DaemonUnreachable(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	err = c.Ping(context.Background())

	var daemonErr *docker.DaemonUnreachableError
	assert.ErrorAs(t, err, &daemonErr)
	assert.Equal(t, "tcp://localhost:0", daemonErr.Host)
}

func TestInfo_DaemonUnreachable(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	_, err = c.Info(context.Background())

	var daemonErr *docker.DaemonUnreachableError
	assert.ErrorAs(t, err, &daemonErr)
	assert.Equal(t, "tcp://localhost:0", daemonErr.Host)
}

func TestPing_LogsStructuredOperationWithInjectedLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	c, err := docker.NewClientWithHostAndLogger("tcp://localhost:0", logger)
	require.NoError(t, err)

	err = c.Ping(context.Background())
	require.Error(t, err)

	assert.Contains(t, buf.String(), `"component":"docker"`)
	assert.Contains(t, buf.String(), `"operation":"ping"`)
}
