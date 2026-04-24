package doctor

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/dolt"
)

// --- 1.1 docker_daemon ---

type dockerDaemonCheck struct {
	checkMetadata
	backend Backend
}

// NewDockerDaemonCheck creates check 1.1: Docker daemon accessible.
func NewDockerDaemonCheck(backend Backend) Check {
	return &dockerDaemonCheck{
		checkMetadata: newHostCheckMetadata("docker_daemon", nil, defaultTimeout),
		backend:       backend,
	}
}

func (c *dockerDaemonCheck) Run(ctx context.Context) CheckResult {
	if err := c.backend.Ping(ctx); err != nil {
		return CheckResult{
			Status:         StatusError,
			Message:        "Docker daemon is not running",
			Detail:         err.Error(),
			Recommendation: "Start Docker, or check that the current user is in the docker group",
		}
	}
	info, err := c.backend.Info(ctx)
	if err != nil {
		return CheckResult{
			Status:         StatusError,
			Message:        "Docker daemon is not running",
			Detail:         err.Error(),
			Recommendation: "Start Docker, or check that the current user is in the docker group",
		}
	}
	detail := ""
	if info.Version != "" {
		detail = fmt.Sprintf("Docker %s, API %s", info.Version, info.APIVersion)
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "Docker daemon running",
		Detail:  detail,
	}
}

// --- 1.2 base_image ---

type baseImageCheck struct {
	checkMetadata
	backend Backend
	image   string
}

// NewBaseImageCheck creates check 1.2: base image exists.
func NewBaseImageCheck(backend Backend, image string) Check {
	return &baseImageCheck{
		checkMetadata: newHostCheckMetadata("base_image", []string{"docker_daemon"}, defaultTimeout),
		backend:       backend,
		image:         image,
	}
}

func (c *baseImageCheck) Run(ctx context.Context) CheckResult {
	info, found, err := c.backend.ImageInspect(ctx, c.image)
	if err != nil {
		return CheckResult{
			Status:  StatusError,
			Message: "Base image check failed",
			Detail:  err.Error(),
		}
	}
	if !found {
		return CheckResult{
			Status:         StatusSkip,
			Message:        "Base image not found",
			Detail:         c.image,
			Recommendation: "Run 'havn build'",
		}
	}
	detail := c.image
	if info.Created != "" {
		detail = fmt.Sprintf("%s (built %s)", c.image, info.Created)
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "Base image exists",
		Detail:  detail,
	}
}

// --- 1.3 network ---

type networkCheck struct {
	checkMetadata
	backend Backend
	network string
}

// NewNetworkCheck creates check 1.3: Docker network exists.
func NewNetworkCheck(backend Backend, network string) Check {
	return &networkCheck{
		checkMetadata: newHostCheckMetadata("network", []string{"docker_daemon"}, defaultTimeout),
		backend:       backend,
		network:       network,
	}
}

func (c *networkCheck) Run(ctx context.Context) CheckResult {
	info, found, err := c.backend.NetworkInspect(ctx, c.network)
	if err != nil {
		return CheckResult{
			Status:  StatusError,
			Message: "Network check failed",
			Detail:  err.Error(),
		}
	}
	if !found {
		return CheckResult{
			Status:         StatusSkip,
			Message:        "Network does not exist",
			Detail:         c.network,
			Recommendation: "Network is auto-created on first 'havn' start",
		}
	}
	detail := c.network
	if info.ContainerCount > 0 {
		detail = fmt.Sprintf("%s (bridge, %d containers connected)", c.network, info.ContainerCount)
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "Network exists",
		Detail:  detail,
	}
}

// --- 1.4 volumes ---

type volumesCheck struct {
	checkMetadata
	backend Backend
	volumes []string
}

// NewVolumesCheck creates check 1.4: named volumes exist.
func NewVolumesCheck(backend Backend, volumes []string) Check {
	return &volumesCheck{
		checkMetadata: newHostCheckMetadata("volumes", []string{"docker_daemon"}, defaultTimeout),
		backend:       backend,
		volumes:       volumes,
	}
}

func (c *volumesCheck) Run(ctx context.Context) CheckResult {
	var missing []string
	for _, v := range c.volumes {
		found, err := c.backend.VolumeInspect(ctx, v)
		if err != nil {
			return CheckResult{
				Status:  StatusError,
				Message: "Volume check failed",
				Detail:  err.Error(),
			}
		}
		if !found {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		return CheckResult{
			Status:         StatusSkip,
			Message:        "Volumes missing",
			Detail:         strings.Join(missing, ", "),
			Recommendation: "Volumes are auto-created on first 'havn' start",
		}
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "Volumes exist",
		Detail:  strings.Join(c.volumes, ", "),
	}
}

// --- 1.5 global_config ---

type globalConfigCheck struct {
	checkMetadata
	path string
}

// NewGlobalConfigCheck creates check 1.5: global config parses.
func NewGlobalConfigCheck(path string) Check {
	return &globalConfigCheck{
		checkMetadata: newHostCheckMetadata("global_config", nil, defaultTimeout),
		path:          path,
	}
}

func (c *globalConfigCheck) Run(_ context.Context) CheckResult {
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		return CheckResult{
			Status:  StatusWarn,
			Message: "No global config found (using defaults)",
			Detail:  c.path,
		}
	}
	_, err := config.LoadFile(c.path)
	if err != nil {
		return CheckResult{
			Status:         StatusError,
			Message:        "Global config syntax error",
			Detail:         err.Error(),
			Recommendation: "Fix the syntax error in " + c.path,
		}
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "Global config valid",
		Detail:  c.path,
	}
}

