package dolt_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/dolt"
)

func TestNotManagedError_Message(t *testing.T) {
	err := &dolt.NotManagedError{Name: "havn-dolt"}

	assert.EqualError(t, err, `container "havn-dolt" exists but was not created by havn`)
}

func TestNotManagedError_TypedError(t *testing.T) {
	err := &dolt.NotManagedError{Name: "havn-dolt"}

	assert.Equal(t, "dolt_not_managed", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-dolt"}, err.ErrorDetails())
}

func TestServerNotRunningError_Message(t *testing.T) {
	err := &dolt.ServerNotRunningError{Name: "havn-dolt"}

	assert.EqualError(t, err, `container "havn-dolt" is not running`)
}

func TestServerNotRunningError_TypedError(t *testing.T) {
	err := &dolt.ServerNotRunningError{Name: "havn-dolt"}

	assert.Equal(t, "dolt_server_not_running", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-dolt"}, err.ErrorDetails())
}

func TestStartError_WrapsUnderlying(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := &dolt.StartError{Err: cause}

	assert.EqualError(t, err, "start dolt server: connection refused")
	assert.ErrorIs(t, err, cause)
}

func TestStartError_TypedError(t *testing.T) {
	err := &dolt.StartError{Err: fmt.Errorf("connection refused")}

	assert.Equal(t, "dolt_start_failed", err.ErrorType())
	assert.Equal(t, map[string]any{"error": "connection refused"}, err.ErrorDetails())
}

func TestHealthCheckTimeoutError_Message(t *testing.T) {
	err := &dolt.HealthCheckTimeoutError{Timeout: 30 * time.Second}

	assert.EqualError(t, err, "dolt health check timed out after 30s")
}

func TestHealthCheckTimeoutError_TypedError(t *testing.T) {
	err := &dolt.HealthCheckTimeoutError{Timeout: 30 * time.Second}

	assert.Equal(t, "dolt_health_check_timeout", err.ErrorType())
	assert.Equal(t, map[string]any{"timeout": "30s"}, err.ErrorDetails())
}

func TestDatabaseCreateError_Message(t *testing.T) {
	cause := fmt.Errorf("access denied")
	err := &dolt.DatabaseCreateError{Name: "myproject", Err: cause}

	assert.EqualError(t, err, `create database "myproject": access denied`)
}

func TestDatabaseCreateError_WrapsUnderlying(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := &dolt.DatabaseCreateError{Name: "myproject", Err: cause}

	assert.ErrorIs(t, err, cause)
}

func TestDatabaseCreateError_TypedError(t *testing.T) {
	err := &dolt.DatabaseCreateError{Name: "myproject", Err: fmt.Errorf("access denied")}

	assert.Equal(t, "dolt_database_create_failed", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "myproject", "error": "access denied"}, err.ErrorDetails())
}
