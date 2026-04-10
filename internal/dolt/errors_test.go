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

func TestStartError_WrapsUnderlying(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := &dolt.StartError{Err: cause}

	assert.EqualError(t, err, "start dolt server: connection refused")
	assert.ErrorIs(t, err, cause)
}

func TestHealthCheckTimeoutError_Message(t *testing.T) {
	err := &dolt.HealthCheckTimeoutError{Timeout: 30 * time.Second}

	assert.EqualError(t, err, "dolt health check timed out after 30s")
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
