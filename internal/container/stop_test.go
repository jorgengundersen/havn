package container_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/container"
)

// fakeStopBackend records calls and returns configured responses.
type fakeStopBackend struct {
	containers []container.RawContainer
	listErr    error
	stopCalls  []string
	stopErr    error
	stopErrMap map[string]error // per-container errors, takes precedence over stopErr
}

func (f *fakeStopBackend) ContainerList(_ context.Context, _ map[string]string) ([]container.RawContainer, error) {
	return f.containers, f.listErr
}

func (f *fakeStopBackend) ContainerStop(_ context.Context, cname string, _ time.Duration) error {
	f.stopCalls = append(f.stopCalls, cname)
	if f.stopErrMap != nil {
		if err, ok := f.stopErrMap[cname]; ok {
			return err
		}
	}
	return f.stopErr
}

func TestStop_RunningContainer(t *testing.T) {
	ctx := context.Background()
	backend := &fakeStopBackend{}

	err := container.Stop(ctx, backend, "havn-user-api")

	require.NoError(t, err)
	assert.Equal(t, []string{"havn-user-api"}, backend.stopCalls)
}

func TestStop_NotFound(t *testing.T) {
	ctx := context.Background()
	backend := &fakeStopBackend{
		stopErr: &container.NotFoundError{Name: "havn-user-api"},
	}

	err := container.Stop(ctx, backend, "havn-user-api")

	var notFound *container.NotFoundError
	assert.ErrorAs(t, err, &notFound)
	assert.Equal(t, "havn-user-api", notFound.Name)
}

func TestStop_PathArgDerivesContainerName(t *testing.T) {
	ctx := context.Background()
	backend := &fakeStopBackend{}

	err := container.Stop(ctx, backend, "/home/user/api")

	require.NoError(t, err)
	assert.Equal(t, []string{"havn-user-api"}, backend.stopCalls)
}

func TestStopAll_StopsAllExceptDolt(t *testing.T) {
	ctx := context.Background()
	backend := &fakeStopBackend{
		containers: []container.RawContainer{
			{Name: "havn-user-api", Labels: map[string]string{"managed-by": "havn"}},
			{Name: "havn-user-web", Labels: map[string]string{"managed-by": "havn"}},
			{Name: "havn-dolt", Labels: map[string]string{"managed-by": "havn"}},
		},
	}

	result, err := container.StopAll(ctx, backend)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"havn-user-api", "havn-user-web"}, result.Stopped)
	assert.Empty(t, result.Failed)
	assert.ElementsMatch(t, []string{"havn-user-api", "havn-user-web"}, backend.stopCalls)
}

func TestStopAll_BestEffortContinuesAfterFailure(t *testing.T) {
	ctx := context.Background()
	stopErrs := map[string]error{
		"havn-user-web": assert.AnError,
	}
	backend := &fakeStopBackend{
		containers: []container.RawContainer{
			{Name: "havn-user-api", Labels: map[string]string{"managed-by": "havn"}},
			{Name: "havn-user-web", Labels: map[string]string{"managed-by": "havn"}},
			{Name: "havn-user-cli", Labels: map[string]string{"managed-by": "havn"}},
		},
		stopErrMap: stopErrs,
	}

	result, err := container.StopAll(ctx, backend)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"havn-user-api", "havn-user-cli"}, result.Stopped)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, "havn-user-web", result.Failed[0].Name)
	assert.ErrorIs(t, result.Failed[0].Err, assert.AnError)
	// All three containers were attempted (best-effort)
	assert.Len(t, backend.stopCalls, 3)
}

func TestStopAll_EmptyList(t *testing.T) {
	ctx := context.Background()
	backend := &fakeStopBackend{}

	result, err := container.StopAll(ctx, backend)

	require.NoError(t, err)
	assert.Empty(t, result.Stopped)
	assert.Empty(t, result.Failed)
}

func TestStopAll_ListError(t *testing.T) {
	ctx := context.Background()
	backend := &fakeStopBackend{listErr: assert.AnError}

	_, err := container.StopAll(ctx, backend)

	assert.ErrorIs(t, err, assert.AnError)
}
