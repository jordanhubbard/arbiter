# Beads Directory

This directory contains the beads (work items) tracking system for the Arbiter project.

## Structure

- `beads/` - Active work items (tasks, features, bugs)
- `decisions/` - Decision beads requiring resolution
- `closed/` - Completed beads (archived)

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

## Usage

Beads can be managed through:
1. The Arbiter API (`/api/v1/beads`)
2. Direct file manipulation in this directory
3. The `bd` CLI tool (if installed)

## Current Work

All active work should have a corresponding bead in the `beads/` directory.
