package container_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/container"
)

type fakeEnterBackend struct {
	inspectState container.State
	inspectErr   error
	inspectName  string
}

func (f *fakeEnterBackend) ContainerInspect(_ context.Context, name string) (container.State, error) {
	f.inspectName = name
	return f.inspectState, f.inspectErr
}

type fakeEnterExecBackend struct {
	interactiveExitCode int
	interactiveErr      error
	interactiveName     string
	interactiveCmd      []string
	interactiveWorkdir  string
}

type fakeEnterNixRegistryPreparer struct {
	calls []string
	err   error
}

func (f *fakeEnterNixRegistryPreparer) Prepare(_ context.Context, containerName string) error {
	f.calls = append(f.calls, containerName)
	return f.err
}

func (f *fakeEnterExecBackend) ContainerExec(_ context.Context, _ string, _ []string) error {
	return nil
}

func (f *fakeEnterExecBackend) ContainerExecInteractive(_ context.Context, name string, cmd []string, workdir string) (int, error) {
	f.interactiveName = name
	f.interactiveCmd = cmd
	f.interactiveWorkdir = workdir
	return f.interactiveExitCode, f.interactiveErr
}

func TestEnter_RunningContainer_UsesHostPathForLookupAndContainerPathForPlainBashWorkdir(t *testing.T) {
	ctx := context.Background()
	exec := &fakeEnterExecBackend{interactiveExitCode: 0}
	registry := &fakeEnterNixRegistryPreparer{}
	backend := &fakeEnterBackend{inspectState: container.State{ID: "abc123", Running: true}}
	deps := container.EnterDeps{
		Container:   backend,
		Exec:        exec,
		NixRegistry: registry,
	}

	exitCode, err := container.Enter(ctx, deps, container.ProjectPaths{
		HostPath:      "/home/alice/src/api",
		ContainerPath: "/home/devuser/work/api",
	})

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "havn-src-api", backend.inspectName)
	assert.Equal(t, "havn-src-api", exec.interactiveName)
	assert.Equal(t, []string{"bash"}, exec.interactiveCmd)
	assert.Equal(t, "/home/devuser/work/api", exec.interactiveWorkdir)
	assert.Equal(t, []string{"havn-src-api"}, registry.calls)
}

func TestEnter_RunningContainer_RelativeProjectPathDerivesContainerName(t *testing.T) {
	workspace := t.TempDir()
	projectPath := filepath.Join(workspace, "user", "project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))
	t.Chdir(projectPath)

	ctx := context.Background()
	exec := &fakeEnterExecBackend{interactiveExitCode: 0}
	deps := container.EnterDeps{
		Container: &fakeEnterBackend{inspectState: container.State{ID: "abc123", Running: true}},
		Exec:      exec,
	}

	exitCode, err := container.Enter(ctx, deps, container.ProjectPaths{HostPath: ".", ContainerPath: "/home/devuser/work/project"})

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Equal(t, "/home/devuser/work/project", exec.interactiveWorkdir)
}

func TestEnter_RunningContainer_DotSlashProjectPathDerivesContainerName(t *testing.T) {
	workspace := t.TempDir()
	basePath := filepath.Join(workspace, "user")
	projectPath := filepath.Join(basePath, "project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))
	t.Chdir(basePath)

	ctx := context.Background()
	exec := &fakeEnterExecBackend{interactiveExitCode: 0}
	deps := container.EnterDeps{
		Container: &fakeEnterBackend{inspectState: container.State{ID: "abc123", Running: true}},
		Exec:      exec,
	}

	exitCode, err := container.Enter(ctx, deps, container.ProjectPaths{HostPath: "./project", ContainerPath: "/home/devuser/work/project"})

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Equal(t, "/home/devuser/work/project", exec.interactiveWorkdir)
}

func TestEnter_RunningContainer_DotDotProjectPathDerivesContainerName(t *testing.T) {
	workspace := t.TempDir()
	parentPath := filepath.Join(workspace, "user")
	projectPath := filepath.Join(parentPath, "project")
	currentPath := filepath.Join(parentPath, "current")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))
	require.NoError(t, os.MkdirAll(currentPath, 0o755))
	t.Chdir(currentPath)

	ctx := context.Background()
	exec := &fakeEnterExecBackend{interactiveExitCode: 0}
	deps := container.EnterDeps{
		Container: &fakeEnterBackend{inspectState: container.State{ID: "abc123", Running: true}},
		Exec:      exec,
	}

	exitCode, err := container.Enter(ctx, deps, container.ProjectPaths{HostPath: "../project", ContainerPath: "/home/devuser/work/project"})

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Equal(t, "/home/devuser/work/project", exec.interactiveWorkdir)
}

