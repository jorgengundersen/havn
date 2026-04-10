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
		execFunc: func(cmd []string) (string, error) {
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
		execOutput: "+--------------------+\n| Database           |\n+--------------------+\n| mydb               |\n+--------------------+\n",
	}
	mgr := dolt.NewManager(backend)
	cfg := config.Config{
		Dolt: config.DoltConfig{Database: "mydb"},
	}

	_, err := mgr.Import(context.Background(), projectDir, cfg, true)

	assert.NoError(t, err)
	assert.NotEmpty(t, backend.copiedData)
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
		execFunc: func(cmd []string) (string, error) {
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
		execOutput: "+--------------------+\n| Database           |\n+--------------------+\n| information_schema |\n+--------------------+\n",
	}
	mgr := dolt.NewManager(backend)

	err := mgr.Export(context.Background(), "nonexistent", t.TempDir())

	var notFound *dolt.DatabaseNotFoundError
	assert.ErrorAs(t, err, &notFound)
	assert.Equal(t, "nonexistent", notFound.Name)
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
