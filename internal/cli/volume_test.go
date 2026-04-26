package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/volume"
)

type fakeVolumeBackend struct {
	existing  map[string]bool
	inspectFn func(name string) error
}

func (f fakeVolumeBackend) VolumeInspect(_ context.Context, name string) error {
	if f.inspectFn != nil {
		return f.inspectFn(name)
	}
	if f.existing[name] {
		return nil
	}
	return &volume.NotFoundError{Name: name}
}

func (f fakeVolumeBackend) VolumeCreate(_ context.Context, _ string) error {
	return nil
}

func executeCommandWithDeps(deps cli.Deps, args ...string) (stdout, stderr string, err error) {
	root := cli.NewRoot(deps)
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)
	root.SetArgs(args)
	err = root.Execute()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func TestVolumeListCommand_JSONOutput(t *testing.T) {
	mgr := volume.NewManager(fakeVolumeBackend{existing: map[string]bool{"havn-nix": true}})

	stdout, _, err := executeCommandWithDeps(cli.Deps{VolumeManager: mgr}, "volume", "list", "--json")

	require.NoError(t, err)
	var got []map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	assert.Len(t, got, 4)
	assert.Equal(t, "havn-nix", got[0]["name"])
	assert.Equal(t, "/nix", got[0]["mount"])
	assert.Equal(t, true, got[0]["exists"])
}

func TestVolumeListCommand_HumanOutput(t *testing.T) {
	mgr := volume.NewManager(fakeVolumeBackend{existing: map[string]bool{"havn-cache": true}})

	stdout, _, err := executeCommandWithDeps(cli.Deps{VolumeManager: mgr}, "volume", "list")

	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	require.Len(t, lines, 4)
	assert.Contains(t, lines[2], "havn-cache\t/home/devuser/.cache\texists")
	assert.Contains(t, lines[0], "havn-nix\t/nix\tmissing")
}

func TestVolumeCommand_PrintsHelp(t *testing.T) {
	_, _, err := executeCommand("volume")

	require.NoError(t, err)
}

func TestVolumeListCommand_InspectBackendError(t *testing.T) {
	mgr := volume.NewManager(fakeVolumeBackend{
		inspectFn: func(name string) error {
			if name == "havn-cache" {
				return fmt.Errorf("daemon unavailable")
			}
			return nil
		},
	})

	stdout, _, err := executeCommandWithDeps(cli.Deps{VolumeManager: mgr}, "volume", "list")

	require.Error(t, err)
	assert.Empty(t, stdout)
	assert.ErrorContains(t, err, "havn volume list")
	assert.ErrorContains(t, err, "inspect volume \"havn-cache\"")
	assert.ErrorContains(t, err, "daemon unavailable")
}
