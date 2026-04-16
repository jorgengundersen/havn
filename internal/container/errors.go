package container

import "fmt"

// NotFoundError indicates a container does not exist.
type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("container %q not found", e.Name)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *NotFoundError) ErrorType() string {
	return "container_not_found"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *NotFoundError) ErrorDetails() map[string]any {
	return map[string]any{"name": e.Name}
}

// NetworkNotFoundError indicates a Docker network does not exist.
type NetworkNotFoundError struct {
	Name string
}

func (e *NetworkNotFoundError) Error() string {
	return fmt.Sprintf("network %q not found", e.Name)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *NetworkNotFoundError) ErrorType() string {
	return "network_not_found"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *NetworkNotFoundError) ErrorDetails() map[string]any {
	return map[string]any{"name": e.Name}
}

// ImageNotFoundError indicates an image does not exist locally.
type ImageNotFoundError struct {
	Name string
}

func (e *ImageNotFoundError) Error() string {
	return fmt.Sprintf("image %q not found", e.Name)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *ImageNotFoundError) ErrorType() string {
	return "image_not_found"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *ImageNotFoundError) ErrorDetails() map[string]any {
	return map[string]any{"name": e.Name}
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

// EnterContainerNotRunningError indicates plain-shell entry was requested while
// the project container is missing or stopped.
type EnterContainerNotRunningError struct {
	Name        string
	ProjectPath string
	State       string
}

func (e *EnterContainerNotRunningError) Error() string {
	return fmt.Sprintf("container %q is %s; run 'havn up %s' first", e.Name, e.State, e.ProjectPath)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *EnterContainerNotRunningError) ErrorType() string {
	return "enter_container_not_running"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *EnterContainerNotRunningError) ErrorDetails() map[string]any {
	return map[string]any{
		"name":         e.Name,
		"project_path": e.ProjectPath,
		"state":        e.State,
	}
}
