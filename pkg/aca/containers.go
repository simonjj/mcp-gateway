package aca

import (
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
	"github.com/docker/mcp-gateway/pkg/runtime"
)

// ACAContainerSpec represents the specification for an MCP server container in ACA
type ACAContainerSpec struct {
	Name      string
	Image     string
	Resources *ACAResources
	Env       []EnvVar
	Probes    *ContainerProbes
}

// EnvVar represents an environment variable for a container
type EnvVar struct {
	Name      string
	Value     string
	SecretRef string // Optional: reference to a secret in the Container App environment
}

// ContainerProbes represents health/readiness probes for a container
type ContainerProbes struct {
	Liveness  *Probe
	Readiness *Probe
}

// Probe represents a container probe configuration
type Probe struct {
	HTTPGet           *HTTPGetAction
	InitialDelaySeconds int32
	PeriodSeconds      int32
}

// HTTPGetAction represents an HTTP GET probe
type HTTPGetAction struct {
	Path string
	Port int32
}

// buildContainerSpec converts a ServerConfig to an ARM Container Apps container specification
// This applies the naming convention, validates transport, and applies resource limits
func buildContainerSpec(config runtime.ServerConfig, resources *ACAResources) (*armappcontainers.Container, error) {
	// Validation
	if config.Name == "" {
		return nil, fmt.Errorf("server name is required")
	}

	if config.Image == "" {
		return nil, fmt.Errorf("image is required")
	}

	// Validate transport - stdio not supported in ACA mode
	if config.Transport == "stdio" {
		return nil, fmt.Errorf("stdio transport not supported in ACA mode")
	}

	if config.Transport != "streaming" && config.Transport != "sse" {
		return nil, fmt.Errorf("transport must be 'streaming' or 'sse'")
	}

	// Apply naming convention
	containerName := "mcp-server-" + config.Name

	// Default port if not specified
	port := config.Port
	if port == 0 {
		port = 8080
	}

	// Parse resource limits
	cpu, err := parseCPUFloat(resources.CPU)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU resource: %w", err)
	}

	// Build container spec
	container := &armappcontainers.Container{
		Name:  &containerName,
		Image: &config.Image,
		Resources: &armappcontainers.ContainerResources{
			CPU:    &cpu,
			Memory: &resources.Memory,
		},
	}

	// Add environment variables
	if len(config.Env) > 0 {
		envVars := make([]*armappcontainers.EnvironmentVar, 0, len(config.Env))
		for key, value := range config.Env {
			envVars = append(envVars, &armappcontainers.EnvironmentVar{
				Name:  strPtr(key),
				Value: strPtr(value),
			})
		}
		container.Env = envVars
	}

	// Add MCP transport type as environment variable for server config
	transportEnv := &armappcontainers.EnvironmentVar{
		Name:  strPtr("MCP_TRANSPORT"),
		Value: strPtr(config.Transport),
	}
	if container.Env == nil {
		container.Env = []*armappcontainers.EnvironmentVar{transportEnv}
	} else {
		container.Env = append(container.Env, transportEnv)
	}

	// Add port if transport requires HTTP
	if config.Transport == "streaming" || config.Transport == "sse" {
		portInt32 := int32(port)
		container.Probes = []*armappcontainers.ContainerAppProbe{
			{
				TCPSocket: &armappcontainers.ContainerAppProbeTCPSocket{
					Port: &portInt32,
				},
				InitialDelaySeconds: int32Ptr(5),
				PeriodSeconds:       int32Ptr(10),
			},
		}
	}

	return container, nil
}

// parseCPUFloat converts CPU string (e.g., "0.5", "500m") to float64
func parseCPUFloat(cpuStr string) (float64, error) {
	// If already a decimal number, parse it
	if len(cpuStr) == 0 {
		return 0, fmt.Errorf("empty CPU value")
	}

	// Handle millicores (e.g., "500m")
	if cpuStr[len(cpuStr)-1] == 'm' {
		millis := cpuStr[:len(cpuStr)-1]
		m, err := strconv.ParseFloat(millis, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid millicores value: %w", err)
		}
		return m / 1000.0, nil
	}

	// Handle cores (e.g., "0.5", "1", "2")
	cores, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid CPU cores value: %w", err)
	}

	return cores, nil
}

// Helper functions to create pointers
func strPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}
