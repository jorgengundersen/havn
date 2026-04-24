package container_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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
	execFn    func(name string, cmd []string) error

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

type fakeNixRegistryPreparer struct {
	calls []string
	err   error
}

func (f *fakeNixRegistryPreparer) Prepare(_ context.Context, containerName string) error {
	f.calls = append(f.calls, containerName)
	return f.err
}

func (f *fakePortChecker) EnsureAvailable(ports []string) error {
	f.checkedPorts = append([]string(nil), ports...)
	return f.err
}

type execCall struct {
	name string
	cmd  []string
}

func cmdHasToken(cmd []string, token string) bool {
	for _, value := range cmd {
		if value == token {
			return true
		}
	}
	return false
}

func (f *fakeExecBackend) ContainerExec(_ context.Context, name string, cmd []string) error {
	f.execCalls = append(f.execCalls, execCall{name: name, cmd: cmd})
	if f.execFn != nil {
		if err := f.execFn(name, cmd); err != nil {
			return err
		}
	}
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
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "develop", "github:user/env#default"}, exec.interactiveCmd)
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
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "-v", "-L", "develop", "github:user/env#default"}, exec.interactiveCmd)
}

func TestStartOrAttach_RunningContainer_DotProjectPathDerivesContainerName(t *testing.T) {
	workspace := t.TempDir()
	projectPath := filepath.Join(workspace, "user", "project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))
	t.Chdir(projectPath)

	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	deps := container.StartDeps{
		Container: &fakeStartBackend{inspectState: container.State{ID: "abc123", Running: true}},
		Exec:      exec,
		Status:    func(string) {},
	}
	cfg := config.Config{Env: "github:user/env", Shell: "default"}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, ".")

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Equal(t, ".", exec.interactiveWorkdir)
}

func TestStartOrAttach_RunningContainer_DotSlashProjectPathDerivesContainerName(t *testing.T) {
	workspace := t.TempDir()
	basePath := filepath.Join(workspace, "user")
	projectPath := filepath.Join(basePath, "project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))
	t.Chdir(basePath)

	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	deps := container.StartDeps{
		Container: &fakeStartBackend{inspectState: container.State{ID: "abc123", Running: true}},
		Exec:      exec,
		Status:    func(string) {},
	}
	cfg := config.Config{Env: "github:user/env", Shell: "default"}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, "./project")

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Equal(t, "./project", exec.interactiveWorkdir)
}

func TestStartOrAttach_RunningContainer_DotDotProjectPathDerivesContainerName(t *testing.T) {
	workspace := t.TempDir()
	parentPath := filepath.Join(workspace, "user")
	projectPath := filepath.Join(parentPath, "project")
	currentPath := filepath.Join(parentPath, "current")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))
	require.NoError(t, os.MkdirAll(currentPath, 0o755))
	t.Chdir(currentPath)

	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	deps := container.StartDeps{
		Container: &fakeStartBackend{inspectState: container.State{ID: "abc123", Running: true}},
		Exec:      exec,
		Status:    func(string) {},
	}
	cfg := config.Config{Env: "github:user/env", Shell: "default"}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, "../project")

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Equal(t, "../project", exec.interactiveWorkdir)
}

func TestStartOrAttach_RunningContainer_PreparesNixRegistry(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	registry := &fakeNixRegistryPreparer{}
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec:        exec,
		NixRegistry: registry,
		Status:      func(string) {},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, []string{"havn-user-project"}, registry.calls)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
}

func TestStartOrAttach_RunningContainer_PreparesStartupSessionBeforeAttach(t *testing.T) {
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

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	require.Len(t, exec.execCalls, 2)
	assert.Equal(t, "havn-user-project", exec.execCalls[0].name)
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "develop", "github:user/env#default", "--command", "true"}, exec.execCalls[0].cmd)
	assert.Equal(t, "havn-user-project", exec.execCalls[1].name)
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "run", "github:user/env#havn-session-prepare"}, exec.execCalls[1].cmd)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
}

