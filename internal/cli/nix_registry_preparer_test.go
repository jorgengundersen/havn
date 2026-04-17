package cli

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
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
			"chown devuser:devuser '/home/devuser/.local/state/nix/registry.json'":                             {{ExitCode: 0}},
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

type statefulNixRegistryRuntimeBackend struct {
	files    map[string][]byte
	symlinks map[string]string
	dirs     map[string]struct{}
	owners   map[string]string

	execCalls []string
	copyCalls []copyCall
}

func newStatefulNixRegistryRuntimeBackend() *statefulNixRegistryRuntimeBackend {
	return &statefulNixRegistryRuntimeBackend{
		files:    make(map[string][]byte),
		symlinks: make(map[string]string),
		dirs:     make(map[string]struct{}),
		owners:   make(map[string]string),
	}
}

func (f *statefulNixRegistryRuntimeBackend) ContainerExec(_ context.Context, _ string, opts docker.ExecOpts) (docker.ExecResult, error) {
	if len(opts.Cmd) == 0 {
		return docker.ExecResult{}, fmt.Errorf("missing exec command")
	}

	cmd := opts.Cmd[len(opts.Cmd)-1]
	f.execCalls = append(f.execCalls, cmd)

	switch {
	case strings.HasPrefix(cmd, "test -f '"):
		filePath, ok := singleQuotedArg(cmd, "test -f ")
		if !ok {
			return docker.ExecResult{}, fmt.Errorf("unsupported test command: %s", cmd)
		}
		resolved := f.resolvePath(filePath)
		if _, exists := f.files[resolved]; exists {
			return docker.ExecResult{ExitCode: 0}, nil
		}
		return docker.ExecResult{ExitCode: 1}, nil
	case strings.HasPrefix(cmd, "cat '"):
		filePath, ok := singleQuotedArg(cmd, "cat ")
		if !ok {
			return docker.ExecResult{}, fmt.Errorf("unsupported cat command: %s", cmd)
		}
		resolved := f.resolvePath(filePath)
		data, exists := f.files[resolved]
		if !exists {
			return docker.ExecResult{ExitCode: 1, Stderr: []byte("No such file")}, nil
		}
		return docker.ExecResult{ExitCode: 0, Stdout: append([]byte(nil), data...)}, nil
	case strings.HasPrefix(cmd, "mkdir -p '"):
		dirPath, ok := singleQuotedArg(cmd, "mkdir -p ")
		if !ok {
			return docker.ExecResult{}, fmt.Errorf("unsupported mkdir command: %s", cmd)
		}
		f.dirs[dirPath] = struct{}{}
		return docker.ExecResult{ExitCode: 0}, nil
	case strings.HasPrefix(cmd, "ln -sfn '"):
		src, dst, ok := twoQuotedArgs(cmd, "ln -sfn ")
		if !ok {
			return docker.ExecResult{}, fmt.Errorf("unsupported ln command: %s", cmd)
		}
		f.symlinks[dst] = src
		return docker.ExecResult{ExitCode: 0}, nil
	case strings.HasPrefix(cmd, "chown "):
		owner, filePath, ok := ownerAndPathArg(cmd)
		if !ok {
			return docker.ExecResult{}, fmt.Errorf("unsupported chown command: %s", cmd)
		}
		resolved := f.resolvePath(filePath)
		if _, exists := f.files[resolved]; !exists {
			return docker.ExecResult{ExitCode: 1, Stderr: []byte("No such file")}, nil
		}
		f.owners[resolved] = owner
		return docker.ExecResult{ExitCode: 0}, nil
	default:
		return docker.ExecResult{}, fmt.Errorf("unexpected exec command: %s", cmd)
	}
}

func (f *statefulNixRegistryRuntimeBackend) CopyToContainer(_ context.Context, _ string, dstPath string, tarStream io.Reader) error {
	data, err := io.ReadAll(tarStream)
	if err != nil {
		return err
	}

	f.copyCalls = append(f.copyCalls, copyCall{dstPath: dstPath, tarData: data})

	tr := tar.NewReader(bytes.NewReader(data))
	header, err := tr.Next()
	if err != nil {
		return err
	}
	content, err := io.ReadAll(tr)
	if err != nil {
		return err
	}

	resolvedPath := path.Join(dstPath, strings.TrimPrefix(header.Name, "./"))
	f.files[resolvedPath] = append([]byte(nil), content...)
	f.owners[resolvedPath] = "root:root"
	return nil
}

func (f *statefulNixRegistryRuntimeBackend) setFile(filePath string, content []byte) {
	f.files[filePath] = append([]byte(nil), content...)
	f.owners[filePath] = "devuser:devuser"
}

