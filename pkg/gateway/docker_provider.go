package gateway

import (
	"context"

	"github.com/docker/mcp-gateway/pkg/docker"
	"github.com/docker/mcp-gateway/pkg/runtime"
)

// DockerProvider implements RuntimeProvider for Docker Engine
type DockerProvider struct {
	docker docker.Client
}

// NewDockerProvider creates a new Docker runtime provider
func NewDockerProvider(dockerClient interface{}) runtime.RuntimeProvider {
	// Type assert to docker.Client
	if dc, ok := dockerClient.(docker.Client); ok {
		return &DockerProvider{
			docker: dc,
		}
	}
	// Should never happen if called correctly, but return a nil provider
	return &DockerProvider{}
}

// Initialize authenticates and prepares the Docker runtime
func (d *DockerProvider) Initialize(ctx context.Context) error {
	// Docker client is already initialized, no additional setup needed
	return nil
}

// Cleanup removes all MCP server containers
func (d *DockerProvider) Cleanup(ctx context.Context) error {
	// TODO: Implement cleanup logic for MCP server containers
	// For MVP, this is a stub - full implementation in future
	return nil
}

// EnableServer creates and starts an MCP server container
func (d *DockerProvider) EnableServer(ctx context.Context, server runtime.ServerConfig) error {
	// TODO: Implement server enablement logic
	// For MVP, this is a stub - full implementation uses existing docker client pool logic
	return nil
}

// DisableServer stops and removes an MCP server container
func (d *DockerProvider) DisableServer(ctx context.Context, serverName string) error {
	// TODO: Implement server disablement logic
	// For MVP, this is a stub - full implementation in future
	return nil
}

// ListServers returns status of all MCP server containers
func (d *DockerProvider) ListServers(ctx context.Context) ([]runtime.ServerStatus, error) {
	// TODO: Implement listing logic
	// For MVP, this is a stub - full implementation in future
	return nil, nil
}

// GetServerStatus returns status of a specific server
func (d *DockerProvider) GetServerStatus(ctx context.Context, serverName string) (*runtime.ServerStatus, error) {
	// TODO: Implement status query logic
	// For MVP, this is a stub - full implementation in future
	return nil, nil
}
