package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/mount"
	"github.com/jorgengundersen/havn/internal/name"
)

// StartBackend abstracts container lifecycle operations for StartOrAttach.
type StartBackend interface {
	ContainerInspect(ctx context.Context, name string) (State, error)
	ContainerCreate(ctx context.Context, opts CreateOpts) (string, error)
	ContainerStart(ctx context.Context, id string) error
}

// State holds the result of inspecting a container.
type State struct {
	ID      string
	Running bool
}

// NetworkBackend abstracts network operations for StartOrAttach.
type NetworkBackend interface {
	NetworkInspect(ctx context.Context, name string) error
	NetworkCreate(ctx context.Context, name string) error
}

// VolumeEnsurer ensures named volumes exist.
type VolumeEnsurer interface {
	EnsureExists(ctx context.Context, name string) error
}

// MountResolver resolves mount specifications for container creation.
type MountResolver interface {
	Resolve(cfg config.Config, projectPath string) (mount.ResolveResult, error)
}

// DoltSetup ensures the Dolt server and project database are ready.
type DoltSetup interface {
	EnsureReady(ctx context.Context, cfg config.Config) (map[string]string, error)
	MigrationNotice(ctx context.Context, cfg config.Config, projectPath string) (string, error)
}

// ExecBackend runs commands inside containers.
type ExecBackend interface {
	ContainerExec(ctx context.Context, name string, cmd []string) error
	ContainerExecInteractive(ctx context.Context, name string, cmd []string, workdir string) (int, error)
}

// NixRegistryPreparer prepares persistent in-container registry aliases.
type NixRegistryPreparer interface {
	Prepare(ctx context.Context, containerName string) error
}

// CreateOpts holds parameters for creating a container.
type CreateOpts struct {
	Name       string
	Image      string
	Network    string
	Ports      []string
	Mounts     []mount.Spec
	Env        map[string]string
	Labels     map[string]string
	Entrypoint []string
	User       string
	CPUs       int
	Memory     string
	MemorySwap string
	AutoRemove bool
}

// PortChecker validates host-port availability for requested publishes.
type PortChecker interface {
	EnsureAvailable(ports []string) error
}

// StartDeps aggregates all dependencies for StartOrAttach.
type StartDeps struct {
	Container                     StartBackend
	Image                         ImageBackend
	Network                       NetworkBackend
	Volume                        VolumeEnsurer
	Mount                         MountResolver
	Dolt                          DoltSetup // nil to skip Dolt setup
	Exec                          ExecBackend
	NixRegistry                   NixRegistryPreparer
	PortChecker                   PortChecker
	Status                        func(msg string)
	StartupCheckTelemetry         *StartupCheckTelemetry
	StartupCheckHeartbeatInterval time.Duration
	StartupCheckHeartbeatTicker   StartupCheckHeartbeatTickerFactory
}

// StartupCheckHeartbeatTickerFactory creates a heartbeat ticker channel and a
// stop function for startup-check status updates.
type StartupCheckHeartbeatTickerFactory func(interval time.Duration) (<-chan time.Time, func())

// StartOptions controls startup behavior that is invocation-scoped.
type StartOptions struct {
	VerboseStartup        bool
	Mode                  StartupMode
	StartupChecks         StartupCheckMode
	StartupCheckTelemetry *StartupCheckTelemetry
}

// StartupMode selects whether startup enters an interactive shell.
type StartupMode int

// StartupCheckMode selects which startup-check phases run before shell handoff
// or no-attach completion.
type StartupCheckMode int

const (
	// StartupModeAttach runs startup then enters the configured dev shell.
	StartupModeAttach StartupMode = iota
	// StartupModeNoAttach runs lifecycle startup and exits without shell attach.
	StartupModeNoAttach
)

const (
	// StartupCheckDefault uses command-default startup-check behavior.
	StartupCheckDefault StartupCheckMode = iota
	// StartupCheckValidate runs required devShell validation only.
	StartupCheckValidate
	// StartupCheckPrepare runs validation and optional prepare.
	StartupCheckPrepare
)

// StartOrAttach implements the startup orchestration (steps 3-10).
// If the container is already running, it attaches to it. Otherwise, it
// ensures all infrastructure exists, creates the container, runs init,
// and attaches to the devShell. Returns the exit code from the shell session.
func StartOrAttach(ctx context.Context, deps StartDeps, cfg config.Config, projectPath string) (int, error) {
	return StartOrAttachWithOptions(ctx, deps, cfg, projectPath, StartOptions{})
}

