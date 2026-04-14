package dolt

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jorgengundersen/havn/internal/config"
)

const doltDataDir = "/var/lib/dolt"

// ImportResult holds the outcome of a database import.
type ImportResult struct {
	DatabaseName string
	Overwrote    bool
	Warnings     []string
}

// Import copies a Dolt database from a project directory into the shared
// server's data volume. It reads the database name from cfg.Dolt.Database
// (or derives it from the project directory name), locates the database at
// <projectPath>/.beads/dolt/<dbname>/, and copies it into the container.
func (m *Manager) Import(ctx context.Context, projectPath string, cfg config.Config, force bool) (ImportResult, error) {
	if err := m.ensureRunningManaged(ctx); err != nil {
		return ImportResult{}, err
	}

	dbName := cfg.Dolt.Database
	if dbName == "" {
		dbName = filepath.Base(projectPath)
	}
	if err := validateDatabaseIdentifier(dbName); err != nil {
		return ImportResult{}, err
	}

	srcDir := filepath.Join(projectPath, ".beads", "dolt", dbName)
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return ImportResult{}, &DatabaseNotFoundError{Name: dbName}
	}

	destinationExisted, err := m.databaseExists(ctx, dbName)
	if err != nil {
		return ImportResult{}, &ImportError{Err: fmt.Errorf("check database: %w", err)}
	}
	if destinationExisted && !force {
		return ImportResult{}, &DatabaseExistsError{Name: dbName}
	}
	if destinationExisted && force {
		destinationPath := filepath.Join(doltDataDir, dbName)
		if _, err := m.backend.ContainerExec(ctx, containerName, []string{"rm", "-rf", destinationPath}); err != nil {
			return ImportResult{}, &ImportError{Err: fmt.Errorf("remove existing database: %w", err)}
		}
	}

	tarData, err := tarDirectory(srcDir, dbName)
	if err != nil {
		return ImportResult{}, &ImportError{Err: fmt.Errorf("tar database: %w", err)}
	}

	if err := m.backend.CopyToContainer(ctx, containerName, doltDataDir, tarData); err != nil {
		return ImportResult{}, &ImportError{Err: fmt.Errorf("copy to container: %w", err)}
	}

	exists, err := m.databaseExists(ctx, dbName)
	if err != nil {
		return ImportResult{}, &ImportError{Err: fmt.Errorf("verify database: %w", err)}
	}
	if !exists {
		return ImportResult{}, &ImportError{Err: fmt.Errorf("database %q not visible after import", dbName)}
	}

	result := ImportResult{DatabaseName: dbName, Overwrote: destinationExisted && force}

	warnings := m.verifyProjectID(ctx, projectPath, dbName)
	result.Warnings = warnings

	return result, nil
}

// Export copies a Dolt database from the shared server's data volume to
// <destPath>/.beads/dolt/<dbName>/.
func (m *Manager) Export(ctx context.Context, dbName string, destPath string) error {
	if err := m.ensureRunningManaged(ctx); err != nil {
		return err
	}

	if err := validateDatabaseIdentifier(dbName); err != nil {
		return err
	}

	exists, err := m.databaseExists(ctx, dbName)
	if err != nil {
		return &ExportError{Err: fmt.Errorf("check database: %w", err)}
	}
	if !exists {
		return &DatabaseNotFoundError{Name: dbName}
	}

	srcPath := fmt.Sprintf("%s/%s", doltDataDir, dbName)
	tarData, err := m.backend.CopyFromContainer(ctx, containerName, srcPath)
	if err != nil {
		return &ExportError{Err: fmt.Errorf("copy from container: %w", err)}
	}

	destDir := filepath.Join(destPath, ".beads", "dolt")
	if err := untarToDirectory(tarData, destDir); err != nil {
		return &ExportError{Err: fmt.Errorf("untar database: %w", err)}
	}

	if _, err := os.Stat(filepath.Join(destDir, dbName)); err != nil {
		if os.IsNotExist(err) {
			return &ExportError{Err: fmt.Errorf("database %q not found in destination after export", dbName)}
		}
		return &ExportError{Err: fmt.Errorf("verify destination: %w", err)}
	}

	return nil
}

// untarToDirectory extracts a tar archive into destDir.
func untarToDirectory(tarData []byte, destDir string) error {
	tr := tar.NewReader(bytes.NewReader(tarData))

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		// Prevent path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("tar entry %q escapes destination directory", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := extractFile(target, header.Mode, tr); err != nil {
				return err
			}
		}
	}

	return nil
}

// verifyProjectID compares the project_id from .beads/metadata.json with
// the _project_id stored in the database metadata table. Returns warnings
// on mismatch; skips silently if either side has no project_id.
func (m *Manager) verifyProjectID(ctx context.Context, projectPath string, dbName string) []string {
	metadataPath := filepath.Join(projectPath, ".beads", "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil // no metadata.json — skip verification
	}

	var meta struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(data, &meta); err != nil || meta.ProjectID == "" {
		return nil // no project_id — skip verification
	}

	query := fmt.Sprintf("SELECT value FROM `%s`.metadata WHERE `key` = '_project_id'", dbName)
	output, err := m.backend.ContainerExec(ctx, containerName, []string{
		"dolt", "sql", "-q", query,
	})
	if err != nil {
		return nil // query failed — skip verification
	}

	dbProjectID := parseScalarResult(output)
	if dbProjectID == "" {
		return nil // no _project_id in db — skip verification
	}

	if meta.ProjectID != dbProjectID {
		return []string{
			fmt.Sprintf("project_id mismatch: local=%s, database=%s", meta.ProjectID, dbProjectID),
		}
	}

	return nil
}

// parseScalarResult extracts a single value from dolt sql tabular output.
func parseScalarResult(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "+") || strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "value") {
			continue
		}
		if strings.HasPrefix(trimmed, "|") {
			return strings.TrimSpace(strings.Trim(trimmed, "| "))
		}
	}
	return ""
}

// databaseExists checks whether a database with the given name exists on the
// shared Dolt server by running SHOW DATABASES and scanning the output.
func (m *Manager) databaseExists(ctx context.Context, dbName string) (bool, error) {
	output, err := m.backend.ContainerExec(ctx, containerName, []string{
		"dolt", "sql", "-q", "SHOW DATABASES",
	})
	if err != nil {
		return false, fmt.Errorf("show databases: %w", err)
	}

	for _, name := range ParseDatabaseNames(output) {
		if name == dbName {
			return true, nil
		}
	}

	return false, nil
}

// tarDirectory creates a tar archive of srcDir with entries rooted at prefix.
func tarDirectory(srcDir string, prefix string) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.Join(prefix, rel)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		return copyFileToTar(tw, path)
	})
	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func copyFileToTar(tw *tar.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = io.Copy(tw, f)
	return err
}

func extractFile(target string, mode int64, r io.Reader) error {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(mode))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = io.Copy(f, r)
	return err
}