func TestStartOrAttach_RunningContainer_RecordsStartupCheckPhaseTelemetry(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	telemetry := container.NewStartupCheckTelemetry()
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec:                  exec,
		StartupCheckTelemetry: telemetry,
		Status:                func(string) {},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	events := telemetry.Events()
	require.Len(t, events, 4)
	assert.Equal(t, container.StartupCheckPhaseValidation, events[0].Phase)
	assert.Equal(t, container.StartupCheckPhaseOutcomeStart, events[0].Outcome)
	assert.Equal(t, container.StartupCheckPhaseValidation, events[1].Phase)
	assert.Equal(t, container.StartupCheckPhaseOutcomeFinish, events[1].Outcome)
	assert.Equal(t, container.StartupCheckPhasePrepare, events[2].Phase)
	assert.Equal(t, container.StartupCheckPhaseOutcomeStart, events[2].Outcome)
	assert.Equal(t, container.StartupCheckPhasePrepare, events[3].Phase)
	assert.Equal(t, container.StartupCheckPhaseOutcomeFinish, events[3].Outcome)
}

func TestStartOrAttach_RunningContainer_EmitsStartupCheckPhaseStatusMessages(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	var statusMessages []string
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec: exec,
		Status: func(msg string) {
			statusMessages = append(statusMessages, msg)
		},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, statusMessages, "Startup check phase validation started")
	assert.Contains(t, statusMessages, "Startup check phase prepare started")
	assert.Condition(t, func() bool {
		for _, msg := range statusMessages {
			if msg == "Startup check phase validation completed in 0s" {
				return true
			}
		}
		return false
	})
	assert.Condition(t, func() bool {
		for _, msg := range statusMessages {
			if msg == "Startup check phase prepare completed in 0s" {
				return true
			}
		}
		return false
	})
}

func TestStartOrAttach_RunningContainer_RecordsStartupCheckPhaseCancellation(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{}
	exec.execFn = func(_ string, cmd []string) error {
		if cmdHasToken(cmd, "develop") {
			return context.Canceled
		}
		return nil
	}
	telemetry := container.NewStartupCheckTelemetry()
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec:                  exec,
		StartupCheckTelemetry: telemetry,
		Status:                func(string) {},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "validate required devShell")
	events := telemetry.Events()
	require.Len(t, events, 2)
	assert.Equal(t, container.StartupCheckPhaseOutcomeStart, events[0].Outcome)
	assert.Equal(t, container.StartupCheckPhaseOutcomeCancel, events[1].Outcome)
	require.NotNil(t, events[1].Interruption)
	assert.Equal(t, "context_canceled", events[1].Interruption.Cause)
	assert.Equal(t, context.Canceled.Error(), events[1].Interruption.Detail)
}

func TestStartOrAttach_RunningContainer_ReportsInterruptedStartupCheckPhase(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{}
	exec.execFn = func(_ string, cmd []string) error {
		if cmdHasToken(cmd, "develop") {
			return context.Canceled
		}
		return nil
	}
	var statusMessages []string
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec: exec,
		Status: func(msg string) {
			statusMessages = append(statusMessages, msg)
		},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "validate required devShell")
	assert.Contains(t, statusMessages, "Startup check phase validation interrupted: context canceled")
}

func TestStartOrAttach_RunningContainer_EmitsHeartbeatForLongRunningStartupCheckPhase(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	heartbeatTicks := make(chan time.Time, 1)
	validationStarted := make(chan struct{})
	releaseValidation := make(chan struct{})
	heartbeatObserved := make(chan struct{}, 1)
	exec.execFn = func(_ string, cmd []string) error {
		if cmdHasToken(cmd, "develop") {
			close(validationStarted)
			<-releaseValidation
		}
		return nil
	}
	var statusMessages []string
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec: exec,
		Status: func(msg string) {
			statusMessages = append(statusMessages, msg)
			if strings.HasPrefix(msg, "Startup check phase validation still running (") {
				select {
				case heartbeatObserved <- struct{}{}:
				default:
				}
			}
		},
		StartupCheckHeartbeatInterval: time.Minute,
		StartupCheckHeartbeatTicker: func(time.Duration) (<-chan time.Time, func()) {
			return heartbeatTicks, func() {}
		},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	result := make(chan error, 1)
	go func() {
		_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)
		result <- err
	}()

	<-validationStarted
	heartbeatTicks <- time.Now()
	select {
	case <-heartbeatObserved:
	case <-time.After(time.Second):
		close(releaseValidation)
		require.FailNow(t, "expected heartbeat status message")
	}
	close(releaseValidation)
	err := <-result

	require.NoError(t, err)
	heartbeatFound := false
	for _, msg := range statusMessages {
		if strings.HasPrefix(msg, "Startup check phase validation still running (") {
			heartbeatFound = true
			break
		}
	}
	assert.True(t, heartbeatFound)
}