// Start runs startup orchestration without interactive shell attach.
func Start(ctx context.Context, deps StartDeps, cfg config.Config, projectPath string) error {
	return StartWithOptions(ctx, deps, cfg, projectPath, StartOptions{})
}

// StartWithOptions runs startup orchestration without interactive shell attach
// and supports invocation-scoped options.
func StartWithOptions(ctx context.Context, deps StartDeps, cfg config.Config, projectPath string, opts StartOptions) error {
	opts.Mode = StartupModeNoAttach
	_, err := StartOrAttachWithOptions(ctx, deps, cfg, projectPath, opts)
	return err
}

// StartOrAttachWithOptions runs startup orchestration with invocation-scoped
// runtime options.
func StartOrAttachWithOptions(ctx context.Context, deps StartDeps, cfg config.Config, projectPath string, opts StartOptions) (int, error) {
	resolvedOpts, err := normalizeStartOptions(opts)
	if err != nil {
		return 0, err
	}
	opts = resolvedOpts

	if deps.StartupCheckTelemetry == nil {
		deps.StartupCheckTelemetry = opts.StartupCheckTelemetry
	}

	cname, err := deriveContainerName(projectPath)
	if err != nil {
		return 0, err
	}

	// Step 3: check if container is running.
	state, err := deps.Container.ContainerInspect(ctx, string(cname))
	if err != nil {
		var notFound *NotFoundError
		if !errors.As(err, &notFound) {
			return 0, fmt.Errorf("inspect container %q: %w", cname, err)
		}
		// Container doesn't exist — proceed to create.
	} else if state.Running {
		if err := prepareNixRegistry(ctx, deps, string(cname)); err != nil {
			return 0, err
		}
		if shouldRunValidation(opts) {
			if err := prepareStartupSession(ctx, deps, string(cname), cfg, projectPath, opts); err != nil {
				return 0, err
			}
		}
		if opts.Mode == StartupModeNoAttach {
			return 0, nil
		}
		return attach(ctx, deps.Exec, string(cname), cfg, projectPath, opts)
	} else {
		if err := deps.Container.ContainerStart(ctx, state.ID); err != nil {
			return 0, fmt.Errorf("start container %q: %w", cname, err)
		}

		// Step 9: post-start init (sshd).
		if err := deps.Exec.ContainerExec(ctx, string(cname), sshdCmd); err != nil {
			return 0, fmt.Errorf("init sshd in container %q: %w", cname, err)
		}
		if err := prepareNixRegistry(ctx, deps, string(cname)); err != nil {
			return 0, err
		}
		if shouldRunValidation(opts) {
			if err := prepareStartupSession(ctx, deps, string(cname), cfg, projectPath, opts); err != nil {
				return 0, err
			}
		}

		if opts.Mode == StartupModeNoAttach {
			return 0, nil
		}
		return attach(ctx, deps.Exec, string(cname), cfg, projectPath, opts)
	}

	// Steps 4-7: ensure infrastructure.
	if err := ensureInfrastructure(ctx, deps, cfg); err != nil {
		return 0, err
	}

	// Step 8: create and start container.
	id, err := createContainer(ctx, deps, cfg, string(cname), projectPath)
	if err != nil {
		return 0, err
	}
	deps.Status(fmt.Sprintf(
		"Created container %s with resources cpus=%d memory=%s memory_swap=%s",
		cname,
		cfg.Resources.CPUs,
		cfg.Resources.Memory,
		cfg.Resources.MemorySwap,
	))
	if err := deps.Container.ContainerStart(ctx, id); err != nil {
		return 0, fmt.Errorf("start container %q: %w", cname, err)
	}

	// Step 9: post-start init (sshd).
	if err := deps.Exec.ContainerExec(ctx, string(cname), sshdCmd); err != nil {
		return 0, fmt.Errorf("init sshd in container %q: %w", cname, err)
	}

	if err := prepareNixRegistry(ctx, deps, string(cname)); err != nil {
		return 0, err
	}

	if shouldRunValidation(opts) {
		if err := prepareStartupSession(ctx, deps, string(cname), cfg, projectPath, opts); err != nil {
			return 0, err
		}
	}

	if opts.Mode == StartupModeNoAttach {
		return 0, nil
	}

	// Step 10: attach to devShell.
	return attach(ctx, deps.Exec, string(cname), cfg, projectPath, opts)
}

