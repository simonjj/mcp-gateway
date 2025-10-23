package gateway

import (
	"fmt"

	"github.com/docker/mcp-gateway/pkg/aca"
	"github.com/docker/mcp-gateway/pkg/log"
	"github.com/docker/mcp-gateway/pkg/runtime"
)

// NewRuntimeProvider creates a RuntimeProvider based on the runtime mode
// Mode: "ACA" for Azure Container Apps, otherwise defaults to Docker Engine
// dockerClient should be a docker.Client interface implementation
func NewRuntimeProvider(mode string, dockerClient interface{}) (runtime.RuntimeProvider, error) {
	switch mode {
	case "ACA":
		log.Log("Runtime mode: Azure Container Apps (ACA)")
		return aca.NewACAProvider(), nil
	default:
		// Default to Docker provider for empty string, "docker", or any invalid value
		if mode == "" {
			log.Log("Runtime mode: Docker Engine (default - MCP_RUNTIME not set)")
		} else if mode == "docker" {
			log.Log("Runtime mode: Docker Engine (MCP_RUNTIME=docker)")
		} else {
			log.Log(fmt.Sprintf("Runtime mode: Docker Engine (MCP_RUNTIME=%s not recognized, using default)", mode))
		}
		return NewDockerProvider(dockerClient), nil
	}
}
