package cli

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/docker"
)

type fakeNixRegistryRuntimeBackend struct {
	execResponses map[string][]docker.ExecResult
	execCalls     []string

	copyCalls []copyCall
}

type copyCall struct {
	dstPath string
	tarData []byte
}

func (f *fakeNixRegistryRuntimeBackend) ContainerExec(_ context.Context, _ string, opts docker.ExecOpts) (docker.ExecResult, error) {
	if len(opts.Cmd) == 0 {
		return docker.ExecResult{}, fmt.Errorf("missing exec command")
	}

	cmd := opts.Cmd[len(opts.Cmd)-1]
	f.execCalls = append(f.execCalls, cmd)

	responses, ok := f.execResponses[cmd]
	if !ok || len(responses) == 0 {
		return docker.ExecResult{}, fmt.Errorf("unexpected exec command: %s", cmd)
	}

	result := responses[0]
	f.execResponses[cmd] = responses[1:]
	return result, nil
}

func (f *fakeNixRegistryRuntimeBackend) CopyToContainer(_ context.Context, _ string, dstPath string, tarStream io.Reader) error {
	data, err := io.ReadAll(tarStream)
	if err != nil {
		return err
	}

	f.copyCalls = append(f.copyCalls, copyCall{dstPath: dstPath, tarData: data})
	return nil
}

func TestNixRegistryPreparer_Prepare_NoRegistryFiles_WiresStateRegistryAndLegacySymlink(t *testing.T) {
	backend := &fakeNixRegistryRuntimeBackend{
		execResponses: map[string][]docker.ExecResult{
			"test -f '/home/devuser/.local/state/nix/registry.json'":                                           {{ExitCode: 1}},
			"test -f '/home/devuser/.config/nix/registry.json'":                                                {{ExitCode: 1}},
			"mkdir -p '/home/devuser/.local/state/nix'":                                                        {{ExitCode: 0}},
			"mkdir -p '/home/devuser/.config/nix'":                                                             {{ExitCode: 0}},
			"ln -sfn '/home/devuser/.local/state/nix/registry.json' '/home/devuser/.config/nix/registry.json'": {{ExitCode: 0}},
		},
	}

	preparer := nixRegistryPreparer{docker: backend}

	err := preparer.Prepare(context.Background(), "havn-user-project")

	require.NoError(t, err)
	require.Len(t, backend.copyCalls, 1)
	assert.Equal(t, "/home/devuser/.local/state/nix", backend.copyCalls[0].dstPath)

	name, content := extractSingleTarFile(t, backend.copyCalls[0].tarData)
	assert.Equal(t, "registry.json", name)
	assert.JSONEq(t, `{"version":2,"flakes":[]}`, string(content))

	assert.Contains(t, backend.execCalls, "mkdir -p '/home/devuser/.config/nix'")
	assert.Contains(t, backend.execCalls, "ln -sfn '/home/devuser/.local/state/nix/registry.json' '/home/devuser/.config/nix/registry.json'")
}

func extractSingleTarFile(t *testing.T, tarData []byte) (string, []byte) {
	t.Helper()

	tr := tar.NewReader(bytes.NewReader(tarData))
	header, err := tr.Next()
	require.NoError(t, err)

	content, err := io.ReadAll(tr)
	require.NoError(t, err)

	_, err = tr.Next()
	require.ErrorIs(t, err, io.EOF)

	return strings.TrimPrefix(header.Name, "./"), content
}
