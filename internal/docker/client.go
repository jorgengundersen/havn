package docker

import (
	"github.com/docker/docker/client"
)

// Client wraps the Docker SDK client. The SDK client is a private field,
// never exposed outside this package.
type Client struct {
	docker *client.Client
}

// NewClient creates a Client using environment-based configuration
// (DOCKER_HOST, DOCKER_API_VERSION, etc.). Construction is lazy — no
// connection is made until a method is called.
func NewClient() (*Client, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{docker: c}, nil
}

// NewClientWithHost creates a Client targeting a specific Docker host.
func NewClientWithHost(host string) (*Client, error) {
	c, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{docker: c}, nil
}
