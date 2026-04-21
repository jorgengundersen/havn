package cli_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/container"
)

type matrixStartService struct {
	called  bool
	lastCfg config.Config
	lastOpt container.StartOptions
}

func (s *matrixStartService) StartOrAttach(_ context.Context, cfg config.Config, _ string, _ func(string), opts container.StartOptions) (int, error) {
	s.called = true
	s.lastCfg = cfg
	s.lastOpt = opts

	scenario := filepath.Base(cfg.Env)
	switch scenario {
	case "missing_required_devshell":
		return 0, fmt.Errorf("validate required devShell %q in container \"havn-user-project\": missing devShell", cfg.Shell)
	case "optional_prepare_failure":
		return 0, fmt.Errorf("run optional startup capability havn-session-prepare in container \"havn-user-project\": prepare failed")
	default:
		return 0, nil
	}
}

type matrixEnterService struct {
	called bool
}

func (s *matrixEnterService) Enter(_ context.Context, _ string) (int, error) {
	s.called = true
	return 0, nil
}

func fixtureFlakeRef(t *testing.T, scenario string) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(wd, "testdata", "fixture_flakes", scenario)
}

func writeProjectConfig(t *testing.T, projectPath string, envRef string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(projectPath, ".havn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".havn", "config.toml"), []byte(fmt.Sprintf("env = %q\nshell = \"default\"\n", envRef)), 0o644))
}

func TestStartupContractMatrix_RootAndUpAndEnter(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectPath := filepath.Join(homeDir, "work", "sample-project")
	require.NoError(t, os.MkdirAll(projectPath, 0o755))

	tests := []struct {
		name       string
		scenario   string
		wantRootOK bool
		wantUpOK   bool
		errPart    string
	}{
		{name: "missing required devShell", scenario: "missing_required_devshell", wantRootOK: false, wantUpOK: false, errPart: "validate required devShell"},
		{name: "missing optional prepare capability", scenario: "missing_optional_prepare", wantRootOK: true, wantUpOK: true},
		{name: "optional prepare capability succeeds", scenario: "optional_prepare_success", wantRootOK: true, wantUpOK: true},
		{name: "optional prepare capability fails", scenario: "optional_prepare_failure", wantRootOK: false, wantUpOK: false, errPart: "havn-session-prepare"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envRef := fixtureFlakeRef(t, tt.scenario)
			writeProjectConfig(t, projectPath, envRef)

			startSvc := &matrixStartService{}
			_, _, rootErr := executeCommandWithDeps(cli.Deps{StartService: startSvc}, projectPath)
			if tt.wantRootOK {
				require.NoError(t, rootErr)
			} else {
				require.Error(t, rootErr)
				assert.Contains(t, rootErr.Error(), tt.errPart)
			}
			require.True(t, startSvc.called)
			assert.Equal(t, envRef, startSvc.lastCfg.Env)
			assert.Equal(t, container.StartupModeAttach, startSvc.lastOpt.Mode)

			startSvc = &matrixStartService{}
			_, _, upErr := executeCommandWithDeps(cli.Deps{StartService: startSvc}, "up", projectPath)
			if tt.wantUpOK {
				require.NoError(t, upErr)
			} else {
				require.Error(t, upErr)
				assert.Contains(t, upErr.Error(), "havn up:")
				assert.Contains(t, upErr.Error(), tt.errPart)
			}
			require.True(t, startSvc.called)
			assert.Equal(t, envRef, startSvc.lastCfg.Env)
			assert.Equal(t, container.StartupModeNoAttach, startSvc.lastOpt.Mode)

			enterSvc := &matrixEnterService{}
			_, _, enterErr := executeCommandWithDeps(cli.Deps{EnterService: enterSvc}, "enter", projectPath)
			require.NoError(t, enterErr)
			assert.True(t, enterSvc.called)
		})
	}
}
