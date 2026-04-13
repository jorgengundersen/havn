package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldRenderCLIError_SuppressedExitError(t *testing.T) {
	err := &ExitError{Code: 1, Err: errors.New("doctor found warnings"), SuppressOutput: true}

	assert.False(t, shouldRenderCLIError(err))
}
