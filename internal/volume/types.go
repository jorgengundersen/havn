// Package volume manages havn volume lifecycle — listing expected volumes,
// checking existence, and ensuring volumes exist during container startup.
package volume

// Entry represents a single havn volume with its mount point and existence status.
type Entry struct {
	Name   string `json:"name"`
	Mount  string `json:"mount"`
	Exists bool   `json:"exists"`
}
