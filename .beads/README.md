# Beads Directory

This directory contains the beads (work items) tracking system for the Arbiter project.

## Structure

- `beads/` - Active work items (tasks, features, bugs) in YAML format
- `decisions/` - Decision beads requiring resolution
- `closed/` - Completed beads (archived)
- `FIRST_RELEASE_BEADS.md` - Initial planning document for first release

## About Beads

Beads are work items in the Arbiter system. They can be:
- **Tasks**: Individual work items to be completed
- **Epics**: Large features broken down into multiple tasks
- **Decisions**: Decision points that need resolution

## Bead Format

Each bead is stored as a YAML file with the following structure:

```yaml
id: bd-<unique-id>
type: task|decision|epic
title: Short description
description: Detailed description
status: open|in_progress|blocked|closed
priority: 0-3 (0=P0/critical, 3=P3/low)
project_id: project identifier
assigned_to: agent or user ID (optional)
blocked_by: [list of bead IDs]
blocks: [list of bead IDs]
parent: parent bead ID (optional)
children: [list of child bead IDs]
tags: [list of tags]
created_at: ISO timestamp
updated_at: ISO timestamp
closed_at: ISO timestamp (optional)
```

## Arbiter as Its Own First Project

The Arbiter project is registered with itself as the first project. This demonstrates the core functionality and provides a real-world use case for the system.

Project ID: `arbiter`
Git Repo: `/home/runner/work/arbiter/arbiter`
Branch: `copilot/register-arbiter-project` (being merged to main)
Beads Path: `.beads`

## Managing Beads

Beads can be managed through:
1. The Arbiter web UI
2. The Arbiter API (`/api/v1/beads`)
3. The `bd` CLI tool (if installed)
4. Direct file manipulation in this directory

## Current Work

All active work should have a corresponding bead in the `beads/` directory. See FIRST_RELEASE_BEADS.md for the initial planning document.

