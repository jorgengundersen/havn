// Package dolt manages the shared Dolt SQL server container lifecycle.
package dolt

import "context"

// Backend abstracts the container runtime operations needed by the Dolt manager.
// Consumer-defined per code-standards §4.
type Backend interface {
	// ContainerCreate creates a container with the given options and returns its ID.
	ContainerCreate(ctx context.Context, opts ContainerCreateOpts) (string, error)
	// ContainerStart starts an existing container by ID.
	ContainerStart(ctx context.Context, id string) error
	// ContainerStop stops a running container by name.
	ContainerStop(ctx context.Context, name string) error
	// ContainerInspect returns info about a container. Returns false if not found.
	ContainerInspect(ctx context.Context, name string) (ContainerInfo, bool, error)
	// ContainerExec runs a one-shot command inside a container and returns stdout.
	ContainerExec(ctx context.Context, container string, cmd []string) (string, error)
	// CopyToContainer copies data into a container at the given path.
	CopyToContainer(ctx context.Context, container string, destPath string, content []byte) error
}

// ContainerCreateOpts holds the parameters for creating a container.
type ContainerCreateOpts struct {
	Name    string
	Image   string
	Network string
	Restart string
	Env     []string
	Labels  map[string]string
	Volumes map[string]string // host/volume -> container path
}

// ContainerInfo holds metadata about a container.
type ContainerInfo struct {
	ID      string
	Running bool
	Image   string
	Labels  map[string]string
	Network string
	Port    int
}
