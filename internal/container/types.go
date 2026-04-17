package container

import "github.com/jorgengundersen/havn/internal/name"

// Info holds metadata for a havn-managed container.
// Fields match the havn list --json output shape.
type Info struct {
	Name       name.ContainerName `json:"name"`
	Path       string             `json:"path"`
	Image      string             `json:"image"`
	Status     string             `json:"status"`
	Shell      string             `json:"shell"`
	CPUs       int                `json:"cpus"`
	Memory     string             `json:"memory"`
	MemorySwap string             `json:"memory_swap"`
	Dolt       bool               `json:"dolt"`
}
