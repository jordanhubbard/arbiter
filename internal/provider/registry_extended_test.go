package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Registry: Upsert
// ---------------------------------------------------------------------------

func TestRegistryUpsert_NewProvider(t *testing.T) {
	r := NewRegistry()

	err := r.Upsert(&ProviderConfig{
		ID:       "upsert-new",
		Name:     "New",
		Type:     "mock",
		Endpoint: "",
		Model:    "m",
	})
	if err != nil {
		t.Fatalf("Upsert new: %v", err)
	}

	got, err := r.Get("upsert-new")
	if err != nil {
		t.Fatalf("Get after upsert: %v", err)
	}
	if got.Config.Name != "New" {
		t.Errorf("name = %q, want %q", got.Config.Name, "New")
	}
	// Should default status to "pending"
	if got.Config.Status != "pending" {
		t.Errorf("status = %q, want %q", got.Config.Status, "pending")
	}
}

func TestRegistryUpsert_ReplaceExisting(t *testing.T) {
	r := NewRegistry()

	_ = r.Upsert(&ProviderConfig{ID: "up1", Name: "V1", Type: "mock", Model: "m"})
	_ = r.Upsert(&ProviderConfig{ID: "up1", Name: "V2", Type: "mock", Model: "m"})

	got, _ := r.Get("up1")
	if got.Config.Name != "V2" {
		t.Errorf("expected replaced name V2, got %q", got.Config.Name)
	}
}

func TestRegistryUpsert_PreservesStatus(t *testing.T) {
	r := NewRegistry()

	err := r.Upsert(&ProviderConfig{ID: "up2", Type: "mock", Model: "m", Status: "healthy"})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, _ := r.Get("up2")
	if got.Config.Status != "healthy" {
		t.Errorf("status = %q, want %q", got.Config.Status, "healthy")
	}
}

func TestRegistryUpsert_UnsupportedType(t *testing.T) {
	r := NewRegistry()
	err := r.Upsert(&ProviderConfig{ID: "bad", Type: "badtype", Model: "m"})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
	if !strings.Contains(err.Error(), "unsupported provider type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRegistryUpsert_AllTypes(t *testing.T) {
	types := []string{"openai", "anthropic", "local", "custom", "vllm", "ollama", "mock"}
	r := NewRegistry()
	for _, tp := range types {
		err := r.Upsert(&ProviderConfig{
			ID:       "t-" + tp,
			Type:     tp,
			Endpoint: "http://localhost",
			Model:    "m",
		})
		if err != nil {
			t.Errorf("Upsert(%q): %v", tp, err)
		}
	}
	if len(r.List()) != len(types) {
		t.Errorf("expected %d providers, got %d", len(types), len(r.List()))
	}
}

// ---------------------------------------------------------------------------
// Registry: Clear
// ---------------------------------------------------------------------------

func TestRegistryClear(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&ProviderConfig{ID: "a", Type: "mock", Model: "m"})
	_ = r.Register(&ProviderConfig{ID: "b", Type: "mock", Model: "m"})
	if len(r.List()) != 2 {
		t.Fatalf("expected 2 before Clear, got %d", len(r.List()))
	}
	r.Clear()
	if len(r.List()) != 0 {
		t.Errorf("expected 0 after Clear, got %d", len(r.List()))
	}
}

// ---------------------------------------------------------------------------
// Registry: Get not found
// ---------------------------------------------------------------------------

func TestRegistryGet_NotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Registry: Unregister not found
// ---------------------------------------------------------------------------

func TestRegistryUnregister_NotFound(t *testing.T) {
	r := NewRegistry()
	err := r.Unregister("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}

// ---------------------------------------------------------------------------
// Registry: IsActive edge cases
// ---------------------------------------------------------------------------

func TestRegistryIsActive_NonExistent(t *testing.T) {
	r := NewRegistry()
	if r.IsActive("nope") {
		t.Error("IsActive should be false for nonexistent provider")
	}
}

func TestRegistryIsActive_DisabledStatus(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{ID: "dis", Type: "mock", Model: "m", Status: "disabled"})
	if r.IsActive("dis") {
		t.Error("IsActive should be false for disabled")
	}
}

func TestRegistryIsActive_ErrorStatus(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{ID: "err", Type: "mock", Model: "m", Status: "error"})
	if r.IsActive("err") {
		t.Error("IsActive should be false for error status")
	}
}

