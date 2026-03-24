package cli_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
)

func TestConfigCommand_PrintsHelp(t *testing.T) {
	_, _, err := executeCommand("config")

	require.NoError(t, err)
}

func TestConfigShowCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("config", "show")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn config show:")
}
