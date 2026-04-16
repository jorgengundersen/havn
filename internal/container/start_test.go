package container_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/mount"
)

// --- fakes ---

type fakeStartBackend struct {
	inspectState container.State
	inspectErr   error

	createdOpts container.CreateOpts
	createID    string
	createErr   error

	startedID string
	startErr  error
}

func (f *fakeStartBackend) ContainerInspect(_ context.Context, _ string) (container.State, error) {
	return f.inspectState, f.inspectErr
}

func (f *fakeStartBackend) ContainerCreate(_ context.Context, opts container.CreateOpts) (string, error) {
	f.createdOpts = opts
	return f.createID, f.createErr
}

func (f *fakeStartBackend) ContainerStart(_ context.Context, id string) error {
	f.startedID = id
	return f.startErr
}

type fakeNetworkBackend struct {
	inspectErr error
	createErr  error
	created    bool
}

func (f *fakeNetworkBackend) NetworkInspect(_ context.Context, _ string) error {
	return f.inspectErr
}

func (f *fakeNetworkBackend) NetworkCreate(_ context.Context, _ string) error {
	f.created = true
	return f.createErr
}

type fakeVolumeEnsurer struct {
	ensuredNames []string
	ensureErr    error
}

func (f *fakeVolumeEnsurer) EnsureExists(_ context.Context, name string) error {
	f.ensuredNames = append(f.ensuredNames, name)
	return f.ensureErr
}

type fakeMountResolver struct {
	result mount.ResolveResult
	err    error
}

func (f *fakeMountResolver) Resolve(_ config.Config, _ string) (mount.ResolveResult, error) {
	return f.result, f.err
}

type fakeDoltSetup struct {
	env    map[string]string
	err    error
	notice string
}

func (f *fakeDoltSetup) EnsureReady(_ context.Context, _ config.Config) (map[string]string, error) {
	return f.env, f.err
}

func (f *fakeDoltSetup) MigrationNotice(_ context.Context, _ config.Config, _ string) (string, error) {
	return f.notice, f.err
}

type fakeExecBackend struct {
	execCalls []execCall
	execErr   error

	interactiveExitCode int
	interactiveErr      error
	interactiveName     string
	interactiveCmd      []string
	interactiveWorkdir  string
}

type fakePortChecker struct {
	checkedPorts []string
	err          error
}

func (f *fakePortChecker) EnsureAvailable(ports []string) error {
	f.checkedPorts = append([]string(nil), ports...)
	return f.err
}

type execCall struct {
	name string
	cmd  []string
}

func (f *fakeExecBackend) ContainerExec(_ context.Context, name string, cmd []string) error {
	f.execCalls = append(f.execCalls, execCall{name: name, cmd: cmd})
	return f.execErr
}

func (f *fakeExecBackend) ContainerExecInteractive(_ context.Context, name string, cmd []string, workdir string) (int, error) {
	f.interactiveName = name
	f.interactiveCmd = cmd
	f.interactiveWorkdir = workdir
	return f.interactiveExitCode, f.interactiveErr
}

func defaultTestConfig() config.Config {
	return config.Config{
		Env:     "github:user/env",
		Shell:   "default",
		Image:   "havn-base:latest",
		Network: "havn-net",
		Resources: config.ResourceConfig{
			CPUs:       4,
			Memory:     "8g",
			MemorySwap: "12g",
		},
		Volumes: config.VolumeConfig{
			Nix:   "havn-nix",
			Data:  "havn-data",
			Cache: "havn-cache",
			State: "havn-state",
		},
	}
}

const testProjectPath = "/home/devuser/Repos/github.com/user/project"

// --- tests ---

func TestStartOrAttach_RunningContainer_DefaultStartupRetainsNixBuildLogs(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec:   exec,
		Status: func(string) {},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, "/home/devuser/Repos/github.com/user/project")

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "develop", "github:user/env#default", "-c", "bash"}, exec.interactiveCmd)
	assert.Equal(t, "/home/devuser/Repos/github.com/user/project", exec.interactiveWorkdir)
}

