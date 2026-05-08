package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jorgengundersen/havn/internal/name"
	"github.com/jorgengundersen/havn/internal/projectpath"
)

type projectContext struct {
	HostPath      string
	ContainerPath string
}

func projectContextFromWorkingDir() (projectContext, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return projectContext{}, err
	}

	return projectContextFromTarget(cwd)
}

func projectContextFromWorkingDirForStartup() (projectContext, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return projectContext{}, err
	}

	return projectContextFromStartupTarget(cwd)
}

func projectContextFromTarget(target string) (projectContext, error) {
	absPath, err := filepath.Abs(target)
	if err != nil {
		return projectContext{}, err
	}
	absPath = filepath.Clean(absPath)

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return projectContext{}, fmt.Errorf("directory not found: %s", absPath)
	}

	ctx := projectContext{HostPath: absPath}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ctx, nil
	}

	paths, err := projectpath.Resolve(absPath, homeDir)
	if err != nil {
		var outsideHomeErr *projectpath.OutsideHomeError
		if errors.As(err, &outsideHomeErr) {
			return ctx, nil
		}
		return projectContext{}, err
	}
	ctx.ContainerPath = paths.ContainerPath

	return ctx, nil
}

func projectContextFromStartupTarget(target string) (projectContext, error) {
	projectCtx, err := projectContextFromTarget(target)
	if err != nil {
		return projectContext{}, err
	}
	if projectCtx.ContainerPath == "" {
		return projectContext{}, errors.New("project path must be under your home directory")
	}

	return projectCtx, nil
}

func (p projectContext) ProjectConfigPath() string {
	return filepath.Join(p.HostPath, ".havn", "config.toml")
}

func (p projectContext) ProjectFlakePath() string {
	return filepath.Join(p.HostPath, ".havn", "flake.nix")
}

func (p projectContext) ProjectDefaultEnvironmentFlakePath() string {
	return filepath.Join(p.HostPath, ".havn", "environments", "default", "flake.nix")
}

func (p projectContext) DefaultDoltDatabase() string {
	return filepath.Base(p.HostPath)
}

func (p projectContext) ContainerName() (string, error) {
	parent, project, err := name.SplitProjectPath(p.HostPath)
	if err != nil {
		return "", err
	}

	cname, err := name.DeriveContainerName(parent, project)
	if err != nil {
		return "", err
	}

	return string(cname), nil
}
