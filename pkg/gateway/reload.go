package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/docker/mcp-gateway/pkg/aca"
	"github.com/docker/mcp-gateway/pkg/log"
	"github.com/docker/mcp-gateway/pkg/prompts"
	"github.com/docker/mcp-gateway/pkg/runtime"
)

func (g *Gateway) reloadConfiguration(ctx context.Context, configuration Configuration, serverNames []string, clientConfig *clientConfig) error {
	// Which servers are enabled in the registry.yaml?
	if len(serverNames) == 0 {
		serverNames = configuration.ServerNames()
	}
	if len(serverNames) == 0 {
		log.Log("- No server is enabled")
	} else {
		log.Log("- Those servers are enabled:", strings.Join(serverNames, ", "))
	}

	// In ACA mode, enable servers via runtime provider before listing capabilities
	if g.RuntimeMode == "ACA" {
		if err := g.enableServersInACA(ctx, serverNames, configuration); err != nil {
			return err
		}
	}

	// List all the available tools.
	startList := time.Now()
	log.Log("- Listing MCP tools...")
	capabilities, err := g.listCapabilities(ctx, serverNames, clientConfig)
	if err != nil {
		return fmt.Errorf("listing resources: %w", err)
	}
	log.Log(">", len(capabilities.Tools), "tools listed in", time.Since(startList))

	// Update capabilities
	// Clear existing capabilities per server and register new ones

	// Lock for reading/writing capability tracking
	g.capabilitiesMu.Lock()
	defer g.capabilitiesMu.Unlock()

	// Clear all existing capabilities from tracked servers
	for _, oldCaps := range g.serverCapabilities {
		if len(oldCaps.ToolNames) > 0 {
			g.mcpServer.RemoveTools(oldCaps.ToolNames...)
		}
		if len(oldCaps.PromptNames) > 0 {
			g.mcpServer.RemovePrompts(oldCaps.PromptNames...)
		}
		if len(oldCaps.ResourceURIs) > 0 {
			g.mcpServer.RemoveResources(oldCaps.ResourceURIs...)
		}
		if len(oldCaps.ResourceTemplateURIs) > 0 {
			g.mcpServer.RemoveResourceTemplates(oldCaps.ResourceTemplateURIs...)
		}
	}

	// Clear the tracking map - we'll rebuild it
	g.serverCapabilities = make(map[string]*ServerCapabilities)

	// Add new capabilities and track them per server
	for _, tool := range capabilities.Tools {
		g.mcpServer.AddTool(tool.Tool, tool.Handler)

		// Track by server
		if g.serverCapabilities[tool.ServerName] == nil {
			g.serverCapabilities[tool.ServerName] = &ServerCapabilities{}
		}
		g.serverCapabilities[tool.ServerName].ToolNames = append(
			g.serverCapabilities[tool.ServerName].ToolNames,
			tool.Tool.Name,
		)
	}

	// Add internal tools when dynamic-tools feature is enabled
	if g.DynamicTools {
		log.Log("- Adding internal tools (dynamic-tools feature enabled)")

		// Add mcp-find tool
		mcpFindTool := g.createMcpFindTool(configuration)
		g.mcpServer.AddTool(mcpFindTool.Tool, mcpFindTool.Handler)

		// Add mcp-add tool
		mcpAddTool := g.createMcpAddTool(clientConfig)
		g.mcpServer.AddTool(mcpAddTool.Tool, mcpAddTool.Handler)

		// Add mcp-remove tool
		mcpRemoveTool := g.createMcpRemoveTool()
		g.mcpServer.AddTool(mcpRemoveTool.Tool, mcpRemoveTool.Handler)

		// Add mcp-registry-import tool
		mcpRegistryImportTool := g.createMcpRegistryImportTool(configuration, clientConfig)
		g.mcpServer.AddTool(mcpRegistryImportTool.Tool, mcpRegistryImportTool.Handler)

		// Add mcp-config-set tool
		mcpConfigSetTool := g.createMcpConfigSetTool(clientConfig)
		g.mcpServer.AddTool(mcpConfigSetTool.Tool, mcpConfigSetTool.Handler)

		// Add codemode
		codeModeTool := g.createCodeModeTool(clientConfig)
		g.mcpServer.AddTool(codeModeTool.Tool, codeModeTool.Handler)

		prompts.AddDiscoverPrompt(g.mcpServer)

		log.Log("  > mcp-discover: prompt for learning about dynamic server management")
		log.Log("  > mcp-find: tool for finding MCP servers in the catalog")
		log.Log("  > mcp-add: tool for adding MCP servers to the registry")
		log.Log("  > mcp-remove: tool for removing MCP servers from the registry")
		log.Log("  > mcp-registry-import: tool for importing servers from MCP registry URLs")
		log.Log("  > mcp-config-set: tool for setting configuration values for MCP servers")
		log.Log("  > code-mode: write code that calls other MCPs directly")
	}

	for _, prompt := range capabilities.Prompts {
		g.mcpServer.AddPrompt(prompt.Prompt, prompt.Handler)

		// Track by server
		if g.serverCapabilities[prompt.ServerName] == nil {
			g.serverCapabilities[prompt.ServerName] = &ServerCapabilities{}
		}
		g.serverCapabilities[prompt.ServerName].PromptNames = append(
			g.serverCapabilities[prompt.ServerName].PromptNames,
			prompt.Prompt.Name,
		)
	}

	for _, resource := range capabilities.Resources {
		g.mcpServer.AddResource(resource.Resource, resource.Handler)

		// Track by server
		if g.serverCapabilities[resource.ServerName] == nil {
			g.serverCapabilities[resource.ServerName] = &ServerCapabilities{}
		}
		g.serverCapabilities[resource.ServerName].ResourceURIs = append(
			g.serverCapabilities[resource.ServerName].ResourceURIs,
			resource.Resource.URI,
		)
	}

	// Resource templates are handled as regular resources in the new SDK
	for _, template := range capabilities.ResourceTemplates {
		// Convert ResourceTemplate to Resource
		resource := &mcp.ResourceTemplate{
			URITemplate: template.ResourceTemplate.URITemplate,
			Name:        template.ResourceTemplate.Name,
			Description: template.ResourceTemplate.Description,
			MIMEType:    template.ResourceTemplate.MIMEType,
		}
		g.mcpServer.AddResourceTemplate(resource, template.Handler)

		// Track by server
		if g.serverCapabilities[template.ServerName] == nil {
			g.serverCapabilities[template.ServerName] = &ServerCapabilities{}
		}
		g.serverCapabilities[template.ServerName].ResourceTemplateURIs = append(
			g.serverCapabilities[template.ServerName].ResourceTemplateURIs,
			resource.URITemplate,
		)
	}

	g.health.SetHealthy()

	return nil
}

