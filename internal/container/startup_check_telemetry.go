package container

import "time"

// StartupCheckPhase identifies a startup-check phase.
type StartupCheckPhase string

const (
	// StartupCheckPhaseValidation identifies required devShell validation.
	StartupCheckPhaseValidation StartupCheckPhase = "validation"
	// StartupCheckPhasePrepare identifies optional startup preparation.
	StartupCheckPhasePrepare StartupCheckPhase = "prepare"
)

// StartupCheckPhaseOutcome identifies phase boundary outcomes.
type StartupCheckPhaseOutcome string

const (
	// StartupCheckPhaseOutcomeStart marks phase start.
	StartupCheckPhaseOutcomeStart StartupCheckPhaseOutcome = "start"
	// StartupCheckPhaseOutcomeFinish marks phase completion.
	StartupCheckPhaseOutcomeFinish StartupCheckPhaseOutcome = "finish"
	// StartupCheckPhaseOutcomeError marks phase failure.
	StartupCheckPhaseOutcomeError StartupCheckPhaseOutcome = "error"
	// StartupCheckPhaseOutcomeCancel marks interrupted phase cancellation.
	StartupCheckPhaseOutcomeCancel StartupCheckPhaseOutcome = "cancel"
)

// StartupCheckInterruption captures interruption context for canceled phases.
type StartupCheckInterruption struct {
	Cause  string
	Detail string
}

// StartupCheckPhaseEvent captures startup-check phase boundaries and timing.
type StartupCheckPhaseEvent struct {
	Phase        StartupCheckPhase
	Outcome      StartupCheckPhaseOutcome
	At           time.Time
	Duration     time.Duration
	Error        string
	Interruption *StartupCheckInterruption
}

// StartupCheckTelemetry captures startup-check phase lifecycle events.
type StartupCheckTelemetry struct {
	now        func() time.Time
	events     []StartupCheckPhaseEvent
	phaseStart map[StartupCheckPhase]time.Time
}

// NewStartupCheckTelemetry creates a startup-check telemetry recorder.
func NewStartupCheckTelemetry() *StartupCheckTelemetry {
	return NewStartupCheckTelemetryWithClock(time.Now)
}

// NewStartupCheckTelemetryWithClock creates a startup-check telemetry recorder
// with an injectable clock.
func NewStartupCheckTelemetryWithClock(now func() time.Time) *StartupCheckTelemetry {
	if now == nil {
		now = time.Now
	}
	return &StartupCheckTelemetry{
		now:        now,
		events:     nil,
		phaseStart: map[StartupCheckPhase]time.Time{},
	}
}

// StartPhase records startup-check phase start.
func (t *StartupCheckTelemetry) StartPhase(phase StartupCheckPhase) {
	if t == nil {
		return
	}
	now := t.now()
	t.phaseStart[phase] = now
	t.events = append(t.events, StartupCheckPhaseEvent{
		Phase:   phase,
		Outcome: StartupCheckPhaseOutcomeStart,
		At:      now,
	})
}

// FinishPhase records startup-check phase successful completion.
func (t *StartupCheckTelemetry) FinishPhase(phase StartupCheckPhase) {
	t.finishPhase(phase, StartupCheckPhaseOutcomeFinish, "", nil)
}

// ErrorPhase records startup-check phase failure.
func (t *StartupCheckTelemetry) ErrorPhase(phase StartupCheckPhase, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	t.finishPhase(phase, StartupCheckPhaseOutcomeError, errMsg, nil)
}

// CancelPhase records startup-check phase cancellation with interruption context.
func (t *StartupCheckTelemetry) CancelPhase(phase StartupCheckPhase, interruption StartupCheckInterruption) {
	t.finishPhase(phase, StartupCheckPhaseOutcomeCancel, interruption.Detail, &interruption)
}

// Events returns a copy of recorded startup-check phase events.
func (t *StartupCheckTelemetry) Events() []StartupCheckPhaseEvent {
	if t == nil {
		return nil
	}
	out := make([]StartupCheckPhaseEvent, len(t.events))
	copy(out, t.events)
	return out
}

func (t *StartupCheckTelemetry) finishPhase(phase StartupCheckPhase, outcome StartupCheckPhaseOutcome, errMsg string, interruption *StartupCheckInterruption) {
	if t == nil {
		return
	}
	now := t.now()
	startedAt, ok := t.phaseStart[phase]
	if ok {
		delete(t.phaseStart, phase)
	}
	duration := time.Duration(0)
	if ok {
		duration = now.Sub(startedAt)
	}
	t.events = append(t.events, StartupCheckPhaseEvent{
		Phase:        phase,
		Outcome:      outcome,
		At:           now,
		Duration:     duration,
		Error:        errMsg,
		Interruption: interruption,
	})
}
