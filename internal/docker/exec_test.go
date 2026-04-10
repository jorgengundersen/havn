package docker_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/docker"
)

func TestAttachOpts_FieldsExist(t *testing.T) {
	opts := docker.AttachOpts{
		Cmd:     []string{"bash"},
		Env:     []string{"TERM=xterm-256color"},
		Workdir: "/workspace",
		User:    "devuser",
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}

	assert.Equal(t, []string{"bash"}, opts.Cmd)
	assert.Equal(t, []string{"TERM=xterm-256color"}, opts.Env)
	assert.Equal(t, "/workspace", opts.Workdir)
	assert.Equal(t, "devuser", opts.User)
	assert.Implements(t, (*io.Reader)(nil), opts.Stdin)
	assert.Implements(t, (*io.Writer)(nil), opts.Stdout)
	assert.Implements(t, (*io.Writer)(nil), opts.Stderr)
}

func TestTerminalFd_ReturnsNegativeForNonTerminal(t *testing.T) {
	buf := &bytes.Buffer{}
	fd := docker.TerminalFd(buf)
	assert.Equal(t, -1, fd)
}

func TestTerminalFd_ReturnsFdForFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test")
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	fd := docker.TerminalFd(f)
	assert.Equal(t, int(f.Fd()), fd)
}

func TestExecOpts_FieldsExist(t *testing.T) {
	opts := docker.ExecOpts{
		Cmd:     []string{"test", "-d", "/nix/store"},
		Env:     []string{"PATH=/usr/bin"},
		Workdir: "/workspace",
		User:    "devuser",
	}

	assert.Equal(t, []string{"test", "-d", "/nix/store"}, opts.Cmd)
	assert.Equal(t, []string{"PATH=/usr/bin"}, opts.Env)
	assert.Equal(t, "/workspace", opts.Workdir)
	assert.Equal(t, "devuser", opts.User)
}

func TestExecResult_FieldsExist(t *testing.T) {
	result := docker.ExecResult{
		ExitCode: 42,
		Stdout:   []byte("hello stdout"),
		Stderr:   []byte("hello stderr"),
	}

	assert.Equal(t, 42, result.ExitCode)
	assert.Equal(t, []byte("hello stdout"), result.Stdout)
	assert.Equal(t, []byte("hello stderr"), result.Stderr)
}

func TestContainerExec_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	_, err = c.ContainerExec(context.Background(), "nonexistent", docker.ExecOpts{
		Cmd: []string{"echo", "hello"},
	})

	assert.Error(t, err)
}

func TestContainerExec_WrapsErrorWithContext(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	_, err = c.ContainerExec(context.Background(), "nonexistent", docker.ExecOpts{
		Cmd: []string{"echo", "hello"},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "docker exec create:")
}

func TestContainerExec_RespectsContextCancellation(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = c.ContainerExec(ctx, "nonexistent", docker.ExecOpts{
		Cmd: []string{"echo", "hello"},
	})

	assert.Error(t, err)
}

func TestContainerAttach_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	_, err = c.ContainerAttach(context.Background(), "nonexistent", docker.AttachOpts{
		Cmd:    []string{"bash"},
		Stdin:  &bytes.Buffer{},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	assert.Error(t, err)
}
