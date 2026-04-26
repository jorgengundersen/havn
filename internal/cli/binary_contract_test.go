package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHAVNBinary_CLIContractAtProcessBoundary(t *testing.T) {
	binaryPath := buildHAVNBinary(t)
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	t.Run("config show writes human output to stdout", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "config", "show")

		assert.Equal(t, 0, exitCode)
		assert.Contains(t, stdout, "Configuration:")
		assert.Contains(t, stdout, "Dolt:")
		assert.Empty(t, stderr)
	})

	t.Run("config show writes json output to stdout", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "--json", "config", "show")

		assert.Equal(t, 0, exitCode)
		assert.Empty(t, stderr)

		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
		assert.Equal(t, "path:.", payload["env"])
		assert.Contains(t, payload, "source")
		assert.NotContains(t, payload, "status")
		assert.NotContains(t, payload, "message")
	})

	t.Run("doctor json writes query payload to stdout without action wrapper", func(t *testing.T) {
		homeProjectDir := filepath.Join(homeDir, "work", "sample-project")
		require.NoError(t, os.MkdirAll(homeProjectDir, 0o755))

		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, homeProjectDir, homeDir, "--json", "doctor")

		assert.Contains(t, []int{0, 1, 2}, exitCode)
		assert.Empty(t, stderr)

		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
		assert.Contains(t, payload, "status")
		assert.Contains(t, payload, "summary")
		assert.Contains(t, payload, "checks")
		assert.NotContains(t, payload, "message")
	})

	t.Run("json mode writes errors to stderr with exit code 1", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "--json", "stop")

		assert.Equal(t, 1, exitCode)
		assert.Empty(t, stdout)

		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(stderr), &payload))
		errMsg, ok := payload["error"].(string)
		require.True(t, ok)
		assert.Contains(t, errMsg, "havn stop: requires a container name/path or --all")
	})

	t.Run("human mode writes errors to stderr with exit code 1", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "stop")

		assert.Equal(t, 1, exitCode)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Error: havn stop: requires a container name/path or --all")
	})

	t.Run("namespaced command is reachable", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "volume", "--help")

		assert.Equal(t, 0, exitCode)
		assert.Contains(t, stdout, "Manage havn volumes")
		assert.Contains(t, stdout, "list")
		assert.Empty(t, stderr)
	})

	t.Run("up help exposes startup-check flags on stdout", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "up", "--help")

		assert.Equal(t, 0, exitCode)
		assert.Contains(t, stdout, "--validate")
		assert.Contains(t, stdout, "--prepare")
		assert.Empty(t, stderr)
	})

	t.Run("up validate with shell flag fails on stderr in human mode", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "up", "--validate", "--shell", "zsh")

		assert.Equal(t, 1, exitCode)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Error: unknown flag: --shell")
	})

	t.Run("up prepare with shell flag fails on stderr in json mode", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "--json", "up", "--prepare", "--shell", "zsh")

		assert.Equal(t, 1, exitCode)
		assert.Empty(t, stdout)

		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(stderr), &payload))
		errMsg, ok := payload["error"].(string)
		require.True(t, ok)
		assert.Contains(t, errMsg, "unknown flag: --shell")
	})

	t.Run("json fallback error payload shape is consistent across command errors", func(t *testing.T) {
		tests := []struct {
			name      string
			args      []string
			errSubstr string
		}{
			{name: "action validation error", args: []string{"--json", "stop"}, errSubstr: "requires a container name/path or --all"},
			{name: "flag parsing error", args: []string{"--json", "up", "--validate", "--shell", "zsh"}, errSubstr: "unknown flag: --shell"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, tt.args...)

				assert.Equal(t, 1, exitCode)
				assert.Empty(t, stdout)

				var payload map[string]any
				require.NoError(t, json.Unmarshal([]byte(stderr), &payload))
				errMsg, ok := payload["error"].(string)
				require.True(t, ok)
				assert.Contains(t, errMsg, tt.errSubstr)
				assert.NotContains(t, payload, "type")
				assert.NotContains(t, payload, "details")
			})
		}
	})

	t.Run("up validate and prepare with shell flag fails in human mode", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "up", "--validate", "--prepare", "--shell", "zsh")

		assert.Equal(t, 1, exitCode)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Error: unknown flag: --shell")
	})

	t.Run("up validate and prepare with shell flag fails in json mode", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "--json", "up", "--validate", "--prepare", "--shell", "zsh")

		assert.Equal(t, 1, exitCode)
		assert.Empty(t, stdout)

		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(stderr), &payload))
		errMsg, ok := payload["error"].(string)
		require.True(t, ok)
		assert.Contains(t, errMsg, "unknown flag: --shell")
	})

	t.Run("dolt drop requires explicit confirmation in human mode", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "dolt", "drop", "mydb")

		assert.Equal(t, 1, exitCode)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Error: havn dolt drop requires --yes")
	})

	t.Run("dolt drop requires explicit confirmation in json mode", func(t *testing.T) {
		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, projectDir, homeDir, "--json", "dolt", "drop", "mydb")

		assert.Equal(t, 1, exitCode)
		assert.Empty(t, stdout)

		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(stderr), &payload))
		errMsg, ok := payload["error"].(string)
		require.True(t, ok)
		assert.Equal(t, "havn dolt drop requires --yes", errMsg)
		assert.NotContains(t, payload, "type")
		assert.NotContains(t, payload, "details")
	})

	t.Run("dolt status parse failure reports typed json error payload", func(t *testing.T) {
		brokenProjectDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(brokenProjectDir, ".havn"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(brokenProjectDir, ".havn", "config.toml"), []byte("[dolt\nport = 3308\n"), 0o644))

		stdout, stderr, exitCode := runHAVNBinary(t, binaryPath, brokenProjectDir, homeDir, "--json", "dolt", "status")

		assert.Equal(t, 1, exitCode)
		assert.Empty(t, stdout)

		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(stderr), &payload))
		errMsg, ok := payload["error"].(string)
		require.True(t, ok)
		assert.Contains(t, errMsg, "Config parse error")
		assert.Equal(t, "config_parse_error", payload["type"])
		details, ok := payload["details"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(2), details["line"])
	})

	t.Run("root json reports typed payload for volume-not-found errors", func(t *testing.T) {
		fixtureBinary := buildCLIProcessFixtureBinary(t, "./internal/cli/testdata/volume_not_found")
		homeProjectDir := filepath.Join(homeDir, "work", "volume-project")
		require.NoError(t, os.MkdirAll(homeProjectDir, 0o755))

		stdout, stderr, exitCode := runHAVNBinary(t, fixtureBinary, homeProjectDir, homeDir, "--json")

		assert.Equal(t, 1, exitCode)
		assert.Empty(t, stdout)

		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(stderr), &payload))
		errMsg, ok := payload["error"].(string)
		require.True(t, ok)
		assert.Equal(t, `Volume "havn-dolt-data" not found`, errMsg)
		assert.Equal(t, "volume_not_found", payload["type"])
		details, ok := payload["details"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "havn-dolt-data", details["name"])
	})
}

func buildHAVNBinary(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	binaryPath := filepath.Join(t.TempDir(), "havn-test")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "./cmd/havn")
	buildCmd.Dir = repoRoot
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoErrorf(t, err, "build havn binary: %s", string(buildOutput))

	return binaryPath
}

func buildCLIProcessFixtureBinary(t *testing.T, fixturePath string) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	binaryPath := filepath.Join(t.TempDir(), "havn-fixture")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, fixturePath)
	buildCmd.Dir = repoRoot
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoErrorf(t, err, "build cli fixture binary: %s", string(buildOutput))

	return binaryPath
}

func runHAVNBinary(t *testing.T, binaryPath, workingDir, homeDir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = workingDir
	cmd.Env = append(os.Environ(), "HOME="+homeDir)

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err == nil {
		return stdoutBuf.String(), stderrBuf.String(), 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return stdoutBuf.String(), stderrBuf.String(), exitErr.ExitCode()
	}

	require.NoError(t, err)
	return "", "", 0
}
