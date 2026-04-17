package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/container"
)

type fakeListBackend struct {
	containers []container.RawContainer
	err        error
}

func (f fakeListBackend) ContainerList(_ context.Context, _ map[string]string) ([]container.RawContainer, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.containers, nil
}

func executeCommand(args ...string) (stdout, stderr string, err error) {
	root := cli.NewRoot(cli.Deps{})
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)
	root.SetArgs(args)
	err = root.Execute()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func TestListCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("list")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn list:")
}

func TestListCommand_JSONOutput(t *testing.T) {
	backend := fakeListBackend{containers: []container.RawContainer{{
		Name:   "havn-user-api",
		Image:  "havn-base:latest",
		Status: "running",
		Labels: map[string]string{
			container.LabelPath:       "/home/devuser/Repos/github.com/user/api",
			container.LabelShell:      "go",
			container.LabelCPUs:       "4",
			container.LabelMemory:     "8g",
			container.LabelMemorySwap: "12g",
			container.LabelDolt:       "true",
		},
	}}}

	stdout, _, err := executeCommandWithDeps(cli.Deps{ContainerList: backend}, "list", "--json")

	require.NoError(t, err)
	var got []map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "havn-user-api", got[0]["name"])
	assert.Equal(t, "/home/devuser/Repos/github.com/user/api", got[0]["path"])
	assert.Equal(t, "havn-base:latest", got[0]["image"])
	assert.Equal(t, "running", got[0]["status"])
	assert.Equal(t, "go", got[0]["shell"])
	assert.Equal(t, float64(4), got[0]["cpus"])
	assert.Equal(t, "8g", got[0]["memory"])
	assert.Equal(t, "12g", got[0]["memory_swap"])
	assert.Equal(t, true, got[0]["dolt"])
}

func TestListCommand_HumanOutput(t *testing.T) {
	backend := fakeListBackend{containers: []container.RawContainer{{
		Name:   "havn-user-api",
		Image:  "havn-base:latest",
		Status: "running",
		Labels: map[string]string{
			container.LabelPath:   "/home/devuser/Repos/github.com/user/api",
			container.LabelShell:  "go",
			container.LabelCPUs:   "4",
			container.LabelMemory: "8g",
			container.LabelDolt:   "false",
		},
	}}}

	stdout, _, err := executeCommandWithDeps(cli.Deps{ContainerList: backend}, "list")

	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	require.Len(t, lines, 1)
	assert.Equal(t, "havn-user-api\t/home/devuser/Repos/github.com/user/api", lines[0])
}