func TestStartOrAttach_RunningContainer_VerboseStartupEnablesDetailedNixLogs(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec:   exec,
		Status: func(string) {},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	exitCode, err := container.StartOrAttachWithOptions(ctx, deps, cfg, testProjectPath, container.StartOptions{VerboseStartup: true})

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "-v", "-L", "develop", "github:user/env#default", "-c", "bash"}, exec.interactiveCmd)
}

func TestStart_RunningContainer_CompletesWithoutInteractiveAttach(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec:   exec,
		Status: func(string) {},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	err := container.Start(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Empty(t, exec.interactiveName)
	assert.Empty(t, exec.execCalls)
}

func TestStartOrAttach_NewContainer(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	exec := &fakeExecBackend{interactiveExitCode: 0}
	mounts := &fakeMountResolver{
		result: mount.ResolveResult{
			Mounts: []mount.Spec{
				{Source: testProjectPath, Target: testProjectPath, Type: "bind"},
			},
			Env: map[string]string{"SSH_AUTH_SOCK": "/ssh-agent"},
		},
	}
	cfg := defaultTestConfig()
	deps := container.StartDeps{
		Container: cb,
		Image:     &fakeImageBackend{existsResult: true},
		Network:   &fakeNetworkBackend{},
		Volume:    &fakeVolumeEnsurer{},
		Mount:     mounts,
		Exec:      exec,
		Status:    func(string) {},
	}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	// Container was created with correct opts.
	assert.Equal(t, "havn-user-project", cb.createdOpts.Name)
	assert.Equal(t, "havn-base:latest", cb.createdOpts.Image)
	assert.Equal(t, "havn-net", cb.createdOpts.Network)
	assert.Equal(t, "devuser", cb.createdOpts.User)
	assert.Equal(t, []string{"tini", "--", "sleep", "infinity"}, cb.createdOpts.Entrypoint)
	assert.True(t, cb.createdOpts.AutoRemove)
	assert.Equal(t, 4, cb.createdOpts.CPUs)
	assert.Equal(t, "8g", cb.createdOpts.Memory)
	assert.Equal(t, "12g", cb.createdOpts.MemorySwap)

	// Labels include managed-by and metadata.
	assert.Equal(t, "havn", cb.createdOpts.Labels["managed-by"])
	assert.Equal(t, testProjectPath, cb.createdOpts.Labels["havn.path"])
	assert.Equal(t, "default", cb.createdOpts.Labels["havn.shell"])
	assert.Equal(t, "4", cb.createdOpts.Labels["havn.cpus"])
	assert.Equal(t, "8g", cb.createdOpts.Labels["havn.memory"])
	assert.Equal(t, "false", cb.createdOpts.Labels["havn.dolt"])

	// Env includes SSH from mount resolution.
	assert.Equal(t, "/ssh-agent", cb.createdOpts.Env["SSH_AUTH_SOCK"])

	// Container was started.
	assert.Equal(t, "new-123", cb.startedID)

	// Sshd init was called (best-effort).
	require.Len(t, exec.execCalls, 1)
	assert.Equal(t, "havn-user-project", exec.execCalls[0].name)

	// Interactive shell was attached.
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Equal(t, testProjectPath, exec.interactiveWorkdir)
}

func TestStart_NewContainer_StartsAndInitsWithoutInteractiveAttach(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	exec := &fakeExecBackend{interactiveExitCode: 0}
	mounts := &fakeMountResolver{
		result: mount.ResolveResult{
			Mounts: []mount.Spec{
				{Source: testProjectPath, Target: testProjectPath, Type: "bind"},
			},
			Env: map[string]string{"SSH_AUTH_SOCK": "/ssh-agent"},
		},
	}
	cfg := defaultTestConfig()
	deps := container.StartDeps{
		Container: cb,
		Image:     &fakeImageBackend{existsResult: true},
		Network:   &fakeNetworkBackend{},
		Volume:    &fakeVolumeEnsurer{},
		Mount:     mounts,
		Exec:      exec,
		Status:    func(string) {},
	}

	err := container.Start(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, "new-123", cb.startedID)
	require.Len(t, exec.execCalls, 1)
	assert.Equal(t, "havn-user-project", exec.execCalls[0].name)
	assert.Empty(t, exec.interactiveName)
}

func TestStartOrAttach_StoppedContainer_StartsExisting(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectState: container.State{ID: "stopped-123", Running: false},
	}
	exec := &fakeExecBackend{interactiveExitCode: 0}
	deps := container.StartDeps{
		Container: cb,
		Exec:      exec,
		Status:    func(string) {},
	}
	cfg := defaultTestConfig()

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "stopped-123", cb.startedID)
	assert.Empty(t, cb.createdOpts.Name, "should not create a new container when an existing one is stopped")
	assert.Equal(t, "havn-user-project", exec.interactiveName)
}

