package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
)

func TestEffectiveConfigOrchestrator_Resolve_UsesSharedPrecedence(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	globalPath := filepath.Join(t.TempDir(), "global.toml")
	require.NoError(t, os.WriteFile(globalPath, []byte("shell = \"zsh\"\n"), 0o644))

	projectPath := filepath.Join(homeDir, "workspace", "sample")
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte("shell = \"go\"\n"), 0o644))

	t.Setenv("HAVN_SHELL", "rust")
	flagShell := "fish"

	orchestrator := newEffectiveConfigOrchestrator(globalPath)
	cfg, src, err := orchestrator.ResolveWithSource(projectContext{Path: projectPath}, config.Overrides{Shell: &flagShell})

	require.NoError(t, err)
	assert.Equal(t, "fish", cfg.Shell)
	assert.Equal(t, "flag", src["shell"])
}

func TestEffectiveConfigOrchestrator_ResolveWithSource_ErrorsOnMalformedEnvOverride(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("HAVN_CPUS", "not-a-number")

	globalPath := filepath.Join(t.TempDir(), "global.toml")
	require.NoError(t, os.WriteFile(globalPath, []byte(""), 0o644))

	projectPath := filepath.Join(homeDir, "workspace", "sample")
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte(""), 0o644))

	orchestrator := newEffectiveConfigOrchestrator(globalPath)
	_, _, err := orchestrator.ResolveWithSource(projectContext{Path: projectPath}, config.Overrides{})

	require.Error(t, err)
	assert.ErrorContains(t, err, "HAVN_CPUS")
}
