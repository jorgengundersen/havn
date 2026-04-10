package docker_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/docker"
)

func TestNetworkInfo_FieldsExist(t *testing.T) {
	info := docker.NetworkInfo{
		Name:   "havn-net",
		ID:     "abc123",
		Driver: "bridge",
		ConnectedContainers: []docker.ConnectedContainer{
			{ID: "cid1", Name: "havn-user-api"},
		},
	}

	assert.Equal(t, "havn-net", info.Name)
	assert.Equal(t, "abc123", info.ID)
	assert.Equal(t, "bridge", info.Driver)
	assert.Len(t, info.ConnectedContainers, 1)
	assert.Equal(t, "cid1", info.ConnectedContainers[0].ID)
	assert.Equal(t, "havn-user-api", info.ConnectedContainers[0].Name)
}

func TestNetworkCreateOpts_FieldsExist(t *testing.T) {
	opts := docker.NetworkCreateOpts{
		Name:   "havn-net",
		Labels: map[string]string{"managed-by": "havn"},
	}

	assert.Equal(t, "havn-net", opts.Name)
	assert.Equal(t, map[string]string{"managed-by": "havn"}, opts.Labels)
}

func TestNetworkListFilters_FieldsExist(t *testing.T) {
	filters := docker.NetworkListFilters{
		NamePrefix: "havn-",
	}

	assert.Equal(t, "havn-", filters.NamePrefix)
}

func TestNetworkInspect_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	_, err = c.NetworkInspect(context.Background(), "nonexistent")

	assert.Error(t, err)
}

func TestNetworkCreate_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	err = c.NetworkCreate(context.Background(), docker.NetworkCreateOpts{
		Name:   "test-network",
		Labels: map[string]string{"managed-by": "havn"},
	})

	assert.Error(t, err)
}

func TestErrNetworkAlreadyExists_IsSentinel(t *testing.T) {
	assert.EqualError(t, docker.ErrNetworkAlreadyExists, "network already exists")
}

func TestNetworkList_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	_, err = c.NetworkList(context.Background(), docker.NetworkListFilters{
		NamePrefix: "havn-",
	})

	assert.Error(t, err)
}
