package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/analytics"
)

// mockStorage implements analytics.Storage for testing
type mockStorage struct {
	logs     []*analytics.RequestLog
	err      error
	stats    *analytics.LogStats
	statsErr error
}

func (m *mockStorage) SaveLog(ctx context.Context, log *analytics.RequestLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockStorage) GetLogs(ctx context.Context, filter *analytics.LogFilter) ([]*analytics.RequestLog, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.logs, nil
}

func (m *mockStorage) GetLogStats(ctx context.Context, filter *analytics.LogFilter) (*analytics.LogStats, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.stats, nil
}

func (m *mockStorage) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func TestNewAnalyzer(t *testing.T) {
	storage := &mockStorage{}

	// Test with nil config (should use defaults)
	a := NewAnalyzer(storage, nil)
	if a == nil {
		t.Fatal("Expected non-nil analyzer")
	}
	if a.config == nil {
		t.Fatal("Expected non-nil config with nil input")
	}
	if a.config.MinOccurrences != 2 {
		t.Errorf("Expected default MinOccurrences=2, got %d", a.config.MinOccurrences)
	}

	// Test with custom config
	customConfig := &AnalysisConfig{
		TimeWindow:     24 * time.Hour,
		MinOccurrences: 5,
		MinSavingsUSD:  0.05,
	}
	a2 := NewAnalyzer(storage, customConfig)
	if a2.config.MinOccurrences != 5 {
		t.Errorf("Expected MinOccurrences=5, got %d", a2.config.MinOccurrences)
	}
}

func TestDefaultAnalysisConfig(t *testing.T) {
	config := DefaultAnalysisConfig()
	if config == nil {
		t.Fatal("Expected non-nil default config")
	}
	if config.TimeWindow != 7*24*time.Hour {
		t.Errorf("Expected 7 day time window, got %v", config.TimeWindow)
	}
	if config.MinOccurrences != 2 {
		t.Errorf("Expected MinOccurrences=2, got %d", config.MinOccurrences)
	}
	if config.MinSavingsUSD != 0.01 {
		t.Errorf("Expected MinSavingsUSD=0.01, got %f", config.MinSavingsUSD)
	}
	if config.AutoEnable != false {
		t.Error("Expected AutoEnable=false")
	}
	if config.AutoEnableMinUSD != 10.0 {
		t.Errorf("Expected AutoEnableMinUSD=10.0, got %f", config.AutoEnableMinUSD)
	}
	if config.AutoEnableMinRate != 0.5 {
		t.Errorf("Expected AutoEnableMinRate=0.5, got %f", config.AutoEnableMinRate)
	}
}

func TestAnalyze_EmptyLogs(t *testing.T) {
	storage := &mockStorage{logs: []*analytics.RequestLog{}}
	a := NewAnalyzer(storage, nil)
	ctx := context.Background()

	report, err := a.Analyze(ctx)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if report == nil {
		t.Fatal("Expected non-nil report")
	}
	if report.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests, got %d", report.TotalRequests)
	}
	if report.DuplicatePercent != 0 {
		t.Errorf("Expected 0 duplicate percent, got %f", report.DuplicatePercent)
	}
	if len(report.Recommendations) != 1 {
		t.Errorf("Expected 1 recommendation for empty logs, got %d", len(report.Recommendations))
	}
}

func TestAnalyze_StorageError(t *testing.T) {
	storage := &mockStorage{err: fmt.Errorf("storage error")}
	a := NewAnalyzer(storage, nil)
	ctx := context.Background()

	_, err := a.Analyze(ctx)
	if err == nil {
		t.Fatal("Expected error from Analyze")
	}
}

