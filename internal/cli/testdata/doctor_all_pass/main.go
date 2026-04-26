package main

import (
	"context"
	"os"

	"github.com/jorgengundersen/havn/internal/cli"
	"github.com/jorgengundersen/havn/internal/doctor"
)

type passingDoctorBackend struct{}

func (passingDoctorBackend) Ping(context.Context) error { return nil }

func (passingDoctorBackend) Info(context.Context) (doctor.RuntimeInfo, error) {
	return doctor.RuntimeInfo{Version: "24.0.7", APIVersion: "1.43"}, nil
}

func (passingDoctorBackend) ImageInspect(context.Context, string) (doctor.ImageInfo, bool, error) {
	return doctor.ImageInfo{Created: "2026-04-01"}, true, nil
}

func (passingDoctorBackend) NetworkInspect(context.Context, string) (doctor.NetworkInfo, bool, error) {
	return doctor.NetworkInfo{}, true, nil
}

func (passingDoctorBackend) VolumeInspect(context.Context, string) (bool, error) {
	return true, nil
}

func (passingDoctorBackend) ContainerInspect(context.Context, string) (doctor.ContainerInfo, bool, error) {
	return doctor.ContainerInfo{}, false, nil
}

func (passingDoctorBackend) ContainerExec(context.Context, string, []string) (string, error) {
	return "", nil
}

func (passingDoctorBackend) ListContainers(context.Context, map[string]string) ([]string, error) {
	return nil, nil
}

func main() {
	root := cli.NewRoot(cli.Deps{DoctorBackend: passingDoctorBackend{}})
	if err := root.Execute(); err != nil {
		jsonMode, _ := root.PersistentFlags().GetBool("json")
		verboseMode, _ := root.PersistentFlags().GetBool("verbose")
		out := cli.NewOutput(os.Stdout, os.Stderr, jsonMode, verboseMode)
		out.Error(err)
		os.Exit(cli.ExitCode(err))
	}
	os.Exit(0)
}
