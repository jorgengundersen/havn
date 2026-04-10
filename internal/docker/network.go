package docker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockernetwork "github.com/docker/docker/api/types/network"
)

// ErrNetworkAlreadyExists is returned by NetworkCreate when the requested
// network name is already in use.
var ErrNetworkAlreadyExists = errors.New("network already exists")

// ConnectedContainer identifies a container attached to a network.
type ConnectedContainer struct {
	ID   string
	Name string
}

// NetworkInfo holds read-only state of a Docker network.
type NetworkInfo struct {
	Name                string
	ID                  string
	Driver              string
	ConnectedContainers []ConnectedContainer
}

// NetworkCreateOpts holds parameters for creating a network.
type NetworkCreateOpts struct {
	Name   string
	Labels map[string]string
}

// NetworkListFilters holds filter criteria for listing networks.
type NetworkListFilters struct {
	NamePrefix string // network name prefix filter
}

// NetworkCreate creates a named network with the given options. Returns
// ErrNetworkAlreadyExists if a network with the same name already exists.
func (c *Client) NetworkCreate(ctx context.Context, opts NetworkCreateOpts) error {
	_, err := c.docker.NetworkCreate(ctx, opts.Name, dockernetwork.CreateOptions{
		Labels: opts.Labels,
	})
	if err != nil {
		if cerrdefs.IsConflict(err) || cerrdefs.IsAlreadyExists(err) {
			return ErrNetworkAlreadyExists
		}
		return fmt.Errorf("docker network create: %w", err)
	}
	return nil
}

// NetworkInspect returns information about a network by name. Returns
// *NetworkNotFoundError if the network does not exist.
func (c *Client) NetworkInspect(ctx context.Context, name string) (NetworkInfo, error) {
	nw, err := c.docker.NetworkInspect(ctx, name, dockernetwork.InspectOptions{})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return NetworkInfo{}, &NetworkNotFoundError{Name: name}
		}
		return NetworkInfo{}, fmt.Errorf("docker network inspect: %w", err)
	}

	containers := make([]ConnectedContainer, 0, len(nw.Containers))
	for id, ep := range nw.Containers {
		containers = append(containers, ConnectedContainer{
			ID:   id,
			Name: ep.Name,
		})
	}

	return NetworkInfo{
		Name:                nw.Name,
		ID:                  nw.ID,
		Driver:              nw.Driver,
		ConnectedContainers: containers,
	}, nil
}

// NetworkList returns networks matching the given filters. Returns an empty
// slice (not nil) when no networks match.
func (c *Client) NetworkList(ctx context.Context, filters NetworkListFilters) ([]NetworkInfo, error) {
	args := dockerfilters.NewArgs()
	if filters.NamePrefix != "" {
		args.Add("name", filters.NamePrefix)
	}

	networks, err := c.docker.NetworkList(ctx, dockernetwork.ListOptions{
		Filters: args,
	})
	if err != nil {
		return nil, fmt.Errorf("docker network list: %w", err)
	}

	result := make([]NetworkInfo, 0, len(networks))
	for _, nw := range networks {
		if filters.NamePrefix != "" && !strings.HasPrefix(nw.Name, filters.NamePrefix) {
			continue
		}

		containers := make([]ConnectedContainer, 0, len(nw.Containers))
		for id, ep := range nw.Containers {
			containers = append(containers, ConnectedContainer{
				ID:   id,
				Name: ep.Name,
			})
		}

		result = append(result, NetworkInfo{
			Name:                nw.Name,
			ID:                  nw.ID,
			Driver:              nw.Driver,
			ConnectedContainers: containers,
		})
	}

	return result, nil
}
