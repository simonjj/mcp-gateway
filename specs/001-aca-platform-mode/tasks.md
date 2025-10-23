# Tasks: Azure Container Apps Runtime Mode

**Input**: Design documents from `/specs/001-aca-platform-mode/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/azure-api-interactions.md, quickstart.md

**Tests**: Unit and integration tests are included per constitution requirement (Principle III)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md, this is a single Go CLI project with the following structure:
- `pkg/` - Core library packages
- `cmd/docker-mcp/` - CLI command definitions
- `test/` - Test fixtures
- `examples/` - Usage examples

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, dependencies, and basic structure

- [x] T001 Add Azure SDK dependencies to go.mod: azidentity v1.7.0 and armappcontainers/v3 v3.1.0
- [x] T002 Run `go mod tidy` to resolve transitive dependencies
- [x] T003 [P] Create pkg/aca/ directory structure for ACA provider implementation
- [x] T004 [P] Create examples/aca-deployment/ directory for deployment examples

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core abstractions and interfaces that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Define RuntimeProvider interface in pkg/gateway/provider.go with methods: Initialize, Cleanup, EnableServer, DisableServer, ListServers, GetServerStatus
- [x] T006 Define ServerConfig struct in pkg/gateway/provider.go with fields: Name, Image, Transport, Port, Env, Secrets
- [x] T007 Define ServerStatus struct in pkg/gateway/provider.go with fields: Name, State, Image, Started, ExitCode, Message
- [x] T008 Define ContainerState enum in pkg/gateway/provider.go with values: running, stopped, crashed, creating, unknown
- [x] T009 Extract existing Docker logic into DockerProvider implementation in pkg/gateway/docker_provider.go implementing RuntimeProvider interface
- [x] T010 Add MCP_RUNTIME environment variable parsing in cmd/docker-mcp/commands/gateway.go
- [x] T011 Add runtime provider factory function NewRuntimeProvider in pkg/gateway/provider.go that returns DockerProvider or ACAProvider based on mode
- [x] T012 Modify gateway initialization in pkg/gateway/run.go to use RuntimeProvider abstraction instead of direct Docker calls

**Checkpoint**: Foundation ready - RuntimeProvider abstraction established, user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Deploy Gateway on Azure Container Apps (Priority: P1) 🎯 MVP

**Goal**: Enable gateway deployment to ACA with automatic detection of ACA mode via MCP_RUNTIME environment variable and authentication using System Assigned Identity

**Independent Test**: Deploy gateway to ACA with MCP_RUNTIME=ACA set. Gateway starts successfully, authenticates using System Assigned Identity, discovers its own ACA app metadata, and then makes sure that its configuration and the running MCP servers are in-sync. 

### Unit Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T013 [P] [US1] Create pkg/aca/provider_test.go with test for Initialize method (auth, metadata discovery)
- [x] T014 [P] [US1] Create pkg/aca/metadata_test.go with tests for IMDS queries and metadata parsing
- [ ] T015 [P] [US1] Create pkg/aca/auth_test.go with tests for DefaultAzureCredential initialization
- [x] T016 [P] [US1] Create pkg/aca/resources_test.go with tests for resource limit discovery and parsing

### Implementation for User Story 1

- [x] T017 [P] [US1] Define ACAMetadata struct in pkg/aca/metadata.go with fields: SubscriptionID, ResourceGroup, AppName, Location
- [x] T018 [P] [US1] Define ACAResources struct in pkg/aca/resources.go with fields: CPU, Memory
- [x] T019 [P] [US1] Define ProviderError type in pkg/aca/provider.go with error type enum (authentication, permission, quota, not_found, configuration, transient)
- [x] T020 [US1] Implement queryIMDS function in pkg/aca/metadata.go to query Azure Instance Metadata Service at http://169.254.169.254/metadata/instance
- [x] T021 [US1] Implement parseMetadata function in pkg/aca/metadata.go to extract subscription ID, resource group, app name from IMDS response
- [x] T022 [US1] Implement NewACAProvider constructor in pkg/aca/provider.go that returns ACAProvider struct with credential and client fields
- [x] T023 [US1] Implement Initialize method in pkg/aca/provider.go: create DefaultAzureCredential, discover metadata via IMDS, create armappcontainers client
- [x] T024 [US1] Implement discoverResourceLimits function in pkg/aca/resources.go to query gateway container's CPU/memory from ACA app spec
- [x] T025 [US1] Implement Cleanup method in pkg/aca/provider.go to remove all existing MCP server containers (filter by prefix "mcp-server-")
- [x] T026 [US1] Add error handling in pkg/aca/provider.go for authentication failures (401, 403) with clear permission messages
- [x] T027 [US1] Add error handling in pkg/aca/provider.go for IMDS failures with retry logic (3 retries, exponential backoff)
- [x] T028 [US1] Add logging statements in pkg/aca/provider.go for runtime mode detection, metadata discovery, and cleanup operations

### Integration Tests for User Story 1

- [ ] T029 [US1] Create pkg/aca/integration_test.go with test for full Initialize + Cleanup flow (requires real ACA environment)
- [ ] T030 [US1] Add integration test for permission error handling (test with identity lacking permissions)

**Checkpoint**: At this point, User Story 1 should be fully functional - gateway can deploy to ACA, authenticate, discover metadata, and clean up existing containers

---

## Phase 4: User Story 2 - Enable MCP Servers in ACA Mode (Priority: P2)

**Goal**: Enable gateway to create MCP server containers as new container definitions within the same ACA app, applying discovered resource limits to all servers

**Independent Test**: With gateway running in ACA mode, enable a server via configuration (e.g., --servers=duckduckgo). Gateway adds new container to ACA app, ACA platform starts the container, and server becomes available to AI clients.

### Unit Tests for User Story 2

- [ ] T031 [P] [US2] Create pkg/aca/containers_test.go with tests for buildContainerSpec function
- [ ] T032 [P] [US2] Add test in pkg/aca/provider_test.go for EnableServer method with valid ServerConfig
- [ ] T033 [P] [US2] Add test in pkg/aca/provider_test.go for EnableServer with invalid transport (stdio should fail)
- [ ] T034 [P] [US2] Add test in pkg/aca/provider_test.go for GetServerStatus method

### Implementation for User Story 2

- [ ] T035 [P] [US2] Define ACAContainerSpec struct in pkg/aca/containers.go with fields: Name, Image, Resources, Env, Probes
- [ ] T036 [P] [US2] Define EnvVar struct in pkg/aca/containers.go with fields: Name, Value, SecretRef
- [ ] T037 [US2] Implement buildContainerSpec function in pkg/aca/containers.go to convert ServerConfig to armappcontainers.Container
- [ ] T038 [US2] Implement EnableServer method in pkg/aca/provider.go: validate transport (reject stdio), get current app, add container to template, update app via BeginUpdate
- [ ] T039 [US2] Add container naming convention in pkg/aca/containers.go: prefix "mcp-server-" + server name
- [ ] T040 [US2] Apply discovered resource limits in pkg/aca/containers.go when building container spec (use ACAResources from Initialize)
- [ ] T041 [US2] Implement GetServerStatus method in pkg/aca/provider.go to query container state from ACA app template and return ServerStatus
- [ ] T042 [US2] Implement ListServers method in pkg/aca/provider.go to return status of all MCP server containers (filter by "mcp-server-" prefix)
- [ ] T043 [US2] Add validation in pkg/aca/provider.go for ServerConfig: Name required, Image required, Transport must be "streaming" or "sse"
- [ ] T044 [US2] Add error handling for quota errors (409, 429) in pkg/aca/provider.go with quota details in error message
- [ ] T045 [US2] Add error handling for image pull failures in pkg/aca/provider.go with clear registry guidance
- [ ] T046 [US2] Add logging for container creation, resource limit application, and status queries in pkg/aca/provider.go

### Integration Tests for User Story 2

- [ ] T047 [US2] Add integration test in pkg/aca/integration_test.go for EnableServer + GetServerStatus (verify container created and running)
- [ ] T048 [US2] Add integration test in pkg/aca/integration_test.go for multiple server enablement (5 servers concurrently)
- [ ] T049 [US2] Add integration test in pkg/aca/integration_test.go for stdio transport rejection

**Checkpoint**: At this point, User Stories 1 AND 2 should both work - gateway can deploy to ACA and enable MCP servers as containers

---

## Phase 5: User Story 3 - Disable and Remove MCP Servers in ACA Mode (Priority: P3)

**Goal**: Enable gateway to remove MCP server containers by updating ACA app configuration, allowing ACA platform to clean up resources

**Independent Test**: With servers running in ACA mode, update configuration to remove a server. Gateway removes container from ACA app configuration, and ACA platform stops and removes the container.

### Unit Tests for User Story 3

- [ ] T050 [P] [US3] Add test in pkg/aca/provider_test.go for DisableServer method
- [ ] T051 [P] [US3] Add test in pkg/aca/provider_test.go for DisableServer with non-existent server

### Implementation for User Story 3

- [ ] T052 [US3] Implement DisableServer method in pkg/aca/provider.go: get current app, filter out container by name, update app via BeginUpdate
- [ ] T053 [US3] Add idempotency check in pkg/aca/provider.go: DisableServer succeeds if container already removed
- [ ] T054 [US3] Add error handling for long-running operation failures in pkg/aca/provider.go (poll until done with timeout)
- [ ] T055 [US3] Add logging for container removal operations in pkg/aca/provider.go

### Integration Tests for User Story 3

- [ ] T056 [US3] Add integration test in pkg/aca/integration_test.go for full lifecycle: EnableServer → GetServerStatus → DisableServer → verify removed
- [ ] T057 [US3] Add integration test in pkg/aca/integration_test.go for removing all servers (cleanup validation)

**Checkpoint**: All core user stories complete - gateway can deploy, enable, and disable MCP servers in ACA mode

---

## Phase 6: User Story 4 - Runtime Mode Backward Compatibility (Priority: P1)

**Goal**: Ensure existing Docker Engine deployments continue to work without modification when MCP_RUNTIME is not set or set to non-ACA value

**Independent Test**: Deploy gateway without setting MCP_RUNTIME. All existing Docker Engine functionality works identically to current version.

### Unit Tests for User Story 4

- [x] T058 [P] [US4] Add test in pkg/gateway/provider_test.go for NewRuntimeProvider with no MCP_RUNTIME env var (should return DockerProvider)
- [x] T059 [P] [US4] Add test in pkg/gateway/provider_test.go for NewRuntimeProvider with MCP_RUNTIME=docker (should return DockerProvider)
- [x] T060 [P] [US4] Add test in pkg/gateway/provider_test.go for NewRuntimeProvider with MCP_RUNTIME=ACA (should return ACAProvider)

### Implementation for User Story 4

- [x] T061 [US4] Verify DockerProvider implementation in pkg/gateway/docker_provider.go implements all RuntimeProvider interface methods
- [x] T062 [US4] Add default case in NewRuntimeProvider function in pkg/gateway/provider.go to return DockerProvider when MCP_RUNTIME is unset or invalid
- [x] T063 [US4] Add logging in pkg/gateway/provider.go for runtime mode selection (Docker vs ACA)

### Integration Tests for User Story 4

- [ ] T064 [US4] Run existing integration tests with MCP_RUNTIME unset to verify Docker mode backward compatibility (make integration)
- [ ] T065 [US4] Verify no regression in existing Docker functionality (compare before/after behavior)

**Checkpoint**: Backward compatibility confirmed - existing Docker deployments unaffected

---

## Phase 7: User Story 5 - End-to-End ACA Deployment Testing (Priority: P2)

**Goal**: Validate ACA runtime mode by deploying gateway and simple MCP server to Azure Container Apps using decomposed YAML and CLI commands

**Independent Test**: Deploy gateway to ACA with MCP_RUNTIME=ACA and --servers=duckduckgo. Verify gateway and server containers both start, and gateway can route MCP protocol requests to server.

### Deployment Scripts and Documentation

- [ ] T066 [P] [US5] Create examples/aca-deployment/README.md with step-by-step deployment instructions based on quickstart.md
- [ ] T067 [P] [US5] Create examples/aca-deployment/deploy-gateway.sh script to deploy gateway to ACA using az containerapp create
- [ ] T068 [P] [US5] Create examples/aca-deployment/deploy-permissions.sh script to grant System Assigned Identity Contributor role
- [ ] T069 [P] [US5] Create examples/aca-deployment/cleanup.sh script to delete all deployed resources

### End-to-End Validation

- [ ] T070 [US5] Manually execute deploy-gateway.sh and verify gateway starts in ACA mode (check logs)
- [ ] T071 [US5] Manually execute deploy-permissions.sh and verify role assignment succeeds
- [ ] T072 [US5] Configure gateway with SERVERS=duckduckgo environment variable and verify server container created
- [ ] T073 [US5] Connect MCP client to gateway and verify duckduckgo server tools are discoverable
- [ ] T074 [US5] Invoke a test tool from duckduckgo server and verify successful response
- [ ] T075 [US5] Check container logs for both gateway and server using az containerapp logs show
- [ ] T076 [US5] Verify deployment completes within 5 minutes (success criteria SC-012)
- [ ] T077 [US5] Execute cleanup.sh and verify all resources deleted

**Checkpoint**: End-to-end ACA deployment validated with real gateway and server

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, performance validation, and final quality checks

- [ ] T078 [P] Update README.md with MCP_RUNTIME environment variable documentation and ACA deployment section
- [ ] T079 [P] Update CLAUDE.md with ACA architecture overview, RuntimeProvider abstraction, and deployment patterns
- [ ] T080 [P] Add Go doc comments to all exported functions in pkg/aca/ package
- [ ] T081 [P] Add Go doc comments to RuntimeProvider interface and related types in pkg/gateway/provider.go
- [ ] T082 Validate gateway startup time in ACA mode is within 10 seconds of Docker mode (success criteria SC-005)
- [ ] T083 Validate server enable/disable operations complete within 30 seconds (success criteria SC-003)
- [ ] T084 Validate gateway can manage 5 concurrent MCP servers in ACA mode (success criteria SC-002)
- [ ] T085 Run make test to verify all unit tests pass
- [ ] T086 Run make integration to verify all integration tests pass (Docker mode)
- [ ] T087 Run quickstart.md deployment guide end-to-end and verify all steps work
- [ ] T088 Code review: verify constitution compliance (isolated pkg/aca/, minimal file changes, justified dependencies)
- [ ] T089 Security review: verify no secrets in logs, proper error message sanitization
- [ ] T090 Performance profiling: verify ACA API calls use appropriate timeouts and retries

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - US1 (Deploy to ACA) - Can start after Foundational
  - US2 (Enable Servers) - Depends on US1 (needs Initialize/Cleanup from US1)
  - US3 (Disable Servers) - Depends on US2 (needs EnableServer to test DisableServer)
  - US4 (Backward Compat) - Can start after Foundational (parallel with US1-3)
  - US5 (E2E Testing) - Depends on US1, US2, US3 (needs full lifecycle)
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Depends on User Story 1 completion (needs Initialize, Cleanup, metadata, resources)
- **User Story 3 (P3)**: Depends on User Story 2 completion (needs EnableServer to test DisableServer)
- **User Story 4 (P1)**: Depends on Foundational (Phase 2) - No dependencies on other stories (can run parallel with US1)
- **User Story 5 (P2)**: Depends on User Stories 1, 2, 3 completion (needs full lifecycle for E2E testing)

### Within Each User Story

- Unit tests MUST be written and FAIL before implementation
- Struct definitions before functions that use them
- Helper functions before main methods
- Core implementation before error handling and logging
- Story complete before moving to next priority

### Parallel Opportunities

- **Phase 1**: T003 and T004 can run in parallel (different directories)
- **Phase 2**: T005-T008 can run in parallel (different files/structs)
- **US1 Tests**: T013, T014, T015, T016 can run in parallel
- **US1 Implementation**: T017, T018, T019 can run in parallel (different files)
- **US2 Tests**: T031, T032, T033, T034 can run in parallel
- **US2 Implementation**: T035, T036 can run in parallel (different structs in same file)
- **US3 Tests**: T050, T051 can run in parallel
- **US4 Tests**: T058, T059, T060 can run in parallel
- **US5 Scripts**: T066, T067, T068, T069 can run in parallel (different files)
- **Phase 8**: T078, T079, T080, T081 can run in parallel (different files)
- **User Stories**: US1 and US4 can run in parallel after Foundational phase

---

## Parallel Example: User Story 1

```bash
# Launch all unit tests for User Story 1 together:
Task T013: "Create pkg/aca/provider_test.go with test for Initialize method"
Task T014: "Create pkg/aca/metadata_test.go with tests for IMDS queries"
Task T015: "Create pkg/aca/auth_test.go with tests for DefaultAzureCredential"
Task T016: "Create pkg/aca/resources_test.go with tests for resource limit discovery"

