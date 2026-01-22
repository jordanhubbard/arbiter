# AgentiCorp API Capabilities Summary

**Date:** 2026-01-22  
**System Status:** Running and Healthy  
**Authentication:** admin/admin (working, token-based JWT)

## Executive Summary

**Yes to all your questions!** 

1. ✅ **All UI features are implemented through API calls**
2. ✅ **You can call the CEO/agents via API/curl** 
3. ✅ **You can invoke agents via API to review the codebase**
4. ✅ **You can check provider status via API**

---

## Current System Status

### Running Containers
```
- agenticorp (port 8080) - HEALTHY
- temporal (port 7233) - HEALTHY  
- temporal-postgresql - HEALTHY
- temporal-ui (port 8088) - HEALTHY
```

### Active Agents (Currently 10/10 - at capacity)
```
1. agent-1769059130-Project Manager (Default) - paused
2. agent-1769059130-Documentation Manager (Default) - paused
3. agent-1769059130-Web Designer (Default) - paused ⭐
4. agent-1769059130-CFO (Default) - paused
5. agent-1769059130-QA Engineer (Default) - paused
6. agent-1769059130-Devops Engineer (Default) - paused
7. agent-1769059130-Code Reviewer (Default) - paused
8. agent-1769059130-CEO (Default) - paused ⭐
9. agent-1769059130-Product Manager (Default) - paused
10. agent-1769059130-Engineering Manager (Default) - paused
```

**Note:** All agents are currently PAUSED because the only registered provider is a mock provider (`mock-local`) which doesn't support real work.

### Registered Providers
```json
{
  "id": "mock-local",
  "name": "Local Mock Provider",
  "type": "mock",
  "status": "active",
  "endpoint": "mock://local"
}
```

**Action needed:** Register a real LLM provider to enable agent work.

---

## API Endpoints - Complete Coverage

### Authentication & Authorization ✅
```bash
# Login
POST /api/v1/auth/login
{"username":"admin","password":"admin"}

# Get current user
GET /api/v1/auth/me

# Change password
POST /api/v1/auth/change-password

# Generate API key
POST /api/v1/auth/api-keys

# List users (admin)
GET /api/v1/auth/users
```

### Provider Management ✅
```bash
# List providers (includes health status)
GET /api/v1/providers
Authorization: Bearer <token>

# Register new provider
POST /api/v1/providers
{
  "name": "My Provider",
  "type": "ollama",
  "endpoint": "http://localhost:11434",
  "model": "llama3.2",
  "api_key": "optional"
}

# Get provider details
GET /api/v1/providers/{id}

# Get provider models
GET /api/v1/providers/{id}/models

# Delete provider
DELETE /api/v1/providers/{id}
```

### Agent Management ✅
```bash
# List all agents
GET /api/v1/agents

# Spawn new agent
POST /api/v1/agents
{
  "persona_name": "default/web-designer",
  "project_id": "agenticorp-self",
  "provider_id": "mock-local"
}

# Get agent details
GET /api/v1/agents/{id}

# Stop agent
DELETE /api/v1/agents/{id}

# Pause/resume agent
POST /api/v1/agents/{id}/pause
POST /api/v1/agents/{id}/resume
```

### CEO REPL (Direct Agent Invocation) ✅
```bash
# Ask the CEO agent a question
POST /api/v1/repl
{
  "message": "Review the web UI and provide recommendations",
  "timeout_sec": 180
}

# Returns a response from the CEO persona
```

### Work Submission (Non-Bead Prompts) ✅
```bash
# Submit ad-hoc work to an agent
POST /api/v1/work
{
  "prompt": "Analyze the current web UI design",
  "agent_id": "optional-specific-agent",
  "project_id": "agenticorp-self"
}
```

### Personas ✅
```bash
# List all personas
GET /api/v1/personas

# Get specific persona
GET /api/v1/personas/default/web-designer

# Update persona (live editing)
PUT /api/v1/personas/{name}
```

### Projects ✅
```bash
# List projects
GET /api/v1/projects

# Get project details
GET /api/v1/projects/agenticorp-self

# Create project
POST /api/v1/projects

# Sync git
POST /api/v1/projects/git/sync
POST /api/v1/projects/git/commit
POST /api/v1/projects/git/push
```

