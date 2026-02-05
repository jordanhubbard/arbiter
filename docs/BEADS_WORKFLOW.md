# Beads Workflow Guide

This guide explains how to use the beads system for tracking work in the AgentiCorp project.

## What are Beads?

Beads are work items that represent tasks, decisions, features, or bugs. The beads system helps track work progress, dependencies, and enables multi-agent coordination.

## Filing a Bead for Your Work

**IMPORTANT**: All work done on this repository should have a corresponding bead filed. This enables proper tracking and coordination.

### Manual Method (when bd CLI is not available)

Beads are stored in `.beads/issues.jsonl` and are intended to be managed by the
`bd` CLI. If you must work without the CLI, prefer using the API endpoints
below so the JSONL remains consistent.

### Using the bd CLI (recommended)

```bash
# Initialize beads in a repository
bd init

# Create a new bead
bd create "Title of work" -p 2 -d "Description"

# List beads
bd list

# Update a bead
bd update ac-XXX --status in_progress

# Close a bead
bd close ac-XXX
```

### Using the AgentiCorp API

```bash
# Create a bead
curl -X POST http://localhost:8080/api/v1/beads \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Title of work",
    "description": "Description",
    "priority": 2,
    "project_id": "agenticorp",
    "type": "task"
  }'

# List beads
curl http://localhost:8080/api/v1/beads?project_id=agenticorp

# Update a bead
curl -X PATCH http://localhost:8080/api/v1/beads/ac-XXX \
  -H "Content-Type: application/json" \
  -d '{"status": "in_progress"}'
```

## Bead Types

- **task**: Regular work item (feature, bug fix, improvement)
- **decision**: Requires a decision before proceeding
- **epic**: Large work item composed of multiple smaller beads

## Priority Levels

- **P0 (0)**: Critical - needs immediate human attention
- **P1 (1)**: High priority
- **P2 (2)**: Medium priority (default)
- **P3 (3)**: Low priority

## Status Values

- **open**: Ready to work on
- **in_progress**: Currently being worked on
- **blocked**: Waiting on another bead or decision
- **closed**: Completed

## Dispatch and Redispatch

AgentiCorp automatically dispatches open beads to idle agents. The system tracks dispatch history in the bead's context:

- **Auto-redispatch**: Open and in-progress beads are automatically marked for redispatch, allowing them to be picked up by idle agents if work stalls
- **Dispatch hop limit**: If a bead is dispatched more than `max_hops` times (default: 20) without being closed, it is escalated to **P0 priority** and a **CEO decision bead** is created
- **Escalation prevents loops**: This ensures stuck or problematic work doesn't cycle indefinitely and gets human attention, while allowing complex investigations to proceed through multiple iterations

Dispatch history is stored in the bead's `context.dispatch_history` field and includes timestamps, agent IDs, and outcomes.

## Best Practices

1. **File beads early**: Create a bead when you start work, not when you're done
2. **Be descriptive**: Good titles and descriptions help others understand the work
3. **Update regularly**: Keep the status current
4. **Link dependencies**: Use `blocks` and `blocked_by` to show relationships
5. **Use appropriate priority**: Help others understand urgency
6. **Add context**: Branch names, related issues, or other relevant info
7. **Close beads when done**: Prevent unnecessary redispatch by closing completed work

## Example Workflow

```bash
# Starting new work
1. Create a bead for the work
2. Set status to "in_progress"
3. Add your name/agent ID to assigned_to
4. Create a branch (optional but recommended)

# During work
5. Update the bead status as needed
6. Add notes or context

# Completing work
7. Close the bead (`bd close ac-XXX`)
8. Sync the bead store if needed (`bd sync`)
```

## Decision Beads

When work requires a decision:

1. Create a decision bead with `bd create`:
   ```bash
   bd create "Decision title" --type decision -p 1 -d "Decision context"
   ```
2. Set parent bead status to "blocked"
3. Wait for decision resolution
4. Once decided, update parent bead and continue work

## For AI Agents

When you are assigned work:

1. Check `bd list open` for open beads
2. Claim a bead by setting `assigned_to` to your agent ID (or `bd update --claim`)
3. Update status to "in_progress"
4. Perform the work
5. File decision beads when uncertain (P0 for critical decisions)
6. Update the bead when complete
7. Release file locks and mark bead as closed

## Integration with Git

- Create one bead per PR/branch when possible
- Reference bead IDs in commit messages
- Link beads to GitHub issues when applicable

## Questions?

See the main README.md or QUICKSTART.md for more information about the AgentiCorp system.
