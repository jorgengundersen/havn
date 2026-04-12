package cli

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/doctor"
	"github.com/jorgengundersen/havn/internal/dolt"
	"github.com/jorgengundersen/havn/internal/volume"
)

var _ container.Backend = dockerContainerBackend{}
var _ container.StopBackend = dockerContainerBackend{}
var _ doctor.Backend = dockerDoctorBackend{}
var _ volume.Backend = dockerVolumeBackend{}
var _ dolt.Backend = dockerDoltBackend{}

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
	return b.docker.ContainerStop(ctx, name, docker.StopOpts{Timeout: seconds})
}

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
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr == "" {
			stderr = "command failed"
		}
		return "", fmt.Errorf("container exec exited %d: %s", result.ExitCode, stderr)
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

type dockerVolumeBackend struct {
	docker *docker.Client
}

func (b dockerVolumeBackend) VolumeInspect(ctx context.Context, name string) error {
	_, err := b.docker.VolumeInspect(ctx, name)
	return err
}

func (b dockerVolumeBackend) VolumeCreate(ctx context.Context, name string) error {
	return b.docker.VolumeCreate(ctx, docker.VolumeCreateOpts{Name: name})
}

type dockerDoltBackend struct {
	docker *docker.Client
}

func (b dockerDoltBackend) ContainerCreate(ctx context.Context, opts dolt.ContainerCreateOpts) (string, error) {
	volumeMounts := make([]docker.VolumeMount, 0, len(opts.Volumes))
	for name, target := range opts.Volumes {
		volumeMounts = append(volumeMounts, docker.VolumeMount{Name: name, Target: target})
	}

	return b.docker.ContainerCreate(ctx, docker.CreateOpts{
		Name:          opts.Name,
		Image:         opts.Image,
		Network:       opts.Network,
		RestartPolicy: opts.Restart,
		Env:           envSliceToMap(opts.Env),
		Labels:        opts.Labels,
		VolumeMounts:  volumeMounts,
	})
}

func (b dockerDoltBackend) ContainerStart(ctx context.Context, id string) error {
	return b.docker.ContainerStart(ctx, id)
}

func (b dockerDoltBackend) ContainerStop(ctx context.Context, name string) error {
	return b.docker.ContainerStop(ctx, name, docker.StopOpts{Timeout: 10})
}

func (b dockerDoltBackend) ContainerInspect(ctx context.Context, name string) (dolt.ContainerInfo, bool, error) {
	info, err := b.docker.ContainerInspect(ctx, name)
	if err != nil {
		var notFound *docker.ContainerNotFoundError
		if errors.As(err, &notFound) {
			return dolt.ContainerInfo{}, false, nil
		}
		return dolt.ContainerInfo{}, false, err
	}

	network := ""
	if len(info.Networks) > 0 {
		network = info.Networks[0]
	}

	return dolt.ContainerInfo{
		ID:      info.ID,
		Running: strings.EqualFold(info.Status, "running"),
		Image:   info.Image,
		Labels:  info.Labels,
		Network: network,
	}, true, nil
}

func (b dockerDoltBackend) ContainerExec(ctx context.Context, containerName string, cmd []string) (string, error) {
	result, err := b.docker.ContainerExec(ctx, containerName, docker.ExecOpts{Cmd: cmd})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr == "" {
			stderr = "command failed"
		}
		return "", fmt.Errorf("container exec exited %d: %s", result.ExitCode, stderr)
	}
	return string(result.Stdout), nil
}

func (b dockerDoltBackend) ContainerExecInteractive(ctx context.Context, containerName string, cmd []string) error {
	exitCode, err := b.docker.ContainerAttach(ctx, containerName, docker.AttachOpts{
		Cmd:    cmd,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("container exec exited %d", exitCode)
	}
	return nil
}

func (b dockerDoltBackend) CopyToContainer(ctx context.Context, containerName, destPath string, content []byte) error {
	if destPath == "/etc/dolt/servercfg.d" {
		content = tarSingleFile("config.yaml", content)
	}
	return b.docker.CopyToContainer(ctx, containerName, destPath, bytes.NewReader(content))
}

func (b dockerDoltBackend) CopyFromContainer(ctx context.Context, containerName, srcPath string) ([]byte, error) {
	rc, err := b.docker.CopyFromContainer(ctx, containerName, srcPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, pair := range env {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 1 {
			m[parts[0]] = ""
			continue
		}
		m[parts[0]] = parts[1]
	}
	return m
}

func tarSingleFile(name string, content []byte) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))})
	_, _ = tw.Write(content)
	_ = tw.Close()
	return buf.Bytes()
}
