# Kubernetes Best Practices for Loom

## Current State Assessment

### ✅ Already Implemented
- **Non-root user**: Dockerfile runs as `loom` user (UID 1000)
- **Health probes**: `/health`, `/health/live`, `/health/ready` endpoints
- **Graceful shutdown**: SIGTERM handling with 10-second timeout
- **Multi-stage builds**: Optimized Docker image with separate builder stage
- **Config via environment**: `TEMPORAL_HOST`, `LOOM_PASSWORD`, etc.

### 🔄 Partially Implemented
- **Observability**: Basic health checks, but no Prometheus metrics
- **Security**: Non-root user, but no security context, capabilities drop, or read-only filesystem

### ❌ Missing / Needs Improvement

---

## Priority 1: Critical for Production

### 1. Resource Limits and Requests

**Why**: Prevents resource starvation, enables HPA, ensures predictable scheduling

**Current**: No resource limits defined

**Recommendation**: Add to Kubernetes manifests

```yaml
# k8s/deployment.yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

**Action Items**:
- [ ] Profile application under load to determine appropriate values
- [ ] Set requests based on baseline usage
- [ ] Set limits 2-3x requests to handle spikes
- [ ] Consider separate limits for worker vs API pods

---

### 2. Prometheus Metrics Endpoint

**Why**: Essential for monitoring, alerting, and autoscaling

**Current**: No metrics endpoint

**Recommendation**: Add `/metrics` endpoint with standard Go metrics

```go
// internal/metrics/prometheus.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    BeadsProcessed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "loom_beads_processed_total",
            Help: "Total number of beads processed",
        },
        []string{"status", "priority"},
    )
    
    AgentExecutionDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "loom_agent_execution_duration_seconds",
            Help:    "Agent execution duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"agent_id", "persona"},
    )
    
    ProviderRequests = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "loom_provider_requests_total",
            Help: "Total provider requests",
        },
        []string{"provider_id", "status"},
    )
    
    CommandExecutions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "loom_shell_commands_total",
            Help: "Total shell commands executed",
        },
        []string{"exit_code"},
    )
)
```

**Add to server**:
```go
// In internal/api/server.go
import "github.com/prometheus/client_golang/prometheus/promhttp"

// In SetupRoutes()
mux.Handle("/metrics", promhttp.Handler())
```

**Action Items**:
- [ ] Add prometheus client library
- [ ] Implement metrics for: beads, agents, providers, commands
- [ ] Add `/metrics` endpoint (no auth required)
- [ ] Set up Prometheus scraping config

---

### 3. Structured Logging (JSON)

**Why**: Better log aggregation, querying, and analysis in K8s

**Current**: Standard Go log with `log.Printf`

**Recommendation**: Use structured logger (zerolog or zap)

```go
// Use zerolog for structured logging
import "github.com/rs/zerolog/log"

// Instead of:
log.Printf("Command executed: %s", cmd)

// Use:
log.Info().
    Str("agent_id", agentID).
    Str("bead_id", beadID).
    Str("command", cmd).
    Int("exit_code", exitCode).
    Dur("duration", duration).
    Msg("Command executed")
```

**JSON Output**:
```json
{"level":"info","agent_id":"agent-123","bead_id":"bd-001","command":"ls","exit_code":0,"duration":5,"time":"2026-01-22T23:00:00Z","message":"Command executed"}
```

**Action Items**:
- [ ] Replace `log` with `zerolog` or `zap`
- [ ] Add structured fields to all log statements
- [ ] Set log level via environment variable
- [ ] Configure JSON output format

---

### 4. Security Context and Capabilities

**Why**: Principle of least privilege, defense in depth

**Current**: Runs as non-root but no additional restrictions

**Recommendation**: Add security context to K8s manifests

```yaml
# k8s/deployment.yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop:
      - ALL
    add:
      - NET_BIND_SERVICE  # Only if binding to port < 1024

# Volume mounts for writable paths
volumeMounts:
  - name: tmp
    mountPath: /tmp
  - name: data
    mountPath: /app/data
  - name: cache
    mountPath: /app/.cache

volumes:
  - name: tmp
    emptyDir: {}
  - name: cache
    emptyDir: {}
  - name: data
    persistentVolumeClaim:
      claimName: loom-data
```

**Action Items**:
- [ ] Test with `readOnlyRootFilesystem: true`
- [ ] Identify writable paths and mount as emptyDir
- [ ] Drop all capabilities
- [ ] Set `allowPrivilegeEscalation: false`

---

### 5. Secrets Management

**Why**: Never commit secrets to git, rotate easily, audit access

**Current**: Passwords in environment variables

**Recommendation**: Use Kubernetes Secrets or External Secrets Operator

```yaml
# k8s/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: loom-secrets
type: Opaque
stringData:
  admin-password: "changeme"  # Generated securely
  jwt-secret: "generated-secret-key"
  
