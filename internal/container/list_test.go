package container_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/name"
)

// fakeContainerBackend records calls and returns configured responses.
type fakeContainerBackend struct {
	containers []container.RawContainer
	listErr    error
}

func (f *fakeContainerBackend) ContainerList(_ context.Context, _ map[string]string) ([]container.RawContainer, error) {
	return f.containers, f.listErr
}

func TestList_EmptyResult(t *testing.T) {
	ctx := context.Background()
	backend := &fakeContainerBackend{}

	got, err := container.List(ctx, backend)

	require.NoError(t, err)
	assert.Empty(t, got)
	assert.NotNil(t, got, "empty result must be [] not nil")
}

func TestList_PopulatesFieldsFromLabels(t *testing.T) {
	ctx := context.Background()
	backend := &fakeContainerBackend{
		containers: []container.RawContainer{
			{
				Name:   "havn-user-api",
				Image:  "havn-base:latest",
				Status: "running",
				Labels: map[string]string{
					"managed-by":  "havn",
					"havn.path":   "/home/devuser/Repos/github.com/user/api",
					"havn.shell":  "go",
					"havn.cpus":   "4",
					"havn.memory": "8g",
					"havn.dolt":   "true",
				},
			},
		},
	}

	got, err := container.List(ctx, backend)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, name.ContainerName("havn-user-api"), got[0].Name)
	assert.Equal(t, "/home/devuser/Repos/github.com/user/api", got[0].Path)
	assert.Equal(t, "havn-base:latest", got[0].Image)
	assert.Equal(t, "running", got[0].Status)
	assert.Equal(t, "go", got[0].Shell)
	assert.Equal(t, 4, got[0].CPUs)
	assert.Equal(t, "8g", got[0].Memory)
	assert.True(t, got[0].Dolt)
}

func TestList_BackendError(t *testing.T) {
	ctx := context.Background()
	backend := &fakeContainerBackend{listErr: assert.AnError}

	got, err := container.List(ctx, backend)

	assert.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, got)
}

func TestList_MultipleContainers(t *testing.T) {
	ctx := context.Background()
	backend := &fakeContainerBackend{
		containers: []container.RawContainer{
			{
				Name:   "havn-user-api",
				Image:  "havn-base:latest",
				Status: "running",
				Labels: map[string]string{
					"managed-by":  "havn",
					"havn.path":   "/home/user/api",
					"havn.shell":  "go",
					"havn.cpus":   "4",
					"havn.memory": "8g",
					"havn.dolt":   "true",
				},
			},
			{
				Name:   "havn-user-web",
				Image:  "havn-base:latest",
				Status: "running",
				Labels: map[string]string{
					"managed-by":  "havn",
					"havn.path":   "/home/user/web",
					"havn.shell":  "node",
					"havn.cpus":   "2",
					"havn.memory": "4g",
					"havn.dolt":   "false",
				},
			},
		},
	}

	got, err := container.List(ctx, backend)

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, name.ContainerName("havn-user-api"), got[0].Name)
	assert.Equal(t, name.ContainerName("havn-user-web"), got[1].Name)
	assert.Equal(t, "node", got[1].Shell)
	assert.False(t, got[1].Dolt)
}

func TestList_MissingLabelsDefaultToZeroValues(t *testing.T) {
	ctx := context.Background()
	backend := &fakeContainerBackend{
		containers: []container.RawContainer{
			{
				Name:   "havn-user-api",
				Image:  "havn-base:latest",
				Status: "running",
				Labels: map[string]string{
					"managed-by": "havn",
				},
			},
		},
	}

	got, err := container.List(ctx, backend)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "", got[0].Path)
	assert.Equal(t, "", got[0].Shell)
	assert.Equal(t, 0, got[0].CPUs)
	assert.Equal(t, "", got[0].Memory)
	assert.False(t, got[0].Dolt)
}
