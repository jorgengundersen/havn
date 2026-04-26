package ci_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakefile_BoundaryConfidenceUsesStableMembershipRunner(t *testing.T) {
	makefilePath := filepath.Join("..", "..", "Makefile")
	content, err := os.ReadFile(makefilePath)
	require.NoError(t, err)

	text := string(content)
	assert.Contains(t, text, "go run ./internal/ci/cmd/boundaryconfidence")
	assert.NotContains(t, text, "TestDoctorCommand_")
	assert.NotContains(t, text, "TestSharedDoltLifecycleAndReadiness_Integration")
}
