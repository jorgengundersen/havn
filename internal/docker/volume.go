package docker

import (
	"context"
	"fmt"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockervolume "github.com/docker/docker/api/types/volume"
)

// VolumeInfo holds read-only state of a Docker volume.
type VolumeInfo struct {
	Name       string
	Driver     string
	Labels     map[string]string
	Mountpoint string
	CreatedAt  string
}

// VolumeCreateOpts holds parameters for creating a volume.
type VolumeCreateOpts struct {
	Name   string
	Labels map[string]string
}

// VolumeListFilters holds filter criteria for listing volumes.
type VolumeListFilters struct {
	Labels     map[string]string // label key=value pairs to match
	NamePrefix string            // volume name prefix filter
}

// VolumeInspect returns information about a volume by name. Returns
// *VolumeNotFoundError if the volume does not exist.
func (c *Client) VolumeInspect(ctx context.Context, name string) (VolumeInfo, error) {
	vol, err := c.docker.VolumeInspect(ctx, name)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return VolumeInfo{}, &VolumeNotFoundError{Name: name}
		}
		return VolumeInfo{}, fmt.Errorf("docker volume inspect: %w", err)
	}
	return VolumeInfo{
		Name:       vol.Name,
		Driver:     vol.Driver,
		Labels:     vol.Labels,
		Mountpoint: vol.Mountpoint,
		CreatedAt:  vol.CreatedAt,
	}, nil
}

// VolumeCreate creates a named volume with the given options.
func (c *Client) VolumeCreate(ctx context.Context, opts VolumeCreateOpts) error {
	_, err := c.docker.VolumeCreate(ctx, dockervolume.CreateOptions{
		Name:   opts.Name,
		Labels: opts.Labels,
	})
	if err != nil {
		return fmt.Errorf("docker volume create: %w", err)
	}
	return nil
}

// VolumeList returns volumes matching the given filters. Returns an empty
// slice (not nil) when no volumes match.
func (c *Client) VolumeList(ctx context.Context, filters VolumeListFilters) ([]VolumeInfo, error) {
	args := dockerfilters.NewArgs()
	for k, v := range filters.Labels {
		args.Add("label", k+"="+v)
	}
	if filters.NamePrefix != "" {
		args.Add("name", filters.NamePrefix)
	}

	resp, err := c.docker.VolumeList(ctx, dockervolume.ListOptions{
		Filters: args,
	})
	if err != nil {
		return nil, fmt.Errorf("docker volume list: %w", err)
	}

	result := make([]VolumeInfo, 0, len(resp.Volumes))
	for _, vol := range resp.Volumes {
		if vol == nil {
			continue
		}
		if filters.NamePrefix != "" && !strings.HasPrefix(vol.Name, filters.NamePrefix) {
			continue
		}
		result = append(result, VolumeInfo{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Labels:     vol.Labels,
			Mountpoint: vol.Mountpoint,
			CreatedAt:  vol.CreatedAt,
		})
	}

	return result, nil
}
