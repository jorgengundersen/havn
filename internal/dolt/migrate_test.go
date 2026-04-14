package dolt_test

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/dolt"
)

func TestImport_SourceDirNotFound(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "abc",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{Database: "mydb"},
	}

	_, err := mgr.Import(context.Background(), "/nonexistent/path", cfg, false)

	var notFound *dolt.DatabaseNotFoundError
	assert.ErrorAs(t, err, &notFound)
	assert.Equal(t, "mydb", notFound.Name)
}

func TestImport_DatabaseExistsNoForce(t *testing.T) {
	projectDir := t.TempDir()
	dbDir := projectDir + "/.beads/dolt/mydb/.dolt"
	require.NoError(t, os.MkdirAll(dbDir, 0o755))

	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execOutput: "+--------------------+\n| Database           |\n+--------------------+\n| mydb               |\n+--------------------+\n",
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{Database: "mydb"},
	}

	_, err := mgr.Import(context.Background(), projectDir, cfg, false)

	var exists *dolt.DatabaseExistsError
	assert.ErrorAs(t, err, &exists)
	assert.Equal(t, "mydb", exists.Name)
}

func TestImport_Success(t *testing.T) {
	projectDir := t.TempDir()
	dbDir := projectDir + "/.beads/dolt/mydb"
	require.NoError(t, os.MkdirAll(dbDir, 0o755))
	require.NoError(t, os.WriteFile(dbDir+"/manifest", []byte("dolt-manifest"), 0o644))

	callCount := 0
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execFunc: func(_ []string) (string, error) {
			callCount++
			if callCount == 1 {
				return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n+--------------------+\n", nil
			}
			return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n| mydb               |\n+--------------------+\n", nil
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{Database: "mydb"},
	}

	result, err := mgr.Import(context.Background(), projectDir, cfg, false)

	require.NoError(t, err)
	assert.Equal(t, "mydb", result.DatabaseName)
	assert.False(t, result.Overwrote)
	assert.Empty(t, result.Warnings)
	assert.Equal(t, "/var/lib/dolt", backend.copiedPath)
	assert.NotEmpty(t, backend.copiedData)
	assert.Len(t, backend.execCalls, 2)
}

func TestImport_ForceOverwriteExisting(t *testing.T) {
	projectDir := t.TempDir()
	dbDir := projectDir + "/.beads/dolt/mydb"
	require.NoError(t, os.MkdirAll(dbDir, 0o755))
	require.NoError(t, os.WriteFile(dbDir+"/manifest", []byte("data"), 0o644))

	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execOutput: "+--------------------+\n| Database           |\n+--------------------+\n| mydb               |\n+--------------------+\n",
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{Database: "mydb"},
	}

	result, err := mgr.Import(context.Background(), projectDir, cfg, true)

	assert.NoError(t, err)
	assert.True(t, result.Overwrote)
	assert.NotEmpty(t, backend.copiedData)
}

