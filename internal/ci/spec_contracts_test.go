package ci_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecs_ConfigContractDocumentsStableSourceAndConfigOnlyOverrides(t *testing.T) {
	configPath := filepath.Join("..", "..", "specs", "configuration.md")
	configContent, err := os.ReadFile(configPath)
	require.NoError(t, err)

	configuration := string(configContent)
	assert.Contains(t, configuration, "The `source` object is part of the stable contract")
	assert.Contains(t, configuration, "`havn` must include source metadata for at least these fields")
	assert.Contains(t, configuration, "`memory_swap` is intentionally config-only")

	principlesPath := filepath.Join("..", "..", "specs", "architecture-principles.md")
	principlesContent, err := os.ReadFile(principlesPath)
	require.NoError(t, err)

	principles := string(principlesContent)
	assert.Contains(t, principles, "Intentional config-only settings are valid")
	assert.Contains(t, principles, "be config-only when users need stable project/global defaults")
}

func TestSpecs_PortExposureContractIsExplicitAndCoherent(t *testing.T) {
	configPath := filepath.Join("..", "..", "specs", "configuration.md")
	configContent, err := os.ReadFile(configPath)
	require.NoError(t, err)

	configuration := string(configContent)
	assert.Contains(t, configuration, "`--port` is SSH-only")
	assert.Contains(t, configuration, "merged with any configured `ports` entries into the final Docker publish set")
	assert.Contains(t, configuration, "Startup fails if any requested host port in the final Docker publish set is not")

	cliPath := filepath.Join("..", "..", "specs", "cli-framework.md")
	cliContent, err := os.ReadFile(cliPath)
	require.NoError(t, err)

	cli := string(cliContent)
	assert.Contains(t, cli, "Root-only flags apply only to `havn [path]` and are not inherited by")
	assert.Contains(t, cli, "- `--port <port>`")
}

func TestSpecs_StartupPathAndConfigMountHomeScopeAreExplicit(t *testing.T) {
	configPath := filepath.Join("..", "..", "specs", "configuration.md")
	configContent, err := os.ReadFile(configPath)
	require.NoError(t, err)

	configuration := string(configContent)
	assert.Contains(t, configuration, "For `havn [path]` startup, the resolved project path must be under the user's home directory")
	assert.Contains(t, configuration, "`mounts.config` entries must resolve to paths under the user's home directory")
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
	assert.Contains(t, contributing, "`integration-tests` and `boundary-confidence` are required merge checks on `main`")
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
	assert.Contains(t, cliRef, "planned, not part of the")
	assert.Contains(t, cliRef, "Implemented")
	assert.Contains(t, cliRef, "Partial")
	assert.Contains(t, cliRef, "Planned")
}

func TestDocs_PartialDerivativeGuidesCallOutConcreteCurrentGaps(t *testing.T) {
	cliRefPath := filepath.Join("..", "..", "docs", "cli-reference.md")
	cliRefContent, err := os.ReadFile(cliRefPath)
	require.NoError(t, err)

	cliRef := string(cliRefContent)
	assert.Contains(t, cliRef, "## Current partial-support gaps")
	assert.Contains(t, cliRef, "`havn config show` currently publishes source provenance")

	doltGuidePath := filepath.Join("..", "..", "docs", "dolt-beads-guide.md")
	doltGuideContent, err := os.ReadFile(doltGuidePath)
	require.NoError(t, err)

	doltGuide := string(doltGuideContent)
	assert.Contains(t, doltGuide, "## Current partial-support gaps")
	assert.Contains(t, doltGuide, "project-identity verification during import currently reports mismatch as warnings")

	configGuidePath := filepath.Join("..", "..", "docs", "configuration-guide.md")
	configGuideContent, err := os.ReadFile(configGuidePath)
	require.NoError(t, err)

	configGuide := string(configGuideContent)
	assert.Contains(t, configGuide, "## Current partial-support gaps")
	assert.Contains(t, configGuide, "`havn config show` does not yet expose every provenance detail")

	doctorGuidePath := filepath.Join("..", "..", "docs", "doctor-troubleshooting.md")
	doctorGuideContent, err := os.ReadFile(doctorGuidePath)
	require.NoError(t, err)

	doctorGuide := string(doctorGuideContent)
	assert.NotContains(t, doctorGuide, "## Current partial-support gaps")
}