func (f *statefulNixRegistryRuntimeBackend) fileOwner(filePath string) (string, bool) {
	resolved := f.resolvePath(filePath)
	owner, ok := f.owners[resolved]
	return owner, ok
}

func (f *statefulNixRegistryRuntimeBackend) fileContent(filePath string) ([]byte, bool) {
	resolved := f.resolvePath(filePath)
	content, ok := f.files[resolved]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), content...), true
}

func (f *statefulNixRegistryRuntimeBackend) resolvePath(filePath string) string {
	for {
		target, ok := f.symlinks[filePath]
		if !ok {
			return filePath
		}
		filePath = target
	}
}

func singleQuotedArg(cmd string, prefix string) (string, bool) {
	trimmed := strings.TrimPrefix(cmd, prefix)
	if !strings.HasPrefix(trimmed, "'") || !strings.HasSuffix(trimmed, "'") {
		return "", false
	}
	return strings.Trim(trimmed, "'"), true
}

func twoQuotedArgs(cmd string, prefix string) (string, string, bool) {
	trimmed := strings.TrimPrefix(cmd, prefix)
	parts := strings.Split(trimmed, "' '")
	if len(parts) != 2 {
		return "", "", false
	}
	src := strings.TrimPrefix(parts[0], "'")
	dst := strings.TrimSuffix(parts[1], "'")
	if src == parts[0] || dst == parts[1] {
		return "", "", false
	}
	return src, dst, true
}

func ownerAndPathArg(cmd string) (string, string, bool) {
	parts := strings.SplitN(cmd, " ", 3)
	if len(parts) != 3 {
		return "", "", false
	}
	owner := parts[1]
	pathArg := parts[2]
	if !strings.HasPrefix(pathArg, "'") || !strings.HasSuffix(pathArg, "'") {
		return "", "", false
	}
	return owner, strings.Trim(pathArg, "'"), true
}

func TestNixRegistryPreparer_Prepare_RecreatedContainerWithSharedState_PreservesAliases(t *testing.T) {
	backend := newStatefulNixRegistryRuntimeBackend()
	preparer := nixRegistryPreparer{docker: backend}

	err := preparer.Prepare(context.Background(), "havn-user-project")
	require.NoError(t, err)
	require.Len(t, backend.copyCalls, 1)

	backend.setFile(stateRegistryPath, []byte(`{"version":2,"flakes":[{"from":{"id":"flake:devenv"},"to":{"type":"github","owner":"cachix","repo":"devenv"}}]}`))
	delete(backend.symlinks, legacyRegistryPath)

	err = preparer.Prepare(context.Background(), "havn-user-project")
	require.NoError(t, err)

	content, exists := backend.fileContent(stateRegistryPath)
	require.True(t, exists)
	assert.JSONEq(t, `{"version":2,"flakes":[{"from":{"id":"flake:devenv"},"to":{"type":"github","owner":"cachix","repo":"devenv"}}]}`,
		string(content))
	assert.Equal(t, stateRegistryPath, backend.symlinks[legacyRegistryPath])
	assert.Len(t, backend.copyCalls, 1, "prepare should not rewrite shared state when aliases already exist")
}

func TestNixRegistryPreparer_Prepare_FixesStateRegistryOwnershipAfterWrite(t *testing.T) {
	backend := newStatefulNixRegistryRuntimeBackend()
	preparer := nixRegistryPreparer{docker: backend}

	err := preparer.Prepare(context.Background(), "havn-user-project")

	require.NoError(t, err)
	owner, ok := backend.fileOwner(stateRegistryPath)
	require.True(t, ok)
	assert.Equal(t, "devuser:devuser", owner)
	assert.Contains(t, backend.execCalls, "chown devuser:devuser '/home/devuser/.local/state/nix/registry.json'")
}

func TestNixRegistryPreparer_Prepare_MalformedPersistentState_FailsSafely(t *testing.T) {
	backend := newStatefulNixRegistryRuntimeBackend()
	backend.setFile(stateRegistryPath, []byte(`{"version":2,"flakes":[`))

	preparer := nixRegistryPreparer{docker: backend}

	err := preparer.Prepare(context.Background(), "havn-user-project")
	require.Error(t, err)
	assert.ErrorContains(t, err, "parse nix registry file")
	assert.ErrorContains(t, err, stateRegistryPath)
	assert.Empty(t, backend.copyCalls, "prepare must not overwrite malformed persistent state")
	_, symlinkWritten := backend.symlinks[legacyRegistryPath]
	assert.False(t, symlinkWritten, "prepare should fail before mutating legacy registry wiring")
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
