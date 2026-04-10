package doctor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/doctor"
)

func TestStatus_StringValues(t *testing.T) {
	tests := []struct {
		name   string
		status doctor.Status
		want   string
	}{
		{name: "pass", status: doctor.StatusPass, want: "pass"},
		{name: "warn", status: doctor.StatusWarn, want: "warn"},
		{name: "error", status: doctor.StatusError, want: "error"},
		{name: "skip", status: doctor.StatusSkip, want: "skip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.status))
		})
	}
}

func TestCheckResult_Fields(t *testing.T) {
	result := doctor.CheckResult{
		Status:         doctor.StatusWarn,
		Message:        "SSH agent not forwarding",
		Detail:         "SSH_AUTH_SOCK not set inside container",
		Recommendation: "Ensure ssh-agent is running on host",
	}

	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Equal(t, "SSH agent not forwarding", result.Message)
	assert.Equal(t, "SSH_AUTH_SOCK not set inside container", result.Detail)
	assert.Equal(t, "Ensure ssh-agent is running on host", result.Recommendation)
}