func TestStart_StoppedContainer_StartsAndInitsWithoutInteractiveAttach(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectState: container.State{ID: "stopped-123", Running: false},
	}
	exec := &fakeExecBackend{interactiveExitCode: 0}
	deps := container.StartDeps{
		Container: cb,
		Exec:      exec,
		Status:    func(string) {},
	}
	cfg := defaultTestConfig()

	err := container.Start(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, "stopped-123", cb.startedID)
	require.Len(t, exec.execCalls, 1)
	assert.Equal(t, "havn-user-project", exec.execCalls[0].name)
	assert.Empty(t, exec.interactiveName)
}

func TestStartOrAttach_StoppedContainer_InitFailure_AbortsStartup(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectState: container.State{ID: "stopped-123", Running: false},
	}
	exec := &fakeExecBackend{execErr: fmt.Errorf("sshd failed")}
	deps := container.StartDeps{
		Container: cb,
		Exec:      exec,
		Status:    func(string) {},
	}
	cfg := defaultTestConfig()

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.Equal(t, 0, exitCode)
	assert.ErrorContains(t, err, "init sshd in container \"havn-user-project\"")
	assert.ErrorContains(t, err, "sshd failed")
	assert.Empty(t, exec.interactiveName)
}

func TestStartOrAttach_ImageMissing_BuildsFirst(t *testing.T) {
	ctx := context.Background()
	imgBackend := &fakeImageBackend{existsResult: false}
	var statusMessages []string
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: &container.NotFoundError{Name: "havn-user-project"},
			createID:   "new-123",
		},
		Image:   imgBackend,
		Network: &fakeNetworkBackend{},
		Volume:  &fakeVolumeEnsurer{},
		Mount:   &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:    &fakeExecBackend{},
		Status:  func(msg string) { statusMessages = append(statusMessages, msg) },
	}
	cfg := defaultTestConfig()

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, "havn-base:latest", imgBackend.buildOpts.Tag)
	assert.Equal(t, "docker/", imgBackend.buildOpts.ContextPath)
	assert.Equal(t, strconv.Itoa(os.Getuid()), imgBackend.buildOpts.BuildArgs["UID"])
	assert.Equal(t, strconv.Itoa(os.Getgid()), imgBackend.buildOpts.BuildArgs["GID"])
	assert.Contains(t, statusMessages, "Image havn-base:latest not found, building...")
}

func TestStartOrAttach_NetworkMissing_CreatesWithStatus(t *testing.T) {
	ctx := context.Background()
	net := &fakeNetworkBackend{inspectErr: &container.NetworkNotFoundError{Name: "havn-net"}}
	var statusMessages []string
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: &container.NotFoundError{Name: "havn-user-project"},
			createID:   "new-123",
		},
		Image:   &fakeImageBackend{existsResult: true},
		Network: net,
		Volume:  &fakeVolumeEnsurer{},
		Mount:   &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:    &fakeExecBackend{},
		Status:  func(msg string) { statusMessages = append(statusMessages, msg) },
	}
	cfg := defaultTestConfig()

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.True(t, net.created)
	assert.Contains(t, statusMessages, "Network havn-net not found, creating...")
}

