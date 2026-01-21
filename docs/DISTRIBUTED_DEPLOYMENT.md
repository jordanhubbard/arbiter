# Distributed Deployment Guide

This guide explains how to deploy multiple AgentiCorp instances for high availability and load distribution.

## Overview

AgentiCorp supports distributed deployment with:
- **Shared State**: PostgreSQL database for synchronized state
- **Distributed Locking**: Coordination between instances
- **Instance Registry**: Track active instances
- **Load Balancing**: Distribute requests across instances
- **Automatic Failover**: Continue operating if instances fail

## Architecture

```
                    ┌─────────────┐
                    │Load Balancer│
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
      ┌────▼────┐     ┌────▼────┐    ┌────▼────┐
      │Instance │     │Instance │    │Instance │
      │   #1    │     │   #2    │    │   #3    │
      └────┬────┘     └────┬────┘    └────┬────┘
           │               │               │
           └───────────────┼───────────────┘
                           │
                    ┌──────▼──────┐
                    │  PostgreSQL │
                    │(Shared State)│
                    └─────────────┘
```

## Prerequisites

- PostgreSQL 12+ (for shared state)
- Load balancer (Nginx, HAProxy, etc.)
- Multiple servers/containers for instances

## Quick Start

### 1. Setup PostgreSQL

```bash
# Install PostgreSQL
sudo apt-get install postgresql postgresql-contrib

# Create database and user
sudo -u postgres psql
CREATE DATABASE agenticorp;
CREATE USER agenticorp_user WITH ENCRYPTED PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE agenticorp TO agenticorp_user;
\q
```

### 2. Configure AgentiCorp

Update `config.yaml`:

```yaml
database:
  type: postgres
  dsn: "postgresql://agenticorp_user:your_secure_password@postgres-host:5432/agenticorp?sslmode=require"

server:
  http_port: 8080
  enable_http: true
```

### 3. Start Multiple Instances

```bash
# Instance 1
PORT=8080 ./agenticorp &

# Instance 2  
PORT=8081 ./agenticorp &

# Instance 3
PORT=8082 ./agenticorp &
```

### 4. Configure Load Balancer

**Nginx Example:**

```nginx
upstream agenticorp_backend {
    least_conn;  # Use least connections algorithm
    
    server 127.0.0.1:8080 max_fails=3 fail_timeout=30s;
    server 127.0.0.1:8081 max_fails=3 fail_timeout=30s;
    server 127.0.0.1:8082 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name agenticorp.example.com;
    
    location / {
        proxy_pass http://agenticorp_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        
        # Health check
        proxy_next_upstream error timeout invalid_header http_500 http_502 http_503;
    }
    
    location /health {
        proxy_pass http://agenticorp_backend;
        access_log off;
    }
}
```

## Distributed Locking

AgentiCorp uses distributed locks to prevent conflicts in shared operations.

### Lock Usage Example

Locks are automatically used for:
- Agent assignment
- Bead status updates
- Provider registration
- Configuration changes

### Manual Lock Usage (Advanced)

```go
// Acquire lock
lock, err := db.AcquireLock(ctx, "my-operation", 30*time.Second)
if err != nil {
    return fmt.Errorf("failed to acquire lock: %w", err)
}
defer lock.Release(ctx)

// Perform operation while holding lock
// ...
```

## Instance Registry

All instances register themselves for coordination and monitoring.

### View Active Instances

```bash
# Via API
curl http://localhost:8080/api/v1/instances

# Via database
psql agenticorp -c "SELECT * FROM instances WHERE last_heartbeat > NOW() - INTERVAL '60 seconds';"
```

### Instance Metadata

Each instance reports:
- Instance ID (UUID)
- Hostname
- Start time
- Last heartbeat
- Status
- Metadata (version, etc.)

## State Synchronization

State is automatically synchronized via the shared PostgreSQL database:

- **Providers**: Shared across all instances
- **Agents**: Assignments visible to all instances
- **Beads**: Status updates propagate immediately
- **Config**: Configuration changes apply to all
- **Logs**: Request logs centralized

## High Availability Features

### Automatic Failover

- If an instance fails, others continue operating
- Load balancer detects failures via health checks
- Locks expire and can be reclaimed
- No manual intervention required

### Split-Brain Prevention

- Distributed locks prevent concurrent operations
- Database transactions ensure consistency
- Instance registry tracks active instances
- Stale instances automatically cleaned up

### Data Consistency

- ACID transactions via PostgreSQL
- Optimistic locking for updates
- Version tracking for entities
- Conflict resolution strategies

## Monitoring

### Health Checks

```bash
# Check instance health
curl http://localhost:8080/health

# Expected response:
{
  "healthy": true,
  "instance_id": "abc-123",
  "database": "connected",
  "active_instances": 3
}
```

### Metrics

Monitor these metrics:
- Active instances count
- Database connection pool
- Lock acquisition times
- Request distribution
- Failover events

