package ci_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositorySettings_RequireCoreCIChecksForMain(t *testing.T) {
	settingsPath := filepath.Join("..", "..", ".github", "settings.yml")

	content, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	settings := string(content)
	assert.Contains(t, settings, "branches:")
	assert.Contains(t, settings, "- name: main")
	assert.Contains(t, settings, "protection:")
	assert.Contains(t, settings, "required_status_checks:")
	assert.Contains(t, settings, "strict: true")
	assert.Contains(t, settings, "contexts:")
	assert.Contains(t, settings, "- quality-gates")
	assert.Contains(t, settings, "- integration-tests")
}