func TestStartOrAttach_NetworkInspectError_Aborts(t *testing.T) {
	ctx := context.Background()
	net := &fakeNetworkBackend{inspectErr: fmt.Errorf("permission denied")}
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		},
		Image:   &fakeImageBackend{existsResult: true},
		Network: net,
		Volume:  &fakeVolumeEnsurer{},
		Status:  func(string) {},
	}
	cfg := defaultTestConfig()

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "inspect network \"havn-net\"")
	assert.False(t, net.created)
}

func TestStartOrAttach_DoltEnabled(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	doltSetup := &fakeDoltSetup{
		env: map[string]string{
			"BEADS_DOLT_SERVER_HOST": "havn-dolt",
			"BEADS_DOLT_SERVER_PORT": "3308",
		},
	}
	deps := container.StartDeps{
		Container: cb,
		Image:     &fakeImageBackend{existsResult: true},
		Network:   &fakeNetworkBackend{},
		Volume:    &fakeVolumeEnsurer{},
		Mount:     &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Dolt:      doltSetup,
		Exec:      &fakeExecBackend{},
		Status:    func(string) {},
	}
	cfg := defaultTestConfig()
	cfg.Dolt.Enabled = true
	cfg.Dolt.Database = "mydb"

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, "havn-dolt", cb.createdOpts.Env["BEADS_DOLT_SERVER_HOST"])
	assert.Equal(t, "3308", cb.createdOpts.Env["BEADS_DOLT_SERVER_PORT"])
	assert.Equal(t, "true", cb.createdOpts.Labels["havn.dolt"])
}

func TestStartOrAttach_DoltEnabled_WithMigrationNotice_ShowsStatus(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	var statusMessages []string
	doltSetup := &fakeDoltSetup{
		env: map[string]string{
			"BEADS_DOLT_SERVER_HOST": "havn-dolt",
			"BEADS_DOLT_SERVER_PORT": "3308",
		},
		notice: "Found local beads database at .beads/dolt/mydb for \"mydb\"; migrate with: havn dolt import /home/devuser/Repos/github.com/user/project",
	}
	deps := container.StartDeps{
		Container: cb,
		Image:     &fakeImageBackend{existsResult: true},
		Network:   &fakeNetworkBackend{},
		Volume:    &fakeVolumeEnsurer{},
		Mount:     &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Dolt:      doltSetup,
		Exec:      &fakeExecBackend{},
		Status:    func(msg string) { statusMessages = append(statusMessages, msg) },
	}
	cfg := defaultTestConfig()
	cfg.Dolt.Enabled = true
	cfg.Dolt.Database = "mydb"

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Contains(t, statusMessages, doltSetup.notice)
}

func TestStartOrAttach_DoltDisabled_SkipsSetup(t *testing.T) {
	ctx := context.Background()
	doltSetup := &fakeDoltSetup{
		env: map[string]string{"SHOULD_NOT": "appear"},
	}
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	deps := container.StartDeps{
		Container: cb,
		Image:     &fakeImageBackend{existsResult: true},
		Network:   &fakeNetworkBackend{},
		Volume:    &fakeVolumeEnsurer{},
		Mount:     &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Dolt:      doltSetup,
		Exec:      &fakeExecBackend{},
		Status:    func(string) {},
	}
	cfg := defaultTestConfig()
	cfg.Dolt.Enabled = false

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	_, hasDoltEnv := cb.createdOpts.Env["SHOULD_NOT"]
	assert.False(t, hasDoltEnv, "dolt env vars should not be present when dolt is disabled")
}

func TestStartOrAttach_InitFailure_AbortsStartup(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	// Override ContainerExec to return an error for sshd init.
	exec.execErr = fmt.Errorf("sshd failed")
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: &container.NotFoundError{Name: "havn-user-project"},
			createID:   "new-123",
		},
		Image:   &fakeImageBackend{existsResult: true},
		Network: &fakeNetworkBackend{},
		Volume:  &fakeVolumeEnsurer{},
		Mount:   &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:    exec,
		Status:  func(string) {},
	}
	cfg := defaultTestConfig()

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.Equal(t, 0, exitCode)
	assert.ErrorContains(t, err, "sshd failed")
	assert.Empty(t, exec.interactiveName, "should not attach to shell when sshd init fails")
}

