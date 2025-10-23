# Implementation Plan: Azure Container Apps Runtime Mode

**Branch**: `001-aca-platform-mode` | **Date**: 2025-10-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-aca-platform-mode/spec.md`

## Summary

Add Azure Container Apps (ACA) as an alternative runtime mode for MCP Gateway server container management. The gateway will support two modes controlled by the `MCP_RUNTIME` environment variable:
- **Docker mode** (default): Existing behavior using Docker Engine API
- **ACA mode** (`MCP_RUNTIME=ACA`): Manage MCP servers as containers within the same Azure Container App

**Technical Approach**:
1. Create runtime provider abstraction to decouple container management from specific platforms
2. Implement ACA provider using Azure SDK for Go with System Assigned Identity authentication
3. Extend gateway initialization to detect runtime mode and instantiate appropriate provider
4. Support HTTP-based transports only for ACA mode (streaming/SSE); exclude stdio servers
5. Apply configuration-as-source-of-truth pattern: clean existing containers on startup
6. Mirror gateway resource limits to all MCP server containers for consistent resource allocation

## Technical Context

**Language/Version**: Go 1.24.4  
**Primary Dependencies**: 
- Azure SDK for Go (azidentity, armappcontainers) - **NEW**
- Existing: docker/docker v28.2.2, spf13/cobra v1.9.1, modelcontextprotocol/go-sdk v1.0.0
**Storage**: Configuration files (YAML), no database required  
**Testing**: Go test framework, existing integration test patterns (`make test`, `make integration`)  
**Target Platform**: Azure Container Apps + Docker Engine (dual platform support)  
**Project Type**: Single Go CLI project with plugin architecture  
**Performance Goals**: 
- ACA mode startup within 10 seconds of Docker mode
- Server enable/disable operations < 30 seconds
- Support 5+ concurrent MCP servers
**Constraints**: 
- HTTP-only transports in ACA mode (no stdio)
- Single ACA app scope (no cross-app management)
- System Assigned Identity required for ACA auth
- Configuration-driven container state (no reconciliation loop)
**Scale/Scope**: 
- Incremental feature in existing ~50k LOC codebase
- New package `pkg/aca/` (~2-3k LOC estimated)
- Modify ~5-10 existing files in `pkg/gateway/` and `cmd/docker-mcp/commands/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Respectful Contribution
- ✅ **PASS**: Changes align with existing Go CLI architecture
- ✅ **PASS**: Scope limited to ACA runtime feature, no refactoring of existing Docker code
- ⚠️ **REVIEW**: New dependency Azure SDK for Go (~15 transitive deps) - **Justification**: Official Microsoft SDK, well-maintained, essential for ACA API access, no viable alternative

### II. Appropriate Scope
- ✅ **PASS**: New `pkg/aca/` package isolates ACA-specific code
- ✅ **PASS**: Minimal changes to existing files (gateway initialization, config parsing)
- ✅ **PASS**: No beautification or style changes to existing code

### III. Test Coverage
- ✅ **PASS**: Will include unit tests for `pkg/aca/` package
- ✅ **PASS**: Integration tests for ACA mode lifecycle
- ✅ **PASS**: End-to-end deployment test (gateway + duckduckgo)
- ✅ **PASS**: Existing Docker mode tests remain unchanged

### IV. Minimal Dependencies
- ⚠️ **REVIEW**: Adding `github.com/Azure/azure-sdk-for-go/sdk/azidentity` and `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers`
  - **Rationale**: Official Azure SDKs, actively maintained by Microsoft
  - **License**: MIT (compatible with project)
  - **Alternatives Considered**: 
    - Direct REST API calls: Rejected (complex auth, pagination, error handling)
    - Third-party Azure libraries: Rejected (prefer official SDK)
  - **Transitive Impact**: ~15 additional dependencies (azcore, azcore/policy, etc.)

### V. Documentation and Comments
- ✅ **PASS**: Will update README with `MCP_RUNTIME` environment variable documentation
- ✅ **PASS**: Go doc comments for all exported functions in `pkg/aca/`
- ✅ **PASS**: Update CLAUDE.md with ACA architecture overview

### Extension-Specific Constraints
- ✅ **PASS**: Using Azure SDK for Go (official SDKs)
- ⚠️ **NOTE**: Environment variable is `MCP_RUNTIME=ACA` (not `ACA_MODE` as in constitution example)
- ✅ **PASS**: System Assigned Identity authentication
- ✅ **PASS**: Maintains Docker Engine backward compatibility
- ✅ **PASS**: ACA code isolated in `pkg/aca/` package

**Overall Status**: ✅ **PASS** with justified dependency addition

**Constitution Re-check (Post Phase 1)**:

After completing design phase:
- ✅ **Confirmed**: RuntimeProvider interface provides clean abstraction
- ✅ **Confirmed**: All ACA code isolated in `pkg/aca/` package
- ✅ **Confirmed**: Minimal modifications to existing files (4-5 files touched)
- ✅ **Confirmed**: Azure SDK dependencies justified and documented
- ✅ **Confirmed**: Test coverage plan includes unit and integration tests
- ✅ **Confirmed**: Documentation plan includes README, CLAUDE.md, quickstart

