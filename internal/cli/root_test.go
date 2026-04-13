package cli_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/docker"
)

type fakeStartService struct {
	called      bool
	lastCfg     config.Config
	lastProject string
	exitCode    int
	err         error
}

func (f *fakeStartService) StartOrAttach(_ context.Context, cfg config.Config, projectPath string, _ func(string)) (int, error) {
	f.called = true
	f.lastCfg = cfg
	f.lastProject = projectPath
	return f.exitCode, f.err
}

func TestNewRoot_ReturnsCommand(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	require.NotNil(t, root)
	assert.Equal(t, "havn [flags] [path]", root.Use)
}

func TestNewRoot_SilencesCobraOutput(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	assert.True(t, root.SilenceErrors)
	assert.True(t, root.SilenceUsage)
}

func TestNewRoot_PersistentFlags(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	jsonFlag := root.PersistentFlags().Lookup("json")
	require.NotNil(t, jsonFlag, "--json persistent flag should exist")
	assert.Equal(t, "false", jsonFlag.DefValue)

	verboseFlag := root.PersistentFlags().Lookup("verbose")
	require.NotNil(t, verboseFlag, "--verbose persistent flag should exist")
	assert.Equal(t, "false", verboseFlag.DefValue)

	configFlag := root.PersistentFlags().Lookup("config")
	require.NotNil(t, configFlag, "--config persistent flag should exist")
	assert.Equal(t, "", configFlag.DefValue)
}

func TestNewRoot_LocalContainerFlags(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	flags := map[string]string{
		"shell":  "",
		"env":    "",
		"memory": "",
		"port":   "",
		"image":  "",
	}
	for name, defVal := range flags {
		f := root.Flags().Lookup(name)
		require.NotNil(t, f, "--%s local flag should exist", name)
		assert.Equal(t, defVal, f.DefValue, "--%s default", name)
	}

	cpusFlag := root.Flags().Lookup("cpus")
	require.NotNil(t, cpusFlag, "--cpus local flag should exist")
	assert.Equal(t, "0", cpusFlag.DefValue)

	noDoltFlag := root.Flags().Lookup("no-dolt")
	require.NotNil(t, noDoltFlag, "--no-dolt local flag should exist")
	assert.Equal(t, "false", noDoltFlag.DefValue)
}

func TestNewRoot_ContainerFlags_NotPersistent(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	localOnly := []string{"shell", "env", "cpus", "memory", "port", "no-dolt", "image"}
	for _, name := range localOnly {
		f := root.PersistentFlags().Lookup(name)
		assert.Nil(t, f, "--%s should not be a persistent flag", name)
	}
}

func TestNewRoot_PathArgDefaultsToDot(t *testing.T) {
	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{})

	err := root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.Contains(t, root.Use, "[path]")
}

func TestNewRoot_AcceptsExplicitPath(t *testing.T) {
	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	root.SetArgs([]string{homeDir})

	err = root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.called)
}

func TestNewRoot_RejectsTooManyArgs(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	root.SetArgs([]string{"/path/one", "/path/two"})

	err := root.Execute()

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestNewRoot_HelpIncludesAllFlags(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetArgs([]string{"--help"})

	err := root.Execute()
	require.NoError(t, err)

	help := out.String()
	flags := []string{
		"--json", "--verbose", "--config",
		"--shell", "--env", "--cpus", "--memory", "--port", "--no-dolt", "--image",
	}
	for _, f := range flags {
		assert.Contains(t, help, f, "help output should include %s", f)
	}
}

func TestDeps_AcceptsDockerClient(t *testing.T) {
	c, err := docker.NewClient()
	require.NoError(t, err)

	deps := cli.Deps{Docker: c}
	root := cli.NewRoot(deps)

	assert.NotNil(t, root)
}

func TestNewRoot_HasVersion(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	assert.NotEmpty(t, root.Version, "root command should have a version set")
}

func TestNewRoot_RunE_InvokesStartService(t *testing.T) {
	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{"."})

	err := root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.called)
}

func TestNewRoot_RunE_DefaultsDoltDatabaseToProjectNameWhenEnabled(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte("[dolt]\nenabled = true\n"), 0o644))

	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{projectPath})

	err := root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.True(t, svc.lastCfg.Dolt.Enabled)
	assert.Equal(t, "sample-project", svc.lastCfg.Dolt.Database)
}

func TestNewRoot_RunE_ReturnsNotImplementedWithoutStartService(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	root.SetArgs([]string{"."})

	err := root.Execute()

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
}

func TestNewRoot_RunE_PropagatesShellExitCode(t *testing.T) {
	svc := &fakeStartService{exitCode: 42}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{"."})

	err := root.Execute()

	require.Error(t, err)
	var shellExit *cli.ShellExitError
	require.ErrorAs(t, err, &shellExit)
	assert.Equal(t, 42, shellExit.Code)
}

func TestNewRoot_RejectsPathOutsideHome(t *testing.T) {
	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{"/tmp"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "under your home directory")
	assert.False(t, svc.called)
}

func TestNewRoot_PersistentPreRun_PropagatesLoggerToDockerClient(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	dockerClient, err := docker.NewClientWithHost("tcp://localhost:0")
	require.NoError(t, err)

	root := cli.NewRoot(cli.Deps{Docker: dockerClient, Logger: logger})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"doctor", "--all"})

	_ = root.Execute()

	assert.Contains(t, logBuf.String(), `"component":"docker"`)
	assert.Contains(t, logBuf.String(), `"operation":"ping"`)
}