// --- 1.6 project_config ---

type projectConfigCheck struct {
	checkMetadata
	path                   string
	effectiveValidationErr error
}

// NewProjectConfigCheck creates check 1.6: project config parses and merges.
func NewProjectConfigCheck(path string, effectiveValidationErr error) Check {
	return &projectConfigCheck{
		checkMetadata:          newHostCheckMetadata("project_config", nil, defaultTimeout),
		path:                   path,
		effectiveValidationErr: effectiveValidationErr,
	}
}

func (c *projectConfigCheck) Run(_ context.Context) CheckResult {
	if c.effectiveValidationErr != nil {
		return CheckResult{
			Status:         StatusError,
			Message:        "Project config validation error",
			Detail:         c.effectiveValidationErr.Error(),
			Recommendation: "Fix invalid project config values",
		}
	}

	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		return CheckResult{
			Status:  StatusPass,
			Message: "Project config valid",
			Detail:  "no project config (using defaults)",
		}
	}
	_, err := config.LoadFile(c.path)
	if err != nil {
		return CheckResult{
			Status:         StatusError,
			Message:        "Project config syntax error",
			Detail:         err.Error(),
			Recommendation: "Fix the syntax error in " + c.path,
		}
	}
	return CheckResult{
		Status:  StatusPass,
		Message: "Project config valid",
		Detail:  c.path,
	}
}

// --- 1.7 dolt_server ---

type doltServerCheck struct {
	checkMetadata
	backend     Backend
	doltEnabled bool
}

// NewDoltServerCheck creates check 1.7: Dolt container running and responsive.
func NewDoltServerCheck(backend Backend, doltEnabled bool) Check {
	return &doltServerCheck{
		checkMetadata: newHostCheckMetadata("dolt_server", []string{"docker_daemon"}, defaultTimeout),
		backend:       backend,
		doltEnabled:   doltEnabled,
	}
}

func (c *doltServerCheck) Run(ctx context.Context) CheckResult {
	if !c.doltEnabled {
		return CheckResult{
			Status:  StatusSkip,
			Message: "Dolt not enabled",
		}
	}

	info, found, err := c.backend.ContainerInspect(ctx, "havn-dolt")
	if err != nil {
		return CheckResult{
			Status:  StatusError,
			Message: "Dolt server check failed",
			Detail:  err.Error(),
		}
	}
	if !found || !info.Running {
		return CheckResult{
			Status:         StatusError,
			Message:        "Dolt server is not running",
			Recommendation: "Run 'havn dolt start'",
		}
	}

	if info.Labels["managed-by"] != "havn" {
		return CheckResult{
			Status:         StatusWarn,
			Message:        "Container havn-dolt exists but was not created by havn",
			Recommendation: "Remove the container or rename it to avoid conflict",
		}
	}

	_, err = c.backend.ContainerExec(ctx, "havn-dolt", []string{"dolt", "sql", "-q", "SELECT 1"})
	if err != nil {
		return CheckResult{
			Status:         StatusError,
			Message:        "Dolt server is running but not accepting queries",
			Detail:         err.Error(),
			Recommendation: "Check 'docker logs havn-dolt' for errors; consider 'havn dolt stop && havn dolt start'",
		}
	}

	return CheckResult{
		Status:  StatusPass,
		Message: "Dolt server running",
	}
}

// --- 1.8 dolt_database ---

type doltDatabaseCheck struct {
	checkMetadata
	backend     Backend
	doltEnabled bool
	database    string
}

// NewDoltDatabaseCheck creates check 1.8: project database exists on server.
func NewDoltDatabaseCheck(backend Backend, doltEnabled bool, database string) Check {
	return &doltDatabaseCheck{
		checkMetadata: newHostCheckMetadata("dolt_database", []string{"dolt_server"}, defaultTimeout),
		backend:       backend,
		doltEnabled:   doltEnabled,
		database:      database,
	}
}

func (c *doltDatabaseCheck) Run(ctx context.Context) CheckResult {
	if !c.doltEnabled {
		return CheckResult{
			Status:  StatusSkip,
			Message: "Dolt not enabled",
		}
	}
	if c.database == "" {
		return CheckResult{
			Status:  StatusSkip,
			Message: "no project database configured",
		}
	}

	output, err := c.backend.ContainerExec(ctx, "havn-dolt", []string{"dolt", "sql", "-q", "SHOW DATABASES"})
	if err != nil {
		return CheckResult{
			Status:  StatusError,
			Message: "Failed to list databases",
			Detail:  err.Error(),
		}
	}

	for _, name := range dolt.ParseDatabaseNames(output) {
		if name == c.database {
			return CheckResult{
				Status:  StatusPass,
				Message: fmt.Sprintf("Dolt database '%s' exists", c.database),
			}
		}
	}

	return CheckResult{
		Status:         StatusWarn,
		Message:        fmt.Sprintf("Database '%s' does not exist on the shared server", c.database),
		Recommendation: fmt.Sprintf("Use beads migration workflows (run 'bd migrate --help' and see docs/dolt-beads-guide.md), or run 'bd init' inside the container for '%s'", c.database),
	}
}
