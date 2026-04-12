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

func TestCLIAdapters_UsePointerCompileTimeAssertionsForDoctorVolumeAndDoltBackends(t *testing.T) {
	adaptersPath := filepath.Join("..", "cli", "adapters.go")
	adaptersContent, err := os.ReadFile(adaptersPath)
	require.NoError(t, err)

	adapters := string(adaptersContent)
	assert.Contains(t, adapters, "var _ doctor.Backend = (*dockerDoctorBackend)(nil)")
	assert.Contains(t, adapters, "var _ volume.Backend = (*dockerVolumeBackend)(nil)")
	assert.Contains(t, adapters, "var _ dolt.Backend = (*dockerDoltBackend)(nil)")
}

func TestDocs_DoctorTroubleshootingGuideCoversCoreOperationalFlow(t *testing.T) {
	guidePath := filepath.Join("..", "..", "docs", "doctor-troubleshooting.md")
	guideContent, err := os.ReadFile(guidePath)
	require.NoError(t, err)

	guide := string(guideContent)
	assert.Contains(t, guide, "# havn doctor troubleshooting guide")
	assert.Contains(t, guide, "`havn doctor` is the first command to run")
	assert.Contains(t, guide, "## Flags and output modes")
	assert.Contains(t, guide, "## Exit codes")
	assert.Contains(t, guide, "## Troubleshooting flows")
	assert.Contains(t, guide, "### Docker daemon check failed")
	assert.Contains(t, guide, "### Base image check warned")
	assert.Contains(t, guide, "### Dolt server check failed")
	assert.Contains(t, guide, "### Beads health warned")
}

func TestDocs_READMELinksDoctorTroubleshootingGuide(t *testing.T) {
	readmePath := filepath.Join("..", "..", "README.md")
	readmeContent, err := os.ReadFile(readmePath)
	require.NoError(t, err)

	readme := string(readmeContent)
	assert.Contains(t, readme, "[Doctor troubleshooting guide](docs/doctor-troubleshooting.md)")
}

func TestDocs_ContributingGuideCoversDeveloperWorkflow(t *testing.T) {
	contributingPath := filepath.Join("..", "..", "CONTRIBUTING.md")
	contributingContent, err := os.ReadFile(contributingPath)
	require.NoError(t, err)

	contributing := string(contributingContent)
	assert.Contains(t, contributing, "# Contributing to havn")
	assert.Contains(t, contributing, "## Prerequisites")
	assert.Contains(t, contributing, "## Local setup")
	assert.Contains(t, contributing, "## Development workflow")
	assert.Contains(t, contributing, "## Quality gates")
	assert.Contains(t, contributing, "## Repository structure")
	assert.Contains(t, contributing, "## Working with bd issues")
	assert.Contains(t, contributing, "## Pull request workflow")
}

func TestDocs_CLIReferenceDocumentsCommandSurfaceAndSupportMatrix(t *testing.T) {
	cliRefPath := filepath.Join("..", "..", "docs", "cli-reference.md")
	cliRefContent, err := os.ReadFile(cliRefPath)
	require.NoError(t, err)

	cliRef := string(cliRefContent)
	assert.Contains(t, cliRef, "# havn CLI reference")
	assert.Contains(t, cliRef, "## Global flags")
	assert.Contains(t, cliRef, "## Output modes and JSON conventions")
	assert.Contains(t, cliRef, "## Command reference")
	assert.Contains(t, cliRef, "## Support matrix")
	assert.Contains(t, cliRef, "havn [path]")
	assert.Contains(t, cliRef, "havn list")
	assert.Contains(t, cliRef, "havn stop")
	assert.Contains(t, cliRef, "havn build")
	assert.Contains(t, cliRef, "havn config show")
	assert.Contains(t, cliRef, "havn volume list")
	assert.Contains(t, cliRef, "havn doctor")
	assert.Contains(t, cliRef, "havn dolt start")
	assert.Contains(t, cliRef, "havn completion")
	assert.Contains(t, cliRef, "Implemented")
	assert.Contains(t, cliRef, "Planned")
}
