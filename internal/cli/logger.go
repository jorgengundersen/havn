package cli

import (
	"log/slog"
	"os"
)

// SetupLogger creates a slog.Logger configured per code-standards §5.
// TextHandler by default, JSONHandler when jsonOutput is true.
// Level is Info by default, Debug when verbose is true.
// Output is always os.Stderr.
func SetupLogger(verbose, jsonOutput bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	if verbose {
		opts.Level = slog.LevelDebug
	}
	var handler slog.Handler
	if jsonOutput {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	return slog.New(handler)
}
