package container_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/container"
)

func TestStartupCheckTelemetry_StartAndFinishPhase(t *testing.T) {
	times := []time.Time{
		time.Unix(10, 0),
		time.Unix(14, 0),
	}
	nowCalls := 0
	telemetry := container.NewStartupCheckTelemetryWithClock(func() time.Time {
		current := times[nowCalls]
		nowCalls++
		return current
	})

	telemetry.StartPhase(container.StartupCheckPhaseValidation)
	telemetry.FinishPhase(container.StartupCheckPhaseValidation)

	events := telemetry.Events()
	require.Len(t, events, 2)
	assert.Equal(t, container.StartupCheckPhaseValidation, events[0].Phase)
	assert.Equal(t, container.StartupCheckPhaseOutcomeStart, events[0].Outcome)
	assert.Zero(t, events[0].Duration)
	assert.Equal(t, container.StartupCheckPhaseValidation, events[1].Phase)
	assert.Equal(t, container.StartupCheckPhaseOutcomeFinish, events[1].Outcome)
	assert.Equal(t, 4*time.Second, events[1].Duration)
}

func TestStartupCheckTelemetry_StartAndErrorPhase(t *testing.T) {
	times := []time.Time{
		time.Unix(20, 0),
		time.Unix(23, 0),
	}
	nowCalls := 0
	telemetry := container.NewStartupCheckTelemetryWithClock(func() time.Time {
		current := times[nowCalls]
		nowCalls++
		return current
	})

	telemetry.StartPhase(container.StartupCheckPhasePrepare)
	telemetry.ErrorPhase(container.StartupCheckPhasePrepare, assert.AnError)

	events := telemetry.Events()
	require.Len(t, events, 2)
	assert.Equal(t, container.StartupCheckPhaseOutcomeStart, events[0].Outcome)
	assert.Equal(t, container.StartupCheckPhaseOutcomeError, events[1].Outcome)
	assert.Equal(t, 3*time.Second, events[1].Duration)
	assert.Equal(t, assert.AnError.Error(), events[1].Error)
	assert.Nil(t, events[1].Interruption)
}

func TestStartupCheckTelemetry_StartAndCancelPhase(t *testing.T) {
	times := []time.Time{
		time.Unix(40, 0),
		time.Unix(47, 0),
	}
	nowCalls := 0
	telemetry := container.NewStartupCheckTelemetryWithClock(func() time.Time {
		current := times[nowCalls]
		nowCalls++
		return current
	})

	telemetry.StartPhase(container.StartupCheckPhaseValidation)
	telemetry.CancelPhase(container.StartupCheckPhaseValidation, container.StartupCheckInterruption{
		Cause:  "context_canceled",
		Detail: context.Canceled.Error(),
	})

	events := telemetry.Events()
	require.Len(t, events, 2)
	assert.Equal(t, container.StartupCheckPhaseOutcomeCancel, events[1].Outcome)
	assert.Equal(t, 7*time.Second, events[1].Duration)
	require.NotNil(t, events[1].Interruption)
	assert.Equal(t, "context_canceled", events[1].Interruption.Cause)
	assert.Equal(t, context.Canceled.Error(), events[1].Interruption.Detail)
}
