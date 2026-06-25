package cli

import (
	"fmt"
	"strings"

	"github.com/jorgengundersen/havn/internal/docker"
)

func execResultError(result docker.ExecResult) error {
	msg := strings.TrimSpace(string(result.Stderr))
	if msg == "" {
		msg = strings.TrimSpace(string(result.Stdout))
	}
	if msg == "" {
		msg = "command failed"
	}

	return fmt.Errorf("container exec exited %d: %s", result.ExitCode, msg)
}
