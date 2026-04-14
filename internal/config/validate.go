package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	sizeFormatRe  = regexp.MustCompile(`^[1-9][0-9]*[bkmg]$`)
	portMappingRe = regexp.MustCompile(`^[0-9]+:[0-9]+(/(?:tcp|udp))?$`)
	mountConfigRe = regexp.MustCompile(`^.+:(ro|rw)$`)
)

// Validate checks a merged Config for invalid values. Returns *ValidationError
// for the first invalid field, or nil if the config is valid.
func Validate(cfg Config) error {
	if cfg.Resources.CPUs <= 0 {
		return &ValidationError{Field: "resources.cpus", Reason: "must be greater than 0"}
	}
	if !sizeFormatRe.MatchString(cfg.Resources.Memory) {
		return &ValidationError{
			Field:  "resources.memory",
			Reason: "must be a positive integer followed by a unit suffix (b, k, m, g)",
		}
	}
	if cfg.Resources.MemorySwap != "" && !sizeFormatRe.MatchString(cfg.Resources.MemorySwap) {
		return &ValidationError{
			Field:  "resources.memory_swap",
			Reason: "must be a positive integer followed by a unit suffix (b, k, m, g)",
		}
	}
	if cfg.Dolt.Port < 1 || cfg.Dolt.Port > 65535 {
		return &ValidationError{Field: "dolt.port", Reason: "must be between 1 and 65535"}
	}
	for _, m := range cfg.Mounts.Config {
		if !mountConfigRe.MatchString(m) {
			return &ValidationError{
				Field:  "mounts.config",
				Reason: fmt.Sprintf("%q is not a valid mount entry (expected path:ro or path:rw)", m),
			}
		}
	}
	for _, p := range cfg.Ports {
		if !portMappingRe.MatchString(p) {
			return &ValidationError{
				Field:  "ports",
				Reason: fmt.Sprintf("%q is not a valid port mapping (expected host:container or host:container/proto)", p),
			}
		}

		hostPort, containerPort, err := parsePortMapping(p)
		if err != nil {
			return &ValidationError{Field: "ports", Reason: err.Error()}
		}
		if hostPort < 1 || hostPort > 65535 {
			return &ValidationError{Field: "ports", Reason: fmt.Sprintf("%q host port must be between 1 and 65535", p)}
		}
		if containerPort < 1 || containerPort > 65535 {
			return &ValidationError{Field: "ports", Reason: fmt.Sprintf("%q container port must be between 1 and 65535", p)}
		}
	}
	if _, err := ResolveProjectEnvironment(cfg.Environment); err != nil {
		return err
	}
	return nil
}

func parsePortMapping(mapping string) (int, int, error) {
	parts := strings.SplitN(mapping, "/", 2)
	hostAndContainer := strings.SplitN(parts[0], ":", 2)
	if len(hostAndContainer) != 2 {
		return 0, 0, fmt.Errorf("%q is not a valid port mapping", mapping)
	}

	hostPort, err := strconv.Atoi(hostAndContainer[0])
	if err != nil {
		return 0, 0, fmt.Errorf("%q host port must be numeric", mapping)
	}

	containerPort, err := strconv.Atoi(hostAndContainer[1])
	if err != nil {
		return 0, 0, fmt.Errorf("%q container port must be numeric", mapping)
	}

	return hostPort, containerPort, nil
}
