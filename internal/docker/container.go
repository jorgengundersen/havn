package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockermount "github.com/docker/docker/api/types/mount"
	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// BindMount represents a host-to-container bind mount.
type BindMount struct {
	Source   string
	Target   string
	ReadOnly bool
}

// VolumeMount represents a named volume mount.
type VolumeMount struct {
	Name   string
	Target string
}

// MountInfo describes a mount attached to a container.
type MountInfo struct {
	Source string
	Target string
	Mode   string // e.g. "rw", "ro"
}

// ContainerInfo holds read-only state of a container.
type ContainerInfo struct {
	ID       string
	Name     string
	Image    string
	Status   string // e.g. "running", "exited"
	Labels   map[string]string
	Mounts   []MountInfo
	Networks []string
	Env      []string
}

// ContainerListFilters holds filter criteria for listing containers.
type ContainerListFilters struct {
	Labels     map[string]string // label key=value pairs to match
	NamePrefix string            // container name prefix filter
	Status     string            // e.g. "running", "exited"
}

// CreateOpts holds parameters for creating a container.
type CreateOpts struct {
	Image         string
	Name          string
	Network       string
	Ports         []string
	Env           map[string]string
	Labels        map[string]string
	BindMounts    []BindMount
	VolumeMounts  []VolumeMount
	RestartPolicy string
	TTY           bool
	Workdir       string
	Cmd           []string
	Entrypoint    []string
	User          string
	CPUs          int
	Memory        string
	MemorySwap    string
	AutoRemove    bool
}

// StopOpts holds parameters for stopping a container.
type StopOpts struct {
	Timeout int // seconds
}

// RemoveOpts holds parameters for removing a container.
type RemoveOpts struct {
	Force         bool
	RemoveVolumes bool
}

// ContainerCreate creates a container from the given options. Returns the
// container ID on success. Returns *ImageNotFoundError if the image does
// not exist locally.
func (c *Client) ContainerCreate(ctx context.Context, opts CreateOpts) (string, error) {
	cfg := &dockercontainer.Config{
		Image:      opts.Image,
		Env:        EnvSlice(opts.Env),
		Labels:     opts.Labels,
		Tty:        opts.TTY,
		WorkingDir: opts.Workdir,
		Cmd:        opts.Cmd,
		Entrypoint: opts.Entrypoint,
		User:       opts.User,
	}

	hostCfg := &dockercontainer.HostConfig{
		Mounts:     BuildMounts(opts.BindMounts, opts.VolumeMounts),
		AutoRemove: opts.AutoRemove,
		Resources: dockercontainer.Resources{
			NanoCPUs: int64(opts.CPUs) * 1e9,
			Memory:   ParseMemoryBytes(opts.Memory),
		},
	}

	exposedPorts, portBindings, err := BuildPortBindings(opts.Ports)
	if err != nil {
		return "", err
	}
	if len(exposedPorts) > 0 {
		cfg.ExposedPorts = exposedPorts
		hostCfg.PortBindings = portBindings
	}
	if opts.MemorySwap != "" {
		hostCfg.MemorySwap = ParseMemoryBytes(opts.MemorySwap)
	}
	if opts.RestartPolicy != "" {
		hostCfg.RestartPolicy = dockercontainer.RestartPolicy{
			Name: dockercontainer.RestartPolicyMode(opts.RestartPolicy),
		}
	}

	var networkCfg *dockernetwork.NetworkingConfig
	if opts.Network != "" {
		networkCfg = &dockernetwork.NetworkingConfig{
			EndpointsConfig: map[string]*dockernetwork.EndpointSettings{
				opts.Network: {},
			},
		}
	}

	resp, err := c.docker.ContainerCreate(ctx, cfg, hostCfg, networkCfg, nil, opts.Name)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return "", &ImageNotFoundError{Name: opts.Image}
		}
		return "", fmt.Errorf("docker create: %w", err)
	}
	return resp.ID, nil
}

// ContainerStart starts a container by name or ID. Returns
// *ContainerNotFoundError if the container does not exist.
func (c *Client) ContainerStart(ctx context.Context, nameOrID string) error {
	err := c.docker.ContainerStart(ctx, nameOrID, dockercontainer.StartOptions{})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return &ContainerNotFoundError{Name: nameOrID}
		}
		return fmt.Errorf("docker start: %w", err)
	}
	return nil
}

// ContainerStop stops a container by name or ID with the given options.
// Returns *ContainerNotFoundError if the container does not exist.
func (c *Client) ContainerStop(ctx context.Context, nameOrID string, opts StopOpts) error {
	timeout := opts.Timeout
	err := c.docker.ContainerStop(ctx, nameOrID, dockercontainer.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return &ContainerNotFoundError{Name: nameOrID}
		}
		return fmt.Errorf("docker stop: %w", err)
	}
	return nil
}

// ContainerRemove removes a container by name or ID. Returns
// *ContainerNotFoundError if the container does not exist.
func (c *Client) ContainerRemove(ctx context.Context, nameOrID string, opts RemoveOpts) error {
	err := c.docker.ContainerRemove(ctx, nameOrID, dockercontainer.RemoveOptions{
		Force:         opts.Force,
		RemoveVolumes: opts.RemoveVolumes,
	})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return &ContainerNotFoundError{Name: nameOrID}
		}
		return fmt.Errorf("docker remove: %w", err)
	}
	return nil
}

