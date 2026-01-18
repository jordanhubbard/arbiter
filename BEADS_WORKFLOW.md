# Beads Workflow Guide

This guide explains how to use the beads system for tracking work in the Arbiter project.

## What are Beads?

Beads are work items that represent tasks, decisions, features, or bugs. The beads system helps track work progress, dependencies, and enables multi-agent coordination.

## Filing a Bead for Your Work

**IMPORTANT**: All work done on this repository should have a corresponding bead filed. This enables proper tracking and coordination.

### Manual Method (when bd CLI is not available)

1. Create a new YAML file in `.beads/beads/` with a unique ID:
   ```bash
   # Example: bd-002-add-new-feature.yaml
   ```

2. Use this template:
   ```yaml
   id: bd-XXX
   type: task  # or: decision, epic, bug
   title: Brief description of work
   description: |
     Detailed description of what needs to be done
   status: open  # open, in_progress, blocked, closed
   priority: 2  # 0=P0/critical, 1=P1/high, 2=P2/medium, 3=P3/low
   project_id: arbiter
   assigned_to: your-name-or-agent-id
   blocked_by: []
   blocks: []
   parent: null
   children: []
   tags: [relevant, tags]
   created_at: YYYY-MM-DDTHH:MM:SSZ
   updated_at: YYYY-MM-DDTHH:MM:SSZ
   context:
     branch: your-branch-name
     issue: related-issue-if-any
   ```

3. Update the bead as work progresses:
   - Change `status` field
   - Update `updated_at` timestamp
   - Add context or notes

4. When work is complete:
   - Set `status: closed`
   - Add `closed_at: YYYY-MM-DDTHH:MM:SSZ`
   - Move file to `.beads/closed/` directory

### Using the bd CLI (when available)

```bash
# Initialize beads in a repository
bd init

# Create a new bead
bd create "Title of work" -p 2 -d "Description"

# List beads
bd list

# Update a bead
bd update bd-XXX --status in_progress

# Close a bead
bd close bd-XXX
```

### Using the Arbiter API

```bash
# Create a bead
curl -X POST http://localhost:8080/api/v1/beads \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Title of work",
    "description": "Description",
    "priority": 2,
    "project_id": "arbiter",
    "type": "task"
  }'

# List beads
curl http://localhost:8080/api/v1/beads?project_id=arbiter

# Update a bead
curl -X PATCH http://localhost:8080/api/v1/beads/bd-XXX \
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

## Best Practices

1. **File beads early**: Create a bead when you start work, not when you're done
2. **Be descriptive**: Good titles and descriptions help others understand the work
3. **Update regularly**: Keep the status current
4. **Link dependencies**: Use `blocks` and `blocked_by` to show relationships
5. **Use appropriate priority**: Help others understand urgency
6. **Add context**: Branch names, related issues, or other relevant info

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
7. Set status to "closed"
8. Add closed_at timestamp
9. Move to .beads/closed/ directory
```

## Decision Beads

When work requires a decision:

1. Create a decision bead in `.beads/decisions/`
2. Include:
   - Question to be answered
   - Available options
   - Your recommendation (if any)
   - Priority level
3. Set parent bead status to "blocked"
4. Wait for decision resolution
5. Once decided, update parent bead and continue work

## For AI Agents

When you are assigned work:

1. Check `.beads/beads/` for open beads
2. Claim a bead by setting `assigned_to` to your agent ID
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

See the main README.md or QUICKSTART.md for more information about the Arbiter system.
