package config

import (
	"fmt"
	"os"
	"regexp"
)

var passthroughValueRe = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)

func isReservedRuntimeEnvKey(key string) bool {
	if key == "SSH_AUTH_SOCK" {
		return true
	}
	return len(key) >= len("BEADS_DOLT_") && key[:len("BEADS_DOLT_")] == "BEADS_DOLT_"
}

// ResolveProjectEnvironment validates and resolves [environment] entries.
// Literal values pass through unchanged. A value of ${VAR} is replaced by
// the current host value of VAR.
func ResolveProjectEnvironment(projectEnv map[string]string) (map[string]string, error) {
	if len(projectEnv) == 0 {
		return nil, nil
	}

	resolved := make(map[string]string, len(projectEnv))
	for key, value := range projectEnv {
		if isReservedRuntimeEnvKey(key) {
			return nil, &ValidationError{
				Field:  "environment." + key,
				Reason: "is reserved for havn-managed runtime environment",
			}
		}
		if matched := passthroughValueRe.FindStringSubmatch(value); len(matched) == 2 {
			hostVar := matched[1]
			hostValue, ok := os.LookupEnv(hostVar)
			if !ok {
				return nil, &ValidationError{
					Field:  "environment." + key,
					Reason: fmt.Sprintf("references unset host variable %q", hostVar),
				}
			}
			resolved[key] = hostValue
			continue
		}
		resolved[key] = value
	}

	return resolved, nil
}
