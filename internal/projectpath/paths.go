package projectpath

import (
	"fmt"
	"path/filepath"
	"strings"
)

const ContainerHome = "/home/devuser"

type ProjectPaths struct {
	HostPath      string
	ContainerPath string
}

type OutsideHomeError struct {
	HostPath string
	HostHome string
}

func (e *OutsideHomeError) Error() string {
	return fmt.Sprintf("project path %q is outside home directory %q", e.HostPath, e.HostHome)
}

func Resolve(hostProjectPath, hostHome string) (ProjectPaths, error) {
	rel, err := filepath.Rel(hostHome, hostProjectPath)
	if err != nil {
		return ProjectPaths{}, fmt.Errorf("resolve project path relative to home: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ProjectPaths{}, &OutsideHomeError{HostPath: hostProjectPath, HostHome: hostHome}
	}

	return ProjectPaths{
		HostPath:      hostProjectPath,
		ContainerPath: filepath.Join(ContainerHome, rel),
	}, nil
}
