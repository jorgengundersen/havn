package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
