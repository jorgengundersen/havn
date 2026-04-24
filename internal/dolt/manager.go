package dolt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jorgengundersen/havn/internal/config"
)

const (
	containerName   = "havn-dolt"
	managedByLabel  = "managed-by"
	managedByValue  = "havn"
	healthPollDelay = 500 * time.Millisecond
	healthTimeout   = 30 * time.Second
)

// Status represents shared Dolt server state used by the CLI layer.
type Status struct {
	Running       bool   `json:"running"`
	Container     string `json:"container,omitempty"`
	Image         string `json:"image,omitempty"`
	Network       string `json:"network,omitempty"`
	ManagedByHavn bool   `json:"managed_by_havn,omitempty"`
}

// Manager handles the Dolt server container lifecycle.
type Manager struct {
	backend       Backend
	healthTimeout time.Duration
}

// StartProgressStage identifies a start-progress lifecycle stage.
type StartProgressStage string

const (
	StartProgressImageAcquisitionStarted StartProgressStage = "image_acquisition_started"
	StartProgressStartupResumed          StartProgressStage = "startup_resumed"
)

// StartProgressEvent captures startup progress state for the Dolt lifecycle.
type StartProgressEvent struct {
	Stage StartProgressStage
	Image string
}

// NewManager creates a Manager with the given backend.
func NewManager(backend Backend) *Manager {
	return &Manager{backend: backend, healthTimeout: healthTimeout}
}

// NewManagerWithHealthTimeout creates a Manager with a custom health check timeout.
func NewManagerWithHealthTimeout(backend Backend, timeout time.Duration) *Manager {
	return &Manager{backend: backend, healthTimeout: timeout}
}

// Start ensures the Dolt server container is running. If the container
// does not exist, it creates one with the config and polls for health.
// If the container exists but lacks the managed-by=havn label, it returns
// *NotManagedError. If the container exists but is stopped, it starts it.
func (m *Manager) Start(ctx context.Context, cfg config.Config) error {
	return m.StartWithProgress(ctx, cfg, nil)
}

// StartWithProgress ensures the Dolt server container is running and emits
// progress events for startup flows that acquire images before retrying.
func (m *Manager) StartWithProgress(ctx context.Context, cfg config.Config, progress func(StartProgressEvent)) error {
	info, found, err := m.backend.ContainerInspect(ctx, containerName)
	if err != nil {
		return &StartError{Err: fmt.Errorf("inspect container: %w", err)}
	}

	if found {
		return m.startExisting(ctx, info)
	}

	return m.startNew(ctx, cfg, progress)
}

func (m *Manager) startExisting(ctx context.Context, info ContainerInfo) error {
	if info.Labels[managedByLabel] != managedByValue {
		return &NotManagedError{Name: containerName}
	}
	if info.Running {
		return m.pollHealth(ctx)
	}
	if err := m.backend.ContainerStart(ctx, info.ID); err != nil {
		return &StartError{Err: fmt.Errorf("start existing container: %w", err)}
	}
	return m.pollHealth(ctx)
}

func (m *Manager) startNew(ctx context.Context, cfg config.Config, progress func(StartProgressEvent)) error {
	createOpts := ContainerCreateOpts{
		Name:    containerName,
		Image:   cfg.Dolt.Image,
		Network: cfg.Network,
		Restart: "unless-stopped",
		Env:     []string{"DOLT_ROOT_HOST=%"},
		Labels:  map[string]string{managedByLabel: managedByValue},
		Volumes: map[string]string{
			"havn-dolt-data":   "/var/lib/dolt",
			"havn-dolt-config": "/etc/dolt/servercfg.d",
		},
	}

	id, err := m.backend.ContainerCreate(ctx, createOpts)
	if err != nil {
		var imageNotFound *ImageNotFoundError
		if !errors.As(err, &imageNotFound) {
			return &StartError{Err: fmt.Errorf("create container: %w", err)}
		}

		reportStartProgress(progress, StartProgressEvent{
			Stage: StartProgressImageAcquisitionStarted,
			Image: imageNotFound.Image,
		})

		if err := m.backend.ImagePull(ctx, imageNotFound.Image); err != nil {
			return &StartError{Err: fmt.Errorf("pull image %q: %w", imageNotFound.Image, err)}
		}

		reportStartProgress(progress, StartProgressEvent{
			Stage: StartProgressStartupResumed,
			Image: imageNotFound.Image,
		})

		id, err = m.backend.ContainerCreate(ctx, createOpts)
		if err != nil {
			return &StartError{Err: fmt.Errorf("create container: %w", err)}
		}
	}

	configData := GenerateConfig(cfg)
	if err := m.backend.CopyToContainer(ctx, containerName, "/etc/dolt/servercfg.d", configData); err != nil {
		return &StartError{Err: fmt.Errorf("copy config: %w", err)}
	}

	if err := m.backend.ContainerStart(ctx, id); err != nil {
		return &StartError{Err: fmt.Errorf("start container: %w", err)}
	}

	return m.pollHealth(ctx)
}

func reportStartProgress(progress func(StartProgressEvent), event StartProgressEvent) {
	if progress == nil {
		return
	}
	progress(event)
}

func (m *Manager) pollHealth(ctx context.Context) error {
	deadline := time.After(m.healthTimeout)
	for {
		select {
		case <-ctx.Done():
			return &StartError{Err: ctx.Err()}
		case <-deadline:
			return &HealthCheckTimeoutError{Timeout: m.healthTimeout}
		default:
			_, err := m.backend.ContainerExec(ctx, containerName, []string{"dolt", "sql", "-q", "SELECT 1"})
			if err == nil {
				return nil
			}
			time.Sleep(healthPollDelay)
		}
	}
}

// Stop stops the Dolt server container.
func (m *Manager) Stop(ctx context.Context) error {
	if err := m.ensureRunningManaged(ctx); err != nil {
		return err
	}

	return m.backend.ContainerStop(ctx, containerName)
}

// Status returns the current state of the Dolt server container.
func (m *Manager) Status(ctx context.Context) (Status, error) {
	info, found, err := m.backend.ContainerInspect(ctx, containerName)
	if err != nil {
		return Status{}, err
	}
	if !found {
		return Status{Running: false}, nil
	}

	return Status{
		Running:       info.Running,
		Container:     containerName,
		Image:         info.Image,
		Network:       info.Network,
		ManagedByHavn: info.Labels[managedByLabel] == managedByValue,
	}, nil
}

func (m *Manager) ensureRunningManaged(ctx context.Context) error {
	info, found, err := m.backend.ContainerInspect(ctx, containerName)
	if err != nil {
		return fmt.Errorf("inspect container: %w", err)
	}
	if found && info.Labels[managedByLabel] != managedByValue {
		return &NotManagedError{Name: containerName}
	}
	if !found || !info.Running {
		return &ServerNotRunningError{Name: containerName}
	}
	return nil
}
