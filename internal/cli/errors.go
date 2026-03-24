package cli

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ExitError wraps an error with a specific process exit code.
type ExitError struct {
	Code int
	Err  error
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
// Domain error type translations will be added as domain packages land.
func FormatError(err error) string {
	return err.Error()
}

// Error writes an error to stderr, formatted based on the output mode.
// In JSON mode: {"error": "message"}. In normal mode: "Error: message".
func (o *Output) Error(err error) {
	msg := FormatError(err)
	if o.json {
		_ = json.NewEncoder(o.stderr).Encode(map[string]string{"error": msg})
		return
	}
	_, _ = fmt.Fprintf(o.stderr, "Error: %s\n", msg)
}
