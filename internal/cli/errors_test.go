package cli_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/dolt"
	"github.com/jorgengundersen/havn/internal/volume"
)

func TestExitError_ExposesCodeAndWrapsErr(t *testing.T) {
	inner := errors.New("boom")
	exitErr := &cli.ExitError{Code: 42, Err: inner}

	assert.Equal(t, 42, exitErr.Code)
	assert.Equal(t, "boom", exitErr.Error())
	assert.ErrorIs(t, exitErr, inner)
}

func TestExitCode_ReturnsCodeFromExitError(t *testing.T) {
	err := &cli.ExitError{Code: 3, Err: errors.New("fail")}

	assert.Equal(t, 3, cli.ExitCode(err))
}

func TestExitCode_DefaultsTo1ForPlainError(t *testing.T) {
	err := errors.New("plain error")

	assert.Equal(t, 1, cli.ExitCode(err))
}

func TestFormatError_ReturnsErrorMessage(t *testing.T) {
	err := errors.New("something went wrong")

	assert.Equal(t, "something went wrong", cli.FormatError(err))
}

func TestFormatError_StartError(t *testing.T) {
	err := &dolt.StartError{Err: errors.New("connection refused")}

	assert.Equal(t, "Failed to start Dolt server: connection refused. Run `havn doctor` to diagnose", cli.FormatError(err))
}

func TestFormatError_HealthCheckTimeoutError(t *testing.T) {
	err := &dolt.HealthCheckTimeoutError{Timeout: 30 * time.Second}

	assert.Equal(t, "Dolt server started but not responding. Check `docker logs havn-dolt`", cli.FormatError(err))
}

func TestFormatError_NotManagedError(t *testing.T) {
	err := &dolt.NotManagedError{Name: "havn-dolt"}

	assert.Equal(t, `container "havn-dolt" exists but was not created by havn`, cli.FormatError(err))
}

func TestFormatError_ServerNotRunningError(t *testing.T) {
	err := &dolt.ServerNotRunningError{Name: "havn-dolt"}

	assert.Equal(t, "Dolt server is not running. Run `havn dolt start`", cli.FormatError(err))
}

func TestFormatError_DatabaseCreateError(t *testing.T) {
	err := &dolt.DatabaseCreateError{Name: "myproject", Err: errors.New("access denied")}

	assert.Equal(t, "Failed to create database 'myproject': access denied", cli.FormatError(err))
}

func TestTypedError_ParseErrorSatisfiesInterface(t *testing.T) {
	var typed cli.TypedError = &config.ParseError{File: "f", Line: 1, Detail: "d"}

	assert.Equal(t, "config_parse_error", typed.ErrorType())
	assert.Equal(t, map[string]any{"file": "f", "line": 1, "detail": "d"}, typed.ErrorDetails())
}

func TestTypedError_ValidationErrorSatisfiesInterface(t *testing.T) {
	var typed cli.TypedError = &config.ValidationError{Field: "f", Reason: "r"}

	assert.Equal(t, "config_validation_error", typed.ErrorType())
	assert.Equal(t, map[string]any{"field": "f", "reason": "r"}, typed.ErrorDetails())
}

func TestFormatError_ParseError(t *testing.T) {
	err := &config.ParseError{File: "config.toml", Line: 5, Detail: "unexpected key"}

	assert.Equal(t, "Config parse error at config.toml:5: unexpected key", cli.FormatError(err))
}

func TestFormatError_ValidationError(t *testing.T) {
	err := &config.ValidationError{Field: "resources.cpus", Reason: "must be greater than 0"}

	assert.Equal(t, "Invalid config: resources.cpus: must be greater than 0", cli.FormatError(err))
}

func TestFormatError_DaemonUnreachableError(t *testing.T) {
	err := &docker.DaemonUnreachableError{Host: "unix:///var/run/docker.sock"}

	assert.Equal(t, "Docker is not running. Start Docker and try again", cli.FormatError(err))
}

func TestFormatError_ContainerNotFoundError(t *testing.T) {
	err := &container.NotFoundError{Name: "havn-user-api"}

	assert.Equal(t, `Failed to find container "havn-user-api"`, cli.FormatError(err))
}

func TestFormatError_ImageNotFoundError(t *testing.T) {
	err := &container.ImageNotFoundError{Name: "havn-base:latest"}

	assert.Equal(t, `Image "havn-base:latest" not found — run 'havn build' first`, cli.FormatError(err))
}

