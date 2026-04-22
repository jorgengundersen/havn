package cli_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/container"
)

type fakeStopBackend struct {
	containers []container.RawContainer
	stopErrMap map[string]error
	stopCalls  []string
}

func (f *fakeStopBackend) ContainerList(_ context.Context, _ map[string]string) ([]container.RawContainer, error) {
	return f.containers, nil
}

func (f *fakeStopBackend) ContainerStop(_ context.Context, name string, _ time.Duration) error {
	f.stopCalls = append(f.stopCalls, name)
	if f.stopErrMap != nil {
		if err, ok := f.stopErrMap[name]; ok {
			return err
		}
	}
	return nil
}

func TestStopCommand_RequiresNameOrAll(t *testing.T) {
	backend := &fakeStopBackend{}

	_, _, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "stop")

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "requires")
}

func TestStopCommand_HasAllFlag(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	stopCmd, _, err := root.Find([]string{"stop"})

	require.NoError(t, err)
	f := stopCmd.Flags().Lookup("all")
	require.NotNil(t, f, "--all flag should exist")
	assert.Equal(t, "false", f.DefValue)
}

func TestStopCommand_RejectsTooManyArgs(t *testing.T) {
	_, _, err := executeCommand("stop", "one", "two")

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestStopCommand_StopsNamedContainer(t *testing.T) {
	backend := &fakeStopBackend{}

	_, _, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "stop", "havn-user-api")

	require.NoError(t, err)
	assert.Equal(t, []string{"havn-user-api"}, backend.stopCalls)
}

func TestStopCommand_LiteralTargetReportsContainerName(t *testing.T) {
	backend := &fakeStopBackend{}

	_, stderr, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "stop", "havn-user-api")

	require.NoError(t, err)
	assert.Contains(t, stderr, "Stopped havn-user-api")
}

func TestStopCommand_DotPathStopsResolvedProjectContainer(t *testing.T) {
	backend := &fakeStopBackend{}

	workspace := t.TempDir()
	projectPath := filepath.Join(workspace, "user", "api")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))
	t.Chdir(projectPath)

	_, _, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "stop", ".")

	require.NoError(t, err)
	assert.Equal(t, []string{"havn-user-api"}, backend.stopCalls)
}

func TestStopCommand_DotPathReportsResolvedContainerName(t *testing.T) {
	backend := &fakeStopBackend{}

	workspace := t.TempDir()
	projectPath := filepath.Join(workspace, "user", "api")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))
	t.Chdir(projectPath)

	_, stderr, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "stop", ".")

	require.NoError(t, err)
	assert.Contains(t, stderr, "Stopped havn-user-api")
	assert.NotContains(t, stderr, "Stopped .")
}

func TestStopCommand_JSONOutputIncludesResolvedLiteralContainer(t *testing.T) {
	backend := &fakeStopBackend{}

	stdout, _, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "--json", "stop", "havn-user-api")

	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok","message":"container stopped","container":"havn-user-api"}`+"\n", stdout)
}

func TestStopCommand_JSONOutputIncludesResolvedPathContainer(t *testing.T) {
	backend := &fakeStopBackend{}

	workspace := t.TempDir()
	projectPath := filepath.Join(workspace, "user", "api")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))
	t.Chdir(projectPath)

	stdout, _, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "--json", "stop", ".")

	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok","message":"container stopped","container":"havn-user-api"}`+"\n", stdout)
}

func TestStopCommand_StopAllBestEffort(t *testing.T) {
	backend := &fakeStopBackend{
		containers: []container.RawContainer{
			{Name: "havn-user-api", Status: "running", Labels: map[string]string{"managed-by": "havn", container.LabelPath: "/home/user/api"}},
			{Name: "havn-user-web", Status: "running", Labels: map[string]string{"managed-by": "havn", container.LabelPath: "/home/user/web"}},
			{Name: "havn-dolt", Labels: map[string]string{"managed-by": "havn"}},
		},
		stopErrMap: map[string]error{"havn-user-web": errors.New("boom")},
	}

	_, stderr, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "stop", "--all")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 stopped, 1 failed")
	assert.ElementsMatch(t, []string{"havn-user-api", "havn-user-web"}, backend.stopCalls)
	assert.True(t, strings.Contains(stderr, "Stopped havn-user-api"))
	assert.True(t, strings.Contains(stderr, "Failed to stop havn-user-web: boom"))
}

func TestStopCommand_StopAllJSONOutput(t *testing.T) {
	backend := &fakeStopBackend{
		containers: []container.RawContainer{{Name: "havn-user-api", Status: "running", Labels: map[string]string{"managed-by": "havn", container.LabelPath: "/home/user/api"}}},
	}

	stdout, _, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "--json", "stop", "--all")

	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok","message":"1 stopped, 0 failed"}`+"\n", stdout)
}

func TestStopCommand_StopAllJSONPartialFailureReturnsError(t *testing.T) {
	backend := &fakeStopBackend{
		containers: []container.RawContainer{
			{Name: "havn-user-api", Status: "running", Labels: map[string]string{"managed-by": "havn", container.LabelPath: "/home/user/api"}},
			{Name: "havn-user-web", Status: "running", Labels: map[string]string{"managed-by": "havn", container.LabelPath: "/home/user/web"}},
		},
		stopErrMap: map[string]error{"havn-user-web": errors.New("boom")},
	}

	stdout, _, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "--json", "stop", "--all")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 stopped, 1 failed")
	assert.JSONEq(t, `{"status":"error","message":"1 stopped, 1 failed"}`+"\n", stdout)
}
