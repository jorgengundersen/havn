package doctor

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// --- 2.1 nix_store ---

type nixStoreCheck struct {
	checkMetadata
	backend   Backend
	container string
}

// NewNixStoreCheck creates check 2.1: /nix/store mounted and readable.
func NewNixStoreCheck(backend Backend, container string) Check {
	return &nixStoreCheck{
		checkMetadata: newContainerCheckMetadata("nix_store", container, []string{"docker_daemon"}, defaultTimeout),
		backend:       backend,
		container:     container,
	}
}

func (c *nixStoreCheck) Run(ctx context.Context) CheckResult {
	_, err := c.backend.ContainerExec(ctx, c.container, []string{"test", "-d", "/nix/store"})
	if err != nil {
		return CheckResult{
			Status:         StatusError,
			Message:        "Nix store volume is not mounted or is corrupt",
			Detail:         err.Error(),
			Recommendation: "Stop the container and restart with 'havn .'; if persistent, inspect the havn-nix volume",
		}
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "Nix store mounted",
	}
}

// --- 2.2 nix_devshell ---

type nixDevshellCheck struct {
	checkMetadata
	backend   Backend
	container string
	flakeRef  string
	shell     string
}

const devshellTimeout = 60 * time.Second

// NewNixDevshellCheck creates check 2.2: configured devShell evaluates.
func NewNixDevshellCheck(backend Backend, container, flakeRef, shell string) Check {
	return &nixDevshellCheck{
		checkMetadata: newContainerCheckMetadata("nix_devshell", container, []string{"docker_daemon"}, devshellTimeout),
		backend:       backend,
		container:     container,
		flakeRef:      flakeRef,
		shell:         shell,
	}
}

func (c *nixDevshellCheck) Run(ctx context.Context) CheckResult {
	ref := fmt.Sprintf("%s#%s", c.flakeRef, c.shell)
	_, err := c.backend.ContainerExec(ctx, c.container, []string{"nix", "develop", ref, "--command", "true"})
	if err != nil {
		return CheckResult{
			Status:         StatusWarn,
			Message:        "Nix devShell failed to evaluate",
			Detail:         err.Error(),
			Recommendation: fmt.Sprintf("Check the flake ref in config; run 'nix develop %s' manually for detailed errors", ref),
		}
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "devShell evaluates",
		Detail:  ref,
	}
}

// --- 2.3 project_mount ---

type projectMountCheck struct {
	checkMetadata
	backend     Backend
	container   string
	projectPath string
}

// NewProjectMountCheck creates check 2.3: project directory mounted and writable.
func NewProjectMountCheck(backend Backend, container, projectPath string) Check {
	return &projectMountCheck{
		checkMetadata: newContainerCheckMetadata("project_mount", container, []string{"docker_daemon"}, defaultTimeout),
		backend:       backend,
		container:     container,
		projectPath:   projectPath,
	}
}

func (c *projectMountCheck) Run(ctx context.Context) CheckResult {
	_, err := c.backend.ContainerExec(ctx, c.container, []string{"test", "-w", c.projectPath})
	if err != nil {
		return CheckResult{
			Status:         StatusError,
			Message:        "Project directory not accessible inside container",
			Detail:         err.Error(),
			Recommendation: "Stop the container and restart with 'havn .'",
		}
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "Project directory writable",
		Detail:  c.projectPath,
	}
}

// --- 2.3b config_mounts ---

type configMountsCheck struct {
	checkMetadata
	backend   Backend
	container string
	mounts    []ConfigMountExpectation
}

// ConfigMountExpectation describes a config mount path and expected mode.
type ConfigMountExpectation struct {
	Target   string
	ReadOnly bool
}

// NewConfigMountsCheck creates check 2.3b: config bind mounts present.
func NewConfigMountsCheck(backend Backend, container string, mounts []ConfigMountExpectation) Check {
	return &configMountsCheck{
		checkMetadata: newContainerCheckMetadata("config_mounts", container, []string{"docker_daemon"}, defaultTimeout),
		backend:       backend,
		container:     container,
		mounts:        mounts,
	}
}

func (c *configMountsCheck) Run(ctx context.Context) CheckResult {
	var missing []string
	var modeMismatch []string
	for _, m := range c.mounts {
		_, err := c.backend.ContainerExec(ctx, c.container, []string{"test", "-e", m.Target})
		if err != nil {
			missing = append(missing, m.Target)
			continue
		}

		_, writableErr := c.backend.ContainerExec(ctx, c.container, []string{"test", "-w", m.Target})
		if m.ReadOnly && writableErr == nil {
			modeMismatch = append(modeMismatch, m.Target+" expected ro")
		}
		if !m.ReadOnly && writableErr != nil {
			modeMismatch = append(modeMismatch, m.Target+" expected rw")
		}
	}
	if len(missing) > 0 || len(modeMismatch) > 0 {
		details := make([]string, 0, len(missing)+len(modeMismatch))
		if len(missing) > 0 {
			details = append(details, "missing: "+strings.Join(missing, ", "))
		}
		if len(modeMismatch) > 0 {
			details = append(details, "mode mismatch: "+strings.Join(modeMismatch, ", "))
		}
		return CheckResult{
			Status:         StatusWarn,
			Message:        "Config mounts do not match expected runtime wiring",
			Detail:         strings.Join(details, "; "),
			Recommendation: "Check configured mounts and restart with 'havn .'",
		}
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "Config mounts present",
	}
}