func TestSpecs_CodeStandardsDocumentsSharedCLIOrchestrationBoundary(t *testing.T) {
	codeStandardsPath := filepath.Join("..", "..", "specs", "code-standards.md")
	codeStandardsContent, err := os.ReadFile(codeStandardsPath)
	require.NoError(t, err)

	codeStandards := string(codeStandardsContent)
	assert.Contains(t, codeStandards, "### Shared CLI orchestration boundary")
	assert.Contains(t, codeStandards, "`projectContext` and `effectiveConfigOrchestrator`")
	assert.Contains(t, codeStandards, "`internal/cli` command handlers stay thin")
	assert.Contains(t, codeStandards, "must not duplicate project-path and effective-config resolution")
}

func TestSpecs_SpecGovernanceHasSingleCanonicalSource(t *testing.T) {
	readmePath := filepath.Join("..", "..", "specs", "README.md")
	readmeContent, err := os.ReadFile(readmePath)
	require.NoError(t, err)

	readme := string(readmeContent)
	assert.Contains(t, readme, "## Spec Governance")
	assert.Contains(t, readme, "### Authority levels")
	assert.Contains(t, readme, "### Support-status labels")
	assert.Contains(t, readme, "## Cross-Spec Invariants")

	agentsPath := filepath.Join("..", "..", "specs", "AGENTS.md")
	agentsContent, err := os.ReadFile(agentsPath)
	require.NoError(t, err)

	agents := string(agentsContent)
	assert.Contains(t, agents, "The canonical governance source is `specs/README.md`")
	assert.NotContains(t, agents, "## Spec Governance")
	assert.NotContains(t, agents, "## Shared Vocabulary")
	assert.NotContains(t, agents, "## Cross-Spec Invariants")
}

func TestQualityGates_FormattingContractValidatesCommittedTreeConsistently(t *testing.T) {
	makefilePath := filepath.Join("..", "..", "Makefile")
	makefileContent, err := os.ReadFile(makefilePath)
	require.NoError(t, err)

	makefile := string(makefileContent)
	assert.Contains(t, makefile, "fmt-check:")
	assert.Contains(t, makefile, "check: fmt-check lint test build")

	hookPath := filepath.Join("..", "..", "lefthook.yml")
	hookContent, err := os.ReadFile(hookPath)
	require.NoError(t, err)

	hook := string(hookContent)
	assert.Contains(t, hook, "run: make fmt-check")
	assert.NotContains(t, hook, "gofmt -l . | grep .")

	qualityGatesPath := filepath.Join("..", "..", "specs", "quality-gates.md")
	qualityGatesContent, err := os.ReadFile(qualityGatesPath)
	require.NoError(t, err)

	qualityGates := string(qualityGatesContent)
	assert.Contains(t, qualityGates, "`make fmt-check`")
	assert.Contains(t, qualityGates, "Validate formatting without rewriting files")
}

func TestSpecs_BuildAndTestMergeGateRequirementsMatchEnforcement(t *testing.T) {
	testStandardsPath := filepath.Join("..", "..", "specs", "test-standards.md")
	testStandardsContent, err := os.ReadFile(testStandardsPath)
	require.NoError(t, err)

	testStandards := string(testStandardsContent)
	assert.Contains(t, testStandards, "`make build` (`go build -o bin/havn ./cmd/havn`)")

	qualityGatesPath := filepath.Join("..", "..", "specs", "quality-gates.md")
	qualityGatesContent, err := os.ReadFile(qualityGatesPath)
	require.NoError(t, err)

	qualityGates := string(qualityGatesContent)
	assert.Contains(t, qualityGates, "`integration-tests` and `boundary-confidence` are required merge checks")
}

func TestSpecs_SharedDoltStatusContractNarrowlyExcludesRuntimePort(t *testing.T) {
	sharedDoltPath := filepath.Join("..", "..", "specs", "shared-dolt-server.md")
	sharedDoltContent, err := os.ReadFile(sharedDoltPath)
	require.NoError(t, err)

	sharedDolt := string(sharedDoltContent)
	assert.Contains(t, sharedDolt, "`havn dolt status` currently does not report a runtime port")
	assert.NotContains(t, sharedDolt, "\"port\": 3308")
}

func TestDocs_RequiredMergeChecksListIncludesCoreAndDedicatedSuites(t *testing.T) {
	qualityGatesPath := filepath.Join("..", "..", "specs", "quality-gates.md")
	qualityGatesContent, err := os.ReadFile(qualityGatesPath)
	require.NoError(t, err)

	qualityGates := string(qualityGatesContent)
	assert.Contains(t, qualityGates, "`quality-gates`, `integration-tests`, and `boundary-confidence` are required merge checks")

	contributingPath := filepath.Join("..", "..", "CONTRIBUTING.md")
	contributingContent, err := os.ReadFile(contributingPath)
	require.NoError(t, err)

	contributing := string(contributingContent)
	assert.Contains(t, contributing, "`quality-gates`, `integration-tests`, and `boundary-confidence` are required merge checks on `main`")
}
