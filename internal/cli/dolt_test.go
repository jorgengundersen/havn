package cli_test

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
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
	createOpts      dolt.ContainerCreateOpts
	createFunc      func(opts dolt.ContainerCreateOpts) (string, error)
	stopErr         error
	pullErr         error
	pullFunc        func(image string) error
	pullCalls       []string
	execOutput      string
	execErr         error
	execFunc        func(cmd []string) (string, error)
	copyToData      []byte
	copyToDest      string
	copyFromData    []byte
	copyFromErr     error
	interactiveErr  error
	lastExec        []string
	lastInteractive []string
	lastStoppedName string
}

func (f *fakeDoltBackend) ContainerCreate(_ context.Context, opts dolt.ContainerCreateOpts) (string, error) {
	f.createOpts = opts
	if f.createFunc != nil {
		return f.createFunc(opts)
	}
	return "created-id", nil
}

func (f *fakeDoltBackend) ImagePull(_ context.Context, image string) error {
	f.pullCalls = append(f.pullCalls, image)
	if f.pullFunc != nil {
		return f.pullFunc(image)
	}
	return f.pullErr
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
	if f.execFunc != nil {
		return f.execFunc(cmd)
	}
	if f.execErr != nil {
		return "", f.execErr
	}
	return f.execOutput, nil
}

func (f *fakeDoltBackend) ContainerExecInteractive(_ context.Context, _ string, cmd []string) error {
	f.lastInteractive = cmd
	return f.interactiveErr
}

func (f *fakeDoltBackend) CopyToContainer(_ context.Context, _ string, destPath string, data []byte) error {
	f.copyToDest = destPath
	f.copyToData = append([]byte(nil), data...)
	return nil
}

func (f *fakeDoltBackend) CopyFromContainer(_ context.Context, _ string, _ string) ([]byte, error) {
	return f.copyFromData, f.copyFromErr
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
	assert.Contains(t, stderr, "Shared Dolt server started")
}

func TestDoltStartCommand_UsesEffectiveProjectDoltConfig(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectDir := t.TempDir()
	require.NoError(t, os.MkdirAll(projectDir+"/.havn", 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.havn/config.toml", []byte("network = \"custom-net\"\n[dolt]\nport = 4400\nimage = \"dolthub/dolt-sql-server:v2\"\n"), 0o644))

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWD))
	})

	backend := &fakeDoltBackend{
		execFunc: func(_ []string) (string, error) {
			return "", nil
		},
	}

	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, _, err = executeDoltWithRoot(root, "dolt", "start")

	require.NoError(t, err)
	assert.Equal(t, "custom-net", backend.createOpts.Network)
	assert.Equal(t, "dolthub/dolt-sql-server:v2", backend.createOpts.Image)
	assert.Equal(t, "/etc/dolt/servercfg.d", backend.copyToDest)
	assert.Contains(t, string(backend.copyToData), "port: 4400")
}

