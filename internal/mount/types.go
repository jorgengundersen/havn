package mount

// Spec describes a single mount to attach to a container.
type Spec struct {
	Source   string
	Target   string
	ReadOnly bool
	Type     string // "bind" or "volume"
}

// ResolveResult holds all mounts and environment variables needed for
// container creation.
type ResolveResult struct {
	Mounts []Spec
	Env    map[string]string
}

// ResolveOpts bundles injectable I/O callbacks so Resolve stays pure
// from the caller's perspective.
type ResolveOpts struct {
	Glob        func(pattern string) ([]string, error)
	Exists      func(path string) bool
	SSHAuthSock string
}