// stringSliceToSet converts a slice to a map for efficient lookup
func stringSliceToSet(slice []string) map[string]bool {
	set := make(map[string]bool, len(slice))
	for _, s := range slice {
		set[s] = true
	}
	return set
}

// diffStringSlices returns items that are in 'newer' but not in 'older' (additions),
// and items that are in 'older' but not in 'newer' (removals)
func diffStringSlices(older, newer []string) (additions, removals []string) {
	oldSet := stringSliceToSet(older)
	newSet := stringSliceToSet(newer)

	for s := range newSet {
		if !oldSet[s] {
			additions = append(additions, s)
		}
	}

	for s := range oldSet {
		if !newSet[s] {
			removals = append(removals, s)
		}
	}

	return additions, removals
}

func (g *Gateway) reloadServerConfiguration(ctx context.Context, serverName string, clientConfig *clientConfig) error {
	// Find the server configuration in current config
	serverConfig, _, found := g.configuration.Find(serverName)
	if !found || serverConfig == nil {
		return fmt.Errorf("server %s not found in configuration", serverName)
	}

	// Get current newServerCaps from the server (this reflects the server's current state after it notified us of changes)
	newServerCaps, err := g.listCapabilities(ctx, []string{serverName}, clientConfig)
	if err != nil {
		return fmt.Errorf("failed to list capabilities for %s: %w", serverName, err)
	}

	// Lock for reading/writing capability tracking
	g.capabilitiesMu.Lock()
	defer g.capabilitiesMu.Unlock()

	// Get old capabilities for this server
	oldCaps := g.serverCapabilities[serverName]
	if oldCaps == nil {
		oldCaps = &ServerCapabilities{}
	}

	// Build new capability name/URI lists
	newCaps := &ServerCapabilities{
		ToolNames:            make([]string, 0, len(newServerCaps.Tools)),
		PromptNames:          make([]string, 0, len(newServerCaps.Prompts)),
		ResourceURIs:         make([]string, 0, len(newServerCaps.Resources)),
		ResourceTemplateURIs: make([]string, 0, len(newServerCaps.ResourceTemplates)),
	}

	for _, tool := range newServerCaps.Tools {
		newCaps.ToolNames = append(newCaps.ToolNames, tool.Tool.Name)
	}
	for _, prompt := range newServerCaps.Prompts {
		newCaps.PromptNames = append(newCaps.PromptNames, prompt.Prompt.Name)
	}
	for _, resource := range newServerCaps.Resources {
		newCaps.ResourceURIs = append(newCaps.ResourceURIs, resource.Resource.URI)
	}
	for _, template := range newServerCaps.ResourceTemplates {
		newCaps.ResourceTemplateURIs = append(newCaps.ResourceTemplateURIs, template.ResourceTemplate.URITemplate)
	}

	// Determine what changed
	addedTools, removedTools := diffStringSlices(oldCaps.ToolNames, newCaps.ToolNames)
	addedPrompts, removedPrompts := diffStringSlices(oldCaps.PromptNames, newCaps.PromptNames)
	addedResources, removedResources := diffStringSlices(oldCaps.ResourceURIs, newCaps.ResourceURIs)
	addedTemplates, removedTemplates := diffStringSlices(oldCaps.ResourceTemplateURIs, newCaps.ResourceTemplateURIs)

	// Remove old capabilities that are no longer present
	if len(removedTools) > 0 {
		g.mcpServer.RemoveTools(removedTools...)
		log.Log("  - Removed", len(removedTools), "tools for", serverName)
	}

	if len(removedPrompts) > 0 {
		g.mcpServer.RemovePrompts(removedPrompts...)
		log.Log("  - Removed", len(removedPrompts), "prompts for", serverName)
	}

	if len(removedResources) > 0 {
		g.mcpServer.RemoveResources(removedResources...)
		log.Log("  - Removed", len(removedResources), "resources for", serverName)
	}

	if len(removedTemplates) > 0 {
		g.mcpServer.RemoveResourceTemplates(removedTemplates...)
		log.Log("  - Removed", len(removedTemplates), "resource templates for", serverName)
	}

	// Add/update all capabilities from this server
	for _, tool := range addedTools {
		if registration, err := newServerCaps.getToolByName(tool); err == nil {
			g.mcpServer.AddTool(registration.Tool, registration.Handler)
		}
	}
	if len(addedTools) > 0 {
		log.Log("  - Added/updated", len(addedTools), "tools for", serverName)
	}

	for _, prompt := range addedPrompts {
		if registration, err := newServerCaps.getPromptByName(prompt); err == nil {
			g.mcpServer.AddPrompt(registration.Prompt, registration.Handler)
		}
	}
	if len(addedPrompts) > 0 {
		log.Log("  - Added/updated", len(addedPrompts), "prompts for", serverName)
	}

	for _, resource := range addedResources {
		if registration, err := newServerCaps.getResourceByURI(resource); err == nil {
			g.mcpServer.AddResource(registration.Resource, registration.Handler)
		}
	}
	if len(addedResources) > 0 {
		log.Log("  - Added/updated", len(addedResources), "resources for", serverName)
	}

	for _, template := range addedTemplates {
		if registration, err := newServerCaps.getResourceTemplateByURITemplate(template); err == nil {
			g.mcpServer.AddResourceTemplate(&registration.ResourceTemplate, registration.Handler)
		}
	}
	if len(addedTemplates) > 0 {
		log.Log("  - Added/updated", len(addedTemplates), "resource templates for", serverName)
	}

	// Update tracking with new capabilities
	g.serverCapabilities[serverName] = newCaps

	return nil
}

