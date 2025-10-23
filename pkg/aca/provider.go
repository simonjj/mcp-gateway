package aca

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
	"github.com/docker/mcp-gateway/pkg/runtime"
)

// ACAProvider implements RuntimeProvider for Azure Container Apps
type ACAProvider struct {
	metadata     *ACAMetadata
	resources    *ACAResources
	credential   *azidentity.DefaultAzureCredential
	client       *armappcontainers.ContainerAppsClient
	initialized  bool
}

// ErrorType represents different error categories
type ErrorType string

const (
	ErrorAuthentication ErrorType = "authentication"
	ErrorPermission     ErrorType = "permission"
	ErrorQuota          ErrorType = "quota"
	ErrorNotFound       ErrorType = "not_found"
	ErrorConfiguration  ErrorType = "configuration"
	ErrorTransient      ErrorType = "transient"
)

// ProviderError represents an ACA provider error
type ProviderError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s error: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s error: %s", e.Type, e.Message)
}

// NewACAProvider creates a new ACA runtime provider
func NewACAProvider() *ACAProvider {
	return &ACAProvider{}
}

// Initialize authenticates and prepares the ACA runtime
func (p *ACAProvider) Initialize(ctx context.Context) error {
	if p.initialized {
		return nil
	}

	// Step 1: Create DefaultAzureCredential for authentication
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return &ProviderError{
			Type:    ErrorAuthentication,
			Message: "failed to create Azure credential",
			Cause:   err,
		}
	}
	p.credential = cred

	// Step 2: Discover metadata from IMDS
	metadata, err := DiscoverMetadata(ctx)
	if err != nil {
		return &ProviderError{
			Type:    ErrorConfiguration,
			Message: "failed to discover ACA environment metadata",
			Cause:   err,
		}
	}
	p.metadata = metadata

	// Step 3: Create Container Apps client
	client, err := armappcontainers.NewContainerAppsClient(metadata.SubscriptionID, cred, nil)
	if err != nil {
		return &ProviderError{
			Type:    ErrorConfiguration,
			Message: "failed to create Container Apps client",
			Cause:   err,
		}
	}
	p.client = client

	// Step 4: Discover resource limits from gateway container
	resources, err := p.discoverResourceLimits(ctx)
	if err != nil {
		// Log warning but continue with defaults
		resources = DefaultResources()
	}
	p.resources = resources

	p.initialized = true
	return nil
}

// discoverResourceLimits queries the gateway container's resource allocation
func (p *ACAProvider) discoverResourceLimits(ctx context.Context) (*ACAResources, error) {
	if p.client == nil || p.metadata == nil {
		return nil, fmt.Errorf("client or metadata not initialized")
	}

	// Get current Container App configuration
	resp, err := p.client.Get(ctx, p.metadata.ResourceGroup, p.metadata.AppName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get container app: %w", err)
	}

	// Find gateway container (assume it's the first container or find by name pattern)
	if resp.Properties == nil || resp.Properties.Template == nil || resp.Properties.Template.Containers == nil {
		return nil, fmt.Errorf("no containers found in app template")
	}

	containers := resp.Properties.Template.Containers
	if len(containers) == 0 {
		return nil, fmt.Errorf("no containers found")
	}

	// Use first container's resources (gateway container)
	container := containers[0]
	if container.Resources == nil {
		return nil, fmt.Errorf("no resources defined for gateway container")
	}

	cpu := ""
	if container.Resources.CPU != nil {
		// CPU is a *float64, convert to string
		cpu = fmt.Sprintf("%.2f", *container.Resources.CPU)
	}

	memory := ""
	if container.Resources.Memory != nil {
		// Memory is a string pointer
		memory = *container.Resources.Memory
	}

	return ParseResources(cpu, memory)
}

// Cleanup removes all MCP server containers
func (p *ACAProvider) Cleanup(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	// Get current app configuration
	resp, err := p.client.Get(ctx, p.metadata.ResourceGroup, p.metadata.AppName, nil)
	if err != nil {
		return p.wrapError(err, "failed to get container app for cleanup")
	}

	if resp.Properties == nil || resp.Properties.Template == nil || resp.Properties.Template.Containers == nil {
		// No containers to clean up
		return nil
	}

	// Filter out MCP server containers (those with "mcp-server-" prefix)
	var remainingContainers []*armappcontainers.Container
	for _, container := range resp.Properties.Template.Containers {
		if container.Name != nil && !strings.HasPrefix(*container.Name, "mcp-server-") {
			remainingContainers = append(remainingContainers, container)
		}
	}

	// If no changes needed, skip update
	if len(remainingContainers) == len(resp.Properties.Template.Containers) {
		return nil
	}

	// Update app with cleaned container list
	resp.Properties.Template.Containers = remainingContainers

	poller, err := p.client.BeginUpdate(ctx, p.metadata.ResourceGroup, p.metadata.AppName, resp.ContainerApp, nil)
	if err != nil {
		return p.wrapError(err, "failed to start cleanup update")
	}

	// Wait for update to complete (with timeout)
	updateCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	_, err = poller.PollUntilDone(updateCtx, nil)
	if err != nil {
		return p.wrapError(err, "cleanup update failed")
	}

	return nil
}

// EnableServer creates and starts an MCP server container (stub for MVP)
func (p *ACAProvider) EnableServer(ctx context.Context, server runtime.ServerConfig) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}
	// TODO: Implement in US2
	return fmt.Errorf("EnableServer not yet implemented in MVP")
}

// DisableServer stops and removes an MCP server container (stub for MVP)
func (p *ACAProvider) DisableServer(ctx context.Context, serverName string) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}
	// TODO: Implement in US3
	return fmt.Errorf("DisableServer not yet implemented in MVP")
}

// ListServers returns status of all MCP server containers (stub for MVP)
func (p *ACAProvider) ListServers(ctx context.Context) ([]runtime.ServerStatus, error) {
	if !p.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}
	// TODO: Implement in US2
	return nil, fmt.Errorf("ListServers not yet implemented in MVP")
}

// GetServerStatus returns status of a specific server (stub for MVP)
func (p *ACAProvider) GetServerStatus(ctx context.Context, serverName string) (*runtime.ServerStatus, error) {
	if !p.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}
	// TODO: Implement in US2
	return nil, fmt.Errorf("GetServerStatus not yet implemented in MVP")
}

// wrapError wraps an error with appropriate error type based on the error content
func (p *ACAProvider) wrapError(err error, message string) error {
	errStr := strings.ToLower(err.Error())

	if strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") {
		return &ProviderError{
			Type:    ErrorAuthentication,
			Message: message + " (authentication failed)",
			Cause:   err,
		}
	}

	if strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden") {
		return &ProviderError{
			Type:    ErrorPermission,
			Message: message + " (permission denied - ensure System Assigned Identity has Contributor role)",
			Cause:   err,
		}
	}

	if strings.Contains(errStr, "404") || strings.Contains(errStr, "not found") {
		return &ProviderError{
			Type:    ErrorNotFound,
			Message: message + " (resource not found)",
			Cause:   err,
		}
	}

	if strings.Contains(errStr, "409") || strings.Contains(errStr, "429") || strings.Contains(errStr, "quota") {
		return &ProviderError{
			Type:    ErrorQuota,
			Message: message + " (quota exceeded or rate limited)",
			Cause:   err,
		}
	}

	// Default to transient error
	return &ProviderError{
		Type:    ErrorTransient,
		Message: message,
		Cause:   err,
	}
}