### Beads (Work Items) ✅
```bash
# List beads
GET /api/v1/beads?project_id=agenticorp-self&status=open

# Create bead
POST /api/v1/beads

# Update bead
PATCH /api/v1/beads/{id}

# Claim bead (assign to agent)
POST /api/v1/beads/{id}/claim
```

### Decisions ✅
```bash
# List decision beads
GET /api/v1/decisions

# Make a decision
POST /api/v1/decisions/{id}/decide
```

### System Status ✅
```bash
# Get overall system status
GET /api/v1/system/status

# Health check
GET /api/v1/health
```

### Analytics ✅
```bash
# Get usage logs
GET /api/v1/analytics/logs

# Get statistics
GET /api/v1/analytics/stats

# Get cost report
GET /api/v1/analytics/costs

# Export data
GET /api/v1/analytics/export
```

---

## Web UI Implementation

The web UI is a **single-page application** that uses **100% API calls** for all functionality:

### UI Sections (All API-Backed)
1. **Project Viewer** - `/api/v1/projects`, `/api/v1/beads`, `/api/v1/agents`
2. **Kanban Board** - `/api/v1/beads` with filtering
3. **Providers** - `/api/v1/providers` (registration, health checks)
4. **Agents** - `/api/v1/agents` (spawning, monitoring)
5. **Personas** - `/api/v1/personas` (viewing, live editing)
6. **Projects** - `/api/v1/projects` (CRUD operations)
7. **Decisions** - `/api/v1/decisions` (decision making)
8. **CEO REPL** - `/api/v1/repl` (direct agent queries) ⭐
9. **Streaming Test** - `/api/v1/chat/completions/stream`
10. **Users** - `/api/v1/auth/users` (user management)
11. **Analytics** - `/api/v1/analytics/*` (dashboards, reports)

### Key Features
- **Real-time updates** via Server-Sent Events (EventSource)
- **Streaming responses** for chat completions
- **Auto-refresh** every 5 seconds
- **JWT authentication** with token refresh
- **Modal dialogs** for forms
- **Toast notifications** for feedback

---

## How to Invoke Agents via curl

### 1. Check Provider Status First

```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.token')

# Check providers
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/providers | jq .
```

### 2. Register a Real Provider (if needed)

```bash
# Example: Register Ollama
curl -X POST http://localhost:8080/api/v1/providers \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ollama Local",
    "type": "ollama",
    "endpoint": "http://localhost:11434",
    "model": "llama3.2"
  }' | jq .
```

### 3. Invoke CEO Agent via REPL

```bash
# Ask CEO to review the codebase
curl -X POST http://localhost:8080/api/v1/repl \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Please review the AgentiCorp web UI codebase and provide recommendations for improvements. Focus on accessibility, usability, and modern web standards.",
    "timeout_sec": 300
  }' | jq .
```

### 4. Ask Web Designer to Review

**Option A: Via existing agent (currently paused)**

First, we need to resume or create a new web designer agent. Since we're at capacity, let's use the REPL to route to the appropriate persona:

```bash
# The CEO can delegate to web designer internally
curl -X POST http://localhost:8080/api/v1/repl \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Delegate to the web-designer persona: Please review our web UI at /web/static/ and provide a detailed UX/UI assessment with specific recommendations.",
    "timeout_sec": 300
  }' | jq .
```

**Option B: Remove an agent and spawn web designer**

```bash
# Stop an agent to free up capacity
curl -X DELETE http://localhost:8080/api/v1/agents/agent-1769059130-CFO%20(Default) \
  -H "Authorization: Bearer $TOKEN"

# Spawn web designer with a real provider (once registered)
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "persona_name": "default/web-designer",
    "project_id": "agenticorp-self",
    "provider_id": "your-real-provider-id"
  }' | jq .
```

### 5. Submit Direct Work

```bash
# Submit work directly without creating a bead
curl -X POST http://localhost:8080/api/v1/work \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Review web/static/index.html and web/static/js/app.js. Evaluate accessibility, performance, and UX. Provide 5 specific, actionable recommendations.",
    "project_id": "agenticorp-self"
  }' | jq .
```

---

## Available Personas

