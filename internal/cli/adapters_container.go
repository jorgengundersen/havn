package cli

import (
	"context"
	"time"

	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
)

type dockerContainerBackend struct {
	docker *docker.Client
}

func (b dockerContainerBackend) ContainerList(ctx context.Context, filters map[string]string) ([]container.RawContainer, error) {
	list, err := b.docker.ContainerList(ctx, docker.ContainerListFilters{Labels: filters})
	if err != nil {
		return nil, err
	}

	out := make([]container.RawContainer, 0, len(list))
	for _, c := range list {
		out = append(out, container.RawContainer{
			Name:   c.Name,
			Image:  c.Image,
			Status: c.Status,
			Labels: c.Labels,
		})
	}

	return out, nil
}

func (b dockerContainerBackend) ContainerStop(ctx context.Context, name string, timeout time.Duration) error {
	seconds := int(timeout / time.Second)
	if seconds <= 0 {
		seconds = 1
	}
	err := b.docker.ContainerStop(ctx, name, docker.StopOpts{Timeout: seconds})
	return normalizeContainerBoundaryError(err)
}
