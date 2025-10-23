package runtime

import (
	"context"
	"time"
)

// RuntimeProvider is an abstraction for container runtime backends (Docker, ACA)
type RuntimeProvider interface {
	// Initialize authenticates and prepares the runtime
	Initialize(ctx context.Context) error

	// Cleanup removes all MCP server containers
	Cleanup(ctx context.Context) error

	// EnableServer creates and starts an MCP server container
	EnableServer(ctx context.Context, server ServerConfig) error

	// DisableServer stops and removes an MCP server container
	DisableServer(ctx context.Context, serverName string) error

	// ListServers returns status of all MCP server containers
	ListServers(ctx context.Context) ([]ServerStatus, error)

	// GetServerStatus returns status of a specific server
	GetServerStatus(ctx context.Context, serverName string) (*ServerStatus, error)
}

// ServerConfig is the configuration for enabling an MCP server
type ServerConfig struct {
	Name      string            // Server name (e.g., "duckduckgo")
	Image     string            // Container image (e.g., "docker/mcp-duckduckgo:latest")
	Transport string            // MCP transport ("streaming", "sse", or "stdio")
	Port      int               // HTTP port for transport (default: 8080)
	Env       map[string]string // Environment variables
	Secrets   map[string]string // Secret references (provider-specific)
}

// ServerStatus represents the runtime status of an MCP server container
type ServerStatus struct {
	Name     string         // Server name
	State    ContainerState // Container state
	Image    string         // Container image
	Started  *time.Time     // Start time (nil if not started)
	ExitCode *int           // Exit code if crashed (nil if running)
	Message  string         // Status message or error
}

// ContainerState represents the state of a container
type ContainerState string

const (
	StateRunning  ContainerState = "running"
	StateStopped  ContainerState = "stopped"
	StateCrashed  ContainerState = "crashed"
	StateCreating ContainerState = "creating"
	StateUnknown  ContainerState = "unknown"
)
