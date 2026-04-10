// Package container manages container lifecycle operations.
package container

import (
	"context"
	"strconv"
)

// ImageBackend abstracts the image operations needed by the container package.
// Consumer-defined per code-standards §4.
type ImageBackend interface {
	// ImageBuild builds an image from a Dockerfile context.
	ImageBuild(ctx context.Context, opts ImageBuildOpts) error
	// ImageExists checks whether an image with the given name exists locally.
	ImageExists(ctx context.Context, name string) (bool, error)
}

// ImageBuildOpts holds the parameters for building an image.
type ImageBuildOpts struct {
	Tag         string
	ContextPath string
	BuildArgs   map[string]string
}

// BuildOpts holds the parameters for Build.
type BuildOpts struct {
	ImageName   string
	ContextPath string
	UID         int
	GID         int
}

// Build builds the base image using the given backend.
func Build(ctx context.Context, backend ImageBackend, opts BuildOpts) error {
	err := backend.ImageBuild(ctx, ImageBuildOpts{
		Tag:         opts.ImageName,
		ContextPath: opts.ContextPath,
		BuildArgs: map[string]string{
			"UID": strconv.Itoa(opts.UID),
			"GID": strconv.Itoa(opts.GID),
		},
	})
	if err != nil {
		return &BuildError{Err: err}
	}
	return nil
}
