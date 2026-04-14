package cli

import (
	"github.com/jorgengundersen/havn/internal/container"
	"github.com/jorgengundersen/havn/internal/doctor"
	"github.com/jorgengundersen/havn/internal/dolt"
	"github.com/jorgengundersen/havn/internal/volume"
)

var _ container.Backend = dockerContainerBackend{}
var _ container.StopBackend = dockerContainerBackend{}
var _ container.ImageBackend = dockerImageBackend{}
var _ doctor.Backend = (*dockerDoctorBackend)(nil)
var _ volume.Backend = (*dockerVolumeBackend)(nil)
var _ dolt.Backend = (*dockerDoltBackend)(nil)
var _ StartService = dockerStartService{}
var _ container.StartBackend = dockerStartBackend{}
var _ container.NetworkBackend = dockerStartBackend{}
var _ container.VolumeEnsurer = (*volume.Manager)(nil)
var _ container.DoltSetup = (*dolt.Setup)(nil)
var _ container.ExecBackend = dockerStartBackend{}
var _ container.MountResolver = hostMountResolver{}
