package docker

import (
	"context"
	"fmt"
	"io"

	cerrdefs "github.com/containerd/errdefs"
	dockercontainer "github.com/docker/docker/api/types/container"
)

// CopyToContainer copies a tar stream into the container at dstPath.
// Returns *ContainerNotFoundError if the container does not exist.
func (c *Client) CopyToContainer(ctx context.Context, nameOrID string, dstPath string, tarStream io.Reader) error {
	err := c.docker.CopyToContainer(ctx, nameOrID, dstPath, tarStream, dockercontainer.CopyToContainerOptions{})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return &ContainerNotFoundError{Name: nameOrID}
		}
		return fmt.Errorf("docker copy to container: %w", err)
	}
	return nil
}

// CopyFromContainer returns a tar stream of the contents at srcPath inside
// the container. The caller must close the returned ReadCloser.
// Returns *ContainerNotFoundError if the container does not exist.
func (c *Client) CopyFromContainer(ctx context.Context, nameOrID string, srcPath string) (io.ReadCloser, error) {
	rc, _, err := c.docker.CopyFromContainer(ctx, nameOrID, srcPath)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return nil, &ContainerNotFoundError{Name: nameOrID}
		}
		return nil, fmt.Errorf("docker copy from container: %w", err)
	}
	return rc, nil
}
