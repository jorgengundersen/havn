package cli_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
)

func TestVolumeCommand_PrintsHelp(t *testing.T) {
	_, _, err := executeCommand("volume")

	require.NoError(t, err)
}

func TestVolumeListCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("volume", "list")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn volume list:")
}