---
# In deployment
env:
  - name: LOOM_PASSWORD
    valueFrom:
      secretKeyRef:
        name: loom-secrets
        key: admin-password
  - name: JWT_SECRET
    valueFrom:
      secretKeyRef:
        name: loom-secrets
        key: jwt-secret
```

**For External Secrets (HashiCorp Vault, AWS Secrets Manager)**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: loom-external
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: loom-secrets
  data:
    - secretKey: admin-password
      remoteRef:
        key: loom/admin
        property: password
```

**Action Items**:
- [ ] Create Kubernetes Secret manifests
- [ ] Remove secrets from docker-compose.yml
- [ ] Consider External Secrets Operator for production
- [ ] Implement secret rotation support

---

## Priority 2: High Value Improvements

### 6. Horizontal Pod Autoscaler (HPA)

**Why**: Auto-scale based on CPU, memory, or custom metrics

**Recommendation**: Enable HPA based on CPU and custom metrics

```yaml
# k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: loom
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: loom
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Pods
      pods:
        metric:
          name: loom_beads_queue_length
        target:
          type: AverageValue
          averageValue: "10"
```

**Action Items**:
- [ ] Set resource requests (required for HPA)
- [ ] Deploy metrics-server in cluster
- [ ] Configure HPA with conservative scaling
- [ ] Test scaling behavior under load

---

### 7. Pod Disruption Budget (PDB)

**Why**: Ensures availability during voluntary disruptions (upgrades, drains)

**Recommendation**: Maintain minimum availability

```yaml
# k8s/pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: loom
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: loom
```

**Action Items**:
- [ ] Create PDB for loom
- [ ] Create separate PDB for temporal
- [ ] Set `minAvailable: 1` or `maxUnavailable: 1`

---

### 8. StatefulSet for Database

**Why**: Stable network identity, ordered deployment, persistent storage

**Current**: SQLite with PVC (works for single instance)

**Recommendation**: For production, use StatefulSet with PostgreSQL

```yaml
# k8s/statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: loom-db
spec:
  serviceName: loom-db
  replicas: 1
  selector:
    matchLabels:
      app: loom-db
  template:
    metadata:
      labels:
        app: loom-db
    spec:
      containers:
        - name: postgres
          image: postgres:15-alpine
          env:
            - name: POSTGRES_DB
              value: loom
            - name: POSTGRES_USER
              valueFrom:
                secretKeyRef:
                  name: db-credentials
                  key: username
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: db-credentials
                  key: password
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 10Gi
```

**Action Items**:
- [ ] Migrate from SQLite to PostgreSQL for production
- [ ] Use StatefulSet for database
- [ ] Configure backup and restore procedures
- [ ] Consider managed database (RDS, Cloud SQL) for production

---

### 9. Network Policies

**Why**: Limit network access between pods (zero-trust)

**Recommendation**: Restrict traffic to only what's needed

```yaml
# k8s/network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: loom
spec:
  podSelector:
    matchLabels:
      app: loom
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080
  egress:
    # Allow DNS
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
      ports:
        - protocol: UDP
          port: 53
    # Allow Temporal
    - to:
        - podSelector:
            matchLabels:
              app: temporal
      ports:
        - protocol: TCP
          port: 7233
    # Allow database
    - to:
        - podSelector:
            matchLabels:
              app: loom-db
      ports:
        - protocol: TCP
          port: 5432
    # Allow external HTTPS (for git, providers)
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443
```

**Action Items**:
- [ ] Enable network policies in cluster
- [ ] Define egress rules for required external services
- [ ] Test thoroughly (can break unexpected dependencies)

---

### 10. Init Containers

**Why**: Run setup tasks before main container starts

**Recommendation**: Use for database migrations, config validation

```yaml
# k8s/deployment.yaml
initContainers:
  - name: migrate
    image: loom:latest
    command: ["/app/loom", "-migrate"]
    env:
      - name: DATABASE_URL
        valueFrom:
          secretKeyRef:
            name: db-credentials
            key: url
  - name: wait-for-temporal
    image: busybox:1.36
    command:
      - sh
      - -c
      - |
        until nc -z temporal 7233; do
          echo "Waiting for Temporal..."
          sleep 2
        done
```

**Action Items**:
- [ ] Add `-migrate` flag to run migrations
- [ ] Use init container for migrations
- [ ] Add wait-for-temporal init container
- [ ] Validate config before main container starts

---

## Priority 3: Advanced Features

### 11. Service Mesh (Istio/Linkerd)

**Why**: mTLS, traffic management, observability

**Recommendation**: Consider for multi-cluster or high-security environments

**Features**:
- Automatic mTLS between services
- Circuit breaking and retries
- Distributed tracing
- Traffic splitting (canary deployments)

**Action Items**:
- [ ] Evaluate service mesh need (complexity vs value)
- [ ] Start with Linkerd (lighter than Istio)
- [ ] Enable mTLS for inter-service communication

---

### 12. Pod Topology Spread Constraints

**Why**: Spread pods across zones/nodes for high availability

