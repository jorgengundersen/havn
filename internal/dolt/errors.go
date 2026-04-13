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

// ErrorType returns the stable snake_case identifier for this error.
func (e *StartError) ErrorType() string {
	return "dolt_start_failed"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *StartError) ErrorDetails() map[string]any {
	return map[string]any{"error": e.Err.Error()}
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

// ErrorType returns the stable snake_case identifier for this error.
func (e *HealthCheckTimeoutError) ErrorType() string {
	return "dolt_health_check_timeout"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *HealthCheckTimeoutError) ErrorDetails() map[string]any {
	return map[string]any{"timeout": e.Timeout.String()}
}

// NotManagedError indicates a container exists but lacks the managed-by=havn label.
type NotManagedError struct {
	Name string
}

func (e *NotManagedError) Error() string {
	return fmt.Sprintf("container %q exists but was not created by havn", e.Name)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *NotManagedError) ErrorType() string {
	return "dolt_not_managed"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *NotManagedError) ErrorDetails() map[string]any {
	return map[string]any{"name": e.Name}
}

// ServerNotRunningError indicates the shared Dolt server container is absent
// or not running for commands that require a live server.
type ServerNotRunningError struct {
	Name string
}

func (e *ServerNotRunningError) Error() string {
	return fmt.Sprintf("container %q is not running", e.Name)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *ServerNotRunningError) ErrorType() string {
	return "dolt_server_not_running"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *ServerNotRunningError) ErrorDetails() map[string]any {
	return map[string]any{"name": e.Name}
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

// ErrorType returns the stable snake_case identifier for this error.
func (e *DatabaseCreateError) ErrorType() string {
	return "dolt_database_create_failed"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *DatabaseCreateError) ErrorDetails() map[string]any {
	return map[string]any{"name": e.Name, "error": e.Err.Error()}
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

// InvalidDatabaseIdentifierError indicates a database name does not match the
// supported identifier contract.
type InvalidDatabaseIdentifierError struct {
	Name string
}

func (e *InvalidDatabaseIdentifierError) Error() string {
	return fmt.Sprintf("invalid database identifier %q: only letters, numbers, underscores (_), hyphens (-), and dots (.) are supported", e.Name)
}
