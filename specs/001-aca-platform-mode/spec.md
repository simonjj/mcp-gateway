# Feature Specification: Azure Container Apps Runtime Mode

**Feature Branch**: `001-aca-platform-mode`  
**Created**: 2025-10-22  
**Status**: Draft  
**Input**: User description: "The mcp-gateway manages the lifecycle for the mcp servers as simple Docker containers instantiated and deleted via access to the Docker engine directly. For more details see CLAUDE.md. Going forward I want to run it on Azure Container Apps and want the mcp servers to be spun up as other containers on the same ACA app. The method to use (ACA-native or Docker engine) should be controllable via a environment variable called MCP_RUNTIME which should be set to ACA for the new ACA-native mode to be used otherwise always use the current Docker method. To have the permissions, the mcp-gateway will be assigned a System Assigned Identity as part of the deployment (something that happens outside of this project). This identity will have the permissions to modify the ACA app the gateway is using itself in order to add/delete/change the mcp server containers. Lastly, I'd like to do testing by actually deploying the new mcp-gateway as a native app to Azure Container Apps and testing with a very simple mcp server (see ./mcp-gateway/test/simple-test.yaml). Don't use the `az cli compose` functionality for this but instead decompose the yaml into a deployment using az cli and setting command/env variables accordingly."

## Clarifications

### Session 2025-10-22

- Q: When the gateway starts in ACA mode, how should it handle existing MCP server containers that may already be running in the ACA app? → A: The gateway always starts fresh - it removes all existing MCP server containers on startup and creates new ones based on configuration
- Q: How should the gateway determine resource limits (CPU/memory) for MCP server containers in ACA mode? → A: All MCP server containers get identical fixed resource limits, mirroring the gateway's own resource allocation (same CPU and memory as the gateway container)
- Q: How should the gateway communicate with MCP server containers when they run as separate containers in the same ACA app? → A: Use HTTP-based communication (streaming or SSE transport) only; stdio-based servers are not supported in ACA mode for this implementation
- Q: When an MCP server container crashes or fails in ACA mode, what should the restart policy be? → A: Never restart - crashed containers stay stopped until gateway restarts or configuration changes
- Q: After startup validation, should the gateway continue to validate Azure API permissions during operation? → A: Only validate permissions once at startup; runtime Azure API errors are logged and reported but don't trigger re-validation

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Deploy Gateway on Azure Container Apps (Priority: P1)

A developer or operator deploys the MCP Gateway to Azure Container Apps and configures it to manage MCP servers as native ACA containers instead of Docker Engine containers. The gateway automatically detects it's running in ACA mode via the `MCP_RUNTIME` environment variable and uses the System Assigned Identity to manage container lifecycle.

**Why this priority**: This is the foundational capability that enables all ACA deployment scenarios. Without this, the gateway cannot run on Azure Container Apps in the new mode.

**Independent Test**: Deploy the gateway to ACA with `MCP_RUNTIME=ACA` environment variable set. The gateway starts successfully, authenticates using System Assigned Identity, and is ready to manage MCP servers on the same ACA app.

**Acceptance Scenarios**:

1. **Given** the gateway is deployed to Azure Container Apps with `MCP_RUNTIME=ACA` set, **When** the gateway starts, **Then** it initializes in ACA mode and authenticates using System Assigned Identity
2. **Given** the gateway is deployed to Azure Container Apps with `MCP_RUNTIME=ACA` set, **When** startup completes, **Then** the gateway can query the ACA environment to discover its own app and resource group
2a. **Given** the gateway starts in ACA mode and existing MCP server containers are present in the ACA app, **When** initialization occurs, **Then** the gateway removes all existing MCP server containers before creating new ones from configuration
3. **Given** the gateway is deployed with `MCP_RUNTIME` unset or set to any value other than `ACA`, **When** the gateway starts, **Then** it operates in Docker Engine mode using the existing container management logic
4. **Given** the gateway is running in ACA mode, **When** System Assigned Identity lacks required permissions, **Then** the gateway logs clear error messages indicating permission issues and fails gracefully

---

### User Story 2 - Enable MCP Servers in ACA Mode (Priority: P2)

A user enables one or more MCP servers through the gateway configuration or CLI. When running in ACA mode, the gateway creates new container definitions within the same ACA app instead of launching Docker containers.

**Why this priority**: This is the core MCP server lifecycle functionality adapted for ACA. Users need to be able to enable servers just like they do in Docker mode.

**Independent Test**: With the gateway running in ACA mode, enable a server (e.g., via `--servers=duckduckgo` flag). The gateway adds a new container to the ACA app configuration, and the ACA platform starts the container. The server becomes available to AI clients.

**Acceptance Scenarios**:

