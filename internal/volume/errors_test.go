package volume_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/volume"
)

func TestNotFoundError_TypedError(t *testing.T) {
	err := &volume.NotFoundError{Name: "havn-dolt-data"}

	assert.Equal(t, "volume_not_found", err.ErrorType())
	assert.Equal(t, map[string]any{"name": "havn-dolt-data"}, err.ErrorDetails())
}