func TestDoltStartCommand_MissingImageReportsAcquisitionProgress(t *testing.T) {
	createCalls := 0
	backend := &fakeDoltBackend{
		createFunc: func(opts dolt.ContainerCreateOpts) (string, error) {
			createCalls++
			if createCalls == 1 {
				return "", &dolt.ImageNotFoundError{Image: opts.Image}
			}
			return "created-id", nil
		},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, stderr, err := executeDoltWithRoot(root, "dolt", "start")

	require.NoError(t, err)
	assert.Contains(t, stderr, "Starting shared Dolt server")
	assert.Contains(t, stderr, "acquiring image")
	assert.Contains(t, stderr, "resuming shared Dolt startup")
	assert.Contains(t, stderr, "completed after image acquisition")
}

func TestDoltStartCommand_MissingImageJSONOmitsAcquisitionProgress(t *testing.T) {
	createCalls := 0
	backend := &fakeDoltBackend{
		createFunc: func(opts dolt.ContainerCreateOpts) (string, error) {
			createCalls++
			if createCalls == 1 {
				return "", &dolt.ImageNotFoundError{Image: opts.Image}
			}
			return "created-id", nil
		},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "--json", "dolt", "start")

	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok","message":"dolt server started"}`+"\n", stdout)
	assert.Contains(t, stderr, "Starting shared Dolt server")
	assert.NotContains(t, stderr, "acquiring image")
	assert.NotContains(t, stderr, "resuming shared Dolt startup")
	assert.NotContains(t, stderr, "completed after image acquisition")
}

func TestDoltStartCommand_MissingImagePullFailureReportsContext(t *testing.T) {
	backend := &fakeDoltBackend{
		createFunc: func(opts dolt.ContainerCreateOpts) (string, error) {
			return "", &dolt.ImageNotFoundError{Image: opts.Image}
		},
		pullErr: errors.New("registry unavailable"),
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, stderr, err := executeDoltWithRoot(root, "dolt", "start")

	require.Error(t, err)
	assert.Contains(t, stderr, "acquiring image")
	assert.Contains(t, stderr, "failed during image acquisition path")
	assert.Contains(t, err.Error(), "havn dolt start:")
}

func TestDoltStopCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "stop")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt stop:")
}

func TestDoltStopCommand_StopsSharedDoltServer(t *testing.T) {
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo:  dolt.ContainerInfo{ID: "running-id", Running: true, Labels: map[string]string{"managed-by": "havn"}},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "dolt", "stop")

	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Equal(t, "havn-dolt", backend.lastStoppedName)
	assert.Contains(t, stderr, "Stopping shared Dolt server")
	assert.Contains(t, stderr, "Shared Dolt server stopped")
}

func TestDoltStopCommand_WhenServerNotRunning_ReturnsGuidance(t *testing.T) {
	backend := &fakeDoltBackend{}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, _, err := executeDoltWithRoot(root, "dolt", "stop")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "container \"havn-dolt\" is not running")
}

func TestDoltStatusCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "status")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt status:")
}

func TestDoltStatusCommand_PrintsJSONStatus(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWD))
	})

	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			Running: true,
			Image:   "dolthub/dolt-sql-server:latest",
			Network: "havn-net",
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, _, err := executeDoltWithRoot(root, "--json", "dolt", "status")

	require.NoError(t, err)
	assert.JSONEq(t, `{"running":true,"configured_sql_port":3308,"container":"havn-dolt","image":"dolthub/dolt-sql-server:latest","network":"havn-net","managed_by_havn":true}`+"\n", stdout)
}

func TestDoltStatusCommand_NotRunningPrintsJSONConfiguredPortOnly(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWD))
	})

	backend := &fakeDoltBackend{inspectFound: false}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, _, err := executeDoltWithRoot(root, "--json", "dolt", "status")

	require.NoError(t, err)
	assert.JSONEq(t, `{"running":false,"configured_sql_port":3308}`+"\n", stdout)
}

func TestDoltStatusCommand_PrintsConfiguredPortAndRuntimeGuidance(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWD))
	})

	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			Running: true,
			Image:   "dolthub/dolt-sql-server:latest",
			Network: "havn-net",
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, _, err := executeDoltWithRoot(root, "dolt", "status")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Dolt server is running (havn-dolt)")
	assert.Contains(t, stdout, "Configured SQL port: 3308")
	assert.Contains(t, stdout, "Runtime port verification is external")
}

func TestDoltStatusCommand_NotRunningPrintsConfiguredPortAndRuntimeGuidance(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWD))
	})

	backend := &fakeDoltBackend{inspectFound: false}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, _, err := executeDoltWithRoot(root, "dolt", "status")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Dolt server is not running")
	assert.Contains(t, stdout, "Configured SQL port: 3308")
	assert.Contains(t, stdout, "Runtime port verification is external")
}

func TestDoltDatabasesCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "databases")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt databases:")
}

