package config

import "fmt"

// ParseError represents a TOML parse failure with file location context.
type ParseError struct {
	File   string
	Line   int
	Detail string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("config parse error at %s:%d: %s", e.File, e.Line, e.Detail)
}

// ErrorType returns the stable identifier for structured JSON output.
func (e *ParseError) ErrorType() string {
	return "config_parse_error"
}

// ErrorDetails returns structured fields for JSON output.
func (e *ParseError) ErrorDetails() map[string]any {
	return map[string]any{
		"file":   e.File,
		"line":   e.Line,
		"detail": e.Detail,
	}
}

// ValidationError represents an invalid config value detected during validation.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid config: %s: %s", e.Field, e.Reason)
}

// ErrorType returns the stable identifier for structured JSON output.
func (e *ValidationError) ErrorType() string {
	return "config_validation_error"
}

// ErrorDetails returns structured fields for JSON output.
func (e *ValidationError) ErrorDetails() map[string]any {
	return map[string]any{
		"field":  e.Field,
		"reason": e.Reason,
	}
}
