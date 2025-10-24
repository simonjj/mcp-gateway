package aca

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
	"github.com/docker/mcp-gateway/pkg/log"
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

	log.Logf("ACA Provider: Starting initialization...")

	// Step 1: Create DefaultAzureCredential for authentication
	log.Logf("ACA Provider: Creating Azure credential using DefaultAzureCredential (Managed Identity)...")
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Logf("ACA Provider: Failed to create Azure credential: %v", err)
		return &ProviderError{
			Type:    ErrorAuthentication,
			Message: "failed to create Azure credential",
			Cause:   err,
		}
	}
	p.credential = cred
	log.Logf("ACA Provider: ✓ Azure credential created successfully")

	// Step 2: Discover metadata from environment variables (ACA standard) or IMDS (Azure VMs)
	log.Logf("ACA Provider: Discovering environment metadata...")
	metadata, err := DiscoverMetadata(ctx)
	if err != nil {
		log.Logf("ACA Provider: Failed to discover metadata: %v", err)
		return &ProviderError{
			Type:    ErrorConfiguration,
			Message: "failed to discover ACA environment metadata. Ensure these environment variables are set: AZURE_SUBSCRIPTION_ID, AZURE_RESOURCE_GROUP, and AZURE_APP_NAME (or CONTAINER_APP_NAME)",
			Cause:   err,
		}
	}
	p.metadata = metadata
	log.Logf("ACA Provider: ✓ Metadata discovered:")
	log.Logf("ACA Provider:   - Subscription ID: %s", metadata.SubscriptionID)
	log.Logf("ACA Provider:   - Resource Group: %s", metadata.ResourceGroup)
	log.Logf("ACA Provider:   - App Name: %s", metadata.AppName)
	if metadata.Location != "" {
		log.Logf("ACA Provider:   - Location: %s", metadata.Location)
	}

	// Step 3: Create Container Apps client
	log.Logf("ACA Provider: Creating Container Apps client...")
	client, err := armappcontainers.NewContainerAppsClient(metadata.SubscriptionID, cred, nil)
	if err != nil {
		log.Logf("ACA Provider: Failed to create Container Apps client: %v", err)
		return &ProviderError{
			Type:    ErrorConfiguration,
			Message: "failed to create Container Apps client",
			Cause:   err,
		}
	}
	p.client = client
	log.Logf("ACA Provider: ✓ Container Apps client created successfully")

	// Step 4: Discover resource limits from gateway container
	log.Logf("ACA Provider: Discovering resource limits...")
	resources, err := p.discoverResourceLimits(ctx)
	if err != nil {
		// Log warning but continue with defaults
		log.Logf("ACA Provider: Could not discover resources, using defaults: %v", err)
		resources = DefaultResources()
	}
	p.resources = resources
	log.Logf("ACA Provider: ✓ Using resources: CPU=%s, Memory=%s", resources.CPU, resources.Memory)

	p.initialized = true
	log.Logf("ACA Provider: ✓✓✓ Initialization completed successfully ✓✓✓")
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

// EnableServer creates and starts an MCP server container
func (p *ACAProvider) EnableServer(ctx context.Context, server runtime.ServerConfig) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	log.Logf("Enabling MCP server '%s' with image '%s' and transport '%s'", server.Name, server.Image, server.Transport)

	// Validate transport - stdio not supported
	if server.Transport == "stdio" {
		log.Logf("Rejected server '%s': stdio transport not supported in ACA mode", server.Name)
		return fmt.Errorf("stdio transport not supported in ACA mode")
	}

	// Build container spec
	containerSpec, err := buildContainerSpec(server, p.resources)
	if err != nil {
		log.Logf("Failed to build container spec for server '%s': %v", server.Name, err)
		return p.wrapError(err, "failed to build container spec")
	}

	log.Logf("Container spec for '%s': CPU=%s, Memory=%s", server.Name, p.resources.CPU, p.resources.Memory)

	// Get current app configuration
	resp, err := p.client.Get(ctx, p.metadata.ResourceGroup, p.metadata.AppName, nil)
	if err != nil {
		return p.wrapError(err, "failed to get container app")
	}

	if resp.Properties == nil || resp.Properties.Template == nil {
		return fmt.Errorf("invalid container app response: missing template")
	}

	// Check if container already exists
	containers := resp.Properties.Template.Containers
	for _, c := range containers {
		if c.Name != nil && *c.Name == *containerSpec.Name {
			// Container already exists, update is idempotent
			log.Logf("Server '%s' already enabled (idempotent)", server.Name)
			return nil
		}
	}

	// Add new container to template
	resp.Properties.Template.Containers = append(containers, containerSpec)

	// Update the app
	poller, err := p.client.BeginUpdate(ctx, p.metadata.ResourceGroup, p.metadata.AppName, resp.ContainerApp, nil)
	if err != nil {
		return p.wrapError(err, fmt.Sprintf("failed to start update for server %s", server.Name))
	}

	// Wait for update to complete (with timeout)
	log.Logf("Waiting for server '%s' to be enabled (timeout: 5m)", server.Name)
	updateCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	_, err = poller.PollUntilDone(updateCtx, nil)
	if err != nil {
		log.Logf("Failed to enable server '%s': %v", server.Name, err)
		return p.wrapError(err, fmt.Sprintf("server %s update failed", server.Name))
	}

	log.Logf("Successfully enabled server '%s'", server.Name)
	return nil
}