# Launch all struct definitions for User Story 1 together:
Task T017: "Define ACAMetadata struct in pkg/aca/metadata.go"
Task T018: "Define ACAResources struct in pkg/aca/resources.go"
Task T019: "Define ProviderError type in pkg/aca/provider.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 4 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Deploy to ACA)
4. Complete Phase 6: User Story 4 (Backward Compatibility)
5. **STOP and VALIDATE**: Test both ACA mode and Docker mode independently
6. Deploy/demo if ready

This gives you:
- ✅ Gateway runs in ACA mode with auth and metadata discovery
- ✅ Existing Docker deployments unaffected
- ❌ Cannot enable/disable servers yet (US2/US3 needed)

### Full Feature Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Gateway deploys to ACA
3. Add User Story 2 → Test independently → Can enable servers
4. Add User Story 3 → Test independently → Can disable servers
5. Complete User Story 4 → Test independently → Docker mode preserved
6. Complete User Story 5 → Test E2E → Full lifecycle validated
7. Complete Polish → Documentation, performance validation, final checks

### Parallel Team Strategy

With multiple developers after Foundational phase completes:

1. **Developer A**: User Story 1 (Deploy to ACA) - T013-T030
2. **Developer B**: User Story 4 (Backward Compatibility) - T058-T065 (parallel with A)
3. **Developer C**: Setup deployment scripts for US5 - T066-T069 (parallel prep)

