package doctor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// This uses package doctor (not doctor_test) to verify internal metadata
// behavior that external tests cannot observe directly.
func TestHostCheckMetadata_PrerequisitesAreDefensivelyCopied(t *testing.T) {
	meta := newHostCheckMetadata("base_image", []string{"docker_daemon"}, defaultTimeout)

	prerequisites := meta.Prerequisites()
	prerequisites[0] = "mutated"

	assert.Equal(t, []string{"docker_daemon"}, meta.Prerequisites())
}
