package ci_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIWorkflow_CoreAndIntegrationJobsConfigured(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "ci.yml")

	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)

	workflow := string(content)
	assert.Contains(t, workflow, "on:")
	assert.Contains(t, workflow, "push:")
	assert.Contains(t, workflow, "pull_request:")
	assert.Contains(t, workflow, "make check")
	assert.Contains(t, workflow, "make test-integration")
	assert.Contains(t, workflow, "docker info")
}
