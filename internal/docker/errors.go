// Package docker wraps the Docker SDK behind havn-native types.
package docker

import "fmt"

// DaemonUnreachableError indicates the Docker daemon cannot be contacted.
type DaemonUnreachableError struct {
	Host string
}

func (e *DaemonUnreachableError) Error() string {
	return fmt.Sprintf("daemon unreachable at %q", e.Host)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *DaemonUnreachableError) ErrorType() string {
	return "daemon_unreachable"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *DaemonUnreachableError) ErrorDetails() map[string]any {
	return map[string]any{"host": e.Host}
}
