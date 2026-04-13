package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jorgengundersen/havn/internal/name"
)

type projectContext struct {
	Path string
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

	return projectContext{Path: absPath}, nil
}

func projectContextFromStartupTarget(target string) (projectContext, error) {
	projectCtx, err := projectContextFromTarget(target)
	if err != nil {
		return projectContext{}, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return projectContext{}, err
	}

	rel, err := filepath.Rel(homeDir, projectCtx.Path)
	if err != nil {
		return projectContext{}, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return projectContext{}, errors.New("project path must be under your home directory")
	}

	return projectCtx, nil
}

func (p projectContext) ProjectConfigPath() string {
	return filepath.Join(p.Path, ".havn", "config.toml")
}

func (p projectContext) ProjectFlakePath() string {
	return filepath.Join(p.Path, ".havn", "flake.nix")
}

func (p projectContext) DefaultDoltDatabase() string {
	return filepath.Base(p.Path)
}

func (p projectContext) ContainerName() (string, error) {
	parent, project, err := name.SplitProjectPath(p.Path)
	if err != nil {
		return "", err
	}

	cname, err := name.DeriveContainerName(parent, project)
	if err != nil {
		return "", err
	}

	return string(cname), nil
}
