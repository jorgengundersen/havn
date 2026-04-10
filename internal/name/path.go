package name

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SplitProjectPath extracts the last two path segments from an absolute path.
// It returns (parent, project, nil) where parent is the second-to-last segment
// and project is the last segment.
func SplitProjectPath(absPath string) (parent, project string, err error) {
	if !filepath.IsAbs(absPath) {
		return "", "", fmt.Errorf("path must be absolute: %q", absPath)
	}

	cleaned := filepath.Clean(absPath)
	segments := splitSegments(cleaned)

	if len(segments) < 2 {
		return "", "", fmt.Errorf("path must have at least 2 segments below root: %q", absPath)
	}

	return segments[len(segments)-2], segments[len(segments)-1], nil
}

func splitSegments(path string) []string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