// ---------------------------------------------------------------------------
// Registry: SetMetricsCallback + SendChatCompletion
// ---------------------------------------------------------------------------

func TestRegistrySendChatCompletion_Mock(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "mock1", Type: "mock", Model: "mock-model", Status: "healthy",
	})

	req := &ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hello"}},
	}

	resp, err := r.SendChatCompletion(context.Background(), "mock1", req)
	if err != nil {
		t.Fatalf("SendChatCompletion: %v", err)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("expected at least one choice")
	}
	if !strings.Contains(resp.Choices[0].Message.Content, "[mock]") {
		t.Errorf("expected mock prefix, got %q", resp.Choices[0].Message.Content)
	}
	// Model should default from config
	if req.Model != "mock-model" {
		t.Errorf("expected model to default to mock-model, got %q", req.Model)
	}
}

func TestRegistrySendChatCompletion_NotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.SendChatCompletion(context.Background(), "nope", &ChatCompletionRequest{})
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestRegistrySendChatCompletion_Disabled(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "dis", Type: "mock", Model: "m", Status: "disabled",
	})
	_, err := r.SendChatCompletion(context.Background(), "dis", &ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for disabled provider")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("expected 'disabled' in error, got %v", err)
	}
}

func TestRegistrySendChatCompletion_MetricsCallback(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "cb", Type: "mock", Model: "m", Status: "healthy",
	})

	var called bool
	var cbProviderID string
	var cbSuccess bool

	r.SetMetricsCallback(func(providerID string, success bool, latencyMs int64, totalTokens int64) {
		called = true
		cbProviderID = providerID
		cbSuccess = success
	})

	_, err := r.SendChatCompletion(context.Background(), "cb", &ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("metrics callback was not called")
	}
	if cbProviderID != "cb" {
		t.Errorf("callback providerID = %q, want %q", cbProviderID, "cb")
	}
	if !cbSuccess {
		t.Error("callback success = false, want true")
	}
}

// ---------------------------------------------------------------------------
// Registry: SendChatCompletionStream
// ---------------------------------------------------------------------------

func TestRegistrySendChatCompletionStream_Mock(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "sm", Type: "mock", Model: "mock-model", Status: "healthy",
	})

	var chunks []*StreamChunk
	err := r.SendChatCompletionStream(context.Background(), "sm", &ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("SendChatCompletionStream: %v", err)
	}
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestRegistrySendChatCompletionStream_NotFound(t *testing.T) {
	r := NewRegistry()
	err := r.SendChatCompletionStream(context.Background(), "nope", &ChatCompletionRequest{}, func(chunk *StreamChunk) error { return nil })
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestRegistrySendChatCompletionStream_MetricsCallback(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "scb", Type: "mock", Model: "m", Status: "healthy",
	})

	var called bool
	r.SetMetricsCallback(func(providerID string, success bool, latencyMs int64, totalTokens int64) {
		called = true
	})

	_ = r.SendChatCompletionStream(context.Background(), "scb", &ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "x"}},
	}, func(chunk *StreamChunk) error { return nil })

	if !called {
		t.Error("metrics callback was not called for streaming")
	}
}

// ---------------------------------------------------------------------------
// Registry: GetModels
// ---------------------------------------------------------------------------

func TestRegistryGetModels_Mock(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{ID: "gm", Type: "mock", Model: "m"})

	models, err := r.GetModels(context.Background(), "gm")
	if err != nil {
		t.Fatalf("GetModels: %v", err)
	}
	if len(models) == 0 {
		t.Error("expected at least one model from mock")
	}
	if models[0].ID != "mock-model" {
		t.Errorf("model ID = %q, want %q", models[0].ID, "mock-model")
	}
}

func TestRegistryGetModels_NotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.GetModels(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}

// ---------------------------------------------------------------------------
// Registry: GetScorer, SetScoringWeights, GetScoringWeights
// ---------------------------------------------------------------------------

func TestRegistryGetScorer(t *testing.T) {
	r := NewRegistry()
	s := r.GetScorer()
	if s == nil {
		t.Fatal("GetScorer returned nil")
	}
}