func TestAnalyze_WithDuplicates(t *testing.T) {
	now := time.Now()
	logs := []*analytics.RequestLog{
		{
			ID:          "log-1",
			Timestamp:   now.Add(-6 * time.Hour),
			ProviderID:  "openai",
			ModelName:   "gpt-4",
			RequestBody: "same request body",
			TotalTokens: 100,
			CostUSD:     0.10,
			LatencyMs:   500,
			StatusCode:  200,
		},
		{
			ID:          "log-2",
			Timestamp:   now.Add(-5 * time.Hour),
			ProviderID:  "openai",
			ModelName:   "gpt-4",
			RequestBody: "same request body",
			TotalTokens: 100,
			CostUSD:     0.10,
			LatencyMs:   500,
			StatusCode:  200,
		},
		{
			ID:          "log-3",
			Timestamp:   now.Add(-4 * time.Hour),
			ProviderID:  "openai",
			ModelName:   "gpt-4",
			RequestBody: "same request body",
			TotalTokens: 100,
			CostUSD:     0.10,
			LatencyMs:   500,
			StatusCode:  200,
		},
		{
			ID:          "log-4",
			Timestamp:   now.Add(-3 * time.Hour),
			ProviderID:  "openai",
			ModelName:   "gpt-4",
			RequestBody: "different request body",
			TotalTokens: 50,
			CostUSD:     0.05,
			LatencyMs:   300,
			StatusCode:  200,
		},
	}

	storage := &mockStorage{logs: logs}
	config := &AnalysisConfig{
		TimeWindow:     7 * 24 * time.Hour,
		MinOccurrences: 2,
		MinSavingsUSD:  0.01,
	}
	a := NewAnalyzer(storage, config)
	ctx := context.Background()

	report, err := a.Analyze(ctx)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if report.TotalRequests != 4 {
		t.Errorf("Expected 4 total requests, got %d", report.TotalRequests)
	}
	if report.DuplicateCount < 1 {
		t.Errorf("Expected at least 1 duplicate, got %d", report.DuplicateCount)
	}
	if report.DuplicatePercent <= 0 {
		t.Errorf("Expected positive duplicate percent, got %f", report.DuplicatePercent)
	}
	if len(report.Opportunities) == 0 {
		t.Error("Expected at least one opportunity")
	}
	if report.TotalSavingsUSD <= 0 {
		t.Errorf("Expected positive savings, got %f", report.TotalSavingsUSD)
	}
	if report.MonthlyProjection <= 0 {
		t.Errorf("Expected positive monthly projection, got %f", report.MonthlyProjection)
	}
	if len(report.Recommendations) == 0 {
		t.Error("Expected recommendations")
	}
}

func TestAnalyze_SkipsFailedRequests(t *testing.T) {
	now := time.Now()
	logs := []*analytics.RequestLog{
		{
			ID:          "log-1",
			Timestamp:   now.Add(-2 * time.Hour),
			ProviderID:  "openai",
			ModelName:   "gpt-4",
			RequestBody: "same request",
			TotalTokens: 100,
			CostUSD:     0.10,
			LatencyMs:   500,
			StatusCode:  200,
		},
		{
			ID:          "log-2",
			Timestamp:   now.Add(-1 * time.Hour),
			ProviderID:  "openai",
			ModelName:   "gpt-4",
			RequestBody: "same request",
			TotalTokens: 100,
			CostUSD:     0.10,
			LatencyMs:   500,
			StatusCode:  500, // Failed request
		},
	}

	storage := &mockStorage{logs: logs}
	a := NewAnalyzer(storage, nil)
	ctx := context.Background()

	report, err := a.Analyze(ctx)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	// The failed request should be skipped, so only 1 successful request -> no duplicates
	if len(report.Opportunities) != 0 {
		t.Errorf("Expected 0 opportunities (failed requests skipped), got %d", len(report.Opportunities))
	}
}

func TestAnalyze_HighPriorityOpportunity(t *testing.T) {
	now := time.Now()
	var logs []*analytics.RequestLog

	// Create many duplicates with high cost to trigger high priority
	for i := 0; i < 20; i++ {
		logs = append(logs, &analytics.RequestLog{
			ID:          fmt.Sprintf("log-%d", i),
			Timestamp:   now.Add(-time.Duration(20-i) * time.Hour),
			ProviderID:  "openai",
			ModelName:   "gpt-4",
			RequestBody: "expensive request",
			TotalTokens: 5000,
			CostUSD:     0.50,
			LatencyMs:   2000,
			StatusCode:  200,
		})
	}

	storage := &mockStorage{logs: logs}
	config := &AnalysisConfig{
		TimeWindow:     7 * 24 * time.Hour,
		MinOccurrences: 2,
		MinSavingsUSD:  0.01,
	}
	a := NewAnalyzer(storage, config)
	ctx := context.Background()

	report, err := a.Analyze(ctx)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if len(report.Opportunities) == 0 {
		t.Fatal("Expected opportunities")
	}
	if report.Opportunities[0].Priority != "high" {
		t.Errorf("Expected high priority, got %s", report.Opportunities[0].Priority)
	}
	// Should have recommendation about priority
	foundPriority := false
	for _, rec := range report.Recommendations {
		if len(rec) > 0 {
			foundPriority = true
		}
	}
	if !foundPriority {
		t.Error("Expected priority recommendation")
	}
}

