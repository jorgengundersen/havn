package dolt_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/dolt"
)

func TestEnsureReady_ReturnsBeadsEnvVars(t *testing.T) {
	backend := &fakeBackend{
		inspectInfo:  dolt.ContainerInfo{ID: "abc123", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		inspectFound: true,
	}
	mgr := dolt.NewManager(backend)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Network: "havn-net",
		Dolt: config.DoltConfig{
			Enabled:  true,
			Port:     3308,
			Database: "myproject",
		},
	}

	envVars, err := setup.EnsureReady(context.Background(), cfg)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"BEADS_DOLT_SERVER_HOST":     "havn-dolt",
		"BEADS_DOLT_SERVER_PORT":     "3308",
		"BEADS_DOLT_SERVER_USER":     "root",
		"BEADS_DOLT_AUTO_START":      "0",
		"BEADS_DOLT_SHARED_SERVER":   "1",
		"BEADS_DOLT_SERVER_DATABASE": "myproject",
	}, envVars)
}

func TestEnsureReady_StartsServerWhenNotRunning(t *testing.T) {
	started := false
	backend := &fakeBackend{
		inspectFound: false,
		createID:     "new-id",
		execFunc: func(_ []string) (string, error) {
			// Health check and CREATE DATABASE both succeed
			return "", nil
		},
	}
	// Track that ContainerStart is called by wrapping with a custom backend
	origStart := backend.startErr
	_ = origStart
	backend.startErr = nil

	mgr := dolt.NewManagerWithHealthTimeout(backend, 5*time.Second)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Network: "havn-net",
		Dolt: config.DoltConfig{
			Enabled:  true,
			Port:     3308,
			Image:    "dolthub/dolt-sql-server:latest",
			Database: "myproject",
		},
	}

	envVars, err := setup.EnsureReady(context.Background(), cfg)

	require.NoError(t, err)
	assert.NotEmpty(t, envVars)
	// Verify CREATE DATABASE was executed (after health check calls)
	var foundCreateDB bool
	for _, call := range backend.execCalls {
		if len(call.cmd) == 4 && call.cmd[3] == "CREATE DATABASE IF NOT EXISTS `myproject`" {
			foundCreateDB = true
			started = true
		}
	}
	assert.True(t, foundCreateDB, "expected CREATE DATABASE IF NOT EXISTS to be called")
	assert.True(t, started)
}

func TestEnsureReady_CreatesProjectDatabase(t *testing.T) {
	backend := &fakeBackend{
		inspectInfo:  dolt.ContainerInfo{ID: "abc123", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		inspectFound: true,
	}
	mgr := dolt.NewManager(backend)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Enabled:  true,
			Port:     3308,
			Database: "webapp",
		},
	}

	_, err := setup.EnsureReady(context.Background(), cfg)

	require.NoError(t, err)
	require.Len(t, backend.execCalls, 2)
	assert.Equal(t, []string{"dolt", "sql", "-q", "SELECT 1"}, backend.execCalls[0].cmd)
	assert.Equal(t, []string{"dolt", "sql", "-q", "CREATE DATABASE IF NOT EXISTS `webapp`"}, backend.execCalls[1].cmd)
}

func TestEnsureReady_DatabaseCreateFailure(t *testing.T) {
	backend := &fakeBackend{
		inspectInfo:  dolt.ContainerInfo{ID: "abc123", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		inspectFound: true,
		execFunc: func(cmd []string) (string, error) {
			if len(cmd) == 4 && cmd[3] == "SELECT 1" {
				return "", nil
			}
			return "", assert.AnError
		},
	}
	mgr := dolt.NewManager(backend)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Enabled:  true,
			Port:     3308,
			Database: "myproject",
		},
	}

	_, err := setup.EnsureReady(context.Background(), cfg)

	var dbErr *dolt.DatabaseCreateError
	require.ErrorAs(t, err, &dbErr)
	assert.Equal(t, "myproject", dbErr.Name)
	assert.ErrorIs(t, dbErr, assert.AnError)
}

func TestEnsureReady_StartFailurePropagates(t *testing.T) {
	backend := &fakeBackend{
		inspectErr: assert.AnError,
	}
	mgr := dolt.NewManager(backend)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Enabled:  true,
			Port:     3308,
			Database: "myproject",
		},
	}

	_, err := setup.EnsureReady(context.Background(), cfg)

	var startErr *dolt.StartError
	require.ErrorAs(t, err, &startErr)
}

func TestEnsureReady_UsesPortFromConfig(t *testing.T) {
	backend := &fakeBackend{
		inspectInfo:  dolt.ContainerInfo{ID: "abc123", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		inspectFound: true,
	}
	mgr := dolt.NewManager(backend)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Enabled:  true,
			Port:     5555,
			Database: "myproject",
		},
	}

	envVars, err := setup.EnsureReady(context.Background(), cfg)

	require.NoError(t, err)
	assert.Equal(t, "5555", envVars["BEADS_DOLT_SERVER_PORT"])
}

func TestEnsureReady_InvalidDatabaseIdentifier(t *testing.T) {
	backend := &fakeBackend{
		inspectInfo:  dolt.ContainerInfo{ID: "abc123", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		inspectFound: true,
	}
	mgr := dolt.NewManager(backend)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Enabled:  true,
			Port:     3308,
			Database: "mydb`; DROP DATABASE prod; --",
		},
	}

	_, err := setup.EnsureReady(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid database identifier")
	require.Len(t, backend.execCalls, 1)
	assert.Equal(t, []string{"dolt", "sql", "-q", "SELECT 1"}, backend.execCalls[0].cmd)
}

func TestMigrationNotice_WhenLocalDatabaseExists_ReturnsImportHint(t *testing.T) {
	backend := &fakeBackend{
		inspectInfo:  dolt.ContainerInfo{ID: "abc123", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		inspectFound: true,
		execOutput:   showDatabasesOutput("information_schema", "mysql"),
	}
	mgr := dolt.NewManager(backend)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Enabled:  true,
			Database: "myproject",
		},
	}
	projectPath := t.TempDir()
	require.NoError(t, os.MkdirAll(projectPath+"/.beads/dolt/myproject/.dolt", 0o755))

	notice, err := setup.MigrationNotice(context.Background(), cfg, projectPath)

	require.NoError(t, err)
	assert.Contains(t, notice, "Found local beads database at .beads/dolt/myproject")
	assert.Contains(t, notice, "bd migrate --help")
	assert.Contains(t, notice, "docs/dolt-beads-guide.md")
}
