package cli_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/dolt"
)

type fakeDoltBackend struct {
	inspectInfo     dolt.ContainerInfo
	inspectFound    bool
	inspectErr      error
	stopErr         error
	execOutput      string
	execErr         error
	interactiveErr  error
	lastExec        []string
	lastInteractive []string
	lastStoppedName string
}

func (f *fakeDoltBackend) ContainerCreate(_ context.Context, _ dolt.ContainerCreateOpts) (string, error) {
	return "", nil
}

func (f *fakeDoltBackend) ContainerStart(_ context.Context, _ string) error {
	return nil
}

func (f *fakeDoltBackend) ContainerStop(_ context.Context, name string) error {
	f.lastStoppedName = name
	return f.stopErr
}

func (f *fakeDoltBackend) ContainerInspect(_ context.Context, _ string) (dolt.ContainerInfo, bool, error) {
	return f.inspectInfo, f.inspectFound, f.inspectErr
}

func (f *fakeDoltBackend) ContainerExec(_ context.Context, _ string, cmd []string) (string, error) {
	f.lastExec = cmd
	if f.execErr != nil {
		return "", f.execErr
	}
	return f.execOutput, nil
}

func (f *fakeDoltBackend) ContainerExecInteractive(_ context.Context, _ string, cmd []string) error {
	f.lastInteractive = cmd
	return f.interactiveErr
}

func (f *fakeDoltBackend) CopyToContainer(_ context.Context, _ string, _ string, _ []byte) error {
	return nil
}

func (f *fakeDoltBackend) CopyFromContainer(_ context.Context, _ string, _ string) ([]byte, error) {
	return nil, nil
}

func executeDoltWithRoot(root *cobra.Command, args ...string) (stdout, stderr string, err error) {
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)
	root.SetArgs(args)
	err = root.Execute()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func TestDoltCommand_PrintsHelp(t *testing.T) {
	_, _, err := executeCommand("dolt")

	require.NoError(t, err)
}

func TestDoltStartCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "start")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt start:")
}

func TestDoltStartCommand_StartsSharedDoltServer(t *testing.T) {
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "running-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "dolt", "start")

	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "Starting shared Dolt server")
}

func TestDoltStopCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "stop")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt stop:")
}

func TestDoltStopCommand_StopsSharedDoltServer(t *testing.T) {
	backend := &fakeDoltBackend{}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "dolt", "stop")

	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Equal(t, "havn-dolt", backend.lastStoppedName)
	assert.Contains(t, stderr, "Stopping shared Dolt server")
}

func TestDoltStatusCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "status")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt status:")
}

func TestDoltStatusCommand_PrintsJSONStatus(t *testing.T) {
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			Running: true,
			Image:   "dolthub/dolt-sql-server:latest",
			Network: "havn-net",
			Port:    3308,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, _, err := executeDoltWithRoot(root, "--json", "dolt", "status")

	require.NoError(t, err)
	assert.JSONEq(t, `{"running":true,"container":"havn-dolt","image":"dolthub/dolt-sql-server:latest","port":3308,"network":"havn-net","managed_by_havn":true}`+"\n", stdout)
}

func TestDoltDatabasesCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "databases")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt databases:")
}

func TestDoltDatabasesCommand_PrintsJSONDatabaseList(t *testing.T) {
	backend := &fakeDoltBackend{
		execOutput: "+--------------------+\n| Database           |\n+--------------------+\n| api                |\n| web                |\n+--------------------+\n",
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, _, err := executeDoltWithRoot(root, "--json", "dolt", "databases")

	require.NoError(t, err)
	assert.JSONEq(t, `["api","web"]`+"\n", stdout)
}

func TestDoltDropCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "drop", "mydb")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt drop:")
}

func TestDoltDropCommand_RequiresName(t *testing.T) {
	_, _, err := executeCommand("dolt", "drop")

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestDoltDropCommand_HasYesFlag(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	dropCmd, _, err := root.Find([]string{"dolt", "drop"})

	require.NoError(t, err)
	f := dropCmd.Flags().Lookup("yes")
	require.NotNil(t, f, "--yes flag should exist")
	assert.Equal(t, "false", f.DefValue)
}

func TestDoltDropCommand_RequiresYesFlag(t *testing.T) {
	backend := &fakeDoltBackend{}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, _, err := executeDoltWithRoot(root, "dolt", "drop", "mydb")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
}

func TestDoltDropCommand_DropsDatabaseWhenConfirmed(t *testing.T) {
	backend := &fakeDoltBackend{}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "dolt", "drop", "mydb", "--yes")

	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Equal(t, []string{"dolt", "sql", "-q", "DROP DATABASE `mydb`"}, backend.lastExec)
	assert.Contains(t, stderr, "Dropping database mydb")
}

func TestDoltConnectCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "connect")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt connect:")
}

func TestDoltConnectCommand_OpensInteractiveShell(t *testing.T) {
	backend := &fakeDoltBackend{}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "dolt", "connect")

	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Equal(t, []string{"dolt", "sql"}, backend.lastInteractive)
	assert.Contains(t, stderr, "Connecting to shared Dolt SQL shell")
}

func TestDoltImportCommand_RequiresPath(t *testing.T) {
	_, _, err := executeCommand("dolt", "import")

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestDoltImportCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "import", "/some/path")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt import:")
}

func TestDoltExportCommand_RequiresName(t *testing.T) {
	_, _, err := executeCommand("dolt", "export")

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestDoltExportCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "export", "mydb")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt export:")
}
