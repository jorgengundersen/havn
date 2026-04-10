package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/dolt"
)

// TypedError is implemented by domain errors that expose machine-readable
// type identifiers and structured details for JSON output.
type TypedError interface {
	ErrorType() string
	ErrorDetails() map[string]any
}

// ExitError wraps an error with a specific process exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// ExitCode extracts the exit code from an ExitError, defaulting to 1.
func ExitCode(err error) int {
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return 1
}

// FormatError translates an error into a user-facing message.
func FormatError(err error) string {
	var daemonErr *docker.DaemonUnreachableError
	if errors.As(err, &daemonErr) {
		return "Docker is not running. Start Docker and try again"
	}

	var containerNotFound *docker.ContainerNotFoundError
	if errors.As(err, &containerNotFound) {
		return fmt.Sprintf("Failed to find container %q", containerNotFound.Name)
	}

	var imageNotFound *docker.ImageNotFoundError
	if errors.As(err, &imageNotFound) {
		return fmt.Sprintf("Image %q not found — run 'havn build' first", imageNotFound.Name)
	}

	var parseErr *config.ParseError
	if errors.As(err, &parseErr) {
		return fmt.Sprintf("Config parse error at %s:%d: %s", parseErr.File, parseErr.Line, parseErr.Detail)
	}

	var valErr *config.ValidationError
	if errors.As(err, &valErr) {
		return fmt.Sprintf("Invalid config: %s: %s", valErr.Field, valErr.Reason)
	}

	var startErr *dolt.StartError
	if errors.As(err, &startErr) {
		return fmt.Sprintf("Failed to start Dolt server: %s. Run `havn doctor` to diagnose", startErr.Err)
	}

	var healthErr *dolt.HealthCheckTimeoutError
	if errors.As(err, &healthErr) {
		return "Dolt server started but not responding. Check `docker logs havn-dolt`"
	}

	var notManaged *dolt.NotManagedError
	if errors.As(err, &notManaged) {
		return notManaged.Error()
	}

	var dbCreateErr *dolt.DatabaseCreateError
	if errors.As(err, &dbCreateErr) {
		return fmt.Sprintf("Failed to create database '%s': %s", dbCreateErr.Name, dbCreateErr.Err)
	}

	return err.Error()
}

// Error writes an error to stderr, formatted based on the output mode.
// In JSON mode with a TypedError: {"error": "message", "type": "...", "details": {...}}.
// In JSON mode without TypedError: {"error": "message"}.
// In normal mode: "Error: message".
func (o *Output) Error(err error) {
	msg := FormatError(err)
	if o.json {
		var typed TypedError
		if errors.As(err, &typed) {
			payload := map[string]any{
				"error":   msg,
				"type":    typed.ErrorType(),
				"details": typed.ErrorDetails(),
			}
			_ = json.NewEncoder(o.stderr).Encode(payload)
			return
		}
		_ = json.NewEncoder(o.stderr).Encode(map[string]string{"error": msg})
		return
	}
	_, _ = fmt.Fprintf(o.stderr, "Error: %s\n", msg)
}
