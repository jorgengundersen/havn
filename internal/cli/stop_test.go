package cli_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
)

func TestStopCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("stop")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn stop:")
}

func TestStopCommand_HasAllFlag(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	stopCmd, _, err := root.Find([]string{"stop"})

	require.NoError(t, err)
	f := stopCmd.Flags().Lookup("all")
	require.NotNil(t, f, "--all flag should exist")
	assert.Equal(t, "false", f.DefValue)
}

func TestStopCommand_HasYesFlag(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	stopCmd, _, err := root.Find([]string{"stop"})

	require.NoError(t, err)
	f := stopCmd.Flags().Lookup("yes")
	require.NotNil(t, f, "--yes flag should exist")
	assert.Equal(t, "false", f.DefValue)
}
