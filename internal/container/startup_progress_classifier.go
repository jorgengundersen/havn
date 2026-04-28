package container

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type startupProgressActivity string

const (
	startupProgressActivityEvaluate startupProgressActivity = "evaluate"
	startupProgressActivityFetch    startupProgressActivity = "fetch"
	startupProgressActivityBuild    startupProgressActivity = "build"
	startupProgressActivityOther    startupProgressActivity = "other"
)

type startupProgressClassification struct {
	Activity   startupProgressActivity
	Summary    string
	Current    *int
	Total      *int
	DoneBytes  string
	TotalBytes string
	Rate       string
}

var progressFractionPattern = regexp.MustCompile(`(\d+)\s*/\s*(\d+)`)
var progressBytesPattern = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?[KMGT]i?B)\s*/\s*([0-9]+(?:\.[0-9]+)?[KMGT]i?B)`)
var progressRatePattern = regexp.MustCompile(`@\s*([0-9]+(?:\.[0-9]+)?[KMGT]i?B/s)`)

func classifyStartupProgress(line string) startupProgressClassification {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return startupProgressClassification{Activity: startupProgressActivityOther, Summary: ""}
	}

	lower := strings.ToLower(trimmed)
	result := startupProgressClassification{Activity: startupProgressActivityOther, Summary: trimmed}

	switch {
	case strings.Contains(lower, "fetch") || strings.Contains(lower, "downloading") || strings.Contains(lower, "copying path") || strings.Contains(lower, "substitute"):
		result.Activity = startupProgressActivityFetch
	case strings.Contains(lower, "building") || strings.Contains(lower, "derivation") || strings.Contains(lower, "compiling"):
		result.Activity = startupProgressActivityBuild
	case strings.Contains(lower, "evaluat") || strings.Contains(lower, "flake") || strings.Contains(lower, "resolving"):
		result.Activity = startupProgressActivityEvaluate
	}

	if m := progressFractionPattern.FindStringSubmatch(trimmed); len(m) == 3 {
		if current, err := strconv.Atoi(m[1]); err == nil {
			result.Current = &current
		}
		if total, err := strconv.Atoi(m[2]); err == nil {
			result.Total = &total
		}
	}

	if m := progressBytesPattern.FindStringSubmatch(trimmed); len(m) == 3 {
		result.DoneBytes = m[1]
		result.TotalBytes = m[2]
	}

	if m := progressRatePattern.FindStringSubmatch(trimmed); len(m) == 2 {
		result.Rate = m[1]
	}

	return result
}

func renderStartupProgress(c startupProgressClassification) string {
	if c.Summary == "" {
		return ""
	}
	parts := []string{fmt.Sprintf("%s: %s", c.Activity, c.Summary)}
	metrics := make([]string, 0, 3)
	if c.Current != nil && c.Total != nil {
		metrics = append(metrics, fmt.Sprintf("%d/%d", *c.Current, *c.Total))
	}
	if c.DoneBytes != "" && c.TotalBytes != "" {
		metrics = append(metrics, fmt.Sprintf("%s/%s", c.DoneBytes, c.TotalBytes))
	}
	if c.Rate != "" {
		metrics = append(metrics, "@ "+c.Rate)
	}
	if len(metrics) == 0 {
		return parts[0]
	}
	return parts[0] + " (" + strings.Join(metrics, ", ") + ")"
}