func TestStartOrAttach_RunningContainer_UsesDefaultHeartbeatCadenceForStartupCheckPhases(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	heartbeatIntervals := make([]time.Duration, 0, 2)

	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec: exec,
		Status: func(string) {
		},
		StartupCheckHeartbeatTicker: func(interval time.Duration) (<-chan time.Time, func()) {
			heartbeatIntervals = append(heartbeatIntervals, interval)
			return make(chan time.Time), func() {}
		},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, []time.Duration{10 * time.Second, 10 * time.Second}, heartbeatIntervals)
}

func TestStartOrAttach_RunningContainer_ReportsStartupCheckPhaseFailureStatus(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{}
	exec.execFn = func(_ string, cmd []string) error {
		if cmdHasToken(cmd, "run") {
			return fmt.Errorf("prepare hook failed")
		}
		return nil
	}
	var statusMessages []string
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec: exec,
		Status: func(msg string) {
			statusMessages = append(statusMessages, msg)
		},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "run optional startup capability havn-session-prepare")
	assert.Contains(t, statusMessages, "Startup check phase prepare failed: prepare hook failed")
}

func TestStartOrAttach_RunningContainer_RecordsStartupCheckPhaseFailure(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{}
	exec.execFn = func(_ string, cmd []string) error {
		if cmdHasToken(cmd, "run") {
			return fmt.Errorf("prepare hook failed")
		}
		return nil
	}
	telemetry := container.NewStartupCheckTelemetry()
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec:                  exec,
		StartupCheckTelemetry: telemetry,
		Status:                func(string) {},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	_, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.ErrorContains(t, err, "run optional startup capability havn-session-prepare")
	events := telemetry.Events()
	require.Len(t, events, 4)
	assert.Equal(t, container.StartupCheckPhaseOutcomeFinish, events[1].Outcome)
	assert.Equal(t, container.StartupCheckPhaseOutcomeStart, events[2].Outcome)
	assert.Equal(t, container.StartupCheckPhaseOutcomeError, events[3].Outcome)
	assert.Equal(t, "prepare hook failed", events[3].Error)
	assert.Nil(t, events[3].Interruption)
}

func TestStartOrAttach_RunningContainer_NixRegistryPrepareFailure_Aborts(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec:        exec,
		NixRegistry: &fakeNixRegistryPreparer{err: fmt.Errorf("malformed nix registry data")},
		Status:      func(string) {},
	}
	cfg := config.Config{
		Env:   "github:user/env",
		Shell: "default",
	}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.Equal(t, 0, exitCode)
	assert.ErrorContains(t, err, "prepare nix registry aliases in container \"havn-user-project\"")
	assert.ErrorContains(t, err, "malformed nix registry data")
	assert.Empty(t, exec.interactiveName)
}

func TestStartOrAttach_RunningContainer_PrepareCapabilityFailure_AbortsWithGuidance(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{}
	exec.execFn = func(_ string, cmd []string) error {
		if cmdHasToken(cmd, "run") {
			return fmt.Errorf("prepare hook failed")
		}
		return nil
	}
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

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.Equal(t, 0, exitCode)
	assert.ErrorContains(t, err, "run optional startup capability havn-session-prepare in container \"havn-user-project\"")
	assert.ErrorContains(t, err, "havn enter "+testProjectPath)
	assert.Empty(t, exec.interactiveName)
}