func TestTypedError_ContainerNotFoundErrorSatisfiesInterface(t *testing.T) {
	var typed cli.TypedError = &docker.ContainerNotFoundError{Name: "havn-user-api"}

	assert.Equal(t, "container_not_found", typed.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-user-api"}, typed.ErrorDetails())
}

func TestTypedError_ImageNotFoundErrorSatisfiesInterface(t *testing.T) {
	var typed cli.TypedError = &docker.ImageNotFoundError{Name: "havn-base:latest"}

	assert.Equal(t, "image_not_found", typed.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-base:latest"}, typed.ErrorDetails())
}

func TestFormatError_NetworkNotFoundError(t *testing.T) {
	err := &container.NetworkNotFoundError{Name: "havn-net"}

	assert.Equal(t, `Network "havn-net" not found`, cli.FormatError(err))
}

func TestFormatError_VolumeNotFoundError(t *testing.T) {
	err := &volume.NotFoundError{Name: "havn-dolt-data"}

	assert.Equal(t, `Volume "havn-dolt-data" not found`, cli.FormatError(err))
}

func TestTypedError_VolumeNotFoundErrorSatisfiesInterface(t *testing.T) {
	var typed cli.TypedError = &docker.VolumeNotFoundError{Name: "havn-dolt-data"}

	assert.Equal(t, "volume_not_found", typed.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-dolt-data"}, typed.ErrorDetails())
}

func TestTypedError_NetworkNotFoundErrorSatisfiesInterface(t *testing.T) {
	var typed cli.TypedError = &docker.NetworkNotFoundError{Name: "havn-net"}

	assert.Equal(t, "network_not_found", typed.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-net"}, typed.ErrorDetails())
}

func TestTypedError_DaemonUnreachableErrorSatisfiesInterface(t *testing.T) {
	var typed cli.TypedError = &docker.DaemonUnreachableError{Host: "unix:///var/run/docker.sock"}

	assert.Equal(t, "daemon_unreachable", typed.ErrorType())
	assert.Equal(t, map[string]any{"host": "unix:///var/run/docker.sock"}, typed.ErrorDetails())
}

func TestOutput_Error_JSONMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, true, false)

	out.Error(errors.New("something broke"))

	assert.Empty(t, stdout.String(), "error should not write to stdout")
	assert.JSONEq(t, `{"error": "something broke"}`, stderr.String())
}

func TestOutput_Error_JSONMode_TypedError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, true, false)

	err := &config.ParseError{File: "config.toml", Line: 5, Detail: "unexpected key"}
	out.Error(err)

	assert.Empty(t, stdout.String(), "error should not write to stdout")
	assert.JSONEq(t, `{
		"error": "Config parse error at config.toml:5: unexpected key",
		"type": "config_parse_error",
		"details": {"file": "config.toml", "line": 5, "detail": "unexpected key"}
	}`, stderr.String())
}

func TestOutput_Error_JSONMode_NetworkNotFoundError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, true, false)

	err := &container.NetworkNotFoundError{Name: "havn-net"}
	out.Error(err)

	assert.Empty(t, stdout.String(), "error should not write to stdout")
	assert.JSONEq(t, `{
		"error": "Network \"havn-net\" not found",
		"type": "network_not_found",
		"details": {"name": "havn-net"}
	}`, stderr.String())
}

func TestOutput_Error_JSONMode_VolumeNotFoundError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, true, false)

	err := &volume.NotFoundError{Name: "havn-dolt-data"}
	out.Error(err)

	assert.Empty(t, stdout.String(), "error should not write to stdout")
	assert.JSONEq(t, `{
		"error": "Volume \"havn-dolt-data\" not found",
		"type": "volume_not_found",
		"details": {"name": "havn-dolt-data"}
	}`, stderr.String())
}

func TestOutput_Error_NormalMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, false, false)

	out.Error(errors.New("disk full"))

	assert.Empty(t, stdout.String(), "error should not write to stdout")
	assert.Equal(t, "Error: disk full\n", stderr.String())
}

func TestOutput_Error_NormalMode_TypedError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, false, false)

	err := &config.ValidationError{Field: "resources.cpus", Reason: "must be positive"}
	out.Error(err)

	assert.Empty(t, stdout.String(), "error should not write to stdout")
	assert.Equal(t, "Error: Invalid config: resources.cpus: must be positive\n", stderr.String())
}
