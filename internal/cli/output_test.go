package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
)

func TestStatus_WritesToStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, false, false)

	out.Status("starting container")

	assert.Empty(t, stdout.String(), "status should not write to stdout")
	assert.Contains(t, stderr.String(), "starting container")
}

func TestData_WritesToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, false, false)

	out.Data("container-id-123")

	assert.Contains(t, stdout.String(), "container-id-123")
	assert.Empty(t, stderr.String(), "data should not write to stderr")
}

func TestDataJSON_ProducesValidJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, true, false)

	out.DataJSON(map[string]string{"id": "abc-123"})

	assert.Empty(t, stderr.String(), "JSON data should not write to stderr")
	require.True(t, json.Valid(stdout.Bytes()), "output should be valid JSON")

	var result map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "abc-123", result["id"])
}

func TestVerbose_WritesToStderr_NotStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, false, true)

	out.Verbose("debug info here")

	assert.Empty(t, stdout.String(), "verbose should not write to stdout")
	assert.Contains(t, stderr.String(), "debug info here")
}

func TestVerbose_SilentWhenNotEnabled(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, false, false)

	out.Verbose("debug info here")

	assert.Empty(t, stdout.String(), "verbose should not write to stdout")
	assert.Empty(t, stderr.String(), "verbose should not write to stderr when disabled")
}

func TestIsJSON_ReflectsMode(t *testing.T) {
	jsonOut := cli.NewOutput(nil, nil, true, false)
	assert.True(t, jsonOut.IsJSON())

	plainOut := cli.NewOutput(nil, nil, false, false)
	assert.False(t, plainOut.IsJSON())
}