// enableServersInACA enables MCP servers as containers in Azure Container Apps
func (g *Gateway) enableServersInACA(ctx context.Context, serverNames []string, configuration Configuration) error {
	log.Log("- Enabling servers in ACA mode...")

	for _, serverName := range serverNames {
		serverConfig, _, found := configuration.Find(serverName)
		if !found || serverConfig == nil {
			log.Logf("  > Server '%s' not found in configuration, skipping", serverName)
			continue
		}

		// Determine transport type
		transport := "stdio" // default
		if serverConfig.Spec.Remote.Transport != "" {
			transport = serverConfig.Spec.Remote.Transport
		} else if serverConfig.Spec.SSEEndpoint != "" {
			transport = "sse"
		} else if serverConfig.Spec.Remote.URL != "" {
			transport = "sse"
		}

		// Check if this is stdio and if we have a wrapper
		image := serverConfig.Spec.Image
		if transport == "stdio" {
			if wrapperImage, hasWrapper := aca.GetSSEWrapper(image); hasWrapper {
				log.Logf("  > Server '%s' is stdio, using SSE wrapper: %s", serverName, wrapperImage)
				image = wrapperImage
				transport = "sse"
				
				// Update the catalog entry so listCapabilities() sees the correct transport
				serverConfig.Spec.Image = wrapperImage
				serverConfig.Spec.Remote.Transport = "sse"
				serverConfig.Spec.Remote.URL = "http://localhost:3000/sse"
			} else {
				log.Logf("")
				log.Logf("ERROR: Server '%s' uses stdio transport which is not supported in ACA mode.", serverName)
				log.Logf("       No SSE wrapper is available for image: %s", image)
				log.Logf("")
				log.Logf("Stdio servers cannot run in Azure Container Apps because stdin/stdout")
				log.Logf("pipes do not cross container boundaries.")
				log.Logf("")
				log.Logf("Please either:")
				log.Logf("  1. Use an SSE or streaming transport server instead")
				log.Logf("  2. Build an SSE wrapper for this server (see sse-wrapped-stdio/README.md)")
				log.Logf("  3. Run in Docker mode (unset MCP_RUNTIME environment variable)")
				log.Logf("")
				return fmt.Errorf("stdio server '%s' not supported in ACA mode", serverName)
			}
		}

		// Build environment variables
		env := make(map[string]string)
		for _, e := range serverConfig.Spec.Env {
			env[e.Name] = e.Value
		}

		// Build secrets (convert to env vars for ACA)
		for _, secret := range serverConfig.Spec.Secrets {
			if val, ok := serverConfig.Secrets[secret.Name]; ok {
				env[secret.Env] = val
			}
		}

		// Determine port (SSE wrappers use 3000, others may specify)
		port := 3000
		if transport == "stdio" {
			port = 3000 // wrapper port
		}

		// Create runtime.ServerConfig
		runtimeConfig := runtime.ServerConfig{
			Name:      serverName,
			Image:     image,
			Transport: transport,
			Port:      port,
			Env:       env,
			Secrets:   serverConfig.Secrets,
		}

		// Enable the server via runtime provider
		if err := g.runtimeProvider.EnableServer(ctx, runtimeConfig); err != nil {
			return fmt.Errorf("failed to enable server '%s': %w", serverName, err)
		}
	}

	log.Log("  > All servers enabled in ACA")
	return nil
}

