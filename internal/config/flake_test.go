package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/config"
)

func TestResolveFlake_FlagWins(t *testing.T) {
	flagEnv := "path:./custom"
	cfg, src := config.Resolve(
		config.Config{Env: "github:global/env"},
		config.Config{Env: "github:project/env"},
		config.Overrides{},
		config.Overrides{Env: &flagEnv},
	)

	got := config.ResolveFlake(cfg, src, true)
	assert.Equal(t, "path:./custom", got)
}

func TestResolveFlake_EnvVarWins(t *testing.T) {
	envVal := "github:envvar/env"
	cfg, src := config.Resolve(
		config.Config{Env: "github:global/env"},
		config.Config{Env: "github:project/env"},
		config.Overrides{Env: &envVal},
		config.Overrides{},
	)

	got := config.ResolveFlake(cfg, src, true)
	assert.Equal(t, "github:envvar/env", got)
}

func TestResolveFlake_ProjectConfigWins(t *testing.T) {
	cfg, src := config.Resolve(
		config.Config{Env: "github:global/env"},
		config.Config{Env: "github:project/env"},
		config.Overrides{},
		config.Overrides{},
	)

	got := config.ResolveFlake(cfg, src, true)
	assert.Equal(t, "github:project/env", got)
}

func TestResolveFlake_FlakeNixFallback(t *testing.T) {
	cfg, src := config.Resolve(
		config.Config{Env: "github:global/env"},
		config.Config{},
		config.Overrides{},
		config.Overrides{},
	)

	got := config.ResolveFlake(cfg, src, true)
	assert.Equal(t, "path:./.havn", got)
}

func TestResolveFlake_GlobalDefault(t *testing.T) {
	cfg, src := config.Resolve(
		config.Config{Env: "github:global/env"},
		config.Config{},
		config.Overrides{},
		config.Overrides{},
	)

	got := config.ResolveFlake(cfg, src, false)
	assert.Equal(t, "github:global/env", got)
}

func TestResolveFlake_DefaultWhenNothingSet(t *testing.T) {
	cfg, src := config.Resolve(
		config.Config{},
		config.Config{},
		config.Overrides{},
		config.Overrides{},
	)

	got := config.ResolveFlake(cfg, src, false)
	assert.Equal(t, "github:jorgengundersen/dev-environments", got)
}