Then sequentially:
4. **Developer A**: User Story 2 (after US1 complete)
5. **Developer A**: User Story 3 (after US2 complete)
6. **All**: User Story 5 E2E testing together
7. **All**: Polish tasks in parallel

---

## Task Summary

**Total Tasks**: 90

**Tasks per Phase**:
- Phase 1 (Setup): 4 tasks
- Phase 2 (Foundational): 8 tasks
- Phase 3 (US1 - Deploy to ACA): 18 tasks
- Phase 4 (US2 - Enable Servers): 19 tasks
- Phase 5 (US3 - Disable Servers): 6 tasks
- Phase 6 (US4 - Backward Compat): 5 tasks
- Phase 7 (US5 - E2E Testing): 12 tasks
- Phase 8 (Polish): 13 tasks

**Tasks per User Story**:
- US1: 18 tasks (13 implementation + 5 tests)
- US2: 19 tasks (15 implementation + 4 tests)
- US3: 6 tasks (4 implementation + 2 tests)
- US4: 5 tasks (3 implementation + 2 tests)
- US5: 12 tasks (12 validation/deployment)

**Parallel Opportunities**: 28 tasks marked with [P]

**Independent Test Criteria**:
- US1: Gateway deploys to ACA, authenticates, discovers metadata, cleans up containers
- US2: Gateway enables servers as ACA containers with proper resource limits
- US3: Gateway disables servers and removes from ACA app
- US4: Docker mode works without MCP_RUNTIME set
- US5: Full E2E deployment with gateway + duckduckgo server works end-to-end

**Suggested MVP Scope**: 
- Phase 1 (Setup) + Phase 2 (Foundational) + Phase 3 (US1) + Phase 6 (US4) = 35 tasks
- Delivers: Gateway runs in both ACA and Docker modes with auth and metadata discovery
- Enables: Testing both runtime modes, validates backward compatibility

---

## Notes

- [P] tasks = different files, no dependencies, can run in parallel
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Unit tests MUST fail before implementation (TDD approach per constitution)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Constitution compliance: All ACA code isolated in pkg/aca/, minimal changes to existing files
- Performance targets: Startup <10s overhead, enable/disable <30s, 5+ concurrent servers
