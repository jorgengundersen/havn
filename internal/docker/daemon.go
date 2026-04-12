package docker

import (
	"context"
	"log/slog"
)

// DaemonInfo holds metadata about the Docker daemon.
type DaemonInfo struct {
	ServerVersion string
	APIVersion    string
}

// Ping checks if the Docker daemon is reachable. Returns
// *DaemonUnreachableError when it is not; nil when it is.
func (c *Client) Ping(ctx context.Context) error {
	c.logger().Debug("checking docker daemon reachability", slog.String("operation", "ping"))
	_, err := c.docker.Ping(ctx)
	if err != nil {
		return &DaemonUnreachableError{Host: c.docker.DaemonHost()}
	}
	c.logger().Debug("docker daemon reachable", slog.String("operation", "ping"))
	return nil
}

// Info returns metadata about the Docker daemon. Returns
// *DaemonUnreachableError when the daemon is unreachable.
func (c *Client) Info(ctx context.Context) (DaemonInfo, error) {
	c.logger().Debug("inspecting docker daemon", slog.String("operation", "info"))
	info, err := c.docker.Info(ctx)
	if err != nil {
		return DaemonInfo{}, &DaemonUnreachableError{Host: c.docker.DaemonHost()}
	}
	c.logger().Debug("docker daemon info loaded", slog.String("operation", "info"))
	return DaemonInfo{
		ServerVersion: info.ServerVersion,
		APIVersion:    c.docker.ClientVersion(),
	}, nil
}
