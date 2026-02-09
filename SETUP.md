# Loom Setup Guide

Everything you need to get Loom running and create your first project.

## Prerequisites

- Docker (20.10+)
- Docker Compose (1.29+)
- Go 1.25+ (for local development only)
- Make (optional, for convenience commands)

## Running with Docker (Recommended)

The Docker setup includes:
- Loom application server (port 8080)
- Temporal server (port 7233)
- Temporal UI (port 8088)
- PostgreSQL database for Temporal

```bash
# Build and run all services
docker compose up -d

# Verify everything is healthy
docker compose ps

# View logs
docker compose logs -f loom

# Stop all services
docker compose down
```

### Using Make Commands

```bash
# Build and run
make docker-run

# Build Docker image
make docker-build

# Stop services
make docker-stop

# Clean Docker resources
make docker-clean
```

## Connecting to the UI

Once the services are running:

- **Loom Web UI**: http://localhost:8080
- **Temporal UI**: http://localhost:8088 — view workflow executions, inspect history, monitor active workflows, debug failures

## Configuration

Structural configuration is managed via `config.yaml` (server, temporal, agents, projects). Secrets like API keys are **never** stored in `config.yaml` — see [Registering Providers](#registering-providers) below.

```yaml
server:
  http_port: 8081
  enable_http: true

temporal:
  host: localhost:7233
  namespace: default
  task_queue: loom-tasks

agents:
  max_concurrent: 6
  default_persona_path: ./personas
  heartbeat_interval: 30s
```

Environment variables in `config.yaml` are expanded automatically — `${MY_VAR}` is replaced with the value of `MY_VAR` from the environment.

## Registering Providers

Providers (LLM backends) are registered via the Loom API, **not** in `config.yaml`. This keeps API keys out of version control and allows per-deployment configuration.

### Quick Start: Register a Provider

```bash
curl -X POST http://localhost:8081/api/v1/providers \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"my-provider\",\"name\":\"My Provider\",\"type\":\"openai\",\"endpoint\":\"http://localhost:8000/v1\",\"model\":\"my-model\",\"api_key\":\"$MY_API_KEY\"}"
```

API keys are stored in Loom's encrypted vault and persist across restarts.

### Using bootstrap.local

For repeatable setup, create a `bootstrap.local` script (gitignored) that registers all your providers:

```bash
cp bootstrap.local.example bootstrap.local
chmod +x bootstrap.local
# Edit bootstrap.local with your providers and API keys
./bootstrap.local
```

The file uses a `register_provider` helper function:

```bash
#!/bin/bash
LOOM_URL="${LOOM_URL:-http://localhost:8081}"

register_provider() {
  local id="$1" name="$2" type="$3" endpoint="$4" model="$5" api_key="${6:-}"
  # ... builds JSON and calls POST /api/v1/providers
}

# Local GPU (no API key)
register_provider "local-gpu" "Local GPU" "local" \
  "http://gpu-server:8000/v1" "nvidia/Nemotron-30B"

# Cloud provider (API key from environment)
register_provider "nvidia-cloud" "NVIDIA Cloud" "openai" \
  "https://inference-api.nvidia.com/v1" "nvidia/openai/gpt-oss-20b" \
  "$NVIDIA_API_KEY"
```

**When to use `bootstrap.local`:**
- After a fresh install or database wipe
- When adding a new provider to your deployment
- To document your provider setup in a reproducible way

**Environment variables:** Set API keys in your shell environment (`~/.zshenv`, `~/.bashrc`, or `.env`) and reference them with `$VAR_NAME` in `bootstrap.local`. They are passed to `curl` via shell expansion — they never touch disk in plaintext.

Providers persist in the database — you only need to run `bootstrap.local` once per fresh deployment.

## Bootstrapping Your First Project

Projects are registered via `config.yaml` under `projects:` and persisted in the database.

Required fields:
- `id`, `name`, `git_repo`, `branch`, `beads_path`

Optional fields:
- `is_perpetual` (never closes)
- `context` (recommended: build/test/lint commands and other agent-relevant context)

Example:

```yaml
projects:
  - id: loom
    name: Loom
    git_repo: https://github.com/jordanhubbard/loom
    branch: main
    beads_path: .beads
    is_perpetual: true
    context:
      test: go test ./...
      vet: go vet ./...
```

Loom "dogfoods" itself by registering this repo as a project and loading beads from the project's `.beads/` directory.

## Local Development

### Building Locally

```bash
# Install dependencies
go mod download

# Build the binary
go build -o loom ./cmd/loom

# Run the application
./loom
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/temporal/...
```

### Development with Temporal

For local development with Temporal:

1. Start Temporal server:
```bash
docker compose up -d temporal temporal-postgresql temporal-ui
```

2. Build and run loom locally:
```bash
go build -o loom ./cmd/loom
./loom
```

3. Access Temporal UI:
```bash
open http://localhost:8088
```

## Monitoring

### Event Stream

Monitor real-time events:
```bash
# Watch all events
curl -N http://localhost:8080/api/v1/events/stream

# Monitor specific project
curl -N "http://localhost:8080/api/v1/events/stream?project_id=my-project"
```

### Logs

View service logs:
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f loom
docker compose logs -f temporal
```

## Troubleshooting

### Temporal Connection Issues

If loom can't connect to Temporal:

1. Check Temporal is running:
```bash
docker compose ps temporal
```

2. Check Temporal logs:
```bash
docker compose logs temporal
```

3. Verify connectivity:
```bash
docker exec loom nc -zv temporal 7233
```

### Workflow Not Starting

If workflows aren't starting:

1. Check worker is running:
```bash
docker compose logs loom | grep "Temporal worker"
```

2. Verify task queue in Temporal UI
3. Check workflow registration in logs

### Event Stream Not Working

If event stream endpoint returns errors:

1. Verify Temporal is enabled in config
2. Check event bus initialization:
```bash
docker compose logs loom | grep "event bus"
```
