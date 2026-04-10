package container

import "context"

// Backend abstracts the container operations needed by the container
// package. Consumer-defined per code-standards §4.
type Backend interface {
	// ContainerList returns containers matching the given label filters.
	ContainerList(ctx context.Context, filters map[string]string) ([]RawContainer, error)
}

// RawContainer holds the raw data returned by the backend for a single
// container. Labels are decoded by domain code, not by the backend.
type RawContainer struct {
	Name   string
	Image  string
	Status string
	Labels map[string]string
}
