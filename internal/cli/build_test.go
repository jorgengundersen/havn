package cli_test

import (
	"bytes"
	"context"
	"io"
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
