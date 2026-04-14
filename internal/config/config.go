// Package config defines the havn configuration types and TOML loading.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config represents the full havn configuration, combining global and
// project-level settings. Fields use toml tags matching the schema in
// specs/configuration.md.
type Config struct {
	Env       string         `toml:"env"`
	Shell     string         `toml:"shell"`
	Image     string         `toml:"image"`
	Network   string         `toml:"network"`
	Ports     []string       `toml:"ports"`
	Resources ResourceConfig `toml:"resources"`
	Volumes   VolumeConfig   `toml:"volumes"`
	Mounts    MountConfig    `toml:"mounts"`
	Dolt      DoltConfig     `toml:"dolt"`

	Environment map[string]string `toml:"environment"`
}

// ResourceConfig controls container resource limits.
type ResourceConfig struct {
	CPUs       int    `toml:"cpus" json:"cpus"`
	Memory     string `toml:"memory" json:"memory"`
	MemorySwap string `toml:"memory_swap" json:"memory_swap"`
}

// VolumeConfig maps logical volume roles to Docker volume names.
type VolumeConfig struct {
	Nix   string `toml:"nix" json:"nix"`
	Data  string `toml:"data" json:"data"`
	Cache string `toml:"cache" json:"cache"`
	State string `toml:"state" json:"state"`
}

// MountConfig describes host bind mounts and SSH forwarding.
type MountConfig struct {
	Config []string  `toml:"config"`
	SSH    SSHConfig `toml:"ssh"`
}

// SSHConfig controls SSH agent and key forwarding into the container.
type SSHConfig struct {
	ForwardAgent   bool `toml:"forward_agent" json:"forward_agent"`
	AuthorizedKeys bool `toml:"authorized_keys" json:"authorized_keys"`
}

// DoltConfig controls the shared Dolt SQL server.
type DoltConfig struct {
	Enabled  bool   `toml:"enabled" json:"enabled"`
	Port     int    `toml:"port" json:"port"`
	Image    string `toml:"image" json:"image"`
	Database string `toml:"database" json:"database"`
}

// FileMetadata captures which config keys were explicitly defined in a TOML
// file, including explicit false booleans.
type FileMetadata struct {
	MountSSHForwardAgentSet   bool
	MountSSHAuthorizedKeysSet bool
	DoltEnabledSet            bool
}

// LoadFile parses a TOML file into a Config. A missing file returns a zero
// Config and nil error (not an error per havn-doctor §1.5). A parse error
// returns *ParseError with file path and line number.
func LoadFile(path string) (Config, error) {
	cfg, _, err := LoadFileWithMetadata(path)
	return cfg, err
}

// LoadFileWithMetadata parses a TOML file into a Config and returns metadata
// about explicitly-defined keys needed for precedence-sensitive boolean merges.
func LoadFileWithMetadata(path string) (Config, FileMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, FileMetadata{}, nil
		}
		return Config{}, FileMetadata{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		var parseErr toml.ParseError
		if errors.As(err, &parseErr) {
			return Config{}, FileMetadata{}, &ParseError{
				File:   path,
				Line:   parseErr.Position.Line,
				Detail: parseErr.Message,
			}
		}
		return Config{}, FileMetadata{}, &ParseError{
			File:   path,
			Line:   0,
			Detail: err.Error(),
		}
	}

	meta := FileMetadata{
		MountSSHForwardAgentSet:   md.IsDefined("mounts", "ssh", "forward_agent"),
		MountSSHAuthorizedKeysSet: md.IsDefined("mounts", "ssh", "authorized_keys"),
		DoltEnabledSet:            md.IsDefined("dolt", "enabled"),
	}

	return cfg, meta, nil
}

// Default returns a Config populated with all spec-defined default values.
func Default() Config {
	return Config{
		Env:     "path:.",
		Shell:   "default",
		Image:   "havn-base:latest",
		Network: "havn-net",
		Resources: ResourceConfig{
			CPUs:       4,
			Memory:     "8g",
			MemorySwap: "12g",
		},
		Volumes: VolumeConfig{
			Nix:   "havn-nix",
			Data:  "havn-data",
			Cache: "havn-cache",
			State: "havn-state",
		},
		Mounts: MountConfig{
			SSH: SSHConfig{
				ForwardAgent:   true,
				AuthorizedKeys: true,
			},
		},
		Dolt: DoltConfig{
			Enabled: false,
			Port:    3308,
			Image:   "dolthub/dolt-sql-server:latest",
		},
	}
}
