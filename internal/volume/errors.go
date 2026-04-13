package volume

import "fmt"

// NotFoundError indicates a named volume does not exist.
type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("volume %q not found", e.Name)
}
