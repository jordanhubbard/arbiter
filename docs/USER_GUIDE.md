# Arbiter User Guide

This guide helps new users run Arbiter, register projects, and work with agents and beads.

## Getting Started

### Prerequisites

- Docker 20.10+
- Docker Compose 1.29+
- Go 1.24+ (optional for local development)

### Start Arbiter

```bash
docker-compose up -d
```

Once running, Arbiter serves the API on `:8080` and the Temporal UI on `:8088`.

## Project Registration

Projects are registered in `config.yaml` under `projects:`. Required fields:

- `id`
- `name`
- `git_repo`
- `branch`
- `beads_path`

Optional fields:

- `is_perpetual` (never closes)
- `is_sticky` (auto-registered on startup)
- `context` (build/test/lint commands and other agent guidance)

Example:

```yaml
projects:
  - id: arbiter
    name: Arbiter
    git_repo: https://github.com/jordanhubbard/arbiter
    branch: main
    beads_path: .beads
    is_perpetual: true
    is_sticky: true
    context:
      build_command: "make build"
      test_command: "make test"
```

Arbiter loads beads from each project’s `beads_path` and uses them to build the work graph.

## Personas and Agents

Default personas live under `personas/default/`. The system persona(s) live under
`personas/arbiter/`.

Agents are created from personas and attached to projects. The Project Viewer UI
shows agent assignments and bead progress in real time.

## Beads

Beads are YAML work items stored in `.beads/beads/` for each project. They drive
the work graph and include metadata such as priority, status, and dependencies.

Key fields:

- `id`, `type`, `title`, `description`
- `status`, `priority`, `project_id`
- `assigned_to`, `blocked_by`, `blocks`, `parent`, `children`

## Operational Workflow

1. Register projects in `config.yaml`.
2. Start Arbiter (docker-compose or binary).
3. Confirm beads are loaded in the UI and API.
4. Assign agents to projects and monitor progress.
5. Use decisions/approvals for escalations (e.g., CEO workflow).

## Troubleshooting

- If beads fail to load, run `make lint-yaml` to validate YAML.
- If providers are missing, register them in the Providers UI and re-negotiate models.
- If no work is dispatched, check the Project Viewer for blocked beads or missing agents.
