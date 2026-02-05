# Auto-Bug Dispatch System

## Overview

The **Auto-Bug Dispatch System** automatically triages and routes auto-filed bugs to the appropriate technical agents for investigation and fixing. This creates a self-healing workflow where bugs detected in production are automatically assigned to specialists.

## Architecture

### Components

1. **Auto-Filing** (`internal/api/handlers_auto_file.go`)
   - Frontend/backend errors are automatically reported
   - Creates beads with `[auto-filed]` prefix
   - Initially assigned to QA Engineer

2. **Auto-Bug Router** (`internal/dispatch/autobug_router.go`)
   - Analyzes bug type (JavaScript, Go, API, Build, etc.)
   - Determines appropriate persona/role
   - Adds persona hint to bead title

3. **Dispatcher** (`internal/dispatch/dispatcher.go`)
   - Integrates auto-bug routing into dispatch loop
   - Allows P0 auto-filed bugs to be dispatched (normally P0 beads require CEO approval)
   - Routes beads to agents based on persona hints

4. **Persona Matcher** (`internal/dispatch/persona_matcher.go`)
   - Matches persona hints to actual agents
   - Finds idle agents with matching roles

## Workflow

```
┌─────────────────────┐
│  Error Occurs       │
│  (UI or Backend)    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Auto-File Bug      │
│  via /api/auto-file │
│  ├─ Create bead     │
│  ├─ Tag: auto-filed │
│  └─ Assign: QA      │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Dispatcher Runs    │
│  GetReadyBeads()    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Auto-Bug Router    │
│  Analyzes Bug Type  │
│  ├─ JS Error?       │
│  ├─ Go Error?       │
│  ├─ Build Error?    │
│  └─ API Error?      │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Add Persona Hint   │
│  Update Title       │
│  ├─ [web-designer]  │
│  ├─ [backend-eng]   │
│  └─ [devops-eng]    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Persona Matcher    │
│  Find Idle Agent    │
│  with Matching Role │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Dispatch to Agent  │
│  ├─ Claim bead      │
│  ├─ Execute task    │
│  └─ Fix bug         │
└─────────────────────┘
```

## Bug Type Routing

### Frontend JavaScript Errors → Web Designer
**Indicators:**
- `javascript`, `syntaxerror`, `referenceerror`, `typeerror`
- `undefined`, `is not a function`, `is not defined`
- `ui error`, `uncaught`
- Tags: `frontend`, `javascript`, `js_error`

**Example:**
```
Input:  [auto-filed] [frontend] UI Error: ReferenceError: apiCall is not defined
Output: [web-designer] [auto-filed] [frontend] UI Error: ReferenceError: apiCall is not defined
```

### Backend Go Errors → Backend Engineer
**Indicators:**
- `panic`, `runtime error`, `nil pointer`
- `invalid memory`, `undefined:`, `cannot use`
- `go build`, `compilation error`
- Tags: `backend`, `golang`, `go_error`

**Example:**
```
Input:  [auto-filed] [backend] panic: runtime error: nil pointer dereference
Output: [backend-engineer] [auto-filed] [backend] panic: runtime error: nil pointer dereference
```

### Build/Deployment Errors → DevOps Engineer
**Indicators:**
- `build failed`, `docker`, `dockerfile`
- `makefile`, `ci/cd`, `pipeline`, `container`
- `deployment`, `compile`
- Tags: `build`, `deployment`, `docker`

**Example:**
```
Input:  [auto-filed] build failed - Docker compilation error
Output: [devops-engineer] [auto-filed] build failed - Docker compilation error
```

### API/HTTP Errors → Backend Engineer
**Indicators:**
- `api error`, `api request failed`, `http`
- `status code`, `endpoint`, `404`, `500`, `502`
- Tags: `api`, `api_error`, `http`

**Example:**
```
Input:  [auto-filed] [frontend] API Error: 500 Internal Server Error
Output: [backend-engineer] [auto-filed] [frontend] API Error: 500 Internal Server Error
```

### Database Errors → Backend Engineer
**Indicators:**
- `database`, `sql`, `query`, `postgres`, `sqlite`
- `connection refused`, `deadlock`, `constraint`
- Tags: `database`, `sql`, `db_error`

### CSS/Styling Errors → Web Designer
**Indicators:**
- `css`, `style`, `layout`, `rendering`
- `flexbox`, `grid`, `responsive`, `display`
- Tags: `css`, `styling`, `ui`

## Configuration

### Allowing P0 Auto-Filed Bugs

By default, P0 beads require CEO approval before dispatch. The auto-bug system makes an exception:

```go
// Skip P0 beads UNLESS they are auto-filed bugs
isAutoFiled := strings.Contains(strings.ToLower(b.Title), "[auto-filed]")
if b.Priority == models.BeadPriorityP0 && !isAutoFiled {
    skippedReasons["p0_priority"]++
    continue
}
```