func TestStartOrAttach_RunningContainer_MissingOptionalPrepareCapability_Continues(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	var statusMessages []string
	exec.execFn = func(_ string, cmd []string) error {
		if cmdHasToken(cmd, "run") {
			return fmt.Errorf("flake does not provide attribute 'apps.x86_64-linux.havn-session-prepare'")
		}
		return nil
	}
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec: exec,
		Status: func(msg string) {
			statusMessages = append(statusMessages, msg)
		},
	}
	cfg := config.Config{
		Env:   "./testdata/fixture_flakes/missing_optional_prepare",
		Shell: "default",
	}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Contains(t, exec.execCalls, execCall{
		name: "havn-user-project",
		cmd:  []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "run", "./testdata/fixture_flakes/missing_optional_prepare#havn-session-prepare"},
	})
	assert.Contains(t, statusMessages, "Optional startup capability havn-session-prepare not provided; continuing startup")
}

func TestStartOrAttach_RunningContainer_MissingRequiredDevShell_AbortsBeforePrepare(t *testing.T) {
	ctx := context.Background()
	exec := &fakeExecBackend{interactiveExitCode: 0}
	exec.execFn = func(_ string, cmd []string) error {
		if cmdHasToken(cmd, "develop") {
			return fmt.Errorf("flake does not provide attribute 'devShells.x86_64-linux.default'")
		}
		return nil
	}
	deps := container.StartDeps{
		Container: &fakeStartBackend{
			inspectState: container.State{ID: "abc123", Running: true},
		},
		Exec:   exec,
		Status: func(string) {},
	}
	cfg := config.Config{
		Env:   "./testdata/fixture_flakes/missing_required_devshell",
		Shell: "default",
	}

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	assert.Equal(t, 0, exitCode)
	assert.ErrorContains(t, err, "validate required devShell \"default\"")
	assert.ErrorContains(t, err, "havn enter "+testProjectPath)
	assert.Empty(t, exec.interactiveName)
	assert.Len(t, exec.execCalls, 1)
}

func TestStart_RunningContainer_SkipsStartupChecksWithoutInteractiveAttachByDefault(t *testing.T) {
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

func TestStartWithOptions_RunningContainer_ValidateOnlyRunsValidationPhase(t *testing.T) {
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

	err := container.StartWithOptions(ctx, deps, cfg, testProjectPath, container.StartOptions{StartupChecks: container.StartupCheckValidate})

	require.NoError(t, err)
	require.Len(t, exec.execCalls, 1)
	assert.Equal(t, "havn-user-project", exec.execCalls[0].name)
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "develop", "github:user/env#default", "--command", "true"}, exec.execCalls[0].cmd)
	assert.Empty(t, exec.interactiveName)
}

func TestStartWithOptions_InvalidStartupCheckMode_ReturnsError(t *testing.T) {
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

	err := container.StartWithOptions(ctx, deps, cfg, testProjectPath, container.StartOptions{StartupChecks: container.StartupCheckMode(99)})

	assert.ErrorContains(t, err, "invalid startup check mode")
	assert.Empty(t, exec.execCalls)
	assert.Empty(t, exec.interactiveName)
}

func TestStartWithOptions_RunningContainer_PrepareModeRunsValidationAndPrepare(t *testing.T) {
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

	err := container.StartWithOptions(ctx, deps, cfg, testProjectPath, container.StartOptions{StartupChecks: container.StartupCheckPrepare})

	require.NoError(t, err)
	require.Len(t, exec.execCalls, 2)
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "develop", "github:user/env#default", "--command", "true"}, exec.execCalls[0].cmd)
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "run", "github:user/env#havn-session-prepare"}, exec.execCalls[1].cmd)
	assert.Empty(t, exec.interactiveName)
}

