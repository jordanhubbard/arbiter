# AgentiCorp Entities Reference

Complete reference for all entities (data structures) in the AgentiCorp system.

## Table of Contents

1. [Agent](#agent)
2. [Bead](#bead)
3. [Provider](#provider)
4. [Project](#project)
5. [Decision](#decision)
6. [Persona](#persona)
7. [Workflow](#workflow)
8. [Temporal DSL Instruction](#temporal-dsl-instruction)

---

## Agent

An autonomous AI entity with a specific role and behavioral instructions.

### Model Definition

**Database Table**: `agents`

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Unique agent identifier |
| `name` | string | Human-readable agent name |
| `project_id` | UUID | Associated project ID |
| `persona_id` | UUID | Persona (role) definition ID |
| `role` | string | Org chart role (CEO, CFO, Engineer, etc.) |
| `status` | string | Current status: `idle`, `working`, `paused`, `complete` |
| `description` | string | What this agent does |
| `capabilities` | string | JSON array of capabilities |
| `current_bead_id` | UUID | Currently assigned bead (if working) |
| `messages` | string | Message log (JSON) |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

### Status Transitions

```
┌─────┐
│idle │ ─── provider_down ──→ ┌────────┐
└──┬──┘                        │paused  │
   │                           └──┬─────┘
   │ assign_bead                  │ provider_up
   │                              │
   ▼                              ▼
┌─────────┐                    ┌─────┐
│working  │  ─── complete ──→  │idle │
└─────────┘                    └─────┘
```

### Lifecycle

1. **Creation**: Agent created when assigned to a project
2. **Initial State**: `paused` (waiting for provider)
3. **Provider Activation**: Transitions to `idle` when provider becomes healthy
4. **Work Assignment**: Receives bead, transitions to `working`
5. **Completion**: Bead processed, transitions to `idle`
6. **Pause**: On provider health issue

### Example

```json
{
  "id": "agent-ceo-default",
  "name": "CEO (Default)",
  "project_id": "proj-agenticorp",
  "persona_id": "persona-ceo",
  "role": "ceo",
  "status": "idle",
  "description": "Chief Executive Officer - Strategic oversight",
  "capabilities": ["decision_making", "strategic_planning", "escalation"],
  "created_at": "2026-01-20T08:00:00Z",
  "updated_at": "2026-01-20T08:15:00Z"
}
```

---

## Bead

A discrete unit of work (task, story, decision request) in the system.

### Model Definition

**Database Table**: `beads`  
**File Storage**: `.beads/beads/*.yaml`

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (required) |
| `type` | string | Work type: `feature`, `bugfix`, `test`, `decision`, `analysis`, `review` |
| `title` | string | Short description |
| `description` | string | Detailed work description |
| `project_id` | UUID | Associated project |
| `assigned_to` | UUID | Agent assignment (null = unassigned) |
| `status` | string | `open`, `in_progress`, `blocked`, `done` |
| `priority` | int | 1-5 (5 = highest) |
| `parent_id` | string | Parent bead ID (for sub-tasks) |
| `blocked_by` | []string | IDs of blocking beads |
| `blocks` | []string | IDs of beads this blocks |
| `children_ids` | []string | Sub-task IDs |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |
| `completed_at` | timestamp | Completion time |

### Status Transitions

```
┌──────┐
│open  │ ─── assign ──→ ┌─────────────┐
└──────┘                │in_progress  │
   ▲                    └──┬──────────┘
   │                       │ complete
   │ unblock               │
   │                       ▼
┌──────────┐            ┌─────┐
│blocked   │            │done │
└──────────┘            └─────┘
```

### Dependencies

**Blocking Logic**:
- A bead can't transition to `in_progress` if any `blocked_by` beads are not `done`
- When a bead completes, dependent beads become available
- Circular dependencies are detected at load time

**Hierarchy**:
- Parent/child relationships for sub-tasks
- Parent completion doesn't require all children done
- Children inherit some priority from parent

### Example YAML

```yaml
id: bd-001-feature-auth
type: feature
title: Implement OAuth Integration
description: |
  Add OAuth 2.0 support for GitHub and Google login.
  Must support token refresh and logout.
project_id: proj-app
assigned_to: agent-engineering-001
status: in_progress
priority: 4
blocked_by:
  - bd-001-deps-oauth-lib
blocks:
  - bd-002-feature-user-settings
parent_id: null
children_ids:
  - bd-001-sub-github-oauth
  - bd-001-sub-google-oauth
created_at: "2026-01-20T08:00:00Z"
updated_at: "2026-01-20T09:30:00Z"
```

### Type Conventions

| Type | Purpose | Typical Duration |
|------|---------|------------------|
| `feature` | New capability | Hours to days |
| `bugfix` | Issue fix | Minutes to hours |
| `test` | Quality assurance | Minutes to hours |
| `decision` | Approval workflow | Real-time to hours |
| `analysis` | Investigation | Minutes to hours |
| `review` | Code/design review | Minutes to hours |

---

## Provider

An external LLM service that executes agent work.

### Model Definition

**Database Table**: `providers`  
**Registration**: UI, API, or `config.yaml`

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique provider identifier |
| `name` | string | Display name |
| `type` | string | `local`, `ollama`, `openai`, `anthropic`, `vllm`, `custom` |
| `endpoint` | string | Base URL for API calls |
| `model` | string | Model ID (e.g., `nvidia/Nemotron`) |
| `configured_model` | string | Initially configured model |
| `selected_model` | string | Currently active model |
| `status` | string | `pending`, `active`, `error` |
| `requires_key` | bool | Whether API key needed |
| `key_id` | string | Reference to encrypted key in keymanager |
| `last_heartbeat_at` | timestamp | Last health check time |
| `last_heartbeat_latency_ms` | int | Response time of last check |
| `last_heartbeat_error` | string | Last error message |
| `description` | string | Notes about provider |
| `created_at` | timestamp | Registration time |
| `updated_at` | timestamp | Last update time |

### Status Workflow

```
┌─────────┐
│pending  │ ──── health_check ───→ ┌────────┐
└─────────┘                        │active  │
                                   └───┬────┘
                                       │ heartbeat_fail
                                       ▼
                                    ┌──────┐
                                    │error │
                                    └──────┘
```

### Health Check

- **Initial**: Performed immediately on registration
- **Periodic**: Every 30 seconds via Temporal heartbeat
- **Check**: GET `/v1/models` endpoint
- **Activation**: Automatic when first successful check
- **Agent Resume**: Agents resume automatically when provider becomes active

### Example

```json
{
  "id": "puck",
  "name": "Puck vLLM",
  "type": "local",
  "endpoint": "http://puck.local:8000",
  "model": "nvidia/NVIDIA-Nemotron-3-Nano-30B-A3B-FP8",
  "status": "active",
  "requires_key": false,
  "last_heartbeat_at": "2026-01-20T08:30:00Z",
  "last_heartbeat_latency_ms": 45,
  "last_heartbeat_error": "",
  "created_at": "2026-01-20T08:00:00Z",
  "updated_at": "2026-01-20T08:30:00Z"
}
```

---

## Project

A container for beads, agents, and related work.

### Model Definition

**Database Table**: `projects`  
**Configuration**: `config.yaml` or UI

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier |
| `name` | string | Display name |
| `description` | string | Project purpose |
| `git_repo` | string | Git repository URL |
| `branch` | string | Git branch to track |
| `beads_path` | string | Path to beads (relative to repo) |
| `is_sticky` | bool | Auto-register on startup |
| `is_perpetual` | bool | Never closes, continuous operation |
| `status` | string | `active`, `archived`, `suspended` |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

### Lifecycle

1. **Creation**: Defined in config.yaml or created via UI
2. **Initialization**: Beads loaded from git
3. **Operation**: Agents assigned, work dispatched
4. **Completion**: When all beads done (unless perpetual)
5. **Archive**: Marked complete, beads no longer processed

### Example

```yaml
id: agenticorp
name: AgentiCorp
description: Agent orchestration and workflow engine
git_repo: https://github.com/jordanhubbard/agenticorp
branch: main
beads_path: .beads
is_sticky: true
is_perpetual: true
created_at: "2026-01-20T07:00:00Z"
updated_at: "2026-01-20T08:00:00Z"
```

---

## Decision

An escalation request requiring human/agent approval.

### Model Definition

**Database Table**: `decisions`

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier |
| `project_id` | UUID | Associated project |
| `question` | string | Decision being made |
| `requester_id` | UUID | Agent requesting decision |
| `status` | string | `pending`, `approved`, `denied` |
| `options` | []string | Possible responses |
| `response` | string | Chosen response |
| `responder_id` | UUID | Who made decision |
| `created_at` | timestamp | Request time |
| `responded_at` | timestamp | Response time |

### Workflow

```
Agent Requests Decision
         ↓
Decision Created (status=pending)
         ↓
UI Shows for Approval
         ↓
Human/CEO Approves or Denies
         ↓
Decision Updated (status=approved/denied)
         ↓
Agent Notified via Signal
         ↓
Agent Resumes Work
```

### Example

```json
{
  "id": "decision-001",
  "project_id": "proj-app",
  "question": "Should we proceed with 50K infrastructure investment?",
  "requester_id": "agent-cfo",
  "status": "approved",
  "options": ["approve", "deny", "defer"],
  "response": "approve",
  "responder_id": "agent-ceo",
  "created_at": "2026-01-20T08:00:00Z",
  "responded_at": "2026-01-20T08:05:00Z"
}
```

---

## Persona

Instructions, guidelines, and behavioral rules for agent roles.

### Model Definition

**File Storage**: `personas/*/name.md`  
**Categories**: `personas/default/` (standard roles), `personas/agenticorp/` (system roles)

**Structure**:

```markdown
# Role Name

## Description
What this role does and its responsibilities.

## Instructions
Detailed behavioral instructions and guidelines.

## Capabilities
- List of capabilities
- What this agent can request
- What decisions it can make

## Constraints
- Limitations and guardrails
- Escalation triggers

## Temporal DSL (Optional)
Can include <temporal>...</temporal> blocks for:
- Workflows to trigger on certain conditions
- Schedules for recurring tasks
- Queries for status checks
```

### Example: CFO Persona

```markdown
# Chief Financial Officer (CFO)

## Description
Responsible for financial oversight, budget approval, and cost management.

## Instructions
- Review all budget requests over $10,000
- Monitor ongoing project costs daily
- Alert CEO of cost overruns > 20%
- Approve or deny capital requests

## Capabilities
- Decision approval (budget, capital)
- Financial queries
- Cost analysis
- Budget forecast

## Constraints
- Cannot approve over $500,000 (needs CEO)
- Must document all decisions
- Escalate financial anomalies

## Temporal DSL

When approving large budgets:
<temporal>
WORKFLOW: LogBudgetApproval
  INPUT: {"amount": 100000, "category": "infrastructure"}
  WAIT: false
END
</temporal>

Daily monitoring:
<temporal>
SCHEDULE: DailyBudgetReview
  INTERVAL: 24h
  INPUT: {"scope": "all_projects"}
END
</temporal>
```

---

## Workflow

Temporal workflow definition representing a long-running business process.

### Key Characteristics

- **Durable**: Survives process crashes/restarts
- **Reliable**: Built-in retry logic
- **Observable**: Full history available
- **Signalable**: Can receive updates while running
- **Queryable**: Can report state on demand

### Core Workflows

| Workflow | Purpose | Interval | Inputs |
|----------|---------|----------|--------|
| `AgentiCorpHeartbeatWorkflow` | Master clock | 10s | - |
| `DispatcherWorkflow` | Route work | 5s | Project ID (optional) |
| `ProviderHeartbeatWorkflow` | Health checks | 30s | Provider ID |
| `BeadProcessingWorkflow` | Execute bead | On-demand | Bead ID, Agent ID |
| `DecisionWorkflow` | Escalation | On-demand | Decision details |

### Workflow Lifecycle

```
Start
  ↓
Execute Activities
  ↓
Receive Signals
  ↓
Respond to Queries
  ↓
Decision
  ├─ Retry: Go back to Execute
  └─ Complete: Return Result
```

---

## Temporal DSL Instruction

A parsed Temporal DSL command specifying an operation.

### Model Definition

**Type**: `TemporalInstruction` in `internal/temporal/dsl_types.go`

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `type` | enum | Instruction type (WORKFLOW, SCHEDULE, QUERY, etc.) |
| `name` | string | Workflow/Activity name |
| `workflow_id` | string | For QUERY/SIGNAL/CANCEL operations |
| `input` | map | JSON input parameters |
| `timeout` | duration | Execution timeout |
| `retry` | int | Retry attempts |
| `wait` | bool | Wait for completion |
| `interval` | duration | For SCHEDULE instructions |
| `query_type` | string | Query type name |
| `signal_name` | string | Signal name |
| `signal_data` | map | Signal payload |

### Instruction Types

**WORKFLOW**: Schedule a workflow
```markdown
<temporal>
WORKFLOW: ProcessBatch
  INPUT: {"batch_id": "123"}
  TIMEOUT: 5m
  WAIT: true
END
</temporal>
```

**SCHEDULE**: Recurring execution
```markdown
<temporal>
SCHEDULE: HourlyCheck
  INTERVAL: 1h
  INPUT: {"type": "comprehensive"}
END
</temporal>
```

**QUERY**: Get workflow state
```markdown
<temporal>
QUERY: wf-123
  TYPE: get_status
END
</temporal>
```

**SIGNAL**: Send message to workflow
```markdown
<temporal>
SIGNAL: wf-123
  NAME: approve
  DATA: {"amount": 50000}
END
</temporal>
```

**ACTIVITY**: Execute activity directly
```markdown
<temporal>
ACTIVITY: FetchData
  INPUT: {"source": "api"}
  TIMEOUT: 2m
END
</temporal>
```

**CANCEL**: Stop workflow
```markdown
<temporal>
CANCEL: wf-123
END
</temporal>
```

**LIST**: List running workflows
```markdown
<temporal>
LIST
END
</temporal>
```

See `docs/TEMPORAL_DSL.md` for comprehensive DSL documentation.

---

## Entity Relationships

```
┌─────────┐
│Project  │
└────┬────┘
     │
     ├─── 1 to Many ──→ ┌──────┐
     │                  │Bead  │
     │                  └──────┘
     │
     └─── 1 to Many ──→ ┌───────┐
                        │Agent  │
                        └───┬───┘
                            │
                        assigned_to
                            │
                            ▼
                        ┌──────┐
                        │Bead  │
                        └──────┘

┌──────────┐
│Provider  │
└────┬─────┘
     │
     │ serves
     │
     ▼
┌──────────────┐
│Agent         │
└──────────────┘
     │
     │ requests
     │
     ▼
┌──────────────┐
│Decision      │
└──────────────┘

┌──────────────────────┐
│Temporal Instruction  │
└──────┬───────────────┘
       │
       ├─ WORKFLOW (starts) ──→ Workflow Execution
       ├─ QUERY (queries)   ──→ Workflow State
       ├─ SIGNAL (updates)  ──→ Running Workflow
       └─ SCHEDULE (creates) → Recurring Execution
```

---

## Data Persistence

All entities are persisted to:

1. **SQLite Database** (`agenticorp.db`)
   - Primary storage for agents, beads, providers, projects, decisions
   - Transactional consistency
   - Auto-backup capability

2. **YAML Files** (Beads)
   - `.beads/beads/*.yaml` in each project repo
   - Source of truth for work definitions
   - Versioned in git

3. **Temporal Server**
   - Workflow execution state
   - Full history of all operations
   - Durable, recoverable

## Entity Lifecycle Summary

| Entity | Created | Modified | Deleted | Persisted |
|--------|---------|----------|---------|-----------|
| Agent | On project assignment | Status changes | On project delete | Database |
| Bead | On project load | Status/assignment changes | Manual | Database + YAML |
| Provider | UI/API/config | Health status | UI/API | Database |
| Project | config.yaml/UI | Config changes | UI | Database |
| Decision | Agent escalation | Response | Auto-cleanup | Database |
| Persona | File creation | Content edit | File delete | File + Database |
| Workflow | Temporal DSL | Signals/queries | Timeout | Temporal |
