package docker_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/docker"
)

func TestVolumeCreateOpts_FieldsExist(t *testing.T) {
	opts := docker.VolumeCreateOpts{
		Name:   "havn-nix",
		Labels: map[string]string{"managed-by": "havn"},
	}

	assert.Equal(t, "havn-nix", opts.Name)
	assert.Equal(t, map[string]string{"managed-by": "havn"}, opts.Labels)
}

func TestVolumeListFilters_FieldsExist(t *testing.T) {
	filters := docker.VolumeListFilters{
		Labels:     map[string]string{"managed-by": "havn"},
		NamePrefix: "havn-",
	}

	assert.Equal(t, map[string]string{"managed-by": "havn"}, filters.Labels)
	assert.Equal(t, "havn-", filters.NamePrefix)
}

func TestVolumeInfo_FieldsExist(t *testing.T) {
	info := docker.VolumeInfo{
		Name:       "havn-nix",
		Driver:     "local",
		Labels:     map[string]string{"managed-by": "havn"},
		Mountpoint: "/var/lib/docker/volumes/havn-nix/_data",
		CreatedAt:  "2026-04-10T12:00:00Z",
	}

	assert.Equal(t, "havn-nix", info.Name)
	assert.Equal(t, "local", info.Driver)
	assert.Equal(t, map[string]string{"managed-by": "havn"}, info.Labels)
	assert.Equal(t, "/var/lib/docker/volumes/havn-nix/_data", info.Mountpoint)
	assert.Equal(t, "2026-04-10T12:00:00Z", info.CreatedAt)
}

func TestVolumeInspect_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	_, err = c.VolumeInspect(context.Background(), "nonexistent")

	assert.Error(t, err)
}

func TestVolumeCreate_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	err = c.VolumeCreate(context.Background(), docker.VolumeCreateOpts{
		Name:   "test-volume",
		Labels: map[string]string{"managed-by": "havn"},
	})

	assert.Error(t, err)
}

func TestVolumeList_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	_, err = c.VolumeList(context.Background(), docker.VolumeListFilters{
		Labels: map[string]string{"managed-by": "havn"},
	})

	assert.Error(t, err)
}
