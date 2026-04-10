package doctor

import "context"

// Backend abstracts the operations doctor checks need from the container runtime.
// Consumer-defined per code-standards §4.
type Backend interface {
	// Ping checks if the container runtime daemon is accessible.
	Ping(ctx context.Context) error
	// Info returns runtime version information.
	Info(ctx context.Context) (RuntimeInfo, error)
	// ImageInspect checks if an image exists. Returns false if not found.
	ImageInspect(ctx context.Context, image string) (ImageInfo, bool, error)
	// NetworkInspect checks if a network exists. Returns false if not found.
	NetworkInspect(ctx context.Context, network string) (NetworkInfo, bool, error)
	// VolumeInspect checks if a volume exists. Returns false if not found.
	VolumeInspect(ctx context.Context, volume string) (bool, error)
	// ContainerInspect returns info about a container. Returns false if not found.
	ContainerInspect(ctx context.Context, name string) (ContainerInfo, bool, error)
	// ContainerExec runs a command inside a container and returns stdout.
	ContainerExec(ctx context.Context, container string, cmd []string) (string, error)
}

// RuntimeInfo holds version data from the container runtime.
type RuntimeInfo struct {
	Version    string
	APIVersion string
}

// ImageInfo holds metadata about a container image.
type ImageInfo struct {
	ID      string
	Created string
}

// NetworkInfo holds metadata about a network.
type NetworkInfo struct {
	ContainerCount int
}

// ContainerInfo holds metadata about a container.
type ContainerInfo struct {
	Running bool
	Image   string
	Labels  map[string]string
}
