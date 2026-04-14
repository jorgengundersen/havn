package cli

import (
	"context"
	"errors"

	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/volume"
)

type dockerVolumeBackend struct {
	docker *docker.Client
}

func (b dockerVolumeBackend) VolumeInspect(ctx context.Context, name string) error {
	_, err := b.docker.VolumeInspect(ctx, name)
	if err != nil {
		var notFound *docker.VolumeNotFoundError
		if errors.As(err, &notFound) {
			return &volume.NotFoundError{Name: notFound.Name}
		}
	}
	return err
}

func (b dockerVolumeBackend) VolumeCreate(ctx context.Context, name string) error {
	return b.docker.VolumeCreate(ctx, docker.VolumeCreateOpts{Name: name})
}
