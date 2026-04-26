package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/dolt"
	"github.com/jorgengundersen/havn/internal/mount"
	"github.com/jorgengundersen/havn/internal/volume"
)

type dockerStartService struct {
	docker *docker.Client
}

func (s dockerStartService) StartOrAttach(ctx context.Context, cfg config.Config, projectPath string, status func(msg string), opts container.StartOptions) (int, error) {
	startBackend := dockerStartBackend(s)
	doltBackend := dockerDoltBackend(s)
	volumeBackend := dockerVolumeBackend(s)
	deps := container.StartDeps{
		Container: startBackend,
		Image: dockerImageBackend{
			docker: s.docker,
			output: io.Discard,
		},
		Network:     startBackend,
		Volume:      volume.NewManager(volumeBackend),
		Mount:       hostMountResolver{},
		Dolt:        dolt.NewSetup(dolt.NewManager(doltBackend), doltBackend),
		Exec:        startBackend,
		NixRegistry: nixRegistryPreparer{docker: s.docker},
		PortChecker: hostPortChecker{},
		Status:      status,
	}

	return container.StartOrAttachWithOptions(ctx, deps, cfg, projectPath, opts)
}

type dockerStartBackend struct {
	docker *docker.Client
}

func (b dockerStartBackend) ContainerInspect(ctx context.Context, name string) (container.State, error) {
	info, err := b.docker.ContainerInspect(ctx, name)
	if err != nil {
		return container.State{}, normalizeContainerBoundaryError(err)
	}

	return container.State{
		ID:      info.ID,
		Running: strings.EqualFold(info.Status, "running"),
	}, nil
}

func (b dockerStartBackend) ContainerCreate(ctx context.Context, opts container.CreateOpts) (string, error) {
	bindMounts := make([]docker.BindMount, 0, len(opts.Mounts))
	volumeMounts := make([]docker.VolumeMount, 0, len(opts.Mounts))
	for _, m := range opts.Mounts {
		if m.Type == "volume" {
			volumeMounts = append(volumeMounts, docker.VolumeMount{Name: m.Source, Target: m.Target})
			continue
		}
		bindMounts = append(bindMounts, docker.BindMount{Source: m.Source, Target: m.Target, ReadOnly: m.ReadOnly})
	}

	id, err := b.docker.ContainerCreate(ctx, docker.CreateOpts{
		Image:        opts.Image,
		Name:         opts.Name,
		Network:      opts.Network,
		Ports:        opts.Ports,
		Env:          opts.Env,
		Labels:       opts.Labels,
		BindMounts:   bindMounts,
		VolumeMounts: volumeMounts,
		Entrypoint:   opts.Entrypoint,
		User:         opts.User,
		CPUs:         opts.CPUs,
		Memory:       opts.Memory,
		MemorySwap:   opts.MemorySwap,
		AutoRemove:   opts.AutoRemove,
	})
	if err != nil {
		return "", normalizeContainerBoundaryError(err)
	}

	return id, nil
}

func (b dockerStartBackend) ContainerStart(ctx context.Context, id string) error {
	err := b.docker.ContainerStart(ctx, id)
	return normalizeContainerBoundaryError(err)
}

func (b dockerStartBackend) NetworkInspect(ctx context.Context, name string) error {
	_, err := b.docker.NetworkInspect(ctx, name)
	return normalizeContainerBoundaryError(err)
}

func (b dockerStartBackend) NetworkCreate(ctx context.Context, name string) error {
	return b.docker.NetworkCreate(ctx, docker.NetworkCreateOpts{Name: name})
}

func (b dockerStartBackend) ContainerExec(ctx context.Context, name string, cmd []string) error {
	result, err := b.docker.ContainerExec(ctx, name, docker.ExecOpts{Cmd: cmd})
	if err != nil {
		return normalizeContainerBoundaryError(err)
	}
	if result.ExitCode != 0 {
		return execResultError(result)
	}
	return nil
}

func (b dockerStartBackend) ContainerExecInteractive(ctx context.Context, name string, cmd []string, workdir string) (int, error) {
	return b.docker.ContainerAttach(ctx, name, docker.AttachOpts{
		Cmd:     cmd,
		Workdir: workdir,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
}

type hostPortChecker struct{}

func (hostPortChecker) EnsureAvailable(ports []string) error {
	for _, mapping := range ports {
		hostPort, ok := hostPortFromMapping(mapping)
		if !ok {
			continue
		}
		ln, err := net.Listen("tcp", net.JoinHostPort("", hostPort))
		if err != nil {
			return fmt.Errorf("requested host port %s is unavailable: %w", hostPort, err)
		}
		_ = ln.Close()
	}
	return nil
}

func hostPortFromMapping(mapping string) (string, bool) {
	parts := strings.SplitN(mapping, ":", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", false
	}
	hostPort := parts[0]
	if slash := strings.IndexRune(hostPort, '/'); slash >= 0 {
		hostPort = hostPort[:slash]
	}
	return hostPort, hostPort != ""
}

type hostMountResolver struct{}

func (hostMountResolver) Resolve(cfg config.Config, projectPath string) (mount.ResolveResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return mount.ResolveResult{}, err
	}

	return mount.Resolve(cfg, projectPath, homeDir, mount.ResolveOpts{
		Glob:        filepath.Glob,
		Exists:      pathExists,
		SSHAuthSock: os.Getenv("SSH_AUTH_SOCK"),
	})
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
