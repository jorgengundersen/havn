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

func TestSpecs_PortExposureContractIsExplicitAndCoherent(t *testing.T) {
	overviewPath := filepath.Join("..", "..", "specs", "havn-overview.md")
	overviewContent, err := os.ReadFile(overviewPath)
	require.NoError(t, err)

	overview := string(overviewContent)
	assert.Contains(t, overview, "`--port` accepts a single host port number")
	assert.Contains(t, overview, "`ports` accepts explicit host:container mappings")
	assert.Contains(t, overview, "`--port` and `ports` are merged into one Docker publish set")
	assert.Contains(t, overview, "A startup request fails if any requested host port is unavailable")

	cliPath := filepath.Join("..", "..", "specs", "cli-framework.md")
	cliContent, err := os.ReadFile(cliPath)
	require.NoError(t, err)

	cli := string(cliContent)
	assert.Contains(t, cli, "root.Flags().StringVar(&opts.Port, \"port\", \"\", \"publish container SSH on host port\")")
}

func TestSpecs_InterfaceAssertionImplementorStrategyIsExplicit(t *testing.T) {
	codeStandardsPath := filepath.Join("..", "..", "specs", "code-standards.md")
	codeStandardsContent, err := os.ReadFile(codeStandardsPath)
	require.NoError(t, err)

	codeStandards := string(codeStandardsContent)
	assert.Contains(t, codeStandards, "Current Docker-adjacent implementors")
	assert.Contains(t, codeStandards, "`container.Backend`, `container.StopBackend` -> `cli.dockerContainerBackend`")
	assert.Contains(t, codeStandards, "`doctor.Backend` -> `cli.dockerDoctorBackend`")
	assert.Contains(t, codeStandards, "`volume.Backend` -> `cli.dockerVolumeBackend`")
	assert.Contains(t, codeStandards, "`dolt.Backend` -> `cli.dockerDoltBackend`")
	assert.Contains(t, codeStandards, "The assertions belong on these adapter types")
}
