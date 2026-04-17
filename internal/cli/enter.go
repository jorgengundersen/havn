package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
)

// EnterService is the CLI-facing plain-shell entry dependency for the enter
// command.
type EnterService interface {
	Enter(ctx context.Context, projectPath string) (int, error)
}

type dockerEnterService struct {
	docker *docker.Client
}

func (s dockerEnterService) Enter(ctx context.Context, projectPath string) (int, error) {
	backend := dockerStartBackend(s)
	return container.Enter(ctx, container.EnterDeps{
		Container:   backend,
		Exec:        backend,
		NixRegistry: nixRegistryPreparer{docker: s.docker},
	}, projectPath)
}

func newEnterCmd(service EnterService) *cobra.Command {
	return &cobra.Command{
		Use:   "enter [path]",
		Short: "Enter running container without nix develop",
		Long: "Enter a running project container with plain bash.\n\n" +
			"Home Manager is not activated automatically by `havn enter`. " +
			"For manual Home Manager activation from inside the entered session, run:\n\n" +
			"  nix --extra-experimental-features \"nix-command flakes\" --option keep-build-log true develop <env>#<shell> -c home-manager switch --flake <env>",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if service == nil {
				return fmt.Errorf("havn enter: %w", ErrNotImplemented)
			}

			target := "."
			if len(args) == 1 {
				target = args[0]
			}

			projectCtx, err := projectContextFromStartupTarget(target)
			if err != nil {
				return err
			}

			exitCode, err := service.Enter(cmd.Context(), projectCtx.Path)
			if err != nil {
				return fmt.Errorf("havn enter: %w", err)
			}
			if exitCode != 0 {
				return &ShellExitError{Code: exitCode}
			}

			return nil
		},
	}
}
