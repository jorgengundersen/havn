// Package doctor implements the havn doctor diagnostic framework.
package doctor

import (
	"context"
	"time"
)

// Status represents the outcome of a single diagnostic check.
type Status string

// Possible check statuses.
const (
	StatusPass  Status = "pass"
	StatusWarn  Status = "warn"
	StatusError Status = "error"
	StatusSkip  Status = "skip"
)

// CheckResult holds the outcome of running a single check.
type CheckResult struct {
	Status         Status
	Message        string
	Detail         string
	Recommendation string
}

// Check is a single diagnostic check that can be run by the runner.
type Check interface {
	// ID returns the stable machine-readable identifier (e.g. "docker_daemon").
	ID() string
	// Tier returns "host" or "container".
	Tier() string
	// Container returns the container name for container-tier checks, or "" for host-tier.
	Container() string
	// Prerequisites returns the IDs of checks that must pass before this one runs.
	Prerequisites() []string
	// Timeout returns the maximum duration for this check.
	Timeout() time.Duration
	// Run executes the check and returns the result.
	Run(ctx context.Context) CheckResult
}
