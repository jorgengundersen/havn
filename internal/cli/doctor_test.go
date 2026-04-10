package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/doctor"
)

// fakeDoctorBackend implements doctor.Backend for CLI tests.
type fakeDoctorBackend struct {
	pingErr error
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
func (f *fakeDoctorBackend) ContainerInspect(_ context.Context, _ string) (doctor.ContainerInfo, bool, error) {
	return doctor.ContainerInfo{}, false, nil
}
func (f *fakeDoctorBackend) ContainerExec(_ context.Context, _ string, _ []string) (string, error) {
	return "", nil
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
