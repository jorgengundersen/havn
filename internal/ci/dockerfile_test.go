package ci_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerfile_BuildSupportsExistingUIDAndGID(t *testing.T) {
	dockerfilePath := filepath.Join("..", "..", "docker", "Dockerfile")

	content, err := os.ReadFile(dockerfilePath)
	require.NoError(t, err)

	dockerfile := string(content)
	assert.Contains(t, dockerfile, "if ! getent group \"${GID}\" >/dev/null 2>&1; then")
	assert.Contains(t, dockerfile, "existing_gid_group=\"$(getent group \"${GID}\" | cut -d: -f1)\"")
	assert.Contains(t, dockerfile, "groupmod --gid \"${GID}\" devuser")
	assert.Contains(t, dockerfile, "existing_uid_user=\"$(getent passwd \"${UID}\" | cut -d: -f1 || true)\"")
	assert.Contains(t, dockerfile, "usermod -l devuser --gid \"${GID}\" --shell /bin/bash \"${existing_uid_user}\"")
	assert.Contains(t, dockerfile, "usermod --home /home/devuser devuser")
	assert.Contains(t, dockerfile, "useradd --uid \"${UID}\" --gid \"${GID}\" --shell /bin/bash --create-home devuser")
	assert.Contains(t, dockerfile, "usermod --uid \"${UID}\" --gid \"${GID}\" --shell /bin/bash devuser")
	assert.Contains(t, dockerfile, "install -d -m 0755 -o devuser -g \"${GID}\" /home/devuser")
	assert.Contains(t, dockerfile, "install -d -m 0755 -o devuser -g \"${GID}\" /nix /nix/var /nix/var/nix")
	assert.Contains(t, dockerfile, "install -d -m 0700 -o devuser -g \"${GID}\" /home/devuser/.ssh")
	assert.Contains(t, dockerfile, "install -d -m 0755 -o devuser -g \"${GID}\" /home/devuser/.local/share /home/devuser/.cache /home/devuser/.local/state")
}
