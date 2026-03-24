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
	_, _, err := executeCommand("dolt", "drop", "mydb")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt drop:")
}

func TestDoltDropCommand_RequiresName(t *testing.T) {
	_, _, err := executeCommand("dolt", "drop")

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestDoltDropCommand_HasYesFlag(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	dropCmd, _, err := root.Find([]string{"dolt", "drop"})

	require.NoError(t, err)
	f := dropCmd.Flags().Lookup("yes")
	require.NotNil(t, f, "--yes flag should exist")
	assert.Equal(t, "false", f.DefValue)
}

func TestDoltConnectCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "connect")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt connect:")
}

func TestDoltImportCommand_RequiresPath(t *testing.T) {
	_, _, err := executeCommand("dolt", "import")

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestDoltImportCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "import", "/some/path")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt import:")
}

func TestDoltExportCommand_RequiresName(t *testing.T) {
	_, _, err := executeCommand("dolt", "export")

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestDoltExportCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "export", "mydb")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt export:")
}