1. **Given** the gateway is running in ACA mode and a server is specified in configuration, **When** the gateway starts, **Then** the gateway adds a container definition to the ACA app and the server container starts
2. **Given** the gateway is running in ACA mode and multiple servers are enabled, **When** the user queries server status, **Then** all enabled servers show their status from ACA container state
3. **Given** the gateway is running in ACA mode, **When** a server container image is not available in the ACA-accessible registry, **Then** the gateway reports an error with clear guidance on registry requirements
4. **Given** the gateway is running in ACA mode and a server requires secrets, **When** the server is enabled, **Then** secrets are properly configured as ACA environment variables or secret references

---

### User Story 3 - Disable and Remove MCP Servers in ACA Mode (Priority: P3)

A user disables or removes MCP servers by changing the gateway configuration. When running in ACA mode, the gateway removes the corresponding container definitions from the ACA app, allowing the platform to clean up resources.

**Why this priority**: Server lifecycle management requires both creation and deletion. This completes the basic CRUD operations for MCP servers in ACA.

**Independent Test**: With servers running in ACA mode, update configuration to remove a server. The gateway removes the container from the ACA app configuration, and the ACA platform stops and removes the container.

**Acceptance Scenarios**:

1. **Given** the gateway is running in ACA mode and a server is enabled, **When** the server is removed from configuration and gateway restarts, **Then** the gateway removes the container definition from the ACA app and the container is terminated
2. **Given** the gateway is running in ACA mode with multiple servers, **When** all servers are removed from configuration, **Then** all MCP server containers are removed from the ACA app
3. **Given** the gateway is running in ACA mode and a server is removed, **When** the user queries server status, **Then** the server is no longer present in the ACA app

---

### User Story 4 - Runtime Mode Backward Compatibility (Priority: P1)

Existing Docker Engine deployments continue to work without modification. When the `MCP_RUNTIME` environment variable is not set or is set to any value other than `ACA`, the gateway uses the existing Docker Engine-based container management.

**Why this priority**: Preserving backward compatibility ensures existing users are not disrupted and allows gradual migration to ACA.

**Independent Test**: Deploy the gateway without setting `MCP_RUNTIME`. All existing Docker Engine functionality works identically to the current version.

**Acceptance Scenarios**:

1. **Given** the gateway is deployed without the `MCP_RUNTIME` variable set, **When** the gateway starts, **Then** it defaults to Docker Engine mode
2. **Given** the gateway is running in Docker Engine mode, **When** users enable or inspect servers, **Then** all operations work using the existing Docker API
3. **Given** the gateway configuration files from a Docker deployment, **When** the same configuration is used in ACA mode, **Then** the gateway interprets server definitions appropriately for the runtime

---

### User Story 5 - End-to-End ACA Deployment Testing (Priority: P2)

A developer tests the new ACA runtime mode by deploying the gateway and a simple MCP server to Azure Container Apps using decomposed YAML configuration and manual CLI commands, validating that the gateway can successfully manage MCP server containers in the ACA environment.

**Why this priority**: Automated end-to-end testing in the actual ACA environment is critical to validate the feature works in production conditions.

**Independent Test**: Deploy the gateway to ACA with `MCP_RUNTIME=ACA` and `--servers=duckduckgo` configuration. Verify the gateway container and duckduckgo server container both start, and the gateway can route MCP protocol requests to the server.

**Acceptance Scenarios**:

1. **Given** a simple compose file with gateway and server configuration, **When** the configuration is decomposed and deployed to ACA using CLI commands, **Then** both gateway and server containers start successfully
2. **Given** the gateway and server are running in ACA, **When** an MCP client connects to the gateway, **Then** the client can discover and invoke tools from the server
3. **Given** the deployment is successful, **When** monitoring container logs, **Then** gateway logs show successful ACA mode initialization and server container management
4. **Given** the test deployment is running, **When** configuration is updated to add or remove servers, **Then** the gateway responds by modifying the ACA app container definitions accordingly

---

### Edge Cases

