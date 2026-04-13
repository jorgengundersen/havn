package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jorgengundersen/havn/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage havn configuration",
	}

	cmd.AddCommand(newConfigShowCmd())

	return cmd
}

// configShowOutput is the JSON output shape for havn config show --json.
// It embeds the effective config fields and adds a source map.
type configShowOutput struct {
	Env         string                `json:"env"`
	Shell       string                `json:"shell"`
	Image       string                `json:"image"`
	Network     string                `json:"network"`
	Environment map[string]string     `json:"environment,omitempty"`
	Ports       []string              `json:"ports,omitempty"`
	Resources   config.ResourceConfig `json:"resources"`
	Volumes     config.VolumeConfig   `json:"volumes"`
	Mounts      mountOutput           `json:"mounts"`
	Dolt        config.DoltConfig     `json:"dolt"`
	Source      map[string]any        `json:"source"`
}

type mountOutput struct {
	Config []string         `json:"config,omitempty"`
	SSH    config.SSHConfig `json:"ssh"`
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show effective merged config",
		RunE:  runConfigShow,
	}
}

func runConfigShow(cmd *cobra.Command, _ []string) error {
	jsonMode, _ := cmd.Flags().GetBool("json")
	verbose, _ := cmd.Flags().GetBool("verbose")
	out := NewOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, verbose)
	globalPath, _ := cmd.Flags().GetString("config")

	if globalPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("config show: %w", err)
		}
		globalPath = filepath.Join(homeDir, ".config", "havn", "config.toml")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("config show: %w", err)
	}

	projectPath := filepath.Join(cwd, ".havn", "config.toml")
	flakePath := filepath.Join(cwd, ".havn", "flake.nix")

	global, globalMeta, err := config.LoadFileWithMetadata(globalPath)
	if err != nil {
		return fmt.Errorf("config show: %w", err)
	}
	project, projectMeta, err := config.LoadFileWithMetadata(projectPath)
	if err != nil {
		return fmt.Errorf("config show: %w", err)
	}

	envOv := config.EnvOverrides()
	cfg, src := config.ResolveWithMetadata(global, globalMeta, project, projectMeta, envOv, config.Overrides{})
	if _, err := os.Stat(flakePath); err == nil {
		cfg.Env = config.ResolveFlake(cfg, src, true)
		if src["env"] == "default" || src["env"] == "global" {
			src["env"] = "project"
		}
	} else {
		cfg.Env = config.ResolveFlake(cfg, src, false)
	}

	if cfg.Dolt.Enabled && cfg.Dolt.Database == "" {
		cfg.Dolt.Database = filepath.Base(cwd)
	}

	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("config show: %w", err)
	}

	if out.IsJSON() {
		return out.DataJSON(configShowOutput{
			Env:         cfg.Env,
			Shell:       cfg.Shell,
			Image:       cfg.Image,
			Network:     cfg.Network,
			Environment: cfg.Environment,
			Ports:       cfg.Ports,
			Resources:   cfg.Resources,
			Volumes:     cfg.Volumes,
			Mounts: mountOutput{
				Config: cfg.Mounts.Config,
				SSH:    cfg.Mounts.SSH,
			},
			Dolt:   cfg.Dolt,
			Source: sourceOutput(src),
		})
	}

	out.Data(formatConfigHuman(cfg, src))
	return nil
}

func sourceOutput(src config.Source) map[string]any {
	return map[string]any{
		"env":     sourceValue(src, "env"),
		"shell":   sourceValue(src, "shell"),
		"image":   sourceValue(src, "image"),
		"network": sourceValue(src, "network"),
		"resources": map[string]string{
			"cpus":        sourceValue(src, "cpus"),
			"memory":      sourceValue(src, "memory"),
			"memory_swap": sourceValue(src, "memory_swap"),
		},
		"dolt": map[string]string{
			"enabled":  sourceValue(src, "dolt_enabled"),
			"database": sourceValue(src, "dolt_database"),
			"port":     sourceValue(src, "dolt_port"),
			"image":    sourceValue(src, "dolt_image"),
		},
	}
}

func sourceValue(src config.Source, key string) string {
	origin := src[key]
	if origin == "" {
		return "default"
	}
	return origin
}

func formatConfigHuman(cfg config.Config, src config.Source) string {
	var b strings.Builder

	writeField := func(label, value, field string) {
		origin := src[field]
		if origin == "" {
			origin = "default"
		}
		fmt.Fprintf(&b, "  %-14s %s  (%s)\n", label+":", value, origin)
	}

	b.WriteString("Configuration:\n")
	writeField("env", cfg.Env, "env")
	writeField("shell", cfg.Shell, "shell")
	writeField("image", cfg.Image, "image")
	writeField("network", cfg.Network, "network")

	b.WriteString("\nResources:\n")
	writeField("cpus", fmt.Sprintf("%d", cfg.Resources.CPUs), "cpus")
	writeField("memory", cfg.Resources.Memory, "memory")
	writeField("memory_swap", cfg.Resources.MemorySwap, "memory_swap")

	b.WriteString("\nVolumes:\n")
	fmt.Fprintf(&b, "  %-14s %s\n", "nix:", cfg.Volumes.Nix)
	fmt.Fprintf(&b, "  %-14s %s\n", "data:", cfg.Volumes.Data)
	fmt.Fprintf(&b, "  %-14s %s\n", "cache:", cfg.Volumes.Cache)
	fmt.Fprintf(&b, "  %-14s %s\n", "state:", cfg.Volumes.State)

	if len(cfg.Mounts.Config) > 0 {
		b.WriteString("\nMounts:\n")
		for _, m := range cfg.Mounts.Config {
			fmt.Fprintf(&b, "  - %s\n", m)
		}
	}

	b.WriteString("\nSSH:\n")
	fmt.Fprintf(&b, "  %-14s %v\n", "forward_agent:", cfg.Mounts.SSH.ForwardAgent)
	fmt.Fprintf(&b, "  %-14s %v\n", "authorized_keys:", cfg.Mounts.SSH.AuthorizedKeys)

	b.WriteString("\nDolt:\n")
	fmt.Fprintf(&b, "  %-14s %v\n", "enabled:", cfg.Dolt.Enabled)
	fmt.Fprintf(&b, "  %-14s %d\n", "port:", cfg.Dolt.Port)
	fmt.Fprintf(&b, "  %-14s %s\n", "image:", cfg.Dolt.Image)
	fmt.Fprintf(&b, "  %-14s %s\n", "database:", cfg.Dolt.Database)

	if len(cfg.Environment) > 0 {
		b.WriteString("\nEnvironment:\n")
		for k, v := range cfg.Environment {
			fmt.Fprintf(&b, "  %-14s %s\n", k+":", v)
		}
	}

	if len(cfg.Ports) > 0 {
		b.WriteString("\nPorts:\n")
		for _, p := range cfg.Ports {
			fmt.Fprintf(&b, "  - %s\n", p)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}
