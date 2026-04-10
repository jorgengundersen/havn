// Package mount resolves bind mounts and named volumes for container creation.
package mount

import "fmt"

// InvalidMountEntryError is returned when a mount config entry cannot be
// parsed (e.g. missing mode or unknown mode).
type InvalidMountEntryError struct {
	Entry  string
	Reason string
}

func (e *InvalidMountEntryError) Error() string {
	return fmt.Sprintf("invalid mount entry %q: %s", e.Entry, e.Reason)
}
