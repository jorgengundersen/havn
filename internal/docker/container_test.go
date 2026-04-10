package docker_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/docker"
)

func TestCreateOpts_FieldsExist(t *testing.T) {
	opts := docker.CreateOpts{
		Image:         "havn-base:latest",
		Name:          "havn-user-api",
		Network:       "havn-net",
		Env:           map[string]string{"FOO": "bar"},
		Labels:        map[string]string{"managed-by": "havn"},
		BindMounts:    []docker.BindMount{{Source: "/host", Target: "/container"}},
		VolumeMounts:  []docker.VolumeMount{{Name: "data", Target: "/data"}},
		RestartPolicy: "unless-stopped",
		TTY:           true,
		Workdir:       "/workspace",
		Cmd:           []string{"bash"},
		Entrypoint:    []string{"tini", "--"},
		User:          "devuser",
		CPUs:          2,
		Memory:        "4g",
		MemorySwap:    "4g",
		AutoRemove:    true,
	}

	assert.Equal(t, "havn-base:latest", opts.Image)
	assert.Equal(t, "havn-user-api", opts.Name)
	assert.Equal(t, "havn-net", opts.Network)
	assert.Equal(t, map[string]string{"managed-by": "havn"}, opts.Labels)
	assert.Equal(t, "unless-stopped", opts.RestartPolicy)
	assert.True(t, opts.TTY)
	assert.Equal(t, "/workspace", opts.Workdir)
	assert.Equal(t, "devuser", opts.User)
}

func TestStopOpts_FieldsExist(t *testing.T) {
	opts := docker.StopOpts{
		Timeout: 10,
	}

	assert.Equal(t, 10, opts.Timeout)
}

func TestRemoveOpts_FieldsExist(t *testing.T) {
	opts := docker.RemoveOpts{
		Force:         true,
		RemoveVolumes: true,
	}

	assert.True(t, opts.Force)
	assert.True(t, opts.RemoveVolumes)
}

func TestContainerCreate_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	_, err = c.ContainerCreate(context.Background(), docker.CreateOpts{
		Image: "alpine:latest",
		Name:  "test-container",
	})

	assert.Error(t, err)
}

func TestContainerStart_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	err = c.ContainerStart(context.Background(), "nonexistent")

	assert.Error(t, err)
}

func TestContainerStop_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	err = c.ContainerStop(context.Background(), "nonexistent", docker.StopOpts{Timeout: 5})

	assert.Error(t, err)
}

func TestContainerRemove_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	err = c.ContainerRemove(context.Background(), "nonexistent", docker.RemoveOpts{})

	assert.Error(t, err)
}

func TestEnvSlice(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]string
		want  int
	}{
		{name: "nil map", input: nil, want: 0},
		{name: "empty map", input: map[string]string{}, want: 0},
		{name: "single entry", input: map[string]string{"FOO": "bar"}, want: 1},
		{name: "multiple entries", input: map[string]string{"A": "1", "B": "2"}, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := docker.EnvSlice(tt.input)
			assert.Len(t, got, tt.want)
		})
	}
}

func TestBuildMounts_CombinesBindsAndVolumes(t *testing.T) {
	binds := []docker.BindMount{{Source: "/host", Target: "/container"}}
	volumes := []docker.VolumeMount{{Name: "data", Target: "/data"}}

	got := docker.BuildMounts(binds, volumes)

	assert.Len(t, got, 2)
}

func TestBuildMounts_EmptyInputs(t *testing.T) {
	got := docker.BuildMounts(nil, nil)

	assert.Empty(t, got)
}

func TestEnvSlice_Format(t *testing.T) {
	got := docker.EnvSlice(map[string]string{"FOO": "bar"})

	require.Len(t, got, 1)
	assert.Equal(t, "FOO=bar", got[0])
}

func TestParseMemoryBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{name: "empty string", input: "", want: 0},
		{name: "gigabytes lowercase", input: "4g", want: 4 * 1024 * 1024 * 1024},
		{name: "gigabytes uppercase", input: "4G", want: 4 * 1024 * 1024 * 1024},
		{name: "megabytes lowercase", input: "512m", want: 512 * 1024 * 1024},
		{name: "megabytes uppercase", input: "512M", want: 512 * 1024 * 1024},
		{name: "kilobytes lowercase", input: "1024k", want: 1024 * 1024},
		{name: "plain bytes", input: "1048576", want: 1048576},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := docker.ParseMemoryBytes(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
