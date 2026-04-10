package name_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/name"
)

func TestSplitProjectPath_StandardPath(t *testing.T) {
	parent, project, err := name.SplitProjectPath("/home/devuser/Repos/github.com/user/api")
	require.NoError(t, err)
	assert.Equal(t, "user", parent)
	assert.Equal(t, "api", project)
}

func TestSplitProjectPath_RelativePathReturnsError(t *testing.T) {
	_, _, err := name.SplitProjectPath("relative/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "absolute")
}

func TestSplitProjectPath_TooShallowReturnsError(t *testing.T) {
	_, _, err := name.SplitProjectPath("/too-shallow")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 segments")
}

func TestSplitProjectPath_RootReturnsError(t *testing.T) {
	_, _, err := name.SplitProjectPath("/")
	assert.Error(t, err)
}
