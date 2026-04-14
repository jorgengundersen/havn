package ci_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type ciWorkflow struct {
	On   map[string]any         `yaml:"on"`
	Jobs map[string]workflowJob `yaml:"jobs"`
}

type workflowJob struct {
	Needs string         `yaml:"needs"`
	Steps []workflowStep `yaml:"steps"`
}

type workflowStep struct {
	Run string `yaml:"run"`
}

func (j workflowJob) runCommands() []string {
	commands := make([]string, 0, len(j.Steps))
	for _, step := range j.Steps {
		if step.Run == "" {
			continue
		}
		commands = append(commands, step.Run)
	}

	return commands
}

func (w ciWorkflow) requiredJob(t *testing.T, name string) workflowJob {
	t.Helper()
	require.Contains(t, w.Jobs, name)
	return w.Jobs[name]
}

func requireCIWorkflow(t *testing.T) ciWorkflow {
	t.Helper()
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "ci.yml")

	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)

	var workflow ciWorkflow
	err = yaml.Unmarshal(content, &workflow)
	require.NoError(t, err)

	return workflow
}
