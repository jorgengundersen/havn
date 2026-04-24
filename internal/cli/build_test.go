package cli_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/container"
)

type fakeBuildService struct {
	lastOpts   container.BuildOpts
	lastOutput io.Writer
	err        error
}

func (f *fakeBuildService) Build(_ context.Context, opts container.BuildOpts, output io.Writer) error {
	f.lastOpts = opts
	f.lastOutput = output
	return f.err
}

func TestBuildCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("build")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn build:")
}

func TestBuildCommand_CallsBuildServiceWithSpecContract(t *testing.T) {
	service := &fakeBuildService{}
	root := cli.NewRoot(cli.Deps{BuildService: service})
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)
	root.SetArgs([]string{"build"})

	err := root.Execute()

	require.NoError(t, err)
	assert.Equal(t, "havn-base:latest", service.lastOpts.ImageName)
	assert.Equal(t, "docker/", service.lastOpts.ContextPath)
	assert.NotZero(t, service.lastOpts.UID)
	assert.NotZero(t, service.lastOpts.GID)
	assert.Same(t, stderrBuf, service.lastOutput)
	assert.Contains(t, stderrBuf.String(), "Building base image...")
	assert.Contains(t, stderrBuf.String(), "Base image built")
	assert.Empty(t, stdoutBuf.String())
}

func TestBuildCommand_JSONSuccessOutput(t *testing.T) {
	service := &fakeBuildService{}
	root := cli.NewRoot(cli.Deps{BuildService: service})
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)
	root.SetArgs([]string{"--json", "build"})

	err := root.Execute()

	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok","message":"base image built"}`+"\n", stdoutBuf.String())
	assert.Contains(t, stderrBuf.String(), "Building base image...")
}

func TestBuildCommand_UsesConfiguredImageFromEnv(t *testing.T) {
	t.Setenv("HAVN_IMAGE", "havn-custom:dev")

	service := &fakeBuildService{}
	root := cli.NewRoot(cli.Deps{BuildService: service})
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)
	root.SetArgs([]string{"build", "--config", filepath.Join(t.TempDir(), "config.toml")})

	err := root.Execute()

	require.NoError(t, err)
	assert.Equal(t, "havn-custom:dev", service.lastOpts.ImageName)
}

func TestBuildCommand_ImageFlagOverridesEnv(t *testing.T) {
	t.Setenv("HAVN_IMAGE", "havn-custom:dev")

	service := &fakeBuildService{}
	root := cli.NewRoot(cli.Deps{BuildService: service})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"build", "--image", "havn-flag:latest", "--config", filepath.Join(t.TempDir(), "config.toml")})

	err := root.Execute()

	require.NoError(t, err)
	assert.Equal(t, "havn-flag:latest", service.lastOpts.ImageName)
}

func TestBuildCommand_UsesProjectConfigImage(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectDir := filepath.Join(homeDir, "workspace", "sample")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, ".havn", "config.toml"), []byte("image = \"havn-project:dev\"\n"), 0o644))

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWD))
	})

	service := &fakeBuildService{}
	root := cli.NewRoot(cli.Deps{BuildService: service})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"build"})

	err = root.Execute()

	require.NoError(t, err)
	assert.Equal(t, "havn-project:dev", service.lastOpts.ImageName)
}
