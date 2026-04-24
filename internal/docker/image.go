package docker

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
)

// ImageInfo holds read-only metadata about a Docker image.
type ImageInfo struct {
	ID        string
	Tag       string
	CreatedAt string
}

// ImageInspect returns metadata about a Docker image by name or ID. Returns
// *ImageNotFoundError if the image does not exist locally.
func (c *Client) ImageInspect(ctx context.Context, name string) (ImageInfo, error) {
	resp, err := c.docker.ImageInspect(ctx, name)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return ImageInfo{}, &ImageNotFoundError{Name: name}
		}
		return ImageInfo{}, fmt.Errorf("docker image inspect: %w", err)
	}

	tag := name
	if len(resp.RepoTags) > 0 {
		tag = resp.RepoTags[0]
	}

	return ImageInfo{
		ID:        resp.ID,
		Tag:       tag,
		CreatedAt: resp.Created,
	}, nil
}

// ImageExists checks whether an image exists locally. Returns (false, nil)
// when the image is not found — not an error. Other failures (e.g., daemon
// unreachable) are returned as errors.
func (c *Client) ImageExists(ctx context.Context, name string) (bool, error) {
	_, err := c.docker.ImageInspect(ctx, name)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("docker image exists: %w", err)
	}
	return true, nil
}

// ImagePull pulls an image reference from the configured registry and streams
// daemon output to output when provided.
func (c *Client) ImagePull(ctx context.Context, ref string, output io.Writer) error {
	rc, err := c.docker.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("docker image pull: %w", err)
	}
	defer func() { _ = rc.Close() }()

	if output == nil {
		output = io.Discard
	}

	if _, err := io.Copy(output, rc); err != nil {
		return fmt.Errorf("docker image pull: read output: %w", err)
	}

	return nil
}

// BuildOpts holds parameters for building a Docker image.
type BuildOpts struct {
	Tag        string            // image tag to apply
	Context    string            // path to build context directory
	Dockerfile string            // Dockerfile path relative to context
	BuildArgs  map[string]string // build-time variables
	Output     io.Writer         // streaming build output destination
}

// ImageBuild builds a Docker image from the given build context. Build output
// is streamed to opts.Output. Returns *ImageBuildError if the build itself
// fails (Dockerfile error, etc.).
func (c *Client) ImageBuild(ctx context.Context, opts BuildOpts) error {
	contextDir := opts.Context
	if contextDir == "" {
		contextDir = "."
	}

	absContext, err := filepath.Abs(contextDir)
	if err != nil {
		return fmt.Errorf("docker build: resolve context path: %w", err)
	}

	info, err := os.Stat(absContext)
	if err != nil {
		return fmt.Errorf("docker build: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("docker build: context path %q is not a directory", absContext)
	}

	pr, pw := io.Pipe()
	go func() {
		pw.CloseWithError(tarDir(absContext, pw))
	}()

	dockerfile := opts.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}

	buildArgs := make(map[string]*string, len(opts.BuildArgs))
	for k, v := range opts.BuildArgs {
		v := v
		buildArgs[k] = &v
	}

	resp, err := c.docker.ImageBuild(ctx, pr, build.ImageBuildOptions{
		Tags:       []string{opts.Tag},
		Dockerfile: dockerfile,
		BuildArgs:  buildArgs,
		Version:    build.BuilderV1,
		Remove:     true,
	})
	if err != nil {
		return fmt.Errorf("docker build: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return streamBuildOutput(resp.Body, opts.Output)
}

// tarDir writes a tar archive of dir to w using stdlib archive/tar.
func tarDir(dir string, w io.Writer) error {
	tw := tar.NewWriter(w)

	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)

		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			header.Linkname = target
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		return copyFileToTar(tw, path)
	})
	if walkErr != nil {
		_ = tw.Close()
		return walkErr
	}
	return tw.Close()
}

func copyFileToTar(tw *tar.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = io.Copy(tw, f)
	return err
}

// streamBuildOutput reads Docker build JSON messages from r, writes human-
// readable output to w, and returns *ImageBuildError if the build failed.
func streamBuildOutput(r io.Reader, w io.Writer) error {
	if w == nil {
		w = io.Discard
	}

	dec := json.NewDecoder(r)
	for {
		var msg struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("docker build: decode output: %w", err)
		}
		if msg.Error != "" {
			return &ImageBuildError{
				Detail: msg.Error,
			}
		}
		if msg.Stream != "" {
			if _, err := fmt.Fprint(w, msg.Stream); err != nil {
				return fmt.Errorf("docker build: write output: %w", err)
			}
		}
	}
}
