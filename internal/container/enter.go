package container

import (
	"context"
	"errors"
	"fmt"
)

// EnterContainerBackend inspects project containers for plain-shell entry.
type EnterContainerBackend interface {
	ContainerInspect(ctx context.Context, name string) (State, error)
}

// EnterDeps aggregates dependencies for plain-shell container entry.
type EnterDeps struct {
	Container   EnterContainerBackend
	Exec        ExecBackend
	NixRegistry NixRegistryPreparer
}

// Enter attaches to a running project container with a plain bash shell.
func Enter(ctx context.Context, deps EnterDeps, projectPath string) (int, error) {
	cname, err := deriveContainerName(projectPath)
	if err != nil {
		return 0, err
	}

	state, err := deps.Container.ContainerInspect(ctx, string(cname))
	if err != nil {
		var notFound *NotFoundError
		if errors.As(err, &notFound) {
			return 0, &EnterContainerNotRunningError{Name: string(cname), ProjectPath: projectPath, State: "missing"}
		}
		return 0, fmt.Errorf("inspect container %q: %w", cname, err)
	}
	if !state.Running {
		return 0, &EnterContainerNotRunningError{Name: string(cname), ProjectPath: projectPath, State: "stopped"}
	}
	if deps.NixRegistry != nil {
		if err := deps.NixRegistry.Prepare(ctx, string(cname)); err != nil {
			return 0, fmt.Errorf("prepare nix registry aliases in container %q: %w", cname, err)
		}
	}

	return deps.Exec.ContainerExecInteractive(ctx, string(cname), []string{"bash"}, projectPath)
}
