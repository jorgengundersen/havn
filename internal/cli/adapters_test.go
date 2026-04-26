package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/docker"
	"github.com/jorgengundersen/havn/internal/dolt"
)

func TestExecResultError_UsesDefaultMessageWhenStderrEmpty(t *testing.T) {
	err := execResultError(docker.ExecResult{ExitCode: 17, Stderr: []byte("   \n\t")})

	assert.EqualError(t, err, "container exec exited 17: command failed")
}

func TestToDoltContainerInfo_UsesFirstNetworkAndRunningStatus(t *testing.T) {
	got := toDoltContainerInfo(docker.ContainerInfo{
		ID:       "ctr-id",
		Status:   "RUNNING",
		Image:    "dolthub/dolt-sql-server:latest",
		Labels:   map[string]string{"managed-by": "havn"},
		Networks: []string{"havn-net", "other-net"},
	})

	assert.Equal(t, dolt.ContainerInfo{
		ID:      "ctr-id",
		Running: true,
		Image:   "dolthub/dolt-sql-server:latest",
		Labels:  map[string]string{"managed-by": "havn"},
		Network: "havn-net",
	}, got)
}

func TestNormalizeContainerBoundaryError_TranslatesDockerNotFoundErrors(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "container not found",
			err:  &docker.ContainerNotFoundError{Name: "havn-user-api"},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				var translated *container.NotFoundError
				assert.ErrorAs(t, err, &translated)
				assert.Equal(t, "havn-user-api", translated.Name)
			},
		},
		{
			name: "image not found",
			err:  &docker.ImageNotFoundError{Name: "havn-base:latest"},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				var translated *container.ImageNotFoundError
				assert.ErrorAs(t, err, &translated)
				assert.Equal(t, "havn-base:latest", translated.Name)
			},
		},
		{
			name: "network not found",
			err:  &docker.NetworkNotFoundError{Name: "havn-net"},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				var translated *container.NetworkNotFoundError
				assert.ErrorAs(t, err, &translated)
				assert.Equal(t, "havn-net", translated.Name)
			},
		},
		{
			name: "non-docker error unchanged",
			err:  errors.New("boom"),
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				assert.EqualError(t, err, "boom")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := normalizeContainerBoundaryError(tt.err)
			tt.assertErr(t, err)
		})
	}
}
