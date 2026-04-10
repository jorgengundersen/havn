package cli_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/dolt"
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

func TestFormatError_DatabaseCreateError(t *testing.T) {
	err := &dolt.DatabaseCreateError{Name: "myproject", Err: errors.New("access denied")}

	assert.Equal(t, "Failed to create database 'myproject': access denied", cli.FormatError(err))
}

func TestOutput_Error_JSONMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, true, false)

	out.Error(errors.New("something broke"))

	assert.Empty(t, stdout.String(), "error should not write to stdout")
	assert.JSONEq(t, `{"error": "something broke"}`, stderr.String())
}

func TestOutput_Error_NormalMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr, false, false)

	out.Error(errors.New("disk full"))

	assert.Empty(t, stdout.String(), "error should not write to stdout")
	assert.Equal(t, "Error: disk full\n", stderr.String())
}