func TestStartOrAttachWithOptions_InvalidStartupMode_ReturnsError(t *testing.T) {
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

	_, err := container.StartOrAttachWithOptions(ctx, deps, cfg, testProjectPath, container.StartOptions{Mode: container.StartupMode(99)})

	assert.ErrorContains(t, err, "invalid startup mode")
	assert.Empty(t, exec.execCalls)
	assert.Empty(t, exec.interactiveName)
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
	assert.Equal(t, "12g", cb.createdOpts.Labels["havn.memory_swap"])
	assert.Equal(t, "false", cb.createdOpts.Labels["havn.dolt"])

	// Env includes SSH from mount resolution.
	assert.Equal(t, "/ssh-agent", cb.createdOpts.Env["SSH_AUTH_SOCK"])

	// Container was started.
	assert.Equal(t, "new-123", cb.startedID)

	// Sshd init and startup preparation were called.
	require.Len(t, exec.execCalls, 3)
	assert.Equal(t, "havn-user-project", exec.execCalls[0].name)
	assert.Equal(t, []string{"sudo", "/usr/sbin/sshd"}, exec.execCalls[0].cmd)
	assert.Equal(t, "havn-user-project", exec.execCalls[1].name)
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "develop", "github:user/env#default", "--command", "true"}, exec.execCalls[1].cmd)
	assert.Equal(t, "havn-user-project", exec.execCalls[2].name)
	assert.Equal(t, []string{"nix", "--extra-experimental-features", "nix-command flakes", "--option", "keep-build-log", "true", "run", "github:user/env#havn-session-prepare"}, exec.execCalls[2].cmd)

	// Interactive shell was attached.
	assert.Equal(t, "havn-user-project", exec.interactiveName)
	assert.Equal(t, testProjectPath, exec.interactiveWorkdir)
}

func TestStartOrAttach_NewContainer_ReportsAppliedResourceLimits(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectErr: &container.NotFoundError{Name: "havn-user-project"},
		createID:   "new-123",
	}
	exec := &fakeExecBackend{interactiveExitCode: 0}
	mounts := &fakeMountResolver{
		result: mount.ResolveResult{
			Mounts: []mount.Spec{{Source: testProjectPath, Target: testProjectPath, Type: "bind"}},
			Env:    map[string]string{"SSH_AUTH_SOCK": "/ssh-agent"},
		},
	}
	var statusMessages []string

	deps := container.StartDeps{
		Container: cb,
		Image:     &fakeImageBackend{existsResult: true},
		Network:   &fakeNetworkBackend{},
		Volume:    &fakeVolumeEnsurer{},
		Mount:     mounts,
		Exec:      exec,
		Status:    func(msg string) { statusMessages = append(statusMessages, msg) },
	}

	_, err := container.StartOrAttach(ctx, deps, defaultTestConfig(), testProjectPath)

	require.NoError(t, err)
	assert.Contains(t, statusMessages, "Created container havn-user-project with resources cpus=4 memory=8g memory_swap=12g")
}

func TestStart_NewContainer_StartsAndInitsWithoutStartupChecksByDefault(t *testing.T) {
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
	assert.Equal(t, []string{"sudo", "/usr/sbin/sshd"}, exec.execCalls[0].cmd)
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

func TestStartOrAttach_StoppedContainer_ReusesExistingResourceLimits(t *testing.T) {
	ctx := context.Background()
	cb := &fakeStartBackend{
		inspectState: container.State{ID: "stopped-123", Running: false},
	}
	exec := &fakeExecBackend{interactiveExitCode: 0}
	var statusMessages []string
	deps := container.StartDeps{
		Container: cb,
		Exec:      exec,
		Status:    func(msg string) { statusMessages = append(statusMessages, msg) },
	}
	cfg := defaultTestConfig()
	cfg.Resources.CPUs = 9
	cfg.Resources.Memory = "20g"
	cfg.Resources.MemorySwap = "24g"

	exitCode, err := container.StartOrAttach(ctx, deps, cfg, testProjectPath)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "stopped-123", cb.startedID)
	assert.Empty(t, cb.createdOpts.Name, "should not recreate existing container with new resource config")
	assert.NotContains(t, statusMessages, "Created container")
	assert.Equal(t, "havn-user-project", exec.interactiveName)
}

func TestStart_StoppedContainer_StartsAndInitsWithoutStartupChecksByDefault(t *testing.T) {
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
	assert.Equal(t, []string{"sudo", "/usr/sbin/sshd"}, exec.execCalls[0].cmd)
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
		notice: "Found local beads database at .beads/dolt/mydb for \"mydb\"; follow beads migration workflows (run 'bd migrate --help' and see docs/dolt-beads-guide.md) for /home/devuser/Repos/github.com/user/project",
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
