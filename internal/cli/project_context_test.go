package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectContextFromWorkingDir_ResolvesPathsAndIdentity(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "workspace", "myproject")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectPath))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	ctx, err := projectContextFromWorkingDirForStartup()
	require.NoError(t, err)

	assert.Equal(t, projectPath, ctx.Path)
	assert.Equal(t, filepath.Join(projectPath, ".havn", "config.toml"), ctx.ProjectConfigPath())
	assert.Equal(t, filepath.Join(projectPath, ".havn", "flake.nix"), ctx.ProjectFlakePath())
	assert.Equal(t, filepath.Join(projectPath, ".havn", "environments", "default", "flake.nix"), ctx.ProjectDefaultEnvironmentFlakePath())
	assert.Equal(t, "myproject", ctx.DefaultDoltDatabase())

	containerName, err := ctx.ContainerName()
	require.NoError(t, err)
	assert.Equal(t, "havn-workspace-myproject", containerName)
}

func TestProjectContextFromTarget_AllowsPathOutsideHomeForNonStartupCommands(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := t.TempDir()

	ctx, err := projectContextFromTarget(projectPath)
	require.NoError(t, err)
	assert.Equal(t, projectPath, ctx.Path)
}

func TestProjectContextFromStartupTarget_RejectsPathOutsideHome(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := t.TempDir()

	_, err := projectContextFromStartupTarget(projectPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "under your home directory")
}
