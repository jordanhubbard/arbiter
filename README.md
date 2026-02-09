## Important Note from Loom's co-creator, Jordan Hubbard:
* Loom's first prompts may have come from me, but its own name as well as its story below is all written by Loom itself.  Loom also became fully self-maintaining before its initial release as this was a key release milestone.

# Loom

> *"From a single thread of an idea, we weave complete software."*

**Autonomous AI Agent Orchestration Platform**

## The Story of Loom

For thousands of years, master weavers have transformed simple threads into magnificent tapestries. They didn't work alone—apprentices prepared threads, dyers created colors, designers planned patterns. Each specialist contributed their expertise, and the master weaver coordinated them all through the loom.

Software development follows the same ancient pattern. A project manager plans the architecture, engineers write code, QA tests functionality, designers craft interfaces. But traditionally, these specialists worked in sequence, with humans coordinating every handoff, every decision, every integration.

**Loom changes this.**

Just as a master weaver operates a loom to coordinate multiple threads into unified fabric, Loom orchestrates multiple AI agents—each with specialized expertise—to weave complete software from a simple Product Requirements Document (PRD). You provide the thread of an idea. Loom weaves it into reality.

Give Loom a PRD, and watch as:
- A **Project Manager** expands your requirements with best practices
- An **Engineering Manager** builds the core architecture
- **QA Engineers** write comprehensive tests
- **DevOps Engineers** configure infrastructure
- **Designers** craft the user interface
- A **Code Reviewer** ensures quality throughout

All working autonomously, in parallel, coordinated by Loom's workflow engine. From PRD to working MVP in hours, not weeks.

This is the craft of software, elevated by the ancient wisdom of the loom.

---

## Quick Start

Want to get Loom running? See **[SETUP.md](SETUP.md)** for installation, connecting to the UI, and bootstrapping your first project.

---

## What is Loom?

Loom is a lightweight AI coding agent orchestration system that manages workflows, handles agent lifecycle, and provides real-time event streaming for monitoring and coordination.

**Core Capabilities:**

- **🚀 Project Bootstrap**: Create complete projects from a PRD
  - Autonomous PRD expansion with best practices
  - Automatic epic and story breakdown
  - Agent work assignment and orchestration
  - CEO review and approval workflows

- **🤖 Multi-Agent Orchestration**: Specialized AI agents working in harmony
  - Project Managers, Engineers, QA, DevOps, Designers
  - Parallel task execution with dependency management
  - Autonomous decision-making within defined guardrails

- **🔄 Workflow Engine**: Temporal-based reliable execution
  - Durable workflows that survive failures
  - Complex multi-step processes
  - Human-in-the-loop approval gates

- **📊 Git-Backed Issue Tracking**: Beads system for persistent work tracking
  - Issues survive context compaction
  - Dependency tracking (blockers, blocked-by)
  - Cross-session context recovery

- **🎯 Intelligent Routing**: Smart work assignment
  - Role-based bead matching
  - Workflow-based task progression
  - Priority and tag-based filtering

## Documentation

**Start here**: [SETUP.md](SETUP.md) — Get Loom running in minutes

**By persona:**
- **User** — [User Guide](docs/USER_GUIDE.md) — Log in, create projects, monitor progress, work with decisions
- **Administrator** — [Admin Guide](docs/ADMIN_GUIDE.md) — Providers, deploy keys, users, monitoring, backup
- **Developer** — [Architecture](docs/ARCHITECTURE.md) — System design, mermaid diagrams, components

**Reference:**
- [Authentication & RBAC](docs/AUTH.md) — JWT, API keys, roles, permission matrix
- [Entities Reference](docs/ENTITIES_REFERENCE.md) — All data structures explained
- [Temporal DSL Guide](docs/TEMPORAL_DSL.md) — Workflow language for agents
- [Analytics Guide](docs/ANALYTICS_GUIDE.md) — Usage monitoring and cost tracking
- [Usage Pattern Analysis](docs/usage-pattern-analysis.md) — Pattern detection and optimization
- [Activity Feed & Notifications](docs/activity-notifications-implementation.md) — Event tracking and alerts
- [Developer Guide](AGENTS.md) — For contributors and custom agents

## Features

- 🤖 **Agent Orchestration**: Spawn and manage AI agents with different personas
- 🔄 **Workflow Management**: Temporal-based workflow orchestration for reliable task execution
- 📊 **Work Graph**: Track dependencies and relationships between work items (beads)
- 🔐 **Decision Framework**: Approval workflows for agent decisions
- 🔏 **API Auth & RBAC**: JWT bearer tokens, API keys, and role-based permissions
- 📡 **Real-time Events**: Server-Sent Events (SSE) for live status updates
- 🔔 **Activity Feed & Notifications**: Team activity tracking with intelligent user notifications
- 🎯 **Smart Routing**: Intelligent task assignment and agent coordination
- 🔒 **Secure**: Encrypted secret storage and secure credential management
- 📈 **Analytics & Cost Tracking**: Real-time usage monitoring, cost tracking, and spending alerts
- 🔍 **Usage Pattern Analysis**: Multi-dimensional pattern clustering, anomaly detection, and cost optimization recommendations