func TestAnalyze_AutoEnable(t *testing.T) {
	now := time.Now()
	var logs []*analytics.RequestLog

	// Create many high-value duplicates to trigger auto-enable
	for i := 0; i < 100; i++ {
		logs = append(logs, &analytics.RequestLog{
			ID:          fmt.Sprintf("log-%d", i),
			Timestamp:   now.Add(-time.Duration(100-i) * time.Minute),
			ProviderID:  "openai",
			ModelName:   "gpt-4",
			RequestBody: "auto-enable request",
			TotalTokens: 10000,
			CostUSD:     1.00,
			LatencyMs:   3000,
			StatusCode:  200,
		})
	}

	storage := &mockStorage{logs: logs}
	config := &AnalysisConfig{
		TimeWindow:        7 * 24 * time.Hour,
		MinOccurrences:    2,
		MinSavingsUSD:     0.01,
		AutoEnable:        true,
		AutoEnableMinUSD:  1.0,
		AutoEnableMinRate: 0.5,
	}
	a := NewAnalyzer(storage, config)
	ctx := context.Background()

	report, err := a.Analyze(ctx)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if len(report.Opportunities) == 0 {
		t.Fatal("Expected opportunities")
	}

	// Check that at least one opportunity qualifies for auto-enable
	hasAutoEnable := false
	for _, opp := range report.Opportunities {
		if opp.AutoEnableable {
			hasAutoEnable = true
			break
		}
	}
	if !hasAutoEnable {
		t.Error("Expected at least one auto-enableable opportunity")
	}

	// Check recommendations mention auto-optimization
	foundAutoRec := false
	for _, rec := range report.Recommendations {
		if len(rec) > 0 {
			foundAutoRec = true
		}
	}
	if !foundAutoRec {
		t.Error("Expected auto-optimization recommendation")
	}
}

func TestSuggestTTL(t *testing.T) {
	a := NewAnalyzer(&mockStorage{}, nil)

	// count <= 1 should return default 1 hour
	ttl := a.suggestTTL(time.Now(), time.Now(), 1)
	if ttl != 1*time.Hour {
		t.Errorf("Expected 1h for count=1, got %v", ttl)
	}

	ttl = a.suggestTTL(time.Now(), time.Now(), 0)
	if ttl != 1*time.Hour {
		t.Errorf("Expected 1h for count=0, got %v", ttl)
	}

	// Very short interval -> clamp to 5 minutes
	now := time.Now()
	ttl = a.suggestTTL(now, now.Add(1*time.Minute), 10)
	if ttl < 5*time.Minute {
		t.Errorf("Expected at least 5m minimum, got %v", ttl)
	}

	// Very long interval -> clamp to 24 hours
	ttl = a.suggestTTL(now, now.Add(100*24*time.Hour), 3)
	if ttl > 24*time.Hour {
		t.Errorf("Expected at most 24h maximum, got %v", ttl)
	}

	// Moderate interval -> round to hours
	ttl = a.suggestTTL(now, now.Add(6*time.Hour), 4)
	// avgInterval = 6h/3 = 2h, suggested = 4h, rounded to 4h
	if ttl != 4*time.Hour {
		t.Errorf("Expected 4h, got %v", ttl)
	}

	// Interval that rounds to 15-minute increments (less than 1 hour)
	ttl = a.suggestTTL(now, now.Add(20*time.Minute), 3)
	// avgInterval = 20m/2 = 10m, suggested = 20m, rounded to nearest 15m
	if ttl < 5*time.Minute || ttl > 1*time.Hour {
		t.Errorf("Expected between 5m and 1h, got %v", ttl)
	}
}

func TestDeterminePriority(t *testing.T) {
	a := NewAnalyzer(&mockStorage{}, nil)

	// High priority: >$1 savings and >70% hit rate
	p := a.determinePriority(2.0, 80)
	if p != "high" {
		t.Errorf("Expected 'high', got '%s'", p)
	}

	// Medium priority: >$0.10 savings and >50% hit rate
	p = a.determinePriority(0.50, 60)
	if p != "medium" {
		t.Errorf("Expected 'medium', got '%s'", p)
	}

	// Low priority: below thresholds
	p = a.determinePriority(0.05, 30)
	if p != "low" {
		t.Errorf("Expected 'low', got '%s'", p)
	}

	// Edge: high savings but low hit rate
	p = a.determinePriority(5.0, 40)
	if p != "low" {
		t.Errorf("Expected 'low' for high savings but low hit rate, got '%s'", p)
	}

	// Edge: low savings but high hit rate
	p = a.determinePriority(0.05, 90)
	if p != "low" {
		t.Errorf("Expected 'low' for low savings but high hit rate, got '%s'", p)
	}
}

