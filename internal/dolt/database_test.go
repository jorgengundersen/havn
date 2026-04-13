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
		execErr: assert.AnError,
	}
	mgr := dolt.NewManager(backend)

	_, err := mgr.Databases(context.Background())

	assert.ErrorIs(t, err, assert.AnError)
}

func TestDrop_ExecutesDropDatabase(t *testing.T) {
	backend := &fakeBackend{}
	mgr := dolt.NewManager(backend)

	err := mgr.Drop(context.Background(), "myproject")

	require.NoError(t, err)
	require.Len(t, backend.execCalls, 1)
	assert.Equal(t, []string{"dolt", "sql", "-q", "DROP DATABASE `myproject`"}, backend.execCalls[0].cmd)
}

func TestDrop_ExecError(t *testing.T) {
	backend := &fakeBackend{
		execErr: assert.AnError,
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Drop(context.Background(), "myproject")

	assert.ErrorIs(t, err, assert.AnError)
}

func TestConnect_CallsInteractiveExec(t *testing.T) {
	backend := &fakeBackend{}
	mgr := dolt.NewManager(backend)

	err := mgr.Connect(context.Background())

	require.NoError(t, err)
	require.Len(t, backend.interactiveCalls, 1)
	assert.Equal(t, []string{"dolt", "sql"}, backend.interactiveCalls[0].cmd)
}

func TestConnect_ExecError(t *testing.T) {
	backend := &fakeBackend{
		interactiveErr: assert.AnError,
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Connect(context.Background())

	assert.ErrorIs(t, err, assert.AnError)
}

func TestDrop_InvalidDatabaseIdentifier(t *testing.T) {
	backend := &fakeBackend{}
	mgr := dolt.NewManager(backend)

	err := mgr.Drop(context.Background(), "mydb`; DROP DATABASE prod; --")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid database identifier")
	assert.Empty(t, backend.execCalls)
}
