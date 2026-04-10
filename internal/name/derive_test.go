package name_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/name"
)

func TestDeriveContainerName(t *testing.T) {
	tests := []struct {
		name    string
		parent  string
		project string
		want    name.ContainerName
	}{
		{
			name:    "standard path",
			parent:  "user",
			project: "api",
			want:    "havn-user-api",
		},
		{
			name:    "special characters sanitized",
			parent:  "user",
			project: "my.project",
			want:    "havn-user-my-project",
		},
		{
			name:    "uppercase lowered",
			parent:  "User",
			project: "MyApp",
			want:    "havn-user-myapp",
		},
		{
			name:    "consecutive special chars collapsed",
			parent:  "user",
			project: "my..project",
			want:    "havn-user-my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := name.DeriveContainerName(tt.parent, tt.project)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeriveContainerName_EmptyAfterSanitization(t *testing.T) {
	_, err := name.DeriveContainerName("...", "...")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty after sanitization")
}

func TestDeriveContainerName_ExceedsLengthLimit(t *testing.T) {
	long := strings.Repeat("a", 130)
	_, err := name.DeriveContainerName(long, "project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestDeriveContainerName_Deterministic(t *testing.T) {
	a, err := name.DeriveContainerName("user", "api")
	assert.NoError(t, err)
	b, err := name.DeriveContainerName("user", "api")
	assert.NoError(t, err)
	assert.Equal(t, a, b)
}
