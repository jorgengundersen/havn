package cli_test

import (
	"context"
	"errors"
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

func TestStopCommand_StopAllBestEffort(t *testing.T) {
	backend := &fakeStopBackend{
		containers: []container.RawContainer{
			{Name: "havn-user-api", Labels: map[string]string{"managed-by": "havn"}},
			{Name: "havn-user-web", Labels: map[string]string{"managed-by": "havn"}},
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
		containers: []container.RawContainer{{Name: "havn-user-api", Labels: map[string]string{"managed-by": "havn"}}},
	}

	stdout, _, err := executeCommandWithDeps(cli.Deps{ContainerStop: backend}, "--json", "stop", "--all")

	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok","message":"1 stopped, 0 failed"}`+"\n", stdout)
}
