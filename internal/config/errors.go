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
