package config

import (
	"os"
	"strconv"
)

// Overrides carries explicitly-set values from flags or environment variables.
// Nil fields mean "not set" — they do not override lower-precedence values.
// Non-nil fields override regardless of their value (including zero).
type Overrides struct {
	Shell   *string
	Env     *string
	CPUs    *int
	Memory  *string
	SSHPort *string
	Image   *string
}

// Source maps config field names to the precedence level that provided the
// effective value: "default", "global", "project", "env", or "flag".
type Source map[string]string

// Resolve merges configuration from all precedence levels and returns the
// effective config plus a source map. Priority: flag > env > project > global > default.
func Resolve(global, project Config, envOverrides, flagOverrides Overrides) (Config, Source) {
	cfg := Default()
	src := Source{
		"env":         "default",
		"shell":       "default",
		"image":       "default",
		"network":     "default",
		"cpus":        "default",
		"memory":      "default",
		"memory_swap": "default",
	}

	// Layer 2: global config
	applyConfig(&cfg, global, "global", src)

	// Layer 3: project config
	applyConfig(&cfg, project, "project", src)

	// Layer 4: env var overrides
	applyOverrides(&cfg, envOverrides, "env", src)

	// Layer 5: flag overrides
	applyOverrides(&cfg, flagOverrides, "flag", src)

	return cfg, src
}

// applyConfig overlays non-zero fields from layer onto cfg.
func applyConfig(cfg *Config, layer Config, label string, src Source) {
	if layer.Env != "" {
		cfg.Env = layer.Env
		src["env"] = label
	}
	if layer.Shell != "" {
		cfg.Shell = layer.Shell
		src["shell"] = label
	}
	if layer.Image != "" {
		cfg.Image = layer.Image
		src["image"] = label
	}
	if layer.Network != "" {
		cfg.Network = layer.Network
		src["network"] = label
	}
	if layer.Resources.CPUs != 0 {
		cfg.Resources.CPUs = layer.Resources.CPUs
		src["cpus"] = label
	}
	if layer.Resources.Memory != "" {
		cfg.Resources.Memory = layer.Resources.Memory
		src["memory"] = label
	}
	if layer.Resources.MemorySwap != "" {
		cfg.Resources.MemorySwap = layer.Resources.MemorySwap
		src["memory_swap"] = label
	}
	if layer.Volumes.Nix != "" {
		cfg.Volumes.Nix = layer.Volumes.Nix
	}
	if layer.Volumes.Data != "" {
		cfg.Volumes.Data = layer.Volumes.Data
	}
	if layer.Volumes.Cache != "" {
		cfg.Volumes.Cache = layer.Volumes.Cache
	}
	if layer.Volumes.State != "" {
		cfg.Volumes.State = layer.Volumes.State
	}
	if layer.Dolt.Enabled {
		cfg.Dolt.Enabled = true
	}
	if layer.Dolt.Port != 0 {
		cfg.Dolt.Port = layer.Dolt.Port
	}
	if layer.Dolt.Image != "" {
		cfg.Dolt.Image = layer.Dolt.Image
	}
	if layer.Dolt.Database != "" {
		cfg.Dolt.Database = layer.Dolt.Database
	}
	// Mount config entries append rather than replace.
	if len(layer.Mounts.Config) > 0 {
		cfg.Mounts.Config = append(cfg.Mounts.Config, layer.Mounts.Config...)
	}
	// Port mappings append rather than replace.
	if len(layer.Ports) > 0 {
		cfg.Ports = append(cfg.Ports, layer.Ports...)
	}
	// Environment entries merge by key; later layers override duplicates.
	if len(layer.Environment) > 0 {
		if cfg.Environment == nil {
			cfg.Environment = make(map[string]string)
		}
		for k, v := range layer.Environment {
			cfg.Environment[k] = v
		}
	}
}

// applyOverrides overlays non-nil pointer fields from an Overrides onto cfg.
func applyOverrides(cfg *Config, ov Overrides, label string, src Source) {
	if ov.Shell != nil {
		cfg.Shell = *ov.Shell
		src["shell"] = label
	}
	if ov.Env != nil {
		cfg.Env = *ov.Env
		src["env"] = label
	}
	if ov.CPUs != nil {
		cfg.Resources.CPUs = *ov.CPUs
		src["cpus"] = label
	}
	if ov.Memory != nil {
		cfg.Resources.Memory = *ov.Memory
		src["memory"] = label
	}
	if ov.Image != nil {
		cfg.Image = *ov.Image
		src["image"] = label
	}
	if ov.SSHPort != nil {
		cfg.Ports = append(cfg.Ports, *ov.SSHPort+":22")
	}
}

// EnvOverrides reads the documented HAVN_* environment variables and returns
// an Overrides with non-nil fields for those that are set.
func EnvOverrides() Overrides {
	var ov Overrides
	if v, ok := os.LookupEnv("HAVN_SHELL"); ok {
		ov.Shell = &v
	}
	if v, ok := os.LookupEnv("HAVN_ENV"); ok {
		ov.Env = &v
	}
	if v, ok := os.LookupEnv("HAVN_CPUS"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			ov.CPUs = &n
		}
	}
	if v, ok := os.LookupEnv("HAVN_MEMORY"); ok {
		ov.Memory = &v
	}
	if v, ok := os.LookupEnv("HAVN_SSH_PORT"); ok {
		ov.SSHPort = &v
	}
	if v, ok := os.LookupEnv("HAVN_IMAGE"); ok {
		ov.Image = &v
	}
	return ov
}
