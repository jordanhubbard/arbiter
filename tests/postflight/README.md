# AgentiCorp Post-Flight API Tests

Automated API validation tests that run after container startup to ensure all endpoints are responding correctly.

## Overview

The post-flight test suite validates:
- All documented REST API endpoints
- HTTP response codes
- JSON response structure
- Service availability and readiness

## Running Tests

### Via Makefile (Recommended)

```bash
# Test against default localhost:8080
make test-api

# Test against custom URL
BASE_URL=http://myserver:8080 make test-api

# Verbose output
VERBOSE=1 make test-api
```

### Direct Execution

```bash
# Default settings
./tests/postflight/api_test.sh

# Custom configuration
BASE_URL=http://myserver:8080 TIMEOUT=10 VERBOSE=1 ./tests/postflight/api_test.sh
```

## Configuration

Environment variables:
- `BASE_URL` - Base URL of AgentiCorp service (default: `http://localhost:8080`)
- `TIMEOUT` - Request timeout in seconds (default: `5`)
- `VERBOSE` - Verbose output (default: `0`, set to `1` for details)
- `AUTH_USER` - Username for authenticated endpoints (default: `admin`)
- `AUTH_PASSWORD` - Password for authenticated endpoints (default: `admin`)

## Requirements

- bash
- curl
- jq (JSON processor)

## Test Coverage

The test suite validates these endpoints:

### Health & Status
- `GET /api/v1/health` - Health check
- `GET /api/v1/system/status` - System status

### Providers
- `GET /api/v1/providers` - List all providers

### Projects
- `GET /api/v1/projects` - List all projects
- `GET /api/v1/projects/{id}` - Get project by ID
- `GET /api/v1/org-charts/{projectId}` - Get project org chart

### Agents
- `GET /api/v1/agents` - List all agents

### Beads (Work Items)
- `GET /api/v1/beads` - List all beads
- `GET /api/v1/beads/{id}` - Get bead by ID

### Decisions
- `GET /api/v1/decisions` - List all decisions

### Personas
- `GET /api/v1/personas` - List all personas

### Events
- `GET /api/v1/events/stats` - Event statistics
- `GET /api/v1/events/stream` - SSE event stream (connection test only)

### Work Graph
- `GET /api/v1/work-graph?project_id={id}` - Dependency graph

## Exit Codes

- `0` - All tests passed
- `1` - One or more tests failed

## Integration

### CI/CD Pipeline

```yaml
# Example GitHub Actions step
- name: Run Post-Flight Tests
  run: |
    docker compose up -d
    make test-api
```

### Docker Compose Healthcheck

```yaml
services:
  agenticorp:
    healthcheck:
      test: ["CMD", "/app/tests/postflight/api_test.sh"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Troubleshooting

### Service not available
- Ensure AgentiCorp is running: `docker compose ps`
- Check logs: `docker compose logs agenticorp`
- Verify port binding: `curl http://localhost:8080/api/v1/health`

### jq not found
```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# Alpine
apk add jq
```

### Tests timeout
- Increase timeout: `TIMEOUT=30 make test-api`
- Check network connectivity
- Verify service performance

### Auth failures (401/403)
- Confirm credentials: `AUTH_USER` / `AUTH_PASSWORD`
- Default admin credentials are `admin` / `admin` unless changed

## Sticky Test Behavior

This test suite is marked as a **sticky test** in bead `bd-038-postflight-api-tests.yaml`.
It should be run:
- After every container startup
- After any API endpoint changes
- Before declaring a deployment ready
- As part of CI/CD pipeline

## Development

To add new endpoint tests:

1. Add test to `api_test.sh` using `test_endpoint` function:
   ```bash
   test_endpoint "GET" "/api/v1/new-endpoint" "200" "" "Description"
   ```

2. Update this README with the new endpoint
3. Update test count in bead acceptance criteria

## License

Part of the AgentiCorp project.
