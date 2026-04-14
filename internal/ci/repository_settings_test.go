package ci_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepositorySettings_RequireCoreCIChecksForMain(t *testing.T) {
	settings := requireRepositorySettings(t)

	mainBranch := settings.requiredMainBranchProtection(t)
	assert.True(t, mainBranch.Protection.RequiredStatusChecks.Strict)
	assert.ElementsMatch(t, []string{"quality-gates", "integration-tests", "boundary-confidence"}, mainBranch.Protection.RequiredStatusChecks.Contexts)
}
