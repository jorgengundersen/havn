package docker

import (
	"log/slog"
	"os"

	"github.com/docker/docker/client"
)

// Client wraps the Docker SDK client. The SDK client is a private field,
// never exposed outside this package.
type Client struct {
	docker *client.Client
	log    *slog.Logger
}

// NewClient creates a Client using environment-based configuration
// (DOCKER_HOST, DOCKER_API_VERSION, etc.). Construction is lazy — no
// connection is made until a method is called.
func NewClient() (*Client, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{docker: c, log: defaultLogger()}, nil
}

// NewClientWithHost creates a Client targeting a specific Docker host.
func NewClientWithHost(host string) (*Client, error) {
	c, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{docker: c, log: defaultLogger()}, nil
}

// NewClientWithHostAndLogger creates a Client targeting a specific Docker host
// and uses the injected logger for structured diagnostics.
func NewClientWithHostAndLogger(host string, logger *slog.Logger) (*Client, error) {
	c, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{docker: c, log: withDockerComponent(logger)}, nil
}

// SetLogger replaces the client's logger used for structured diagnostics.
func (c *Client) SetLogger(logger *slog.Logger) {
	c.log = withDockerComponent(logger)
}

func (c *Client) logger() *slog.Logger {
	if c.log == nil {
		c.log = defaultLogger()
	}
	return c.log
}

func defaultLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})).With(slog.String("component", "docker"))
}

func withDockerComponent(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return defaultLogger()
	}
	return logger.With(slog.String("component", "docker"))
}