func TestDoltDatabasesCommand_PrintsJSONDatabaseList(t *testing.T) {
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo:  dolt.ContainerInfo{ID: "running-id", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		execOutput:   "+--------------------+\n| Database           |\n+--------------------+\n| api                |\n| web                |\n+--------------------+\n",
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, _, err := executeDoltWithRoot(root, "--json", "dolt", "databases")

	require.NoError(t, err)
	assert.JSONEq(t, `["api","web"]`+"\n", stdout)
}

func TestDoltDatabasesCommand_WhenServerNotRunning_ReturnsGuidance(t *testing.T) {
	backend := &fakeDoltBackend{}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, _, err := executeDoltWithRoot(root, "dolt", "databases")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "container \"havn-dolt\" is not running")
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
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo:  dolt.ContainerInfo{ID: "running-id", Running: true, Labels: map[string]string{"managed-by": "havn"}},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "dolt", "drop", "mydb", "--yes")

	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Equal(t, []string{"dolt", "sql", "-q", "DROP DATABASE `mydb`"}, backend.lastExec)
	assert.Contains(t, stderr, "Dropping database mydb")
	assert.Contains(t, stderr, "Database mydb dropped")
}

func TestDoltConnectCommand_ReturnsNotImplemented(t *testing.T) {
	_, _, err := executeCommand("dolt", "connect")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn dolt connect:")
}

func TestDoltConnectCommand_OpensInteractiveShell(t *testing.T) {
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo:  dolt.ContainerInfo{ID: "running-id", Running: true, Labels: map[string]string{"managed-by": "havn"}},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "dolt", "connect")

	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Equal(t, []string{"dolt", "sql"}, backend.lastInteractive)
	assert.Contains(t, stderr, "Connecting to shared Dolt SQL shell")
}

func TestDoltConnectCommand_EmitsCompletionSignalAfterShellExits(t *testing.T) {
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo:  dolt.ContainerInfo{ID: "running-id", Running: true, Labels: map[string]string{"managed-by": "havn"}},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, stderr, err := executeDoltWithRoot(root, "dolt", "connect")

	require.NoError(t, err)
	assert.Contains(t, stderr, "Shared Dolt SQL shell session ended")
}

func TestDoltConnectCommand_WhenServerNotRunning_ReturnsGuidance(t *testing.T) {
	backend := &fakeDoltBackend{}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, _, err := executeDoltWithRoot(root, "dolt", "connect")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "container \"havn-dolt\" is not running")
}

func TestDoltImportCommand_RequiresPath(t *testing.T) {
	_, _, err := executeCommand("dolt", "import")

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestDoltImportCommand_ImportsDatabase(t *testing.T) {
	projectDir := t.TempDir()
	dbName := "sample"
	require.NoError(t, os.MkdirAll(projectDir+"/.havn", 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.havn/config.toml", []byte("[dolt]\ndatabase = \"sample\"\n"), 0o644))
	require.NoError(t, os.MkdirAll(projectDir+"/.beads/dolt/"+dbName, 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.beads/dolt/"+dbName+"/manifest", []byte("data"), 0o644))

	callCount := 0
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "running-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execFunc: func(cmd []string) (string, error) {
			if len(cmd) == 4 && cmd[3] == "SHOW DATABASES" {
				callCount++
				if callCount == 1 {
					return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n+--------------------+\n", nil
				}
				return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n| sample             |\n+--------------------+\n", nil
			}
			return "", nil
		},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "dolt", "import", projectDir)

	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "Importing Dolt database")
	assert.Contains(t, stderr, "Database sample imported")
}

