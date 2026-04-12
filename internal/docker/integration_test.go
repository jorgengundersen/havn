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

func TestContainerList_NamePrefixContract_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)
	tag := buildIntegrationImage(t, c)

	prefix := fmt.Sprintf("havn-prefix-%d", time.Now().UnixNano())
	matchingName := prefix + "-match"
	nonPrefixName := "x" + prefix + "-non-prefix"

	createAndStartIntegrationContainerWithName(t, c, tag, matchingName)
	createAndStartIntegrationContainerWithName(t, c, tag, nonPrefixName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containers, err := c.ContainerList(ctx, docker.ContainerListFilters{NamePrefix: prefix})
	require.NoError(t, err)

	names := make([]string, 0, len(containers))
	for _, container := range containers {
		names = append(names, container.Name)
	}

	assert.Contains(t, names, matchingName)
	assert.NotContains(t, names, nonPrefixName)
}

func TestContainerCreateStartInspect_Translation_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)
	tag := buildIntegrationImage(t, c)

	hostDir := t.TempDir()

	containerName := fmt.Sprintf("havn-container-%d", time.Now().UnixNano())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	id, err := c.ContainerCreate(ctx, docker.CreateOpts{
		Image: tag,
		Name:  containerName,
		Env: map[string]string{
			"HAVN_SAMPLE": "value",
		},
		Labels: map[string]string{
			"havn.test": "container-translation",
		},
		BindMounts: []docker.BindMount{{
			Source: hostDir,
			Target: "/workspace",
		}},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = c.ContainerRemove(cleanupCtx, id, docker.RemoveOpts{Force: true, RemoveVolumes: true})
	})

	require.NoError(t, c.ContainerStart(ctx, id))

	info, err := c.ContainerInspect(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, info.ID)
	assert.Equal(t, containerName, info.Name)
	assert.Equal(t, tag, info.Image)
	assert.Equal(t, "running", info.Status)
	assert.Equal(t, "container-translation", info.Labels["havn.test"])
	assert.Contains(t, info.Env, "HAVN_SAMPLE=value")
	assert.Contains(t, info.Networks, "bridge")

	mountsByTarget := make(map[string]docker.MountInfo, len(info.Mounts))
	for _, mount := range info.Mounts {
		mountsByTarget[mount.Target] = mount
	}
	assert.Equal(t, hostDir, mountsByTarget["/workspace"].Source)
}

func TestContainerStopRemove_LifecycleSuccess_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)
	tag := buildIntegrationImage(t, c)

	containerName := fmt.Sprintf("havn-lifecycle-%d", time.Now().UnixNano())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	id, err := c.ContainerCreate(ctx, docker.CreateOpts{
		Image: tag,
		Name:  containerName,
	})
	require.NoError(t, err)

	require.NoError(t, c.ContainerStart(ctx, id))
	require.NoError(t, c.ContainerStop(ctx, id, docker.StopOpts{Timeout: 10}))

	stoppedInfo, err := c.ContainerInspect(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "exited", stoppedInfo.Status)

	require.NoError(t, c.ContainerRemove(ctx, id, docker.RemoveOpts{Force: false, RemoveVolumes: true}))

	_, err = c.ContainerInspect(ctx, id)
	require.Error(t, err)
	var notFoundErr *docker.ContainerNotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestNetworkCreateInspectList_Contract_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)
	tag := buildIntegrationImage(t, c)

	prefix := fmt.Sprintf("havn-net-%d", time.Now().UnixNano())
	matchingName := prefix + "-match"
	nonPrefixName := "x" + prefix + "-non-prefix"
	containerName := prefix + "-container"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, c.NetworkCreate(ctx, docker.NetworkCreateOpts{
		Name: matchingName,
		Labels: map[string]string{
			"havn.test": "network-contract",
		},
	}))
	t.Cleanup(func() {
		cleanupDockerCommand(t, "network", "rm", matchingName)
	})

	require.NoError(t, c.NetworkCreate(ctx, docker.NetworkCreateOpts{
		Name: nonPrefixName,
		Labels: map[string]string{
			"havn.test": "network-contract",
		},
	}))
	t.Cleanup(func() {
		cleanupDockerCommand(t, "network", "rm", nonPrefixName)
	})

	createAndStartIntegrationContainerOnNetwork(t, c, tag, containerName, matchingName)

	inspected, err := c.NetworkInspect(ctx, matchingName)
	require.NoError(t, err)
	assert.Equal(t, matchingName, inspected.Name)
	assert.NotEmpty(t, inspected.ID)
	assert.NotEmpty(t, inspected.Driver)

	connected := make([]string, 0, len(inspected.ConnectedContainers))
	for _, ctr := range inspected.ConnectedContainers {
		connected = append(connected, ctr.Name)
	}
	assert.Contains(t, connected, containerName)

	listed, err := c.NetworkList(ctx, docker.NetworkListFilters{NamePrefix: prefix})
	require.NoError(t, err)

	listedNames := make([]string, 0, len(listed))
	for _, nw := range listed {
		listedNames = append(listedNames, nw.Name)
	}
	assert.Contains(t, listedNames, matchingName)
	assert.NotContains(t, listedNames, nonPrefixName)

	var listedMatch *docker.NetworkInfo
	for i := range listed {
		if listed[i].Name == matchingName {
			listedMatch = &listed[i]
			break
		}
	}
	require.NotNil(t, listedMatch)

	listedConnected := make([]string, 0, len(listedMatch.ConnectedContainers))
	for _, ctr := range listedMatch.ConnectedContainers {
		listedConnected = append(listedConnected, ctr.Name)
	}
	assert.Contains(t, listedConnected, containerName)
}

