package doctor

import (
	"context"
	"time"
)

// ReportCheck holds a single check's result along with its metadata.
type ReportCheck struct {
	Tier           string
	Container      string
	Name           string
	Status         Status
	Message        string
	Detail         string
	Recommendation string
}

// Summary counts check outcomes.
type Summary struct {
	Passed   int
	Warnings int
	Errors   int
}

// Report holds the full diagnostic report.
type Report struct {
	Status  Status
	Summary Summary
	Checks  []ReportCheck
}

// Runner executes checks in order, respecting prerequisites and timeouts.
type Runner struct {
	checks []Check
}

// NewRunner creates a runner with the given checks, executed in order.
func NewRunner(checks []Check) *Runner {
	return &Runner{checks: checks}
}

// Run executes all checks and returns the report.
func (r *Runner) Run(ctx context.Context) Report {
	results := make(map[string]Status)
	var reportChecks []ReportCheck

	for _, c := range r.checks {
		rc := r.runCheck(ctx, c, results)
		reportChecks = append(reportChecks, rc)
		results[c.ID()] = rc.Status
	}

	return buildReport(reportChecks)
}

func (r *Runner) runCheck(ctx context.Context, c Check, results map[string]Status) ReportCheck {
	// Check prerequisites.
	for _, prereq := range c.Prerequisites() {
		if s, ok := results[prereq]; ok && s != StatusPass && s != StatusWarn {
			return ReportCheck{
				Tier:    c.Tier(),
				Name:    c.ID(),
				Status:  StatusSkip,
				Message: "skipped: prerequisite " + prereq + " failed",
			}
		}
	}

	// Run with timeout.
	timeout := c.Timeout()
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultCh := make(chan CheckResult, 1)
	go func() {
		resultCh <- c.Run(checkCtx)
	}()

	select {
	case result := <-resultCh:
		return ReportCheck{
			Tier:           c.Tier(),
			Name:           c.ID(),
			Status:         result.Status,
			Message:        result.Message,
			Detail:         result.Detail,
			Recommendation: result.Recommendation,
		}
	case <-checkCtx.Done():
		return ReportCheck{
			Tier:    c.Tier(),
			Name:    c.ID(),
			Status:  StatusError,
			Message: "check timed out",
		}
	}
}

func buildReport(checks []ReportCheck) Report {
	var summary Summary
	worst := StatusPass

	for _, c := range checks {
		switch c.Status {
		case StatusPass:
			summary.Passed++
		case StatusWarn:
			summary.Warnings++
		case StatusError:
			summary.Errors++
		}

		if statusSeverity(c.Status) > statusSeverity(worst) {
			worst = c.Status
		}
	}

	return Report{
		Status:  worst,
		Summary: summary,
		Checks:  checks,
	}
}

func statusSeverity(s Status) int {
	switch s {
	case StatusPass:
		return 0
	case StatusSkip:
		return 0
	case StatusWarn:
		return 1
	case StatusError:
		return 2
	default:
		return 0
	}
}

const defaultTimeout = 10 * time.Second
