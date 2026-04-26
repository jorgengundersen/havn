package cli

import (
	"errors"

	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
)

func normalizeContainerBoundaryError(err error) error {
	if err == nil {
		return nil
	}

	var containerNotFound *docker.ContainerNotFoundError
	if errors.As(err, &containerNotFound) {
		return &container.NotFoundError{Name: containerNotFound.Name}
	}

	var imageNotFound *docker.ImageNotFoundError
	if errors.As(err, &imageNotFound) {
		return &container.ImageNotFoundError{Name: imageNotFound.Name}
	}

	var networkNotFound *docker.NetworkNotFoundError
	if errors.As(err, &networkNotFound) {
		return &container.NetworkNotFoundError{Name: networkNotFound.Name}
	}

	return err
}