func TestStartOrAttach_InspectError_Aborts(t *testing.T) {
	ctx := context.Background()
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: fmt.Errorf("docker daemon not running"),
		},
		Status: func(string) {},
	}
	cfg := defaultTestConfig()

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "docker daemon not running")
}

func TestStartOrAttach_VolumeEnsureError_Aborts(t *testing.T) {
	ctx := context.Background()
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		},
		Image:   &fakeImageBackend{existsResult: true},
		Network: &fakeNetworkBackend{},
		Volume:  &fakeVolumeEnsurer{ensureErr: fmt.Errorf("volume create failed")},
		Status:  func(string) {},
	}
	cfg := defaultTestConfig()

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "volume create failed")
}

func TestStartOrAttach_ContainerCreateError_Aborts(t *testing.T) {
	ctx := context.Background()
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: &container.NotFoundError{Name: "havn-user-project"},
			createErr:  fmt.Errorf("name conflict"),
		},
		Image:   &fakeImageBackend{existsResult: true},
		Network: &fakeNetworkBackend{},
		Volume:  &fakeVolumeEnsurer{},
		Mount:   &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:    &fakeExecBackend{},
		Status:  func(string) {},
	}
	cfg := defaultTestConfig()

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "name conflict")
}

func TestStartOrAttach_EnsuresAllVolumes(t *testing.T) {
	ctx := context.Background()
	vol := &fakeVolumeEnsurer{}
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: &container.NotFoundError{Name: "havn-user-project"},
			createID:   "new-123",
		},
		Image:   &fakeImageBackend{existsResult: true},
		Network: &fakeNetworkBackend{},
		Volume:  vol,
		Mount:   &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:    &fakeExecBackend{},
		Status:  func(string) {},
	}
	cfg := defaultTestConfig()

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, []string{"havn-nix", "havn-data", "havn-cache", "havn-state"}, vol.ensuredNames)
}

func TestStartOrAttach_ConfigEnvironment_IncludedInEnv(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	deps := container.StartDeps{
		Container: cb,
		Image:     &fakeImageBackend{existsResult: true},
		Network:   &fakeNetworkBackend{},
		Volume:    &fakeVolumeEnsurer{},
		Mount:     &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:      &fakeExecBackend{},
		Status:    func(string) {},
	}
	cfg := defaultTestConfig()
	cfg.Environment = map[string]string{
		"MY_API_KEY": "secret123",
		"DEBUG":      "true",
	}

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, "secret123", cb.createdOpts.Env["MY_API_KEY"])
	assert.Equal(t, "true", cb.createdOpts.Env["DEBUG"])
}

func TestStartOrAttach_ConfigEnvironment_PassthroughResolvesHostVar(t *testing.T) {
	t.Setenv("MY_API_KEY", "host-secret")

	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	deps := container.StartDeps{
		Container: cb,
		Image:     &fakeImageBackend{existsResult: true},
		Network:   &fakeNetworkBackend{},
		Volume:    &fakeVolumeEnsurer{},
		Mount:     &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:      &fakeExecBackend{},
		Status:    func(string) {},
	}
	cfg := defaultTestConfig()
	cfg.Environment = map[string]string{
		"MY_API_KEY": "${MY_API_KEY}",
	}

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, "host-secret", cb.createdOpts.Env["MY_API_KEY"])
}

func TestStartOrAttach_ConfigEnvironment_ReservedDoltVarRejected(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	deps := container.StartDeps{
		Container: cb,
		Image:     &fakeImageBackend{existsResult: true},
		Network:   &fakeNetworkBackend{},
		Volume:    &fakeVolumeEnsurer{},
		Mount:     &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:      &fakeExecBackend{},
		Status:    func(string) {},
	}
	cfg := defaultTestConfig()
	cfg.Environment = map[string]string{
		"BEADS_DOLT_SERVER_HOST": "custom-host",
	}

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	var valErr *config.ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "environment.BEADS_DOLT_SERVER_HOST", valErr.Field)
}

