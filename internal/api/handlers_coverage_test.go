package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/cache"
	"github.com/jordanhubbard/loom/internal/provider"
	"github.com/jordanhubbard/loom/pkg/config"
	"github.com/jordanhubbard/loom/pkg/models"
)

// ============================================================
// Helper: create a minimal Server with config (no app/DB)
// ============================================================

func newTestServer() *Server {
	cfg := &config.Config{}
	return &Server{
		config:         cfg,
		apiFailureLast: make(map[string]time.Time),
	}
}

func newTestServerWithCache() *Server {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:       true,
			DefaultTTL:    time.Hour,
			MaxSize:       1000,
			MaxMemoryMB:   100,
			CleanupPeriod: 5 * time.Minute,
		},
	}
	c := cache.New(&cache.Config{
		Enabled:       true,
		DefaultTTL:    time.Hour,
		MaxSize:       1000,
		MaxMemoryMB:   100,
		CleanupPeriod: 5 * time.Minute,
	})
	return &Server{
		config:         cfg,
		cache:          c,
		apiFailureLast: make(map[string]time.Time),
	}
}

// ============================================================
// Health endpoint tests
// ============================================================

func TestHandleHealthLive_GET(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	s.handleHealthLive(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "alive" {
		t.Errorf("expected status alive, got %v", resp["status"])
	}
}

func TestHandleHealthLive_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/health/live", nil)
	w := httptest.NewRecorder()

	s.handleHealthLive(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleHealthReady_GET_NoApp(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	s.handleHealthReady(w, req)

	// Without app, database is "unknown" which is not "healthy",
	// so the endpoint should return 503
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["ready"] != false {
		t.Errorf("expected ready=false, got %v", resp["ready"])
	}
}

func TestHandleHealthReady_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/health/ready", nil)
	w := httptest.NewRecorder()

	s.handleHealthReady(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleHealthDetail_GET_NoApp(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	s.handleHealthDetail(w, req)

	// Might be 200 or 503 depending on dependency checks; just ensure valid JSON
	var resp HealthStatus
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status == "" {
		t.Error("expected non-empty status")
	}
	if resp.Dependencies == nil {
		t.Error("expected non-nil dependencies")
	}
	if resp.Metrics == nil {
		t.Error("expected non-nil metrics")
	}
}

func TestHandleHealthDetail_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	s.handleHealthDetail(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Dependency checks (unit tests for nil paths)
// ============================================================

func TestCheckDependencies_NilApp(t *testing.T) {
	s := newTestServer()
	ctx := context.Background()
	deps := s.checkDependencies(ctx)

	if _, ok := deps["database"]; !ok {
		t.Error("expected database key in dependencies")
	}
	if deps["database"].Status != "unknown" {
		t.Errorf("expected database status=unknown, got %s", deps["database"].Status)
	}
	if _, ok := deps["providers"]; !ok {
		t.Error("expected providers key in dependencies")
	}
	if deps["providers"].Status != "unknown" {
		t.Errorf("expected providers status=unknown, got %s", deps["providers"].Status)
	}
}

func TestCheckDatabase_NilApp(t *testing.T) {
	s := newTestServer()
	ctx := context.Background()
	h := s.checkDatabase(ctx)
	if h.Status != "unknown" {
		t.Errorf("expected unknown, got %s", h.Status)
	}
}

func TestCheckCache_NilCache(t *testing.T) {
	s := newTestServer()
	ctx := context.Background()
	h := s.checkCache(ctx)
	if h.Status != "unknown" {
		t.Errorf("expected unknown, got %s", h.Status)
	}
}

func TestCheckCache_WithCache(t *testing.T) {
	s := newTestServerWithCache()
	ctx := context.Background()
	h := s.checkCache(ctx)
	if h.Status != "healthy" {
		t.Errorf("expected healthy, got %s", h.Status)
	}
}

func TestCheckProviders_NilApp(t *testing.T) {
	s := newTestServer()
	ctx := context.Background()
	h := s.checkProviders(ctx)
	if h.Status != "unknown" {
		t.Errorf("expected unknown, got %s", h.Status)
	}
}

// ============================================================
// Cache handler tests
// ============================================================

func TestHandleGetCacheStats_MethodNotAllowed(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/stats", nil)
	w := httptest.NewRecorder()
	s.handleGetCacheStats(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGetCacheStats_Unauthorized(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cache/stats", nil)
	// No X-User-ID header
	w := httptest.NewRecorder()
	s.handleGetCacheStats(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleGetCacheStats_NilCache(t *testing.T) {
	s := newTestServer() // no cache
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cache/stats", nil)
	req.Header.Set("X-User-ID", "admin")
	w := httptest.NewRecorder()
	s.handleGetCacheStats(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestHandleGetCacheStats_Success(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cache/stats", nil)
	req.Header.Set("X-User-ID", "admin")
	w := httptest.NewRecorder()
	s.handleGetCacheStats(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var stats cache.Stats
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
}

func TestHandleGetCacheConfig_MethodNotAllowed(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/config", nil)
	w := httptest.NewRecorder()
	s.handleGetCacheConfig(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGetCacheConfig_Forbidden(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cache/config", nil)
	req.Header.Set("X-Role", "viewer") // not admin
	w := httptest.NewRecorder()
	s.handleGetCacheConfig(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestHandleGetCacheConfig_NilCache(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cache/config", nil)
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleGetCacheConfig(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestHandleGetCacheConfig_Success(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cache/config", nil)
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleGetCacheConfig(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleClearCache_MethodNotAllowed(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cache/clear", nil)
	w := httptest.NewRecorder()
	s.handleClearCache(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleClearCache_Forbidden(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/clear", nil)
	req.Header.Set("X-Role", "viewer")
	w := httptest.NewRecorder()
	s.handleClearCache(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestHandleClearCache_NilCache(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/clear", nil)
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleClearCache(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestHandleClearCache_Success(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/clear", nil)
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleClearCache(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleInvalidateCache_MethodNotAllowed(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cache/invalidate", nil)
	w := httptest.NewRecorder()
	s.handleInvalidateCache(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleInvalidateCache_Forbidden(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/invalidate", nil)
	req.Header.Set("X-Role", "viewer")
	w := httptest.NewRecorder()
	s.handleInvalidateCache(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestHandleInvalidateCache_NilCache(t *testing.T) {
	s := newTestServer()
	body := `{"type":"provider","value":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/invalidate", strings.NewReader(body))
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleInvalidateCache(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestHandleInvalidateCache_InvalidBody(t *testing.T) {
	s := newTestServerWithCache()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/invalidate", strings.NewReader("not-json"))
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleInvalidateCache(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleInvalidateCache_InvalidType(t *testing.T) {
	s := newTestServerWithCache()
	body := `{"type":"unknown","value":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/invalidate", strings.NewReader(body))
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleInvalidateCache(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleInvalidateCache_ByProvider(t *testing.T) {
	s := newTestServerWithCache()
	body := `{"type":"provider","value":"prov-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/invalidate", strings.NewReader(body))
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleInvalidateCache(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleInvalidateCache_ByModel(t *testing.T) {
	s := newTestServerWithCache()
	body := `{"type":"model","value":"gpt-4"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/invalidate", strings.NewReader(body))
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleInvalidateCache(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleInvalidateCache_ByAge(t *testing.T) {
	s := newTestServerWithCache()
	body := `{"type":"age","value":"1h"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/invalidate", strings.NewReader(body))
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleInvalidateCache(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleInvalidateCache_ByAge_InvalidDuration(t *testing.T) {
	s := newTestServerWithCache()
	body := `{"type":"age","value":"notaduration"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/invalidate", strings.NewReader(body))
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleInvalidateCache(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleInvalidateCache_ByPattern(t *testing.T) {
	s := newTestServerWithCache()
	body := `{"type":"pattern","value":"test*"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cache/invalidate", strings.NewReader(body))
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	s.handleInvalidateCache(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ============================================================
// CacheToCacheConfig utility test
// ============================================================

func TestCacheToCacheConfig(t *testing.T) {
	c := &cache.Config{
		Enabled:       true,
		DefaultTTL:    time.Hour,
		MaxSize:       500,
		MaxMemoryMB:   256,
		CleanupPeriod: 10 * time.Minute,
	}
	m := CacheToCacheConfig(c)
	if m["enabled"] != true {
		t.Error("expected enabled=true")
	}
	if m["max_size"] != 500 {
		t.Error("expected max_size=500")
	}
}

// ============================================================
// Routing policies handler tests
// ============================================================

func TestHandleGetRoutingPolicies_GET(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routing/policies", nil)
	w := httptest.NewRecorder()
	s.handleGetRoutingPolicies(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var policies []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &policies); err != nil {
		t.Fatal(err)
	}
	if len(policies) != 4 {
		t.Errorf("expected 4 policies, got %d", len(policies))
	}
}

func TestHandleGetRoutingPolicies_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routing/policies", nil)
	w := httptest.NewRecorder()
	s.handleGetRoutingPolicies(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleSelectProvider_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routing/select", nil)
	w := httptest.NewRecorder()
	s.handleSelectProvider(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleSelectProvider_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routing/select", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleSelectProvider(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================
// Middleware tests
// ============================================================

func TestCORSMiddleware_Options(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AllowedOrigins: []string{"*"},
		},
	}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	handler := s.corsMiddleware(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called for OPTIONS preflight")
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected Access-Control-Allow-Origin: *")
	}
}

func TestCORSMiddleware_SpecificOrigin(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AllowedOrigins: []string{"http://allowed.com"},
		},
	}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.corsMiddleware(next)

	// Matching origin
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Origin", "http://allowed.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "http://allowed.com" {
		t.Error("expected matching origin header")
	}

	// Non-matching origin
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req2.Header.Set("Origin", "http://evil.com")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Header().Get("Access-Control-Allow-Origin") == "http://evil.com" {
		t.Error("should not set Access-Control-Allow-Origin for non-matching origin")
	}
}

func TestCORSMiddleware_AlwaysSetsAllowedMethods(t *testing.T) {
	cfg := &config.Config{}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.corsMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}
}

func TestAuthMiddleware_SkipsHealthPaths(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			EnableAuth: true,
		},
	}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := s.authMiddleware(next)

	paths := []string{
		"/api/v1/health",
		"/health",
		"/health/live",
		"/health/ready",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/",
		"/api/openapi.yaml",
		"/api/v1/events/stream",
		"/api/v1/chat/completions/stream",
		"/api/v1/chat/completions",
		"/api/v1/pair",
		"/static/js/app.js",
	}

	for _, p := range paths {
		called = false
		req := httptest.NewRequest(http.MethodGet, p, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if !called {
			t.Errorf("expected handler to be called for %s", p)
		}
	}
}

func TestAuthMiddleware_AuthDisabled(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			EnableAuth: false,
		},
	}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User-ID") != "admin" {
			t.Error("expected X-User-ID=admin when auth disabled")
		}
		if r.Header.Get("X-Username") != "admin" {
			t.Error("expected X-Username=admin when auth disabled")
		}
		if r.Header.Get("X-Role") != "admin" {
			t.Error("expected X-Role=admin when auth disabled")
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := s.authMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	s := newTestServer()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	handler := s.loggingMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ============================================================
// statusRecorder tests
// ============================================================

func TestStatusRecorder_WriteDefaultsTo200(t *testing.T) {
	w := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: w}
	sr.Write([]byte("hello"))
	if sr.statusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", sr.statusCode)
	}
}

func TestStatusRecorder_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: w}
	sr.WriteHeader(http.StatusNotFound)
	if sr.statusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", sr.statusCode)
	}
}

func TestStatusRecorder_Flush(t *testing.T) {
	w := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: w}
	// Should not panic when underlying writer supports Flusher
	sr.Flush()
}

// ============================================================
// getUserFromContext tests
// ============================================================

func TestGetUserFromContext(t *testing.T) {
	s := newTestServer()

	// With headers
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("X-Username", "alice")
	req.Header.Set("X-Role", "admin")

	user := s.getUserFromContext(req)
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.ID != "user-1" {
		t.Errorf("expected user-1, got %s", user.ID)
	}
	if user.Username != "alice" {
		t.Errorf("expected alice, got %s", user.Username)
	}
	if user.Role != "admin" {
		t.Errorf("expected admin, got %s", user.Role)
	}
	if !user.IsActive {
		t.Error("expected user to be active")
	}

	// Without headers
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	user2 := s.getUserFromContext(req2)
	if user2 != nil {
		t.Error("expected nil user when no headers set")
	}
}

// ============================================================
// Circuit breaker tests
// ============================================================

func TestIsAutoFileCircuitOpen_ClosedByDefault(t *testing.T) {
	s := newTestServer()
	if s.isAutoFileCircuitOpen() {
		t.Error("expected circuit to be closed initially")
	}
}

func TestRecordAutoFileResult_TripsAfterMaxFails(t *testing.T) {
	s := newTestServer()

	for i := 0; i < autoFileCBMaxFails; i++ {
		s.recordAutoFileResult(http.ErrAbortHandler)
	}

	if !s.isAutoFileCircuitOpen() {
		t.Error("expected circuit to be open after max failures")
	}

	// Success resets
	s.autoFileCircuitOpenAt = time.Now().Add(-(autoFileCBResetAfter + time.Second))
	// After cooldown, circuit should half-open and close on next check
	if s.isAutoFileCircuitOpen() {
		t.Error("expected circuit to be half-open after cooldown")
	}
}

func TestRecordAutoFileResult_SuccessResets(t *testing.T) {
	s := newTestServer()
	s.recordAutoFileResult(http.ErrAbortHandler)
	s.recordAutoFileResult(http.ErrAbortHandler)
	s.recordAutoFileResult(nil) // success resets
	if s.autoFileConsecFails != 0 {
		t.Errorf("expected 0 consecutive failures after success, got %d", s.autoFileConsecFails)
	}
}

func TestIsBeadSubsystemHealthy_NilApp(t *testing.T) {
	s := newTestServer()
	if s.isBeadSubsystemHealthy() {
		t.Error("expected unhealthy with nil app")
	}
}

func TestDefaultProjectID_NilApp(t *testing.T) {
	s := newTestServer()
	if s.defaultProjectID() != "" {
		t.Error("expected empty string with nil app")
	}
}

// ============================================================
// shouldThrottleFailure tests
// ============================================================

func TestShouldThrottleFailure_EmptyKey(t *testing.T) {
	s := newTestServer()
	if !s.shouldThrottleFailure("", 2*time.Minute) {
		t.Error("expected throttle for empty key")
	}
}

func TestShouldThrottleFailure_FirstCall(t *testing.T) {
	s := newTestServer()
	if s.shouldThrottleFailure("test-key", 2*time.Minute) {
		t.Error("expected no throttle on first call")
	}
}

func TestShouldThrottleFailure_SecondCallWithinWindow(t *testing.T) {
	s := newTestServer()
	s.shouldThrottleFailure("test-key", 2*time.Minute)
	if !s.shouldThrottleFailure("test-key", 2*time.Minute) {
		t.Error("expected throttle on second call within window")
	}
}

func TestShouldThrottleFailure_DifferentKeys(t *testing.T) {
	s := newTestServer()
	s.shouldThrottleFailure("key-1", 2*time.Minute)
	if s.shouldThrottleFailure("key-2", 2*time.Minute) {
		t.Error("expected no throttle for different key")
	}
}

// ============================================================
// recordAPIFailure tests (exercises code paths without actually filing)
// ============================================================

func TestRecordAPIFailure_SkipsNon5xxStatus(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	// Should not panic with nil app and status < 500
	s.recordAPIFailure(req, http.StatusOK)
	s.recordAPIFailure(req, http.StatusBadRequest)
	s.recordAPIFailure(req, http.StatusNotFound)
}

func TestRecordAPIFailure_NilRequest(t *testing.T) {
	s := newTestServer()
	// Should not panic
	s.recordAPIFailure(nil, 500)
}

func TestRecordAPIFailure_SkipsAutoFilePath(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/auto-file", nil)
	// Should not panic (skips due to auto-file path)
	s.recordAPIFailure(req, 500)
}

func TestRecordAPIFailure_NilApp(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads", nil)
	// Should return early because s.app == nil
	s.recordAPIFailure(req, 500)
}

// ============================================================
// Command handler tests (method checks without app)
// ============================================================

func TestHandleExecuteCommand_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/commands/execute", nil)
	w := httptest.NewRecorder()
	s.HandleExecuteCommand(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleExecuteCommand_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/commands/execute", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.HandleExecuteCommand(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleExecuteCommand_MissingCommand(t *testing.T) {
	s := newTestServer()
	body := `{"agent_id":"a1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/commands/execute", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.HandleExecuteCommand(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleExecuteCommand_MissingAgentID(t *testing.T) {
	s := newTestServer()
	body := `{"command":"ls"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/commands/execute", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.HandleExecuteCommand(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================
// Auto-file handler tests
// ============================================================

func TestHandleAutoFileBug_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/auto-file", nil)
	w := httptest.NewRecorder()
	s.HandleAutoFileBug(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleAutoFileBug_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/auto-file", strings.NewReader("notjson"))
	w := httptest.NewRecorder()
	s.HandleAutoFileBug(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestFormatContext(t *testing.T) {
	// Empty context
	result := formatContext(nil)
	if result != "No additional context" {
		t.Errorf("expected 'No additional context', got %s", result)
	}

	empty := formatContext(map[string]interface{}{})
	if empty != "No additional context" {
		t.Errorf("expected 'No additional context', got %s", empty)
	}

	// With data
	ctx := map[string]interface{}{"key": "value"}
	result2 := formatContext(ctx)
	if !strings.Contains(result2, "key") || !strings.Contains(result2, "value") {
		t.Errorf("expected context with key/value, got %s", result2)
	}
}

// ============================================================
// parseInt utility test
// ============================================================

func TestParseInt(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"10", 10},
		{"abc", 0},
		{"0", 0},
		{"-5", -5},
	}
	for _, tt := range tests {
		got := parseInt(tt.input)
		if got != tt.want {
			t.Errorf("parseInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// ============================================================
// getLogMeta utility test
// ============================================================

func TestGetLogMeta(t *testing.T) {
	// Nil metadata
	if getLogMeta(nil, "key") != "" {
		t.Error("expected empty for nil metadata")
	}

	// Missing key
	meta := map[string]interface{}{"a": "b"}
	if getLogMeta(meta, "missing") != "" {
		t.Error("expected empty for missing key")
	}

	// Non-string value
	meta2 := map[string]interface{}{"num": 42}
	if getLogMeta(meta2, "num") != "" {
		t.Error("expected empty for non-string value")
	}

	// Valid string value
	meta3 := map[string]interface{}{"agent_id": "agent-1"}
	if getLogMeta(meta3, "agent_id") != "agent-1" {
		t.Error("expected agent-1")
	}
}

// ============================================================
// getVersion and getInstanceID tests
// ============================================================

func TestGetVersion(t *testing.T) {
	v := getVersion()
	if v == "" {
		t.Error("expected non-empty version")
	}
}

func TestGetInstanceID(t *testing.T) {
	id := getInstanceID()
	if id == "" {
		t.Error("expected non-empty instance ID")
	}
}

// ============================================================
// handleHealth (the /api/v1/health endpoint) additional tests
// ============================================================

func TestHandleHealth_MethodNotAllowed_DELETE(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleHealth_GET_ResponseStructure(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	var m map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["status"] != "ok" {
		t.Errorf("expected ok, got %s", m["status"])
	}
}

// ============================================================
// Federation handler tests (without app - method checking)
// ============================================================

func TestHandleFederationStatus_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/status", nil)
	w := httptest.NewRecorder()
	s.handleFederationStatus(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleFederationSync_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/federation/sync", nil)
	w := httptest.NewRecorder()
	s.handleFederationSync(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleFederationSync_Disabled(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/sync", nil)
	w := httptest.NewRecorder()
	s.handleFederationSync(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (federation disabled), got %d", w.Code)
	}
}

// ============================================================
// Streaming and pair handler tests (method check + validation)
// ============================================================

func TestHandleStreamChatCompletion_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/completions/stream", nil)
	w := httptest.NewRecorder()
	s.handleStreamChatCompletion(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleStreamChatCompletion_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions/stream", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleStreamChatCompletion(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleStreamChatCompletion_MissingFields(t *testing.T) {
	s := newTestServer()
	body := `{"provider_id":"","messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions/stream", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleStreamChatCompletion(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleChatCompletion_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/completions", nil)
	w := httptest.NewRecorder()
	s.handleChatCompletion(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleChatCompletion_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleChatCompletion(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleChatCompletion_MissingFields(t *testing.T) {
	s := newTestServer()
	body := `{"provider_id":"","messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleChatCompletion(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlePairChat_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pair", nil)
	w := httptest.NewRecorder()
	s.handlePairChat(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandlePairChat_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pair", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handlePairChat(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlePairChat_MissingFields(t *testing.T) {
	s := newTestServer()
	body := `{"agent_id":"","bead_id":"","message":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pair", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handlePairChat(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================
// Work and REPL handler tests
// ============================================================

func TestHandleWork_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/work", nil)
	w := httptest.NewRecorder()
	s.handleWork(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleWork_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/work", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleWork(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleWork_MissingFields(t *testing.T) {
	s := newTestServer()
	body := `{"agent_id":"","project_id":"","prompt":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/work", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleWork(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleRepl_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/repl", nil)
	w := httptest.NewRecorder()
	s.handleRepl(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleRepl_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/repl", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleRepl(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleRepl_MissingMessage(t *testing.T) {
	s := newTestServer()
	body := `{"message":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/repl", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleRepl(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================
// Git handler method checks
// ============================================================

func TestHandleGitSync_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/git/sync", nil)
	w := httptest.NewRecorder()
	s.handleGitSync(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGitSync_MissingProjectID(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/git/sync", nil)
	w := httptest.NewRecorder()
	s.handleGitSync(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleGitCommit_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/git/commit", nil)
	w := httptest.NewRecorder()
	s.handleGitCommit(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGitCommit_MissingProjectID(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/git/commit", nil)
	w := httptest.NewRecorder()
	s.handleGitCommit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleGitCommit_InvalidBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/git/commit?project_id=p1", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleGitCommit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleGitCommit_MissingMessage(t *testing.T) {
	s := newTestServer()
	body := `{"message":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/git/commit?project_id=p1", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleGitCommit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleGitPush_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/git/push", nil)
	w := httptest.NewRecorder()
	s.handleGitPush(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGitPush_MissingProjectID(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/git/push", nil)
	w := httptest.NewRecorder()
	s.handleGitPush(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleGitStatus_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/git/status", nil)
	w := httptest.NewRecorder()
	s.handleGitStatus(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleGitStatus_MissingProjectID(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/git/status", nil)
	w := httptest.NewRecorder()
	s.handleGitStatus(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================
// Logs handler method checks
// ============================================================

func TestHandleLogsRecent_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/logs/recent", nil)
	w := httptest.NewRecorder()
	s.HandleLogsRecent(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleLogsStream_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/logs/stream", nil)
	w := httptest.NewRecorder()
	s.HandleLogsStream(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleLogsExport_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/logs/export", nil)
	w := httptest.NewRecorder()
	s.HandleLogsExport(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// Pair/streaming helper function tests
// ============================================================

func TestDefaultProjectID(t *testing.T) {
	if defaultProjectID("my-proj") != "my-proj" {
		t.Error("expected my-proj")
	}
	if defaultProjectID("") != "loom-self" {
		t.Error("expected loom-self for empty")
	}
}

func TestGetModelTokenLimit(t *testing.T) {
	if getModelTokenLimit("gpt-4") != 8192 {
		t.Error("expected 8192 for gpt-4")
	}
	if getModelTokenLimit("unknown-model") != 100000 {
		t.Error("expected 100000 default for unknown model")
	}
	if getModelTokenLimit("claude-3-opus") != 200000 {
		t.Error("expected 200000 for claude-3-opus")
	}
}

func TestBuildPairSystemPrompt_WithPersona(t *testing.T) {
	agent := &models.Agent{
		Name: "test-agent",
		Persona: &models.Persona{
			Character:            "a testing assistant",
			Mission:              "help test",
			Personality:          "friendly",
			Capabilities:         []string{"testing", "debugging"},
			AutonomyInstructions: "act carefully",
			DecisionInstructions: "think before acting",
		},
	}
	prompt := buildPairSystemPrompt(agent)
	if !strings.Contains(prompt, "a testing assistant") {
		t.Error("expected character in prompt")
	}
	if !strings.Contains(prompt, "help test") {
		t.Error("expected mission in prompt")
	}
	if !strings.Contains(prompt, "friendly") {
		t.Error("expected personality in prompt")
	}
	if !strings.Contains(prompt, "testing") {
		t.Error("expected capabilities in prompt")
	}
	if !strings.Contains(prompt, "act carefully") {
		t.Error("expected autonomy instructions in prompt")
	}
	if !strings.Contains(prompt, "think before acting") {
		t.Error("expected decision instructions in prompt")
	}
	if !strings.Contains(prompt, "Pair-Programming Mode") {
		t.Error("expected pair mode instruction")
	}
}

func TestBuildPairSystemPrompt_WithoutPersona(t *testing.T) {
	agent := &models.Agent{
		Name: "plain-agent",
	}
	prompt := buildPairSystemPrompt(agent)
	if !strings.Contains(prompt, "plain-agent") {
		t.Error("expected agent name in prompt")
	}
}

func TestApplyTokenLimits_UnderLimit(t *testing.T) {
	messages := []provider.ChatMessage{
		{Role: "system", Content: "hello"},
		{Role: "user", Content: "world"},
	}
	result := applyTokenLimits(messages, "gpt-4")
	if len(result) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result))
	}
}

func TestApplyTokenLimits_EmptyMessages(t *testing.T) {
	messages := []provider.ChatMessage{}
	result := applyTokenLimits(messages, "gpt-4")
	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestApplyTokenLimits_OverLimit(t *testing.T) {
	// gpt-3.5-turbo limit is 4096, 80% = 3276
	// Each message ~1000 tokens (4000 chars / 4)
	bigContent := strings.Repeat("x", 4000) // ~1000 tokens
	messages := []provider.ChatMessage{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: bigContent},
		{Role: "assistant", Content: bigContent},
		{Role: "user", Content: bigContent},
		{Role: "assistant", Content: bigContent},
		{Role: "user", Content: bigContent},
	}
	result := applyTokenLimits(messages, "gpt-3.5-turbo")
	// total tokens = 0 + 5*1000 = 5000 > 3276, so should truncate
	// The result should contain a truncation notice
	foundNotice := false
	for _, m := range result {
		if strings.Contains(m.Content, "truncated") {
			foundNotice = true
			break
		}
	}
	if !foundNotice {
		t.Error("expected truncation notice in messages")
	}
	// Should start with system message
	if result[0].Role != "system" {
		t.Error("expected first message to be system")
	}
}

func TestAppendActionPrompt(t *testing.T) {
	messages := []provider.ChatMessage{
		{Role: "user", Content: "hello"},
	}
	result := appendActionPrompt(messages)
	// The result should have a system message prepended (if action prompt is non-empty)
	if len(result) < 1 {
		t.Fatal("expected at least 1 message")
	}
}

// ============================================================
// SetupRoutes with WebUI disabled
// ============================================================

func TestSetupRoutes_DisabledWebUI(t *testing.T) {
	cfg := &config.Config{
		WebUI: config.WebUIConfig{
			Enabled: false,
		},
	}
	s := NewServer(nil, nil, nil, cfg)
	handler := s.SetupRoutes()

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	// Health should still work
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for health, got %d", w.Code)
	}
}

// ============================================================
// GetMetrics test
// ============================================================

func TestGetMetrics(t *testing.T) {
	cfg := &config.Config{}
	s := NewServer(nil, nil, nil, cfg)
	m := s.GetMetrics()
	if m == nil {
		t.Error("expected non-nil metrics")
	}
}

// ============================================================
// NewServer with cache config tests
// ============================================================

func TestNewServer_WithCacheEnabled(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:       true,
			DefaultTTL:    time.Hour,
			MaxSize:       1000,
			CleanupPeriod: time.Minute,
		},
	}
	s := NewServer(nil, nil, nil, cfg)
	if s.cache == nil {
		t.Error("expected cache to be initialized when enabled")
	}
}

func TestNewServer_WithCacheDefaultValues(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: true,
			// Leave defaults at zero to test default filling
		},
	}
	s := NewServer(nil, nil, nil, cfg)
	if s.cache == nil {
		t.Error("expected cache to be initialized with defaults")
	}
}

func TestNewServer_WithRedisBackendFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis fallback test in short mode")
	}
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:  true,
			Backend:  "redis",
			RedisURL: "redis://127.0.0.1:1", // Port 1 - will fail immediately
		},
	}
	s := NewServer(nil, nil, nil, cfg)
	// Should fallback to in-memory cache
	if s.cache == nil {
		t.Error("expected cache to fallback to in-memory when redis fails")
	}
}

func TestNewServer_EmptyConfig(t *testing.T) {
	s := NewServer(nil, nil, nil, &config.Config{})
	if s == nil {
		t.Fatal("expected non-nil server even with empty config")
	}
	if s.apiFailureLast == nil {
		t.Error("expected apiFailureLast map to be initialized")
	}
}

// ============================================================
// ChangePasswordRequest struct test
// ============================================================

func TestChangePasswordRequest_JSON(t *testing.T) {
	body := `{"old_password":"old","new_password":"new"}`
	var req ChangePasswordRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if req.OldPassword != "old" || req.NewPassword != "new" {
		t.Errorf("unexpected values: %+v", req)
	}
}

// ============================================================
// Webhook additional tests (covering more code paths)
// ============================================================

func TestHandleWebhookStatus_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/status", nil)
	w := httptest.NewRecorder()
	s.handleWebhookStatus(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleWebhookStatus_NoSecret(t *testing.T) {
	cfg := &config.Config{}
	s := &Server{config: cfg, apiFailureLast: make(map[string]time.Time)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/status", nil)
	w := httptest.NewRecorder()
	s.handleWebhookStatus(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["webhook_secret_configured"] != false {
		t.Error("expected webhook_secret_configured=false")
	}
}

func TestHandleGitHubWebhook_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/github", nil)
	w := httptest.NewRecorder()
	s.handleGitHubWebhook(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ============================================================
// File handler tests (without fileManager)
// ============================================================

func TestHandleProjectFiles_NilFileManager(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/files/read?path=test.txt", nil)
	w := httptest.NewRecorder()
	s.handleProjectFiles(w, req, "p1", []string{"read"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 (nil file manager), got %d", w.Code)
	}
}

func TestHandleProjectFiles_NoAction(t *testing.T) {
	s := newTestServer()
	s.fileManager = nil
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/files/", nil)
	w := httptest.NewRecorder()
	s.handleProjectFiles(w, req, "p1", []string{""})
	// Should return 404 or 500 (file manager not configured first, then action required)
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
		t.Fatalf("expected error status, got %d", w.Code)
	}
}

// ============================================================
// Streaming types (StreamChatCompletionRequest)
// ============================================================

func TestStreamChatCompletionRequest_JSON(t *testing.T) {
	body := `{"provider_id":"p1","messages":[{"role":"user","content":"hi"}]}`
	var req StreamChatCompletionRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if req.ProviderID != "p1" {
		t.Error("expected p1")
	}
	if len(req.Messages) != 1 {
		t.Error("expected 1 message")
	}
}

func TestAutoFileBugRequest_JSON(t *testing.T) {
	body := `{"title":"test bug","source":"frontend","error_type":"js","message":"oops","severity":"high"}`
	var req AutoFileBugRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if req.Title != "test bug" {
		t.Error("expected test bug title")
	}
	if req.Source != "frontend" {
		t.Error("expected frontend source")
	}
}

func TestPairChatRequest_JSON(t *testing.T) {
	body := `{"agent_id":"a1","bead_id":"b1","message":"hello"}`
	var req PairChatRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if req.AgentID != "a1" || req.BeadID != "b1" || req.Message != "hello" {
		t.Error("unexpected values")
	}
}

// ============================================================
// ProviderRequest struct test
// ============================================================

func TestProviderRequest_JSON(t *testing.T) {
	body := `{"id":"p1","name":"OpenAI","type":"openai","endpoint":"https://api.openai.com","api_key":"sk-123","model":"gpt-4"}`
	var req ProviderRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if req.ID != "p1" {
		t.Error("expected p1")
	}
	if req.APIKey != "sk-123" {
		t.Error("expected api key")
	}
}

// ============================================================
// HealthStatus struct tests
// ============================================================

func TestHealthStatus_JSON(t *testing.T) {
	hs := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    100,
		Version:   "1.0",
		Dependencies: map[string]DepHealth{
			"db": {Status: "healthy", Message: "ok"},
		},
	}
	data, err := json.Marshal(hs)
	if err != nil {
		t.Fatal(err)
	}
	var decoded HealthStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Status != "healthy" {
		t.Error("expected healthy")
	}
}

func TestDepHealth_JSON(t *testing.T) {
	dh := DepHealth{Status: "unhealthy", Message: "timeout", Latency: 500}
	data, err := json.Marshal(dh)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "unhealthy") {
		t.Error("expected unhealthy in JSON")
	}
}

// ============================================================
// MotivationResponse struct test
// ============================================================

func TestMotivationResponse_JSON(t *testing.T) {
	mr := MotivationResponse{
		ID:          "m1",
		Name:        "test",
		Description: "test motivation",
		Type:        "scheduled",
		Status:      "active",
	}
	data, err := json.Marshal(mr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "m1") {
		t.Error("expected m1 in JSON")
	}
}
