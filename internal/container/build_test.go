package container_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/container"
)

// fakeImageBackend records calls to ImageBuild and returns configured responses.
type fakeImageBackend struct {
	buildOpts container.ImageBuildOpts
	buildErr  error

	existsResult bool
	existsErr    error
}

func (f *fakeImageBackend) ImageBuild(_ context.Context, opts container.ImageBuildOpts) error {
	f.buildOpts = opts
	return f.buildErr
}

func (f *fakeImageBackend) ImageExists(_ context.Context, _ string) (bool, error) {
	return f.existsResult, f.existsErr
}

func TestBuild_ErrorWrapped(t *testing.T) {
	ctx := context.Background()
	backendErr := assert.AnError
	backend := &fakeImageBackend{buildErr: backendErr}

	err := container.Build(ctx, backend, container.BuildOpts{
		ImageName:   "havn-base:latest",
		ContextPath: "docker/",
		UID:         1000,
		GID:         1000,
	})

	var buildErr *container.BuildError
	require.ErrorAs(t, err, &buildErr)
	assert.ErrorIs(t, buildErr, backendErr)
}

func TestBuild_PassesCorrectBuildOpts(t *testing.T) {
	tests := []struct {
		name string
		opts container.BuildOpts
		want container.ImageBuildOpts
	}{
		{
			name: "standard uid/gid",
			opts: container.BuildOpts{
				ImageName:   "havn-base:latest",
				ContextPath: "docker/",
				UID:         1000,
				GID:         1000,
			},
			want: container.ImageBuildOpts{
				Tag:         "havn-base:latest",
				ContextPath: "docker/",
				BuildArgs:   map[string]string{"UID": "1000", "GID": "1000"},
			},
		},
		{
			name: "non-standard uid/gid",
			opts: container.BuildOpts{
				ImageName:   "custom:v2",
				ContextPath: "/path/to/context",
				UID:         501,
				GID:         20,
			},
			want: container.ImageBuildOpts{
				Tag:         "custom:v2",
				ContextPath: "/path/to/context",
				BuildArgs:   map[string]string{"UID": "501", "GID": "20"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			backend := &fakeImageBackend{}

			err := container.Build(ctx, backend, tt.opts)
			require.NoError(t, err)

			assert.Equal(t, tt.want.Tag, backend.buildOpts.Tag)
			assert.Equal(t, tt.want.ContextPath, backend.buildOpts.ContextPath)
			assert.Equal(t, tt.want.BuildArgs, backend.buildOpts.BuildArgs)
		})
	}
}
