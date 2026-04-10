// Package name provides named resource types and pure functions for deriving
// deterministic container names from filesystem paths.
package name

import (
	"fmt"
	"regexp"
	"strings"
)

const maxContainerNameLen = 128

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// DeriveContainerName builds a deterministic container name from parent and
// project directory names. The result is always a valid Docker container name.
func DeriveContainerName(parent, project string) (ContainerName, error) {
	p := sanitize(parent)
	proj := sanitize(project)

	if p == "" || proj == "" {
		return "", fmt.Errorf("container name is empty after sanitization (parent=%q, project=%q)", parent, project)
	}

	result := fmt.Sprintf("havn-%s-%s", p, proj)
	if len(result) > maxContainerNameLen {
		return "", fmt.Errorf("container name %q exceeds %d character limit", result, maxContainerNameLen)
	}

	return ContainerName(result), nil
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
