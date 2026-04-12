package ci_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecs_ConfigContractDocumentsStableSourceAndConfigOnlyOverrides(t *testing.T) {
	overviewPath := filepath.Join("..", "..", "specs", "havn-overview.md")
	overviewContent, err := os.ReadFile(overviewPath)
	require.NoError(t, err)

	overview := string(overviewContent)
	assert.Contains(t, overview, "The `source` object is part of the stable `havn config show --json` contract")
	assert.Contains(t, overview, "havn must include source metadata for these fields")
	assert.Contains(t, overview, "`memory_swap` is a config-only setting")

	principlesPath := filepath.Join("..", "..", "specs", "architecture-principles.md")
	principlesContent, err := os.ReadFile(principlesPath)
	require.NoError(t, err)

	principles := string(principlesContent)
	assert.Contains(t, principles, "Intentional config-only settings are valid")
	assert.Contains(t, principles, "be config-only when users need stable project/global defaults")
}