func TestVolumeCreateInspectList_Contract_Integration(t *testing.T) {
	c := requireIntegrationDockerClient(t)

	prefix := fmt.Sprintf("havn-vol-%d", time.Now().UnixNano())
	matchingName := prefix + "-match"
	nonPrefixName := "x" + prefix + "-non-prefix"
	labels := map[string]string{
		"havn.test":  "volume-contract",
		"havn.scope": "integration",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, c.VolumeCreate(ctx, docker.VolumeCreateOpts{
		Name:   matchingName,
		Labels: labels,
	}))
	t.Cleanup(func() {
		cleanupDockerCommand(t, "volume", "rm", matchingName)
	})

	require.NoError(t, c.VolumeCreate(ctx, docker.VolumeCreateOpts{
		Name:   nonPrefixName,
		Labels: labels,
	}))
	t.Cleanup(func() {
		cleanupDockerCommand(t, "volume", "rm", nonPrefixName)
	})

	inspected, err := c.VolumeInspect(ctx, matchingName)
	require.NoError(t, err)
	assert.Equal(t, matchingName, inspected.Name)
	assert.NotEmpty(t, inspected.Driver)
	assert.Equal(t, labels["havn.test"], inspected.Labels["havn.test"])
	assert.Equal(t, labels["havn.scope"], inspected.Labels["havn.scope"])
	assert.NotEmpty(t, inspected.Mountpoint)
	assert.NotEmpty(t, inspected.CreatedAt)

	listed, err := c.VolumeList(ctx, docker.VolumeListFilters{
		Labels:     labels,
		NamePrefix: prefix,
	})
	require.NoError(t, err)

	listedNames := make([]string, 0, len(listed))
	for _, vol := range listed {
		listedNames = append(listedNames, vol.Name)
	}
	assert.Contains(t, listedNames, matchingName)
	assert.NotContains(t, listedNames, nonPrefixName)
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

	name := fmt.Sprintf("havn-integration-%d", time.Now().UnixNano())
	return createAndStartIntegrationContainerWithName(t, c, imageTag, name)
}

func createAndStartIntegrationContainerWithName(t *testing.T, c *docker.Client, imageTag, name string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

func createAndStartIntegrationContainerOnNetwork(t *testing.T, c *docker.Client, imageTag, name, network string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	id, err := c.ContainerCreate(ctx, docker.CreateOpts{
		Image:   imageTag,
		Name:    name,
		Network: network,
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

func cleanupDockerCommand(t *testing.T, resource string, args ...string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmdArgs := append([]string{resource}, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	_, _ = cmd.CombinedOutput()
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
