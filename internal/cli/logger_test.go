package cli_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
)

func TestSetupLogger_DefaultsToInfoLevel(t *testing.T) {
	logger := cli.SetupLogger(false, false)

	require.NotNil(t, logger)
	assert.True(t, logger.Enabled(context.Background(), slog.LevelInfo), "info should be enabled")
	assert.False(t, logger.Enabled(context.Background(), slog.LevelDebug), "debug should not be enabled")
}

func TestSetupLogger_VerboseEnablesDebug(t *testing.T) {
	logger := cli.SetupLogger(true, false)

	assert.True(t, logger.Enabled(context.Background(), slog.LevelDebug), "debug should be enabled when verbose")
	assert.True(t, logger.Enabled(context.Background(), slog.LevelInfo), "info should still be enabled")
}

func TestSetupLogger_JSONUsesJSONHandler(t *testing.T) {
	logger := cli.SetupLogger(false, true)

	handler := logger.Handler()
	assert.IsType(t, &slog.JSONHandler{}, handler, "json mode should use JSONHandler")
}

func TestSetupLogger_DefaultUsesTextHandler(t *testing.T) {
	logger := cli.SetupLogger(false, false)

	handler := logger.Handler()
	assert.IsType(t, &slog.TextHandler{}, handler, "default should use TextHandler")
}

func TestNewRoot_NilLoggerDoesNotPanic(t *testing.T) {
	deps := cli.Deps{}
	require.Nil(t, deps.Logger, "precondition: Logger is nil")

	root := cli.NewRoot(deps)

	require.NotNil(t, root)
}

func TestDeps_LoggerField(t *testing.T) {
	logger := cli.SetupLogger(false, false)
	deps := cli.Deps{Logger: logger}

	assert.Same(t, logger, deps.Logger)
}

func TestSetupLogger_VerboseAndJSON(t *testing.T) {
	logger := cli.SetupLogger(true, true)

	assert.IsType(t, &slog.JSONHandler{}, logger.Handler(), "json mode should use JSONHandler")
	assert.True(t, logger.Enabled(context.Background(), slog.LevelDebug), "debug should be enabled when verbose")
}

func TestNewRoot_PersistentPreRunE_IsSet(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	assert.NotNil(t, root.PersistentPreRunE, "PersistentPreRunE should be set to wire logger from flags")
}
