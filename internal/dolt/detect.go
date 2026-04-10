package dolt

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jorgengundersen/havn/internal/config"
)

// MigrationHint holds the result of automatic migration detection.
// A non-nil hint means a local beads database exists that could be
// migrated to the shared Dolt server.
type MigrationHint struct {
	DatabaseName string
	LocalPath    string
}

// DetectMigration checks whether the project has a local beads database
// that could be migrated to the shared Dolt server. Returns nil when no
// migration is applicable. The server must be running before calling this.
func (s *Setup) DetectMigration(ctx context.Context, cfg config.Config, projectPath string, dirExists func(string) bool) (*MigrationHint, error) {
	dbName := cfg.Dolt.Database
	doltPath := filepath.Join(projectPath, ".beads", "dolt", dbName, ".dolt")

	if !dirExists(doltPath) {
		return nil, nil
	}

	databases, err := s.manager.Databases(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect migration: %w", err)
	}

	for _, db := range databases {
		if db == dbName {
			return nil, nil
		}
	}

	return &MigrationHint{
		DatabaseName: dbName,
		LocalPath:    filepath.Join(".beads", "dolt", dbName),
	}, nil
}
