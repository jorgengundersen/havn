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

// DatabaseExistsError indicates a database already exists on the shared server.
type DatabaseExistsError struct {
	Name string
}

func (e *DatabaseExistsError) Error() string {
	return fmt.Sprintf("database %q already exists; use --force to overwrite", e.Name)
}

// DatabaseNotFoundError indicates a database was not found.
type DatabaseNotFoundError struct {
	Name string
}

func (e *DatabaseNotFoundError) Error() string {
	return fmt.Sprintf("no existing beads database found for %q", e.Name)
}

// ImportError wraps a failure during database import.
type ImportError struct {
	Err error
}

func (e *ImportError) Error() string {
	return fmt.Sprintf("import database: %s", e.Err)
}

func (e *ImportError) Unwrap() error {
	return e.Err
}

// DatabaseCreateError wraps a failure to create a project database on the
// shared Dolt server.
type DatabaseCreateError struct {
	Name string
	Err  error
}

func (e *DatabaseCreateError) Error() string {
	return fmt.Sprintf("create database %q: %s", e.Name, e.Err)
}

func (e *DatabaseCreateError) Unwrap() error {
	return e.Err
}

// ExportError wraps a failure during database export.
type ExportError struct {
	Err error
}

func (e *ExportError) Error() string {
	return fmt.Sprintf("export database: %s", e.Err)
}

func (e *ExportError) Unwrap() error {
	return e.Err
}
