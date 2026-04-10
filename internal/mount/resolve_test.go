package mount_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/mount"
)

func noopOpts() mount.ResolveOpts {
	return mount.ResolveOpts{
		Glob:   func(string) ([]string, error) { return nil, nil },
		Exists: func(string) bool { return false },
	}
}

func TestResolve_ConfigMountExistingFile(t *testing.T) {
	cfg := config.Default()
	cfg.Mounts.Config = []string{".gitconfig:ro"}

	opts := noopOpts()
	opts.Exists = func(path string) bool {
		return path == "/home/user/.gitconfig"
	}
	opts.Glob = func(pattern string) ([]string, error) {
		if pattern == "/home/user/.gitconfig" {
			return []string{"/home/user/.gitconfig"}, nil
		}
		return nil, nil
	}

	result, err := mount.Resolve(cfg, "/projects/api", "/home/user", opts)
	require.NoError(t, err)

	want := mount.Spec{
		Source:   "/home/user/.gitconfig",
		Target:   "/home/devuser/.gitconfig",
		ReadOnly: true,
		Type:     "bind",
	}
	assert.Contains(t, result.Mounts, want)
}

func TestResolve_SSHAgentForwarding(t *testing.T) {
	cfg := config.Default()
	// ForwardAgent is true by default.

	opts := noopOpts()
	opts.SSHAuthSock = "/tmp/ssh-agent.sock"
	opts.Exists = func(path string) bool {
		return path == "/tmp/ssh-agent.sock"
	}

	result, err := mount.Resolve(cfg, "/projects/api", "/home/user", opts)
	require.NoError(t, err)

	assert.Contains(t, result.Mounts, mount.Spec{
		Source: "/tmp/ssh-agent.sock", Target: "/ssh-agent", ReadOnly: true, Type: "bind",
	})
	assert.Equal(t, "/ssh-agent", result.Env["SSH_AUTH_SOCK"])
}

func TestResolve_SSHAgentEmptySocket(t *testing.T) {
	cfg := config.Default()

	opts := noopOpts()
	// SSHAuthSock is empty.

	result, err := mount.Resolve(cfg, "/projects/api", "/home/user", opts)
	require.NoError(t, err)

	for _, m := range result.Mounts {
		assert.NotEqual(t, "/ssh-agent", m.Target, "no SSH mount when socket is empty")
	}
	assert.Empty(t, result.Env["SSH_AUTH_SOCK"])
}

func TestResolve_SSHAuthorizedKeys(t *testing.T) {
	cfg := config.Default()
	// AuthorizedKeys is true by default.

	opts := noopOpts()
	opts.Exists = func(path string) bool {
		return path == "/home/user/.ssh/authorized_keys"
	}

	result, err := mount.Resolve(cfg, "/projects/api", "/home/user", opts)
	require.NoError(t, err)

	assert.Contains(t, result.Mounts, mount.Spec{
		Source: "/home/user/.ssh/authorized_keys", Target: "/home/devuser/.ssh/authorized_keys", ReadOnly: true, Type: "bind",
	})
}

func TestResolve_SSHAuthorizedKeysSkippedWhenMissing(t *testing.T) {
	cfg := config.Default()

	result, err := mount.Resolve(cfg, "/projects/api", "/home/user", noopOpts())
	require.NoError(t, err)

	for _, m := range result.Mounts {
		assert.NotContains(t, m.Target, "authorized_keys")
	}
}

func TestResolve_MalformedEntryMissingMode(t *testing.T) {
	cfg := config.Default()
	cfg.Mounts.Config = []string{".gitconfig"}

	_, err := mount.Resolve(cfg, "/projects/api", "/home/user", noopOpts())

	var invalid *mount.InvalidMountEntryError
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, ".gitconfig", invalid.Entry)
}

func TestResolve_MalformedEntryUnknownMode(t *testing.T) {
	cfg := config.Default()
	cfg.Mounts.Config = []string{".gitconfig:wx"}

	_, err := mount.Resolve(cfg, "/projects/api", "/home/user", noopOpts())

	var invalid *mount.InvalidMountEntryError
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, ".gitconfig:wx", invalid.Entry)
}

func TestResolve_ConfigMountWildcard(t *testing.T) {
	cfg := config.Default()
	cfg.Mounts.Config = []string{".gitconfig-*:ro"}

	opts := noopOpts()
	opts.Glob = func(pattern string) ([]string, error) {
		if pattern == "/home/user/.gitconfig-*" {
			return []string{"/home/user/.gitconfig-work", "/home/user/.gitconfig-personal"}, nil
		}
		return nil, nil
	}
	opts.Exists = func(path string) bool {
		return path == "/home/user/.gitconfig-work" || path == "/home/user/.gitconfig-personal"
	}

	result, err := mount.Resolve(cfg, "/projects/api", "/home/user", opts)
	require.NoError(t, err)

	assert.Contains(t, result.Mounts, mount.Spec{
		Source: "/home/user/.gitconfig-work", Target: "/home/devuser/.gitconfig-work", ReadOnly: true, Type: "bind",
	})
	assert.Contains(t, result.Mounts, mount.Spec{
		Source: "/home/user/.gitconfig-personal", Target: "/home/devuser/.gitconfig-personal", ReadOnly: true, Type: "bind",
	})
}

func TestResolve_ConfigMountSkippedWhenMissing(t *testing.T) {
	cfg := config.Default()
	cfg.Mounts.Config = []string{".gitconfig:ro"}

	opts := noopOpts()
	opts.Glob = func(_ string) ([]string, error) {
		return []string{"/home/user/.gitconfig"}, nil
	}
	// Exists returns false — file not on host.

	result, err := mount.Resolve(cfg, "/projects/api", "/home/user", opts)
	require.NoError(t, err)

	for _, m := range result.Mounts {
		assert.NotEqual(t, "/home/user/.gitconfig", m.Source, "missing file should not be mounted")
	}
}

func TestResolve_NamedVolumes(t *testing.T) {
	cfg := config.Default()
	result, err := mount.Resolve(cfg, "/projects/api", "/home/user", noopOpts())
	require.NoError(t, err)

	want := []mount.Spec{
		{Source: "havn-nix", Target: "/nix", ReadOnly: false, Type: "volume"},
		{Source: "havn-data", Target: "/home/devuser/.local/share", ReadOnly: false, Type: "volume"},
		{Source: "havn-cache", Target: "/home/devuser/.cache", ReadOnly: false, Type: "volume"},
		{Source: "havn-state", Target: "/home/devuser/.local/state", ReadOnly: false, Type: "volume"},
	}
	for _, w := range want {
		assert.Contains(t, result.Mounts, w)
	}
}

func TestResolve_ProjectDirectoryAlwaysPresent(t *testing.T) {
	cfg := config.Default()
	result, err := mount.Resolve(cfg, "/home/user/projects/api", "/home/user", noopOpts())
	require.NoError(t, err)

	want := mount.Spec{
		Source:   "/home/user/projects/api",
		Target:   "/home/user/projects/api",
		ReadOnly: false,
		Type:     "bind",
	}
	assert.Contains(t, result.Mounts, want)
}