func TestRegistrySetGetScoringWeights(t *testing.T) {
	r := NewRegistry()

	custom := ScoringWeights{
		ModelSize:      5.0,
		RoundTrip:      4.0,
		RequestLatency: 3.0,
		Cost:           2.0,
	}
	r.SetScoringWeights(custom)
	got := r.GetScoringWeights()
	if got.ModelSize != 5.0 || got.Cost != 2.0 {
		t.Errorf("weights not set correctly: %+v", got)
	}
}

func TestRegistryGetScoringWeights_NilScorer(t *testing.T) {
	// Construct a registry with nil scorer to test fallback
	r := &Registry{
		providers: make(map[string]*RegisteredProvider),
		scorer:    nil,
	}
	w := r.GetScoringWeights()
	d := DefaultWeights()
	if w.ModelSize != d.ModelSize {
		t.Errorf("expected default weights when scorer is nil")
	}
}

func TestRegistrySetScoringWeights_NilScorer(t *testing.T) {
	r := &Registry{
		providers: make(map[string]*RegisteredProvider),
		scorer:    nil,
	}
	// Should not panic
	r.SetScoringWeights(ScoringWeights{ModelSize: 1})
}

// ---------------------------------------------------------------------------
// Registry: UpdateProviderScore
// ---------------------------------------------------------------------------

func TestRegistryUpdateProviderScore(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "ups", Type: "mock", Model: "m", Status: "healthy",
	})

	r.UpdateProviderScore("ups", 70.0, 2.0)

	p, _ := r.Get("ups")
	if p.Config.ModelParamsB != 70.0 {
		t.Errorf("ModelParamsB = %f, want 70", p.Config.ModelParamsB)
	}
	if p.Config.CostPerMToken != 2.0 {
		t.Errorf("CostPerMToken = %f, want 2", p.Config.CostPerMToken)
	}
	if p.Config.CapabilityScore <= 0 {
		t.Error("CapabilityScore should be > 0 after update")
	}
}

func TestRegistryUpdateProviderScore_NonExistent(t *testing.T) {
	r := NewRegistry()
	// Should not panic
	r.UpdateProviderScore("nope", 30, 0)
}

// ---------------------------------------------------------------------------
// Registry: RecordRequestMetrics
// ---------------------------------------------------------------------------

func TestRegistryRecordRequestMetrics_Success(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "rm", Type: "mock", Model: "m", Status: "healthy",
	})

	r.RecordRequestMetrics("rm", 500, true)

	p, _ := r.Get("rm")
	if p.Config.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want 1", p.Config.TotalRequests)
	}
	if p.Config.SuccessRequests != 1 {
		t.Errorf("SuccessRequests = %d, want 1", p.Config.SuccessRequests)
	}
	if p.Config.AvgLatencyMs != 500.0 {
		t.Errorf("AvgLatencyMs = %f, want 500", p.Config.AvgLatencyMs)
	}
}

func TestRegistryRecordRequestMetrics_Failure(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "rmf", Type: "mock", Model: "m", Status: "healthy",
	})

	r.RecordRequestMetrics("rmf", 200, false)

	p, _ := r.Get("rmf")
	if p.Config.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want 1", p.Config.TotalRequests)
	}
	if p.Config.SuccessRequests != 0 {
		t.Errorf("SuccessRequests = %d, want 0", p.Config.SuccessRequests)
	}
}

func TestRegistryRecordRequestMetrics_RollingAverage(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "rma", Type: "mock", Model: "m", Status: "healthy",
	})

	r.RecordRequestMetrics("rma", 1000, true)
	r.RecordRequestMetrics("rma", 500, true)

	p, _ := r.Get("rma")
	// EMA: first=1000, second = 0.8*1000 + 0.2*500 = 900
	expected := 900.0
	if p.Config.AvgLatencyMs != expected {
		t.Errorf("AvgLatencyMs = %f, want %f", p.Config.AvgLatencyMs, expected)
	}
}

func TestRegistryRecordRequestMetrics_NonExistent(t *testing.T) {
	r := NewRegistry()
	// Should not panic
	r.RecordRequestMetrics("nope", 100, true)
}

// ---------------------------------------------------------------------------
// Registry: UpdateHeartbeatLatency
// ---------------------------------------------------------------------------

