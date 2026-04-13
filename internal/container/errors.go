package container

import "fmt"

// NotFoundError indicates a container does not exist.
type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("container %q not found", e.Name)
}

// NetworkNotFoundError indicates a Docker network does not exist.
type NetworkNotFoundError struct {
	Name string
}

func (e *NetworkNotFoundError) Error() string {
	return fmt.Sprintf("network %q not found", e.Name)
}

// BuildError wraps a failure during image build.
type BuildError struct {
	Err error
}

func (e *BuildError) Error() string {
	return fmt.Sprintf("build image: %s", e.Err)
}

func (e *BuildError) Unwrap() error {
	return e.Err
}
