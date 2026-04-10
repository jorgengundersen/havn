package container

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/mount"
	"github.com/jorgengundersen/havn/internal/name"
)

// StartBackend abstracts container lifecycle operations for StartOrAttach.
type StartBackend interface {
	ContainerInspect(ctx context.Context, name string) (State, error)
	ContainerCreate(ctx context.Context, opts CreateOpts) (string, error)
	ContainerStart(ctx context.Context, id string) error
}

// State holds the result of inspecting a container.
type State struct {
	ID      string
	Running bool
}

// NetworkBackend abstracts network operations for StartOrAttach.
type NetworkBackend interface {
	NetworkInspect(ctx context.Context, name string) error
	NetworkCreate(ctx context.Context, name string) error
}

// VolumeEnsurer ensures named volumes exist.
type VolumeEnsurer interface {
	EnsureExists(ctx context.Context, name string) error
}

// MountResolver resolves mount specifications for container creation.
type MountResolver interface {
	Resolve(cfg config.Config, projectPath string) (mount.ResolveResult, error)
}

// DoltSetup ensures the Dolt server and project database are ready.
type DoltSetup interface {
	EnsureReady(ctx context.Context, cfg config.Config) (map[string]string, error)
}

// ExecBackend runs commands inside containers.
type ExecBackend interface {
	ContainerExec(ctx context.Context, name string, cmd []string) error
	ContainerExecInteractive(ctx context.Context, name string, cmd []string, workdir string) (int, error)
}

// CreateOpts holds parameters for creating a container.
type CreateOpts struct {
	Name       string
	Image      string
	Network    string
	Mounts     []mount.Spec
	Env        map[string]string
	Labels     map[string]string
	Entrypoint []string
	User       string
	CPUs       int
	Memory     string
	MemorySwap string
	AutoRemove bool
}

// StartDeps aggregates all dependencies for StartOrAttach.
type StartDeps struct {
	Container StartBackend
	Image     ImageBackend
	Network   NetworkBackend
	Volume    VolumeEnsurer
	Mount     MountResolver
	Dolt      DoltSetup // nil to skip Dolt setup
	Exec      ExecBackend
	Status    func(msg string)
}

// StartOrAttach implements the startup orchestration (steps 3-10).
// If the container is already running, it attaches to it. Otherwise, it
// ensures all infrastructure exists, creates the container, runs init,
// and attaches to the devShell. Returns the exit code from the shell session.
func StartOrAttach(ctx context.Context, deps StartDeps, cfg config.Config, projectPath string) (int, error) {
	cname, err := deriveContainerName(projectPath)
	if err != nil {
		return 0, err
	}

	// Step 3: check if container is running.
	state, err := deps.Container.ContainerInspect(ctx, string(cname))
	if err != nil {
		var notFound *NotFoundError
		if !errors.As(err, &notFound) {
			return 0, fmt.Errorf("inspect container %q: %w", cname, err)
		}
		// Container doesn't exist — proceed to create.
	} else if state.Running {
		return attach(ctx, deps.Exec, string(cname), cfg, projectPath)
	}

	// Steps 4-7: ensure infrastructure.
	if err := ensureInfrastructure(ctx, deps, cfg); err != nil {
		return 0, err
	}

	// Step 8: create and start container.
	id, err := createContainer(ctx, deps, cfg, string(cname), projectPath)
	if err != nil {
		return 0, err
	}
	if err := deps.Container.ContainerStart(ctx, id); err != nil {
		return 0, fmt.Errorf("start container %q: %w", cname, err)
	}

	// Step 9: post-start init (sshd, best-effort).
	_ = deps.Exec.ContainerExec(ctx, string(cname), sshdCmd)

	// Step 10: attach to devShell.
	return attach(ctx, deps.Exec, string(cname), cfg, projectPath)
}

var sshdCmd = []string{"sudo", "/usr/sbin/sshd"}

const containerUser = "devuser"

