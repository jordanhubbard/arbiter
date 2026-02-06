# Loom Deployment Guide

**Last Updated:** February 2, 2026

This guide covers deploying Loom in various environments, from development to production.

## Table of Contents

1. [Quick Start (Local Development)](#quick-start-local-development)
2. [Docker Deployment](#docker-deployment)
3. [Production Deployment](#production-deployment)
4. [Configuration](#configuration)
5. [Monitoring & Maintenance](#monitoring--maintenance)
6. [Troubleshooting](#troubleshooting)

---

## Quick Start (Local Development)

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- Git
- Make (optional, for convenience commands)

### Local Development Setup

```bash
# Clone the repository
git clone https://github.com/jordanhubbard/Loom.git
cd Loom

# Copy example configuration
cp config.yaml.example config.yaml

# Edit configuration (set API keys, etc.)
vim config.yaml

# Start dependencies (Temporal, PostgreSQL)
docker compose up -d temporal postgres

# Run Loom locally
go run ./cmd/loom

# Or build and run
go build -o loom ./cmd/loom
./loom --config config.yaml
```

Loom will be available at `http://localhost:8080`

---

## Docker Deployment

### Docker Compose (Recommended for Development)

The easiest way to run Loom with all dependencies:

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f loom

# Stop all services
docker compose down

# Clean database and restart fresh
docker compose down -v
docker compose up -d
```

**Services started:**
- Loom API server (port 8080)
- Temporal server (port 7233)
- Temporal UI (port 8088)
- PostgreSQL (port 5432)

### Dockerfile Configuration

The included `Dockerfile` uses multi-stage builds for optimal image size:

```dockerfile
# Build stage: Compiles Go binary
FROM golang:1.25-alpine AS builder

# Runtime stage: Minimal Alpine image
FROM alpine:latest
```

**Build custom image:**

```bash
docker build -t loom:custom .
```

**Run custom image:**

```bash
docker run -d \
  --name loom \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/data:/app/data \
  loom:custom
```

---

## Production Deployment

### Environment Preparation

#### 1. System Requirements

**Minimum (Development/Testing):**
- 2 CPU cores
- 4GB RAM
- 20GB storage

**Recommended (Production):**
- 4+ CPU cores
- 8GB+ RAM
- 100GB SSD storage
- Load balancer for high availability

#### 2. Network Requirements

**Inbound Ports:**
- `8080` - Loom API server
- `8088` - Temporal UI (optional, can be internal-only)

**Outbound Access:**
- LLM provider APIs (OpenAI, Anthropic, etc.)
- Git repositories (if using GitOps)
- Webhook endpoints (if configured)

### Production Configuration

Create a production-ready `config.yaml`:

```yaml
server:
  http_port: 8080
  read_timeout: 30s
  write_timeout: 60s
  idle_timeout: 120s

database:
  type: sqlite
  path: /app/data/loom.db

security:
  enable_auth: true
  jwt_secret: ${JWT_SECRET}  # Use environment variable
  require_https: true
  allowed_origins:
    - https://yourdomain.com

agents:
  max_concurrent: 10
  heartbeat_interval: 30s
  file_lock_timeout: 15m

temporal:
  host: temporal:7233
  namespace: default

providers:
  - id: openai
    name: OpenAI
    type: openai
    endpoint: https://api.openai.com/v1
    enabled: true
    # API key loaded from environment

git:
  project_key_dir: /app/data/projects

# Enable analytics
analytics:
  enabled: true
  retention_days: 90
```

### Deployment Options

#### Option 1: Docker Compose (Single Server)

**Best for:** Small teams, staging environments

```bash
# Use production docker-compose file
docker compose -f docker-compose.prod.yml up -d
```

**Create `docker-compose.prod.yml`:**

```yaml
version: '3.8'

services:
  loom:
    image: loom:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - LOOM_PASSWORD=${LOOM_PASSWORD}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - loom-data:/app/data
      - loom-keys:/app/data/projects
    depends_on:
      - temporal
      - postgres
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  temporal:
    image: temporalio/auto-setup:latest
    restart: unless-stopped
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=${TEMPORAL_DB_PASSWORD}
      - POSTGRES_SEEDS=postgres
    ports:
      - "7233:7233"
      - "8088:8088"
    depends_on:
      - postgres

  postgres:
    image: postgres:16
    restart: unless-stopped
    environment:
      - POSTGRES_PASSWORD=${TEMPORAL_DB_PASSWORD}
      - POSTGRES_USER=temporal
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U temporal"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  loom-data:
  loom-keys:
  postgres-data:
```

#### Option 2: Kubernetes

**Best for:** Large teams, high availability, auto-scaling

See `k8s/` directory for Kubernetes manifests.

```bash
# Apply configurations
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secrets.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

#### Option 3: Cloud Platforms

**AWS ECS / Fargate:**
- Use task definitions for Loom, Temporal, PostgreSQL
- ECS Service Discovery for internal networking
- Application Load Balancer for HTTPS termination
- RDS PostgreSQL for managed database
- EFS for persistent volumes

**Google Cloud Run:**
- Deploy Loom as Cloud Run service
- Use Cloud SQL for PostgreSQL
- Temporal as separate Cloud Run service

**Azure Container Instances:**
- Deploy with Container Groups
- Azure Database for PostgreSQL
- Azure Files for persistence

---

## Configuration

### Environment Variables

Override config.yaml values with environment variables:

```bash
# Authentication
export LOOM_PASSWORD="your-secure-password"
export JWT_SECRET="your-jwt-secret"

# LLM Provider API Keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Temporal
export TEMPORAL_HOST="temporal:7233"
export TEMPORAL_NAMESPACE="default"

# Database
export DATABASE_PATH="/app/data/loom.db"
```

### Secrets Management

**Development:** Use `.env` file (NOT committed to git)

```bash
# .env
LOOM_PASSWORD=dev-password
OPENAI_API_KEY=sk-...
```

**Production Options:**

1. **Docker Secrets:**
   ```bash
   echo "my-secret" | docker secret create loom_password -
   ```

2. **Kubernetes Secrets:**
   ```bash
   kubectl create secret generic loom-secrets \
     --from-literal=password=my-password \
     --from-literal=jwt-secret=my-jwt
   ```

3. **Cloud Provider Secret Managers:**
   - AWS Secrets Manager
   - Google Secret Manager
   - Azure Key Vault

### SSL/TLS Configuration

**Option 1: Reverse Proxy (Recommended)**

Use Nginx, Traefik, or cloud load balancer for SSL termination:

```nginx
server {
    listen 443 ssl http2;
    server_name loom.example.com;

    ssl_certificate /etc/ssl/certs/loom.crt;
    ssl_certificate_key /etc/ssl/private/loom.key;

    location / {
        proxy_pass http://loom:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Option 2: Let's Encrypt with Traefik**

See `docker-compose.traefik.yml` for automatic HTTPS setup.

---

## Monitoring & Maintenance

### Health Checks

Loom provides health endpoints:

```bash
# Liveness check (is the service running?)
curl http://localhost:8080/health/live

# Readiness check (is the service ready to accept traffic?)
curl http://localhost:8080/health

# Detailed system status
curl http://localhost:8080/api/v1/system/status
```

### Metrics & Observability

**Built-in Endpoints:**
- `/api/v1/analytics/logs` - Request logs
- `/api/v1/analytics/stats` - Usage statistics
- `/api/v1/patterns/analysis` - Cost analysis

**Integration Options:**

1. **Prometheus:**
   - Export metrics via `/metrics` endpoint (future feature)
   - Scrape with Prometheus
   - Visualize with Grafana

2. **Logging:**
   - JSON structured logging
   - Forward to ELK Stack, Splunk, or cloud logging services

3. **Tracing:**
   - Temporal UI for workflow tracing
   - OpenTelemetry integration (future)

### Backup & Recovery

**What to backup:**

1. **Database:** `loom.db`
   ```bash
   # Backup
   docker exec loom sqlite3 /app/data/loom.db ".backup /app/data/backup.db"

   # Or volume backup
   docker run --rm -v loom-data:/data -v $(pwd):/backup alpine \
     tar czf /backup/loom-data-$(date +%Y%m%d).tar.gz /data
   ```

2. **Project SSH Keys:** `/app/data/projects/`

3. **Configuration:** `config.yaml`

**Restore:**

```bash
# Stop service
docker compose down

# Restore database
cp backup.db data/loom.db

# Restart
docker compose up -d
```

**Backup Schedule Recommendations:**
- Database: Daily
- Config: On change
- Retention: 30 days minimum

### Database Maintenance

```bash
# Vacuum SQLite database
docker exec loom sqlite3 /app/data/loom.db "VACUUM;"

# Check database integrity
docker exec loom sqlite3 /app/data/loom.db "PRAGMA integrity_check;"

# Clean old analytics logs (90+ days)
curl -X POST http://localhost:8080/api/v1/analytics/cleanup?days=90
```

---

## Troubleshooting

### Common Issues

#### 1. Loom won't start

**Symptoms:** Container exits immediately

**Solutions:**
```bash
# Check logs
docker logs loom

# Common causes:
# - Missing config.yaml
# - Invalid YAML syntax
# - Database locked
# - Port already in use

# Fix port conflict
docker compose down
lsof -i :8080  # Find what's using port 8080
docker compose up -d
```

#### 2. Temporal connection errors

**Symptoms:** "failed to connect to temporal" errors

**Solutions:**
```bash
# Verify Temporal is running
docker ps | grep temporal

# Check Temporal health
curl http://localhost:7233/

# Restart Temporal
docker compose restart temporal

# Check Temporal logs
docker compose logs temporal
```

#### 3. Provider API failures

**Symptoms:** "provider not available" or API errors

**Solutions:**
```bash
# Check provider status
curl http://localhost:8080/api/v1/providers

# Test API key
export OPENAI_API_KEY="sk-..."
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"

# Re-activate provider
curl -X POST http://localhost:8080/api/v1/providers/openai/status \
  -H "Content-Type: application/json" \
  -d '{"status": "active"}'
```

#### 4. High memory usage

**Symptoms:** OOM kills, slow performance

**Solutions:**
```bash
# Check memory usage
docker stats loom

# Limit container memory
docker update --memory 4g --memory-swap 4g loom

# Or in docker-compose.yml:
services:
  loom:
    mem_limit: 4g
    memswap_limit: 4g
```

#### 5. Database locked errors

**Symptoms:** "database is locked" errors

**Solutions:**
```bash
# Check for hung processes
docker exec loom ps aux

# Restart Loom
docker compose restart loom

# If persistent, check for corruption
docker exec loom sqlite3 /app/data/loom.db "PRAGMA integrity_check;"
```

### Debug Mode

Enable debug logging:

```yaml
# config.yaml
logging:
  level: debug
  format: json
```

Or via environment:

```bash
export LOG_LEVEL=debug
```

### Getting Help

1. **Documentation:** Check docs/ directory
2. **GitHub Issues:** https://github.com/jordanhubbard/Loom/issues
3. **Logs:** Always include logs when reporting issues

---

## Security Best Practices

1. **Change default passwords**
   - Set strong LOOM_PASSWORD
   - Rotate JWT secrets regularly

2. **Enable HTTPS**
   - Use SSL/TLS in production
   - Set `require_https: true` in config

3. **Network isolation**
   - Use Docker networks
   - Firewall rules for external access only on port 443

4. **API key security**
   - Use environment variables or secrets managers
   - Never commit API keys to git

5. **Regular updates**
   - Keep Loom updated
   - Update base Docker images
   - Apply security patches

6. **Access control**
   - Enable authentication (`enable_auth: true`)
   - Use RBAC for multi-user environments
   - Review user permissions regularly

7. **Audit logging**
   - Enable analytics logging
   - Monitor for suspicious activity
   - Set up alerts for anomalies

---

## Production Checklist

Before deploying to production:

- [ ] Configuration reviewed and secrets secured
- [ ] HTTPS/SSL configured
- [ ] Authentication enabled
- [ ] Health checks configured
- [ ] Monitoring and alerting set up
- [ ] Backup strategy implemented
- [ ] Load testing completed
- [ ] Documentation updated
- [ ] Disaster recovery plan documented
- [ ] On-call rotation established

---

## Additional Resources

- [Architecture Guide](./ARCHITECTURE.md)
- [User Guide](./USER_GUIDE.md)
- [API Documentation](./API_CAPABILITIES_SUMMARY.md)
- [Contributing](../CONTRIBUTING.md)
