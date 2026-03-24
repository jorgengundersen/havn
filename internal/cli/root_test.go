package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/cli"
)

func TestNewRoot_ReturnsCommand(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	require.NotNil(t, root)
	assert.Equal(t, "havn [flags] [path]", root.Use)
}

func TestNewRoot_SilencesCobraOutput(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	assert.True(t, root.SilenceErrors)
	assert.True(t, root.SilenceUsage)
}

func TestExecute_ReturnsNonZero_WhenNotImplemented(t *testing.T) {
	code := cli.Execute()

	assert.Equal(t, 1, code)
}

func TestNewRoot_PersistentFlags(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	jsonFlag := root.PersistentFlags().Lookup("json")
	require.NotNil(t, jsonFlag, "--json persistent flag should exist")
	assert.Equal(t, "false", jsonFlag.DefValue)

	verboseFlag := root.PersistentFlags().Lookup("verbose")
	require.NotNil(t, verboseFlag, "--verbose persistent flag should exist")
	assert.Equal(t, "false", verboseFlag.DefValue)

	configFlag := root.PersistentFlags().Lookup("config")
	require.NotNil(t, configFlag, "--config persistent flag should exist")
	assert.Equal(t, "", configFlag.DefValue)
}

func TestNewRoot_LocalContainerFlags(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	flags := map[string]string{
		"shell":  "",
		"env":    "",
		"memory": "",
		"port":   "",
		"image":  "",
	}
	for name, defVal := range flags {
		f := root.Flags().Lookup(name)
		require.NotNil(t, f, "--%s local flag should exist", name)
		assert.Equal(t, defVal, f.DefValue, "--%s default", name)
	}

	cpusFlag := root.Flags().Lookup("cpus")
	require.NotNil(t, cpusFlag, "--cpus local flag should exist")
	assert.Equal(t, "0", cpusFlag.DefValue)

	noDoltFlag := root.Flags().Lookup("no-dolt")
	require.NotNil(t, noDoltFlag, "--no-dolt local flag should exist")
	assert.Equal(t, "false", noDoltFlag.DefValue)
}

func TestNewRoot_ContainerFlags_NotPersistent(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	localOnly := []string{"shell", "env", "cpus", "memory", "port", "no-dolt", "image"}
	for _, name := range localOnly {
		f := root.PersistentFlags().Lookup(name)
		assert.Nil(t, f, "--%s should not be a persistent flag", name)
	}
}

func TestNewRoot_PathArgDefaultsToDot(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	root.SetArgs([]string{})

	err := root.Execute()

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
	assert.Contains(t, root.Use, "[path]")
}

func TestNewRoot_AcceptsExplicitPath(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	root.SetArgs([]string{"/some/path"})

	err := root.Execute()

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
}

func TestNewRoot_RejectsTooManyArgs(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	root.SetArgs([]string{"/path/one", "/path/two"})

	err := root.Execute()

	require.Error(t, err)
	assert.NotErrorIs(t, err, cli.ErrNotImplemented)
}

func TestNewRoot_HelpIncludesAllFlags(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetArgs([]string{"--help"})

	err := root.Execute()
	require.NoError(t, err)

	help := out.String()
	flags := []string{
		"--json", "--verbose", "--config",
		"--shell", "--env", "--cpus", "--memory", "--port", "--no-dolt", "--image",
	}
	for _, f := range flags {
		assert.Contains(t, help, f, "help output should include %s", f)
	}
}

func TestNewRoot_HasVersion(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})

	assert.NotEmpty(t, root.Version, "root command should have a version set")
}

func TestNewRoot_RunE_ReturnsNotImplemented(t *testing.T) {
	root := cli.NewRoot(cli.Deps{})
	root.SetArgs([]string{})

	err := root.Execute()

	require.Error(t, err)
	assert.ErrorIs(t, err, cli.ErrNotImplemented)
}
