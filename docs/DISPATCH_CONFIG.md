# Dispatch System Configuration

The dispatch system manages automatic assignment of beads (work items) to available agents. This document covers configuration options and behavior.

## Overview

The dispatcher continuously monitors for open beads and assigns them to appropriate agents based on:
- Agent availability and capacity
- Agent persona type and capabilities
- Bead priority and type
- Dispatch hop count (anti-loop protection)

## Configuration

### Max Dispatch Hops

**Key:** `dispatch.max_hops`
**Default:** `20`
**Range:** `1-100` (recommended: `10-30`)

The maximum number of times a bead can be dispatched (assigned to an agent) before being escalated to CEO priority.

```yaml
dispatch:
  max_hops: 20  # Increased from 5 to support complex investigations
```

**Why this matters:**
- **Too low** (e.g., 5): Complex bug investigations or multi-step features may hit the limit prematurely
- **Too high** (e.g., 100): Genuinely stuck beads take longer to escalate to human attention
- **Recommended** (20): Allows thorough investigation while still catching infinite loops

### Dispatch Hop Behavior

When a bead reaches the `max_hops` limit:

1. **Escalation to P0**: Bead priority is elevated to P0 (critical)
2. **CEO Decision Bead**: A CEO decision bead is automatically created
3. **Context Preservation**: Full dispatch history is preserved for review
4. **Warning Logged**: System logs the escalation for monitoring

Example log output:
```
[Dispatcher] WARNING: Bead bead-abc-123 has been dispatched 20 times, escalating to CEO
```

## Dispatch Tracking

Each bead maintains a dispatch count in its context:

```json
{
  "bead_id": "bead-abc-123",
  "context": {
    "dispatch_count": 15,
    "last_dispatched_at": "2026-02-05T20:00:00Z",
    "dispatch_history": [
      {"agent_id": "agent-eng-1", "timestamp": "..."},
      {"agent_id": "agent-qa-1", "timestamp": "..."}
    ]
  }
}
```

## Common Scenarios

### 1. Complex Bug Investigation

A bug that requires multiple iterations of:
- Reading code → testing → analyzing → modifying → re-testing

**Before (max_hops: 5):**
```
Hop 1: Read error logs
Hop 2: Examine code
Hop 3: Add debug logging
Hop 4: Run tests
Hop 5: Analyze results
→ ESCALATED (investigation incomplete!)
```

**After (max_hops: 20):**
```
Hop 1-10: Deep investigation with multiple test iterations
Hop 11-15: Fix implementation with validation
Hop 16-20: Buffer for edge cases
→ Usually resolves before limit, or escalation is justified
```

### 2. Feature Development with Dependencies

Multi-step feature requiring coordination:

```
Hop 1-5: Design and architecture
Hop 6-12: Implementation
Hop 13-16: Testing
Hop 17-19: Documentation and review
Hop 20: Final validation
```

### 3. Infinite Loop Detection

A truly stuck bead that repeats the same action:

```
Hop 1-20: Same error, no progress
→ ESCALATED (correctly caught!)
```

**Note:** Smart loop detection (Epic 7, Task 2) will differentiate between productive iteration and stuck loops.

## Escalation Threshold

The system escalates beads at `maxHops-1` to give agents a final chance:

```go
if dispatchCount >= maxHops-1 {
    log.Warn("Bead approaching dispatch limit")
    // Agent notified, can request extension or close bead
}

if dispatchCount >= maxHops {
    escalateToCEO(bead)
}
```

## Monitoring

### Check Dispatch Statistics

```bash
# View beads approaching limit
bd list --status=in_progress | grep "dispatch_count"

# Monitor escalations
tail -f logs/dispatcher.log | grep "escalating to CEO"
```

### High Dispatch Count Beads

If you notice beads frequently hitting the limit:

1. **Review the work pattern**: Is the task genuinely complex or is the agent stuck?
2. **Check agent effectiveness**: Are agents making progress or spinning?
3. **Adjust max_hops**: Consider increasing if tasks are legitimately complex
4. **Improve agent personas**: Add more context or better instructions

