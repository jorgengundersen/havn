package cli_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/mount"
)

type fakeStartService struct {
	called      bool
	lastCfg     config.Config
	lastProject string
	lastOpts    container.StartOptions
	exitCode    int
	err         error
}

type rootBoundaryStartService struct {
	doltSetup *rootBoundaryFakeDoltSetup
	container *rootBoundaryFakeStartBackend
}

func newRootBoundaryStartService() *rootBoundaryStartService {
	return &rootBoundaryStartService{
		doltSetup: &rootBoundaryFakeDoltSetup{},
		container: &rootBoundaryFakeStartBackend{},
	}
}

func (s *rootBoundaryStartService) StartOrAttach(ctx context.Context, cfg config.Config, projectPath string, _ func(string), opts container.StartOptions) (int, error) {
	return container.StartOrAttachWithOptions(ctx, container.StartDeps{
		Container: s.container,
		Image:     rootBoundaryFakeImageBackend{},
		Network:   rootBoundaryFakeNetworkBackend{},
		Volume:    rootBoundaryFakeVolumeEnsurer{},
		Mount: rootBoundaryFakeMountResolver{result: mount.ResolveResult{
			Env: map[string]string{},
		}},
		Dolt:   s.doltSetup,
		Exec:   rootBoundaryFakeExecBackend{},
		Status: func(string) {},
	}, cfg, projectPath, opts)
}

type rootBoundaryFakeStartBackend struct {
	createdOpts container.CreateOpts
	createCalls int
}

func (b *rootBoundaryFakeStartBackend) ContainerInspect(_ context.Context, name string) (container.State, error) {
	return container.State{}, &container.NotFoundError{Name: name}
}

func (b *rootBoundaryFakeStartBackend) ContainerCreate(_ context.Context, opts container.CreateOpts) (string, error) {
	b.createCalls++
	b.createdOpts = opts
	return "created-id", nil
}

func (b *rootBoundaryFakeStartBackend) ContainerStart(_ context.Context, _ string) error {
	return nil
}

type rootBoundaryFakeImageBackend struct{}

func (rootBoundaryFakeImageBackend) ImageBuild(_ context.Context, _ container.ImageBuildOpts) error {
	return nil
}

func (rootBoundaryFakeImageBackend) ImageExists(_ context.Context, _ string) (bool, error) {
	return true, nil
}

type rootBoundaryFakeNetworkBackend struct{}

func (rootBoundaryFakeNetworkBackend) NetworkInspect(_ context.Context, _ string) error {
	return nil
}

func (rootBoundaryFakeNetworkBackend) NetworkCreate(_ context.Context, _ string) error {
	return nil
}

type rootBoundaryFakeVolumeEnsurer struct{}

func (rootBoundaryFakeVolumeEnsurer) EnsureExists(_ context.Context, _ string) error {
	return nil
}

type rootBoundaryFakeMountResolver struct {
	result mount.ResolveResult
}

func (r rootBoundaryFakeMountResolver) Resolve(_ config.Config, _ string) (mount.ResolveResult, error) {
	return r.result, nil
}

type rootBoundaryFakeDoltSetup struct {
	called  bool
	lastCfg config.Config
}

func (d *rootBoundaryFakeDoltSetup) EnsureReady(_ context.Context, cfg config.Config) (map[string]string, error) {
	d.called = true
	d.lastCfg = cfg
	return map[string]string{
		"BEADS_DOLT_SHARED_SERVER":   "1",
		"BEADS_DOLT_SERVER_HOST":     "havn-dolt",
		"BEADS_DOLT_SERVER_PORT":     strconv.Itoa(cfg.Dolt.Port),
		"BEADS_DOLT_SERVER_USER":     "root",
		"BEADS_DOLT_SERVER_DATABASE": cfg.Dolt.Database,
		"BEADS_DOLT_AUTO_START":      "0",
	}, nil
}

func (d *rootBoundaryFakeDoltSetup) MigrationNotice(context.Context, config.Config, string) (string, error) {
	return "", nil
}

type rootBoundaryFakeExecBackend struct{}

func (rootBoundaryFakeExecBackend) ContainerExec(_ context.Context, _ string, _ []string) error {
	return nil
}

func (rootBoundaryFakeExecBackend) ContainerExecInteractive(_ context.Context, _ string, _ []string, _ string) (int, error) {
	return 0, nil
}

func (f *fakeStartService) StartOrAttach(_ context.Context, cfg config.Config, projectPath string, _ func(string), opts container.StartOptions) (int, error) {
	f.called = true
	f.lastCfg = cfg
	f.lastProject = projectPath
	f.lastOpts = opts
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

func TestNewRoot_RunE_VerboseFlagEnablesVerboseStartupMode(t *testing.T) {
	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{"--verbose", "."})

	err := root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.True(t, svc.lastOpts.VerboseStartup)
}

func TestNewRoot_RunE_UsesAttachStartupMode(t *testing.T) {
	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{"."})

	err := root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.Equal(t, container.StartupModeAttach, svc.lastOpts.Mode)
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