```yaml
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app: loom
  - maxSkew: 2
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        app: loom
```

**Action Items**:
- [ ] Add topology spread for multi-zone clusters
- [ ] Ensure at least 3 replicas for effective spreading

---

### 13. Admission Controllers & Policy Enforcement

**Why**: Enforce security and best practices automatically

**Recommendation**: Use OPA Gatekeeper or Kyverno

```yaml
# Example Gatekeeper ConstraintTemplate
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata:
  name: k8srequiredlabels
spec:
  crd:
    spec:
      names:
        kind: K8sRequiredLabels
      validation:
        openAPIV3Schema:
          properties:
            labels:
              type: array
              items:
                type: string
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package k8srequiredlabels
        violation[{"msg": msg}] {
          provided := {label | input.review.object.metadata.labels[label]}
          required := {label | label := input.parameters.labels[_]}
          missing := required - provided
          count(missing) > 0
          msg := sprintf("Missing required labels: %v", [missing])
        }
```

**Action Items**:
- [ ] Deploy OPA Gatekeeper or Kyverno
- [ ] Enforce required labels (app, version, environment)
- [ ] Enforce security policies (no root, read-only fs)
- [ ] Enforce resource limits on all pods

---

## Implementation Checklist

### Phase 1: Immediate (1-2 weeks)
- [ ] Add Prometheus metrics endpoint
- [ ] Implement structured logging (JSON)
- [ ] Define resource requests and limits
- [ ] Create Kubernetes Secret manifests
- [ ] Add security context to deployments

### Phase 2: Short-term (1 month)
- [ ] Enable HPA with CPU metrics
- [ ] Create Pod Disruption Budgets
- [ ] Add init containers for migrations
- [ ] Migrate to PostgreSQL (from SQLite)
- [ ] Implement custom metrics for HPA

### Phase 3: Medium-term (2-3 months)
- [ ] Deploy network policies
- [ ] Add topology spread constraints
- [ ] Set up External Secrets Operator
- [ ] Implement StatefulSet for database
- [ ] Configure backup and disaster recovery

### Phase 4: Long-term (3-6 months)
- [ ] Evaluate service mesh
- [ ] Deploy OPA Gatekeeper
- [ ] Multi-cluster setup
- [ ] Disaster recovery testing
- [ ] Chaos engineering

---

## Monitoring & Observability Stack

### Recommended Stack
```yaml
# Prometheus Operator
helm install prometheus-operator prometheus-community/kube-prometheus-stack

# Loki for logs
helm install loki grafana/loki-stack

# Jaeger for tracing (optional)
helm install jaeger jaegertracing/jaeger-operator
```

### Key Metrics to Track
- **Application**:
  - Bead processing rate
  - Agent execution duration
  - Provider request latency
  - Command execution count
  - Error rates

- **Infrastructure**:
  - Pod CPU/memory usage
  - Pod restart count
  - PVC usage
  - Network traffic

### Alerts to Configure
```yaml
# Prometheus alerts
groups:
  - name: loom
    rules:
      - alert: HighErrorRate
        expr: rate(loom_errors_total[5m]) > 0.05
        for: 5m
        annotations:
          summary: "High error rate detected"
      
      - alert: BeadProcessingStalled
        expr: rate(loom_beads_processed_total[5m]) == 0
        for: 15m
        annotations:
          summary: "No beads processed in 15 minutes"
      
      - alert: PodMemoryUsage
        expr: container_memory_usage_bytes{pod=~"loom.*"} / container_spec_memory_limit_bytes > 0.9
        for: 5m
        annotations:
          summary: "Pod using >90% memory"
```

---

## Testing Recommendations

### 1. Chaos Engineering
Use Chaos Mesh or Litmus to test resilience:
```yaml
# Test pod failure
apiVersion: chaos-mesh.org/v1alpha1
kind: PodChaos
metadata:
  name: pod-failure
spec:
  action: pod-failure
  mode: one
  selector:
    namespaces:
      - default
    labelSelectors:
      app: loom
  duration: "30s"
```

### 2. Load Testing
Test HPA and resource limits:
```bash
# Use k6 or locust
k6 run --vus 100 --duration 5m load-test.js
```

### 3. Backup/Restore Testing
Regularly test database backups:
```bash
# Automated backup test
kubectl exec -it loom-db-0 -- pg_dump loom > backup.sql
kubectl delete pvc data-loom-db-0
# Restore and verify
```

---

## Summary: Priority Order

**Must Have (P1)**:
1. Resource limits/requests
2. Prometheus metrics
3. Structured logging
4. Security context
5. Secrets management

**Should Have (P2)**:
6. HPA
7. PDB
8. StatefulSet for DB
9. Network policies
10. Init containers

**Nice to Have (P3)**:
11. Service mesh
12. Topology spread
13. Admission controllers

---

## References

- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [12-Factor App](https://12factor.net/)
- [CNCF Cloud Native Trail Map](https://github.com/cncf/trailmap)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
