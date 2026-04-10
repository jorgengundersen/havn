package doctor_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/doctor"
)

// fakeCheck is a test double implementing Check.
type fakeCheck struct {
	id            string
	tier          string
	container     string
	prerequisites []string
	timeout       time.Duration
	result        doctor.CheckResult
}

func (f *fakeCheck) ID() string                               { return f.id }
func (f *fakeCheck) Tier() string                             { return f.tier }
func (f *fakeCheck) Container() string                        { return f.container }
func (f *fakeCheck) Prerequisites() []string                  { return f.prerequisites }
func (f *fakeCheck) Timeout() time.Duration                   { return f.timeout }
func (f *fakeCheck) Run(_ context.Context) doctor.CheckResult { return f.result }

func TestRunner_AllPass(t *testing.T) {
	checks := []doctor.Check{
		&fakeCheck{
			id:   "check_a",
			tier: "host",
			result: doctor.CheckResult{
				Status:  doctor.StatusPass,
				Message: "check a passed",
			},
		},
		&fakeCheck{
			id:   "check_b",
			tier: "host",
			result: doctor.CheckResult{
				Status:  doctor.StatusPass,
				Message: "check b passed",
			},
		},
	}

	runner := doctor.NewRunner(checks)
	report := runner.Run(context.Background())

	assert.Equal(t, doctor.StatusPass, report.Status)
	assert.Equal(t, 2, report.Summary.Passed)
	assert.Equal(t, 0, report.Summary.Warnings)
	assert.Equal(t, 0, report.Summary.Errors)
	require.Len(t, report.Checks, 2)
	assert.Equal(t, "check_a", report.Checks[0].Name)
	assert.Equal(t, "check_b", report.Checks[1].Name)
}

func TestRunner_OverallStatusIsWorstResult(t *testing.T) {
	checks := []doctor.Check{
		&fakeCheck{
			id:     "ok",
			tier:   "host",
			result: doctor.CheckResult{Status: doctor.StatusPass, Message: "ok"},
		},
		&fakeCheck{
			id:     "warning",
			tier:   "host",
			result: doctor.CheckResult{Status: doctor.StatusWarn, Message: "warn"},
		},
	}

	runner := doctor.NewRunner(checks)
	report := runner.Run(context.Background())

	assert.Equal(t, doctor.StatusWarn, report.Status)
	assert.Equal(t, 1, report.Summary.Passed)
	assert.Equal(t, 1, report.Summary.Warnings)
}

func TestRunner_PrerequisiteFailedSkipsDependents(t *testing.T) {
	checks := []doctor.Check{
		&fakeCheck{
			id:   "docker_daemon",
			tier: "host",
			result: doctor.CheckResult{
				Status:  doctor.StatusError,
				Message: "Docker daemon is not running",
			},
		},
		&fakeCheck{
			id:            "base_image",
			tier:          "host",
			prerequisites: []string{"docker_daemon"},
			result: doctor.CheckResult{
				Status:  doctor.StatusPass,
				Message: "should not run",
			},
		},
	}

	runner := doctor.NewRunner(checks)
	report := runner.Run(context.Background())

	assert.Equal(t, doctor.StatusError, report.Status)
	require.Len(t, report.Checks, 2)
	assert.Equal(t, doctor.StatusError, report.Checks[0].Status)
	assert.Equal(t, doctor.StatusSkip, report.Checks[1].Status)
	assert.Contains(t, report.Checks[1].Message, "prerequisite")
}

func TestRunner_SkippedChecksNotCountedInSummary(t *testing.T) {
	checks := []doctor.Check{
		&fakeCheck{
			id:     "failing",
			tier:   "host",
			result: doctor.CheckResult{Status: doctor.StatusError, Message: "fail"},
		},
		&fakeCheck{
			id:            "dependent",
			tier:          "host",
			prerequisites: []string{"failing"},
			result:        doctor.CheckResult{Status: doctor.StatusPass, Message: "never runs"},
		},
	}

	runner := doctor.NewRunner(checks)
	report := runner.Run(context.Background())

	assert.Equal(t, 0, report.Summary.Passed)
	assert.Equal(t, 1, report.Summary.Errors)
	assert.Equal(t, 0, report.Summary.Warnings)
}

func TestRunner_TimeoutProducesError(t *testing.T) {
	slowCheck := &fakeCheck{
		id:      "slow",
		tier:    "host",
		timeout: 50 * time.Millisecond,
	}
	// Override Run to block until context cancelled.
	blockingCheck := &blockingFakeCheck{fakeCheck: *slowCheck}

	runner := doctor.NewRunner([]doctor.Check{blockingCheck})
	report := runner.Run(context.Background())

	require.Len(t, report.Checks, 1)
	assert.Equal(t, doctor.StatusError, report.Checks[0].Status)
	assert.Contains(t, report.Checks[0].Message, "timed out")
}

// blockingFakeCheck blocks until context is cancelled.
type blockingFakeCheck struct {
	fakeCheck
}

func (b *blockingFakeCheck) Run(ctx context.Context) doctor.CheckResult {
	<-ctx.Done()
	return doctor.CheckResult{Status: doctor.StatusError, Message: "should not be used"}
}

func TestRunner_ErrorOverridesWarn(t *testing.T) {
	checks := []doctor.Check{
		&fakeCheck{
			id:     "warn_check",
			tier:   "host",
			result: doctor.CheckResult{Status: doctor.StatusWarn, Message: "warning"},
		},
		&fakeCheck{
			id:     "error_check",
			tier:   "host",
			result: doctor.CheckResult{Status: doctor.StatusError, Message: "error"},
		},
	}

	runner := doctor.NewRunner(checks)
	report := runner.Run(context.Background())

	assert.Equal(t, doctor.StatusError, report.Status)
}