## Performance Impact

The dispatch hop limit has minimal performance impact:
- **Memory**: ~8 bytes per dispatch record per bead
- **CPU**: O(1) integer comparison per dispatch cycle
- **Storage**: Dispatch history in bead context (JSON)

## Best Practices

### 1. Start Conservative

```yaml
# Development: lower limit to catch issues faster
dispatch:
  max_hops: 15

# Production: higher limit for complex work
dispatch:
  max_hops: 25
```

### 2. Monitor and Adjust

Track escalation rate:
```
Escalations per day: < 5 = good (limit appropriate)
Escalations per day: > 20 = too low (increase limit)
```

### 3. Log Analysis

Regular review of escalated beads:
```bash
# Find recently escalated beads
bd list --priority=0 | grep "dispatch_count.*exceeded"

# Analyze patterns
jq '.context.dispatch_history' .beads/issues.jsonl
```

### 4. Agent-Specific Limits

Future enhancement: Different limits per agent type
```yaml
# Proposal (not yet implemented)
dispatch:
  max_hops:
    default: 20
    qa-engineer: 25  # Testing requires more iterations
    code-reviewer: 15  # Reviews should be quicker
```

## Troubleshooting

### Bead Escalated Too Early

**Symptom**: Bead escalated to CEO but was making progress

**Solution**:
1. Increase `max_hops` in config.yaml
2. Restart agenticorp service
3. Re-open the escalated bead if appropriate

```yaml
dispatch:
  max_hops: 30  # Increased for complex work
```

### Bead Stuck in Loop

**Symptom**: Bead reaches max_hops with no progress

**Root Causes**:
- Agent missing context or instructions
- Circular dependency in code
- Invalid test or build configuration

**Solution**:
1. Review bead context and agent reasoning
2. Add missing context to bead description
3. Fix underlying issue (tests, build, etc.)
4. Manually close bead if unrecoverable

### Escalations Too Frequent

**Symptom**: Many beads escalating unnecessarily

**Analysis**:
```bash
# Count escalations
bd list --priority=0 --closed=false | wc -l

# Review common patterns
bd list --priority=0 | grep "dispatch_count"
```

**Solutions**:
- Increase max_hops if work is genuinely complex
- Improve agent personas and instructions
- Check for systemic issues (broken tests, missing deps)

## Related Configuration

### Agent Configuration

```yaml
agents:
  max_concurrent: 10  # Affects dispatch throughput
  heartbeat_interval: 30s  # Affects dispatch freshness
```

### Bead Configuration

```yaml
beads:
  auto_sync: true  # Ensures dispatch state is persisted
  sync_interval: 5m  # How often dispatch counts sync to git
```

## Implementation Details

### Dispatcher Code

Location: `internal/dispatch/dispatcher.go`

Key functions:
- `SetMaxDispatchHops(maxHops int)`: Configure limit
- `DispatchCycle()`: Main dispatch loop with hop checking
- `escalateBead(bead *models.Bead, reason string)`: Escalation logic

### Configuration Loading

Location: `pkg/config/config.go`

```go
type DispatchConfig struct {
    MaxHops int `yaml:"max_hops" json:"max_hops,omitempty"`
}

// Default configuration
func DefaultConfig() *Config {
    return &Config{
        Dispatch: DispatchConfig{
            MaxHops: 20,  // Updated from 5
        },
    }
}
```

## Version History

- **v1.0**: Initial implementation with max_hops = 5
- **v1.1** (Epic 7): Increased to max_hops = 20 for complex investigations
- **v1.2** (Planned): Smart loop detection to distinguish stuck from productive

## References

- [Auto Bug Dispatch Documentation](auto-bug-dispatch.md)
- [Beads Workflow](BEADS_WORKFLOW.md)
- [User Guide - Dispatch](USER_GUIDE.md#dispatch-system)
- [Dispatcher Source](../internal/dispatch/dispatcher.go)
