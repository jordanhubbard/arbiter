# Arbiter Project Beads

This directory contains the work tracking data for the Arbiter project itself.

## Files

- **FIRST_RELEASE_BEADS.md**: Initial set of beads (work items) for the first release, as determined by the project manager persona

## About Beads

Beads are work items in the Arbiter system. They can be:
- **Tasks**: Individual work items to be completed
- **Epics**: Large features broken down into multiple tasks
- **Decisions**: Decision points that need resolution

Each bead has:
- ID (e.g., BD-001)
- Title
- Description
- Priority (P0-P3)
- Status (open, in_progress, blocked, closed)
- Dependencies (what blocks what)
- Assignment (which agent is working on it)

## Arbiter as Its Own First Project

The Arbiter project is registered with itself as the first project. This demonstrates the core functionality and provides a real-world use case for the system.

Project ID: `arbiter`
Git Repo: `/home/runner/work/arbiter/arbiter`
Branch: `copilot/register-arbiter-project`
Beads Path: `.beads`

## Managing Beads

Beads can be managed through:
1. The Arbiter web UI
2. The Arbiter API
3. The bd CLI tool (if installed)
4. Direct file manipulation (for simple cases)

See FIRST_RELEASE_BEADS.md for the initial work breakdown.
