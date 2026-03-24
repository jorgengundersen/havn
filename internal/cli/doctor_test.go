package cli_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
)

func TestDoctorCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("doctor")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn doctor:")
}

func TestDoctorCommand_HasAllFlag(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	doctorCmd, _, err := root.Find([]string{"doctor"})

	require.NoError(t, err)
	f := doctorCmd.Flags().Lookup("all")
	require.NotNil(t, f, "--all flag should exist")
	assert.Equal(t, "false", f.DefValue)
}
