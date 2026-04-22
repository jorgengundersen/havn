package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/name"
)

func TestUpCommand_IsRegistered(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	upCmd, _, err := root.Find([]string{"up"})

	require.NoError(t, err)
	assert.Equal(t, "up [path]", upCmd.Use)
}

func TestUpCommand_ReturnsNotImplementedWithoutStartService(t *testing.T) {
	_, _, err := executeCommand("up")

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, err.Error(), "havn up:")
}

func TestUpCommand_CallsStartServiceInNoAttachMode(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeStartService{}
	_, _, err := executeCommandWithDeps(cli.Deps{StartService: svc}, "up", projectPath)

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.Equal(t, projectPath, svc.lastProject)
	assert.Equal(t, container.StartupModeNoAttach, svc.lastOpts.Mode)
	assert.Equal(t, container.StartupCheckDefault, svc.lastOpts.StartupChecks)
}

func TestUpCommand_ValidateFlagRequestsValidationChecks(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeStartService{}
	_, _, err := executeCommandWithDeps(cli.Deps{StartService: svc}, "up", "--validate", projectPath)

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.Equal(t, container.StartupModeNoAttach, svc.lastOpts.Mode)
	assert.Equal(t, container.StartupCheckValidate, svc.lastOpts.StartupChecks)
}

func TestUpCommand_PrepareFlagRequestsPrepareChecks(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeStartService{}
	_, _, err := executeCommandWithDeps(cli.Deps{StartService: svc}, "up", "--prepare", projectPath)

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.Equal(t, container.StartupModeNoAttach, svc.lastOpts.Mode)
	assert.Equal(t, container.StartupCheckPrepare, svc.lastOpts.StartupChecks)
}

func TestUpCommand_PrepareFlagWinsWhenValidateAndPrepareAreBothSet(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeStartService{}
	_, _, err := executeCommandWithDeps(cli.Deps{StartService: svc}, "up", "--validate", "--prepare", projectPath)

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.Equal(t, container.StartupModeNoAttach, svc.lastOpts.Mode)
	assert.Equal(t, container.StartupCheckPrepare, svc.lastOpts.StartupChecks)
}

func TestUpCommand_PrepareFlagHelpMentionsValidateImplication(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	upCmd, _, err := root.Find([]string{"up"})

	require.NoError(t, err)
	prepareFlag := upCmd.Flags().Lookup("prepare")
	require.NotNil(t, prepareFlag)
	assert.Contains(t, prepareFlag.Usage, "implies --validate")
}

func TestUpCommand_HelpDescribesLifecycleDefaultAndStartupCheckModes(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	upCmd, _, err := root.Find([]string{"up"})

	require.NoError(t, err)
	assert.Contains(t, upCmd.Long, "lifecycle-only")
	assert.Contains(t, upCmd.Long, "--validate")
	assert.Contains(t, upCmd.Long, "--prepare")
}

func TestUpCommand_DoesNotAcceptShellFlag(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeStartService{}
	_, _, err := executeCommandWithDeps(cli.Deps{StartService: svc}, "up", "--shell", "zsh", projectPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag: --shell")
	assert.False(t, svc.called)
}

func TestUpCommand_AppliesSupportedRuntimeOverrides(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte("env = \"github:project/env\"\nimage = \"havn-project:latest\"\nports = [\"2022:22\"]\n\n[resources]\ncpus = 2\nmemory = \"2g\"\n\n[dolt]\nenabled = true\n"), 0o644))

	svc := &fakeStartService{}
	_, _, err := executeCommandWithDeps(
		cli.Deps{StartService: svc},
		"up",
		"--env", "github:flag/env",
		"--cpus", "6",
		"--memory", "12g",
		"--port", "2244",
		"--image", "havn-flag:latest",
		"--no-dolt",
		projectPath,
	)

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.Equal(t, "github:flag/env", svc.lastCfg.Env)
	assert.Equal(t, 6, svc.lastCfg.Resources.CPUs)
	assert.Equal(t, "12g", svc.lastCfg.Resources.Memory)
	assert.Equal(t, "havn-flag:latest", svc.lastCfg.Image)
	assert.Equal(t, []string{"2022:22", "2244:22"}, svc.lastCfg.Ports)
	assert.False(t, svc.lastCfg.Dolt.Enabled)
}

func TestUpCommand_VerboseFlagEnablesVerboseStartupMode(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeStartService{}
	_, _, err := executeCommandWithDeps(cli.Deps{StartService: svc}, "--verbose", "up", projectPath)

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.True(t, svc.lastOpts.VerboseStartup)
	assert.Equal(t, container.StartupModeNoAttach, svc.lastOpts.Mode)
}

func TestUpCommand_PrintsContainerRunningConfirmationOnSuccess(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeStartService{}
	_, stderr, err := executeCommandWithDeps(cli.Deps{StartService: svc}, "up", projectPath)

	require.NoError(t, err)
	containerName, nameErr := name.DeriveContainerName("work", "sample-project")
	require.NoError(t, nameErr)
	assert.Contains(t, stderr, "Container "+string(containerName)+" is running for project "+projectPath)
}

func TestUpCommand_HomeManagerActivationFailureReturnsCommandScopedErrorWithoutSuccessStatus(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeStartService{err: assert.AnError}
	_, stderr, err := executeCommandWithDeps(cli.Deps{StartService: svc}, "up", projectPath)

	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	assert.Contains(t, err.Error(), "havn up:")
	assert.True(t, svc.called)
	assert.Equal(t, container.StartupModeNoAttach, svc.lastOpts.Mode)
	assert.NotContains(t, stderr, "is running for project")
}
