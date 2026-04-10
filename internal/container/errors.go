package container

import "fmt"

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