// DisableServer stops and removes an MCP server container
func (p *ACAProvider) DisableServer(ctx context.Context, serverName string) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	log.Logf("Disabling MCP server '%s'", serverName)

	// Get current app configuration
	resp, err := p.client.Get(ctx, p.metadata.ResourceGroup, p.metadata.AppName, nil)
	if err != nil {
		return p.wrapError(err, "failed to get container app")
	}

	if resp.Properties == nil || resp.Properties.Template == nil {
		return fmt.Errorf("invalid container app response: missing template")
	}

	// Filter out the target container
	containerName := "mcp-server-" + serverName
	containers := resp.Properties.Template.Containers
	remainingContainers := make([]*armappcontainers.Container, 0, len(containers))
	found := false

	for _, c := range containers {
		if c.Name != nil && *c.Name == containerName {
			found = true
			continue // Skip this container (remove it)
		}
		remainingContainers = append(remainingContainers, c)
	}

	// Idempotent: if container not found, it's already removed
	if !found {
		log.Logf("Server '%s' already disabled (idempotent)", serverName)
		return nil
	}

	log.Logf("Removing server '%s' from container app", serverName)

	// Update container list
	resp.Properties.Template.Containers = remainingContainers

	// Update the app
	poller, err := p.client.BeginUpdate(ctx, p.metadata.ResourceGroup, p.metadata.AppName, resp.ContainerApp, nil)
	if err != nil {
		log.Logf("Failed to start removal for server '%s': %v", serverName, err)
		return p.wrapError(err, fmt.Sprintf("failed to start removal for server %s", serverName))
	}

	// Wait for update to complete (with timeout)
	log.Logf("Waiting for server '%s' to be disabled (timeout: 5m)", serverName)
	updateCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	_, err = poller.PollUntilDone(updateCtx, nil)
	if err != nil {
		log.Logf("Failed to disable server '%s': %v", serverName, err)
		return p.wrapError(err, fmt.Sprintf("server %s removal failed", serverName))
	}

	log.Logf("Successfully disabled server '%s'", serverName)
	return nil
}

// ListServers returns a list of all MCP server containers
func (p *ACAProvider) ListServers(ctx context.Context) ([]runtime.ServerStatus, error) {
	if !p.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	log.Logf("Listing MCP servers")

	// Get current app configuration
	resp, err := p.client.Get(ctx, p.metadata.ResourceGroup, p.metadata.AppName, nil)
	if err != nil {
		return nil, p.wrapError(err, "failed to get container app")
	}

	if resp.Properties == nil || resp.Properties.Template == nil {
		return nil, fmt.Errorf("invalid container app response: missing template")
	}

	// Filter containers with MCP server prefix
	containers := resp.Properties.Template.Containers
	statuses := make([]runtime.ServerStatus, 0)

	for _, c := range containers {
		if c.Name == nil || !strings.HasPrefix(*c.Name, "mcp-server-") {
			continue
		}

		// Extract server name (remove "mcp-server-" prefix)
		serverName := strings.TrimPrefix(*c.Name, "mcp-server-")

		status := runtime.ServerStatus{
			Name:  serverName,
			State: runtime.StateRunning, // ACA manages lifecycle, assume running if in template
			Image: "",
		}

		if c.Image != nil {
			status.Image = *c.Image
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetServerStatus returns status of a specific server
func (p *ACAProvider) GetServerStatus(ctx context.Context, serverName string) (*runtime.ServerStatus, error) {
	if !p.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Get current app configuration
	resp, err := p.client.Get(ctx, p.metadata.ResourceGroup, p.metadata.AppName, nil)
	if err != nil {
		return nil, p.wrapError(err, "failed to get container app")
	}

	if resp.Properties == nil || resp.Properties.Template == nil {
		return nil, fmt.Errorf("invalid container app response: missing template")
	}

	// Find the target container
	containerName := "mcp-server-" + serverName
	containers := resp.Properties.Template.Containers

	for _, c := range containers {
		if c.Name != nil && *c.Name == containerName {
			status := &runtime.ServerStatus{
				Name:  serverName,
				State: runtime.StateRunning, // ACA manages lifecycle
				Image: "",
			}

			if c.Image != nil {
				status.Image = *c.Image
			}

			return status, nil
		}
	}

	// Container not found
	return &runtime.ServerStatus{
		Name:    serverName,
		State:   runtime.StateUnknown,
		Message: "server not found",
	}, nil
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
		quotaMsg := message + " (quota exceeded or rate limited - check Azure subscription quotas and container app limits)"
		return &ProviderError{
			Type:    ErrorQuota,
			Message: quotaMsg,
			Cause:   err,
		}
	}

	// Image pull failures
	if strings.Contains(errStr, "imagepull") || strings.Contains(errStr, "image pull") ||
		strings.Contains(errStr, "pull access denied") || strings.Contains(errStr, "manifest unknown") {
		imageMsg := message + " (image pull failed - verify image exists and container registry is accessible. For private registries, ensure managed identity has AcrPull role)"
		return &ProviderError{
			Type:    ErrorConfiguration,
			Message: imageMsg,
			Cause:   err,
		}
	}

	// Timeout errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		timeoutMsg := message + " (operation timed out - Azure Container Apps may be under heavy load or experiencing issues)"
		return &ProviderError{
			Type:    ErrorTransient,
			Message: timeoutMsg,
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