### Executive/Management
- `default/ceo` - Strategic decisions, high-level planning
- `default/cfo` - Financial analysis, budgeting
- `default/product-manager` - Product strategy, roadmaps
- `default/project-manager` - Task coordination, timelines
- `default/engineering-manager` - Team coordination, technical leadership
- `default/decision-maker` - Decision resolution

### Engineering
- `default/code-reviewer` - Code quality, best practices
- `default/devops-engineer` - Infrastructure, deployment
- `default/qa-engineer` - Testing, quality assurance

### Design & Content
- **`default/web-designer`** ⭐ - UI/UX design, accessibility
- **`default/web-designer-engineer`** ⭐ - Full-stack design+implementation
- `default/documentation-manager` - Documentation quality
- `default/public-relations-manager` - Communication, messaging

### Operations
- `default/housekeeping-bot` - Maintenance, cleanup

---

## Current Limitations

### 1. Provider Required
- Mock provider doesn't execute real work
- Need to register: Ollama, OpenAI, or other LLM provider
- Health checks run automatically on registration

### 2. Agent Capacity
- Currently at 10/10 agents (maximum)
- Need to stop agents to spawn new ones
- OR increase `agents.max_concurrent` in config.yaml

### 3. Agents Paused
- All agents paused due to no viable provider
- Will auto-resume when real provider is active

---

## Recommended Actions

### Immediate: Check Provider Health
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.token')

curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/providers | \
  jq '.[] | {id, name, status, last_heartbeat_at, last_heartbeat_error}'
```

### Next: Register Real Provider
```bash
# If you have Ollama running locally
curl -X POST http://localhost:8080/api/v1/providers \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "ollama-local",
    "name": "Ollama Local LLM",
    "type": "ollama",
    "endpoint": "http://host.docker.internal:11434",
    "model": "llama3.2",
    "description": "Local Ollama instance"
  }' | jq .
```

### Then: Invoke Web Designer Review
```bash
# Method 1: Via CEO REPL (delegates internally)
curl -X POST http://localhost:8080/api/v1/repl \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Have the web-designer persona conduct a comprehensive review of our web UI (web/static/). Focus on: 1) Accessibility compliance, 2) Mobile responsiveness, 3) Visual hierarchy, 4) User interaction patterns, 5) Performance considerations. Provide specific, actionable recommendations.",
    "timeout_sec": 300
  }' | jq .

# Method 2: Direct work submission
curl -X POST http://localhost:8080/api/v1/work \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "As a web designer, review the AgentiCorp UI and provide a detailed UX assessment.",
    "project_id": "agenticorp-self",
    "persona_hint": "web-designer"
  }' | jq .
```

---

## Summary

**Everything you asked about is possible!**

1. ✅ **UI is 100% API-driven** - Every feature uses REST endpoints
2. ✅ **CEO/CLI callable via curl** - `/api/v1/repl` endpoint
3. ✅ **Web designer can review codebase** - Via REPL or direct work submission
4. ✅ **Provider health check available** - `/api/v1/providers` shows status

**Blockers:**
- Need to register a real LLM provider (mock provider insufficient)
- Agents currently paused waiting for viable provider

**Next Steps:**
1. Register an LLM provider (Ollama, OpenAI, etc.)
2. Verify provider status shows "active"
3. Invoke web-designer via API
4. Review results

---

## Example Complete Workflow

```bash
#!/bin/bash
# Complete workflow to review web UI via API

# 1. Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.token')

echo "Logged in: $TOKEN"

# 2. Check providers
echo "\n=== Checking providers ==="
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/providers | jq '.[] | {id, name, status}'

# 3. Check agents
echo "\n=== Checking agents ==="
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/agents | jq '.[] | {id, persona_name, status}'

# 4. If provider is active, ask web designer to review
echo "\n=== Requesting web UI review ==="
curl -X POST http://localhost:8080/api/v1/repl \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Web designer: Please review web/static/index.html, web/static/css/style.css, and web/static/js/app.js. Provide 5 specific UX improvements.",
    "timeout_sec": 300
  }' | jq .
```

---

**Generated:** 2026-01-22 at 13:28 UTC  
**System:** AgentiCorp v1.0.0  
**API Version:** v1  
**Authentication:** JWT Bearer Token