- What happens when the System Assigned Identity's permissions are revoked while the gateway is running? (Resolution: Runtime Azure API errors are logged and reported; gateway restart required to re-validate permissions)
- How does the system handle container image pull failures in ACA mode (e.g., private registries, authentication issues)?
- What happens when the ACA app reaches resource quota limits and cannot start additional containers?
- How does the gateway handle concurrent configuration changes for the same server in ACA mode?
- What happens when the gateway is switched from ACA mode to Docker mode (or vice versa) with running servers?
- How does the system handle ACA API rate limiting or transient failures?
- What happens when an MCP server container crashes in ACA mode? (Resolution: Container stays stopped until gateway restarts or configuration changes)
- How does the gateway behave when deployed to ACA but `MCP_RUNTIME` is not set to `ACA`?
- What happens when the gateway cannot discover its own ACA app metadata?
- How does the gateway handle a request to enable a stdio-based MCP server in ACA mode?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support an `MCP_RUNTIME` environment variable that controls the container management backend
- **FR-002**: System MUST use Docker Engine for container management when `MCP_RUNTIME` is unset or set to any value other than `ACA`
- **FR-003**: System MUST use Azure Container Apps for container management when `MCP_RUNTIME` is set to `ACA`
- **FR-004**: System MUST authenticate to Azure using the System Assigned Identity when running in ACA mode
- **FR-005**: System MUST discover its own ACA app name, resource group, and subscription when running in ACA mode
- **FR-005a**: System MUST remove all existing MCP server containers from the ACA app on startup in ACA mode before creating new containers
- **FR-006**: System MUST add container definitions to the ACA app when enabling MCP servers in ACA mode
- **FR-006a**: System MUST discover the gateway's own resource allocation (CPU and memory) and apply the same limits to all MCP server containers in ACA mode
- **FR-007**: System MUST remove container definitions from the ACA app when disabling MCP servers in ACA mode
- **FR-008**: System MUST query ACA container status to report server health and availability
- **FR-009**: System MUST handle Azure API authentication, including token refresh and error scenarios
- **FR-010**: System MUST validate that the System Assigned Identity has required permissions at startup in ACA mode only; no runtime re-validation occurs
- **FR-011**: System MUST provide clear error messages when ACA operations fail due to permissions, quotas, or API errors
- **FR-012**: System MUST maintain configuration compatibility between Docker and ACA modes
- **FR-013**: System MUST support the same server specification formats in both modes (e.g., `--servers` flag, configuration files)
- **FR-014**: System MUST handle container lifecycle events (started, stopped, crashed) appropriately in ACA mode
- **FR-014a**: System MUST NOT automatically restart crashed MCP server containers; containers remain stopped until gateway restart or configuration change
- **FR-015**: System MUST ensure server containers in ACA mode can communicate with the gateway container using HTTP-based transports (streaming or SSE)
- **FR-015a**: System MUST NOT support stdio-based MCP servers in ACA mode
- **FR-016**: System MUST configure environment variables and secrets for MCP servers in ACA mode
- **FR-017**: System MUST log runtime mode selection and initialization steps for troubleshooting
- **FR-018**: System MUST support testing deployments using decomposed YAML configuration deployed via CLI commands
- **FR-019**: System MUST enable deployment of gateway and simple test server from compose-style configuration to ACA

### Assumptions

- The ACA app running the gateway is deployed with a System Assigned Identity configured
- The System Assigned Identity has been granted appropriate permissions to modify the ACA app (e.g., Azure Container Apps Contributor role)
- MCP server container images are accessible from registries that ACA can pull from (e.g., Docker Hub public registry for standard servers)
- The gateway will manage containers only within its own ACA app (no cross-app management)
- Network policies within ACA allow communication between the gateway container and MCP server containers on the same app
- MCP servers in ACA mode use HTTP-based transports (streaming or SSE); stdio transport is not supported
- The Azure metadata service is accessible from within the ACA environment for identity token retrieval
- ACA resource quotas are sufficient to run the desired number of MCP server containers
- All MCP server containers will use the same resource limits as the gateway container itself
- Testing will use manual CLI deployment commands rather than automated compose translation
- The simple test configuration (gateway + duckduckgo server) represents a minimal viable deployment scenario

### Key Entities

- **Runtime Provider**: Abstraction representing the container management backend (Docker Engine or Azure Container Apps)
- **ACA App**: The Azure Container App instance where the gateway is running, which will also host MCP server containers
- **System Assigned Identity**: Azure managed identity assigned to the ACA app, used for authentication to Azure APIs
- **Container Definition**: Specification of a container to run (image, environment variables, resources, etc.)
- **MCP Server Instance**: A running MCP server container managed by the gateway
- **Runtime Mode**: The active container management mode (Docker or ACA) determined by the `MCP_RUNTIME` environment variable
- **Deployment Configuration**: Decomposed YAML configuration translated to CLI commands for ACA deployment

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Gateway successfully deploys to Azure Container Apps and initializes in ACA mode when `MCP_RUNTIME=ACA` is set
- **SC-002**: Gateway can manage at least 5 MCP servers simultaneously in ACA mode without errors
- **SC-003**: Server enable/disable operations in ACA mode complete within 30 seconds under normal conditions
- **SC-004**: Existing Docker Engine deployments continue to function identically with no configuration changes required
- **SC-005**: Gateway startup time in ACA mode is within 10 seconds of Docker Engine mode startup time
- **SC-006**: 100% of existing server specifications (e.g., `--servers` flag values) work in both Docker and ACA modes without modification
- **SC-007**: Gateway correctly handles and reports at least 90% of common Azure API error scenarios with actionable error messages
- **SC-008**: MCP server containers started in ACA mode respond to protocol requests within the same latency profile as Docker Engine mode (within 10% variance)
- **SC-009**: Gateway can recover from transient Azure API failures and retry operations successfully
- **SC-010**: End-to-end test deployment (gateway + duckduckgo server) succeeds using decomposed CLI commands
- **SC-011**: Test deployment allows MCP clients to successfully discover and invoke tools from the test server
- **SC-012**: Deployment process completes within 5 minutes from initial CLI command to fully operational gateway and server

