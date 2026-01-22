# Persona-Based Routing in AgentiCorp

## Overview

AgentiCorp now supports intelligent persona-based routing, automatically matching work to the most appropriate agent based on hints in bead titles and descriptions.

## Features

### 1. Persona Hint Detection

The dispatcher automatically extracts persona hints from beads using multiple patterns:

**Supported Patterns:**
- `"Ask the <persona> to ..."` - Natural language format
- `"[persona] ..."` or `"[persona]: ..."` - Bracketed format  
- `"persona: message"` - Colon-separated format (CEO CLI style)
- `"For <persona>:"` or `"For <persona> agent:"` - Assignment format
- `"**FOR <persona> AGENT**"` - Markdown bold format

**Examples:**
```plaintext
"Ask the web designer to improve button accessibility"
→ Routes to web-designer agent

"[QA Engineer] Review test coverage for auth module"
→ Routes to qa-engineer agent

"web-designer: What are the top 3 UX improvements?"
→ Routes to web-designer agent
```

### 2. Fuzzy Matching Algorithm

The matcher uses a three-pass approach to find the best agent:

1. **Exact Match**: Compares hint with `PersonaName` (without `default/` prefix)
2. **Partial Match**: Checks if hint is contained in persona name or vice versa
3. **Role Match**: Checks if hint matches the agent's `Role` field

This allows flexible matching:
- `"web designer"` → matches `default/web-designer`
- `"qa"` → matches `default/qa-engineer`
- `"manager"` → matches `default/project-manager`

### 3. Fallback Behavior

If a persona hint is detected but no matching agent is available:
- **P1/P2 beads**: Wait for the right agent rather than misassigning
- **P0 beads**: (Future) May override and assign to any available agent

If no persona hint is detected:
- Assigns to any idle agent with an active provider

### 4. CEO REPL Auto-Bead Creation

All CEO REPL queries now automatically create P0 beads:

**Benefits:**
- **State Preservation**: No queries are lost
- **Full Context**: Response, provider, tokens all stored
- **Traceability**: Each CEO interaction tracked with unique bead ID
- **Persona Routing**: Supports `"persona: message"` format

**API Response:**
```json
{
  "bead_id": "bd-017",
  "provider_id": "sparky-vllm",
  "provider_name": "Sparky vLLM",
  "model": "nvidia/NVIDIA-Nemotron-3-Nano-30B-A3B-FP8",
  "response": "...",
  "tokens_used": 245,
  "latency_ms": 1250
}
```

**Bead Context:**
```json
{
  "context": {
    "source": "ceo-repl",
    "created_by": "ceo",
    "response": "...",
    "provider_id": "sparky-vllm",
    "model": "...",
    "tokens_used": "245"
  },
  "status": "closed"
}
```

## Usage Examples

### Creating Persona-Targeted Beads

```bash
# Via API
curl -X POST http://localhost:8080/api/v1/beads \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Ask the web designer to improve button accessibility",
    "description": "Review and improve accessibility...",
    "type": "task",
    "priority": 1,
    "project_id": "agenticorp-self"
  }'

# Via bd CLI
bd create "Ask the devops engineer to optimize Docker build times" \
  -p 1 -d "Review Dockerfile and suggest multi-stage build improvements"
```

### CEO REPL with Persona Routing

```bash
# Direct persona assignment
curl -X POST http://localhost:8080/api/v1/repl \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "web-designer: What are the top 3 UX improvements needed?"}'

# Natural language (extracted automatically)
curl -X POST http://localhost:8080/api/v1/repl \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "Ask our QA engineer to review the test coverage"}'
```

### Tags for Persona Hints

Add persona-specific tags to beads:
```json
{
  "title": "Optimize database queries",
  "tags": ["performance", "for-devops-engineer", "p1"]
}
```

## Implementation Details

### PersonaMatcher Class

Location: `internal/dispatch/persona_matcher.go`

**Key Methods:**
- `ExtractPersonaHint(bead)` - Extracts persona from title/description/tags
- `FindAgentByPersonaHint(hint, agents)` - Finds best matching agent
- `normalizePersonaHint(hint)` - Normalizes hints to match persona names

### Dispatcher Integration

Location: `internal/dispatch/dispatcher.go`

```go
// Try persona-based routing first
personaHint := d.personaMatcher.ExtractPersonaHint(b)
if personaHint != "" {
    matchedAgent := d.personaMatcher.FindAgentByPersonaHint(personaHint, idleAgents)
    if matchedAgent != nil {
        ag = matchedAgent
        candidate = b
        break
    }
    // If persona hint found but no matching agent, continue to try other beads
    continue
}
```

### CEO REPL Enhancement

Location: `internal/agenticorp/agenticorp.go`

```go
func (a *AgentiCorp) RunReplQuery(ctx context.Context, message string) (*ReplResult, error) {
    // Extract persona hint and clean message
    personaHint, cleanMessage := extractPersonaFromMessage(message)
    
    // Create P0 bead for CEO query
    bead, err := a.beadsManager.CreateBead(beadTitle, cleanMessage, 
        models.BeadPriorityP0, "task", "agenticorp-self")
    
    // ... execute query ...
    
    // Update bead with response
    _ = a.beadsManager.UpdateBead(beadID, map[string]interface{}{
        "context": map[string]string{
            "source": "ceo-repl",
            "response": result.Response,
            // ... other fields
        },
        "status": models.BeadStatusClosed,
    })
    
    return result, nil
}
```

## Best Practices

1. **Use Natural Language**: Write bead titles naturally - the matcher will extract personas
2. **Be Specific**: Include role names when you want targeted routing
3. **Tag Appropriately**: Use tags like `for-<persona>` or `<persona>-only` for explicit routing
4. **Monitor Assignments**: Check that beads are assigned to appropriate agents
5. **CEO REPL**: Use `persona: message` format when you want direct routing

## Future Enhancements

- **Priority Override**: P0 beads may override persona matching to ensure urgent work gets done
- **Multi-Agent Beads**: Support beads that require multiple persona types
- **Learning**: Track successful persona matches to improve accuracy
- **Persona Preferences**: Allow personas to express interest in certain types of work
- **Load Balancing**: Consider agent workload when multiple agents match

## Related Documentation

- [Dispatcher System](./ARCHITECTURE.md#dispatcher)
- [Agent Management](./WORKER_SYSTEM.md)
- [Beads Workflow](./BEADS_WORKFLOW.md)
- [CEO REPL](./USER_GUIDE.md#ceo-repl)
