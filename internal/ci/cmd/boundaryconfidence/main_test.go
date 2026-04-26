package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunPatternForTests_MultipleTestsUsesAnchoredAlternation(t *testing.T) {
	pattern := runPatternForTests([]string{"TestDoctorCommand_AllPassExitZero", "TestHAVNBinary_CLIContractAtProcessBoundary"})

	assert.Equal(t, "^(TestDoctorCommand_AllPassExitZero|TestHAVNBinary_CLIContractAtProcessBoundary)$", pattern)
}

func TestMissingTests_ReturnsSortedMissingMembership(t *testing.T) {
	available := map[string]struct{}{
		"TestA": {},
		"TestC": {},
	}

	missing := missingTests(available, []string{"TestC", "TestB", "TestA"})

	assert.Equal(t, []string{"TestB"}, missing)
}
