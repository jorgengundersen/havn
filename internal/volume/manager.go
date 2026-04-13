package volume

import (
	"context"
	"errors"
	"fmt"

	"github.com/jorgengundersen/havn/internal/config"
)

// Manager provides volume lifecycle operations backed by a Backend.
type Manager struct {
	backend Backend
}

// NewManager creates a Manager with the given backend.
func NewManager(backend Backend) *Manager {
	return &Manager{backend: backend}
}

// List returns all expected volumes with their existence status populated
// by querying the backend.
func (m *Manager) List(ctx context.Context, cfg config.Config) ([]Entry, error) {
	entries := ExpectedVolumes(cfg)
	for i := range entries {
		err := m.backend.VolumeInspect(ctx, entries[i].Name)
		if err == nil {
			entries[i].Exists = true
		}
	}
	return entries, nil
}

// EnsureExists creates a volume if it does not already exist. Idempotent.
func (m *Manager) EnsureExists(ctx context.Context, name string) error {
	if err := m.backend.VolumeInspect(ctx, name); err == nil {
		return nil
	} else {
		var notFound *NotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("inspect volume %q: %w", name, err)
		}
	}
	if err := m.backend.VolumeCreate(ctx, name); err != nil {
		return fmt.Errorf("ensure volume %q: %w", name, err)
	}
	return nil
}

// ExpectedVolumes returns the volume registry derived from the given config.
// Mount paths are fixed constants; volume names come from cfg.Volumes.
// Dolt volumes are included only when cfg.Dolt.Enabled is true.
func ExpectedVolumes(cfg config.Config) []Entry {
	entries := []Entry{
		{Name: cfg.Volumes.Nix, Mount: "/nix"},
		{Name: cfg.Volumes.Data, Mount: "/home/devuser/.local/share"},
		{Name: cfg.Volumes.Cache, Mount: "/home/devuser/.cache"},
		{Name: cfg.Volumes.State, Mount: "/home/devuser/.local/state"},
	}

	if cfg.Dolt.Enabled {
		entries = append(entries,
			Entry{Name: "havn-dolt-data", Mount: "/var/lib/dolt"},
			Entry{Name: "havn-dolt-config", Mount: "/etc/dolt/servercfg.d"},
		)
	}

	return entries
}
