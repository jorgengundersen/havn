package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

type suite struct {
	name    string
	goTags  string
	pkg     string
	testIDs []string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "boundary-confidence: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	suites := []suite{
		{
			name: "cli-boundary-confidence",
			pkg:  "./internal/cli",
			testIDs: []string{
				"TestHAVNBinary_CLIContractAtProcessBoundary",
				"TestDoctorCommand_AllPassExitZero",
				"TestDoctorCommand_JSONOutput",
				"TestDoctorCommand_HasAllFlag",
				"TestDoctorCommand_HasDoltFlag",
				"TestDoctorCommand_DoltFlagHelpTextExplainsAllScopeInteraction",
				"TestDoctorCommand_ExitCode2OnError",
				"TestDoctorCommand_UsesConfigFlagForGlobalConfigCheck",
				"TestDoctorCommand_DoesNotRunDefaultDerivedChecksWhenEffectiveConfigResolutionFails",
				"TestDoctorCommand_AllFlagRunsContainerChecks",
				"TestDoctorCommand_AllFlagSkipsNonProjectContainers",
				"TestDoctorCommand_AllFlagUsesPerContainerProjectPath",
				"TestDoctorCommand_AllFlagReportsTargetConfigResolutionFailure",
				"TestDoctorCommand_NoContainersSkipsTier2",
				"TestDoctorCommand_NoContainersReportsInformationalSkipInJSON",
				"TestDoctorCommand_UsesResolvedConfigMountTargets",
				"TestDoctorCommand_DefaultScopeTargetsCurrentProjectContainer",
				"TestDoctorCommand_DerivesProjectDatabaseNameFromProjectPath",
				"TestDoctorCommand_DoltFlagRunsDoltChecksWhenConfigDisabled",
				"TestDoctorCommand_DoltFlagRunsContainerDoltConnectivityWhenConfigDisabled",
				"TestDoctorCommand_DoltFlagReportsMissingDoltImage",
				"TestDoctorCommand_DoltFlagReportsActionableSkipRecommendationsInJSON",
				"TestDoctorCommand_JSONWarnExitCodeAndStreamSeparation",
			},
		},
		{
			name:    "dolt-integration-boundary-confidence",
			goTags:  "integration",
			pkg:     "./internal/dolt",
			testIDs: []string{"TestSharedDoltLifecycleAndReadiness_Integration"},
		},
	}

	for _, selectedSuite := range suites {
		if err := selectedSuite.verify(); err != nil {
			return err
		}
		if err := selectedSuite.execute(); err != nil {
			return err
		}
	}

	return nil
}

func (s suite) verify() error {
	available, err := s.listPackageTests()
	if err != nil {
		return fmt.Errorf("verify suite %q: %w", s.name, err)
	}

	missing := missingTests(available, s.testIDs)
	if len(missing) > 0 {
		return fmt.Errorf("suite %q has missing test membership: %s", s.name, strings.Join(missing, ", "))
	}

	return nil
}

func (s suite) execute() error {
	fmt.Fprintf(os.Stderr, "running suite %s\n", s.name)

	args := []string{"test"}
	if s.goTags != "" {
		args = append(args, "-tags", s.goTags)
	}
	args = append(args, s.pkg, "-run", runPatternForTests(s.testIDs))

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run suite %q: %w", s.name, err)
	}

	return nil
}

func (s suite) listPackageTests() (map[string]struct{}, error) {
	args := []string{"test"}
	if s.goTags != "" {
		args = append(args, "-tags", s.goTags)
	}
	args = append(args, s.pkg, "-list", "^Test")

	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(string(output))
	}

	available := map[string]struct{}{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Test") {
			available[line] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return available, nil
}

func runPatternForTests(testIDs []string) string {
	parts := make([]string, 0, len(testIDs))
	for _, testID := range testIDs {
		parts = append(parts, regexp.QuoteMeta(testID))
	}

	if len(parts) == 1 {
		return "^" + parts[0] + "$"
	}

	return "^(" + strings.Join(parts, "|") + ")$"
}

func missingTests(available map[string]struct{}, selected []string) []string {
	missing := make([]string, 0)
	for _, testName := range selected {
		if _, ok := available[testName]; ok {
			continue
		}
		missing = append(missing, testName)
	}

	sort.Strings(missing)
	return missing
}
