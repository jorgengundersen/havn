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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "stdout-stderr":
			_, _ = fmt.Fprint(os.Stdout, "hello-out")
			_, _ = fmt.Fprint(os.Stderr, "hello-err")
			os.Exit(7)
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
