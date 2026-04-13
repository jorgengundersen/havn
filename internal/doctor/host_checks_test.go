package doctor_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/doctor"
)

// fakeBackend is a test double for doctor.Backend.
type fakeBackend struct {
	pingErr        error
	info           doctor.RuntimeInfo
	infoErr        error
	images         map[string]doctor.ImageInfo
	imageErr       error
	networks       map[string]doctor.NetworkInfo
	networkErr     error
	volumes        map[string]bool
	volumeErr      error
	containers     map[string]doctor.ContainerInfo
	containerErr   error
	execResults    map[string]string
	execErrors     map[string]error
	execErr        error
	listContainers []string
	listErr        error
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		images:      make(map[string]doctor.ImageInfo),
		networks:    make(map[string]doctor.NetworkInfo),
		volumes:     make(map[string]bool),
		containers:  make(map[string]doctor.ContainerInfo),
		execResults: make(map[string]string),
		execErrors:  make(map[string]error),
	}
}

func (f *fakeBackend) Ping(_ context.Context) error { return f.pingErr }

func (f *fakeBackend) Info(_ context.Context) (doctor.RuntimeInfo, error) {
	return f.info, f.infoErr
}

func (f *fakeBackend) ImageInspect(_ context.Context, image string) (doctor.ImageInfo, bool, error) {
	if f.imageErr != nil {
		return doctor.ImageInfo{}, false, f.imageErr
	}
	info, ok := f.images[image]
	return info, ok, nil
}

func (f *fakeBackend) NetworkInspect(_ context.Context, network string) (doctor.NetworkInfo, bool, error) {
	if f.networkErr != nil {
		return doctor.NetworkInfo{}, false, f.networkErr
	}
	info, ok := f.networks[network]
	return info, ok, nil
}

func (f *fakeBackend) VolumeInspect(_ context.Context, volume string) (bool, error) {
	if f.volumeErr != nil {
		return false, f.volumeErr
	}
	return f.volumes[volume], nil
}

func (f *fakeBackend) ContainerInspect(_ context.Context, name string) (doctor.ContainerInfo, bool, error) {
	if f.containerErr != nil {
		return doctor.ContainerInfo{}, false, f.containerErr
	}
	info, ok := f.containers[name]
	return info, ok, nil
}

func (f *fakeBackend) ContainerExec(_ context.Context, _ string, cmd []string) (string, error) {
	if f.execErr != nil {
		return "", f.execErr
	}
	full := strings.Join(cmd, " ")
	if err, ok := f.execErrors[full]; ok {
		return "", err
	}
	key := cmd[len(cmd)-1]
	if err, ok := f.execErrors[key]; ok {
		return "", err
	}
	if result, ok := f.execResults[full]; ok {
		return result, nil
	}
	return f.execResults[key], nil
}

func (f *fakeBackend) ListContainers(_ context.Context, _ map[string]string) ([]string, error) {
	return f.listContainers, f.listErr
}

// --- docker_daemon check ---

func TestDockerDaemonCheck_Pass(t *testing.T) {
	backend := newFakeBackend()
	backend.info = doctor.RuntimeInfo{Version: "24.0.7", APIVersion: "1.43"}
	check := doctor.NewDockerDaemonCheck(backend)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
	assert.Equal(t, "Docker daemon running", result.Message)
}

func TestDockerDaemonCheck_Error(t *testing.T) {
	backend := newFakeBackend()
	backend.pingErr = errors.New("connection refused")
	check := doctor.NewDockerDaemonCheck(backend)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusError, result.Status)
	assert.Contains(t, result.Recommendation, "Start Docker")
}

func TestDockerDaemonCheck_Metadata(t *testing.T) {
	check := doctor.NewDockerDaemonCheck(newFakeBackend())

	assert.Equal(t, "docker_daemon", check.ID())
	assert.Equal(t, "host", check.Tier())
	assert.Empty(t, check.Prerequisites())
}

// --- base_image check ---

func TestBaseImageCheck_Pass(t *testing.T) {
	backend := newFakeBackend()
	backend.images["havn-base:latest"] = doctor.ImageInfo{ID: "sha256:abc", Created: "2026-03-20"}
	check := doctor.NewBaseImageCheck(backend, "havn-base:latest")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
	assert.Equal(t, "Base image exists", result.Message)
}

func TestBaseImageCheck_Missing(t *testing.T) {
	backend := newFakeBackend()
	check := doctor.NewBaseImageCheck(backend, "havn-base:latest")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Recommendation, "havn build")
}

func TestBaseImageCheck_Prerequisites(t *testing.T) {
	check := doctor.NewBaseImageCheck(newFakeBackend(), "img")
	assert.Equal(t, []string{"docker_daemon"}, check.Prerequisites())
}

// --- network check ---

func TestNetworkCheck_Pass(t *testing.T) {
	backend := newFakeBackend()
	backend.networks["havn-net"] = doctor.NetworkInfo{ContainerCount: 3}
	check := doctor.NewNetworkCheck(backend, "havn-net")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
	assert.Equal(t, "Network exists", result.Message)
}

func TestNetworkCheck_Missing(t *testing.T) {
	backend := newFakeBackend()
	check := doctor.NewNetworkCheck(backend, "havn-net")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Recommendation, "auto-created")
}

