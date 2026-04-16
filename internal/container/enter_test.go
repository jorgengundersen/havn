package container_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/container"
)

type fakeEnterBackend struct {
	inspectState container.State
	inspectErr   error
}

func (f *fakeEnterBackend) ContainerInspect(_ context.Context, _ string) (container.State, error) {
	return f.inspectState, f.inspectErr
}

type fakeEnterExecBackend struct {
	interactiveExitCode int
	interactiveErr      error
	interactiveName     string
	interactiveCmd      []string
	interactiveWorkdir  string
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

func TestEnter_RunningContainer_AttachesPlainBash(t *testing.T) {
	ctx := context.Background()
	exec := &fakeEnterExecBackend{interactiveExitCode: 0}
	deps := container.EnterDeps{
		Container: &fakeEnterBackend{inspectState: container.State{ID: "abc123", Running: true}},
		Exec:      exec,
	}

	exitCode, err := container.Enter(ctx, deps, "/home/devuser/Repos/github.com/user/project")

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Equal(t, []string{"bash"}, exec.interactiveCmd)
	assert.Equal(t, "/home/devuser/Repos/github.com/user/project", exec.interactiveWorkdir)
}

func TestEnter_MissingContainer_ReturnsActionableNotRunningError(t *testing.T) {
	ctx := context.Background()
	deps := container.EnterDeps{
		Container: &fakeEnterBackend{inspectErr: &container.NotFoundError{Name: "havn-user-project"}},
		Exec:      &fakeEnterExecBackend{},
	}

	_, err := container.Enter(ctx, deps, "/home/devuser/Repos/github.com/user/project")

	var notRunning *container.EnterContainerNotRunningError
	require.True(t, errors.As(err, &notRunning))
	assert.Equal(t, "missing", notRunning.State)
	assert.ErrorContains(t, err, "havn up /home/devuser/Repos/github.com/user/project")
}

func TestEnter_StoppedContainer_ReturnsActionableNotRunningError(t *testing.T) {
	ctx := context.Background()
	exec := &fakeEnterExecBackend{}
	deps := container.EnterDeps{
		Container: &fakeEnterBackend{inspectState: container.State{ID: "stopped-123", Running: false}},
		Exec:      exec,
	}

	_, err := container.Enter(ctx, deps, "/home/devuser/Repos/github.com/user/project")

	var notRunning *container.EnterContainerNotRunningError
	require.True(t, errors.As(err, &notRunning))
	assert.Equal(t, "stopped", notRunning.State)
	assert.ErrorContains(t, err, "havn up /home/devuser/Repos/github.com/user/project")
	assert.Empty(t, exec.interactiveName)
}
