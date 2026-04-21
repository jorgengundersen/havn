package ci_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIWorkflow_CoreAndIntegrationJobsConfigured(t *testing.T) {
	workflow := requireCIWorkflow(t)

	require.Contains(t, workflow.On, "push")
	require.Contains(t, workflow.On, "pull_request")

	qualityGatesJob := workflow.requiredJob(t, "quality-gates")
	assert.Contains(t, qualityGatesJob.runCommands(), "make check")

	integrationTestsJob := workflow.requiredJob(t, "integration-tests")
	assert.Equal(t, "quality-gates", integrationTestsJob.Needs)
	assert.Contains(t, integrationTestsJob.runCommands(), "docker info")
	assert.Contains(t, integrationTestsJob.runCommands(), "make test-integration")

	boundaryConfidenceJob := workflow.requiredJob(t, "boundary-confidence")
	assert.Equal(t, "quality-gates", boundaryConfidenceJob.Needs)
	assert.Contains(t, boundaryConfidenceJob.runCommands(), "docker info")
	assert.Contains(t, boundaryConfidenceJob.runCommands(), "make test-boundary-confidence")
}

func TestSmokeWorkflow_ManualNonAuthoritativeCrossRepoValidation(t *testing.T) {
	workflow := requireWorkflow(t, "smoke-cross-repo.yml")

	assert.Contains(t, workflow.On, "workflow_dispatch")
	assert.NotContains(t, workflow.On, "push")
	assert.NotContains(t, workflow.On, "pull_request")

	smokeJob := workflow.requiredJob(t, "cross-repo-smoke")
	assert.True(t, smokeJob.ContinueOnError)
	runCommands := smokeJob.runCommands()
	require.NotEmpty(t, runCommands)
	assert.Contains(t, runCommands[0], "nix eval \"$ref#devShells.${system}.default.drvPath\" >/dev/null")
}
