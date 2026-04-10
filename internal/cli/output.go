package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

// Output enforces stream separation: status to stderr, data to stdout.
type Output struct {
	stdout  io.Writer
	stderr  io.Writer
	json    bool
	verbose bool
}

// NewOutput creates an Output from the given writers and mode flags.
func NewOutput(stdout, stderr io.Writer, jsonMode, verbose bool) *Output {
	return &Output{
		stdout:  stdout,
		stderr:  stderr,
		json:    jsonMode,
		verbose: verbose,
	}
}

// Status writes a status message to stderr.
func (o *Output) Status(msg string) {
	_, _ = fmt.Fprintln(o.stderr, msg)
}

// Data writes data output to stdout.
func (o *Output) Data(msg string) {
	_, _ = fmt.Fprintln(o.stdout, msg)
}

// DataJSON encodes v as JSON and writes it to stdout.
func (o *Output) DataJSON(v any) error {
	enc := json.NewEncoder(o.stdout)
	return enc.Encode(v)
}

// IsJSON reports whether JSON output mode is enabled.
func (o *Output) IsJSON() bool {
	return o.json
}
