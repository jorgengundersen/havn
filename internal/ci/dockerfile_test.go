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
	assert.Contains(t, dockerfile, "if ! getent group \"${GID}\" >/dev/null; then")
	assert.Contains(t, dockerfile, "existing_user=\"$(getent passwd \"${UID}\" | cut -d: -f1 || true)\"")
	assert.Contains(t, dockerfile, "usermod --login devuser --home /home/devuser --move-home --shell /bin/bash --gid \"${GID}\" \"${existing_user}\"")
	assert.Contains(t, dockerfile, "install -d -m 0700 -o devuser -g \"${GID}\" /home/devuser/.ssh")
	assert.Contains(t, dockerfile, "install -d -m 0755 -o devuser -g \"${GID}\" /home/devuser/.local/share /home/devuser/.cache /home/devuser/.local/state")
}