func (g *Gateway) removeServerConfiguration(_ context.Context, serverName string) error {
	// Find the server configuration in current config
	serverConfig, _, found := g.configuration.Find(serverName)
	if !found || serverConfig == nil {
		return fmt.Errorf("server %s not found in configuration", serverName)
	}

	// Lock for reading/writing capability tracking
	g.capabilitiesMu.Lock()
	defer g.capabilitiesMu.Unlock()

	// Get old capabilities for this server
	oldCaps := g.serverCapabilities[serverName]
	if oldCaps == nil {
		oldCaps = &ServerCapabilities{}
	}

	// Remove old capabilities that are no longer present
	if len(oldCaps.ToolNames) > 0 {
		g.mcpServer.RemoveTools(oldCaps.ToolNames...)
		log.Log("  - Removed", len(oldCaps.ToolNames), "tools for", serverName)
	}

	if len(oldCaps.PromptNames) > 0 {
		g.mcpServer.RemovePrompts(oldCaps.PromptNames...)
		log.Log("  - Removed", len(oldCaps.PromptNames), "prompts for", serverName)
	}

	if len(oldCaps.ResourceURIs) > 0 {
		g.mcpServer.RemoveResources(oldCaps.ResourceURIs...)
		log.Log("  - Removed", len(oldCaps.ResourceURIs), "resources for", serverName)
	}

	if len(oldCaps.ResourceTemplateURIs) > 0 {
		g.mcpServer.RemoveResourceTemplates(oldCaps.ResourceTemplateURIs...)
		log.Log("  - Removed", len(oldCaps.ResourceTemplateURIs), "resource templates for", serverName)
	}

	// Update tracking with new capabilities
	delete(g.serverCapabilities, serverName)

	return nil
}
