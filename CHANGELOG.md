# Changelog

All notable changes to AgentiCorp will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

[0;34mℹ️  Generating changelog from v0.0.0 to HEAD...[0m
## [0.0.1] - 2026-01-26

### Added
- auto-file ALL UI errors as beads with toast notifications
- project-specific bead ID prefixes
- add close_bead action for agent bead lifecycle management
- add write_file action for reliable LLM code editing
- implement motivation system for proactive agent workflows
- dashboard, file APIs, and logging filters
- provider tooling for file and git actions
- agenticorp binary to gitignore
- robust JSON extraction for provider responses
- authentication configuration toggle and auto-filing bug tracking system
- comprehensive Kubernetes best practices guide
- shell command execution capability for agents
- /app/data directory creation to Dockerfile for SQLite persistence
- documentation for persona-based routing
- persona-based routing and auto-bead creation from CEO REPL
- vLLM provider support and improve provider debugging
- automated release system
- build and test report
- workflow builder and persona testing (bd-110, bd-111)
- persona templates for common roles (bd-099)
- inline code suggestions (bd-095)
- horizontal scaling guide and benchmarks (bd-093)
- load balancer integration and session affinity (bd-090)
- graceful shutdown and startup (bd-092)
- health check endpoints for monitoring (bd-091)
- Redis backend for distributed caching (bd-084)
- cache invalidation policies (bd-083)
- configurable cache TTL and size limits (bd-082)
- intelligent response caching (bd-081)
- comprehensive work summary for bd-054 epic
- comprehensive analytics and release documentation
- alerting for unusual spending patterns (bd-080)
- comprehensive data export for external analysis (bd-079)
- cost tracking per provider and per user (bd-077)
- beads migration documentation
- Complete request/response logging infrastructure (bd-076)
- Complete advanced provider routing epic (bd-053)
- Complete authentication & authorization epic (bd-052)
- Integrate gitops manager for project repository management
- Add streaming support, gitops infrastructure, and documentation improvements
- comprehensive TemporalManager DSL and complete documentation
- protocol dropdown and encrypted API key storage for providers
- org chart abstraction layer, entity versioning, and post-flight API tests
- kickstart for open beads, fix Docker issues, improve Makefile
- provider heartbeats, REPL, and dockerized workflows
- P0 bead for provider heartbeat self-tests
- P0 beads for dockerized tests and Temporal availability
- debug output for make failure beads
- user guide and GitHub Pages workflow
- sticky projects and YAML linting
- repository rules and ignore obj directory
- model catalog, provider negotiation, and persona defaults
- provider registry, config DB, and global dispatcher
- comprehensive MILESTONES.md document from Project Manager persona
- integration tests and example configuration files
- documentation and tests for worker system
- provider protocol, worker system, and agent worker manager
- product manager persona with future goals documentation
- comprehensive README for public relations manager persona
- public relations manager persona
- QA Engineer persona for first release test planning
- Temporal server infrastructure and event bus implementation
- project state management documentation
- engineering manager persona with design doc review capabilities
- comprehensive tests for project state management
- arbiter persona files
- project state management and arbiter persona
- QA workflow documentation
- QA engineer and project manager personas
- 5 new agent personas: Product Manager, Project Manager, Engineering Manager, Documentation Manager, and DevOps Engineer
- QUICKSTART.md guide for easy onboarding
- Makefile, CONTRIBUTING.md, update .gitignore, remove binary
- HTTP API, web UI, and main application entry point
- core managers (agent, project, beads, decision, filelock, arbiter)
- comprehensive architecture documentation
- persona system, data models, config, and OpenAPI spec
- Go-based containerized architecture for arbiter
- main arbiter application entry point
- main application and web UI dashboard
- core arbiter database and key management system
- arbiter web API and dashboard with service cost tracking
- AI coding agent orchestrator with multi-provider support
- CLI entry point for Arbiter application
- core Arbiter functionality with agent orchestration

### Changed
- bead UI and workflows
- beads after closing P0 issues
- architecture diagram for Analytics v1.1 (bd-011)
- README with analytics features (bd-010)
- provider registration form and enhance error handling
- README with completed features and documentation links

### Fixed
- transform motivations/roles API response to expected format
- disable auth for development (keep code, disable feature)
- UI auth and motivations loading issues
- correct GetProjectWorkDir for Docker environment
- add cache-busting version to JS includes
- handle </think> without opening tag in LLM response parsing
- add missing getAuthHeaders() function for motivations UI
- make analytics tests time-zone agnostic
- strip <think> tags from LLM output before JSON extraction
- CEO REPL to accept 'healthy' provider status
- all failing tests (P0)
- agent creation API to normalize persona names and close duplicate P0 beads
- streaming support by implementing Flusher in statusRecorder middleware
- provider persistence and remove mock provider bootstrap
- Show all providers in streaming test dropdown, not just active ones
- P0: Implement proper write-through cache for agent state
- auto-filing bug reporting system
- Remove example-project to prevent agent assignment regression
- analytics endpoints auth when authentication disabled
- agent persistence and add log viewer UI foundation
- Make health endpoints publicly accessible without authentication
- state persistence: Add volume mount for SQLite database
- Temporal connection for Docker environment
- provider status detection and network connectivity
- UI contrast (WCAG AA) and add project settings gear icon
- UI issues and improve agent management
- bead card contrast and docker compose YAML
- kanban text contrast and add project CRUD bead
- Makefile tee output path
- project is_sticky migrations
- bead YAML parsing warnings
- build ignore rules and Makefile targets
- compilation errors - update codebase to work with new structure
- Remove duplicate main functions and restore clean cmd/arbiter/main.go
- timestamp consistency in project manager
- broken code - remove duplicate implementations and fix syntax errors
- build issues and add persona tests
- Go dependencies and resolve merge conflicts
- JavaScript null handling and model JSON serialization
- Go version to 1.21 for consistency across project
- Go version requirement in README to match go.mod

### Added
- Configuration-based authentication toggle via `security.enable_auth` flag in `config.yaml`
- Authentication can now be completely disabled for development environments
- `CONFIG_PATH` environment variable support for specifying config file location
- Comprehensive authentication configuration documentation (`docs/AUTH_CONFIGURATION.md`)

### Changed
- `main.go` now loads configuration from `config.yaml` instead of using hardcoded defaults
- Docker container now uses volume-mounted config file for live configuration updates
- Default authentication setting changed to `false` for development (set to `true` in production)

### Fixed
- Config file changes now properly apply when container restarts
- Authentication middleware correctly respects `enable_auth` configuration flag

## [Unreleased]

[0;34mℹ️  Generating changelog from v0.0.0 to HEAD...[0m
## [0.0.1] - 2026-01-26

### Added
- auto-file ALL UI errors as beads with toast notifications
- project-specific bead ID prefixes
- add close_bead action for agent bead lifecycle management
- add write_file action for reliable LLM code editing
- implement motivation system for proactive agent workflows
- dashboard, file APIs, and logging filters
- provider tooling for file and git actions
- agenticorp binary to gitignore
- robust JSON extraction for provider responses
- authentication configuration toggle and auto-filing bug tracking system
- comprehensive Kubernetes best practices guide
- shell command execution capability for agents
- /app/data directory creation to Dockerfile for SQLite persistence
- documentation for persona-based routing
- persona-based routing and auto-bead creation from CEO REPL
- vLLM provider support and improve provider debugging
- automated release system
- build and test report
- workflow builder and persona testing (bd-110, bd-111)
- persona templates for common roles (bd-099)
- inline code suggestions (bd-095)
- horizontal scaling guide and benchmarks (bd-093)
- load balancer integration and session affinity (bd-090)
- graceful shutdown and startup (bd-092)
- health check endpoints for monitoring (bd-091)
- Redis backend for distributed caching (bd-084)
- cache invalidation policies (bd-083)
- configurable cache TTL and size limits (bd-082)
- intelligent response caching (bd-081)
- comprehensive work summary for bd-054 epic
- comprehensive analytics and release documentation
- alerting for unusual spending patterns (bd-080)
- comprehensive data export for external analysis (bd-079)
- cost tracking per provider and per user (bd-077)
- beads migration documentation
- Complete request/response logging infrastructure (bd-076)
- Complete advanced provider routing epic (bd-053)
- Complete authentication & authorization epic (bd-052)
- Integrate gitops manager for project repository management
- Add streaming support, gitops infrastructure, and documentation improvements
- comprehensive TemporalManager DSL and complete documentation
- protocol dropdown and encrypted API key storage for providers
- org chart abstraction layer, entity versioning, and post-flight API tests
- kickstart for open beads, fix Docker issues, improve Makefile
- provider heartbeats, REPL, and dockerized workflows
- P0 bead for provider heartbeat self-tests
- P0 beads for dockerized tests and Temporal availability
- debug output for make failure beads
- user guide and GitHub Pages workflow
- sticky projects and YAML linting
- repository rules and ignore obj directory
- model catalog, provider negotiation, and persona defaults
- provider registry, config DB, and global dispatcher
- comprehensive MILESTONES.md document from Project Manager persona
- integration tests and example configuration files
- documentation and tests for worker system
- provider protocol, worker system, and agent worker manager
- product manager persona with future goals documentation
- comprehensive README for public relations manager persona
- public relations manager persona
- QA Engineer persona for first release test planning
- Temporal server infrastructure and event bus implementation
- project state management documentation
- engineering manager persona with design doc review capabilities
- comprehensive tests for project state management
- arbiter persona files
- project state management and arbiter persona
- QA workflow documentation
- QA engineer and project manager personas
- 5 new agent personas: Product Manager, Project Manager, Engineering Manager, Documentation Manager, and DevOps Engineer
- QUICKSTART.md guide for easy onboarding
- Makefile, CONTRIBUTING.md, update .gitignore, remove binary
- HTTP API, web UI, and main application entry point
- core managers (agent, project, beads, decision, filelock, arbiter)
- comprehensive architecture documentation
- persona system, data models, config, and OpenAPI spec
- Go-based containerized architecture for arbiter
- main arbiter application entry point
- main application and web UI dashboard
- core arbiter database and key management system
- arbiter web API and dashboard with service cost tracking
- AI coding agent orchestrator with multi-provider support
- CLI entry point for Arbiter application
- core Arbiter functionality with agent orchestration

### Changed
- bead UI and workflows
- beads after closing P0 issues
- architecture diagram for Analytics v1.1 (bd-011)
- README with analytics features (bd-010)
- provider registration form and enhance error handling
- README with completed features and documentation links

### Fixed
- transform motivations/roles API response to expected format
- disable auth for development (keep code, disable feature)
- UI auth and motivations loading issues
- correct GetProjectWorkDir for Docker environment
- add cache-busting version to JS includes
- handle </think> without opening tag in LLM response parsing
- add missing getAuthHeaders() function for motivations UI
- make analytics tests time-zone agnostic
- strip <think> tags from LLM output before JSON extraction
- CEO REPL to accept 'healthy' provider status
- all failing tests (P0)
- agent creation API to normalize persona names and close duplicate P0 beads
- streaming support by implementing Flusher in statusRecorder middleware
- provider persistence and remove mock provider bootstrap
- Show all providers in streaming test dropdown, not just active ones
- P0: Implement proper write-through cache for agent state
- auto-filing bug reporting system
- Remove example-project to prevent agent assignment regression
- analytics endpoints auth when authentication disabled
- agent persistence and add log viewer UI foundation
- Make health endpoints publicly accessible without authentication
- state persistence: Add volume mount for SQLite database
- Temporal connection for Docker environment
- provider status detection and network connectivity
- UI contrast (WCAG AA) and add project settings gear icon
- UI issues and improve agent management
- bead card contrast and docker compose YAML
- kanban text contrast and add project CRUD bead
- Makefile tee output path
- project is_sticky migrations
- bead YAML parsing warnings
- build ignore rules and Makefile targets
- compilation errors - update codebase to work with new structure
- Remove duplicate main functions and restore clean cmd/arbiter/main.go
- timestamp consistency in project manager
- broken code - remove duplicate implementations and fix syntax errors
- build issues and add persona tests
- Go dependencies and resolve merge conflicts
- JavaScript null handling and model JSON serialization
- Go version to 1.21 for consistency across project
- Go version requirement in README to match go.mod

### Added
- Initial release infrastructure
- Automated release script with semantic versioning
- CHANGELOG generation from git commits

## [2.0.0] - 2026-01-21

### Added
- **Developer Experience (Milestone v2.0)**
  - VS Code extension with AI chat panel and inline code suggestions
  - JetBrains plugin for IntelliJ IDEA, PyCharm, etc.
  - Vim/Neovim plugin with terminal-based chat interface
  - Web-based persona editor with visual UI
  - 15+ pre-built persona templates
  - Visual workflow builder documentation
  - Comprehensive persona testing and validation framework

### Added
- **Extensibility & Scale (Milestone v1.2)**
  - Custom provider plugin system
  - HTTP and native plugin support
  - Plugin manifest and discovery
  - Distributed deployment with PostgreSQL
  - Distributed locking and instance registry
  - Health check endpoints (/health, /health/live, /health/ready)
  - Graceful shutdown and startup managers
  - Load balancing support (Nginx, HAProxy, AWS ALB, GCP)
  - Horizontal scaling guide with auto-scaling patterns

### Added
- **Analytics & Intelligence (Milestone v1.1)**
  - Request logging and analytics system
  - Cost tracking per user and provider
  - Spending alerts and budget monitoring
  - Intelligent response caching with LRU eviction
  - Redis cache backend support
  - Cache invalidation by provider, model, age, pattern

### Features
- Agent orchestration with personas
- Temporal DSL for workflow coordination
- Bead-based work item tracking
- Project state management
- Multi-provider AI routing
- API authentication and authorization
- Docker and Kubernetes deployment support
- Comprehensive documentation (20+ guides)

---

*Note: Version numbers follow semantic versioning (MAJOR.MINOR.PATCH)*
