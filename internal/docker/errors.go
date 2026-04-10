// Package docker wraps the Docker SDK behind havn-native types.
package docker

import "fmt"

// DaemonUnreachableError indicates the Docker daemon cannot be contacted.
type DaemonUnreachableError struct {
	Host string
}

func (e *DaemonUnreachableError) Error() string {
	return fmt.Sprintf("daemon unreachable at %q", e.Host)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *DaemonUnreachableError) ErrorType() string {
	return "daemon_unreachable"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *DaemonUnreachableError) ErrorDetails() map[string]any {
	return map[string]any{"host": e.Host}
}

// ContainerNotFoundError indicates a container does not exist in Docker.
type ContainerNotFoundError struct {
	Name string
}

func (e *ContainerNotFoundError) Error() string {
	return fmt.Sprintf("container %q not found", e.Name)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *ContainerNotFoundError) ErrorType() string {
	return "container_not_found"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *ContainerNotFoundError) ErrorDetails() map[string]any {
	return map[string]any{"name": e.Name}
}

// ImageNotFoundError indicates a Docker image does not exist locally.
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

// VolumeNotFoundError indicates a Docker volume does not exist.
type VolumeNotFoundError struct {
	Name string
}

func (e *VolumeNotFoundError) Error() string {
	return fmt.Sprintf("volume %q not found", e.Name)
}

// ErrorType returns the stable snake_case identifier for this error.
func (e *VolumeNotFoundError) ErrorType() string {
	return "volume_not_found"
}

// ErrorDetails returns structured fields for JSON error output.
func (e *VolumeNotFoundError) ErrorDetails() map[string]any {
	return map[string]any{"name": e.Name}
}
