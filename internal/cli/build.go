package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
)

// BuildService is the CLI-facing build dependency.
type BuildService interface {
	Build(ctx context.Context, opts container.BuildOpts, output io.Writer) error
}

type dockerBuildService struct {
	docker *docker.Client
}

type dockerImageBackend struct {
	docker *docker.Client
	output io.Writer
}

func (s dockerBuildService) Build(ctx context.Context, opts container.BuildOpts, output io.Writer) error {
	backend := dockerImageBackend{docker: s.docker, output: output}
	return container.Build(ctx, backend, opts)
}

func (b dockerImageBackend) ImageBuild(ctx context.Context, opts container.ImageBuildOpts) error {
	err := b.docker.ImageBuild(ctx, docker.BuildOpts{
		Tag:        opts.Tag,
		Context:    opts.ContextPath,
		Dockerfile: "Dockerfile",
		BuildArgs:  opts.BuildArgs,
		Output:     b.output,
	})
	if err != nil {
		var notFound *docker.ImageNotFoundError
		if errors.As(err, &notFound) {
			return &container.ImageNotFoundError{Name: notFound.Name}
		}
	}
	return err
}

func (b dockerImageBackend) ImageExists(ctx context.Context, name string) (bool, error) {
	return b.docker.ImageExists(ctx, name)
}

func newBuildCmd(service BuildService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build base image",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if service == nil {
				return fmt.Errorf("havn build: %w", ErrNotImplemented)
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			verbose, _ := cmd.Flags().GetBool("verbose")
			out := NewOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, verbose)

			imageName, err := resolveBuildImageName(cmd)
			if err != nil {
				return fmt.Errorf("havn build: %w", err)
			}

			out.Status("Building base image...")
			if err := service.Build(cmd.Context(), container.BuildOpts{
				ImageName:   imageName,
				ContextPath: "docker/",
				UID:         os.Getuid(),
				GID:         os.Getgid(),
			}, cmd.ErrOrStderr()); err != nil {
				return fmt.Errorf("havn build: %w", err)
			}

			if out.IsJSON() {
				return out.DataJSON(map[string]string{
					"status":  "ok",
					"message": "base image built",
				})
			}
			return nil
		},
	}

	cmd.Flags().String("image", "", "override base image")

	return cmd
}

func resolveBuildImageName(cmd *cobra.Command) (string, error) {
	projectCtx, err := projectContextFromWorkingDir()
	if err != nil {
		return "", err
	}

	globalPath, _ := cmd.Flags().GetString("config")

	var flagOverrides config.Overrides
	if cmd.Flags().Changed("image") {
		flagImage, _ := cmd.Flags().GetString("image")
		flagOverrides.Image = &flagImage
	}

	orchestrator := newEffectiveConfigOrchestrator(globalPath)
	cfg, err := orchestrator.Resolve(projectCtx, flagOverrides)
	if err != nil {
		return "", err
	}

	return cfg.Image, nil
}