### Logging

Configure centralized logging:

```yaml
logging:
  level: info
  format: json
  output: stdout
  
  # For centralized logging
  syslog:
    enabled: true
    address: "logs.example.com:514"
```

## Deployment Patterns

### Docker Compose

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: agenticorp
      POSTGRES_USER: agenticorp_user
      POSTGRES_PASSWORD: secure_password
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
  
  agenticorp:
    image: agenticorp:latest
    depends_on:
      - postgres
    environment:
      DATABASE_TYPE: postgres
      DATABASE_DSN: postgresql://agenticorp_user:secure_password@postgres:5432/agenticorp
    deploy:
      replicas: 3
    ports:
      - "8080-8082:8080"

volumes:
  postgres-data:
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agenticorp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agenticorp
  template:
    metadata:
      labels:
        app: agenticorp
    spec:
      containers:
      - name: agenticorp
        image: agenticorp:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_TYPE
          value: "postgres"
        - name: DATABASE_DSN
          valueFrom:
            secretKeyRef:
              name: agenticorp-db
              key: dsn
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: agenticorp
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: agenticorp
```

## Performance Tuning

### PostgreSQL

```sql
-- Connection pooling
ALTER SYSTEM SET max_connections = 100;

-- Performance
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET work_mem = '16MB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';

-- WAL configuration for performance
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
```

### Instance Configuration

```yaml
database:
  type: postgres
  dsn: "postgresql://user:pass@host:5432/agenticorp?pool_max_conns=25&pool_min_conns=5"

agents:
  max_concurrent: 20  # Adjust based on instance size
```

### Load Balancer

- Use `least_conn` algorithm for even distribution
- Configure health check interval (5-10s)
- Set appropriate timeouts (30-60s)
- Enable connection pooling

## Troubleshooting

### Instances Can't Connect

```bash
# Check PostgreSQL is accessible
psql "postgresql://user:pass@host:5432/agenticorp"

# Check network connectivity
nc -zv postgres-host 5432

# Check database logs
sudo tail -f /var/log/postgresql/postgresql-15-main.log
```

### Lock Contention

```sql
-- View active locks
SELECT * FROM distributed_locks 
WHERE expires_at > NOW()
ORDER BY acquired_at;

-- Clean up expired locks
DELETE FROM distributed_locks 
WHERE expires_at < NOW();
```

### Stale Instances

```sql
-- View all instances
SELECT * FROM instances 
ORDER BY last_heartbeat DESC;

-- Remove stale instances (no heartbeat for 5 minutes)
DELETE FROM instances 
WHERE last_heartbeat < NOW() - INTERVAL '5 minutes';
```

### Split Brain

If you suspect split brain:

```sql
-- Check for multiple instances claiming same operation
SELECT lock_name, COUNT(*) as instances
FROM distributed_locks
GROUP BY lock_name
HAVING COUNT(*) > 1;
```

## Best Practices

1. **Use Odd Number of Instances**: 3 or 5 instances for better failover
2. **Monitor Health Checks**: Alert on instance failures
3. **Regular Backups**: PostgreSQL backups are critical
4. **Resource Limits**: Set appropriate CPU/memory limits
5. **Connection Pooling**: Use pgBouncer for large deployments
6. **Separate Database**: Don't run PostgreSQL on same host
7. **SSL/TLS**: Always use encrypted connections
8. **Firewall**: Restrict PostgreSQL access to AgentiCorp instances
9. **Log Aggregation**: Use centralized logging (ELK, Splunk)
10. **Disaster Recovery**: Have rollback and recovery procedures

## Security

### Database Security

```yaml
database:
  dsn: "postgresql://user:pass@host:5432/agenticorp?sslmode=require"
  # Use environment variable for sensitive data
  # dsn: ${DATABASE_DSN}
```

### Network Security

- Use VPC/private network for instance-to-database
- Enable SSL/TLS for all connections
- Use firewall rules to restrict access
- Implement rate limiting on load balancer

### Secrets Management

- Use Kubernetes secrets or Vault
- Rotate credentials regularly
- Never commit credentials to git
- Use IAM roles when possible (AWS RDS, etc.)

## Migration from SQLite

To migrate from single-instance SQLite to distributed PostgreSQL:

1. **Export Data**: Use `pg_dump` equivalent for SQLite
2. **Setup PostgreSQL**: Create database and schema
3. **Import Data**: Load data into PostgreSQL
4. **Update Config**: Change to postgres in config.yaml
5. **Test**: Verify with single instance first
6. **Scale**: Add additional instances

## Support

For distributed deployment issues:
- Check logs: `docker logs agenticorp`
- Database status: `psql agenticorp -c "SELECT 1"`
- Instance registry: `GET /api/v1/instances`
- GitHub Issues: Report problems

---

**Distributed deployment enables enterprise-scale AgentiCorp!** 🚀
