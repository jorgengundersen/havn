package docker_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/docker"
)

func TestBuildOpts_FieldsExist(t *testing.T) {
	var output bytes.Buffer
	opts := docker.BuildOpts{
		Tag:        "havn-base:latest",
		Context:    "/path/to/context",
		Dockerfile: "Dockerfile",
		BuildArgs:  map[string]string{"UID": "1000"},
		Output:     &output,
	}

	assert.Equal(t, "havn-base:latest", opts.Tag)
	assert.Equal(t, "/path/to/context", opts.Context)
	assert.Equal(t, "Dockerfile", opts.Dockerfile)
	assert.Equal(t, map[string]string{"UID": "1000"}, opts.BuildArgs)
	assert.Equal(t, &output, opts.Output)
}

func TestImageBuild_ReturnsErrorOnUnreachableDaemon(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	var output bytes.Buffer
	err = c.ImageBuild(context.Background(), docker.BuildOpts{
		Tag:     "test:latest",
		Context: t.TempDir(),
		Output:  &output,
	})

	assert.Error(t, err)
}

func TestImageBuild_WrapsErrorWithContext(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	var output bytes.Buffer
	err = c.ImageBuild(context.Background(), docker.BuildOpts{
		Tag:     "test:latest",
		Context: t.TempDir(),
		Output:  &output,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "docker build")
}

func TestImageBuild_RespectsContextCancellation(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var output bytes.Buffer
	err = c.ImageBuild(ctx, docker.BuildOpts{
		Tag:     "test:latest",
		Context: t.TempDir(),
		Output:  &output,
	})

	assert.Error(t, err)
}

func TestImageBuild_InvalidContextPath(t *testing.T) {
	c, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	var output bytes.Buffer
	err = c.ImageBuild(context.Background(), docker.BuildOpts{
		Tag:     "test:latest",
		Context: "/nonexistent/path/that/does/not/exist",
		Output:  &output,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "docker build")
}

func TestImageBuildError_ImplementsError(t *testing.T) {
	err := &docker.ImageBuildError{Tag: "test:latest", Detail: "step 3 failed"}

	var target error = err
	assert.Equal(t, `image build "test:latest" failed: step 3 failed`, target.Error())
}

func TestImageBuildError_TypedError(t *testing.T) {
	err := &docker.ImageBuildError{Tag: "test:latest", Detail: "step 3 failed"}

	assert.Equal(t, "image_build_failed", err.ErrorType())
	assert.Equal(t, map[string]any{"tag": "test:latest", "detail": "step 3 failed"}, err.ErrorDetails())
}

func TestImageBuildError_EmptyTag(t *testing.T) {
	err := &docker.ImageBuildError{Detail: "syntax error"}

	assert.Equal(t, "image build failed: syntax error", err.Error())
}
