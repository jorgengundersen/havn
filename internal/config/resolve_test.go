package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/config"
)

func TestResolve_DefaultsWhenAllInputsZero(t *testing.T) {
	cfg, src := config.Resolve(
		config.Config{},
		config.Config{},
		config.Overrides{},
		config.Overrides{},
	)

	defaults := config.Default()
	assert.Equal(t, defaults, cfg)

	// All fields should be sourced from "default"
	assert.Equal(t, "default", src["shell"])
	assert.Equal(t, "default", src["env"])
	assert.Equal(t, "default", src["image"])
	assert.Equal(t, "default", src["network"])
	assert.Equal(t, "default", src["cpus"])
	assert.Equal(t, "default", src["memory"])
	assert.Equal(t, "default", src["memory_swap"])
}

func TestResolve_GlobalOverridesDefault(t *testing.T) {
	global := config.Config{
		Shell: "zsh",
		Env:   "github:custom/env",
		Resources: config.ResourceConfig{
			CPUs: 8,
		},
	}

	cfg, src := config.Resolve(global, config.Config{}, config.Overrides{}, config.Overrides{})

	assert.Equal(t, "zsh", cfg.Shell)
	assert.Equal(t, "github:custom/env", cfg.Env)
	assert.Equal(t, 8, cfg.Resources.CPUs)
	// Unset fields retain defaults
	assert.Equal(t, "havn-base:latest", cfg.Image)

	assert.Equal(t, "global", src["shell"])
	assert.Equal(t, "global", src["env"])
	assert.Equal(t, "global", src["cpus"])
	assert.Equal(t, "default", src["image"])
}

func TestResolve_ProjectOverridesGlobal(t *testing.T) {
	global := config.Config{
		Shell: "zsh",
		Resources: config.ResourceConfig{
			CPUs:   8,
			Memory: "16g",
		},
	}
	project := config.Config{
		Shell: "go",
		Resources: config.ResourceConfig{
			CPUs: 16,
		},
	}

	cfg, src := config.Resolve(global, project, config.Overrides{}, config.Overrides{})

	assert.Equal(t, "go", cfg.Shell)
	assert.Equal(t, 16, cfg.Resources.CPUs)
	// Global memory not overridden by project
	assert.Equal(t, "16g", cfg.Resources.Memory)

	assert.Equal(t, "project", src["shell"])
	assert.Equal(t, "project", src["cpus"])
	assert.Equal(t, "global", src["memory"])
}

func TestResolve_EnvOverridesProject(t *testing.T) {
	project := config.Config{
		Shell: "go",
		Resources: config.ResourceConfig{
			CPUs: 16,
		},
	}
	shell := "rust"
	envOv := config.Overrides{Shell: &shell}

	cfg, src := config.Resolve(config.Config{}, project, envOv, config.Overrides{})

	assert.Equal(t, "rust", cfg.Shell)
	assert.Equal(t, 16, cfg.Resources.CPUs)

	assert.Equal(t, "env", src["shell"])
	assert.Equal(t, "project", src["cpus"])
}

func TestResolve_FlagOverridesEnv(t *testing.T) {
	shell := "rust"
	envOv := config.Overrides{Shell: &shell}
	flagShell := "python"
	flagOv := config.Overrides{Shell: &flagShell}

	cfg, src := config.Resolve(config.Config{}, config.Config{}, envOv, flagOv)

	assert.Equal(t, "python", cfg.Shell)
	assert.Equal(t, "flag", src["shell"])
}

func TestResolve_ProjectMountsAppendToGlobal(t *testing.T) {
	global := config.Config{
		Mounts: config.MountConfig{
			Config: []string{".gitconfig:ro"},
		},
	}
	project := config.Config{
		Mounts: config.MountConfig{
			Config: []string{".config/nvim/:ro"},
		},
	}

	cfg, _ := config.Resolve(global, project, config.Overrides{}, config.Overrides{})

	assert.Equal(t, []string{".gitconfig:ro", ".config/nvim/:ro"}, cfg.Mounts.Config)
}

func TestResolve_SSHPortOverrideAddsToPorts(t *testing.T) {
	project := config.Config{
		Ports: []string{"8080:8080"},
	}
	sshPort := "2222:22"
	flagOv := config.Overrides{SSHPort: &sshPort}

	cfg, _ := config.Resolve(config.Config{}, project, config.Overrides{}, flagOv)

	assert.Contains(t, cfg.Ports, "2222:22")
	assert.Contains(t, cfg.Ports, "8080:8080")
}

func TestEnvOverrides_ReadsSetVars(t *testing.T) {
	t.Setenv("HAVN_SHELL", "rust")
	t.Setenv("HAVN_ENV", "path:./my-flake")
	t.Setenv("HAVN_CPUS", "16")
	t.Setenv("HAVN_MEMORY", "32g")
	t.Setenv("HAVN_SSH_PORT", "2222:22")
	t.Setenv("HAVN_IMAGE", "custom:v2")

	ov := config.EnvOverrides()

	assert.NotNil(t, ov.Shell)
	assert.Equal(t, "rust", *ov.Shell)
	assert.NotNil(t, ov.Env)
	assert.Equal(t, "path:./my-flake", *ov.Env)
	assert.NotNil(t, ov.CPUs)
	assert.Equal(t, 16, *ov.CPUs)
	assert.NotNil(t, ov.Memory)
	assert.Equal(t, "32g", *ov.Memory)
	assert.NotNil(t, ov.SSHPort)
	assert.Equal(t, "2222:22", *ov.SSHPort)
	assert.NotNil(t, ov.Image)
	assert.Equal(t, "custom:v2", *ov.Image)
}

func TestEnvOverrides_IgnoresUnsetVars(t *testing.T) {
	ov := config.EnvOverrides()

	assert.Nil(t, ov.Shell)
	assert.Nil(t, ov.Env)
	assert.Nil(t, ov.CPUs)
	assert.Nil(t, ov.Memory)
	assert.Nil(t, ov.SSHPort)
	assert.Nil(t, ov.Image)
}

func TestResolve_NilOverrideFieldsDoNotOverride(t *testing.T) {
	project := config.Config{
		Shell: "go",
		Resources: config.ResourceConfig{
			CPUs: 8,
		},
	}
	// Only Shell is set in env overrides; CPUs is nil
	shell := "rust"
	envOv := config.Overrides{Shell: &shell}

	cfg, src := config.Resolve(config.Config{}, project, envOv, config.Overrides{})

	assert.Equal(t, "rust", cfg.Shell)
	assert.Equal(t, 8, cfg.Resources.CPUs)
	assert.Equal(t, "env", src["shell"])
	assert.Equal(t, "project", src["cpus"])
}
