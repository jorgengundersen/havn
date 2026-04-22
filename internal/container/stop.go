package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
// name.DeriveContainerName. It returns the resolved container name.
func Stop(ctx context.Context, backend StopBackend, target string) (string, error) {
	containerName, err := resolveTarget(target)
	if err != nil {
		return "", fmt.Errorf("resolve target %q: %w", target, err)
	}
	if err := backend.ContainerStop(ctx, containerName, stopTimeout); err != nil {
		return "", err
	}
	return containerName, nil
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

// resolveTarget converts a target to a container name. Path-like targets are
// resolved to existing directories and mapped to deterministic container names.
// Non path-like targets are treated as literal container names.
func resolveTarget(target string) (string, error) {
	if !isPathLikeTarget(target) {
		return target, nil
	}

	absPath, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	absPath = filepath.Clean(absPath)

	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("path does not exist: %s", absPath)
		}
		return "", fmt.Errorf("inspect path %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absPath)
	}

	parent, project, err := name.SplitProjectPath(absPath)
	if err != nil {
		return "", err
	}
	cn, err := name.DeriveContainerName(parent, project)
	if err != nil {
		return "", err
	}
	return string(cn), nil
}

func isPathLikeTarget(target string) bool {
	if filepath.IsAbs(target) || target == "." || target == ".." {
		return true
	}
	return strings.ContainsRune(target, filepath.Separator)
}
