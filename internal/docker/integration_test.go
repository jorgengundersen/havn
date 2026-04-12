//go:build integration

package docker_test

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"

	"github.com/jorgengundersen/havn/internal/docker"
)

func TestDaemonPingAndInfo_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, c.Ping(ctx))

	info, err := c.Info(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, info.ServerVersion)
	assert.NotEmpty(t, info.APIVersion)
}

func TestImageBuildInspectExists_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)
	tag := buildIntegrationImage(t, c)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	info, err := c.ImageInspect(ctx, tag)
	require.NoError(t, err)
	assert.NotEmpty(t, info.ID)
	assert.Equal(t, tag, info.Tag)
	assert.NotEmpty(t, info.CreatedAt)

	exists, err := c.ImageExists(ctx, tag)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCopyToFromContainer_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)
	tag := buildIntegrationImage(t, c)
	containerName := createAndStartIntegrationContainer(t, c, tag)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const (
		containerPath = "/copied.txt"
		fileName      = "copied.txt"
		wantContent   = "copy integration payload"
	)

	err := c.CopyToContainer(ctx, containerName, "/", singleFileTar(t, fileName, wantContent))
	require.NoError(t, err)

	rc, err := c.CopyFromContainer(ctx, containerName, containerPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })

	got := readSingleFileTar(t, rc)
	assert.Equal(t, wantContent, got)
}

func TestContainerExec_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)
	tag := buildIntegrationImage(t, c)
	containerName := createAndStartIntegrationContainer(t, c, tag)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := c.ContainerExec(ctx, containerName, docker.ExecOpts{
		Cmd: []string{"/app", "stdout-stderr"},
	})
	require.NoError(t, err)
	assert.Equal(t, 7, result.ExitCode)
	assert.Equal(t, []byte("hello-out"), result.Stdout)
	assert.Equal(t, []byte("hello-err"), result.Stderr)
}

func TestContainerAttach_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)
	tag := buildIntegrationImage(t, c)
	containerName := createAndStartIntegrationContainer(t, c, tag)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stdin := bytes.NewBufferString("ping\n")
	stdout := &lockedBuffer{}
	var stderr bytes.Buffer

	exitCode, err := c.ContainerAttach(ctx, containerName, docker.AttachOpts{
		Cmd:    []string{"/app", "attach-echo-exit"},
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: &stderr,
	})
	require.NoError(t, err)
	assert.Equal(t, 23, exitCode)
	assert.Contains(t, stdout.String(), "echo:ping")
	assert.Empty(t, stderr.String())
}

func TestContainerAttach_Resize_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)
	tag := buildIntegrationImage(t, c)
	containerName := createAndStartIntegrationContainer(t, c, tag)

	ptyMaster, ptySlave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ptyMaster.Close() })
	t.Cleanup(func() { _ = ptySlave.Close() })

	require.NoError(t, pty.Setsize(ptySlave, &pty.Winsize{Rows: 24, Cols: 80}))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stdout := &lockedBuffer{}
	var stderr bytes.Buffer

	resultCh := make(chan struct {
		exitCode int
		err      error
	}, 1)

	go func() {
		exitCode, attachErr := c.ContainerAttach(ctx, containerName, docker.AttachOpts{
			Cmd:    []string{"/app", "attach-resize"},
			Stdin:  ptySlave,
			Stdout: stdout,
			Stderr: &stderr,
		})
		resultCh <- struct {
			exitCode int
			err      error
		}{
			exitCode: exitCode,
			err:      attachErr,
		}
	}()

	waitForOutput(t, stdout, "initial:", 5*time.Second)

	require.NoError(t, pty.Setsize(ptySlave, &pty.Winsize{Rows: 40, Cols: 100}))
	require.NoError(t, unix.Kill(os.Getpid(), unix.SIGWINCH))

	result := <-resultCh
	require.NoError(t, result.err)
	assert.Equal(t, 29, result.exitCode)
	assert.Contains(t, stdout.String(), "initial:80x24")
	assert.Contains(t, stdout.String(), "resized:100x40")
	assert.Empty(t, stderr.String())
}

type lockedBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func waitForOutput(t *testing.T, buf *lockedBuffer, contains string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), contains) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timeout waiting for output containing %q; output=%q", contains, buf.String())
}

func requireIntegrationDockerClient(t *testing.T) *docker.Client {
	t.Helper()

	c, err := docker.NewClient()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Ping(ctx); err != nil {
		t.Skipf("docker daemon unavailable: %v", err)
	}

	return c
}

func buildIntegrationImage(t *testing.T, c *docker.Client) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	buildContext := t.TempDir()

	mainSource := []byte(`package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "stdout-stderr":
			_, _ = fmt.Fprint(os.Stdout, "hello-out")
			_, _ = fmt.Fprint(os.Stderr, "hello-err")
			os.Exit(7)
		case "attach-echo-exit":
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			_, _ = fmt.Fprintf(os.Stdout, "echo:%s", line)
			os.Exit(23)
		case "attach-resize":
			ws, _ := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
			_, _ = fmt.Fprintf(os.Stdout, "initial:%dx%d\n", ws.Col, ws.Row)

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGWINCH)
			defer signal.Stop(sigCh)

			select {
			case <-sigCh:
				resized, _ := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
				_, _ = fmt.Fprintf(os.Stdout, "resized:%dx%d\n", resized.Col, resized.Row)
				os.Exit(29)
			case <-time.After(5 * time.Second):
				_, _ = fmt.Fprint(os.Stdout, "resize-timeout")
				os.Exit(30)
			}
		case "serve":
			for {
				time.Sleep(1 * time.Second)
			}
		}
	}

	for {
		time.Sleep(1 * time.Second)
	}
}
`)
	require.NoError(t, os.WriteFile(filepath.Join(buildContext, "main.go"), mainSource, 0o644))

	dockerfile := []byte("FROM scratch\nCOPY app /app\nENTRYPOINT [\"/app\",\"serve\"]\n")
	require.NoError(t, os.WriteFile(filepath.Join(buildContext, "Dockerfile"), dockerfile, 0o644))

	binaryPath := filepath.Join(buildContext, "app")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, filepath.Join(buildContext, "main.go"))
	buildOut, err := buildCmd.CombinedOutput()
	require.NoErrorf(t, err, "build helper binary: %s", string(buildOut))

	tag := fmt.Sprintf("havn-integration:%d", time.Now().UnixNano())
	var output bytes.Buffer
	err = c.ImageBuild(ctx, docker.BuildOpts{
		Tag:        tag,
		Context:    buildContext,
		Dockerfile: "Dockerfile",
		Output:     &output,
	})
	require.NoErrorf(t, err, "image build output:\n%s", output.String())

	return tag
}

func createAndStartIntegrationContainer(t *testing.T, c *docker.Client, imageTag string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name := fmt.Sprintf("havn-integration-%d", time.Now().UnixNano())
	id, err := c.ContainerCreate(ctx, docker.CreateOpts{
		Image: imageTag,
		Name:  name,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = c.ContainerRemove(cleanupCtx, id, docker.RemoveOpts{Force: true, RemoveVolumes: true})
	})

	require.NoError(t, c.ContainerStart(ctx, id))

	return name
}

func singleFileTar(t *testing.T, name string, content string) io.Reader {
	t.Helper()

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	data := []byte(content)
	header := &tar.Header{
		Name: name,
		Mode: 0o644,
		Size: int64(len(data)),
	}
	require.NoError(t, tw.WriteHeader(header))
	_, err := tw.Write(data)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	return &buf
}

func readSingleFileTar(t *testing.T, r io.Reader) string {
	t.Helper()

	tr := tar.NewReader(r)
	for {
		_, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}

		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		return string(data)
	}

	t.Fatal("expected tar stream with at least one file")
	return ""
}
