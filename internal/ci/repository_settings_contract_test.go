package ci_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type repositorySettings struct {
	Branches []branchSettings `yaml:"branches"`
}

type branchSettings struct {
	Name       string           `yaml:"name"`
	Protection branchProtection `yaml:"protection"`
}

type branchProtection struct {
	RequiredStatusChecks requiredStatusChecks `yaml:"required_status_checks"`
}

type requiredStatusChecks struct {
	Strict   bool     `yaml:"strict"`
	Contexts []string `yaml:"contexts"`
}

func (s repositorySettings) requiredMainBranchProtection(t *testing.T) branchSettings {
	t.Helper()

	for _, branch := range s.Branches {
		if branch.Name == "main" {
			return branch
		}
	}

	t.Fatalf("main branch settings not found")
	return branchSettings{}
}

func requireRepositorySettings(t *testing.T) repositorySettings {
	t.Helper()
	settingsPath := filepath.Join("..", "..", ".github", "settings.yml")

	content, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings repositorySettings
	err = yaml.Unmarshal(content, &settings)
	require.NoError(t, err)

	return settings
}