func TestImport_ForceOverwriteExisting_UsesCleanReplacementStrategy(t *testing.T) {
	projectDir := t.TempDir()
	dbDir := projectDir + "/.beads/dolt/mydb"
	require.NoError(t, os.MkdirAll(dbDir, 0o755))
	require.NoError(t, os.WriteFile(dbDir+"/manifest", []byte("data"), 0o644))

	showCalls := 0
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execFunc: func(cmd []string) (string, error) {
			if len(cmd) == 4 && cmd[0] == "dolt" && cmd[3] == "SHOW DATABASES" {
				showCalls++
				if showCalls == 1 {
					return "+--------------------+\n| Database           |\n+--------------------+\n| mydb               |\n+--------------------+\n", nil
				}
				return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n| mydb               |\n+--------------------+\n", nil
			}
			if len(cmd) == 3 && cmd[0] == "rm" && cmd[1] == "-rf" && cmd[2] == "/var/lib/dolt/mydb" {
				return "", nil
			}
			return "", nil
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{Dolt: config.DoltConfig{Database: "mydb"}}

	result, err := mgr.Import(context.Background(), projectDir, cfg, true)

	require.NoError(t, err)
	assert.True(t, result.Overwrote)
	assert.NotEmpty(t, backend.copiedData)
	require.Len(t, backend.execCalls, 3)
	assert.Equal(t, []string{"dolt", "sql", "-q", "SHOW DATABASES"}, backend.execCalls[0].cmd)
	assert.Equal(t, []string{"rm", "-rf", "/var/lib/dolt/mydb"}, backend.execCalls[1].cmd)
	assert.Equal(t, []string{"dolt", "sql", "-q", "SHOW DATABASES"}, backend.execCalls[2].cmd)
}

func TestImport_ForceWhenDestinationMissing_DoesNotReportOverwrote(t *testing.T) {
	projectDir := t.TempDir()
	dbDir := projectDir + "/.beads/dolt/mydb"
	require.NoError(t, os.MkdirAll(dbDir, 0o755))
	require.NoError(t, os.WriteFile(dbDir+"/manifest", []byte("data"), 0o644))

	showCalls := 0
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execFunc: func(cmd []string) (string, error) {
			if len(cmd) == 4 && cmd[0] == "dolt" && cmd[3] == "SHOW DATABASES" {
				showCalls++
				if showCalls == 1 {
					return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n+--------------------+\n", nil
				}
				return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n| mydb               |\n+--------------------+\n", nil
			}
			return "", nil
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{Dolt: config.DoltConfig{Database: "mydb"}}

	result, err := mgr.Import(context.Background(), projectDir, cfg, true)

	require.NoError(t, err)
	assert.False(t, result.Overwrote)
	require.Len(t, backend.execCalls, 2)
	assert.Equal(t, []string{"dolt", "sql", "-q", "SHOW DATABASES"}, backend.execCalls[0].cmd)
	assert.Equal(t, []string{"dolt", "sql", "-q", "SHOW DATABASES"}, backend.execCalls[1].cmd)
}

func TestImport_ProjectIDMismatchWarning(t *testing.T) {
	projectDir := t.TempDir()
	dbDir := projectDir + "/.beads/dolt/mydb"
	require.NoError(t, os.MkdirAll(dbDir, 0o755))
	require.NoError(t, os.WriteFile(dbDir+"/manifest", []byte("data"), 0o644))

	// Write metadata.json with a project_id.
	metadataDir := projectDir + "/.beads"
	metadata := map[string]string{"project_id": "local-uuid-111"}
	data, err := json.Marshal(metadata)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(metadataDir+"/metadata.json", data, 0o644))

	callCount := 0
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execFunc: func(_ []string) (string, error) {
			callCount++
			// Calls 1 and 2: SHOW DATABASES (existence check + verification).
			if callCount <= 2 {
				if callCount == 1 {
					return "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n+--------------------+\n", nil
				}
				return "+--------------------+\n| Database           |\n+--------------------+\n| mydb               |\n+--------------------+\n", nil
			}
			// Call 3: query _project_id from metadata table — returns a different UUID.
			return "+--------------+\n| value        |\n+--------------+\n| db-uuid-222  |\n+--------------+\n", nil
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{Database: "mydb"},
	}

	result, err := mgr.Import(context.Background(), projectDir, cfg, false)

	require.NoError(t, err)
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "project_id mismatch")
}

func TestExport_Success(t *testing.T) {
	destDir := t.TempDir()

	// Build a tar archive that CopyFromContainer will return.
	tarData := buildTestTar(t, "mydb", map[string]string{
		"manifest": "dolt-manifest-data",
	})

	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		// SHOW DATABASES returns the database — it exists.
		execOutput:   "+--------------------+\n| Database           |\n+--------------------+\n| mydb               |\n+--------------------+\n",
		copyFromData: tarData,
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Export(context.Background(), "mydb", destDir)

	require.NoError(t, err)
	// Verify the database was extracted to <destDir>/.beads/dolt/mydb/.
	content, err := os.ReadFile(destDir + "/.beads/dolt/mydb/manifest")
	require.NoError(t, err)
	assert.Equal(t, "dolt-manifest-data", string(content))
}

func TestExport_DatabaseNotOnServer(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execOutput: "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n+--------------------+\n",
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Export(context.Background(), "nonexistent", t.TempDir())

	var notFound *dolt.DatabaseNotFoundError
	assert.ErrorAs(t, err, &notFound)
	assert.Equal(t, "nonexistent", notFound.Name)
}

