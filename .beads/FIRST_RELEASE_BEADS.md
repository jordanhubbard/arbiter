# Arbiter Project - First Release Beads

This document defines the work items (beads) for the first release of the Arbiter project, as determined by the project manager persona.

## Release Goal

Deliver a functional MVP of Arbiter that can:
- Manage and track projects
- Spawn and coordinate AI agents with different personas
- Track work items (beads) and their dependencies
- Provide a web UI for monitoring
- Handle basic agent orchestration

## First Release Beads

### Epic: Core Infrastructure (P1)
The foundation needed for Arbiter to function.

#### BD-001: Project Registration and Configuration (P1)
**Type:** task
**Title:** Register arbiter project with itself
**Description:** Create a configuration entry that registers the arbiter project as the first project in the system. This includes setting up the git repo path, branch, beads path, and build/test commands.
**Acceptance Criteria:**
- config.yaml includes arbiter project entry
- Project ID: "arbiter"
- Git repo points to current repository
- Beads path set to ".beads"
- Build and test commands configured

#### BD-002: Bead Storage Initialization (P1)
**Type:** task
**Title:** Initialize .beads directory for arbiter project
**Description:** Create the .beads directory structure where all project beads will be stored and tracked.
**Acceptance Criteria:**
- .beads directory exists in project root
- Directory is git-ignored for local development
- Structure follows beads framework conventions

#### BD-003: Core API Endpoints (P1)
**Type:** task
**Title:** Verify core API endpoints are functional
**Description:** Ensure all essential API endpoints for projects, beads, agents, and personas are working correctly.
**Acceptance Criteria:**
- /api/v1/projects endpoints work
- /api/v1/beads endpoints work
- /api/v1/agents endpoints work
- /api/v1/personas endpoints work
- Basic error handling in place

### Epic: Agent & Persona System (P1)
Enable agents with different capabilities to work on the project.

#### BD-004: Persona Directory Structure (P1)
**Type:** task
**Title:** Ensure persona system is functional
**Description:** Verify that the persona loading and management system works correctly, including all example personas.
**Acceptance Criteria:**
- Personas load from ./personas directory
- code-reviewer persona functional
- decision-maker persona functional
- housekeeping-bot persona functional
- project-manager persona functional
- Persona API returns valid data

#### BD-005: Agent Spawning (P1)
**Type:** task
**Title:** Test agent spawning and lifecycle
**Description:** Ensure agents can be spawned with different personas and managed through their lifecycle.
**Acceptance Criteria:**
- Can spawn agent with any persona
- Agent status tracked correctly
- Agents can claim beads
- Agent cleanup works

### Epic: Work Management (P1)
The system for tracking and coordinating work.

#### BD-006: Bead Creation and Management (P1)
**Type:** task
**Title:** Implement bead CRUD operations
**Description:** Ensure beads can be created, read, updated, and deleted through the API and internally.
**Acceptance Criteria:**
- Create beads with all required fields
- List beads with filtering
- Update bead status
- Track bead dependencies
- Handle bead claiming by agents

#### BD-007: Work Graph Visualization (P2)
**Type:** task
**Title:** Display work dependencies in UI
**Description:** Show the relationship between beads in the web UI to help understand project structure.
**Acceptance Criteria:**
- Work graph API returns proper data
- UI displays bead dependencies
- Blocked beads clearly indicated
- Can navigate between related beads

### Epic: Web Interface (P2)
User interface for monitoring and managing the system.

#### BD-008: Dashboard UI (P2)
**Type:** task
**Title:** Build basic dashboard interface
**Description:** Create a web dashboard that shows projects, active agents, and current beads.
**Acceptance Criteria:**
- Shows list of projects
- Shows active agents and their status
- Shows current beads by status
- Real-time updates (or periodic refresh)
- Clean, usable interface

#### BD-009: Bead Management UI (P2)
**Type:** task
**Title:** Add UI for creating and managing beads
**Description:** Allow users to create new beads and update their status through the web interface.
**Acceptance Criteria:**
- Form to create new beads
- Can set priority and type
- Can assign to agents
- Can update bead status
- Can view bead details

### Epic: Documentation (P2)
Essential documentation for users and developers.

#### BD-010: Quick Start Guide (P2)
**Type:** task
**Title:** Complete QUICKSTART.md
**Description:** Ensure the quick start guide is complete and accurate for new users.
**Acceptance Criteria:**
- Installation instructions tested
- Configuration examples work
- First-run experience documented
- Common issues addressed
- Examples are runnable

#### BD-011: API Documentation (P2)
**Type:** task
**Title:** Document all API endpoints
**Description:** Create or update API documentation for all endpoints.
**Acceptance Criteria:**
- All endpoints documented
- Request/response examples provided
- Error codes explained
- OpenAPI/Swagger spec if possible

### Epic: Testing & Quality (P2)
Ensure the system is reliable and maintainable.

#### BD-012: Core Functionality Tests (P2)
**Type:** task
**Title:** Add tests for core functionality
**Description:** Ensure critical paths have test coverage.
**Acceptance Criteria:**
- Project management tests
- Bead management tests
- Agent lifecycle tests
- API endpoint tests
- Tests pass in CI

#### BD-013: Integration Testing (P3)
**Type:** task
**Title:** End-to-end integration tests
**Description:** Test complete workflows from project creation through agent work.
**Acceptance Criteria:**
- Can create project and spawn agent
- Agent can claim and complete bead
- Dependencies work correctly
- File locking works
- Tests document expected behavior

### Epic: Deployment (P3)
Make it easy to deploy and run Arbiter.

#### BD-014: Docker Compose Setup (P3)
**Type:** task
**Title:** Verify Docker Compose configuration
**Description:** Ensure Arbiter runs correctly in Docker with all dependencies.
**Acceptance Criteria:**
- docker-compose up starts all services
- Configuration persists across restarts
- Logs are accessible
- Health checks work

#### BD-015: Build and Release Process (P3)
**Type:** task
**Title:** Document build and release process
**Description:** Ensure anyone can build and release Arbiter.
**Acceptance Criteria:**
- Make targets work correctly
- Build process documented
- Release tagging process defined
- Binary distribution method decided

## Dependencies

- BD-001 (Project Registration) → Blocks BD-002, BD-006
- BD-002 (Bead Storage) → Blocks BD-006
- BD-003 (Core API) → Blocks BD-008, BD-009
- BD-004 (Personas) → Blocks BD-005
- BD-005 (Agent Spawning) → Blocks BD-007
- BD-006 (Bead Management) → Blocks BD-007, BD-009
- BD-007 (Work Graph) → Blocks BD-008
- BD-010 (Quick Start) → Should reference BD-001, BD-005, BD-006

## Notes

This is the initial plan for the first release. Priorities and scope may adjust based on:
- Technical discoveries during implementation
- User feedback
- Resource availability
- Timeline constraints

The focus is on delivering a working MVP that demonstrates the core value proposition of Arbiter: orchestrating multiple AI agents working on a codebase with coordination and visibility.