## Default Personas

Default personas are available under `./personas/`:

- `personas/loom` — Loom-specific system persona(s)
- `personas/default/ceo` — Human CEO decision maker (tie-breaks / approvals)
- `personas/default/project-manager` — Plans work, files beads, drives delivery
- `personas/default/product-manager` — Identifies feature gaps and writes PRDs for epics
- `personas/default/engineering-manager` — Reviews technical direction and feasibility
- `personas/default/code-reviewer` — Reviews patches for correctness and quality
- `personas/default/qa-engineer` — Testing strategy and verification
- `personas/default/devops-engineer` — Deployment/ops and infrastructure guidance
- `personas/default/documentation-manager` — Keeps docs accurate per doc policy
- `personas/default/decision-maker` — Resolves routine decisions (non-CEO)
- `personas/default/web-designer` — UX/UI guidance
- `personas/default/web-designer-engineer` — UX/UI + implementation guidance
- `personas/default/public-relations-manager` — Messaging/launch communication support
- `personas/default/housekeeping-bot` — Cleanup and hygiene tasks

## Architecture

Loom is built with the following principles:

- **Go-First Implementation**: All primary functionality is implemented in Go for performance and maintainability
- **Containerized Everything**: Every component runs in containers for consistency across environments
- **Temporal Workflows**: Reliable, durable workflow orchestration using Temporal
- **Event-Driven**: Real-time event bus for agent communication and UI updates

## Temporal Workflow Engine

