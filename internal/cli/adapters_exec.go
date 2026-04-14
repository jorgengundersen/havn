package cli

import (
	"fmt"
	"strings"

	"github.com/jorgengundersen/havn/internal/docker"
)

func execResultError(result docker.ExecResult) error {
	stderr := strings.TrimSpace(string(result.Stderr))
	if stderr == "" {
		stderr = "command failed"
	}

	return fmt.Errorf("container exec exited %d: %s", result.ExitCode, stderr)
}