func TestRegistryUpdateHeartbeatLatency(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "hb", Type: "mock", Model: "m", Status: "healthy",
	})

	r.UpdateHeartbeatLatency("hb", 250)

	p, _ := r.Get("hb")
	if p.Config.LastHeartbeatLatencyMs != 250 {
		t.Errorf("LastHeartbeatLatencyMs = %d, want 250", p.Config.LastHeartbeatLatencyMs)
	}
	if p.Config.LastHeartbeatAt.IsZero() {
		t.Error("LastHeartbeatAt should be set")
	}
	if p.Config.CapabilityScore <= 0 {
		t.Error("CapabilityScore should be > 0 after heartbeat update")
	}
}

func TestRegistryUpdateHeartbeatLatency_NonExistent(t *testing.T) {
	r := NewRegistry()
	// Should not panic
	r.UpdateHeartbeatLatency("nope", 100)
}

// ---------------------------------------------------------------------------
// Registry: ListActiveForComplexity
// ---------------------------------------------------------------------------

func TestRegistryListActiveForComplexity(t *testing.T) {
	r := NewRegistry()

	// Register providers of different sizes
	_ = r.Upsert(&ProviderConfig{ID: "s", Type: "mock", Model: "m", Status: "healthy"})
	_ = r.Upsert(&ProviderConfig{ID: "m", Type: "mock", Model: "m", Status: "healthy"})
	_ = r.Upsert(&ProviderConfig{ID: "l", Type: "mock", Model: "m", Status: "healthy"})

	// Give them different model sizes
	r.GetScorer().UpdateProviderMetrics("s", 7, 100, 500, 0)
	r.GetScorer().UpdateProviderMetrics("m", 32, 100, 500, 0)
	r.GetScorer().UpdateProviderMetrics("l", 70, 100, 500, 0)

	// Simple task should prefer small
	providers := r.ListActiveForComplexity(ComplexitySimple)
	if len(providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(providers))
	}
	if providers[0].Config.ID != "s" {
		t.Errorf("expected small first for simple task, got %q", providers[0].Config.ID)
	}
}

func TestRegistryListActiveForComplexity_NoProviders(t *testing.T) {
	r := NewRegistry()
	providers := r.ListActiveForComplexity(ComplexityMedium)
	if len(providers) != 0 {
		t.Errorf("expected empty list, got %d", len(providers))
	}
}

func TestRegistryListActiveForComplexity_SingleProvider(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{ID: "only", Type: "mock", Model: "m", Status: "healthy"})
	providers := r.ListActiveForComplexity(ComplexityMedium)
	if len(providers) != 1 {
		t.Errorf("expected 1, got %d", len(providers))
	}
}

func TestRegistryListActiveForComplexity_ExcludesInactive(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{ID: "active", Type: "mock", Model: "m", Status: "healthy"})
	_ = r.Upsert(&ProviderConfig{ID: "pending", Type: "mock", Model: "m", Status: "pending"})
	providers := r.ListActiveForComplexity(ComplexitySimple)
	if len(providers) != 1 {
		t.Errorf("expected 1 active, got %d", len(providers))
	}
}

// ---------------------------------------------------------------------------
// Registry: SelectProviderForComplexity
// ---------------------------------------------------------------------------

func TestRegistrySelectProviderForComplexity(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{ID: "a", Type: "mock", Model: "m", Status: "healthy"})
	r.GetScorer().UpdateProviderMetrics("a", 30, 100, 500, 0)

	p, score, found := r.SelectProviderForComplexity(ComplexityMedium)
	if !found {
		t.Fatal("expected to find a provider")
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	// Score may be zero for a fresh provider; just verify selection succeeded
	_ = score
}

func TestRegistrySelectProviderForComplexity_NoneAvailable(t *testing.T) {
	r := NewRegistry()
	_, _, found := r.SelectProviderForComplexity(ComplexityMedium)
	if found {
		t.Error("expected found=false with no providers")
	}
}

// ---------------------------------------------------------------------------
// Registry: GetComplexityEstimator
// ---------------------------------------------------------------------------

func TestRegistryGetComplexityEstimator(t *testing.T) {
	r := NewRegistry()
	ce := r.GetComplexityEstimator()
	if ce == nil {
		t.Fatal("GetComplexityEstimator returned nil")
	}
	// Quick sanity check
	level := ce.EstimateComplexity("review code", "")
	if level != ComplexitySimple {
		t.Errorf("unexpected complexity: %s", level.String())
	}
}

// ---------------------------------------------------------------------------
// Registry: ListActive round-robin behavior
// ---------------------------------------------------------------------------

