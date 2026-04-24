package container

import (
	"context"
	"strconv"
	"strings"

	"github.com/jorgengundersen/havn/internal/name"
)

// Label keys used by havn to store container metadata.
const (
	LabelManagedBy  = "managed-by"
	LabelPath       = "havn.path"
	LabelShell      = "havn.shell"
	LabelCPUs       = "havn.cpus"
	LabelMemory     = "havn.memory"
	LabelMemorySwap = "havn.memory_swap"
	LabelDolt       = "havn.dolt"
)

// List returns all havn-managed containers by querying the backend for
// containers with the managed-by=havn label.
func List(ctx context.Context, backend Backend) ([]Info, error) {
	raw, err := backend.ContainerList(ctx, map[string]string{
		LabelManagedBy: "havn",
	})
	if err != nil {
		return nil, err
	}

	result := make([]Info, 0, len(raw))
	for _, r := range raw {
		if !isRunningManagedContainer(r) {
			continue
		}
		result = append(result, containerInfoFromRaw(r))
	}
	return result, nil
}

func isRunningManagedContainer(r RawContainer) bool {
	return strings.ToLower(strings.TrimSpace(r.Status)) == "running"
}

// containerInfoFromRaw decodes a RawContainer's labels into an Info.
func containerInfoFromRaw(r RawContainer) Info {
	cpus, _ := strconv.Atoi(r.Labels[LabelCPUs])
	dolt, _ := strconv.ParseBool(r.Labels[LabelDolt])

	return Info{
		Name:       name.ContainerName(r.Name),
		Path:       r.Labels[LabelPath],
		Image:      r.Image,
		Status:     r.Status,
		Shell:      r.Labels[LabelShell],
		CPUs:       cpus,
		Memory:     r.Labels[LabelMemory],
		MemorySwap: r.Labels[LabelMemorySwap],
		Dolt:       dolt,
	}
}
