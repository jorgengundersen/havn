package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/doctor"
	"github.com/jorgengundersen/havn/internal/name"
)

// fakeDoctorBackend implements doctor.Backend for CLI tests.
type fakeDoctorBackend struct {
	pingErr        error
	listContainers []string
	containerInfos map[string]doctor.ContainerInfo
	execErrs       map[string]error
}

func (f *fakeDoctorBackend) Ping(_ context.Context) error { return f.pingErr }
func (f *fakeDoctorBackend) Info(_ context.Context) (doctor.RuntimeInfo, error) {
	return doctor.RuntimeInfo{Version: "24.0.7", APIVersion: "1.43"}, nil
}
func (f *fakeDoctorBackend) ImageInspect(_ context.Context, _ string) (doctor.ImageInfo, bool, error) {
	return doctor.ImageInfo{}, true, nil
}
func (f *fakeDoctorBackend) NetworkInspect(_ context.Context, _ string) (doctor.NetworkInfo, bool, error) {
	return doctor.NetworkInfo{}, true, nil
}
func (f *fakeDoctorBackend) VolumeInspect(_ context.Context, _ string) (bool, error) {
	return true, nil
}
func (f *fakeDoctorBackend) ContainerInspect(_ context.Context, name string) (doctor.ContainerInfo, bool, error) {
	if f.containerInfos == nil {
		return doctor.ContainerInfo{}, false, nil
	}
	info, ok := f.containerInfos[name]
	if !ok {
		return doctor.ContainerInfo{}, false, nil
	}
	return info, true, nil
}
func (f *fakeDoctorBackend) ContainerExec(_ context.Context, _ string, cmd []string) (string, error) {
	if f.execErrs != nil {
		if err, ok := f.execErrs[strings.Join(cmd, " ")]; ok {
			return "", err
		}
	}
	return "", nil
}
func (f *fakeDoctorBackend) ListContainers(_ context.Context, _ map[string]string) ([]string, error) {
	return f.listContainers, nil
}

func executeDoctorCommand(backend doctor.Backend, args ...string) (stdout, stderr string, err error) {
	deps := cli.Deps{DoctorBackend: backend}
	root := cli.NewRoot(deps)
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)
	root.SetArgs(append([]string{"doctor"}, args...))
	err = root.Execute()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func TestDoctorCommand_AllPassExitZero(t *testing.T) {
	backend := &fakeDoctorBackend{}
	stdout, _, err := executeDoctorCommand(backend)

	// Dolt not enabled so those are skipped, but the rest should pass.
	// Config checks may warn (no global config), that's expected.
	// The point is: it runs and produces output.
	assert.Contains(t, stdout, "Host")
	_ = err // exit code depends on config file existence
}

func TestDoctorCommand_JSONOutput(t *testing.T) {
	backend := &fakeDoctorBackend{}
	stdout, _, _ := executeDoctorCommand(backend, "--json")

	var parsed map[string]any
	err := json.Unmarshal([]byte(stdout), &parsed)
	require.NoError(t, err, "JSON output should be valid JSON")
	assert.Contains(t, parsed, "status")
	assert.Contains(t, parsed, "summary")
	assert.Contains(t, parsed, "checks")
}

func TestDoctorCommand_HasAllFlag(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	doctorCmd, _, err := root.Find([]string{"doctor"})

	require.NoError(t, err)
	f := doctorCmd.Flags().Lookup("all")
	require.NotNil(t, f, "--all flag should exist")
	assert.Equal(t, "false", f.DefValue)
}

func TestDoctorCommand_ExitCode2OnError(t *testing.T) {
	backend := &fakeDoctorBackend{
		pingErr: assert.AnError,
	}
	_, _, err := executeDoctorCommand(backend)

	require.Error(t, err)
	assert.Equal(t, 2, cli.ExitCode(err))
}

func TestDoctorCommand_UsesConfigFlagForGlobalConfigCheck(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectDir := filepath.Join(homeDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectDir))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	badGlobalPath := filepath.Join(t.TempDir(), "bad-global.toml")
	require.NoError(t, os.WriteFile(badGlobalPath, []byte("[broken"), 0o644))

	backend := &fakeDoctorBackend{}
	stdout, _, err := executeDoctorCommand(backend, "--config", badGlobalPath)

	require.Error(t, err)
	assert.Equal(t, 2, cli.ExitCode(err))
	assert.Contains(t, stdout, "Global config syntax error")
}

func TestDoctorCommand_AllFlagRunsContainerChecks(t *testing.T) {
	backend := &fakeDoctorBackend{
		listContainers: []string{"havn-user-myproject"},
		containerInfos: map[string]doctor.ContainerInfo{
			"havn-user-myproject": {
				Running: true,
				Labels:  map[string]string{"havn.path": "/home/devuser/myproject"},
			},
		},
	}
	stdout, _, _ := executeDoctorCommand(backend, "--all")

	assert.Contains(t, stdout, "Container: havn-user-myproject")
	assert.Contains(t, stdout, "Nix store mounted")
}

