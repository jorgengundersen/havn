package dolt_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/dolt"
)

func TestStart_CreatesNewContainer(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: false,
		createID:     "new-id",
		execOutput:   "1",
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{
		Network: "havn-net",
		Dolt: config.DoltConfig{
			Port:  3308,
			Image: "dolthub/dolt-sql-server:latest",
		},
	}

	err := mgr.Start(context.Background(), cfg)

	require.NoError(t, err)
	assert.Equal(t, "havn-dolt", backend.createOpts.Name)
	assert.Equal(t, "dolthub/dolt-sql-server:latest", backend.createOpts.Image)
	assert.Equal(t, "havn-net", backend.createOpts.Network)
	assert.Equal(t, "unless-stopped", backend.createOpts.Restart)
	assert.Equal(t, "havn", backend.createOpts.Labels["managed-by"])
	assert.Contains(t, backend.createOpts.Env, "DOLT_ROOT_HOST=%")
	assert.Equal(t, "/var/lib/dolt", backend.createOpts.Volumes["havn-dolt-data"])
	assert.Equal(t, "/etc/dolt/servercfg.d", backend.createOpts.Volumes["havn-dolt-config"])

	// Verify config was copied
	assert.Contains(t, string(backend.copiedData), "port: 3308")
	assert.Equal(t, "/etc/dolt/servercfg.d", backend.copiedPath)
}

func TestStart_CopiesConfigAfterContainerCreate(t *testing.T) {
	var backend *fakeBackend
	backend = &fakeBackend{
		inspectFound: false,
		createID:     "new-id",
		execOutput:   "1",
		copyFunc: func(_ string, _ []byte) error {
			if !backend.createCalled {
				return fmt.Errorf("copy called before create")
			}
			return nil
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{
		Network: "havn-net",
		Dolt: config.DoltConfig{
			Port:  3308,
			Image: "dolthub/dolt-sql-server:latest",
		},
	}

	err := mgr.Start(context.Background(), cfg)

	require.NoError(t, err)
}

func TestStart_ExistingManagedStopped(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "existing-id",
			Running: false,
			Image:   "dolthub/dolt-sql-server:latest",
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execOutput: "1",
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{Port: 3308, Image: "dolthub/dolt-sql-server:latest"},
	}

	err := mgr.Start(context.Background(), cfg)

	assert.NoError(t, err)
	// Should not have called ContainerCreate (empty createOpts.Name)
	assert.Empty(t, backend.createOpts.Name)
}

func TestStart_ExistingNotManaged(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "foreign-id",
			Running: true,
			Image:   "some-image",
			Labels:  map[string]string{},
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{}

	err := mgr.Start(context.Background(), cfg)

	var notManaged *dolt.NotManagedError
	assert.ErrorAs(t, err, &notManaged)
	assert.Equal(t, "havn-dolt", notManaged.Name)
}

func TestStart_HealthCheckTimeout(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: false,
		createID:     "new-id",
		execErr:      fmt.Errorf("connection refused"),
	}
	mgr := dolt.NewManagerWithHealthTimeout(backend, 50*time.Millisecond)
	cfg := config.Config{
		Network: "havn-net",
		Dolt:    config.DoltConfig{Port: 3308, Image: "dolthub/dolt-sql-server:latest"},
	}

	err := mgr.Start(context.Background(), cfg)

	var timeout *dolt.HealthCheckTimeoutError
	assert.ErrorAs(t, err, &timeout)
}

func TestStart_ExistingManagedRunning(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "running-id",
			Running: true,
			Image:   "dolthub/dolt-sql-server:latest",
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{}

	err := mgr.Start(context.Background(), cfg)

	assert.NoError(t, err)
	assert.NotEmpty(t, backend.execCalls)
	assert.Equal(t, []string{"dolt", "sql", "-q", "SELECT 1"}, backend.execCalls[0].cmd)
}

func TestStop_Success(t *testing.T) {
	backend := &fakeBackend{}
	mgr := dolt.NewManager(backend)

	err := mgr.Stop(context.Background())

	assert.NoError(t, err)
}

func TestStatus_Running(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "abc123",
			Running: true,
			Image:   "dolthub/dolt-sql-server:latest",
			Labels:  map[string]string{"managed-by": "havn"},
			Network: "havn-net",
			Port:    3308,
		},
	}
	mgr := dolt.NewManager(backend)

	status, err := mgr.Status(context.Background())

	require.NoError(t, err)
	assert.True(t, status.Running)
	assert.Equal(t, "havn-dolt", status.Container)
	assert.Equal(t, "dolthub/dolt-sql-server:latest", status.Image)
	assert.Equal(t, 3308, status.Port)
	assert.Equal(t, "havn-net", status.Network)
	assert.True(t, status.ManagedByHavn)
}

func TestStatus_NotRunning(t *testing.T) {
	backend := &fakeBackend{inspectFound: false}
	mgr := dolt.NewManager(backend)

	status, err := mgr.Status(context.Background())

	require.NoError(t, err)
	assert.False(t, status.Running)
}
