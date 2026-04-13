package dolt_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/dolt"
)

func TestDatabases_FiltersSystemDatabases(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execOutput: "+--------------------+\n" +
			"| Database           |\n" +
			"+--------------------+\n" +
			"| api                |\n" +
			"| information_schema |\n" +
			"| mysql              |\n" +
			"| web                |\n" +
			"+--------------------+\n",
	}
	mgr := dolt.NewManager(backend)

	dbs, err := mgr.Databases(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"api", "web"}, dbs)
}

func TestDatabases_OnlySystemDatabases(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execOutput: "+--------------------+\n" +
			"| Database           |\n" +
			"+--------------------+\n" +
			"| information_schema |\n" +
			"| mysql              |\n" +
			"+--------------------+\n",
	}
	mgr := dolt.NewManager(backend)

	dbs, err := mgr.Databases(context.Background())

	require.NoError(t, err)
	assert.Empty(t, dbs)
}

func TestDatabases_ExecError(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execErr: assert.AnError,
	}
	mgr := dolt.NewManager(backend)

	_, err := mgr.Databases(context.Background())

	assert.ErrorIs(t, err, assert.AnError)
}

func TestDrop_ExecutesDropDatabase(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Drop(context.Background(), "myproject")

	require.NoError(t, err)
	require.Len(t, backend.execCalls, 1)
	assert.Equal(t, []string{"dolt", "sql", "-q", "DROP DATABASE `myproject`"}, backend.execCalls[0].cmd)
}

func TestDrop_ExecError(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execErr: assert.AnError,
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Drop(context.Background(), "myproject")

	assert.ErrorIs(t, err, assert.AnError)
}

func TestConnect_CallsInteractiveExec(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Connect(context.Background())

	require.NoError(t, err)
	require.Len(t, backend.interactiveCalls, 1)
	assert.Equal(t, []string{"dolt", "sql"}, backend.interactiveCalls[0].cmd)
}

func TestConnect_ExecError(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		interactiveErr: assert.AnError,
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Connect(context.Background())

	assert.ErrorIs(t, err, assert.AnError)
}

func TestDrop_InvalidDatabaseIdentifier(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Drop(context.Background(), "mydb`; DROP DATABASE prod; --")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid database identifier")
	assert.Empty(t, backend.execCalls)
}

func TestDatabases_UnmanagedContainerConflict(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "foreign-id",
			Running: true,
			Labels:  map[string]string{},
		},
	}
	mgr := dolt.NewManager(backend)

	_, err := mgr.Databases(context.Background())

	var notManaged *dolt.NotManagedError
	assert.ErrorAs(t, err, &notManaged)
	assert.Equal(t, "havn-dolt", notManaged.Name)
}

func TestDatabases_ServerNotRunning(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: false,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	mgr := dolt.NewManager(backend)

	_, err := mgr.Databases(context.Background())

	var notRunning *dolt.ServerNotRunningError
	assert.ErrorAs(t, err, &notRunning)
	assert.Equal(t, "havn-dolt", notRunning.Name)
}
