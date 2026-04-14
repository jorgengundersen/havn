package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/doctor"
)

type dockerDoctorBackend struct {
	docker *docker.Client
}

func (b dockerDoctorBackend) Ping(ctx context.Context) error {
	return b.docker.Ping(ctx)
}

func (b dockerDoctorBackend) Info(ctx context.Context) (doctor.RuntimeInfo, error) {
	info, err := b.docker.Info(ctx)
	if err != nil {
		return doctor.RuntimeInfo{}, err
	}
	return doctor.RuntimeInfo{Version: info.ServerVersion, APIVersion: info.APIVersion}, nil
}

func (b dockerDoctorBackend) ImageInspect(ctx context.Context, image string) (doctor.ImageInfo, bool, error) {
	info, err := b.docker.ImageInspect(ctx, image)
	if err != nil {
		var notFound *docker.ImageNotFoundError
		if errors.As(err, &notFound) {
			return doctor.ImageInfo{}, false, nil
		}
		return doctor.ImageInfo{}, false, err
	}
	return doctor.ImageInfo{ID: info.ID, Created: info.CreatedAt}, true, nil
}

func (b dockerDoctorBackend) NetworkInspect(ctx context.Context, network string) (doctor.NetworkInfo, bool, error) {
	info, err := b.docker.NetworkInspect(ctx, network)
	if err != nil {
		var notFound *docker.NetworkNotFoundError
		if errors.As(err, &notFound) {
			return doctor.NetworkInfo{}, false, nil
		}
		return doctor.NetworkInfo{}, false, err
	}

	containers := make([]string, 0, len(info.ConnectedContainers))
	for _, c := range info.ConnectedContainers {
		if c.Name != "" {
			containers = append(containers, c.Name)
		}
	}

	return doctor.NetworkInfo{ContainerCount: len(containers), Containers: containers}, true, nil
}

func (b dockerDoctorBackend) VolumeInspect(ctx context.Context, volumeName string) (bool, error) {
	_, err := b.docker.VolumeInspect(ctx, volumeName)
	if err != nil {
		var notFound *docker.VolumeNotFoundError
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (b dockerDoctorBackend) ContainerInspect(ctx context.Context, name string) (doctor.ContainerInfo, bool, error) {
	info, err := b.docker.ContainerInspect(ctx, name)
	if err != nil {
		var notFound *docker.ContainerNotFoundError
		if errors.As(err, &notFound) {
			return doctor.ContainerInfo{}, false, nil
		}
		return doctor.ContainerInfo{}, false, err
	}

	return doctor.ContainerInfo{
		Running: strings.EqualFold(info.Status, "running"),
		Image:   info.Image,
		Labels:  info.Labels,
	}, true, nil
}

func (b dockerDoctorBackend) ContainerExec(ctx context.Context, containerName string, cmd []string) (string, error) {
	result, err := b.docker.ContainerExec(ctx, containerName, docker.ExecOpts{Cmd: cmd})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", execResultError(result)
	}
	return string(result.Stdout), nil
}

func (b dockerDoctorBackend) ListContainers(ctx context.Context, labels map[string]string) ([]string, error) {
	list, err := b.docker.ContainerList(ctx, docker.ContainerListFilters{
		Labels: labels,
		Status: "running",
	})
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(list))
	for _, c := range list {
		names = append(names, c.Name)
	}

	return names, nil
}
