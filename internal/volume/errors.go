package volume

import "fmt"

// NotFoundError indicates a named volume does not exist.
type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("volume %q not found", e.Name)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *NotFoundError) ErrorType() string {
	return "volume_not_found"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *NotFoundError) ErrorDetails() map[string]any {
	return map[string]any{"name": e.Name}
}