func prepareNixRegistry(ctx context.Context, deps StartDeps, containerName string) error {
	if deps.NixRegistry == nil {
		return nil
	}
	if err := deps.NixRegistry.Prepare(ctx, containerName); err != nil {
		return fmt.Errorf("prepare nix registry aliases in container %q: %w", containerName, err)
	}
	return nil
}

func prepareStartupSession(ctx context.Context, deps StartDeps, containerName string, cfg config.Config, projectPath string, opts StartOptions) error {
	if !shouldRunValidation(opts) {
		return nil
	}

	if err := runStartupCheckPhase(ctx, deps.StartupCheckTelemetry, deps.Status, deps.StartupCheckHeartbeatInterval, deps.StartupCheckHeartbeatTicker, StartupCheckPhaseValidation, func() error {
		return deps.Exec.ContainerExec(ctx, containerName, requiredDevShellValidationCmd(cfg, opts))
	}); err != nil {
		return fmt.Errorf("validate required devShell %q in container %q: %w (run 'havn enter %s' to debug startup validation manually)", cfg.Shell, containerName, err, projectPath)
	}

	if !shouldRunPrepare(opts) {
		return nil
	}

	err := runStartupCheckPhase(ctx, deps.StartupCheckTelemetry, deps.Status, deps.StartupCheckHeartbeatInterval, deps.StartupCheckHeartbeatTicker, StartupCheckPhasePrepare, func() error {
		return deps.Exec.ContainerExec(ctx, containerName, sessionPrepareCmd(cfg, opts))
	})
	if err == nil {
		return nil
	}
	if isMissingSessionPrepareCapability(err) {
		if deps.Status != nil {
			deps.Status("Optional startup capability havn-session-prepare not provided; continuing startup")
		}
		return nil
	}

	return fmt.Errorf("run optional startup capability havn-session-prepare in container %q: %w (run 'havn enter %s' to debug startup preparation manually)", containerName, err, projectPath)
}

func shouldRunValidation(opts StartOptions) bool {
	switch opts.StartupChecks {
	case StartupCheckValidate, StartupCheckPrepare:
		return true
	case StartupCheckDefault:
		return opts.Mode != StartupModeNoAttach
	default:
		return opts.Mode != StartupModeNoAttach
	}
}

func shouldRunPrepare(opts StartOptions) bool {
	switch opts.StartupChecks {
	case StartupCheckPrepare:
		return true
	case StartupCheckValidate:
		return false
	case StartupCheckDefault:
		return opts.Mode != StartupModeNoAttach
	default:
		return opts.Mode != StartupModeNoAttach
	}
}

func normalizeStartOptions(opts StartOptions) (StartOptions, error) {
	switch opts.Mode {
	case StartupModeAttach, StartupModeNoAttach:
		// valid
	default:
		return StartOptions{}, fmt.Errorf("invalid startup mode %d", opts.Mode)
	}

	switch opts.StartupChecks {
	case StartupCheckDefault, StartupCheckValidate, StartupCheckPrepare:
		// valid
	default:
		return StartOptions{}, fmt.Errorf("invalid startup check mode %d", opts.StartupChecks)
	}

	return opts, nil
}

func runStartupCheckPhase(ctx context.Context, telemetry *StartupCheckTelemetry, status func(msg string), heartbeatInterval time.Duration, tickerFactory StartupCheckHeartbeatTickerFactory, phase StartupCheckPhase, run func() error) error {
	startedAt := time.Now()
	if status != nil {
		status(fmt.Sprintf("Startup check phase %s started", phase))
	}
	heartbeatDone := make(chan struct{})
	heartbeatStop := make(chan struct{})
	if status != nil {
		go emitStartupCheckHeartbeats(phase, status, startedAt, heartbeatInterval, tickerFactory, heartbeatStop, heartbeatDone)
	}
	if telemetry != nil {
		telemetry.StartPhase(phase)
	}

	err := run()
	if status != nil {
		close(heartbeatStop)
		<-heartbeatDone
	}
	if err == nil {
		if telemetry != nil {
			telemetry.FinishPhase(phase)
		}
		if status != nil {
			status(fmt.Sprintf("Startup check phase %s completed in %s", phase, time.Since(startedAt).Round(time.Second)))
		}
		return nil
	}

	if interruption, ok := startupCheckInterruption(ctx, err); ok {
		if telemetry != nil {
			telemetry.CancelPhase(phase, interruption)
		}
		if status != nil {
			status(fmt.Sprintf("Startup check phase %s interrupted: %s", phase, interruption.Detail))
		}
	} else {
		if telemetry != nil {
			telemetry.ErrorPhase(phase, err)
		}
		if status != nil {
			status(fmt.Sprintf("Startup check phase %s failed: %s", phase, err.Error()))
		}
	}

	return err
}