func TestFormatRecommendation(t *testing.T) {
	a := NewAnalyzer(&mockStorage{}, nil)
	dup := &DuplicateRequest{
		ProviderID: "openai",
		ModelName:  "gpt-4",
	}

	// High hit rate -> "Strongly recommend enabling"
	rec := a.formatRecommendation(dup, 5.0, 85, 1*time.Hour)
	if len(rec) == 0 {
		t.Error("Expected non-empty recommendation")
	}

	// Medium hit rate -> "Recommend enabling"
	rec = a.formatRecommendation(dup, 2.0, 65, 30*time.Minute)
	if len(rec) == 0 {
		t.Error("Expected non-empty recommendation")
	}

	// Low hit rate -> "Consider enabling"
	rec = a.formatRecommendation(dup, 0.5, 40, 2*time.Hour)
	if len(rec) == 0 {
		t.Error("Expected non-empty recommendation")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{2 * time.Hour, "2h"},
		{30 * time.Minute, "30m"},
		{5 * time.Minute, "5m"},
		{24 * time.Hour, "24h"},
		{0, "0m"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.d)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, result, tt.expected)
		}
	}
}

func TestTruncateString(t *testing.T) {
	// Short string - no truncation
	result := truncateString("short", 10)
	if result != "short" {
		t.Errorf("Expected 'short', got '%s'", result)
	}

	// Exact length - no truncation
	result = truncateString("exact", 5)
	if result != "exact" {
		t.Errorf("Expected 'exact', got '%s'", result)
	}

	// Long string - truncation
	result = truncateString("this is a long string", 10)
	if result != "this is a ..." {
		t.Errorf("Expected 'this is a ...', got '%s'", result)
	}
}

func TestCalculateSavings(t *testing.T) {
	a := NewAnalyzer(&mockStorage{}, nil)

	// No potential hits
	dup := &DuplicateRequest{
		OccurrenceCount: 1,
		TotalCost:       1.0,
	}
	savings := a.calculateSavings(dup)
	if savings != 0 {
		t.Errorf("Expected 0 savings for single occurrence, got %f", savings)
	}

	// Multiple occurrences
	dup2 := &DuplicateRequest{
		OccurrenceCount: 5,
		TotalCost:       5.0, // $1 per request
	}
	savings = a.calculateSavings(dup2)
	expected := 4.0 // 4 potential hits * $1 each
	if savings != expected {
		t.Errorf("Expected %f savings, got %f", expected, savings)
	}
}

func TestHashRequest(t *testing.T) {
	a := NewAnalyzer(&mockStorage{}, nil)

	// Same inputs should produce same hash
	h1 := a.hashRequest("openai", "gpt-4", "test body")
	h2 := a.hashRequest("openai", "gpt-4", "test body")
	if h1 != h2 {
		t.Error("Same inputs should produce same hash")
	}

	// Different inputs should produce different hashes
	h3 := a.hashRequest("openai", "gpt-4", "different body")
	if h1 == h3 {
		t.Error("Different inputs should produce different hashes")
	}

	h4 := a.hashRequest("anthropic", "gpt-4", "test body")
	if h1 == h4 {
		t.Error("Different provider should produce different hash")
	}

	h5 := a.hashRequest("openai", "gpt-3.5", "test body")
	if h1 == h5 {
		t.Error("Different model should produce different hash")
	}
}

func TestGenerateRecommendations_NoOpportunities(t *testing.T) {
	a := NewAnalyzer(&mockStorage{}, nil)

	recs := a.generateRecommendations(nil)
	if len(recs) != 1 {
		t.Fatalf("Expected 1 recommendation, got %d", len(recs))
	}
	if recs[0] != "No significant caching opportunities detected. Continue monitoring." {
		t.Errorf("Unexpected recommendation: %s", recs[0])
	}

	recs = a.generateRecommendations([]*CacheOpportunity{})
	if len(recs) != 1 {
		t.Fatalf("Expected 1 recommendation, got %d", len(recs))
	}
}