// ContainerList returns containers matching the given filters. Returns an
// empty slice (not nil) when no containers match.
func (c *Client) ContainerList(ctx context.Context, filters ContainerListFilters) ([]ContainerInfo, error) {
	args := dockerfilters.NewArgs()
	for k, v := range filters.Labels {
		args.Add("label", k+"="+v)
	}
	if filters.NamePrefix != "" {
		args.Add("name", filters.NamePrefix)
	}
	if filters.Status != "" {
		args.Add("status", filters.Status)
	}

	containers, err := c.docker.ContainerList(ctx, dockercontainer.ListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return nil, fmt.Errorf("docker list: %w", err)
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, ctr := range containers {
		name := ""
		if len(ctr.Names) > 0 {
			name = strings.TrimPrefix(ctr.Names[0], "/")
		}

		var mounts []MountInfo
		for _, m := range ctr.Mounts {
			source := m.Source
			if source == "" {
				source = m.Name
			}
			mounts = append(mounts, MountInfo{
				Source: source,
				Target: m.Destination,
				Mode:   m.Mode,
			})
		}

		var networks []string
		if ctr.NetworkSettings != nil {
			for netName := range ctr.NetworkSettings.Networks {
				networks = append(networks, netName)
			}
		}

		result = append(result, ContainerInfo{
			ID:       ctr.ID,
			Name:     name,
			Image:    ctr.Image,
			Status:   string(ctr.State),
			Labels:   ctr.Labels,
			Mounts:   mounts,
			Networks: networks,
		})
	}

	return result, nil
}

// ContainerInspect returns detailed information about a container by name or
// ID. Returns *ContainerNotFoundError if the container does not exist.
func (c *Client) ContainerInspect(ctx context.Context, nameOrID string) (ContainerInfo, error) {
	resp, err := c.docker.ContainerInspect(ctx, nameOrID)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return ContainerInfo{}, &ContainerNotFoundError{Name: nameOrID}
		}
		return ContainerInfo{}, fmt.Errorf("docker inspect: %w", err)
	}

	info := ContainerInfo{
		ID:    resp.ID,
		Name:  resp.Name,
		Image: resp.Config.Image,
	}

	// Strip leading "/" from Docker's container name.
	if len(info.Name) > 0 && info.Name[0] == '/' {
		info.Name = info.Name[1:]
	}

	if resp.State != nil {
		info.Status = string(resp.State.Status)
	}

	if resp.Config != nil {
		info.Labels = resp.Config.Labels
		info.Env = resp.Config.Env
	}

	for _, m := range resp.Mounts {
		source := m.Source
		if source == "" {
			source = m.Name
		}
		info.Mounts = append(info.Mounts, MountInfo{
			Source: source,
			Target: m.Destination,
			Mode:   m.Mode,
		})
	}

	if resp.NetworkSettings != nil {
		for name := range resp.NetworkSettings.Networks {
			info.Networks = append(info.Networks, name)
		}
	}

	return info, nil
}

// EnvSlice converts a map of environment variables to the Docker SDK's
// "KEY=VALUE" slice format.
func EnvSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	s := make([]string, 0, len(env))
	for k, v := range env {
		s = append(s, k+"="+v)
	}
	return s
}

// BuildMounts converts havn bind and volume mounts to Docker SDK mounts.
func BuildMounts(binds []BindMount, volumes []VolumeMount) []dockermount.Mount {
	mounts := make([]dockermount.Mount, 0, len(binds)+len(volumes))
	for _, b := range binds {
		mounts = append(mounts, dockermount.Mount{
			Type:     dockermount.TypeBind,
			Source:   b.Source,
			Target:   b.Target,
			ReadOnly: b.ReadOnly,
		})
	}
	for _, v := range volumes {
		mounts = append(mounts, dockermount.Mount{
			Type:   dockermount.TypeVolume,
			Source: v.Name,
			Target: v.Target,
		})
	}
	return mounts
}

// BuildPortBindings converts host:container(/proto) mappings into Docker types.
func BuildPortBindings(ports []string) (nat.PortSet, nat.PortMap, error) {
	if len(ports) == 0 {
		return nil, nil, nil
	}

	exposed := make(nat.PortSet, len(ports))
	bindings := make(nat.PortMap, len(ports))
	for _, mapping := range ports {
		hostPort, containerPort, protocol, ok := parsePortMapping(mapping)
		if !ok {
			return nil, nil, fmt.Errorf("invalid port mapping %q", mapping)
		}
		port := nat.Port(containerPort + "/" + protocol)
		exposed[port] = struct{}{}
		bindings[port] = append(bindings[port], nat.PortBinding{HostIP: "", HostPort: hostPort})
	}

	return exposed, bindings, nil
}

func parsePortMapping(mapping string) (hostPort, containerPort, protocol string, ok bool) {
	parts := strings.SplitN(mapping, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", false
	}

	hostPort = parts[0]
	containerPart := parts[1]
	protocol = "tcp"
	if slash := strings.IndexRune(containerPart, '/'); slash >= 0 {
		containerPort = containerPart[:slash]
		if containerPort == "" {
			return "", "", "", false
		}
		protocol = containerPart[slash+1:]
		if protocol == "" {
			return "", "", "", false
		}
	} else {
		containerPort = containerPart
	}

	if _, err := strconv.Atoi(hostPort); err != nil {
		return "", "", "", false
	}
	if _, err := strconv.Atoi(containerPort); err != nil {
		return "", "", "", false
	}
	if protocol != "tcp" && protocol != "udp" {
		return "", "", "", false
	}

	return hostPort, containerPort, protocol, true
}

// ParseMemoryBytes converts a memory string like "4g" or "512m" to bytes.
// Returns 0 for empty or unrecognized strings.
func ParseMemoryBytes(s string) int64 {
	if s == "" {
		return 0
	}
	n := int64(0)
	for i, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		} else {
			suffix := s[i:]
			switch suffix {
			case "g", "G":
				return n * 1024 * 1024 * 1024
			case "m", "M":
				return n * 1024 * 1024
			case "k", "K":
				return n * 1024
			default:
				return n
			}
		}
	}
	return n
}