func TestRegistryListActive_RoundRobin(t *testing.T) {
	r := NewRegistry()

	// Register two providers with equal scores (same model, same latency)
	_ = r.Upsert(&ProviderConfig{ID: "rr1", Type: "mock", Model: "m", Status: "healthy"})
	_ = r.Upsert(&ProviderConfig{ID: "rr2", Type: "mock", Model: "m", Status: "healthy"})

	// Give them identical scores
	r.GetScorer().UpdateProviderMetrics("rr1", 30, 100, 500, 0)
	r.GetScorer().UpdateProviderMetrics("rr2", 30, 100, 500, 0)

	// Call ListActive multiple times - should rotate first provider
	firstIDs := make(map[string]int)
	for i := 0; i < 10; i++ {
		active := r.ListActive()
		if len(active) < 2 {
			t.Fatalf("expected 2 active, got %d", len(active))
		}
		firstIDs[active[0].Config.ID]++
	}

	// Both should appear as first at least once (round robin)
	if firstIDs["rr1"] == 0 || firstIDs["rr2"] == 0 {
		t.Errorf("round robin not working: rr1=%d rr2=%d", firstIDs["rr1"], firstIDs["rr2"])
	}
}

func TestRegistryListActive_NoActive(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{ID: "p", Type: "mock", Model: "m", Status: "pending"})
	active := r.ListActive()
	if len(active) != 0 {
		t.Errorf("expected 0 active, got %d", len(active))
	}
}

func TestRegistryListActive_SingleProvider(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{ID: "solo", Type: "mock", Model: "m", Status: "healthy"})
	active := r.ListActive()
	if len(active) != 1 {
		t.Errorf("expected 1 active, got %d", len(active))
	}
}

func TestRegistryListActive_SortsByScore(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{ID: "big", Type: "mock", Model: "m", Status: "healthy"})
	_ = r.Upsert(&ProviderConfig{ID: "small", Type: "mock", Model: "m", Status: "healthy"})

	// Give them very different scores so round-robin doesn't interfere
	r.GetScorer().UpdateProviderMetrics("big", 480, 100, 500, 0)
	r.GetScorer().UpdateProviderMetrics("small", 7, 100, 500, 0)

	active := r.ListActive()
	if len(active) < 2 {
		t.Fatalf("expected 2, got %d", len(active))
	}
	if active[0].Config.ID != "big" {
		t.Errorf("expected big first, got %q", active[0].Config.ID)
	}
}

// ---------------------------------------------------------------------------
// Registry: Concurrent access
// ---------------------------------------------------------------------------

func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(&ProviderConfig{
		ID: "conc", Type: "mock", Model: "m", Status: "healthy",
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("conc-%d", i)
			_ = r.Upsert(&ProviderConfig{
				ID: id, Type: "mock", Model: "m", Status: "healthy",
			})
			r.ListActive()
			r.RecordRequestMetrics(id, 100, true)
			r.UpdateHeartbeatLatency(id, 50)
			r.IsActive(id)
			_, _ = r.Get(id)
		}(i)
	}
	wg.Wait()

	// Should have at least the original + some concurrent ones
	if len(r.List()) < 1 {
		t.Error("expected at least 1 provider after concurrent access")
	}
}

// ---------------------------------------------------------------------------
// Registry: SendChatCompletion with httptest server (OpenAI protocol)
// ---------------------------------------------------------------------------

