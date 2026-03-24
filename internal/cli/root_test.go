package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
)

func TestNewRoot_ReturnsCommand(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	require.NotNil(t, root)
	assert.Equal(t, "havn", root.Use)
}

func TestNewRoot_SilencesCobraOutput(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	assert.True(t, root.SilenceErrors)
	assert.True(t, root.SilenceUsage)
}

func TestExecute_ReturnsZero(t *testing.T) {
	code := cli.Execute()

	assert.Equal(t, 0, code)
}

func TestNewRoot_DefaultRunPrintsHavn(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	stdout := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetArgs([]string{})

	err := root.Execute()

	require.NoError(t, err)
	assert.Equal(t, "havn\n", stdout.String())
}