func TestNewRoot_RunE_NewContainerUsesEffectiveDefaultResourceLimits(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := newRootBoundaryStartService()
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{projectPath})

	err := root.Execute()

	require.NoError(t, err)
	assert.Equal(t, 1, svc.container.createCalls)
	assert.Equal(t, 4, svc.container.createdOpts.CPUs)
	assert.Equal(t, "8g", svc.container.createdOpts.Memory)
	assert.Equal(t, "12g", svc.container.createdOpts.MemorySwap)
	assert.Equal(t, "4", svc.container.createdOpts.Labels["havn.cpus"])
	assert.Equal(t, "8g", svc.container.createdOpts.Labels["havn.memory"])
	assert.Equal(t, "12g", svc.container.createdOpts.Labels["havn.memory_swap"])
}

func TestNewRoot_RunE_DoltEnabledStartupPerformsSharedSetupAndInjectsBeadsEnv(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte("[dolt]\nenabled = true\nport = 3308\n"), 0o644))

	svc := newRootBoundaryStartService()
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{projectPath})

	err := root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.doltSetup.called)
	assert.Equal(t, "sample-project", svc.doltSetup.lastCfg.Dolt.Database)
	assert.Equal(t, 1, svc.container.createCalls)
	assert.Equal(t, "1", svc.container.createdOpts.Env["BEADS_DOLT_SHARED_SERVER"])
	assert.Equal(t, "havn-dolt", svc.container.createdOpts.Env["BEADS_DOLT_SERVER_HOST"])
	assert.Equal(t, "3308", svc.container.createdOpts.Env["BEADS_DOLT_SERVER_PORT"])
	assert.Equal(t, "root", svc.container.createdOpts.Env["BEADS_DOLT_SERVER_USER"])
	assert.Equal(t, "sample-project", svc.container.createdOpts.Env["BEADS_DOLT_SERVER_DATABASE"])
	assert.Equal(t, "0", svc.container.createdOpts.Env["BEADS_DOLT_AUTO_START"])
}

func TestNewRoot_RunE_ProjectExplicitFalseOverridesGlobalTrueForStartupBooleans(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	globalDir := filepath.Join(homeDir, ".config", "havn")
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "config.toml"), []byte("[dolt]\nenabled = true\n\n[mounts.ssh]\nforward_agent = true\nauthorized_keys = true\n"), 0o644))

	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte("[dolt]\nenabled = false\n\n[mounts.ssh]\nforward_agent = false\nauthorized_keys = false\n"), 0o644))

	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{projectPath})

	err := root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.False(t, svc.lastCfg.Dolt.Enabled)
	assert.False(t, svc.lastCfg.Mounts.SSH.ForwardAgent)
	assert.False(t, svc.lastCfg.Mounts.SSH.AuthorizedKeys)
}

func TestNewRoot_RunE_ConfigFlagOverridesDefaultGlobalConfigPathForStartup(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	defaultGlobalDir := filepath.Join(homeDir, ".config", "havn")
	require.NoError(t, os.MkdirAll(defaultGlobalDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(defaultGlobalDir, "config.toml"), []byte("shell = \"default-global-shell\"\n"), 0o644))

	customGlobalPath := filepath.Join(homeDir, "custom", "global.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(customGlobalPath), 0o755))
	require.NoError(t, os.WriteFile(customGlobalPath, []byte("shell = \"custom-global-shell\"\n"), 0o644))

	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{"--config", customGlobalPath, projectPath})

	err := root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.Equal(t, "custom-global-shell", svc.lastCfg.Shell)
}

func TestNewRoot_RunE_AppliesRootRuntimeFlagsToStartupConfig(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte("shell = \"project-shell\"\nenv = \"github:project/env\"\nimage = \"havn-project:latest\"\nports = [\"2022:22\"]\n\n[resources]\ncpus = 2\nmemory = \"2g\"\n"), 0o644))

	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{"--shell", "flag-shell", "--env", "github:flag/env", "--cpus", "6", "--memory", "12g", "--port", "2244", "--image", "havn-flag:latest", projectPath})

	err := root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.Equal(t, "flag-shell", svc.lastCfg.Shell)
	assert.Equal(t, "github:flag/env", svc.lastCfg.Env)
	assert.Equal(t, 6, svc.lastCfg.Resources.CPUs)
	assert.Equal(t, "12g", svc.lastCfg.Resources.Memory)
	assert.Equal(t, "havn-flag:latest", svc.lastCfg.Image)
	assert.Equal(t, []string{"2022:22", "2244:22"}, svc.lastCfg.Ports)
}

func TestNewRoot_RunE_NoDoltFlagDisablesDoltWhenConfigEnablesIt(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte("[dolt]\nenabled = true\n"), 0o644))

	svc := &fakeStartService{}
	root := cli.NewRoot(cli.Deps{StartService: svc})
	root.SetArgs([]string{"--no-dolt", projectPath})

	err := root.Execute()

	require.NoError(t, err)
	assert.True(t, svc.called)
	assert.False(t, svc.lastCfg.Dolt.Enabled)
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
