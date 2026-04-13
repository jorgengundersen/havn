package dolt

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/jorgengundersen/havn/internal/config"
)

// Setup orchestrates the Dolt server readiness flow for project container
// startup. It ensures the server is running, the project database exists,
// and returns the beads env vars to inject into the project container.
type Setup struct {
	manager *Manager
	backend Backend
}

// NewSetup creates a Setup with the given manager and backend.
func NewSetup(manager *Manager, backend Backend) *Setup {
	return &Setup{manager: manager, backend: backend}
}

// EnsureReady ensures the Dolt server is running and the project database
// exists. Returns the beads env vars to inject into the project container.
func (s *Setup) EnsureReady(ctx context.Context, cfg config.Config) (map[string]string, error) {
	if err := s.manager.Start(ctx, cfg); err != nil {
		return nil, err
	}

	if err := s.ensureDatabase(ctx, cfg.Dolt.Database); err != nil {
		return nil, err
	}

	return map[string]string{
		"BEADS_DOLT_SERVER_HOST":     containerName,
		"BEADS_DOLT_SERVER_PORT":     strconv.Itoa(cfg.Dolt.Port),
		"BEADS_DOLT_SERVER_USER":     "root",
		"BEADS_DOLT_AUTO_START":      "0",
		"BEADS_DOLT_SHARED_SERVER":   "1",
		"BEADS_DOLT_SERVER_DATABASE": cfg.Dolt.Database,
	}, nil
}

// MigrationNotice checks for a local beads database that can be migrated and
// returns a user-facing migration message when applicable.
func (s *Setup) MigrationNotice(ctx context.Context, cfg config.Config, projectPath string) (string, error) {
	hint, err := s.DetectMigration(ctx, cfg, projectPath, pathExists)
	if err != nil {
		return "", err
	}
	if hint == nil {
		return "", nil
	}
	return fmt.Sprintf("Found local beads database at %s for %q; migrate with: havn dolt import %s", hint.LocalPath, hint.DatabaseName, projectPath), nil
}

func pathExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (s *Setup) ensureDatabase(ctx context.Context, name string) error {
	if err := validateDatabaseIdentifier(name); err != nil {
		return err
	}

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", name)
	_, err := s.backend.ContainerExec(ctx, containerName, []string{
		"dolt", "sql", "-q", query,
	})
	if err != nil {
		return &DatabaseCreateError{Name: name, Err: err}
	}
	return nil
}
