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
		assert.Equal(t, "github:jorgengundersen/dev-environments", payload["env"])
		assert.Contains(t, payload, "source")
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
