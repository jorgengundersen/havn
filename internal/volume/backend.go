package volume

import "context"

// Backend abstracts the volume operations needed by the volume package.
// Consumer-defined per code-standards §4.
type Backend interface {
	// VolumeInspect checks whether a volume exists. Returns nil if it exists,
	// an error otherwise.
	VolumeInspect(ctx context.Context, name string) error
	// VolumeCreate creates a named volume.
	VolumeCreate(ctx context.Context, name string) error
}
