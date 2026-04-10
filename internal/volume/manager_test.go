package volume_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/volume"
)

// fakeBackend implements volume.Backend for testing.
type fakeBackend struct {
	existing  map[string]bool
	created   []string
	inspectFn func(name string) error
}

func (f *fakeBackend) VolumeInspect(_ context.Context, name string) error {
	if f.inspectFn != nil {
		return f.inspectFn(name)
	}
	if f.existing[name] {
		return nil
	}
	return fmt.Errorf("volume %q not found", name)
}

func (f *fakeBackend) VolumeCreate(_ context.Context, name string) error {
	f.created = append(f.created, name)
	return nil
}

func TestExpectedVolumes_DoltDisabled(t *testing.T) {
	cfg := config.Config{
		Volumes: config.VolumeConfig{
			Nix:   "havn-nix",
			Data:  "havn-data",
			Cache: "havn-cache",
			State: "havn-state",
		},
		Dolt: config.DoltConfig{Enabled: false},
	}

	got := volume.ExpectedVolumes(cfg)

	assert.Len(t, got, 4)
	assert.Equal(t, volume.Entry{Name: "havn-nix", Mount: "/nix"}, got[0])
	assert.Equal(t, volume.Entry{Name: "havn-data", Mount: "/home/devuser/.local/share"}, got[1])
	assert.Equal(t, volume.Entry{Name: "havn-cache", Mount: "/home/devuser/.cache"}, got[2])
	assert.Equal(t, volume.Entry{Name: "havn-state", Mount: "/home/devuser/.local/state"}, got[3])
}

func TestExpectedVolumes_DoltEnabled(t *testing.T) {
	cfg := config.Config{
		Volumes: config.VolumeConfig{
			Nix:   "havn-nix",
			Data:  "havn-data",
			Cache: "havn-cache",
			State: "havn-state",
		},
		Dolt: config.DoltConfig{Enabled: true},
	}

	got := volume.ExpectedVolumes(cfg)

	assert.Len(t, got, 6)
	assert.Equal(t, volume.Entry{Name: "havn-dolt-data", Mount: "/var/lib/dolt"}, got[4])
	assert.Equal(t, volume.Entry{Name: "havn-dolt-config", Mount: "/etc/dolt/servercfg.d"}, got[5])
}

func TestList_AllExist(t *testing.T) {
	backend := &fakeBackend{
		existing: map[string]bool{
			"havn-nix": true, "havn-data": true,
			"havn-cache": true, "havn-state": true,
		},
	}
	mgr := volume.NewManager(backend)
	cfg := config.Config{
		Volumes: config.VolumeConfig{
			Nix: "havn-nix", Data: "havn-data",
			Cache: "havn-cache", State: "havn-state",
		},
	}

	got, err := mgr.List(context.Background(), cfg)

	require.NoError(t, err)
	assert.Len(t, got, 4)
	for _, e := range got {
		assert.True(t, e.Exists, "expected %q to exist", e.Name)
	}
}

func TestList_SomeMissing(t *testing.T) {
	backend := &fakeBackend{
		existing: map[string]bool{
			"havn-nix":  true,
			"havn-data": true,
		},
	}
	mgr := volume.NewManager(backend)
	cfg := config.Config{
		Volumes: config.VolumeConfig{
			Nix: "havn-nix", Data: "havn-data",
			Cache: "havn-cache", State: "havn-state",
		},
	}

	got, err := mgr.List(context.Background(), cfg)

	require.NoError(t, err)
	assert.True(t, got[0].Exists, "havn-nix should exist")
	assert.True(t, got[1].Exists, "havn-data should exist")
	assert.False(t, got[2].Exists, "havn-cache should not exist")
	assert.False(t, got[3].Exists, "havn-state should not exist")
}

func TestEnsureExists_AlreadyExists(t *testing.T) {
	backend := &fakeBackend{
		existing: map[string]bool{"havn-nix": true},
	}
	mgr := volume.NewManager(backend)

	err := mgr.EnsureExists(context.Background(), "havn-nix")

	assert.NoError(t, err)
	assert.Empty(t, backend.created, "should not create an existing volume")
}

func TestEnsureExists_Missing(t *testing.T) {
	backend := &fakeBackend{existing: map[string]bool{}}
	mgr := volume.NewManager(backend)

	err := mgr.EnsureExists(context.Background(), "havn-nix")

	assert.NoError(t, err)
	assert.Equal(t, []string{"havn-nix"}, backend.created)
}

func TestEnsureExists_CreateError(t *testing.T) {
	mgr := volume.NewManager(&failingCreateBackend{})

	err := mgr.EnsureExists(context.Background(), "havn-nix")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ensure volume")
}

type failingCreateBackend struct{}

func (f *failingCreateBackend) VolumeInspect(_ context.Context, _ string) error {
	return fmt.Errorf("not found")
}

func (f *failingCreateBackend) VolumeCreate(_ context.Context, _ string) error {
	return fmt.Errorf("permission denied")
}
