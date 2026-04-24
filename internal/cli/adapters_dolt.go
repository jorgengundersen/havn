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

	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/dolt"
)

type dockerDoltBackend struct {
	docker *docker.Client
}

func (b dockerDoltBackend) ContainerCreate(ctx context.Context, opts dolt.ContainerCreateOpts) (string, error) {
	volumeMounts := make([]docker.VolumeMount, 0, len(opts.Volumes))
	for name, target := range opts.Volumes {
		volumeMounts = append(volumeMounts, docker.VolumeMount{Name: name, Target: target})
	}

	id, err := b.docker.ContainerCreate(ctx, docker.CreateOpts{
		Name:          opts.Name,
		Image:         opts.Image,
		Network:       opts.Network,
		RestartPolicy: opts.Restart,
		Env:           envSliceToMap(opts.Env),
		Labels:        opts.Labels,
		VolumeMounts:  volumeMounts,
	})
	if err != nil {
		var imageNotFound *docker.ImageNotFoundError
		if errors.As(err, &imageNotFound) {
			return "", &dolt.ImageNotFoundError{Image: imageNotFound.Name}
		}
		return "", err
	}

	return id, nil
}

func (b dockerDoltBackend) ImagePull(ctx context.Context, image string) error {
	return b.docker.ImagePull(ctx, image, io.Discard)
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

	return toDoltContainerInfo(info), true, nil
}

func (b dockerDoltBackend) ContainerExec(ctx context.Context, containerName string, cmd []string) (string, error) {
	result, err := b.docker.ContainerExec(ctx, containerName, docker.ExecOpts{Cmd: cmd})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", execResultError(result)
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

func toDoltContainerInfo(info docker.ContainerInfo) dolt.ContainerInfo {
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
	}
}
