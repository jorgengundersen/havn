package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/container"
)

type fakeEnterService struct {
	called      bool
	lastProject string
	exitCode    int
	err         error
}

func (f *fakeEnterService) Enter(_ context.Context, projectPath string) (int, error) {
	f.called = true
	f.lastProject = projectPath
	return f.exitCode, f.err
}

func TestEnterCommand_IsRegistered(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	enterCmd, _, err := root.Find([]string{"enter"})

	require.NoError(t, err)
	assert.Equal(t, "enter [path]", enterCmd.Use)
}

func TestEnterCommand_HelpIncludesManualHomeManagerActivationPath(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	enterCmd, _, err := root.Find([]string{"enter"})

	require.NoError(t, err)
	assert.Contains(t, enterCmd.Long, "manual Home Manager activation")
	assert.Contains(t, enterCmd.Long, "home-manager switch --flake")
}

func TestEnterCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("enter")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn enter:")
}

func TestEnterCommand_CallsServiceWithResolvedPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeEnterService{}
	_, _, err := executeCommandWithDeps(cli.Deps{EnterService: svc}, "enter", projectPath)

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.Equal(t, projectPath, svc.lastProject)
}

func TestEnterCommand_PropagatesInteractiveExitCode(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeEnterService{exitCode: 23}
	_, _, err := executeCommandWithDeps(cli.Deps{EnterService: svc}, "enter", projectPath)

	require.Error(t, err)
	var shellExit *cli.ShellExitError
	require.ErrorAs(t, err, &shellExit)
	assert.Equal(t, 23, shellExit.Code)
}

func TestEnterCommand_MissingContainerReturnsActionableError(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeEnterService{err: &container.EnterContainerNotRunningError{
		Name:        "havn-work-sample-project",
		ProjectPath: projectPath,
		State:       "missing",
	}}
	_, _, err := executeCommandWithDeps(cli.Deps{EnterService: svc}, "enter", projectPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "havn enter:")
	assert.Contains(t, err.Error(), "havn up "+projectPath)
}

func TestEnterCommand_PathDefaultsToDot(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectPath))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWD))
	})

	svc := &fakeEnterService{}
	_, _, err = executeCommandWithDeps(cli.Deps{EnterService: svc}, "enter")

	require.NoError(t, err)
	assert.Equal(t, projectPath, svc.lastProject)
}

func TestEnterCommand_NixRegistryPrepareFailureIsWrapped(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeEnterService{err: assert.AnError}
	_, _, err := executeCommandWithDeps(cli.Deps{EnterService: svc}, "enter", projectPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "havn enter:")
	assert.ErrorIs(t, err, assert.AnError)
}