// --- volumes check ---

func TestVolumesCheck_AllPresent(t *testing.T) {
	backend := newFakeBackend()
	volumes := []string{"havn-nix", "havn-data", "havn-cache", "havn-state"}
	for _, v := range volumes {
		backend.volumes[v] = true
	}
	check := doctor.NewVolumesCheck(backend, volumes)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestVolumesCheck_SomeMissing(t *testing.T) {
	backend := newFakeBackend()
	backend.volumes["havn-nix"] = true
	backend.volumes["havn-data"] = true
	volumes := []string{"havn-nix", "havn-data", "havn-cache", "havn-state"}
	check := doctor.NewVolumesCheck(backend, volumes)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Detail, "havn-cache")
	assert.Contains(t, result.Detail, "havn-state")
}

// --- global_config check ---

func TestGlobalConfigCheck_Missing(t *testing.T) {
	check := doctor.NewGlobalConfigCheck("/nonexistent/path/config.toml")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Message, "No global config found")
}

func TestGlobalConfigCheck_Valid(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.toml"
	writeTestFile(t, path, `shell = "default"`)

	check := doctor.NewGlobalConfigCheck(path)
	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestGlobalConfigCheck_ParseError(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.toml"
	writeTestFile(t, path, `[invalid`)

	check := doctor.NewGlobalConfigCheck(path)
	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusError, result.Status)
}

func TestGlobalConfigCheck_NoPrerequisites(t *testing.T) {
	check := doctor.NewGlobalConfigCheck("/any")
	assert.Empty(t, check.Prerequisites())
}

// --- project_config check ---

func TestProjectConfigCheck_Missing(t *testing.T) {
	check := doctor.NewProjectConfigCheck("/nonexistent/.havn/config.toml", nil)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestProjectConfigCheck_Valid(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.toml"
	writeTestFile(t, path, `shell = "go"`)

	check := doctor.NewProjectConfigCheck(path, nil)
	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestProjectConfigCheck_ParseError(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.toml"
	writeTestFile(t, path, `[bad toml`)

	check := doctor.NewProjectConfigCheck(path, nil)
	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusError, result.Status)
}

func TestProjectConfigCheck_ValidationError(t *testing.T) {
	check := doctor.NewProjectConfigCheck("/tmp/project/.havn/config.toml", errors.New("invalid config: resources.cpus: must be greater than 0"))

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusError, result.Status)
	assert.Equal(t, "Project config validation error", result.Message)
	assert.Contains(t, result.Detail, "resources.cpus")
}

// --- dolt_server check ---

func TestDoltServerCheck_NotEnabled(t *testing.T) {
	check := doctor.NewDoltServerCheck(newFakeBackend(), false)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusSkip, result.Status)
	assert.Contains(t, result.Message, "Dolt not enabled")
}

func TestDoltServerCheck_Running(t *testing.T) {
	backend := newFakeBackend()
	backend.containers["havn-dolt"] = doctor.ContainerInfo{
		Running: true,
		Labels:  map[string]string{"managed-by": "havn"},
	}
	backend.execResults["SELECT 1"] = "1"
	check := doctor.NewDoltServerCheck(backend, true)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestDoltServerCheck_NotRunning(t *testing.T) {
	backend := newFakeBackend()
	check := doctor.NewDoltServerCheck(backend, true)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusError, result.Status)
	assert.Contains(t, result.Recommendation, "havn dolt start")
}

func TestDoltServerCheck_MissingLabel(t *testing.T) {
	backend := newFakeBackend()
	backend.containers["havn-dolt"] = doctor.ContainerInfo{
		Running: true,
		Labels:  map[string]string{},
	}
	check := doctor.NewDoltServerCheck(backend, true)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Message, "not created by havn")
}

func TestDoltServerCheck_Unresponsive(t *testing.T) {
	backend := newFakeBackend()
	backend.containers["havn-dolt"] = doctor.ContainerInfo{
		Running: true,
		Labels:  map[string]string{"managed-by": "havn"},
	}
	backend.execErr = errors.New("connection refused")
	check := doctor.NewDoltServerCheck(backend, true)

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusError, result.Status)
	assert.Contains(t, result.Recommendation, "docker logs")
}

// --- dolt_database check ---

func TestDoltDatabaseCheck_NotEnabled(t *testing.T) {
	check := doctor.NewDoltDatabaseCheck(newFakeBackend(), false, "mydb")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusSkip, result.Status)
}

func TestDoltDatabaseCheck_Exists(t *testing.T) {
	backend := newFakeBackend()
	backend.execResults["SHOW DATABASES"] = "information_schema\nmydb\n"
	check := doctor.NewDoltDatabaseCheck(backend, true, "mydb")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, result.Status)
}

func TestDoltDatabaseCheck_Missing(t *testing.T) {
	backend := newFakeBackend()
	backend.execResults["SHOW DATABASES"] = "information_schema\n"
	check := doctor.NewDoltDatabaseCheck(backend, true, "mydb")

	result := check.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Recommendation, "mydb")
}

func TestDoltDatabaseCheck_Prerequisites(t *testing.T) {
	check := doctor.NewDoltDatabaseCheck(newFakeBackend(), true, "db")
	assert.Equal(t, []string{"dolt_server"}, check.Prerequisites())
}

// --- helper ---

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}
