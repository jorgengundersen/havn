package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCommand_PrintsHelp(t *testing.T) {
	_, _, err := executeCommand("config")

	require.NoError(t, err)
}

func TestConfigShowCommand_JSONOutputIncludesSourceObject(t *testing.T) {
	stdout, _, err := executeCommand("config", "show", "--json")

	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))

	assert.Contains(t, result, "source")
	src, ok := result["source"].(map[string]any)
	require.True(t, ok, "source should be a JSON object")
	assert.Equal(t, "default", src["shell"])
}

func TestConfigShowCommand_JSONOutputIncludesNestedSourceForResourcesAndDolt(t *testing.T) {
	stdout, _, err := executeCommand("config", "show", "--json")

	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))

	src, ok := result["source"].(map[string]any)
	require.True(t, ok, "source should be a JSON object")

	resources, ok := src["resources"].(map[string]any)
	require.True(t, ok, "source.resources should be a JSON object")
	assert.Equal(t, "default", resources["cpus"])
	assert.Equal(t, "default", resources["memory"])
	assert.Equal(t, "default", resources["memory_swap"])

	dolt, ok := src["dolt"].(map[string]any)
	require.True(t, ok, "source.dolt should be a JSON object")
	assert.Equal(t, "default", dolt["enabled"])
	assert.Equal(t, "default", dolt["database"])
	assert.Equal(t, "default", dolt["port"])
	assert.Equal(t, "default", dolt["image"])
}

func TestConfigShowCommand_UsesConfigFlagForGlobalConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	globalPath := filepath.Join(t.TempDir(), "custom-global.toml")
	require.NoError(t, os.WriteFile(globalPath, []byte("shell = \"zsh\"\n"), 0o644))

	stdout, _, err := executeCommand("config", "show", "--json", "--config", globalPath)

	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	assert.Equal(t, "zsh", result["shell"])

	src := result["source"].(map[string]any)
	assert.Equal(t, "global", src["shell"])
}

func TestConfigShowCommand_JSONOutputMatchesSpecShape(t *testing.T) {
	stdout, _, err := executeCommand("config", "show", "--json")

	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))

	// Top-level fields per spec
	assert.Contains(t, result, "env")
	assert.Contains(t, result, "shell")
	assert.Contains(t, result, "image")
	assert.Contains(t, result, "network")
	assert.Contains(t, result, "resources")
	assert.Contains(t, result, "volumes")
	assert.Contains(t, result, "mounts")
	assert.Contains(t, result, "dolt")
	assert.Contains(t, result, "source")

	// Nested resources fields
	resources, ok := result["resources"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, resources, "cpus")
	assert.Contains(t, resources, "memory")
	assert.Contains(t, resources, "memory_swap")

	// Nested mounts fields
	mounts, ok := result["mounts"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, mounts, "ssh")
	ssh, ok := mounts["ssh"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, ssh, "forward_agent")
	assert.Contains(t, ssh, "authorized_keys")
}

func TestConfigShowCommand_JSONOutputDefaultValues(t *testing.T) {
	stdout, _, err := executeCommand("config", "show", "--json")

	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))

	// Verify default values match config.Default()
	assert.Equal(t, "github:jorgengundersen/dev-environments", result["env"])
	assert.Equal(t, "default", result["shell"])
	assert.Equal(t, "havn-base:latest", result["image"])
	assert.Equal(t, "havn-net", result["network"])

	resources := result["resources"].(map[string]any)
	assert.Equal(t, float64(4), resources["cpus"])
	assert.Equal(t, "8g", resources["memory"])
	assert.Equal(t, "12g", resources["memory_swap"])

	// All source entries should be "default" with no config files
	src := result["source"].(map[string]any)
	assert.Equal(t, "default", src["env"])
	assert.Equal(t, "default", src["shell"])
	assert.Equal(t, "default", src["image"])
	assert.Equal(t, "default", src["network"])

	resourcesSrc := src["resources"].(map[string]any)
	assert.Equal(t, "default", resourcesSrc["cpus"])
	assert.Equal(t, "default", resourcesSrc["memory"])
	assert.Equal(t, "default", resourcesSrc["memory_swap"])

	dolt := src["dolt"].(map[string]any)
	assert.Equal(t, "default", dolt["enabled"])
	assert.Equal(t, "default", dolt["database"])
	assert.Equal(t, "default", dolt["port"])
	assert.Equal(t, "default", dolt["image"])
}

func TestConfigShowCommand_HumanOutputIncludesSourceAnnotations(t *testing.T) {
	stdout, _, err := executeCommand("config", "show")

	require.NoError(t, err)
	assert.Contains(t, stdout, "(default)")
	assert.Contains(t, stdout, "shell:")
	assert.Contains(t, stdout, "cpus:")
	assert.Contains(t, stdout, "Configuration:")
}

func TestConfigShowCommand_EnvOverrideReflectedInSource(t *testing.T) {
	t.Setenv("HAVN_SHELL", "rust")

	stdout, _, err := executeCommand("config", "show", "--json")

	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))

	assert.Equal(t, "rust", result["shell"])
	src := result["source"].(map[string]any)
	assert.Equal(t, "env", src["shell"])
}

func TestConfigShowCommand_ProjectConfigReflectedInSource(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".havn", "config.toml"),
		[]byte("shell = \"go\"\n\n[resources]\ncpus = 16\n"),
		0o644,
	))
	t.Chdir(dir)

	stdout, _, err := executeCommand("config", "show", "--json")

	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))

	assert.Equal(t, "go", result["shell"])
	resources := result["resources"].(map[string]any)
	assert.Equal(t, float64(16), resources["cpus"])

	src := result["source"].(map[string]any)
	assert.Equal(t, "project", src["shell"])
	assert.Equal(t, "default", src["image"])
	resourcesSrc := src["resources"].(map[string]any)
	assert.Equal(t, "project", resourcesSrc["cpus"])
}

func TestConfigShowCommand_JSONOutputIncludesEnvironmentMap(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".havn", "config.toml"),
		[]byte("[environment]\nAPI_TOKEN = \"${API_TOKEN}\"\n"),
		0o644,
	))
	t.Chdir(dir)

	stdout, _, err := executeCommand("config", "show", "--json")

	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))

	envMap, ok := result["environment"].(map[string]any)
	require.True(t, ok, "environment should be a JSON object")
	assert.Equal(t, "${API_TOKEN}", envMap["API_TOKEN"])
}