const defaultStartupCheckHeartbeatInterval = 10 * time.Second

func emitStartupCheckHeartbeats(phase StartupCheckPhase, status func(msg string), startedAt time.Time, interval time.Duration, tickerFactory StartupCheckHeartbeatTickerFactory, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	if interval <= 0 {
		interval = defaultStartupCheckHeartbeatInterval
	}
	if tickerFactory == nil {
		tickerFactory = defaultStartupCheckHeartbeatTicker
	}
	ticks, stopTicker := tickerFactory(interval)
	defer stopTicker()
	for {
		select {
		case <-stop:
			return
		case <-ticks:
			status(fmt.Sprintf("Startup check phase %s still running (%s elapsed)", phase, time.Since(startedAt).Round(time.Second)))
		}
	}
}

func defaultStartupCheckHeartbeatTicker(interval time.Duration) (<-chan time.Time, func()) {
	ticker := time.NewTicker(interval)
	return ticker.C, ticker.Stop
}

func startupCheckInterruption(ctx context.Context, err error) (StartupCheckInterruption, bool) {
	if errors.Is(err, context.Canceled) {
		return StartupCheckInterruption{Cause: "context_canceled", Detail: err.Error()}, true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return StartupCheckInterruption{Cause: "deadline_exceeded", Detail: err.Error()}, true
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		cause := "context_interrupted"
		if errors.Is(ctxErr, context.Canceled) {
			cause = "context_canceled"
		}
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			cause = "deadline_exceeded"
		}
		return StartupCheckInterruption{Cause: cause, Detail: ctxErr.Error()}, true
	}
	return StartupCheckInterruption{}, false
}

func isMissingSessionPrepareCapability(err error) bool {
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "havn-session-prepare") {
		return false
	}
	if strings.Contains(msg, "does not provide attribute") {
		return true
	}
	if strings.Contains(msg, "attribute 'apps") {
		return true
	}
	return false
}

func nixFlakeCmdBase(opts StartOptions) []string {
	cmd := []string{
		"nix",
		"--extra-experimental-features", "nix-command flakes",
		"--option", "keep-build-log", "true",
	}
	if opts.VerboseStartup {
		cmd = append(cmd, "-v", "-L")
	}
	return cmd
}

func requiredDevShellValidationCmd(cfg config.Config, opts StartOptions) []string {
	cmd := nixFlakeCmdBase(opts)
	cmd = append(cmd, "develop", cfg.Env+"#"+cfg.Shell, "--command", "true")
	return cmd
}

func sessionPrepareCmd(cfg config.Config, opts StartOptions) []string {
	cmd := nixFlakeCmdBase(opts)
	cmd = append(cmd, "run", cfg.Env+"#havn-session-prepare")
	return cmd
}

var sshdCmd = []string{"sudo", "/usr/sbin/sshd"}

const containerUser = "devuser"