func TestEnter_MissingContainer_ReturnsActionableNotRunningError(t *testing.T) {
	ctx := context.Background()
	registry := &fakeEnterNixRegistryPreparer{}
	deps := container.EnterDeps{
		Container:   &fakeEnterBackend{inspectErr: &container.NotFoundError{Name: "havn-user-project"}},
		Exec:        &fakeEnterExecBackend{},
		NixRegistry: registry,
	}

	_, err := container.Enter(ctx, deps, container.ProjectPaths{
		HostPath:      "/home/alice/work/api",
		ContainerPath: "/home/devuser/work/api",
	})

	var notRunning *container.EnterContainerNotRunningError
	require.True(t, errors.As(err, &notRunning))
	assert.Equal(t, "missing", notRunning.State)
	assert.Equal(t, "havn-work-api", notRunning.Name)
	assert.Equal(t, "/home/alice/work/api", notRunning.ProjectPath)
	assert.ErrorContains(t, err, "havn up /home/alice/work/api")
	assert.NotContains(t, err.Error(), "/home/devuser/work/api")
	assert.Empty(t, registry.calls)
}

func TestEnter_StoppedContainer_ReturnsActionableNotRunningError(t *testing.T) {
	ctx := context.Background()
	exec := &fakeEnterExecBackend{}
	registry := &fakeEnterNixRegistryPreparer{}
	deps := container.EnterDeps{
		Container:   &fakeEnterBackend{inspectState: container.State{ID: "stopped-123", Running: false}},
		Exec:        exec,
		NixRegistry: registry,
	}

	_, err := container.Enter(ctx, deps, container.ProjectPaths{
		HostPath:      "/home/alice/work/api",
		ContainerPath: "/home/devuser/work/api",
	})

	var notRunning *container.EnterContainerNotRunningError
	require.True(t, errors.As(err, &notRunning))
	assert.Equal(t, "stopped", notRunning.State)
	assert.Equal(t, "havn-work-api", notRunning.Name)
	assert.Equal(t, "/home/alice/work/api", notRunning.ProjectPath)
	assert.ErrorContains(t, err, "havn up /home/alice/work/api")
	assert.NotContains(t, err.Error(), "/home/devuser/work/api")
	assert.Empty(t, exec.interactiveName)
	assert.Empty(t, registry.calls)
}

func TestEnter_RunningContainer_NixRegistryPrepareFailure_AbortsAttach(t *testing.T) {
	ctx := context.Background()
	exec := &fakeEnterExecBackend{}
	deps := container.EnterDeps{
		Container:   &fakeEnterBackend{inspectState: container.State{ID: "abc123", Running: true}},
		Exec:        exec,
		NixRegistry: &fakeEnterNixRegistryPreparer{err: errors.New("registry unreadable")},
	}

	_, err := container.Enter(ctx, deps, container.ProjectPaths{
		HostPath:      "/home/alice/work/api",
		ContainerPath: "/home/devuser/work/api",
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "prepare nix registry aliases in container \"havn-work-api\"")
	assert.ErrorContains(t, err, "registry unreadable")
	assert.Empty(t, exec.interactiveName)
}