func TestStartOrAttach_ConfigEnvironment_UnsetPassthroughVarRejected(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	deps := container.StartDeps{
		Container: cb,
		Image:     &fakeImageBackend{existsResult: true},
		Network:   &fakeNetworkBackend{},
		Volume:    &fakeVolumeEnsurer{},
		Mount:     &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:      &fakeExecBackend{},
		Status:    func(string) {},
	}
	cfg := defaultTestConfig()
	cfg.Environment = map[string]string{
		"API_KEY": "${UNSET_API_KEY}",
	}

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	var valErr *config.ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "environment.API_KEY", valErr.Field)
}

func TestStartOrAttach_HostPortCheckUsesEffectivePorts(t *testing.T) {
	ctx := context.Background()
	checker := &fakePortChecker{}
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	deps := container.StartDeps{
		Container:   cb,
		Image:       &fakeImageBackend{existsResult: true},
		Network:     &fakeNetworkBackend{},
		Volume:      &fakeVolumeEnsurer{},
		Mount:       &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:        &fakeExecBackend{},
		PortChecker: checker,
		Status:      func(string) {},
	}
	cfg := defaultTestConfig()
	cfg.Ports = []string{"2222:22", "8080:8080"}

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, []string{"2222:22", "8080:8080"}, checker.checkedPorts)
}

func TestStartOrAttach_HostPortCheckFailure_AbortsStartup(t *testing.T) {
	ctx := context.Background()
	checker := &fakePortChecker{err: fmt.Errorf("host port 8080 is not available")}
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	deps := container.StartDeps{
		Container:   cb,
		Image:       &fakeImageBackend{existsResult: true},
		Network:     &fakeNetworkBackend{},
		Volume:      &fakeVolumeEnsurer{},
		Mount:       &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Exec:        &fakeExecBackend{},
		PortChecker: checker,
		Status:      func(string) {},
	}
	cfg := defaultTestConfig()
	cfg.Ports = []string{"8080:8080"}

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "host port 8080 is not available")
	assert.Empty(t, cb.createdOpts.Name)
}

func TestStartOrAttach_ShellExitCode_Propagated(t *testing.T) {
	ctx := context.Background()
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec:   &fakeExecBackend{interactiveExitCode: 42},
		Status: func(string) {},
	}
	cfg := defaultTestConfig()

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, 42, exitCode)
}

func TestStartOrAttach_ImageBuildError_Aborts(t *testing.T) {
	ctx := context.Background()
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		},
		Image:   &fakeImageBackend{existsResult: false, buildErr: fmt.Errorf("docker build failed")},
		Network: &fakeNetworkBackend{},
		Status:  func(string) {},
	}
	cfg := defaultTestConfig()

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	var buildErr *container.BuildError
	assert.ErrorAs(t, err, &buildErr)
}

func TestStartOrAttach_NetworkCreateError_Aborts(t *testing.T) {
	ctx := context.Background()
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		},
		Image:   &fakeImageBackend{existsResult: true},
		Network: &fakeNetworkBackend{inspectErr: &container.NetworkNotFoundError{Name: "havn-net"}, createErr: fmt.Errorf("network create failed")},
		Status:  func(string) {},
	}
	cfg := defaultTestConfig()

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "network create failed")
}

func TestStartOrAttach_DoltSetupError_Aborts(t *testing.T) {
	ctx := context.Background()
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectErr: &container.NotFoundError{Name: "havn-user-project"},
			createID:   "new-123",
		},
		Image:   &fakeImageBackend{existsResult: true},
		Network: &fakeNetworkBackend{},
		Volume:  &fakeVolumeEnsurer{},
		Mount:   &fakeMountResolver{result: mount.ResolveResult{Env: map[string]string{}}},
		Dolt:    &fakeDoltSetup{err: fmt.Errorf("dolt health check timed out")},
		Exec:    &fakeExecBackend{},
		Status:  func(string) {},
	}
	cfg := defaultTestConfig()
	cfg.Dolt.Enabled = true

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "dolt health check timed out")
}
