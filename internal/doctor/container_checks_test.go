package doctor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/doctor"
)

// --- nix_store check ---

func TestNixStoreCheck_Pass(t *testing.T) {
	backend := newFakeBackend()
	check := doctor.NewNixStoreCheck(backend, "havn-user-myproject")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
	assert.Equal(t, "Nix store mounted", result.Message)
}

func TestNixStoreCheck_Failure(t *testing.T) {
	backend := newFakeBackend()
	backend.execErr = errors.New("test -d failed")
	check := doctor.NewNixStoreCheck(backend, "havn-user-myproject")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusError, result.Status)
	assert.Contains(t, result.Recommendation, "havn-nix")
}

func TestNixStoreCheck_Metadata(t *testing.T) {
	check := doctor.NewNixStoreCheck(newFakeBackend(), "havn-user-myproject")

	assert.Equal(t, "nix_store", check.ID())
	assert.Equal(t, "container", check.Tier())
	assert.Equal(t, "havn-user-myproject", check.Container())
	assert.Equal(t, []string{"docker_daemon"}, check.Prerequisites())
}

// --- nix_devshell check ---

func TestNixDevshellCheck_Pass(t *testing.T) {
	backend := newFakeBackend()
	backend.execResults["true"] = ""
	check := doctor.NewNixDevshellCheck(backend, "havn-user-myproject", "github:user/repo", "go")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
	assert.Equal(t, "devShell evaluates", result.Message)
}

func TestNixDevshellCheck_Failure(t *testing.T) {
	backend := newFakeBackend()
	backend.execErr = errors.New("nix evaluation failed")
	check := doctor.NewNixDevshellCheck(backend, "havn-user-myproject", "github:user/repo", "go")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Recommendation, "flake ref")
}

func TestNixDevshellCheck_Timeout60s(t *testing.T) {
	check := doctor.NewNixDevshellCheck(newFakeBackend(), "c", "ref", "shell")

	assert.Equal(t, 60*time.Second, check.Timeout())
}

func TestNixDevshellCheck_Metadata(t *testing.T) {
	check := doctor.NewNixDevshellCheck(newFakeBackend(), "havn-user-myproject", "ref", "shell")

	assert.Equal(t, "nix_devshell", check.ID())
	assert.Equal(t, "container", check.Tier())
	assert.Equal(t, "havn-user-myproject", check.Container())
	assert.Equal(t, []string{"docker_daemon"}, check.Prerequisites())
}

// --- project_mount check ---

func TestProjectMountCheck_Pass(t *testing.T) {
	backend := newFakeBackend()
	check := doctor.NewProjectMountCheck(backend, "havn-user-myproject", "/home/devuser/project")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
	assert.Equal(t, "Project directory writable", result.Message)
}

func TestProjectMountCheck_Failure(t *testing.T) {
	backend := newFakeBackend()
	backend.execErr = errors.New("test -w failed")
	check := doctor.NewProjectMountCheck(backend, "havn-user-myproject", "/home/devuser/project")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusError, result.Status)
	assert.Contains(t, result.Recommendation, "havn .")
}

func TestProjectMountCheck_Metadata(t *testing.T) {
	check := doctor.NewProjectMountCheck(newFakeBackend(), "havn-user-myproject", "/path")

	assert.Equal(t, "project_mount", check.ID())
	assert.Equal(t, "container", check.Tier())
	assert.Equal(t, "havn-user-myproject", check.Container())
	assert.Equal(t, []string{"docker_daemon"}, check.Prerequisites())
}

// --- config_mounts check ---

func TestConfigMountsCheck_AllPresent(t *testing.T) {
	backend := newFakeBackend()
	mounts := []string{"~/.gitconfig", "~/.ssh/config"}
	check := doctor.NewConfigMountsCheck(backend, "havn-user-myproject", mounts)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestConfigMountsCheck_SomeMissing(t *testing.T) {
	backend := newFakeBackend()
	backend.execErrors["~/.ssh/config"] = errors.New("not found")
	mounts := []string{"~/.gitconfig", "~/.ssh/config"}
	check := doctor.NewConfigMountsCheck(backend, "havn-user-myproject", mounts)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Detail, "~/.ssh/config")
}

