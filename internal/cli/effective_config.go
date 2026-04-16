package cli

import (
	"os"
	"path/filepath"

	"github.com/jorgengundersen/havn/internal/config"
)

type effectiveConfigOrchestrator struct {
	globalConfigPath string
}

func newEffectiveConfigOrchestrator(globalConfigPath string) effectiveConfigOrchestrator {
	return effectiveConfigOrchestrator{globalConfigPath: globalConfigPath}
}

func (o effectiveConfigOrchestrator) Resolve(projectCtx projectContext, flagOv config.Overrides) (config.Config, error) {
	cfg, _, err := o.ResolveWithSource(projectCtx, flagOv)
	if err != nil {
		return config.Config{}, err
	}

	return cfg, nil
}

func (o effectiveConfigOrchestrator) ResolveWithSource(projectCtx projectContext, flagOv config.Overrides) (config.Config, config.Source, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config.Config{}, nil, err
	}

	globalPath := o.globalConfigPath
	if globalPath == "" {
		globalPath = filepath.Join(homeDir, ".config", "havn", "config.toml")
	}

	global, globalMeta, err := config.LoadFileWithMetadata(globalPath)
	if err != nil {
		return config.Config{}, nil, err
	}

	project, projectMeta, err := config.LoadFileWithMetadata(projectCtx.ProjectConfigPath())
	if err != nil {
		return config.Config{}, nil, err
	}

	envOverrides, err := config.EnvOverrides()
	if err != nil {
		return config.Config{}, nil, err
	}

	cfg, src := config.ResolveWithMetadata(global, globalMeta, project, projectMeta, envOverrides, flagOv)

	discoveredFlakeRef := discoveredProjectFlakeRef(projectCtx)
	if discoveredFlakeRef != "" {
		cfg.Env = config.ResolveFlake(cfg, src, discoveredFlakeRef)
		if src["env"] == "default" || src["env"] == "global" {
			src["env"] = "project"
		}
	} else {
		cfg.Env = config.ResolveFlake(cfg, src, "")
	}

	if cfg.Dolt.Enabled && cfg.Dolt.Database == "" {
		cfg.Dolt.Database = projectCtx.DefaultDoltDatabase()
	}

	if err := config.Validate(cfg); err != nil {
		return config.Config{}, nil, err
	}

	return cfg, src, nil
}

func discoveredProjectFlakeRef(projectCtx projectContext) string {
	candidates := []struct {
		path string
		ref  string
	}{
		{path: projectCtx.ProjectFlakePath(), ref: "path:./.havn"},
		{path: projectCtx.ProjectDefaultEnvironmentFlakePath(), ref: "path:./.havn/environments/default"},
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate.path); err == nil {
			return candidate.ref
		}
	}

	return ""
}