func TestRegistrySendChatCompletion_OpenAI_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id": "test-1",
				"object": "chat.completion",
				"model": "test-model",
				"choices": [{
					"index": 0,
					"message": {"role": "assistant", "content": "test response"},
					"finish_reason": "stop"
				}],
				"usage": {"prompt_tokens": 5, "completion_tokens": 10, "total_tokens": 15}
			}`))
		}
	}))
	defer server.Close()

	reg := NewRegistry()
	_ = reg.Upsert(&ProviderConfig{
		ID:       "oai",
		Type:     "openai",
		Endpoint: server.URL,
		Model:    "test-model",
		Status:   "healthy",
	})

	resp, err := reg.SendChatCompletion(context.Background(), "oai", &ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("SendChatCompletion: %v", err)
	}
	if resp.Choices[0].Message.Content != "test response" {
		t.Errorf("unexpected content: %q", resp.Choices[0].Message.Content)
	}
}

// ---------------------------------------------------------------------------
// Registry: SendChatCompletion with model rediscovery (404 retry)
// ---------------------------------------------------------------------------

func TestRegistrySendChatCompletion_404Retry(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chat/completions":
			callCount++
			if callCount == 1 {
				// First call returns 404 (model not found)
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error": "model not found"}`))
			} else {
				// Second call succeeds
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"id": "retry-1",
					"object": "chat.completion",
					"model": "new-model",
					"choices": [{
						"index": 0,
						"message": {"role": "assistant", "content": "retried ok"},
						"finish_reason": "stop"
					}],
					"usage": {"total_tokens": 10}
				}`))
			}
		case "/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data": [{"id": "new-model", "object": "model"}]}`))
		}
	}))
	defer server.Close()

	reg := NewRegistry()
	_ = reg.Upsert(&ProviderConfig{
		ID: "retry", Type: "openai", Endpoint: server.URL,
		Model: "old-model", Status: "healthy",
	})

	resp, err := reg.SendChatCompletion(context.Background(), "retry", &ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("expected retry to succeed: %v", err)
	}
	if resp.Choices[0].Message.Content != "retried ok" {
		t.Errorf("unexpected content: %q", resp.Choices[0].Message.Content)
	}
	// Verify model was updated
	p, _ := reg.Get("retry")
	if p.Config.Model != "new-model" {
		t.Errorf("model should be updated to new-model, got %q", p.Config.Model)
	}
}

// ---------------------------------------------------------------------------
// Registry: Register with default status
// ---------------------------------------------------------------------------

func TestRegistryRegister_DefaultStatus(t *testing.T) {
	r := NewRegistry()
	err := r.Register(&ProviderConfig{ID: "ds", Type: "mock", Model: "m"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	p, _ := r.Get("ds")
	if p.Config.Status != "pending" {
		t.Errorf("default status = %q, want %q", p.Config.Status, "pending")
	}
}

// ---------------------------------------------------------------------------
// Registry: Register ollama type
// ---------------------------------------------------------------------------

func TestRegistryRegister_OllamaType(t *testing.T) {
	r := NewRegistry()
	err := r.Register(&ProviderConfig{
		ID:       "ollama1",
		Type:     "ollama",
		Endpoint: "http://localhost:11434",
		Model:    "llama2",
	})
	if err != nil {
		t.Fatalf("Register ollama: %v", err)
	}
	p, _ := r.Get("ollama1")
	if _, ok := p.Protocol.(*OllamaProvider); !ok {
		t.Error("expected OllamaProvider protocol")
	}
}

// ---------------------------------------------------------------------------
// Registry: Register vllm type
// ---------------------------------------------------------------------------

func TestRegistryRegister_VLLMType(t *testing.T) {
	r := NewRegistry()
	err := r.Register(&ProviderConfig{
		ID:       "vllm1",
		Type:     "vllm",
		Endpoint: "http://localhost:8000/v1",
		Model:    "model",
	})
	if err != nil {
		t.Fatalf("Register vllm: %v", err)
	}
	p, _ := r.Get("vllm1")
	if _, ok := p.Protocol.(*OpenAIProvider); !ok {
		t.Error("expected OpenAIProvider protocol for vllm type")
	}
}

// ---------------------------------------------------------------------------
// isProviderHealthy helper
// ---------------------------------------------------------------------------

func TestIsProviderHealthy(t *testing.T) {
	cases := []struct {
		status  string
		healthy bool
	}{
		{"healthy", true},
		{"active", true},
		{"pending", false},
		{"disabled", false},
		{"error", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isProviderHealthy(c.status); got != c.healthy {
			t.Errorf("isProviderHealthy(%q) = %v, want %v", c.status, got, c.healthy)
		}
	}
}

// ---------------------------------------------------------------------------
// ContextLengthError detection in SendChatCompletion
// ---------------------------------------------------------------------------

func TestRegistrySendChatCompletion_ContextLengthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "context length exceeded: maximum is 4096 tokens"}`))
	}))
	defer server.Close()

	reg := NewRegistry()
	_ = reg.Upsert(&ProviderConfig{
		ID: "cle", Type: "openai", Endpoint: server.URL,
		Model: "m", Status: "healthy",
	})

	_, err := reg.SendChatCompletion(context.Background(), "cle", &ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var cle *ContextLengthError
	if !errors.As(err, &cle) {
		t.Errorf("expected ContextLengthError, got %T: %v", err, err)
	}
}
