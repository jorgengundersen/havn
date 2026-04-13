package doctor

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// --- 2.1 nix_store ---

type nixStoreCheck struct {
	backend   Backend
	container string
}

// NewNixStoreCheck creates check 2.1: /nix/store mounted and readable.
func NewNixStoreCheck(backend Backend, container string) Check {
	return &nixStoreCheck{backend: backend, container: container}
}

func (c *nixStoreCheck) ID() string              { return "nix_store" }
func (c *nixStoreCheck) Tier() string            { return "container" }
func (c *nixStoreCheck) Container() string       { return c.container }
func (c *nixStoreCheck) Prerequisites() []string { return []string{"docker_daemon"} }
func (c *nixStoreCheck) Timeout() time.Duration  { return defaultTimeout }

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
	backend   Backend
	container string
	flakeRef  string
	shell     string
}

const devshellTimeout = 60 * time.Second

// NewNixDevshellCheck creates check 2.2: configured devShell evaluates.
func NewNixDevshellCheck(backend Backend, container, flakeRef, shell string) Check {
	return &nixDevshellCheck{backend: backend, container: container, flakeRef: flakeRef, shell: shell}
}

func (c *nixDevshellCheck) ID() string              { return "nix_devshell" }
func (c *nixDevshellCheck) Tier() string            { return "container" }
func (c *nixDevshellCheck) Container() string       { return c.container }
func (c *nixDevshellCheck) Prerequisites() []string { return []string{"docker_daemon"} }
func (c *nixDevshellCheck) Timeout() time.Duration  { return devshellTimeout }

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
	backend     Backend
	container   string
	projectPath string
}

// NewProjectMountCheck creates check 2.3: project directory mounted and writable.
func NewProjectMountCheck(backend Backend, container, projectPath string) Check {
	return &projectMountCheck{backend: backend, container: container, projectPath: projectPath}
}

func (c *projectMountCheck) ID() string              { return "project_mount" }
func (c *projectMountCheck) Tier() string            { return "container" }
func (c *projectMountCheck) Container() string       { return c.container }
func (c *projectMountCheck) Prerequisites() []string { return []string{"docker_daemon"} }
func (c *projectMountCheck) Timeout() time.Duration  { return defaultTimeout }

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
	backend   Backend
	container string
	mounts    []ConfigMountExpectation
}

type ConfigMountExpectation struct {
	Target   string
	ReadOnly bool
}

// NewConfigMountsCheck creates check 2.3b: config bind mounts present.
func NewConfigMountsCheck(backend Backend, container string, mounts []ConfigMountExpectation) Check {
	return &configMountsCheck{backend: backend, container: container, mounts: mounts}
}

func (c *configMountsCheck) ID() string              { return "config_mounts" }
func (c *configMountsCheck) Tier() string            { return "container" }
func (c *configMountsCheck) Container() string       { return c.container }
func (c *configMountsCheck) Prerequisites() []string { return []string{"docker_daemon"} }
func (c *configMountsCheck) Timeout() time.Duration  { return defaultTimeout }

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
	backend    Backend
	container  string
	socketPath string
}

// NewSSHAgentCheck creates check 2.4: SSH agent forwarding works.
func NewSSHAgentCheck(backend Backend, container, socketPath string) Check {
	return &sshAgentCheck{backend: backend, container: container, socketPath: socketPath}
}

func (c *sshAgentCheck) ID() string              { return "ssh_agent" }
func (c *sshAgentCheck) Tier() string            { return "container" }
func (c *sshAgentCheck) Container() string       { return c.container }
func (c *sshAgentCheck) Prerequisites() []string { return []string{"docker_daemon"} }
func (c *sshAgentCheck) Timeout() time.Duration  { return defaultTimeout }

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
	backend     Backend
	container   string
	network     string
	doltEnabled bool
}

// NewDoltConnectivityCheck creates check 2.5: container connected to havn-net.
func NewDoltConnectivityCheck(backend Backend, container, network string, doltEnabled bool) Check {
	return &doltConnectivityCheck{backend: backend, container: container, network: network, doltEnabled: doltEnabled}
}

func (c *doltConnectivityCheck) ID() string              { return "dolt_connectivity" }
func (c *doltConnectivityCheck) Tier() string            { return "container" }
func (c *doltConnectivityCheck) Container() string       { return c.container }
func (c *doltConnectivityCheck) Prerequisites() []string { return []string{"docker_daemon"} }
func (c *doltConnectivityCheck) Timeout() time.Duration  { return defaultTimeout }

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
	backend     Backend
	container   string
	beadsExists bool
}

// NewBeadsHealthCheck creates check 2.6: bd doctor passes.
func NewBeadsHealthCheck(backend Backend, container string, beadsExists bool) Check {
	return &beadsHealthCheck{backend: backend, container: container, beadsExists: beadsExists}
}

func (c *beadsHealthCheck) ID() string              { return "beads_health" }
func (c *beadsHealthCheck) Tier() string            { return "container" }
func (c *beadsHealthCheck) Container() string       { return c.container }
func (c *beadsHealthCheck) Prerequisites() []string { return []string{"docker_daemon"} }
func (c *beadsHealthCheck) Timeout() time.Duration  { return defaultTimeout }

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
