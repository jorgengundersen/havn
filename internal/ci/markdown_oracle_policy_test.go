package ci_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepositoryCI_DoesNotUseMarkdownFilesAsTestOracle(t *testing.T) {
	violations, err := collectMarkdownOracleViolations()
	if err != nil {
		t.Fatalf("collect markdown oracle violations: %v", err)
	}

	assert.Empty(t, violations)
}
