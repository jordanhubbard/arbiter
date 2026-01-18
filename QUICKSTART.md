# Quick Start Guide

Get Arbiter up and running in 5 minutes.

## Prerequisites

- Go 1.24+ installed
- Git
- (Optional) [bd (beads)](https://github.com/steveyegge/beads) for full task tracking

## Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/jordanhubbard/arbiter.git
cd arbiter

# Set up development environment
make dev-setup

# Build and run
make run
```

### Option 2: Pre-built Binary

Download the latest release from [GitHub Releases](https://github.com/jordanhubbard/arbiter/releases).

```bash
# Extract and run
./arbiter -config config.yaml
```

## First Steps

### 1. Start the Server

```bash
# With default configuration
./arbiter

# With custom config
./arbiter -config /path/to/config.yaml
```

The server will start on:
- HTTP: http://localhost:8080
- Web UI: http://localhost:8080

### 2. Access the Web UI

Open your browser to http://localhost:8080

You'll see:
- **Kanban Board**: Work items organized by status
- **Agents**: Currently running agents
- **Decisions**: Decision beads requiring resolution
- **Projects**: Configured projects
- **Personas**: Available agent personalities

### 3. Create a Project

**Via Web UI**: Coming soon in next release

**Via API**:
```bash
curl -X POST http://localhost:8080/api/v1/projects \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Project",
    "git_repo": "/path/to/repo",
    "branch": "main",
    "beads_path": ".beads",
    "context": {
      "build_command": "make build",
      "test_command": "make test"
    }
  }'
```

**Via Config File**:
```yaml
# config.yaml
projects:
  - id: my-project
    name: My Project
    git_repo: /path/to/repo
    branch: main
    beads_path: .beads
    context:
      build_command: "make build"
      test_command: "make test"
```

### 4. Spawn Your First Agent

**Via Web UI**:
1. Click "Spawn New Agent" button
2. Select a persona (e.g., code-reviewer)
3. Select a project
4. Click "Spawn Agent"

**Via API**:
```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-reviewer",
    "persona_name": "examples/code-reviewer",
    "project_id": "my-project"
  }'
```

### 5. Create a Bead (Work Item)

**Via bd CLI** (if installed):
```bash
cd /path/to/repo
bd init  # Initialize beads
bd create "Review authentication logic" -p 1
```

**Via API**:
```bash
curl -X POST http://localhost:8080/api/v1/beads \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Review authentication logic",
    "description": "Check for security vulnerabilities",
    "priority": 1,
    "project_id": "my-project",
    "type": "task"
  }'
```

### 6. Watch Agents Work

Agents will:
1. Query available beads
2. Claim beads that match their capabilities
3. Request file locks before editing
4. Make autonomous decisions (within their authority)
5. File decision beads when they need help
6. Update bead status as they work

Monitor in real-time via the Web UI!

## Example Workflows

### Autonomous Code Review

1. **Spawn a code-reviewer agent**
2. **Create review beads** for files/modules
3. **Agent automatically**:
   - Claims beads
   - Reviews code
   - Fixes obvious bugs
   - Files decision beads for API changes
   - Escalates security issues to P0

### Continuous Maintenance

1. **Spawn a housekeeping-bot agent**
2. **It runs continuously**:
   - Checks dependencies daily
   - Updates patch versions automatically
   - Files decision beads for major upgrades
   - Keeps documentation up to date

### Multi-Agent Feature Development

1. **Spawn multiple agents** on different branches
2. **Or same branch** with file coordination
3. **Arbiter coordinates** file locks
4. **Decision maker agent** resolves conflicts
5. **Work flows autonomously**

## Configuration

### Basic Configuration

Create `config.yaml`:

```yaml
server:
  http_port: 8080
  enable_http: true

beads:
  bd_path: bd  # Path to bd executable
  auto_sync: true

agents:
  max_concurrent: 10
  default_persona_path: ./personas

projects:
  - id: my-project
    name: My Project
    git_repo: /path/to/repo
    branch: main
```

### Enable HTTPS

```yaml
server:
  enable_https: true
  https_port: 8443
  tls_cert_file: /path/to/cert.pem
  tls_key_file: /path/to/key.pem
```

### Enable Authentication

```yaml
security:
  enable_auth: true
  api_keys:
    - "your-secret-api-key-here"
```

Then use API key in requests:
```bash
curl -H "X-API-Key: your-secret-api-key-here" \
  http://localhost:8080/api/v1/agents
```

## Available Personas

### code-reviewer
- **Autonomy**: Semi-autonomous
- **Focus**: Security, correctness, maintainability
- **Best for**: Code review automation

### decision-maker
- **Autonomy**: Full (for non-P0)
- **Focus**: Resolving decision points
- **Best for**: Unblocking other agents

### housekeeping-bot
- **Autonomy**: Full (for maintenance)
- **Focus**: Dependencies, docs, cleanup
- **Best for**: Continuous maintenance

## API Examples

### List All Beads
```bash
curl http://localhost:8080/api/v1/beads
```

### Get Work Graph
```bash
curl http://localhost:8080/api/v1/work-graph?project_id=my-project
```

### Claim a Bead
```bash
curl -X POST http://localhost:8080/api/v1/beads/bd-abc123/claim \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "agent-123"}'
```

### Make a Decision (as user)
```bash
curl -X POST http://localhost:8080/api/v1/decisions/bd-dec-456/decide \
  -H "Content-Type: application/json" \
  -d '{
    "decider_id": "user-jordan",
    "decision": "APPROVE",
    "rationale": "Change is well-tested and documented"
  }'
```

### Request File Lock
```bash
curl -X POST http://localhost:8080/api/v1/file-locks \
  -H "Content-Type: application/json" \
  -d '{
    "file_path": "src/auth.go",
    "project_id": "my-project",
    "agent_id": "agent-123",
    "bead_id": "bd-abc123"
  }'
```

## Troubleshooting

### Port Already in Use

Change the port in config.yaml:
```yaml
server:
  http_port: 8081
```

### Beads CLI Not Found

Either:
1. Install bd: https://github.com/steveyegge/beads
2. Or specify path in config:
```yaml
beads:
  bd_path: /path/to/bd
```

### Agent Not Working

Check:
1. Persona files exist
2. Project is configured correctly
3. Git repo path is correct
4. Agent has access to files

View logs in terminal where arbiter is running.

### Web UI Not Loading

1. Check web/static directory exists
2. Verify config:
```yaml
web_ui:
  enabled: true
  static_path: ./web/static
```

## Next Steps

- **Read the [README](README.md)** for architecture details
- **Check [CONTRIBUTING.md](CONTRIBUTING.md)** to contribute
- **Explore the [API docs](api/openapi.yaml)** for full API reference
- **Create custom personas** for your use case
- **Set up HTTPS** for production deployment

## Getting Help

- **Issues**: https://github.com/jordanhubbard/arbiter/issues
- **Discussions**: https://github.com/jordanhubbard/arbiter/discussions
- **Documentation**: https://github.com/jordanhubbard/arbiter/wiki

## Quick Reference

```bash
# Build
make build

# Run
make run

# Test
make test

# Clean
make clean

# Format code
make fmt

# See all commands
make help
```

Happy orchestrating! 🤖
