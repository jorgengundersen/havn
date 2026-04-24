package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/dolt"
	"github.com/jorgengundersen/havn/internal/volume"
)

// TypedError is implemented by domain errors that expose machine-readable
// type identifiers and structured details for JSON output.
type TypedError interface {
	ErrorType() string
	ErrorDetails() map[string]any
}

// ExitError wraps an error with a specific process exit code.
type ExitError struct {
	Code           int
	Err            error
	SuppressOutput bool
}

// ShellExitError carries the interactive shell exit code from the root command.
type ShellExitError struct {
	Code int
}

func (e *ShellExitError) Error() string {
	return fmt.Sprintf("shell exited with code %d", e.Code)
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

	var containerNotFound *container.NotFoundError
	if errors.As(err, &containerNotFound) {
		return fmt.Sprintf("Failed to find container %q", containerNotFound.Name)
	}

	var imageNotFound *container.ImageNotFoundError
	if errors.As(err, &imageNotFound) {
		return fmt.Sprintf("Image %q not found — run 'havn build' first", imageNotFound.Name)
	}

	var networkNotFound *container.NetworkNotFoundError
	if errors.As(err, &networkNotFound) {
		return fmt.Sprintf("Network %q not found", networkNotFound.Name)
	}

	var volumeNotFound *volume.NotFoundError
	if errors.As(err, &volumeNotFound) {
		return fmt.Sprintf("Volume %q not found", volumeNotFound.Name)
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
		return formatDoltStartError(startErr)
	}

	var healthErr *dolt.HealthCheckTimeoutError
	if errors.As(err, &healthErr) {
		return fmt.Sprintf("Dolt server started but did not become ready within %s. Check `docker logs havn-dolt`, verify shared-network connectivity, then retry `havn dolt start`", healthErr.Timeout)
	}

	var notManaged *dolt.NotManagedError
	if errors.As(err, &notManaged) {
		return fmt.Sprintf("Dolt container %q exists but is not managed by havn. Resolve the name conflict (stop/remove or rename the existing container), then retry the command", notManaged.Name)
	}

	var notRunning *dolt.ServerNotRunningError
	if errors.As(err, &notRunning) {
		return "Dolt server is not running. Start it with `havn dolt start`, then retry the command"
	}

	var dbCreateErr *dolt.DatabaseCreateError
	if errors.As(err, &dbCreateErr) {
		if isConnectivityFailure(dbCreateErr.Err.Error()) {
			return fmt.Sprintf("Failed to create database '%s': shared Dolt connectivity failed (%s). Ensure `havn dolt status` reports running, then retry the command", dbCreateErr.Name, dbCreateErr.Err)
		}
		return fmt.Sprintf("Failed to create database '%s': %s", dbCreateErr.Name, dbCreateErr.Err)
	}

	return err.Error()
}

func formatDoltStartError(startErr *dolt.StartError) string {
	message := startErr.Err.Error()

	if image, detail, ok := parseDoltImagePullFailure(message); ok {
		if isRegistryAuthFailure(detail) {
			return fmt.Sprintf("Failed to start Dolt server: unable to pull image %q due to registry authentication failure. Run `docker login` for the registry, then retry `havn dolt start`", image)
		}
		if isNetworkPullFailure(detail) {
			return fmt.Sprintf("Failed to start Dolt server: unable to pull image %q due to a network or registry connectivity failure. Check registry connectivity or pre-seed the image with `docker pull`/`docker load`, then retry `havn dolt start`", image)
		}
		if isImageReferenceFailure(detail) {
			return fmt.Sprintf("Failed to start Dolt server: image %q could not be pulled because the reference was not found. Verify `dolt.image` and registry access, then retry `havn dolt start`", image)
		}

		return fmt.Sprintf("Failed to start Dolt server: unable to pull image %q (%s). Check registry access and retry `havn dolt start`", image, detail)
	}

	if strings.Contains(message, "create container:") {
		return "Failed to start Dolt server: container creation failed after image acquisition. Check Docker daemon health, shared network/volume availability, and retry `havn dolt start`"
	}

	if strings.Contains(message, "start container:") {
		return "Failed to start Dolt server: container start failed after image acquisition. Inspect `docker logs havn-dolt`, then retry `havn dolt start`"
	}

	if isConnectivityFailure(message) {
		return fmt.Sprintf("Failed to start Dolt server: Docker connectivity failed (%s). Ensure Docker is running and reachable, then retry `havn dolt start`", message)
	}

	return fmt.Sprintf("Failed to start Dolt server: %s. Retry `havn dolt start` after addressing the reported failure", message)
}

func parseDoltImagePullFailure(message string) (image, detail string, ok bool) {
	prefix := "pull image \""
	if !strings.HasPrefix(message, prefix) {
		return "", "", false
	}

	rest := strings.TrimPrefix(message, prefix)
	idx := strings.Index(rest, "\": ")
	if idx == -1 {
		return "", "", false
	}

	return rest[:idx], strings.TrimSpace(rest[idx+3:]), true
}

func isRegistryAuthFailure(detail string) bool {
	lower := strings.ToLower(detail)
	return strings.Contains(lower, "unauthorized") || strings.Contains(lower, "authentication required") || strings.Contains(lower, "denied")
}

func isNetworkPullFailure(detail string) bool {
	lower := strings.ToLower(detail)
	return strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "dial tcp") ||
		strings.Contains(lower, "tls handshake timeout") ||
		strings.Contains(lower, "temporary failure in name resolution")
}

func isConnectivityFailure(detail string) bool {
	lower := strings.ToLower(detail)
	return strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "dial tcp") ||
		strings.Contains(lower, "tls handshake timeout") ||
		strings.Contains(lower, "temporary failure in name resolution") ||
		strings.Contains(lower, "cannot connect to the docker daemon")
}

func isImageReferenceFailure(detail string) bool {
	lower := strings.ToLower(detail)
	return strings.Contains(lower, "manifest unknown") || strings.Contains(lower, "not found") || strings.Contains(lower, "name unknown")
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
