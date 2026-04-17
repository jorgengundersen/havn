package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
)

type nixRegistryPreparer struct {
	docker nixRegistryRuntimeBackend
}

type nixRegistryRuntimeBackend interface {
	ContainerExec(ctx context.Context, nameOrID string, opts docker.ExecOpts) (docker.ExecResult, error)
	CopyToContainer(ctx context.Context, nameOrID string, dstPath string, tarStream io.Reader) error
}

const (
	stateRegistryPath    = "/home/devuser/.local/state/nix/registry.json"
	legacyRegistryPath   = "/home/devuser/.config/nix/registry.json"
	nixUserConfigDir     = "/home/devuser/.config/nix"
	emptyRegistryContent = `{"version":2,"flakes":[]}`
)

func (p nixRegistryPreparer) Prepare(ctx context.Context, containerName string) error {
	registryData, registryExists, err := p.readFileIfExists(ctx, containerName, stateRegistryPath)
	if err != nil {
		return err
	}

	if !registryExists {
		registryData = []byte(emptyRegistryContent)
	}

	legacyData, legacyExists, err := p.readFileIfExists(ctx, containerName, legacyRegistryPath)
	if err != nil {
		return err
	}

	merged, changed, err := container.MergeNixRegistryAliases(registryData, []byte(emptyRegistryContent), stateRegistryPath, "built-in defaults")
	if err != nil {
		return err
	}
	registryData = merged

	if legacyExists {
		merged, mergeChanged, mergeErr := container.MergeNixRegistryAliases(registryData, legacyData, stateRegistryPath, legacyRegistryPath)
		if mergeErr != nil {
			return mergeErr
		}
		registryData = merged
		changed = changed || mergeChanged
	}

	if !registryExists {
		changed = true
	}

	if !changed {
		return p.ensureLegacyRegistrySymlink(ctx, containerName)
	}

	if err := p.ensureDirectory(ctx, containerName, path.Dir(stateRegistryPath)); err != nil {
		return err
	}
	if err := p.writeFile(ctx, containerName, stateRegistryPath, registryData); err != nil {
		return err
	}

	return p.ensureLegacyRegistrySymlink(ctx, containerName)
}

func (p nixRegistryPreparer) ensureLegacyRegistrySymlink(ctx context.Context, containerName string) error {
	if err := p.ensureDirectory(ctx, containerName, nixUserConfigDir); err != nil {
		return err
	}

	cmd := []string{"sh", "-c", "ln -sfn " + shellQuote(stateRegistryPath) + " " + shellQuote(legacyRegistryPath)}
	result, err := p.docker.ContainerExec(ctx, containerName, docker.ExecOpts{Cmd: cmd})
	if err != nil {
		return fmt.Errorf("wire nix registry symlink %q -> %q in container %q: %w", legacyRegistryPath, stateRegistryPath, containerName, err)
	}
	if result.ExitCode != 0 {
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr == "" {
			stderr = fmt.Sprintf("exit code %d", result.ExitCode)
		}
		return fmt.Errorf("wire nix registry symlink %q -> %q in container %q: %s", legacyRegistryPath, stateRegistryPath, containerName, stderr)
	}

	return nil
}

func (p nixRegistryPreparer) readFileIfExists(ctx context.Context, containerName, filePath string) ([]byte, bool, error) {
	exists, err := p.fileExists(ctx, containerName, filePath)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	result, err := p.docker.ContainerExec(ctx, containerName, docker.ExecOpts{Cmd: []string{"sh", "-c", "cat " + shellQuote(filePath)}})
	if err != nil {
		return nil, false, fmt.Errorf("read nix registry file %q in container %q: %w", filePath, containerName, err)
	}
	if result.ExitCode != 0 {
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr == "" {
			stderr = fmt.Sprintf("exit code %d", result.ExitCode)
		}
		return nil, false, fmt.Errorf("read nix registry file %q in container %q: %s (check file permissions and file contents)", filePath, containerName, stderr)
	}

	return result.Stdout, true, nil
}

func (p nixRegistryPreparer) fileExists(ctx context.Context, containerName, filePath string) (bool, error) {
	result, err := p.docker.ContainerExec(ctx, containerName, docker.ExecOpts{Cmd: []string{"sh", "-c", "test -f " + shellQuote(filePath)}})
	if err != nil {
		return false, fmt.Errorf("check nix registry file %q in container %q: %w", filePath, containerName, err)
	}
	if result.ExitCode == 0 {
		return true, nil
	}
	if result.ExitCode == 1 {
		return false, nil
	}

	stderr := strings.TrimSpace(string(result.Stderr))
	if stderr == "" {
		stderr = fmt.Sprintf("exit code %d", result.ExitCode)
	}
	return false, fmt.Errorf("check nix registry file %q in container %q: %s", filePath, containerName, stderr)
}

func (p nixRegistryPreparer) ensureDirectory(ctx context.Context, containerName, dirPath string) error {
	result, err := p.docker.ContainerExec(ctx, containerName, docker.ExecOpts{Cmd: []string{"sh", "-c", "mkdir -p " + shellQuote(dirPath)}})
	if err != nil {
		return fmt.Errorf("create nix registry directory %q in container %q: %w", dirPath, containerName, err)
	}
	if result.ExitCode != 0 {
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr == "" {
			stderr = fmt.Sprintf("exit code %d", result.ExitCode)
		}
		return fmt.Errorf("create nix registry directory %q in container %q: %s", dirPath, containerName, stderr)
	}
	return nil
}

func (p nixRegistryPreparer) writeFile(ctx context.Context, containerName, filePath string, content []byte) error {
	dirPath := path.Dir(filePath)
	fileName := path.Base(filePath)
	tarData := tarSingleFile(fileName, content)
	if err := p.docker.CopyToContainer(ctx, containerName, dirPath, bytes.NewReader(tarData)); err != nil {
		return fmt.Errorf("write nix registry file %q in container %q: %w", filePath, containerName, err)
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
