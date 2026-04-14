package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
)

func TestLoadFile_ValidTOML(t *testing.T) {
	cfg, err := config.LoadFile("testdata/valid.toml")

	assert.NoError(t, err)
	assert.Equal(t, "github:user/custom-env", cfg.Env)
	assert.Equal(t, "go", cfg.Shell)
	assert.Equal(t, "my-image:v2", cfg.Image)
	assert.Equal(t, "my-net", cfg.Network)
	assert.Equal(t, []string{"8080:8080", "3000:3000"}, cfg.Ports)

	assert.Equal(t, 8, cfg.Resources.CPUs)
	assert.Equal(t, "16g", cfg.Resources.Memory)
	assert.Equal(t, "24g", cfg.Resources.MemorySwap)

	assert.Equal(t, "custom-nix", cfg.Volumes.Nix)
	assert.Equal(t, "custom-data", cfg.Volumes.Data)
	assert.Equal(t, "custom-cache", cfg.Volumes.Cache)
	assert.Equal(t, "custom-state", cfg.Volumes.State)

	assert.Equal(t, []string{".gitconfig:ro", ".config/nvim/:ro"}, cfg.Mounts.Config)
	assert.False(t, cfg.Mounts.SSH.ForwardAgent)
	assert.False(t, cfg.Mounts.SSH.AuthorizedKeys)

	assert.True(t, cfg.Dolt.Enabled)
	assert.Equal(t, 3307, cfg.Dolt.Port)
	assert.Equal(t, "dolthub/dolt-sql-server:v1.0", cfg.Dolt.Image)
	assert.Equal(t, "myproject", cfg.Dolt.Database)

	assert.Equal(t, map[string]string{
		"MY_API_KEY": "${MY_API_KEY}",
		"DEBUG":      "true",
	}, cfg.Environment)
}

func TestLoadFile_MalformedTOML(t *testing.T) {
	cfg, err := config.LoadFile("testdata/invalid.toml")

	assert.Equal(t, config.Config{}, cfg)

	var parseErr *config.ParseError
	assert.ErrorAs(t, err, &parseErr)
	assert.Equal(t, "testdata/invalid.toml", parseErr.File)
	assert.Greater(t, parseErr.Line, 0)
	assert.NotEmpty(t, parseErr.Detail)
}

func TestLoadFile_MissingFile(t *testing.T) {
	cfg, err := config.LoadFile("testdata/nonexistent.toml")

	assert.NoError(t, err)
	assert.Equal(t, config.Config{}, cfg)
}

func TestConfig_TOMLRoundTrip(t *testing.T) {
	original := config.Config{
		Env:     "github:user/env",
		Shell:   "go",
		Image:   "my-image:v1",
		Network: "my-net",
		Ports:   []string{"8080:8080"},
		Resources: config.ResourceConfig{
			CPUs:       8,
			Memory:     "16g",
			MemorySwap: "24g",
		},
		Volumes: config.VolumeConfig{
			Nix:   "vol-nix",
			Data:  "vol-data",
			Cache: "vol-cache",
			State: "vol-state",
		},
		Mounts: config.MountConfig{
			Config: []string{".gitconfig:ro"},
			SSH: config.SSHConfig{
				ForwardAgent:   true,
				AuthorizedKeys: true,
			},
		},
		Dolt: config.DoltConfig{
			Enabled:  true,
			Port:     3307,
			Image:    "dolthub/dolt-sql-server:v1",
			Database: "testdb",
		},
		Environment: map[string]string{"KEY": "value"},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.toml")

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, toml.NewEncoder(f).Encode(original))
	require.NoError(t, f.Close())

	loaded, err := config.LoadFile(path)
	require.NoError(t, err)

	assert.Equal(t, original, loaded)
}

func TestDefault_ReturnsSpecValues(t *testing.T) {
	cfg := config.Default()

	assert.Equal(t, "path:.", cfg.Env)
	assert.Equal(t, "default", cfg.Shell)
	assert.Equal(t, "havn-base:latest", cfg.Image)
	assert.Equal(t, "havn-net", cfg.Network)

	assert.Equal(t, 4, cfg.Resources.CPUs)
	assert.Equal(t, "8g", cfg.Resources.Memory)
	assert.Equal(t, "12g", cfg.Resources.MemorySwap)

	assert.Equal(t, "havn-nix", cfg.Volumes.Nix)
	assert.Equal(t, "havn-data", cfg.Volumes.Data)
	assert.Equal(t, "havn-cache", cfg.Volumes.Cache)
	assert.Equal(t, "havn-state", cfg.Volumes.State)

	assert.True(t, cfg.Mounts.SSH.ForwardAgent)
	assert.True(t, cfg.Mounts.SSH.AuthorizedKeys)

	assert.False(t, cfg.Dolt.Enabled)
	assert.Equal(t, 3308, cfg.Dolt.Port)
	assert.Equal(t, "dolthub/dolt-sql-server:latest", cfg.Dolt.Image)
}