func ensureInfrastructure(ctx context.Context, deps StartDeps, cfg config.Config) error {
	// Step 4: ensure base image.
	exists, err := deps.Image.ImageExists(ctx, cfg.Image)
	if err != nil {
		return fmt.Errorf("check image %q: %w", cfg.Image, err)
	}
	if !exists {
		deps.Status(fmt.Sprintf("Image %s not found, building...", cfg.Image))
		if err := deps.Image.ImageBuild(ctx, ImageBuildOpts{Tag: cfg.Image}); err != nil {
			return &BuildError{Err: err}
		}
	}

	// Step 5: ensure network.
	if err := deps.Network.NetworkInspect(ctx, cfg.Network); err != nil {
		deps.Status(fmt.Sprintf("Network %s not found, creating...", cfg.Network))
		if err := deps.Network.NetworkCreate(ctx, cfg.Network); err != nil {
			return fmt.Errorf("create network %q: %w", cfg.Network, err)
		}
	}

	// Step 6: ensure volumes.
	volumes := []string{cfg.Volumes.Nix, cfg.Volumes.Data, cfg.Volumes.Cache, cfg.Volumes.State}
	for _, v := range volumes {
		if err := deps.Volume.EnsureExists(ctx, v); err != nil {
			return fmt.Errorf("ensure volume %q: %w", v, err)
		}
	}

	// Step 7: Dolt setup (if enabled).
	// Handled in createContainer for env var injection.
	return nil
}

func createContainer(ctx context.Context, deps StartDeps, cfg config.Config, cname, projectPath string) (string, error) {
	// Resolve mounts.
	mountResult, err := deps.Mount.Resolve(cfg, projectPath)
	if err != nil {
		return "", fmt.Errorf("resolve mounts: %w", err)
	}

	// Build env vars.
	env := make(map[string]string)
	for k, v := range mountResult.Env {
		env[k] = v
	}
	for k, v := range cfg.Environment {
		env[k] = v
	}

	// Step 7: Dolt setup.
	if deps.Dolt != nil && cfg.Dolt.Enabled {
		doltEnv, err := deps.Dolt.EnsureReady(ctx, cfg)
		if err != nil {
			return "", err
		}
		for k, v := range doltEnv {
			env[k] = v
		}
	}

	// Build labels.
	labels := map[string]string{
		LabelManagedBy: "havn",
		LabelPath:      projectPath,
		LabelShell:     cfg.Shell,
		LabelCPUs:      strconv.Itoa(cfg.Resources.CPUs),
		LabelMemory:    cfg.Resources.Memory,
		LabelDolt:      strconv.FormatBool(cfg.Dolt.Enabled),
	}

	opts := CreateOpts{
		Name:       cname,
		Image:      cfg.Image,
		Network:    cfg.Network,
		Mounts:     mountResult.Mounts,
		Env:        env,
		Labels:     labels,
		Entrypoint: []string{"tini", "--", "sleep", "infinity"},
		User:       containerUser,
		CPUs:       cfg.Resources.CPUs,
		Memory:     cfg.Resources.Memory,
		MemorySwap: cfg.Resources.MemorySwap,
		AutoRemove: true,
	}

	return deps.Container.ContainerCreate(ctx, opts)
}

func deriveContainerName(projectPath string) (name.ContainerName, error) {
	parent, project, err := name.SplitProjectPath(projectPath)
	if err != nil {
		return "", fmt.Errorf("derive container name: %w", err)
	}
	cname, err := name.DeriveContainerName(parent, project)
	if err != nil {
		return "", fmt.Errorf("derive container name: %w", err)
	}
	return cname, nil
}

func attach(ctx context.Context, exec ExecBackend, containerName string, cfg config.Config, projectPath string) (int, error) {
	return exec.ContainerExecInteractive(ctx, containerName, shellCmd(cfg), projectPath)
}

func shellCmd(cfg config.Config) []string {
	return []string{"nix", "develop", cfg.Env + "#" + cfg.Shell, "-c", "bash"}
}
