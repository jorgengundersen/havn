package container

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jorgengundersen/havn/internal/name"
)

// StopBackend abstracts container operations needed by Stop and StopAll.
type StopBackend interface {
	Backend
	ContainerStop(ctx context.Context, name string, timeout time.Duration) error
}

// StopFailure records a single container that failed to stop.
type StopFailure struct {
	Name string
	Err  error
}

// StopResult collects the outcome of a StopAll operation.
type StopResult struct {
	Stopped []string
	Failed  []StopFailure
}

// stopTimeout is the grace period given to a container before it is killed.
const stopTimeout = 10 * time.Second

// Stop stops a single container by name. If target is a filesystem path,
// the container name is derived first via name.SplitProjectPath and
// name.DeriveContainerName.
func Stop(ctx context.Context, backend StopBackend, target string) error {
	containerName, err := resolveTarget(target)
	if err != nil {
		return fmt.Errorf("resolve target %q: %w", target, err)
	}
	return backend.ContainerStop(ctx, containerName, stopTimeout)
}

// doltContainerName is the shared Dolt container excluded from StopAll.
const doltContainerName = "havn-dolt"

// StopAll stops all havn-managed containers except the shared Dolt container.
// Best-effort: a failure to stop one container does not abort the rest.
func StopAll(ctx context.Context, backend StopBackend) (StopResult, error) {
	containers, err := List(ctx, backend)
	if err != nil {
		return StopResult{}, fmt.Errorf("list containers: %w", err)
	}

	var result StopResult
	for _, c := range containers {
		if string(c.Name) == doltContainerName {
			continue
		}
		n := string(c.Name)
		if err := backend.ContainerStop(ctx, n, stopTimeout); err != nil {
			result.Failed = append(result.Failed, StopFailure{Name: n, Err: err})
		} else {
			result.Stopped = append(result.Stopped, n)
		}
	}
	return result, nil
}

// resolveTarget converts a target to a container name. If the target is an
// absolute path, it derives the container name from the path segments.
// Otherwise, it treats the target as a literal container name.
func resolveTarget(target string) (string, error) {
	if !filepath.IsAbs(target) {
		return target, nil
	}
	parent, project, err := name.SplitProjectPath(target)
	if err != nil {
		return "", err
	}
	cn, err := name.DeriveContainerName(parent, project)
	if err != nil {
		return "", err
	}
	return string(cn), nil
}
