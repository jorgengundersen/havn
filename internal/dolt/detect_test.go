package dolt_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/dolt"
)

func TestDetectMigration_AllConditionsMet(t *testing.T) {
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
	dirExists := func(path string) bool {
		return path == "/projects/myapp/.beads/dolt/myproject/.dolt"
	}

	hint, err := setup.DetectMigration(context.Background(), cfg, "/projects/myapp", dirExists)

	require.NoError(t, err)
	require.NotNil(t, hint)
	assert.Equal(t, "myproject", hint.DatabaseName)
	assert.Equal(t, ".beads/dolt/myproject", hint.LocalPath)
}

func TestDetectMigration_NoLocalDatabase(t *testing.T) {
	backend := &fakeBackend{
		inspectInfo:  dolt.ContainerInfo{ID: "abc123", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		inspectFound: true,
	}
	mgr := dolt.NewManager(backend)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Enabled:  true,
			Database: "myproject",
		},
	}
	dirExists := func(_ string) bool { return false }

	hint, err := setup.DetectMigration(context.Background(), cfg, "/projects/myapp", dirExists)

	require.NoError(t, err)
	assert.Nil(t, hint)
}

func TestDetectMigration_DatabaseAlreadyOnServer(t *testing.T) {
	backend := &fakeBackend{
		inspectInfo:  dolt.ContainerInfo{ID: "abc123", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		inspectFound: true,
		execOutput:   showDatabasesOutput("information_schema", "mysql", "myproject"),
	}
	mgr := dolt.NewManager(backend)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Enabled:  true,
			Database: "myproject",
		},
	}
	dirExists := func(_ string) bool { return true }

	hint, err := setup.DetectMigration(context.Background(), cfg, "/projects/myapp", dirExists)

	require.NoError(t, err)
	assert.Nil(t, hint)
}

func TestDetectMigration_DatabaseListError(t *testing.T) {
	backend := &fakeBackend{
		inspectInfo:  dolt.ContainerInfo{ID: "abc123", Running: true, Labels: map[string]string{"managed-by": "havn"}},
		inspectFound: true,
		execErr:      assert.AnError,
	}
	mgr := dolt.NewManager(backend)
	setup := dolt.NewSetup(mgr, backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Enabled:  true,
			Database: "myproject",
		},
	}
	dirExists := func(_ string) bool { return true }

	hint, err := setup.DetectMigration(context.Background(), cfg, "/projects/myapp", dirExists)

	assert.Nil(t, hint)
	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestDetectMigration_ChecksCorrectPath(t *testing.T) {
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
			Database: "customdb",
		},
	}
	var checkedPath string
	dirExists := func(path string) bool {
		checkedPath = path
		return true
	}

	_, err := setup.DetectMigration(context.Background(), cfg, "/home/user/projects/myapp", dirExists)

	require.NoError(t, err)
	assert.Equal(t, "/home/user/projects/myapp/.beads/dolt/customdb/.dolt", checkedPath)
}

func showDatabasesOutput(names ...string) string {
	out := "+--------------------+\n| Database           |\n+--------------------+\n"
	for _, n := range names {
		out += "| " + n + " |\n"
	}
	out += "+--------------------+\n"
	return out
}
