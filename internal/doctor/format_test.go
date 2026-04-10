package doctor_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/doctor"
)

func sampleReport() doctor.Report {
	return doctor.Report{
		Status: doctor.StatusWarn,
		Summary: doctor.Summary{
			Passed:   2,
			Warnings: 1,
			Errors:   0,
		},
		Checks: []doctor.ReportCheck{
			{
				Tier:    "host",
				Name:    "docker_daemon",
				Status:  doctor.StatusPass,
				Message: "Docker daemon running",
				Detail:  "Docker 24.0.7, API 1.43",
			},
			{
				Tier:    "host",
				Name:    "base_image",
				Status:  doctor.StatusPass,
				Message: "Base image exists",
			},
			{
				Tier:           "host",
				Name:           "network",
				Status:         doctor.StatusWarn,
				Message:        "Network does not exist",
				Recommendation: "Network is auto-created on first 'havn' start",
			},
		},
	}
}

func TestFormatHuman_ContainsStatusPrefixes(t *testing.T) {
	output := doctor.FormatHuman(sampleReport())

	assert.Contains(t, output, "[pass]")
	assert.Contains(t, output, "[warn]")
	assert.Contains(t, output, "Host")
	assert.Contains(t, output, "->")
}

func TestFormatHuman_SummaryLine(t *testing.T) {
	output := doctor.FormatHuman(sampleReport())

	assert.Contains(t, output, "1 warning")
	assert.Contains(t, output, "0 errors")
}

func TestFormatVerbose_IncludesDetail(t *testing.T) {
	output := doctor.FormatVerbose(sampleReport())

	assert.Contains(t, output, "Docker 24.0.7")
	assert.Contains(t, output, "[pass]")
	assert.Contains(t, output, "[warn]")
}

func TestFormatJSON_MatchesSchema(t *testing.T) {
	output := doctor.FormatJSON(sampleReport())

	var parsed map[string]any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "warn", parsed["status"])

	summary, ok := parsed["summary"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(2), summary["passed"])
	assert.Equal(t, float64(1), summary["warnings"])
	assert.Equal(t, float64(0), summary["errors"])

	checks, ok := parsed["checks"].([]any)
	require.True(t, ok)
	require.Len(t, checks, 3)

	first := checks[0].(map[string]any)
	assert.Equal(t, "host", first["tier"])
	assert.Equal(t, "docker_daemon", first["name"])
	assert.Equal(t, "pass", first["status"])
}

func TestFormatJSON_OmitsEmptyOptionalFields(t *testing.T) {
	report := doctor.Report{
		Status:  doctor.StatusPass,
		Summary: doctor.Summary{Passed: 1},
		Checks: []doctor.ReportCheck{
			{
				Tier:    "host",
				Name:    "docker_daemon",
				Status:  doctor.StatusPass,
				Message: "Docker daemon running",
			},
		},
	}

	output := doctor.FormatJSON(report)

	// detail and recommendation should not appear when empty
	assert.NotContains(t, output, "detail")
	assert.NotContains(t, output, "recommendation")
}

func TestFormatHuman_SkippedChecks(t *testing.T) {
	report := doctor.Report{
		Status:  doctor.StatusError,
		Summary: doctor.Summary{Errors: 1},
		Checks: []doctor.ReportCheck{
			{Tier: "host", Name: "docker_daemon", Status: doctor.StatusError, Message: "Docker not running"},
			{Tier: "host", Name: "base_image", Status: doctor.StatusSkip, Message: "skipped: prerequisite docker_daemon failed"},
		},
	}

	output := doctor.FormatHuman(report)
	assert.Contains(t, output, "[skip]")
	assert.Contains(t, output, "[error]")

	// Verify error appears before skip in the output
	errorIdx := strings.Index(output, "[error]")
	skipIdx := strings.Index(output, "[skip]")
	assert.Less(t, errorIdx, skipIdx)
}
