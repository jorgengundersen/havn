package main

import (
	"context"
	"os"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/volume"
)

type failingStartService struct{}

func (failingStartService) StartOrAttach(context.Context, config.Config, string, func(string), container.StartOptions) (int, error) {
	return 0, &volume.NotFoundError{Name: "havn-dolt-data"}
}

func main() {
	root := cli.NewRoot(cli.Deps{StartService: failingStartService{}})
	if err := root.Execute(); err != nil {
		jsonMode, _ := root.PersistentFlags().GetBool("json")
		verboseMode, _ := root.PersistentFlags().GetBool("verbose")
		out := cli.NewOutput(os.Stdout, os.Stderr, jsonMode, verboseMode)
		out.Error(err)
		os.Exit(cli.ExitCode(err))
	}
	os.Exit(0)
}
