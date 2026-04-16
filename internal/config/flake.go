package config

// ResolveFlake determines the effective Nix flake reference using the priority
// chain defined in specs/configuration.md:
//
//  1. --env flag
//  2. HAVN_ENV env var
//  3. env in .havn/config.toml (project)
//  4. discovered project flake (if present)
//  5. env in ~/.config/havn/config.toml (global default)
//
// The caller supplies the resolved Config, its Source map, and whether
// a discovered project flake reference exists on disk. Current discovered refs
// are path:./.havn and path:./.havn/environments/default.
func ResolveFlake(cfg Config, src Source, discoveredFlakeRef string) string {
	envSource := src["env"]

	// Levels 1-3: flag, env var, or project config explicitly set env.
	if envSource == "flag" || envSource == "env" || envSource == "project" {
		return cfg.Env
	}

	// Level 4: discovered project flake exists.
	if discoveredFlakeRef != "" {
		return discoveredFlakeRef
	}

	// Level 5: global config or built-in default.
	return cfg.Env
}
