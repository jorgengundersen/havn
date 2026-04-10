package dolt

import (
	"fmt"
	"time"
)

// StartError wraps a failure to start the Dolt server container.
type StartError struct {
	Err error
}

func (e *StartError) Error() string {
	return fmt.Sprintf("start dolt server: %s", e.Err)
}

func (e *StartError) Unwrap() error {
	return e.Err
}

// HealthCheckTimeoutError indicates the Dolt server did not become healthy in time.
type HealthCheckTimeoutError struct {
	Timeout time.Duration
}

func (e *HealthCheckTimeoutError) Error() string {
	return fmt.Sprintf("dolt health check timed out after %s", e.Timeout)
}

// NotManagedError indicates a container exists but lacks the managed-by=havn label.
type NotManagedError struct {
	Name string
}

func (e *NotManagedError) Error() string {
	return fmt.Sprintf("container %q exists but was not created by havn", e.Name)
}