func TestDoltImportCommand_WhenServerNotRunning_ReturnsGuidance(t *testing.T) {
	projectDir := t.TempDir()
	dbName := "sample"
	require.NoError(t, os.MkdirAll(projectDir+"/.havn", 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.havn/config.toml", []byte("[dolt]\ndatabase = \"sample\"\n"), 0o644))
	require.NoError(t, os.MkdirAll(projectDir+"/.beads/dolt/"+dbName, 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.beads/dolt/"+dbName+"/manifest", []byte("data"), 0o644))

	backend := &fakeDoltBackend{
		inspectFound: false,
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, _, err := executeDoltWithRoot(root, "dolt", "import", projectDir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "container \"havn-dolt\" is not running")
}

func TestDoltImportCommand_ForceMakesOverwriteExplicit(t *testing.T) {
	projectDir := t.TempDir()
	dbName := "sample"
	require.NoError(t, os.MkdirAll(projectDir+"/.havn", 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.havn/config.toml", []byte("[dolt]\ndatabase = \"sample\"\n"), 0o644))
	require.NoError(t, os.MkdirAll(projectDir+"/.beads/dolt/"+dbName, 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.beads/dolt/"+dbName+"/manifest", []byte("data"), 0o644))

	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "running-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execOutput: "+--------------------+\n| Database           |\n+--------------------+\n| sample             |\n+--------------------+\n",
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, stderr, err := executeDoltWithRoot(root, "dolt", "import", projectDir, "--force")

	require.NoError(t, err)
	assert.Contains(t, stderr, "Overwriting existing database sample")
}

func TestDoltImportCommand_UsesWarningSeverityForImportWarnings(t *testing.T) {
	projectDir := t.TempDir()
	dbName := "sample"
	require.NoError(t, os.MkdirAll(projectDir+"/.havn", 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.havn/config.toml", []byte("[dolt]\ndatabase = \"sample\"\n"), 0o644))
	require.NoError(t, os.MkdirAll(projectDir+"/.beads/dolt/"+dbName, 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.beads/dolt/"+dbName+"/manifest", []byte("data"), 0o644))
	require.NoError(t, os.MkdirAll(projectDir+"/.beads", 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.beads/metadata.json", []byte(`{"project_id":"local-uuid-111"}`), 0o644))

	callCount := 0
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "running-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execFunc: func(cmd []string) (string, error) {
			if len(cmd) == 4 && cmd[3] == "SHOW DATABASES" {
				callCount++
				if callCount == 1 {
					return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n+--------------------+\n", nil
				}
				return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n| sample             |\n+--------------------+\n", nil
			}

			if len(cmd) == 4 && cmd[3] == "SELECT value FROM `sample`.metadata WHERE `key` = '_project_id'" {
				return "+--------------+\n| value        |\n+--------------+\n| db-uuid-222  |\n+--------------+\n", nil
			}

			return "", nil
		},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, stderr, err := executeDoltWithRoot(root, "dolt", "import", projectDir)

	require.NoError(t, err)
	assert.Contains(t, stderr, "WARNING: project_id mismatch")
	assert.Contains(t, stderr, "WARNING: Ownership boundary [beads_migration_workflow]")
	assert.Contains(t, stderr, "run 'bd migrate --help' and see docs/dolt-beads-guide.md")
}

func TestDoltImportCommand_FailuresAreCommandScoped(t *testing.T) {
	projectDir := t.TempDir()
	dbName := "sample"
	require.NoError(t, os.MkdirAll(projectDir+"/.havn", 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.havn/config.toml", []byte("[dolt]\ndatabase = \"sample\"\n"), 0o644))
	require.NoError(t, os.MkdirAll(projectDir+"/.beads/dolt/"+dbName, 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.beads/dolt/"+dbName+"/manifest", []byte("data"), 0o644))

	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "running-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execErr: errors.New("import failed"),
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, _, err := executeDoltWithRoot(root, "dolt", "import", projectDir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "havn dolt import:")
}

func TestDoltImportCommand_JSONIncludesOwnershipBoundary(t *testing.T) {
	projectDir := t.TempDir()
	dbName := "sample"
	require.NoError(t, os.MkdirAll(projectDir+"/.havn", 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.havn/config.toml", []byte("[dolt]\ndatabase = \"sample\"\n"), 0o644))
	require.NoError(t, os.MkdirAll(projectDir+"/.beads/dolt/"+dbName, 0o755))
	require.NoError(t, os.WriteFile(projectDir+"/.beads/dolt/"+dbName+"/manifest", []byte("data"), 0o644))

	callCount := 0
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "running-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execFunc: func(cmd []string) (string, error) {
			if len(cmd) == 4 && cmd[3] == "SHOW DATABASES" {
				callCount++
				if callCount == 1 {
					return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n+--------------------+\n", nil
				}
				return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n| sample             |\n+--------------------+\n", nil
			}
			return "", nil
		},
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "--json", "dolt", "import", projectDir)

	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok","message":"database imported","database":"sample","path":"`+projectDir+`","overwrote":false,"warnings":[],"ownership_boundary":"beads_migration_workflow"}`+"\n", stdout)
	assert.Contains(t, stderr, "Ownership boundary [beads_migration_workflow]: migration semantics are owned by beads/Dolt workflows")
}

func TestDoltExportCommand_RequiresName(t *testing.T) {
	_, _, err := executeCommand("dolt", "export")

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestDoltExportCommand_ExportsDatabaseToDestination(t *testing.T) {
	destDir := t.TempDir()
	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo:  dolt.ContainerInfo{ID: "running-id", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		execOutput:   "+--------------------+\n| Database           |\n+--------------------+\n| mydb               |\n+--------------------+\n",
		copyFromData: buildTarArchive(t, "mydb", map[string]string{"manifest": "exported-data"}),
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "dolt", "export", "mydb", "--dest", destDir)

	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "Exporting Dolt database")
	assert.Contains(t, stderr, "Database mydb exported to")

	manifest, readErr := os.ReadFile(destDir + "/.beads/dolt/mydb/manifest")
	require.NoError(t, readErr)
	assert.Equal(t, "exported-data", string(manifest))
}

func TestDoltExportCommand_WhenServerNotRunning_ReturnsGuidance(t *testing.T) {
	destDir := t.TempDir()
	backend := &fakeDoltBackend{}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, _, err := executeDoltWithRoot(root, "dolt", "export", "mydb", "--dest", destDir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "container \"havn-dolt\" is not running")
}

func TestDoltExportCommand_JSONIncludesOwnershipBoundary(t *testing.T) {
	destDir := t.TempDir()
	absDestDir, err := filepath.Abs(destDir)
	require.NoError(t, err)

	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo:  dolt.ContainerInfo{ID: "running-id", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		execOutput:   "+--------------------+\n| Database           |\n+--------------------+\n| mydb               |\n+--------------------+\n",
		copyFromData: buildTarArchive(t, "mydb", map[string]string{"manifest": "exported-data"}),
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	stdout, stderr, err := executeDoltWithRoot(root, "--json", "dolt", "export", "mydb", "--dest", destDir)

	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok","message":"database exported","database":"mydb","dest":"`+absDestDir+`","ownership_boundary":"beads_migration_workflow"}`+"\n", stdout)
	assert.Contains(t, stderr, "Ownership boundary [beads_migration_workflow]: migration semantics are owned by beads/Dolt workflows")
	assert.Contains(t, stderr, "run 'bd migrate --help' and see docs/dolt-beads-guide.md")
}

func TestDoltExportCommand_FailuresAreCommandScoped(t *testing.T) {
	destDir := t.TempDir()

	backend := &fakeDoltBackend{
		inspectFound: true,
		inspectInfo:  dolt.ContainerInfo{ID: "running-id", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		execOutput:   "+--------------------+\n| Database           |\n+--------------------+\n| mydb               |\n+--------------------+\n",
		copyFromErr:  errors.New("export failed"),
	}
	root := cli.NewRoot(cli.Deps{DoltManager: dolt.NewManager(backend)})
	_, _, err := executeDoltWithRoot(root, "dolt", "export", "mydb", "--dest", destDir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "havn dolt export:")
}

func buildTarArchive(t *testing.T, prefix string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     prefix + "/",
		Mode:     0o755,
	}))

	for name, content := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     prefix + "/" + name,
			Size:     int64(len(content)),
			Mode:     0o644,
		}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	return buf.Bytes()
}
