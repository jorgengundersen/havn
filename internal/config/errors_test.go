package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/config"
)

func TestValidationError_Error(t *testing.T) {
	err := &config.ValidationError{
		Field:  "resources.cpus",
		Reason: "must be greater than 0",
	}

	assert.Equal(t, "invalid config: resources.cpus: must be greater than 0", err.Error())
}

func TestParseError_TypedError(t *testing.T) {
	err := &config.ParseError{
		File:   "config.toml",
		Line:   5,
		Detail: "unexpected key",
	}

	assert.Equal(t, "config_parse_error", err.ErrorType())
	assert.Equal(t, map[string]any{
		"file":   "config.toml",
		"line":   5,
		"detail": "unexpected key",
	}, err.ErrorDetails())
}

func TestValidationError_TypedError(t *testing.T) {
	err := &config.ValidationError{
		Field:  "resources.cpus",
		Reason: "must be greater than 0",
	}

	assert.Equal(t, "config_validation_error", err.ErrorType())
	assert.Equal(t, map[string]any{
		"field":  "resources.cpus",
		"reason": "must be greater than 0",
	}, err.ErrorDetails())
}
