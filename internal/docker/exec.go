package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	cerrdefs "github.com/containerd/errdefs"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// ExecOpts holds parameters for a non-interactive one-shot exec.
type ExecOpts struct {
	Cmd     []string // Command and arguments to execute
	Env     []string // Environment variables (KEY=VALUE)
	Workdir string   // Working directory inside the container
	User    string   // User to run the command as
}

// ExecResult holds the outcome of a non-interactive exec.
// A non-zero ExitCode is not an error — only exec-plumbing failures return errors.
type ExecResult struct {
	ExitCode int    // Process exit code
	Stdout   []byte // Captured stdout bytes
	Stderr   []byte // Captured stderr bytes
}

// ContainerExec runs a non-interactive one-shot command in the specified
// container and captures stdout/stderr. A non-zero exit code is returned in
// ExecResult, not as an error — only exec-plumbing failures return errors.
// Returns *ContainerNotFoundError if the container does not exist.
func (c *Client) ContainerExec(ctx context.Context, nameOrID string, opts ExecOpts) (ExecResult, error) {
	execResp, err := c.docker.ContainerExecCreate(ctx, nameOrID, dockercontainer.ExecOptions{
		Cmd:          opts.Cmd,
		Env:          opts.Env,
		WorkingDir:   opts.Workdir,
		User:         opts.User,
		Tty:          false,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return ExecResult{}, &ContainerNotFoundError{Name: nameOrID}
		}
		return ExecResult{}, fmt.Errorf("docker exec create: %w", err)
	}

	hijacked, err := c.docker.ContainerExecAttach(ctx, execResp.ID, dockercontainer.ExecAttachOptions{})
	if err != nil {
		return ExecResult{}, fmt.Errorf("docker exec attach: %w", err)
	}
	defer hijacked.Close()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, hijacked.Reader); err != nil {
		return ExecResult{}, fmt.Errorf("docker exec read: %w", err)
	}

	inspect, err := c.docker.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return ExecResult{}, fmt.Errorf("docker exec inspect: %w", err)
	}

	return ExecResult{
		ExitCode: inspect.ExitCode,
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
	}, nil
}

// AttachOpts holds parameters for an interactive tty exec session.
type AttachOpts struct {
	Cmd     []string  // Command and arguments to execute
	Env     []string  // Environment variables (KEY=VALUE)
	Workdir string    // Working directory inside the container
	User    string    // User to run the command as
	Stdin   io.Reader // Host stdin (typically os.Stdin)
	Stdout  io.Writer // Host stdout (typically os.Stdout)
	Stderr  io.Writer // Host stderr (typically os.Stderr)
}

// TerminalFd returns the file descriptor if r provides one via an Fd() method
// (e.g. *os.File), or -1 if the reader is not backed by a file descriptor.
func TerminalFd(r io.Reader) int {
	type fder interface {
		Fd() uintptr
	}
	if f, ok := r.(fder); ok {
		return int(f.Fd())
	}
	return -1
}

// ContainerAttach creates and attaches to an interactive tty exec session
// in the specified container. It proxies stdin/stdout/stderr between the
// host and the container process, handles SIGWINCH for terminal resizing,
// and puts the host terminal into raw mode for the duration of the session.
// Returns the remote process exit code. Returns *ContainerNotFoundError
// if the container does not exist.
func (c *Client) ContainerAttach(ctx context.Context, nameOrID string, opts AttachOpts) (int, error) {
	execResp, err := c.docker.ContainerExecCreate(ctx, nameOrID, dockercontainer.ExecOptions{
		Cmd:          opts.Cmd,
		Env:          opts.Env,
		WorkingDir:   opts.Workdir,
		User:         opts.User,
		Tty:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return 0, &ContainerNotFoundError{Name: nameOrID}
		}
		return 0, fmt.Errorf("docker exec create: %w", err)
	}

	hijacked, err := c.docker.ContainerExecAttach(ctx, execResp.ID, dockercontainer.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		return 0, fmt.Errorf("docker exec attach: %w", err)
	}
	defer hijacked.Close()

	// Put the host terminal into raw mode if stdin is a terminal.
	fd := TerminalFd(opts.Stdin)
	if fd >= 0 && term.IsTerminal(fd) {
		oldState, rawErr := term.MakeRaw(fd)
		if rawErr == nil {
			defer func() { _ = term.Restore(fd, oldState) }()
		}

		// Forward SIGWINCH to resize the container pty.
		c.handleSIGWINCH(ctx, execResp.ID, fd)
	}

	// Proxy I/O between host and container.
	streamDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(opts.Stdout, hijacked.Reader)
		streamDone <- err
	}()
	go func() {
		_, _ = io.Copy(hijacked.Conn, opts.Stdin)
		// Stdin closing is normal (e.g. ctrl-D); ignore errors.
	}()

	// Wait for the output stream to finish (container process exited).
	<-streamDone

	// Retrieve exit code.
	inspect, err := c.docker.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return 0, fmt.Errorf("docker exec inspect: %w", err)
	}
	return inspect.ExitCode, nil
}

// handleSIGWINCH watches for terminal resize signals and propagates the
// new dimensions to the container exec session. It also performs an initial
// resize to sync the container pty with the current host terminal size.
func (c *Client) handleSIGWINCH(ctx context.Context, execID string, fd int) {
	// Initial resize to sync dimensions.
	c.resizeExec(ctx, execID, fd)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, unix.SIGWINCH)
	go func() {
		defer signal.Stop(sigCh)
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-sigCh:
				if !ok {
					return
				}
				c.resizeExec(ctx, execID, fd)
			}
		}
	}()
}

// resizeExec reads the current terminal dimensions and sends them to the
// container exec session. Errors are silently ignored — resize is best-effort.
func (c *Client) resizeExec(ctx context.Context, execID string, fd int) {
	width, height, err := term.GetSize(fd)
	if err != nil {
		return
	}
	_ = c.docker.ContainerExecResize(ctx, execID, dockercontainer.ResizeOptions{
		Height: uint(height),
		Width:  uint(width),
	})
}