func TestExport_DestinationVerificationFailsWhenDatabaseMissing(t *testing.T) {
	destDir := t.TempDir()

	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
		execOutput:   "+--------------------+\n| Database           |\n+--------------------+\n| mydb               |\n+--------------------+\n",
		copyFromData: buildTestTar(t, "otherdb", map[string]string{"manifest": "exported-data"}),
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Export(context.Background(), "mydb", destDir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database \"mydb\" not found in destination")
}

func TestImport_InvalidDatabaseIdentifier(t *testing.T) {
	projectDir := t.TempDir()
	badName := "mydb`; DROP DATABASE prod; --"
	dbDir := projectDir + "/.beads/dolt/" + badName
	require.NoError(t, os.MkdirAll(dbDir, 0o755))
	require.NoError(t, os.WriteFile(dbDir+"/manifest", []byte("data"), 0o644))

	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{Database: badName},
	}

	_, err := mgr.Import(context.Background(), projectDir, cfg, false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid database identifier")
	assert.Empty(t, backend.execCalls)
	assert.Empty(t, backend.copiedData)
}

func TestExport_InvalidDatabaseIdentifier(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: true,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Export(context.Background(), "mydb`; DROP DATABASE prod; --", t.TempDir())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid database identifier")
	assert.Empty(t, backend.execCalls)
}

func TestImport_UnmanagedContainerConflict(t *testing.T) {
	projectDir := t.TempDir()
	dbDir := projectDir + "/.beads/dolt/mydb"
	require.NoError(t, os.MkdirAll(dbDir, 0o755))
	require.NoError(t, os.WriteFile(dbDir+"/manifest", []byte("data"), 0o644))

	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "foreign-id",
			Running: true,
			Labels:  map[string]string{},
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{Dolt: config.DoltConfig{Database: "mydb"}}

	_, err := mgr.Import(context.Background(), projectDir, cfg, false)

	var notManaged *dolt.NotManagedError
	assert.ErrorAs(t, err, &notManaged)
	assert.Equal(t, "havn-dolt", notManaged.Name)
	assert.Empty(t, backend.copiedData)
}

func TestExport_UnmanagedContainerConflict(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "foreign-id",
			Running: true,
			Labels:  map[string]string{},
		},
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Export(context.Background(), "mydb", t.TempDir())

	var notManaged *dolt.NotManagedError
	assert.ErrorAs(t, err, &notManaged)
	assert.Equal(t, "havn-dolt", notManaged.Name)
	assert.Empty(t, backend.execCalls)
}

func TestImport_ServerNotRunning(t *testing.T) {
	projectDir := t.TempDir()
	dbDir := projectDir + "/.beads/dolt/mydb"
	require.NoError(t, os.MkdirAll(dbDir, 0o755))
	require.NoError(t, os.WriteFile(dbDir+"/manifest", []byte("data"), 0o644))

	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: false,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{Dolt: config.DoltConfig{Database: "mydb"}}

	_, err := mgr.Import(context.Background(), projectDir, cfg, false)

	var notRunning *dolt.ServerNotRunningError
	assert.ErrorAs(t, err, &notRunning)
	assert.Equal(t, "havn-dolt", notRunning.Name)
	assert.Empty(t, backend.copiedData)
}

func TestExport_ServerNotRunning(t *testing.T) {
	backend := &fakeBackend{
		inspectFound: true,
		inspectInfo: dolt.ContainerInfo{
			ID:      "managed-id",
			Running: false,
			Labels:  map[string]string{"managed-by": "havn"},
		},
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Export(context.Background(), "mydb", t.TempDir())

	var notRunning *dolt.ServerNotRunningError
	assert.ErrorAs(t, err, &notRunning)
	assert.Equal(t, "havn-dolt", notRunning.Name)
	assert.Empty(t, backend.execCalls)
}

// buildTestTar creates a tar archive with a directory prefix and file contents.
func buildTestTar(t *testing.T, prefix string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Write directory header.
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     prefix + "/",
		Mode:     0o755,
	}))

	for name, content := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     prefix + "/" + name,
			Size:     int64(len(content)),
			Mode:     0o644,
		}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	return buf.Bytes()
}
