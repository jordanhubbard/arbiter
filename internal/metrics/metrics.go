package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sync"
)

// Metrics holds all Prometheus metrics for Loom
type Metrics struct {
	// Agent metrics
	AgentsTotal       *prometheus.GaugeVec
	AgentStatus       *prometheus.GaugeVec
	AgentTaskDuration *prometheus.HistogramVec
	AgentTasksTotal   *prometheus.CounterVec

	// Bead metrics
	BeadsTotal      *prometheus.GaugeVec
	BeadStatus      *prometheus.GaugeVec
	BeadDuration    *prometheus.HistogramVec
	BeadsProcessed  *prometheus.CounterVec
	BeadTransitions *prometheus.CounterVec

	// Provider metrics
	ProvidersTotal   *prometheus.GaugeVec
	ProviderRequests *prometheus.CounterVec
	ProviderErrors   *prometheus.CounterVec
	ProviderLatency  *prometheus.HistogramVec
	ProviderTokens   *prometheus.CounterVec
	ProviderCost     *prometheus.CounterVec

	// Workflow metrics
	WorkflowsTotal     *prometheus.GaugeVec
	WorkflowExecutions *prometheus.CounterVec
	WorkflowDuration   *prometheus.HistogramVec
	WorkflowErrors     *prometheus.CounterVec

	// System metrics
	DatabaseConnections prometheus.Gauge
	CacheHits           prometheus.Counter
	CacheMisses         prometheus.Counter
	EventsPublished     *prometheus.CounterVec
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
}

var (
	metricsOnce   sync.Once
	sharedMetrics *Metrics
)

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	metricsOnce.Do(func() {
		sharedMetrics = &Metrics{
			// Agent metrics
			AgentsTotal: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "loom_agents_total",
					Help: "Total number of agents",
				},
				[]string{"project_id"},
			),
			AgentStatus: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "loom_agent_status",
					Help: "Agent status (1 for active, 0 for inactive)",
				},
				[]string{"agent_id", "status", "project_id", "role"},
			),
			AgentTaskDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "loom_agent_task_duration_seconds",
					Help:    "Duration of agent tasks in seconds",
					Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to 512s
				},
				[]string{"agent_id", "project_id", "success"},
			),
			AgentTasksTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_agent_tasks_total",
					Help: "Total number of tasks executed by agents",
				},
				[]string{"agent_id", "project_id", "result"},
			),

			// Bead metrics
			BeadsTotal: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "loom_beads_total",
					Help: "Total number of beads",
				},
				[]string{"project_id", "type"},
			),
			BeadStatus: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "loom_bead_status",
					Help: "Number of beads by status",
				},
				[]string{"project_id", "status", "priority"},
			),
			BeadDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "loom_bead_duration_seconds",
					Help:    "Time from bead creation to closure in seconds",
					Buckets: prometheus.ExponentialBuckets(60, 2, 12), // 1min to 68hrs
				},
				[]string{"project_id", "type", "priority"},
			),
			BeadsProcessed: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_beads_processed_total",
					Help: "Total number of beads processed",
				},
				[]string{"project_id", "type", "result"},
			),
			BeadTransitions: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_bead_transitions_total",
					Help: "Total number of bead status transitions",
				},
				[]string{"project_id", "from_status", "to_status"},
			),

			// Provider metrics
			ProvidersTotal: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "loom_providers_total",
					Help: "Total number of registered providers",
				},
				[]string{"type", "status"},
			),
			ProviderRequests: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_provider_requests_total",
					Help: "Total number of provider API requests",
				},
				[]string{"provider_id", "model", "success"},
			),
			ProviderErrors: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_provider_errors_total",
					Help: "Total number of provider errors",
				},
				[]string{"provider_id", "error_type"},
			),
			ProviderLatency: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "loom_provider_request_duration_seconds",
					Help:    "Provider API request duration in seconds",
					Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 100ms to 51s
				},
				[]string{"provider_id", "model"},
			),
			ProviderTokens: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_provider_tokens_total",
					Help: "Total tokens processed by provider",
				},
				[]string{"provider_id", "model", "type"}, // type: input, output, total
			),
			ProviderCost: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_provider_cost_usd_cents",
					Help: "Total cost in USD cents",
				},
				[]string{"provider_id", "model", "user_id"},
			),

			// Workflow metrics
			WorkflowsTotal: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "loom_workflows_total",
					Help: "Total number of workflows",
				},
				[]string{"project_id", "workflow_type"},
			),
			WorkflowExecutions: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_workflow_executions_total",
					Help: "Total number of workflow executions",
				},
				[]string{"workflow_type", "status"},
			),
			WorkflowDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "loom_workflow_duration_seconds",
					Help:    "Workflow execution duration in seconds",
					Buckets: prometheus.ExponentialBuckets(1, 2, 12), // 1s to 68min
				},
				[]string{"workflow_type"},
			),
			WorkflowErrors: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_workflow_errors_total",
					Help: "Total number of workflow errors",
				},
				[]string{"workflow_type", "error_type"},
			),

			// System metrics
			DatabaseConnections: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "loom_database_connections",
					Help: "Number of active database connections",
				},
			),
			CacheHits: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "loom_cache_hits_total",
					Help: "Total number of cache hits",
				},
			),
			CacheMisses: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "loom_cache_misses_total",
					Help: "Total number of cache misses",
				},
			),
			EventsPublished: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_events_published_total",
					Help: "Total number of events published",
				},
				[]string{"event_type", "project_id"},
			),
			HTTPRequestsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "loom_http_requests_total",
					Help: "Total number of HTTP requests",
				},
				[]string{"method", "path", "status"},
			),
			HTTPRequestDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "loom_http_request_duration_seconds",
					Help:    "HTTP request duration in seconds",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"method", "path"},
			),
		}
	})

	return sharedMetrics
}

// RecordProviderRequest records a provider API request
func (m *Metrics) RecordProviderRequest(providerID, model string, success bool, latencyMs int64, tokens int64) {
	successStr := "false"
	if success {
		successStr = "true"
	}
	m.ProviderRequests.WithLabelValues(providerID, model, successStr).Inc()
	m.ProviderLatency.WithLabelValues(providerID, model).Observe(float64(latencyMs) / 1000.0)
	if tokens > 0 {
		m.ProviderTokens.WithLabelValues(providerID, model, "total").Add(float64(tokens))
	}
}

// RecordBeadTransition records a bead status transition
func (m *Metrics) RecordBeadTransition(projectID, fromStatus, toStatus string) {
	m.BeadTransitions.WithLabelValues(projectID, fromStatus, toStatus).Inc()
}

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration float64) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}
