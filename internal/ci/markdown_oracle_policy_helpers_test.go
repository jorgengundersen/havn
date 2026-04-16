package ci_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var disallowedMarkdownOracleTokens = []string{
	"specs/",
	"README.md",
	"docs/",
}

func collectMarkdownOracleViolations() ([]string, error) {
	paths, err := filepath.Glob("*_test.go")
	if err != nil {
		return nil, err
	}

	violations := make([]string, 0)
	for _, path := range paths {
		if strings.HasPrefix(path, "markdown_oracle_policy") {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		for _, token := range disallowedMarkdownOracleTokens {
			if strings.Contains(string(content), token) {
				violations = append(violations, fmt.Sprintf("%s contains %q", path, token))
			}
		}
	}

	return violations, nil
}