This allows critical auto-filed bugs to be immediately dispatched to technical agents.

### Dispatch Hop Limit

To prevent infinite redispatch loops, the dispatcher tracks how many times a bead has been dispatched. If a bead exceeds the configured `max_hops` (default: 20), it is escalated to P0 priority and a CEO decision bead is created. This allows complex bug investigations to proceed through multiple iterations while still catching genuinely stuck work. See [DISPATCH_CONFIG.md](DISPATCH_CONFIG.md) for configuration details:

```go
// Check dispatch count and escalate if needed
if dispatchCount >= maxHops {
    // Escalate to P0 and create CEO decision
    updates["priority"] = models.BeadPriorityP0
    updates["status"] = models.BeadStatusOpen
    updates["assigned_to"] = ""
}
```

This ensures stuck beads don't cycle indefinitely and receive human attention.

## Agent Investigation Guidelines

When an agent receives an auto-filed bug bead, the following context is available:

### Bead Title
Contains error summary with persona hint:
```
[backend-engineer] [auto-filed] [frontend] API Error: 500 Internal Server Error
```

### Bead Description
Structured bug report:
```markdown
## Auto-Filed Bug Report

**Source:** frontend
**Error Type:** js_error
**Severity:** high
**Occurred At:** 2026-01-27T00:30:12Z

### Error Message
ReferenceError: apiCall is not defined

### Stack Trace
at app.js:3769:45
at loadMotivations (app.js:3750:10)

### Context
{
  "url": "http://localhost:8080/",
  "line": 3769,
  "column": 45,
  "source_file": "app.js",
  "user_agent": "Chrome/144.0.0.0",
  "viewport": "1803x1045"
}
```

### Agent Actions

1. **Read the error details** from bead description
2. **Search for relevant code** using file/line information
3. **Identify root cause** (e.g., API_BASE declared twice)
4. **Propose fix** in bead comments or create PR
5. **Update bead** with findings and solution

## Testing

Run auto-bug router tests:
```bash
go test ./internal/dispatch -v -run TestAutoBugRouter
```

**Test Coverage:**
- Frontend JS errors → web-designer
- Backend Go errors → backend-engineer
- Build errors → devops-engineer
- API errors → backend-engineer
- Database errors → backend-engineer
- CSS errors → web-designer
- Non-auto-filed bugs → not routed
- Already-triaged bugs → not re-routed

## Future Enhancements

### Phase 1 (Current)
- ✅ Auto-file bugs from errors
- ✅ Analyze bug type
- ✅ Route to appropriate agent
- ✅ Allow P0 auto-filed bugs

### Phase 2 (Planned)
- ⏳ Agent actually fixes the code
- ⏳ Agent creates PR with fix
- ⏳ CEO approval for self-modification
- ⏳ Hot-reload mechanism

### Phase 3 (Future)
- 📋 Learning from past fixes
- 📋 Confidence scoring for auto-fixes
- 📋 A/B testing bug fixes
- 📋 Rollback on regression

## Monitoring

### View Auto-Filed Bugs
```bash
curl http://localhost:8080/api/v1/beads?tags=auto-filed
```

### View Dispatch History
```bash
curl http://localhost:8080/api/v1/beads/{bead-id}
# Check context.dispatch_history
```

### View Routing Logs
```
[Dispatcher] Auto-bug detected: ac-058 - routing to backend-engineer (API error detected)
[Dispatcher] Matched bead ac-058 to agent agent-12345 via persona hint 'backend-engineer'
```

## Troubleshooting

### Bug Not Being Dispatched

1. **Check if auto-filed**:
   ```bash
   curl http://localhost:8080/api/v1/beads/{bead-id} | jq '.title, .tags'
   ```
   Should contain `[auto-filed]` or have `auto-filed` tag.

2. **Check routing analysis**:
   ```bash
   # Look for dispatcher logs
   grep "Auto-bug detected" logs/agenticorp.log
   ```

3. **Check for persona hint**:
   Bug should have persona like `[web-designer]` in title after routing.

4. **Check for idle agents**:
   ```bash
   curl http://localhost:8080/api/v1/agents | jq '.[] | select(.status == "idle")'
   ```

### Bug Routed to Wrong Agent

The routing priority is:
1. Build/deployment errors → devops-engineer
2. Frontend JS errors → web-designer
3. Backend Go errors → backend-engineer
4. API errors → backend-engineer
5. Database errors → backend-engineer
6. CSS errors → web-designer

If a bug matches multiple patterns, the first match wins.

### Agent Not Acting on Bug

Check agent's current work:
```bash
curl http://localhost:8080/api/v1/agents/{agent-id} | jq '.current_bead_id'
```

Agents may be:
- Working on another bead
- Provider is paused/inactive
- Agent is in error state

## See Also

- [Auto-Filing System](./auto-filing.md)
- [Dispatcher Architecture](./dispatcher.md)
- [Persona System](./personas.md)
- [Bead Management](./beads.md)
