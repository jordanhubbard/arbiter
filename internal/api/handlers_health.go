package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"
)

// HealthStatus represents the overall health status.
type HealthStatus struct {
	Status       string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp    time.Time              `json:"timestamp"`
	InstanceID   string                 `json:"instance_id,omitempty"`
	Uptime       int64                  `json:"uptime_seconds"`
	Version      string                 `json:"version,omitempty"`
	Dependencies map[string]DepHealth   `json:"dependencies"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
}

// DepHealth represents the health of a dependency.
type DepHealth struct {
	Status  string `json:"status"` // "healthy", "unhealthy", "unknown"
	Message string `json:"message,omitempty"`
	Latency int64  `json:"latency_ms,omitempty"`
}

var (
	startTime  = time.Now()
	instanceID = getInstanceID()
)

// handleHealthLive handles GET /health/live - Kubernetes liveness probe.
// Returns 200 if the application is running.
func (s *Server) handleHealthLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Liveness check: Is the process alive?
	// This should almost always return 200 unless the process is deadlocked

	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleHealthReady handles GET /health/ready - Kubernetes readiness probe.
// Returns 200 if the application is ready to serve traffic.
func (s *Server) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	deps := s.checkDependencies(ctx)

	// Check if critical dependencies are healthy
	critical := []string{"database"}
	ready := true

	for _, dep := range critical {
		if health, ok := deps[dep]; ok {
			if health.Status != "healthy" {
				ready = false
				break
			}
		}
	}

	response := map[string]interface{}{
		"ready":        ready,
		"timestamp":    time.Now().Format(time.RFC3339),
		"dependencies": deps,
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleHealthDetail handles GET /health - Detailed health information.
func (s *Server) handleHealthDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	deps := s.checkDependencies(ctx)

	// Determine overall status
	overallStatus := "healthy"
	for _, health := range deps {
		if health.Status == "unhealthy" {
			overallStatus = "unhealthy"
			break
		} else if health.Status == "degraded" {
			overallStatus = "degraded"
		}
	}

	// Get runtime metrics
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	metrics := map[string]interface{}{
		"goroutines":   runtime.NumGoroutine(),
		"memory_alloc": mem.Alloc,
		"memory_sys":   mem.Sys,
		"gc_runs":      mem.NumGC,
		"cpu_cores":    runtime.NumCPU(),
	}

	// Add cache stats if available
	if s.cache != nil {
		cacheStats := s.cache.GetStats(ctx)
		metrics["cache_hits"] = cacheStats.Hits
		metrics["cache_misses"] = cacheStats.Misses
		metrics["cache_hit_rate"] = cacheStats.HitRate
	}

	status := HealthStatus{
		Status:       overallStatus,
		Timestamp:    time.Now(),
		InstanceID:   instanceID,
		Uptime:       int64(time.Since(startTime).Seconds()),
		Version:      getVersion(),
		Dependencies: deps,
		Metrics:      metrics,
	}

	httpStatus := http.StatusOK
	if overallStatus == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// checkDependencies checks the health of all dependencies.
func (s *Server) checkDependencies(ctx context.Context) map[string]DepHealth {
	deps := make(map[string]DepHealth)

	// Check database
	deps["database"] = s.checkDatabase(ctx)

	// Check cache (optional)
	if s.cache != nil {
		deps["cache"] = s.checkCache(ctx)
	}

	// Check provider registry
	deps["providers"] = s.checkProviders(ctx)

	// Check analytics (optional)
	if s.analyticsLogger != nil {
		deps["analytics"] = DepHealth{
			Status:  "healthy",
			Message: "operational",
		}
	}

	return deps
}

// checkDatabase checks database connectivity.
func (s *Server) checkDatabase(ctx context.Context) DepHealth {
	if s.app == nil || s.app.GetDatabase() == nil {
		return DepHealth{
			Status:  "unknown",
			Message: "database not configured",
		}
	}

	start := time.Now()
	db := s.app.GetDatabase().DB()

	if err := db.PingContext(ctx); err != nil {
		return DepHealth{
			Status:  "unhealthy",
			Message: err.Error(),
			Latency: time.Since(start).Milliseconds(),
		}
	}

	// Check if we can get active instances (for distributed deployments)
	if s.app.GetDatabase().SupportsHA() {
		_, err := s.app.GetDatabase().ListActiveInstances(ctx)
		if err != nil {
			return DepHealth{
				Status:  "degraded",
				Message: "connected (instance query failed)",
				Latency: time.Since(start).Milliseconds(),
			}
		}
	}

	return DepHealth{
		Status:  "healthy",
		Message: "connected",
		Latency: time.Since(start).Milliseconds(),
	}
}

// checkCache checks cache health.
func (s *Server) checkCache(ctx context.Context) DepHealth {
	if s.cache == nil {
		return DepHealth{
			Status:  "unknown",
			Message: "cache not configured",
		}
	}

	stats := s.cache.GetStats(ctx)

	status := "healthy"
	message := "operational"

	// Check if hit rate is reasonable (if we have data)
	if stats.Hits+stats.Misses > 100 && stats.HitRate < 0.05 {
		status = "degraded"
		message = "low hit rate"
	}

	return DepHealth{
		Status:  status,
		Message: message,
	}
}

// checkProviders checks provider registry health.
func (s *Server) checkProviders(ctx context.Context) DepHealth {
	if s.app == nil {
		return DepHealth{
			Status:  "unknown",
			Message: "not initialized",
		}
	}

	// Count active providers
	// This is a simple check - could be enhanced
	return DepHealth{
		Status:  "healthy",
		Message: "operational",
	}
}

// Helper functions

func getInstanceID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return hostname
}

func getVersion() string {
	// Could be set via build flag: -ldflags "-X main.Version=1.2.3"
	return "1.2.0"
}
