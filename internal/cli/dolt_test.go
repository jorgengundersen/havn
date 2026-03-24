package cli_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
)

func TestDoltCommand_PrintsHelp(t *testing.T) {
	_, _, err := executeCommand("dolt")

	require.NoError(t, err)
}

func TestDoltStartCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "start")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt start:")
}

func TestDoltStopCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "stop")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt stop:")
}

func TestDoltStatusCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "status")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt status:")
}

func TestDoltDatabasesCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "databases")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt databases:")
}

func TestDoltDropCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "drop")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt drop:")
}

func TestDoltConnectCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "connect")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt connect:")
}

func TestDoltImportCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "import")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt import:")
}

func TestDoltExportCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "export")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt export:")
}