func ensureInfrastructure(ctx context.Context, deps StartDeps, cfg config.Config) error {
	// Step 4: ensure base image.
	exists, err := deps.Image.ImageExists(ctx, cfg.Image)
	if err != nil {
		return fmt.Errorf("check image %q: %w", cfg.Image, err)
	}
	if !exists {
		deps.Status(fmt.Sprintf("Image %s not found, building...", cfg.Image))
		if err := Build(ctx, deps.Image, BuildOpts{
			ImageName:   cfg.Image,
			ContextPath: "docker/",
			UID:         os.Getuid(),
			GID:         os.Getgid(),
		}); err != nil {
			return err
		}
	}

	// Step 5: ensure network.
	if err := deps.Network.NetworkInspect(ctx, cfg.Network); err != nil {
		var notFound *NetworkNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("inspect network %q: %w", cfg.Network, err)
		}

		deps.Status(fmt.Sprintf("Network %s not found, creating...", cfg.Network))
		if err := deps.Network.NetworkCreate(ctx, cfg.Network); err != nil {
			return fmt.Errorf("create network %q: %w", cfg.Network, err)
		}
	}

	// Step 6: ensure volumes.
	volumes := []string{cfg.Volumes.Nix, cfg.Volumes.Data, cfg.Volumes.Cache, cfg.Volumes.State}
	for _, v := range volumes {
		if err := deps.Volume.EnsureExists(ctx, v); err != nil {
			return fmt.Errorf("ensure volume %q: %w", v, err)
		}
	}

	// Step 7: Dolt setup (if enabled).
	// Handled in createContainer for env var injection.
	return nil
}

func createContainer(ctx context.Context, deps StartDeps, cfg config.Config, cname, projectPath string) (string, error) {
	// Resolve mounts.
	mountResult, err := deps.Mount.Resolve(cfg, projectPath)
	if err != nil {
		return "", fmt.Errorf("resolve mounts: %w", err)
	}

	// Build env vars.
	env := make(map[string]string)
	for k, v := range mountResult.Env {
		env[k] = v
	}
	resolvedConfigEnv, err := config.ResolveProjectEnvironment(cfg.Environment)
	if err != nil {
		return "", err
	}
	for k, v := range resolvedConfigEnv {
		env[k] = v
	}

	// Step 7: Dolt setup.
	if deps.Dolt != nil && cfg.Dolt.Enabled {
		doltEnv, err := deps.Dolt.EnsureReady(ctx, cfg)
		if err != nil {
			return "", err
		}
		notice, err := deps.Dolt.MigrationNotice(ctx, cfg, projectPath)
		if err != nil {
			return "", err
		}
		if notice != "" {
			deps.Status(notice)
		}
		for k, v := range doltEnv {
			env[k] = v
		}
	}

	// Build labels.
	labels := map[string]string{
		LabelManagedBy:  "havn",
		LabelPath:       projectPath,
		LabelShell:      cfg.Shell,
		LabelCPUs:       strconv.Itoa(cfg.Resources.CPUs),
		LabelMemory:     cfg.Resources.Memory,
		LabelMemorySwap: cfg.Resources.MemorySwap,
		LabelDolt:       strconv.FormatBool(cfg.Dolt.Enabled),
	}

	opts := CreateOpts{
		Name:       cname,
		Image:      cfg.Image,
		Network:    cfg.Network,
		Ports:      cfg.Ports,
		Mounts:     mountResult.Mounts,
		Env:        env,
		Labels:     labels,
		Entrypoint: []string{"tini", "--", "sleep", "infinity"},
		User:       containerUser,
		CPUs:       cfg.Resources.CPUs,
		Memory:     cfg.Resources.Memory,
		MemorySwap: cfg.Resources.MemorySwap,
		AutoRemove: true,
	}

	if deps.PortChecker != nil {
		if err := deps.PortChecker.EnsureAvailable(opts.Ports); err != nil {
			return "", err
		}
	}

	return deps.Container.ContainerCreate(ctx, opts)
}
func deriveContainerName(projectPath string) (name.ContainerName, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("derive container name: resolve path: %w", err)
	}

	parent, project, err := name.SplitProjectPath(filepath.Clean(absPath))
	if err != nil {
		return "", fmt.Errorf("derive container name: %w", err)
	}
	cname, err := name.DeriveContainerName(parent, project)
	if err != nil {
		return "", fmt.Errorf("derive container name: %w", err)
	}
	return cname, nil
}

func attach(ctx context.Context, exec ExecBackend, containerName string, cfg config.Config, projectPath string, opts StartOptions) (int, error) {
	return exec.ContainerExecInteractive(ctx, containerName, shellCmd(cfg, opts), projectPath)
}

func shellCmd(cfg config.Config, opts StartOptions) []string {
	cmd := nixFlakeCmdBase(opts)
	cmd = append(cmd, "develop", cfg.Env+"#"+cfg.Shell, "-c", "bash")
	return cmd
}