**Final Verdict**: ✅ **READY FOR IMPLEMENTATION**

## Project Structure

### Documentation (this feature)

```text
specs/001-aca-platform-mode/
├── plan.md              # This file
├── research.md          # Phase 0: Azure SDK patterns, ACA API research
├── data-model.md        # Phase 1: Runtime provider interface, ACA entities
├── quickstart.md        # Phase 1: Quick deployment guide for ACA mode
├── contracts/           # Phase 1: ACA API interactions (informal, not REST)
└── tasks.md             # Phase 2: Task breakdown (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/docker-mcp/
├── commands/
│   ├── gateway.go                    # MODIFY: Add MCP_RUNTIME env var parsing
│   └── ...                           # (existing files unchanged)

pkg/
├── aca/                              # NEW PACKAGE
│   ├── provider.go                   # ACA runtime provider implementation
│   ├── auth.go                       # System Assigned Identity authentication
│   ├── metadata.go                   # ACA metadata service queries
│   ├── containers.go                 # Container CRUD operations via ACA API
│   ├── resources.go                  # Resource limit discovery and application
│   ├── provider_test.go              # Unit tests
│   └── integration_test.go           # Integration tests (requires ACA environment)
├── gateway/
│   ├── run.go                        # MODIFY: Runtime provider selection logic
│   ├── config.go                     # MODIFY: Parse MCP_RUNTIME env var
│   ├── provider.go                   # NEW: Runtime provider interface
│   ├── docker_provider.go            # NEW: Extract existing Docker logic
│   └── ...                           # (existing files)
└── ...                               # (other existing packages)

test/
├── simple-test.yaml                  # EXISTING: Used for ACA deployment testing
└── ...

examples/
└── aca-deployment/                   # NEW: ACA deployment examples
    ├── README.md                     # Deployment instructions
    ├── deploy-gateway.sh             # Script to deploy gateway to ACA
    └── deploy-test-server.sh         # Script to deploy test MCP server
```

**Structure Decision**: 

This follows Option 1 (Single project) from the template. The implementation creates a new isolated `pkg/aca/` package for all Azure-specific code, following the constitution's requirement for isolation. Existing Docker logic remains in place with minimal modifications to support the provider abstraction pattern.

The structure respects the existing project organization where:
- `cmd/docker-mcp/commands/` contains CLI command definitions
- `pkg/` contains core library packages
- `test/` contains test fixtures
- `examples/` contains usage examples

Key design principle: **New code in new packages, minimal changes to existing files**.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Adding Azure SDK dependencies (~15 transitive deps) | Azure Container Apps API requires authentication and complex REST interactions | Direct REST calls would require implementing: OAuth token management, error handling, pagination, retry logic, API versioning - recreating SDK functionality poorly |

---

## Plan Completion Summary

**Branch**: `001-aca-platform-mode`  
**Spec File**: `specs/001-aca-platform-mode/spec.md`  
**Plan File**: `specs/001-aca-platform-mode/plan.md`

### Generated Artifacts

#### Phase 0: Research
- ✅ `research.md` - Azure SDK patterns, authentication strategies, API interactions, resource limits, error handling

#### Phase 1: Design
- ✅ `data-model.md` - RuntimeProvider interface, ACA entity definitions, configuration structures
- ✅ `contracts/azure-api-interactions.md` - Azure ARM API documentation, rate limits, error codes
- ✅ `quickstart.md` - ACA deployment guide with 7-step walkthrough

#### Supporting Files
- ✅ `.github/copilot-instructions.md` - Agent context updated with Go 1.24.4, project structure, commands

### Quality Gates

**Constitution Check**: ✅ **PASS**  
- All 5 principles validated
- Azure SDK dependency addition justified (no viable alternative)
- Post-Phase 1 re-check confirms clean abstraction and minimal scope

**Readiness Assessment**: ✅ **READY FOR TASK BREAKDOWN**  
- Technical foundation documented
- Design patterns validated
- API contracts defined
- Deployment path proven via quickstart

### Next Steps

1. **Run `/speckit.tasks` command** to generate `tasks.md` with implementation task breakdown
2. **Create feature branch** from main/master branch named `001-aca-platform-mode`
3. **Begin implementation** following the task sequence in tasks.md
4. **Reference artifacts** during coding:
   - `research.md` for Azure SDK usage patterns
   - `data-model.md` for interface definitions
   - `contracts/azure-api-interactions.md` for API details
   - `quickstart.md` for deployment testing

### Estimated Scope

- **New Package**: `pkg/aca/` (~2-3k LOC)
- **Modified Files**: 4-5 files in `pkg/gateway/` and `cmd/docker-mcp/commands/`
- **Test Files**: Unit tests + integration tests (~1k LOC)
- **Documentation**: README updates, CLAUDE.md architecture section, deployment examples

**Total Estimated Lines**: ~3-5k LOC (new + modifications + tests)  
**Complexity**: Medium (new platform integration, established patterns)
