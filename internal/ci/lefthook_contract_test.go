package ci_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type lefthookConfig struct {
	PreCommit lefthookPreCommit `yaml:"pre-commit"`
}

type lefthookPreCommit struct {
	Jobs []lefthookJob `yaml:"jobs"`
}

type lefthookJob struct {
	Run string `yaml:"run"`
}

func (c lefthookConfig) preCommitRuns() []string {
	runs := make([]string, 0, len(c.PreCommit.Jobs))
	for _, job := range c.PreCommit.Jobs {
		if job.Run == "" {
			continue
		}

		runs = append(runs, job.Run)
	}

	return runs
}

func requireLefthookConfig(t *testing.T) lefthookConfig {
	t.Helper()
	configPath := filepath.Join("..", "..", "lefthook.yml")

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var config lefthookConfig
	err = yaml.Unmarshal(content, &config)
	require.NoError(t, err)

	return config
}

func TestLefthookPreCommit_EnforcesContractMatrixGate(t *testing.T) {
	config := requireLefthookConfig(t)

	assert.Contains(t, config.preCommitRuns(), "make test-contract-matrix")
}