Loom uses [Temporal](https://temporal.io) for reliable workflow orchestration. Temporal provides:

- **Durable Execution**: Workflows survive crashes and restarts
- **Event History**: Complete audit trail of all workflow executions
- **Signals & Queries**: Real-time workflow interaction
- **Timeout Management**: Automatic handling of long-running operations

### Temporal Components

The system includes:

1. **Temporal Server**: Core workflow engine (port 7233)
2. **Temporal UI**: Web interface for monitoring workflows (port 8088)
3. **PostgreSQL**: Persistence layer for workflow state
4. **Temporal Worker**: Executes workflow and activity code

### Workflows

Loom implements several key workflows:

#### Agent Lifecycle Workflow
Manages the complete lifecycle of an agent from spawn to shutdown:
- Tracks agent status (spawned, working, idle, shutdown)
- Handles bead assignments
- Responds to queries for current status
- Gracefully shuts down on signal

#### Bead Processing Workflow
Manages work item (bead) lifecycle:
- Tracks status transitions (open, in_progress, blocked, closed)
- Handles agent assignments
- Manages dependencies and blockers
- Provides status queries

#### Decision Workflow
Handles approval workflows with timeout:
- Creates decision points for agent questions
- Waits for human or agent approval
- 48-hour default timeout
- Unblocks dependent work on resolution

### Event Bus

The Temporal-based event bus provides real-time updates:

```
Event Types:
- agent.spawned        - New agent created
- agent.status_change  - Agent status updated
- agent.completed      - Agent finished work
- bead.created         - New work item created
- bead.assigned        - Work assigned to agent
- bead.status_change   - Work status updated
- bead.completed       - Work item finished
- decision.created     - Decision point created
- decision.resolved    - Decision made
- log.message          - System log message
```

## API Endpoints

### Core Resources

```bash
# Health check
GET /api/v1/health

# Agents
GET    /api/v1/agents
POST   /api/v1/agents
GET    /api/v1/agents/{id}

# Beads (work items)
GET    /api/v1/beads
POST   /api/v1/beads
GET    /api/v1/beads/{id}
PUT    /api/v1/beads/{id}

# Decisions
GET    /api/v1/decisions
POST   /api/v1/decisions
PUT    /api/v1/decisions/{id}

# Projects
GET    /api/v1/projects
GET    /api/v1/projects/{id}

# Work Graph
GET    /api/v1/work-graph?project_id={id}

# Analytics & Cost Tracking
GET    /api/v1/analytics/logs
GET    /api/v1/analytics/stats
GET    /api/v1/analytics/costs
GET    /api/v1/analytics/export
GET    /api/v1/analytics/export-stats
```

### Event Streaming

Real-time event streaming via Server-Sent Events:

```bash
# Stream all events
GET /api/v1/events/stream

# Stream events for specific project
GET /api/v1/events/stream?project_id=example-project

# Stream specific event types
GET /api/v1/events/stream?type=agent.spawned

# Get event statistics
GET /api/v1/events/stats
```

Example: Subscribe to events using curl:
```bash
curl -N http://localhost:8080/api/v1/events/stream
```

Example: Subscribe to events using JavaScript:
```javascript
const eventSource = new EventSource('http://localhost:8080/api/v1/events/stream?project_id=my-project');

eventSource.addEventListener('agent.spawned', (e) => {
  const data = JSON.parse(e.data);
  console.log('Agent spawned:', data);
});

eventSource.addEventListener('bead.created', (e) => {
  const data = JSON.parse(e.data);
  console.log('Bead created:', data);
});
```

## Project State Management

Loom supports sophisticated project lifecycle management:

### Project States
- **Open**: Active project with ongoing work
- **Closed**: Completed project with no remaining work
- **Reopened**: Previously closed project that has been reopened

### Features
- **Comments**: Add timestamped comments to track project decisions
- **Closure Workflow**: Close projects only when no open work remains
- **Agent Consensus**: If open work exists, requires agent agreement to close
- **Perpetual Projects**: Mark projects (like Loom itself) that never close

### API Endpoints

```bash
# Close a project
POST /api/v1/projects/{id}/close
{
  "author_id": "agent-123",
  "comment": "All features complete, tests passing"
}

# Reopen a project
POST /api/v1/projects/{id}/reopen
{
  "author_id": "agent-456",
  "comment": "New requirements discovered"
}

# Add a comment
POST /api/v1/projects/{id}/comments
{
  "author_id": "agent-789",
  "comment": "Architecture review complete"
}

# Get project state
GET /api/v1/projects/{id}/state
```

## The Loom Persona

The Loom system includes a special **loom** persona that works on improving the Loom platform itself:

- **Self-Improving**: Continuously enhances the platform
- **Collaborative**: Works with UX, Engineering, PM, and Product personas
- **Perpetual**: The loom project never closes
- **Meta-Circular**: An AI orchestrator that orchestrates its own improvement

See `personas/loom/` for the complete persona definition.

## Completed Features

- [x] Project state management (open, closed, reopened)
- [x] Project comments and closure workflow
- [x] Loom persona for self-improvement
- [x] Perpetual projects that never close
- [x] Provider registration and health checking
- [x] Agent orchestration with personas
- [x] Work item (bead) management with dependencies
- [x] Decision approval workflows
- [x] Real-time event streaming (SSE)
- [x] Temporal workflow orchestration
- [x] Database state persistence (SQLite)
- [x] Web UI for monitoring and control
- [x] Provider status detection and activation
- [x] Master heartbeat and dispatcher workflows
- [x] Temporal DSL for agent workflow requests
- [x] Complete documentation and user manual
- [x] Analytics dashboard with real-time usage monitoring
- [x] Per-user and per-provider cost tracking
- [x] Data export (CSV/JSON) for external analysis
- [x] Spending alerts with anomaly detection
- [x] Privacy-first logging with GDPR compliance

## Planned Features

- [ ] Implement HTTP response streaming for real-time provider output
- [ ] Implement provider pooling and load balancing
- [ ] Add per-provider rate limiting and quotas
- [ ] Support custom provider plugins
- [ ] Add caching layer for frequently used models
- [ ] Implement multi-region provider failover
- [ ] Support for custom authentication to providers
- [ ] Metrics and monitoring endpoints (Prometheus compatible)
- [ ] Email/webhook notifications for alerts
- [ ] Advanced charting and trend analysis

## Project Structure

```
loom/
├── cmd/loom/              # Main application entry point
│   └── main.go
├── internal/
│   ├── agent/               # Agent management
│   ├── loom/             # Core orchestrator
│   ├── beads/               # Work item management
│   ├── decision/            # Decision framework
│   ├── temporal/            # Temporal integration
│   │   ├── client/          # Temporal client wrapper
│   │   ├── workflows/       # Workflow definitions
│   │   ├── activities/      # Activity implementations
│   │   ├── eventbus/        # Event bus implementation
│   │   └── manager.go       # Temporal manager
│   ├── api/                 # HTTP API handlers
│   └── models/              # Data models
├── pkg/
│   ├── config/              # Configuration management
│   └── models/              # Shared models
├── config/
│   └── temporal/            # Temporal configuration
├── docker-compose.yml       # Container orchestration
├── Dockerfile              # Multi-stage Docker build
├── config.yaml.example     # Example configuration
└── README.md              # This file
```

## Development Guidelines

1. **Primary Language**: Implement all core functionality in Go
2. **Containerization**: All services must run in containers
3. **Workflows**: Use Temporal workflows for long-running operations
4. **Events**: Publish events for all state changes
5. **Testing**: Write tests for workflows and activities
6. **Documentation**: Update docs for new features

## Contributing

When contributing to this project:

1. Ensure all code follows the architecture principles
2. All new features must be containerized
3. **File a bead for your work** — See [BEADS_WORKFLOW.md](docs/BEADS_WORKFLOW.md)
4. Write Temporal workflows for async operations
5. Add appropriate event publishing
6. Update documentation
7. Run tests and linters before submitting

For detailed contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

See LICENSE file for details.