func TestGenerateRecommendations_WithOpportunities(t *testing.T) {
	a := NewAnalyzer(&mockStorage{}, nil)

	opportunities := []*CacheOpportunity{
		{
			Pattern:        "openai:gpt-4",
			CostSavableUSD: 5.0,
			HitRatePercent: 80,
			Priority:       "high",
			SuggestedTTL:   1 * time.Hour,
			AutoEnableable: true,
		},
		{
			Pattern:        "anthropic:claude-3",
			CostSavableUSD: 1.0,
			HitRatePercent: 60,
			Priority:       "medium",
			SuggestedTTL:   2 * time.Hour,
			AutoEnableable: false,
		},
	}

	recs := a.generateRecommendations(opportunities)
	if len(recs) < 3 {
		t.Errorf("Expected at least 3 recommendations, got %d", len(recs))
	}
}

func TestDetectDuplicates_SortsBySavings(t *testing.T) {
	a := NewAnalyzer(&mockStorage{}, &AnalysisConfig{
		TimeWindow:     7 * 24 * time.Hour,
		MinOccurrences: 2,
		MinSavingsUSD:  0.001,
	})

	now := time.Now()
	logs := []*analytics.RequestLog{
		// Group A: 3 occurrences, low cost
		{ID: "a1", Timestamp: now.Add(-3 * time.Hour), ProviderID: "p1", ModelName: "m1", RequestBody: "bodyA", TotalTokens: 10, CostUSD: 0.01, LatencyMs: 100, StatusCode: 200},
		{ID: "a2", Timestamp: now.Add(-2 * time.Hour), ProviderID: "p1", ModelName: "m1", RequestBody: "bodyA", TotalTokens: 10, CostUSD: 0.01, LatencyMs: 100, StatusCode: 200},
		{ID: "a3", Timestamp: now.Add(-1 * time.Hour), ProviderID: "p1", ModelName: "m1", RequestBody: "bodyA", TotalTokens: 10, CostUSD: 0.01, LatencyMs: 100, StatusCode: 200},
		// Group B: 2 occurrences, high cost
		{ID: "b1", Timestamp: now.Add(-3 * time.Hour), ProviderID: "p2", ModelName: "m2", RequestBody: "bodyB", TotalTokens: 1000, CostUSD: 1.00, LatencyMs: 2000, StatusCode: 200},
		{ID: "b2", Timestamp: now.Add(-2 * time.Hour), ProviderID: "p2", ModelName: "m2", RequestBody: "bodyB", TotalTokens: 1000, CostUSD: 1.00, LatencyMs: 2000, StatusCode: 200},
	}

	dups := a.detectDuplicates(logs)
	if len(dups) < 2 {
		t.Fatalf("Expected at least 2 duplicate groups, got %d", len(dups))
	}

	// Group B should come first (higher savings)
	if dups[0].ProviderID != "p2" {
		t.Errorf("Expected higher-savings group first, got provider %s", dups[0].ProviderID)
	}
}

func TestIdentifyOpportunities_BelowMinSavings(t *testing.T) {
	a := NewAnalyzer(&mockStorage{}, &AnalysisConfig{
		TimeWindow:     7 * 24 * time.Hour,
		MinOccurrences: 2,
		MinSavingsUSD:  100.0, // Very high threshold
	})

	now := time.Now()
	dups := []*DuplicateRequest{
		{
			RequestHash:     "hash1",
			FirstSeen:       now.Add(-3 * time.Hour),
			LastSeen:        now,
			OccurrenceCount: 3,
			ProviderID:      "p1",
			ModelName:       "m1",
			TotalTokens:     300,
			TotalCost:       0.03, // Below threshold
			AvgLatencyMs:    100,
		},
	}

	opps := a.identifyOpportunities(dups)
	if len(opps) != 0 {
		t.Errorf("Expected 0 opportunities below min savings, got %d", len(opps))
	}
}

func TestIdentifyOpportunities_SingleOccurrence(t *testing.T) {
	a := NewAnalyzer(&mockStorage{}, &AnalysisConfig{
		TimeWindow:     7 * 24 * time.Hour,
		MinOccurrences: 1,
		MinSavingsUSD:  0.001,
	})

	dups := []*DuplicateRequest{
		{
			RequestHash:     "hash1",
			OccurrenceCount: 1, // Only 1 occurrence, no potential hits
			ProviderID:      "p1",
			ModelName:       "m1",
			TotalTokens:     100,
			TotalCost:       1.00,
		},
	}

	opps := a.identifyOpportunities(dups)
	if len(opps) != 0 {
		t.Errorf("Expected 0 opportunities for single occurrence, got %d", len(opps))
	}
}