func TestConfigMountsCheck_Empty(t *testing.T) {
	backend := newFakeBackend()
	check := doctor.NewConfigMountsCheck(backend, "havn-user-myproject", nil)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestConfigMountsCheck_Metadata(t *testing.T) {
	check := doctor.NewConfigMountsCheck(newFakeBackend(), "c", nil)

	assert.Equal(t, "config_mounts", check.ID())
	assert.Equal(t, "container", check.Tier())
	assert.Equal(t, []string{"docker_daemon"}, check.Prerequisites())
}

// --- ssh_agent check ---

func TestSSHAgentCheck_Pass(t *testing.T) {
	backend := newFakeBackend()
	check := doctor.NewSSHAgentCheck(backend, "havn-user-myproject")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestSSHAgentCheck_SocketMissing(t *testing.T) {
	backend := newFakeBackend()
	backend.execErrors["$SSH_AUTH_SOCK"] = errors.New("no socket")
	check := doctor.NewSSHAgentCheck(backend, "havn-user-myproject")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Recommendation, "ssh-agent")
}

func TestSSHAgentCheck_AddFails(t *testing.T) {
	backend := newFakeBackend()
	backend.execErrors["-l"] = errors.New("agent refused")
	check := doctor.NewSSHAgentCheck(backend, "havn-user-myproject")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
}

func TestSSHAgentCheck_Metadata(t *testing.T) {
	check := doctor.NewSSHAgentCheck(newFakeBackend(), "c")

	assert.Equal(t, "ssh_agent", check.ID())
	assert.Equal(t, "container", check.Tier())
	assert.Equal(t, []string{"docker_daemon"}, check.Prerequisites())
}

// --- dolt_connectivity check ---

func TestDoltConnectivityCheck_Pass(t *testing.T) {
	backend := newFakeBackend()
	backend.networks["havn-net"] = doctor.NetworkInfo{
		ContainerCount: 2,
		Containers:     []string{"havn-user-myproject", "havn-dolt"},
	}
	check := doctor.NewDoltConnectivityCheck(backend, "havn-user-myproject", "havn-net", true)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestDoltConnectivityCheck_NotOnNetwork(t *testing.T) {
	backend := newFakeBackend()
	backend.networks["havn-net"] = doctor.NetworkInfo{
		ContainerCount: 1,
		Containers:     []string{"havn-dolt"},
	}
	check := doctor.NewDoltConnectivityCheck(backend, "havn-user-myproject", "havn-net", true)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusError, result.Status)
	assert.Contains(t, result.Recommendation, "havn .")
}

func TestDoltConnectivityCheck_DoltDisabled(t *testing.T) {
	check := doctor.NewDoltConnectivityCheck(newFakeBackend(), "c", "havn-net", false)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusSkip, result.Status)
}

func TestDoltConnectivityCheck_Metadata(t *testing.T) {
	check := doctor.NewDoltConnectivityCheck(newFakeBackend(), "c", "havn-net", true)

	assert.Equal(t, "dolt_connectivity", check.ID())
	assert.Equal(t, "container", check.Tier())
	assert.Equal(t, []string{"docker_daemon"}, check.Prerequisites())
}

// --- beads_health check ---

func TestBeadsHealthCheck_Pass(t *testing.T) {
	backend := newFakeBackend()
	backend.execResults["bd"] = "/usr/bin/bd"
	backend.execResults["--json"] = `{"status":"pass"}`
	check := doctor.NewBeadsHealthCheck(backend, "havn-user-myproject", true)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestBeadsHealthCheck_NoBdInstalled(t *testing.T) {
	backend := newFakeBackend()
	backend.execErrors["bd"] = errors.New("not found")
	check := doctor.NewBeadsHealthCheck(backend, "havn-user-myproject", true)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Recommendation, "devShell")
}

func TestBeadsHealthCheck_DoctorFails(t *testing.T) {
	backend := newFakeBackend()
	backend.execResults["bd"] = "/usr/bin/bd"
	backend.execErrors["--json"] = errors.New("bd doctor failed")
	check := doctor.NewBeadsHealthCheck(backend, "havn-user-myproject", true)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Recommendation, "bd doctor")
}

func TestBeadsHealthCheck_NoBeadsDir(t *testing.T) {
	check := doctor.NewBeadsHealthCheck(newFakeBackend(), "c", false)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusSkip, result.Status)
}

func TestBeadsHealthCheck_Metadata(t *testing.T) {
	check := doctor.NewBeadsHealthCheck(newFakeBackend(), "c", true)

	assert.Equal(t, "beads_health", check.ID())
	assert.Equal(t, "container", check.Tier())
	assert.Equal(t, []string{"docker_daemon"}, check.Prerequisites())
}
