package config

// ResolveFlake determines the effective Nix flake reference using the 5-level
// priority chain defined in specs/havn-overview.md §'Dev environment flake
// resolution':
//
//  1. --env flag
//  2. HAVN_ENV env var
//  3. env in .havn/config.toml (project)
//  4. .havn/flake.nix (if exists → "path:./.havn")
//  5. env in ~/.config/havn/config.toml (global default)
//
// The caller supplies the resolved Config, its Source map, and whether
// .havn/flake.nix exists on disk.
func ResolveFlake(cfg Config, src Source, flakeExists bool) string {
	envSource := src["env"]

	// Levels 1-3: flag, env var, or project config explicitly set env.
	if envSource == "flag" || envSource == "env" || envSource == "project" {
		return cfg.Env
	}

	// Level 4: .havn/flake.nix exists.
	if flakeExists {
		return "path:./.havn"
	}

	// Level 5: global config or built-in default.
	return cfg.Env
}
