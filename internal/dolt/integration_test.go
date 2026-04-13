//go:build integration

package dolt_test

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/dolt"
)

func TestSharedDoltLifecycleAndReadiness_Integration(t *testing.T) {
	dockerClient := requireIntegrationDockerClient(t)
	requireCleanSharedServerState(t, dockerClient)

	networkName := fmt.Sprintf("havn-dolt-int-%d", time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	require.NoError(t, dockerClient.NetworkCreate(ctx, docker.NetworkCreateOpts{Name: networkName}))
	t.Cleanup(func() {
		cleanupDockerCommand(t, "network", "rm", networkName)
	})

	t.Cleanup(func() {
		cleanupSharedDoltState(t)
	})

	backend := newIntegrationDoltBackend(dockerClient)
	manager := dolt.NewManagerWithHealthTimeout(backend, 45*time.Second)
	setup := dolt.NewSetup(manager, backend)

	databaseName := fmt.Sprintf("live_%d", time.Now().UnixNano())
	cfg := config.Config{
		Network: networkName,
		Dolt: config.DoltConfig{
			Enabled:  true,
			Port:     3308,
			Image:    "dolthub/dolt-sql-server:latest",
			Database: databaseName,
		},
	}

	envVars, err := setup.EnsureReady(ctx, cfg)
	require.NoError(t, err)
	assert.Equal(t, "1", envVars["BEADS_DOLT_SHARED_SERVER"])
	assert.Equal(t, "havn-dolt", envVars["BEADS_DOLT_SERVER_HOST"])
	assert.Equal(t, databaseName, envVars["BEADS_DOLT_SERVER_DATABASE"])

	status, err := manager.Status(ctx)
	require.NoError(t, err)
	assert.True(t, status.Running)
	assert.Equal(t, "havn-dolt", status.Container)
	assert.True(t, status.ManagedByHavn)

	databases, err := manager.Databases(ctx)
	require.NoError(t, err)
	assert.Contains(t, databases, databaseName)

	require.NoError(t, manager.Stop(ctx))

	statusAfterStop, err := manager.Status(ctx)
	require.NoError(t, err)
	assert.False(t, statusAfterStop.Running)
}

type integrationDoltBackend struct {
	docker *docker.Client
}

func newIntegrationDoltBackend(client *docker.Client) *integrationDoltBackend {
	return &integrationDoltBackend{docker: client}
}

func (b *integrationDoltBackend) ContainerCreate(ctx context.Context, opts dolt.ContainerCreateOpts) (string, error) {
	volumeMounts := make([]docker.VolumeMount, 0, len(opts.Volumes))
	for name, target := range opts.Volumes {
		volumeMounts = append(volumeMounts, docker.VolumeMount{Name: name, Target: target})
	}

	return b.docker.ContainerCreate(ctx, docker.CreateOpts{
		Name:          opts.Name,
		Image:         opts.Image,
		Network:       opts.Network,
		RestartPolicy: opts.Restart,
		Env:           envSliceToMap(opts.Env),
		Labels:        opts.Labels,
		VolumeMounts:  volumeMounts,
	})
}

func (b *integrationDoltBackend) ContainerStart(ctx context.Context, id string) error {
	return b.docker.ContainerStart(ctx, id)
}

func (b *integrationDoltBackend) ContainerStop(ctx context.Context, name string) error {
	return b.docker.ContainerStop(ctx, name, docker.StopOpts{Timeout: 10})
}

func (b *integrationDoltBackend) ContainerInspect(ctx context.Context, name string) (dolt.ContainerInfo, bool, error) {
	info, err := b.docker.ContainerInspect(ctx, name)
	if err != nil {
		var notFound *docker.ContainerNotFoundError
		if errors.As(err, &notFound) {
			return dolt.ContainerInfo{}, false, nil
		}
		return dolt.ContainerInfo{}, false, err
	}

	network := ""
	if len(info.Networks) > 0 {
		network = info.Networks[0]
	}

	return dolt.ContainerInfo{
		ID:      info.ID,
		Running: strings.EqualFold(info.Status, "running"),
		Image:   info.Image,
		Labels:  info.Labels,
		Network: network,
	}, true, nil
}

func (b *integrationDoltBackend) ContainerExec(ctx context.Context, containerName string, cmd []string) (string, error) {
	result, err := b.docker.ContainerExec(ctx, containerName, docker.ExecOpts{Cmd: cmd})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr == "" {
			stderr = "command failed"
		}
		return "", fmt.Errorf("container exec exited %d: %s", result.ExitCode, stderr)
	}
	return string(result.Stdout), nil
}

func (b *integrationDoltBackend) ContainerExecInteractive(_ context.Context, _ string, _ []string) error {
	return fmt.Errorf("interactive exec is not used by these integration tests")
}

func (b *integrationDoltBackend) CopyToContainer(ctx context.Context, containerName string, destPath string, content []byte) error {
	if destPath == "/etc/dolt/servercfg.d" {
		content = tarSingleFile("config.yaml", content)
	}
	return b.docker.CopyToContainer(ctx, containerName, destPath, bytes.NewReader(content))
}

func (b *integrationDoltBackend) CopyFromContainer(ctx context.Context, containerName string, srcPath string) ([]byte, error) {
	rc, err := b.docker.CopyFromContainer(ctx, containerName, srcPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rc.Close()
	}()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, pair := range env {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 1 {
			m[parts[0]] = ""
			continue
		}
		m[parts[0]] = parts[1]
	}
	return m
}

func tarSingleFile(name string, content []byte) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))})
	_, _ = tw.Write(content)
	_ = tw.Close()
	return buf.Bytes()
}

func requireIntegrationDockerClient(t *testing.T) *docker.Client {
	t.Helper()

	c, err := docker.NewClient()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.Ping(ctx); err != nil {
		t.Skipf("docker daemon unavailable: %v", err)
	}

	return c
}

func requireCleanSharedServerState(t *testing.T, c *docker.Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.ContainerInspect(ctx, "havn-dolt")
	if err == nil {
		t.Skip("shared Dolt container already exists; skipping integration test to avoid mutating user state")
		return
	}

	var notFound *docker.ContainerNotFoundError
	if !assert.ErrorAs(t, err, &notFound) {
		t.Skipf("unable to determine shared Dolt state safely: %v", err)
	}
}

func cleanupSharedDoltState(t *testing.T) {
	t.Helper()

	cleanupDockerCommand(t, "rm", "-f", "havn-dolt")
	cleanupDockerCommand(t, "volume", "rm", "-f", "havn-dolt-data", "havn-dolt-config")
}

func cleanupDockerCommand(t *testing.T, args ...string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	_, _ = cmd.CombinedOutput()
}