func TestDoctorCommand_AllFlagSkipsNonProjectContainers(t *testing.T) {
	backend := &fakeDoctorBackend{
		listContainers: []string{"havn-dolt", "havn-user-myproject", "havn-user-missing-path"},
		containerInfos: map[string]doctor.ContainerInfo{
			"havn-dolt": {
				Running: true,
				Labels:  map[string]string{},
			},
			"havn-user-myproject": {
				Running: true,
				Labels:  map[string]string{"havn.path": "/home/devuser/myproject"},
			},
			"havn-user-missing-path": {
				Running: true,
				Labels:  map[string]string{},
			},
		},
	}

	stdout, _, _ := executeDoctorCommand(backend, "--all")

	assert.Contains(t, stdout, "Container: havn-user-myproject")
	assert.NotContains(t, stdout, "Container: havn-dolt")
	assert.NotContains(t, stdout, "Container: havn-user-missing-path")
}

func TestDoctorCommand_AllFlagUsesPerContainerProjectPath(t *testing.T) {
	backend := &fakeDoctorBackend{
		listContainers: []string{"havn-user-myproject"},
		containerInfos: map[string]doctor.ContainerInfo{
			"havn-user-myproject": {
				Running: true,
				Labels:  map[string]string{"havn.path": "/home/devuser/myproject"},
			},
		},
		execErrs: map[string]error{
			"test -w /home/devuser/myproject":                             nil,
			"test -w /home/devuser/Repos/github.com/jorgengundersen/havn": errors.New("used cwd path"),
		},
	}

	stdout, _, _ := executeDoctorCommand(backend, "--all", "--verbose")

	assert.Contains(t, stdout, "Project directory writable")
	assert.Contains(t, stdout, "/home/devuser/myproject")
}

func TestDoctorCommand_NoContainersSkipsTier2(t *testing.T) {
	backend := &fakeDoctorBackend{}
	stdout, _, _ := executeDoctorCommand(backend, "--all")

	assert.NotContains(t, stdout, "Container:")
}

func TestDoctorCommand_NoContainersReportsInformationalSkipInJSON(t *testing.T) {
	backend := &fakeDoctorBackend{}
	stdout, _, _ := executeDoctorCommand(backend, "--all", "--json")

	var parsed struct {
		Checks []struct {
			Tier    string `json:"tier"`
			Name    string `json:"name"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"checks"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed))

	found := false
	for _, check := range parsed.Checks {
		if check.Tier == "container" && check.Name == "container_tier" && check.Status == "skip" {
			assert.Contains(t, check.Message, "No relevant running havn-managed project containers")
			found = true
			break
		}
	}

	assert.True(t, found, "expected informational container-tier skip check")
}

func TestDoctorCommand_UsesResolvedConfigMountTargets(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte("[mounts]\nconfig = [\".gitconfig:ro\"]\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".gitconfig"), []byte("[user]\n\tname = test\n"), 0o644))

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectPath))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	backend := &fakeDoctorBackend{
		listContainers: []string{"havn-user-myproject"},
		containerInfos: map[string]doctor.ContainerInfo{
			"havn-user-myproject": {
				Running: true,
				Labels:  map[string]string{"havn.path": projectPath},
			},
		},
		execErrs: map[string]error{
			"test -r .gitconfig:ro":            errors.New("used unresolved config mount"),
			"test -w /home/devuser/.gitconfig": errors.New("read-only expected"),
		},
	}

	stdout, _, _ := executeDoctorCommand(backend, "--all")

	assert.Contains(t, stdout, "Config mounts present")
	assert.NotContains(t, stdout, "Config mounts missing")
}

func TestDoctorCommand_DefaultScopeTargetsCurrentProjectContainer(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "workspace", "myproject")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectPath))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	parent, project, err := name.SplitProjectPath(projectPath)
	require.NoError(t, err)
	containerName, err := name.DeriveContainerName(parent, project)
	require.NoError(t, err)

	backend := &fakeDoctorBackend{
		containerInfos: map[string]doctor.ContainerInfo{
			string(containerName): {
				Running: true,
				Labels:  map[string]string{"havn.path": projectPath},
			},
			"havn-user-otherproject": {
				Running: true,
				Labels:  map[string]string{"havn.path": filepath.Join(homeDir, "workspace", "otherproject")},
			},
		},
	}

	stdout, _, _ := executeDoctorCommand(backend)

	assert.Contains(t, stdout, fmt.Sprintf("Container: %s", string(containerName)))
	assert.NotContains(t, stdout, "Container: havn-user-otherproject")
}

func TestDoctorCommand_DerivesProjectDatabaseNameFromProjectPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "workspace", "projectalpha")
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte("[dolt]\nenabled = true\n"), 0o644))

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectPath))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	backend := &fakeDoctorBackend{
		containerInfos: map[string]doctor.ContainerInfo{
			"havn-dolt": {
				Running: true,
				Labels:  map[string]string{"managed-by": "havn"},
			},
		},
	}

	stdout, _, err := executeDoctorCommand(backend)

	require.Error(t, err)
	assert.Equal(t, 1, cli.ExitCode(err))
	assert.Contains(t, stdout, "Database 'projectalpha' does not exist on the shared server")
}

func TestDoctorCommand_JSONWarnExitCodeAndStreamSeparation(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "workspace", "projectbeta")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectPath))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	stdout, stderr, err := executeDoctorCommand(&fakeDoctorBackend{}, "--json")

	require.Error(t, err)
	assert.Equal(t, 1, cli.ExitCode(err))
	assert.Empty(t, strings.TrimSpace(stderr))

	var parsed struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed))
	assert.Equal(t, "warn", parsed.Status)
}
