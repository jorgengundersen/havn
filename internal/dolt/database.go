package dolt

import (
	"context"
	"fmt"
	"strings"
)

// systemDatabases lists database names that SHOW DATABASES returns but are
// not user-created. Filtered out by Databases.
var systemDatabases = map[string]bool{
	"information_schema": true,
	"mysql":              true,
}

// Databases returns the names of user-created databases on the shared Dolt
// server, excluding system databases.
func (m *Manager) Databases(ctx context.Context) ([]string, error) {
	if err := m.ensureRunningManaged(ctx); err != nil {
		return nil, err
	}

	output, err := m.backend.ContainerExec(ctx, containerName, []string{
		"dolt", "sql", "-q", "SHOW DATABASES",
	})
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}

	return parseDatabaseNames(output), nil
}

// Drop executes DROP DATABASE on the shared Dolt server.
// No confirmation logic here — the CLI layer enforces --yes.
func (m *Manager) Drop(ctx context.Context, name string) error {
	if err := m.ensureRunningManaged(ctx); err != nil {
		return err
	}

	if err := validateDatabaseIdentifier(name); err != nil {
		return err
	}

	query := fmt.Sprintf("DROP DATABASE `%s`", name)
	_, err := m.backend.ContainerExec(ctx, containerName, []string{
		"dolt", "sql", "-q", query,
	})
	if err != nil {
		return fmt.Errorf("drop database %q: %w", name, err)
	}
	return nil
}

// Connect opens an interactive SQL shell on the shared Dolt server.
func (m *Manager) Connect(ctx context.Context) error {
	if err := m.ensureRunningManaged(ctx); err != nil {
		return err
	}

	return m.backend.ContainerExecInteractive(ctx, containerName, []string{"dolt", "sql"})
}

// parseDatabaseNames extracts database names from dolt sql tabular output,
// filtering out system databases.
func parseDatabaseNames(output string) []string {
	var names []string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "+") {
			continue
		}
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		name := strings.TrimSpace(strings.Trim(trimmed, "|"))
		if name == "Database" || name == "" {
			continue
		}
		if systemDatabases[name] {
			continue
		}
		names = append(names, name)
	}
	return names
}