// --- 2.4 ssh_agent ---

type sshAgentCheck struct {
	checkMetadata
	backend    Backend
	container  string
	socketPath string
}

// NewSSHAgentCheck creates check 2.4: SSH agent forwarding works.
func NewSSHAgentCheck(backend Backend, container, socketPath string) Check {
	return &sshAgentCheck{
		checkMetadata: newContainerCheckMetadata("ssh_agent", container, []string{"docker_daemon"}, defaultTimeout),
		backend:       backend,
		container:     container,
		socketPath:    socketPath,
	}
}

func (c *sshAgentCheck) Run(ctx context.Context) CheckResult {
	if strings.TrimSpace(c.socketPath) == "" {
		return CheckResult{
			Status:         StatusWarn,
			Message:        "SSH agent not forwarding",
			Recommendation: "Ensure ssh-agent is running on host and SSH_AUTH_SOCK is set",
		}
	}
	_, err := c.backend.ContainerExec(ctx, c.container, []string{"test", "-S", c.socketPath})
	if err != nil {
		return CheckResult{
			Status:         StatusWarn,
			Message:        "SSH agent not forwarding",
			Detail:         err.Error(),
			Recommendation: "Ensure ssh-agent is running on host and SSH_AUTH_SOCK is set",
		}
	}
	_, err = c.backend.ContainerExec(ctx, c.container, []string{"ssh-add", "-l"})
	if err != nil {
		return CheckResult{
			Status:         StatusWarn,
			Message:        "SSH agent not forwarding",
			Detail:         err.Error(),
			Recommendation: "Ensure ssh-agent is running on host and SSH_AUTH_SOCK is set",
		}
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "SSH agent forwarding works",
	}
}

// --- 2.5 dolt_connectivity ---

type doltConnectivityCheck struct {
	checkMetadata
	backend     Backend
	container   string
	network     string
	doltEnabled bool
}

// NewDoltConnectivityCheck creates check 2.5: container connected to havn-net.
func NewDoltConnectivityCheck(backend Backend, container, network string, doltEnabled bool) Check {
	return &doltConnectivityCheck{
		checkMetadata: newContainerCheckMetadata("dolt_connectivity", container, []string{"docker_daemon"}, defaultTimeout),
		backend:       backend,
		container:     container,
		network:       network,
		doltEnabled:   doltEnabled,
	}
}

func (c *doltConnectivityCheck) Run(ctx context.Context) CheckResult {
	if !c.doltEnabled {
		return CheckResult{
			Status:  StatusSkip,
			Message: "Dolt not enabled",
		}
	}

	info, found, err := c.backend.NetworkInspect(ctx, c.network)
	if err != nil {
		return CheckResult{
			Status:  StatusError,
			Message: "Dolt connectivity check failed",
			Detail:  err.Error(),
		}
	}
	if !found {
		return CheckResult{
			Status:         StatusError,
			Message:        fmt.Sprintf("Container is not on the %s network", c.network),
			Recommendation: "Stop and restart the container with 'havn .'",
		}
	}

	for _, name := range info.Containers {
		if name == c.container {
			return CheckResult{
				Status:  StatusPass,
				Message: "Dolt network connected",
				Detail:  fmt.Sprintf("%s is on %s", c.container, c.network),
			}
		}
	}

	return CheckResult{
		Status:         StatusError,
		Message:        fmt.Sprintf("Container is not on the %s network", c.network),
		Recommendation: "Stop and restart the container with 'havn .'",
	}
}

// --- 2.6 beads_health ---

type beadsHealthCheck struct {
	checkMetadata
	backend     Backend
	container   string
	beadsExists bool
}

// NewBeadsHealthCheck creates check 2.6: bd doctor passes.
func NewBeadsHealthCheck(backend Backend, container string, beadsExists bool) Check {
	return &beadsHealthCheck{
		checkMetadata: newContainerCheckMetadata("beads_health", container, []string{"docker_daemon"}, defaultTimeout),
		backend:       backend,
		container:     container,
		beadsExists:   beadsExists,
	}
}

func (c *beadsHealthCheck) Run(ctx context.Context) CheckResult {
	if !c.beadsExists {
		return CheckResult{
			Status:  StatusSkip,
			Message: "No .beads/ directory",
		}
	}

	_, err := c.backend.ContainerExec(ctx, c.container, []string{"which", "bd"})
	if err != nil {
		return CheckResult{
			Status:         StatusWarn,
			Message:        "Beads directory exists but bd is not installed",
			Recommendation: "Ensure the project's Nix devShell includes beads",
		}
	}

	output, err := c.backend.ContainerExec(ctx, c.container, []string{"bd", "doctor", "--json"})
	if err != nil {
		return CheckResult{
			Status:         StatusWarn,
			Message:        "Beads health check failed",
			Detail:         err.Error(),
			Recommendation: "Run 'bd doctor' inside the container for detailed diagnostics",
		}
	}

	return CheckResult{
		Status:  StatusPass,
		Message: "Beads healthy",
		Detail:  output,
	}
}
