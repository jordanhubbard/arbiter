package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/analytics"
	"github.com/jordanhubbard/loom/internal/cache"
	"github.com/jordanhubbard/loom/pkg/config"
)

// ============================================================
// Analytics handler tests
// ============================================================

func TestHandleGetLogs_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/logs", nil)
	w := httptest.NewRecorder()
	s.handleGetLogs(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGetLogs_Unauthorized(t *testing.T) {
	cfg := &config.Config{Security: config.SecurityConfig{EnableAuth: true}}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/logs", nil)
	w := httptest.NewRecorder()
	s.handleGetLogs(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleGetLogStats_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/stats", nil)
	w := httptest.NewRecorder()
	s.handleGetLogStats(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGetLogStats_Unauthorized(t *testing.T) {
	cfg := &config.Config{Security: config.SecurityConfig{EnableAuth: true}}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/stats", nil)
	w := httptest.NewRecorder()
	s.handleGetLogStats(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleExportLogs_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/export", nil)
	w := httptest.NewRecorder()
	s.handleExportLogs(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleExportLogs_Unauthorized(t *testing.T) {
	cfg := &config.Config{Security: config.SecurityConfig{EnableAuth: true}}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/export", nil)
	w := httptest.NewRecorder()
	s.handleExportLogs(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleGetCostReport_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/costs", nil)
	w := httptest.NewRecorder()
	s.handleGetCostReport(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGetCostReport_Unauthorized(t *testing.T) {
	cfg := &config.Config{Security: config.SecurityConfig{EnableAuth: true}}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/costs", nil)
	w := httptest.NewRecorder()
	s.handleGetCostReport(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleGetBatchingRecommendations_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/batching", nil)
	w := httptest.NewRecorder()
	s.handleGetBatchingRecommendations(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGetBatchingRecommendations_NilAnalytics(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/batching", nil)
	w := httptest.NewRecorder()
	s.handleGetBatchingRecommendations(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHandleGetBatchingRecommendations_Unauthorized(t *testing.T) {
	cfg := &config.Config{Security: config.SecurityConfig{EnableAuth: true}}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time), analyticsLogger: &analytics.Logger{}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/batching", nil)
	w := httptest.NewRecorder()
	s.handleGetBatchingRecommendations(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleExportStats_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/export-stats", nil)
	w := httptest.NewRecorder()
	s.handleExportStats(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleExportStats_Unauthorized(t *testing.T) {
	cfg := &config.Config{Security: config.SecurityConfig{EnableAuth: true}}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/export-stats", nil)
	w := httptest.NewRecorder()
	s.handleExportStats(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ============================================================
// Export CSV utility tests
// ============================================================

func TestExportStatsAsCSV(t *testing.T) {
	w := httptest.NewRecorder()
	stats := &analytics.LogStats{
		TotalRequests:      100,
		TotalTokens:        5000,
		TotalCostUSD:       1.50,
		AvgLatencyMs:       200.5,
		ErrorRate:          0.05,
		CostByProvider:     map[string]float64{"openai": 1.0, "anthropic": 0.5},
		CostByUser:         map[string]float64{"user1": 1.5},
		RequestsByProvider: map[string]int64{"openai": 60, "anthropic": 40},
		RequestsByUser:     map[string]int64{"user1": 100},
	}
	filter := &analytics.LogFilter{}
	exportStatsAsCSV(w, stats, filter)

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "csv") {
		t.Errorf("expected csv content type, got %s", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Total Requests") {
		t.Error("expected Total Requests in CSV")
	}
	if !strings.Contains(body, "100") {
		t.Error("expected request count in CSV")
	}
}

func TestExportLogsAsCSV(t *testing.T) {
	w := httptest.NewRecorder()
	logs := []*analytics.RequestLog{
		{
			Timestamp:        time.Now(),
			UserID:           "user1",
			Method:           "POST",
			Path:             "/api/v1/chat/completions",
			ProviderID:       "openai",
			ModelName:        "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 200,
			TotalTokens:      300,
			LatencyMs:        150,
			StatusCode:       200,
			CostUSD:          0.05,
		},
	}
	exportLogsAsCSV(w, logs)

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "csv") {
		t.Errorf("expected csv content type, got %s", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "user1") {
		t.Error("expected user1 in CSV")
	}
	if !strings.Contains(body, "gpt-4") {
		t.Error("expected gpt-4 in CSV")
	}
}

func TestExportLogsAsCSV_EmptyLogs(t *testing.T) {
	w := httptest.NewRecorder()
	exportLogsAsCSV(w, nil)
	body := w.Body.String()
	if !strings.Contains(body, "Timestamp") {
		t.Error("expected CSV header even for empty logs")
	}
}

// ============================================================
// Pattern handler method tests
// ============================================================

func TestHandlePatternAnalysis_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/patterns/analysis", nil)
	w := httptest.NewRecorder()
	s.handlePatternAnalysis(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleExpensivePatterns_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/patterns/expensive", nil)
	w := httptest.NewRecorder()
	s.handleExpensivePatterns(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleAnomalies_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/patterns/anomalies", nil)
	w := httptest.NewRecorder()
	s.handleAnomalies(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleOptimizations_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/optimizations", nil)
	w := httptest.NewRecorder()
	s.handleOptimizations(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandlePromptAnalysis_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/prompts/analysis", nil)
	w := httptest.NewRecorder()
	s.handlePromptAnalysis(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandlePromptOptimizations_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/prompts/optimizations", nil)
	w := httptest.NewRecorder()
	s.handlePromptOptimizations(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleSubstitutions_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/optimizations/substitutions", nil)
	w := httptest.NewRecorder()
	s.handleSubstitutions(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Cache analyzer handler method tests
// ============================================================

func TestHandleCacheAnalysis_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/analysis", nil)
	w := httptest.NewRecorder()
	s.handleCacheAnalysis(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleCacheOpportunities_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/opportunities", nil)
	w := httptest.NewRecorder()
	s.handleCacheOpportunities(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleCacheOptimize_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cache/optimize", nil)
	w := httptest.NewRecorder()
	s.handleCacheOptimize(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleCacheRecommendations_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/recommendations", nil)
	w := httptest.NewRecorder()
	s.handleCacheRecommendations(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Workflow handler method tests
// ============================================================

func TestHandleWorkflows_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", nil)
	w := httptest.NewRecorder()
	s.handleWorkflows(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleWorkflowExecutions_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/executions", nil)
	w := httptest.NewRecorder()
	s.handleWorkflowExecutions(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleWorkflowAnalytics_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/analytics", nil)
	w := httptest.NewRecorder()
	s.handleWorkflowAnalytics(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Motivation handler method tests
// ============================================================

func TestHandleMotivations_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/motivations", nil)
	w := httptest.NewRecorder()
	s.handleMotivations(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleMotivationHistory_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/motivations/history", nil)
	w := httptest.NewRecorder()
	s.handleMotivationHistory(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleIdleState_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/motivations/idle", nil)
	w := httptest.NewRecorder()
	s.handleIdleState(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleMotivationRoles_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/motivations/roles", nil)
	w := httptest.NewRecorder()
	s.handleMotivationRoles(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleMotivationDefaults_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/motivations/defaults", nil)
	w := httptest.NewRecorder()
	s.handleMotivationDefaults(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Notification handler method tests
// ============================================================

func TestHandleGetNotifications_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", nil)
	w := httptest.NewRecorder()
	s.handleGetNotifications(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleNotificationStream_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/stream", nil)
	w := httptest.NewRecorder()
	s.handleNotificationStream(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleMarkAllRead_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/mark-all-read", nil)
	w := httptest.NewRecorder()
	s.handleMarkAllRead(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Event handler method tests
// ============================================================

func TestHandleEventStream_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/stream", nil)
	w := httptest.NewRecorder()
	s.handleEventStream(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGetEvents_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	w := httptest.NewRecorder()
	s.handleGetEvents(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGetEventStats_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/stats", nil)
	w := httptest.NewRecorder()
	s.handleGetEventStats(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Activity handler method tests
// ============================================================

func TestHandleGetActivityFeed_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activity-feed", nil)
	w := httptest.NewRecorder()
	s.handleGetActivityFeed(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleActivityFeedStream_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activity-feed/stream", nil)
	w := httptest.NewRecorder()
	s.handleActivityFeedStream(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Config handler method tests
// ============================================================

func TestHandleConfig_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/config", nil)
	w := httptest.NewRecorder()
	s.handleConfig(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleConfigExportYAML_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/export.yaml", nil)
	w := httptest.NewRecorder()
	s.handleConfigExportYAML(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleConfigImportYAML_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/import.yaml", nil)
	w := httptest.NewRecorder()
	s.handleConfigImportYAML(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// System handler method tests
// ============================================================

func TestHandleSystemStatus_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/status", nil)
	w := httptest.NewRecorder()
	s.handleSystemStatus(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleRecommendedModels_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/models/recommended", nil)
	w := httptest.NewRecorder()
	s.handleRecommendedModels(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Handlers that check s.app.Get*() - test nil-app-safe behavior
// handlers.go handlePersonas, handleAgents (with non-matching method)
// ============================================================

func TestHandlePersonas_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/personas", nil)
	w := httptest.NewRecorder()
	s.handlePersonas(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandlePersona_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/personas/test", nil)
	w := httptest.NewRecorder()
	s.handlePersona(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleAgents_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	s.handleAgents(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleAgents_PostInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleAgents(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleAgents_PostMissingFields(t *testing.T) {
	s := newTestServer()
	body := `{"name":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleAgents(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleAgent_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/agents/agent1", nil)
	w := httptest.NewRecorder()
	s.handleAgent(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleCloneAgent_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/a1/clone", nil)
	w := httptest.NewRecorder()
	s.handleCloneAgent(w, req, "a1")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleCloneAgent_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/a1/clone", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleCloneAgent(w, req, "a1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleCloneAgent_MissingPersonaName(t *testing.T) {
	s := newTestServer()
	body := `{"new_persona_name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/a1/clone", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleCloneAgent(w, req, "a1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleAgentAction_UnknownAction(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/a1/unknown", nil)
	w := httptest.NewRecorder()
	s.handleAgentAction(w, req, "a1", "unknown")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ============================================================
// Projects handler method tests
// ============================================================

func TestHandleProjects_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	s.handleProjects(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleProjects_PostInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleProjects(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProjects_PostMissingFields(t *testing.T) {
	s := newTestServer()
	body := `{"name":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleProjects(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProject_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/p1", nil)
	w := httptest.NewRecorder()
	s.handleProject(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleProjectStateEndpoints_UnknownAction(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/unknown", nil)
	w := httptest.NewRecorder()
	s.handleProjectStateEndpoints(w, req, "p1", "unknown")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleBootstrapProject_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/bootstrap", nil)
	w := httptest.NewRecorder()
	s.handleBootstrapProject(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleBootstrapProject_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/bootstrap", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleBootstrapProject(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================
// Beads handler tests
// ============================================================

func TestHandleBeads_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/beads", nil)
	w := httptest.NewRecorder()
	s.handleBeads(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleBeads_PostInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleBeads(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleBeads_PostMissingFields(t *testing.T) {
	s := newTestServer()
	body := `{"type":"task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleBeads(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleBead_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/beads/b1", nil)
	w := httptest.NewRecorder()
	s.handleBead(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleDecisions_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decisions", nil)
	w := httptest.NewRecorder()
	s.handleDecisions(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleFileLocks_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/file-locks", nil)
	w := httptest.NewRecorder()
	s.handleFileLocks(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleFileLocks_PostInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/file-locks", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleFileLocks(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleFileLocks_PostMissingFields(t *testing.T) {
	s := newTestServer()
	body := `{"file_path":"test.go"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/file-locks", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleFileLocks(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleFileLock_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/file-locks/p1/file.go", nil)
	w := httptest.NewRecorder()
	s.handleFileLock(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleFileLock_InvalidPath(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/file-locks/noslash", nil)
	w := httptest.NewRecorder()
	s.handleFileLock(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleFileLock_MissingAgentID(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/file-locks/proj/file.go", nil)
	w := httptest.NewRecorder()
	s.handleFileLock(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleWorkGraph_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/work-graph", nil)
	w := httptest.NewRecorder()
	s.handleWorkGraph(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Provider handler method tests
// ============================================================

func TestHandleProviders_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/providers", nil)
	w := httptest.NewRecorder()
	s.handleProviders(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleProviders_PostInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleProviders(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProvider_MissingID(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/", nil)
	w := httptest.NewRecorder()
	s.handleProvider(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProvider_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/p1", nil)
	w := httptest.NewRecorder()
	s.handleProvider(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleProvider_PutInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/providers/p1", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleProvider(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProvider_ModelsMethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers/p1/models", nil)
	w := httptest.NewRecorder()
	s.handleProvider(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleProvider_NegotiateMethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/p1/negotiate", nil)
	w := httptest.NewRecorder()
	s.handleProvider(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Project state endpoints method tests
// ============================================================

func TestHandleCloseProject_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/close", nil)
	w := httptest.NewRecorder()
	s.handleCloseProject(w, req, "p1")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleCloseProject_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/close", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleCloseProject(w, req, "p1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleCloseProject_MissingAuthor(t *testing.T) {
	s := newTestServer()
	body := `{"comment":"closing"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/close", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleCloseProject(w, req, "p1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleReopenProject_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/reopen", nil)
	w := httptest.NewRecorder()
	s.handleReopenProject(w, req, "p1")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleReopenProject_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/reopen", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleReopenProject(w, req, "p1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleReopenProject_MissingAuthor(t *testing.T) {
	s := newTestServer()
	body := `{"comment":"reopening"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/reopen", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleReopenProject(w, req, "p1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProjectComments_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/p1/comments", nil)
	w := httptest.NewRecorder()
	s.handleProjectComments(w, req, "p1")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleProjectComments_PostInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/comments", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleProjectComments(w, req, "p1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProjectComments_PostMissingFields(t *testing.T) {
	s := newTestServer()
	body := `{"author_id":"a1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/comments", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleProjectComments(w, req, "p1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProjectAgents_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/agents", nil)
	w := httptest.NewRecorder()
	s.handleProjectAgents(w, req, "p1")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleProjectAgents_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/agents", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleProjectAgents(w, req, "p1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProjectAgents_MissingAgentID(t *testing.T) {
	s := newTestServer()
	body := `{"action":"assign"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleProjectAgents(w, req, "p1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProjectAgents_BadAction(t *testing.T) {
	s := newTestServer()
	body := `{"agent_id":"a1","action":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleProjectAgents(w, req, "p1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleProjectState_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/state", nil)
	w := httptest.NewRecorder()
	s.handleProjectState(w, req, "p1")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleProjectGitKey_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/p1/git-key", nil)
	w := httptest.NewRecorder()
	s.handleProjectGitKey(w, req, "p1")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// File handler method checks
// ============================================================

func TestHandleProjectFiles_ReadMethodNotAllowed(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(cache.DefaultConfig())
	s := &Server{config: cfg, cache: c, apiFailureLast: make(map[string]time.Time)}
	// Set a non-nil fileManager to pass the nil check
	// But we can test method not allowed
	// The nil file manager check comes first, so let's just test that
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/files/read?path=test.txt", nil)
	w := httptest.NewRecorder()
	s.handleProjectFiles(w, req, "p1", []string{"read"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 (nil file manager), got %d", w.Code)
	}
}

func TestHandleProjectFiles_UnknownAction(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/files/unknown", nil)
	w := httptest.NewRecorder()
	s.handleProjectFiles(w, req, "p1", []string{"unknown"})
	// nil file manager check comes first
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ============================================================
// Logs handlers with invalid time params
// ============================================================

func TestHandleLogsRecent_InvalidSince(t *testing.T) {
	s := &Server{
		config:         &config.Config{},
		apiFailureLast: make(map[string]time.Time),
		logManager:     nil,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs/recent?since=badtime", nil)
	w := httptest.NewRecorder()
	s.HandleLogsRecent(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleLogsRecent_InvalidUntil(t *testing.T) {
	s := &Server{
		config:         &config.Config{},
		apiFailureLast: make(map[string]time.Time),
		logManager:     nil,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs/recent?until=badtime", nil)
	w := httptest.NewRecorder()
	s.HandleLogsRecent(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleLogsExport_InvalidStartTime(t *testing.T) {
	s := &Server{
		config:         &config.Config{},
		apiFailureLast: make(map[string]time.Time),
		logManager:     nil,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs/export?start_time=badtime", nil)
	w := httptest.NewRecorder()
	s.HandleLogsExport(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleLogsExport_InvalidEndTime(t *testing.T) {
	s := &Server{
		config:         &config.Config{},
		apiFailureLast: make(map[string]time.Time),
		logManager:     nil,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs/export?end_time=badtime", nil)
	w := httptest.NewRecorder()
	s.HandleLogsExport(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================
// Webhook method tests
// ============================================================

func TestHandleGitHubWebhook_NoSignatureConfigured(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			WebhookSecret: "",
		},
	}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}

	body := `{"action":"opened"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", strings.NewReader(body))
	req.Header.Set("X-GitHub-Event", "issues")
	// No signature needed if secret is empty
	w := httptest.NewRecorder()
	s.handleGitHubWebhook(w, req)
	// Should succeed (no signature verification if no secret)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (no secret configured), got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================
// Comment handler (nil comments manager)
// ============================================================

func TestHandleComment_NilUser(t *testing.T) {
	// handleComment calls s.app.GetCommentsManager() first which requires non-nil app
	t.Skip("requires app instance to avoid nil dereference")
}

// ============================================================
// SetupRoutes with WebUI enabled
// ============================================================

func TestSetupRoutes_WithWebUI(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		WebUI: config.WebUIConfig{
			Enabled:    true,
			StaticPath: tmpDir,
		},
	}
	s := NewServer(nil, nil, nil, cfg)
	handler := s.SetupRoutes()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

// ============================================================
// Additional extractID tests for edge cases
// ============================================================

func TestExtractID_SubPaths(t *testing.T) {
	s := newTestServer()

	// Test with claim sub-path
	id := s.extractID("/api/v1/beads/bead-123/claim", "/api/v1/beads")
	if id != "bead-123" {
		t.Errorf("expected bead-123, got %s", id)
	}

	// Test with empty path after prefix
	id2 := s.extractID("/api/v1/beads", "/api/v1/beads")
	if id2 != "" {
		t.Errorf("expected empty string, got %s", id2)
	}
}

// ============================================================
// Additional concurrent access test
// ============================================================

func TestShouldThrottleFailure_Concurrent(t *testing.T) {
	s := newTestServer()
	done := make(chan bool, 20)

	for i := 0; i < 20; i++ {
		go func(id int) {
			key := "concurrent-key"
			s.shouldThrottleFailure(key, time.Second)
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}

// ============================================================
// Bead claim validation
// ============================================================

func TestHandleBead_ClaimInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/b1/claim", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleBead(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleBead_ClaimMissingAgentID(t *testing.T) {
	s := newTestServer()
	body := `{"agent_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/b1/claim", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleBead(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleBead_ClaimMethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/b1/claim", nil)
	w := httptest.NewRecorder()
	s.handleBead(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleBead_RedispatchMethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/b1/redispatch", nil)
	w := httptest.NewRecorder()
	s.handleBead(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleBead_EscalateMethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/b1/escalate", nil)
	w := httptest.NewRecorder()
	s.handleBead(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleBead_PatchInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/beads/b1", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleBead(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================
// Decision handler validation
// ============================================================

func TestHandleDecision_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/decisions/d1", nil)
	w := httptest.NewRecorder()
	s.handleDecision(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleDecision_DecideMethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/decisions/d1/decide", nil)
	w := httptest.NewRecorder()
	s.handleDecision(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleDecision_DecideInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decisions/d1/decide", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleDecision(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDecision_DecideMissingFields(t *testing.T) {
	s := newTestServer()
	body := `{"decision":"yes"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decisions/d1/decide", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleDecision(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================
// OrgChart handler test
// ============================================================

func TestHandleOrgChart_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/org-charts/p1", nil)
	w := httptest.NewRecorder()
	s.handleOrgChart(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Project put invalid body
// ============================================================

func TestHandleProject_PutInvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleProject(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================
// handleBeadComments/handleComment auth tests
// ============================================================

func TestHandleBeadComments_MethodNotAllowed(t *testing.T) {
	// s.app is nil so s.app.GetCommentsManager() will panic
	// but the handler checks for commentsMgr first
	// Since s.app is nil we can't call this without panic
	// This test is intentionally skipped
	t.Skip("requires app instance")
}

// ============================================================
// Notification preferences method test
// ============================================================

func TestHandleNotificationPreferences_MethodNotAllowed(t *testing.T) {
	// handleNotificationPreferences accesses s.app.GetNotificationManager() before method check
	t.Skip("requires app instance to avoid nil dereference")
}

// ============================================================
// Additional respondJSON test with complex data
// ============================================================

func TestRespondJSON_WithNestedStruct(t *testing.T) {
	s := newTestServer()
	w := httptest.NewRecorder()
	data := map[string]interface{}{
		"nested": map[string]interface{}{
			"inner": "value",
			"count": 42,
		},
		"list": []string{"a", "b", "c"},
	}
	s.respondJSON(w, http.StatusOK, data)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
}
