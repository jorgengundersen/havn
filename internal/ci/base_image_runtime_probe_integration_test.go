//go:build integration

package ci_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBaseImageRuntimeContract_BASE002_Integration(t *testing.T) {
	requireDockerForBaseImageProbe(t)

	tag := buildBaseImageForProbe(t)
	hostUID := strconv.Itoa(os.Getuid())
	hostGID := strconv.Itoa(os.Getgid())

	probes := []struct {
		name   string
		script string
	}{
		{
			name:   "uses ubuntu 24.04",
			script: "grep -Eq '^VERSION_ID=\"24\\.04\"$' /etc/os-release",
		},
		{
			name:   "provides required runtime binaries",
			script: "command -v nix bash tini sudo sleep /usr/sbin/sshd >/dev/null",
		},
		{
			name:   "configures nix flakes in global config",
			script: "grep -Fxq 'experimental-features = nix-command flakes' /etc/nix/nix.conf",
		},
		{
			name:   "maps devuser uid gid to build args",
			script: fmt.Sprintf("test \"$(id -u devuser)\" = \"%s\" && test \"$(id -g devuser)\" = \"%s\"", hostUID, hostGID),
		},
		{
			name:   "creates required directories",
			script: "for path in /home/devuser /home/devuser/.ssh /run/sshd /nix /home/devuser/.local/share /home/devuser/.cache /home/devuser/.local/state; do test -d \"$path\"; done",
		},
	}

	for _, probe := range probes {
		probe := probe
		t.Run(probe.name, func(t *testing.T) {
			runBaseImageProbeScript(t, tag, probe.script)
		})
	}
}

func TestBaseImageRuntimeContract_BASE004_Integration(t *testing.T) {
	requireDockerForBaseImageProbe(t)

	tag := buildBaseImageForProbe(t)

	probes := []struct {
		name   string
		script string
	}{
		{
			name:   "sshd is configured to listen on container port 22",
			script: "/usr/sbin/sshd -T | grep -Fxq 'port 22'",
		},
		{
			name:   "password authentication is disabled",
			script: "/usr/sbin/sshd -T | grep -Fxq 'passwordauthentication no'",
		},
		{
			name:   "public key authentication is enabled",
			script: "/usr/sbin/sshd -T | grep -Fxq 'pubkeyauthentication yes'",
		},
		{
			name:   "authorized_keys path is honored",
			script: "/usr/sbin/sshd -T | grep -E '^authorizedkeysfile ' | grep -Eq '(^| )\\.ssh/authorized_keys( |$)'",
		},
	}

	for _, probe := range probes {
		probe := probe
		t.Run(probe.name, func(t *testing.T) {
			runBaseImageProbeScript(t, tag, probe.script)
		})
	}

	t.Run("image metadata does not pre-publish ssh host ports", func(t *testing.T) {
		output := inspectBaseImageProbeMetadata(t, tag, "{{json .Config.ExposedPorts}}")
		require.Contains(t, []string{"null", "{}"}, output)
	})
}

func requireDockerForBaseImageProbe(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skipf("docker daemon unavailable: %v", err)
	}
}

func buildBaseImageForProbe(t *testing.T) string {
	t.Helper()

	repoRoot := filepath.Join("..", "..")
	tag := fmt.Sprintf("havn-base-probe:%d", time.Now().UnixNano())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"docker",
		"build",
		"-t", tag,
		"--build-arg", "UID="+strconv.Itoa(os.Getuid()),
		"--build-arg", "GID="+strconv.Itoa(os.Getgid()),
		filepath.Join(repoRoot, "docker"),
	)
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "build base image probe tag %q: %s", tag, string(output))

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanupCancel()
		_ = exec.CommandContext(cleanupCtx, "docker", "rmi", "-f", tag).Run()
	})

	return tag
}

func runBaseImageProbeScript(t *testing.T, tag string, script string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--entrypoint", "sh", tag, "-lc", script)
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "probe failed: %s", string(output))
}

func inspectBaseImageProbeMetadata(t *testing.T, tag string, format string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", "--format", format, tag)
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "inspect failed: %s", string(output))

	return strings.TrimSpace(string(output))
}
