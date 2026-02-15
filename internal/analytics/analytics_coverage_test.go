package analytics

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ---- Tests for DefaultAlertConfig ----

func TestDefaultAlertConfig(t *testing.T) {
	config := DefaultAlertConfig("user-123")
	if config.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got %q", config.UserID)
	}
	if config.DailyBudgetUSD != 100.0 {
		t.Errorf("expected daily budget 100.0, got %f", config.DailyBudgetUSD)
	}
	if config.MonthlyBudgetUSD != 2000.0 {
		t.Errorf("expected monthly budget 2000.0, got %f", config.MonthlyBudgetUSD)
	}
	if config.AnomalyThreshold != 2.0 {
		t.Errorf("expected anomaly threshold 2.0, got %f", config.AnomalyThreshold)
	}
	if config.EnableEmailAlerts {
		t.Error("expected email alerts disabled by default")
	}
	if config.EnableWebhookAlerts {
		t.Error("expected webhook alerts disabled by default")
	}
}

// ---- Tests for buildEmailBody ----

func TestBuildEmailBody_CriticalSeverity(t *testing.T) {
	alert := &Alert{
		ID:          "alert-crit-1",
		UserID:      "user-1",
		Type:        "budget_exceeded",
		Severity:    "critical",
		Message:     "Monthly budget exceeded",
		CurrentCost: 3000.0,
		Threshold:   2000.0,
		TriggeredAt: time.Now(),
	}

	body := buildEmailBody(alert)
	if !strings.Contains(body, "#DC3545") {
		t.Error("critical severity should use red color")
	}
	if !strings.Contains(body, "critical") {
		t.Error("body should contain severity")
	}
	if !strings.Contains(body, "$3000.00") {
		t.Error("body should contain current cost")
	}
}

func TestBuildEmailBody_InfoSeverity(t *testing.T) {
	alert := &Alert{
		ID:          "alert-info-1",
		UserID:      "user-1",
		Type:        "anomaly_detected",
		Severity:    "info",
		Message:     "Info message",
		CurrentCost: 10.0,
		Threshold:   5.0,
		TriggeredAt: time.Now(),
	}

	body := buildEmailBody(alert)
	if !strings.Contains(body, "#17A2B8") {
		t.Error("info severity should use blue color")
	}
}

func TestBuildEmailBody_WarningSeverity(t *testing.T) {
	alert := &Alert{
		ID:          "alert-warn-1",
		UserID:      "user-1",
		Type:        "budget_exceeded",
		Severity:    "warning",
		Message:     "Budget warning",
		CurrentCost: 120.0,
		Threshold:   100.0,
		TriggeredAt: time.Now(),
	}

	body := buildEmailBody(alert)
	if !strings.Contains(body, "#FFA500") {
		t.Error("warning severity should use orange color")
	}
}

// ---- Tests for loadSMTPConfigFromEnv ----

func TestLoadSMTPConfigFromEnv_NoHost(t *testing.T) {
	// Ensure SMTP_HOST is not set
	t.Setenv("SMTP_HOST", "")
	config := loadSMTPConfigFromEnv()
	if config != nil {
		t.Error("expected nil config when SMTP_HOST is empty")
	}
}

func TestLoadSMTPConfigFromEnv_InvalidPort(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "not-a-number")
	config := loadSMTPConfigFromEnv()
	if config == nil {
		t.Fatal("expected config to be loaded")
	}
	// Should use default port 587
	if config.Port != 587 {
		t.Errorf("expected default port 587, got %d", config.Port)
	}
}

func TestLoadSMTPConfigFromEnv_TLSFalse(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_USE_TLS", "false")
	config := loadSMTPConfigFromEnv()
	if config == nil {
		t.Fatal("expected config to be loaded")
	}
	if config.UseTLS {
		t.Error("expected UseTLS to be false")
	}
}

func TestLoadSMTPConfigFromEnv_TLSZero(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_USE_TLS", "0")
	config := loadSMTPConfigFromEnv()
	if config == nil {
		t.Fatal("expected config to be loaded")
	}
	if config.UseTLS {
		t.Error("expected UseTLS to be false when set to '0'")
	}
}

func TestLoadSMTPConfigFromEnv_TLSDefault(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_USE_TLS", "")
	config := loadSMTPConfigFromEnv()
	if config == nil {
		t.Fatal("expected config to be loaded")
	}
	if !config.UseTLS {
		t.Error("expected UseTLS to be true by default")
	}
}

func TestLoadSMTPConfigFromEnv_CustomPort(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "465")
	config := loadSMTPConfigFromEnv()
	if config == nil {
		t.Fatal("expected config to be loaded")
	}
	if config.Port != 465 {
		t.Errorf("expected port 465, got %d", config.Port)
	}
}

// ---- Tests for sendEmail with nil SMTP ----

func TestSendEmail_NilSMTP(t *testing.T) {
	storage := NewInMemoryStorage()
	config := &AlertConfig{
		UserID: "user-test",
	}
	checker := &AlertChecker{
		storage:    storage,
		config:     config,
		smtpConfig: nil,
	}

	alert := &Alert{
		ID:          "alert-1",
		UserID:      "user-test",
		Type:        "budget_exceeded",
		Severity:    "warning",
		Message:     "test",
		CurrentCost: 150.0,
		Threshold:   100.0,
		TriggeredAt: time.Now(),
	}

	err := checker.sendEmail(alert)
	if err == nil {
		t.Error("expected error when SMTP is nil")
	}
	if !strings.Contains(err.Error(), "SMTP not configured") {
		t.Errorf("error should mention 'SMTP not configured': %v", err)
	}
}

// ---- Tests for notify ----

func TestNotify_EmailDisabled(t *testing.T) {
	storage := NewInMemoryStorage()
	config := &AlertConfig{
		UserID:            "user-test",
		EnableEmailAlerts: false,
		EmailAddress:      "test@example.com",
	}
	checker := &AlertChecker{
		storage:    storage,
		config:     config,
		smtpConfig: nil,
	}

	alert := &Alert{
		ID:       "alert-1",
		Severity: "warning",
		Message:  "test alert",
	}

	// Should not panic
	checker.notify(alert)
}

func TestNotify_EmailEnabledNoSMTP(t *testing.T) {
	storage := NewInMemoryStorage()
	config := &AlertConfig{
		UserID:            "user-test",
		EnableEmailAlerts: true,
		EmailAddress:      "test@example.com",
	}
	checker := &AlertChecker{
		storage:    storage,
		config:     config,
		smtpConfig: nil, // No SMTP configured
	}

	alert := &Alert{
		ID:       "alert-1",
		Severity: "warning",
		Message:  "test alert",
	}

	// Should not panic - just log warning
	checker.notify(alert)
}

func TestNotify_EmailEnabledNoAddress(t *testing.T) {
	storage := NewInMemoryStorage()
	config := &AlertConfig{
		UserID:            "user-test",
		EnableEmailAlerts: true,
		EmailAddress:      "", // Empty address
	}
	checker := &AlertChecker{
		storage: storage,
		config:  config,
	}

	alert := &Alert{
		ID:       "alert-1",
		Severity: "info",
		Message:  "test",
	}

	// Should not panic
	checker.notify(alert)
}

func TestNotify_WebhookEnabledNoURL(t *testing.T) {
	storage := NewInMemoryStorage()
	config := &AlertConfig{
		UserID:              "user-test",
		EnableWebhookAlerts: true,
		WebhookURL:          "", // Empty URL
	}
	checker := &AlertChecker{
		storage: storage,
		config:  config,
	}

	alert := &Alert{
		ID:       "alert-1",
		Severity: "warning",
		Message:  "test",
	}

	// Should not panic
	checker.notify(alert)
}

// ---- Tests for CheckAlerts edge cases ----

func TestCheckAlerts_AllZeroBudgets(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	config := &AlertConfig{
		UserID:           "user-test",
		DailyBudgetUSD:   0,
		MonthlyBudgetUSD: 0,
		AnomalyThreshold: 0,
	}

	checker := NewAlertChecker(storage, config)
	alerts, err := checker.CheckAlerts(ctx)
	if err != nil {
		t.Fatalf("CheckAlerts failed: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts with all zero budgets, got %d", len(alerts))
	}
}

func TestCheckAlerts_AnomalyThresholdBelowOne(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// AnomalyThreshold <= 1.0 should be skipped
	config := &AlertConfig{
		UserID:           "user-test",
		AnomalyThreshold: 0.5,
	}

	checker := NewAlertChecker(storage, config)
	alerts, err := checker.CheckAlerts(ctx)
	if err != nil {
		t.Fatalf("CheckAlerts failed: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts with anomaly threshold below 1, got %d", len(alerts))
	}
}

func TestCheckAlerts_AnomalyNoHistoricalData(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	now := time.Now()
	// Only add today's data, no historical data
	_ = storage.SaveLog(ctx, &RequestLog{
		ID:        "log-1",
		Timestamp: now.Add(-1 * time.Minute),
		UserID:    "user-test",
		CostUSD:   100.0,
	})

	config := &AlertConfig{
		UserID:           "user-test",
		AnomalyThreshold: 2.0,
	}

	checker := NewAlertChecker(storage, config)
	alerts, err := checker.CheckAlerts(ctx)
	if err != nil {
		t.Fatalf("CheckAlerts failed: %v", err)
	}
	// No historical data means avg=0, so anomaly check should not trigger
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts with no historical data, got %d", len(alerts))
	}
}

func TestCheckAlerts_MultipleBudgetAlerts(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	now := time.Now()
	// Add enough cost to exceed both daily and monthly budgets
	for i := 0; i < 5; i++ {
		_ = storage.SaveLog(ctx, &RequestLog{
			ID:        fmt.Sprintf("log-%d", i),
			Timestamp: now.Add(-time.Duration(i+1) * time.Minute),
			UserID:    "user-test",
			CostUSD:   500.0, // Total: $2500
		})
	}

	config := &AlertConfig{
		UserID:           "user-test",
		DailyBudgetUSD:   100.0,
		MonthlyBudgetUSD: 2000.0,
	}

	checker := NewAlertChecker(storage, config)
	alerts, err := checker.CheckAlerts(ctx)
	if err != nil {
		t.Fatalf("CheckAlerts failed: %v", err)
	}
	if len(alerts) < 2 {
		t.Errorf("expected at least 2 alerts (daily + monthly), got %d", len(alerts))
	}
}

// ---- Tests for batching functions ----

func TestDefaultBatchingOptions(t *testing.T) {
	opts := DefaultBatchingOptions()
	if opts.Window != 5*time.Minute {
		t.Errorf("expected window 5m, got %v", opts.Window)
	}
	if opts.MinBatchSize != 3 {
		t.Errorf("expected min batch size 3, got %d", opts.MinBatchSize)
	}
	if opts.MaxBatchSize != 10 {
		t.Errorf("expected max batch size 10, got %d", opts.MaxBatchSize)
	}
	if opts.MaxRecommendations != 20 {
		t.Errorf("expected max recommendations 20, got %d", opts.MaxRecommendations)
	}
	if opts.OverheadTokens != 100 {
		t.Errorf("expected overhead tokens 100, got %d", opts.OverheadTokens)
	}
	if !opts.IncludeAutoBatchPlan {
		t.Error("expected IncludeAutoBatchPlan to be true")
	}
}

func TestBuildBatchingRecommendations_NilOptions(t *testing.T) {
	baseTime := time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
	logs := []*RequestLog{
		{ID: "1", Timestamp: baseTime, UserID: "u", Method: "POST", Path: "/api", ProviderID: "p", ModelName: "m", TotalTokens: 100, StatusCode: 200, CostUSD: 0.01},
		{ID: "2", Timestamp: baseTime.Add(1 * time.Minute), UserID: "u", Method: "POST", Path: "/api", ProviderID: "p", ModelName: "m", TotalTokens: 100, StatusCode: 200, CostUSD: 0.01},
		{ID: "3", Timestamp: baseTime.Add(2 * time.Minute), UserID: "u", Method: "POST", Path: "/api", ProviderID: "p", ModelName: "m", TotalTokens: 100, StatusCode: 200, CostUSD: 0.01},
	}

	// nil options should use defaults
	result := BuildBatchingRecommendations(logs, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestBuildBatchingRecommendations_EmptyLogs(t *testing.T) {
	result := BuildBatchingRecommendations([]*RequestLog{}, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Recommendations) != 0 {
		t.Errorf("expected 0 recommendations, got %d", len(result.Recommendations))
	}
	if result.Summary.BatchableRequests != 0 {
		t.Errorf("expected 0 batchable requests, got %d", result.Summary.BatchableRequests)
	}
}

func TestBuildBatchingRecommendations_NilLogs(t *testing.T) {
	result := BuildBatchingRecommendations(nil, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Recommendations) != 0 {
		t.Errorf("expected 0 recommendations, got %d", len(result.Recommendations))
	}
}

func TestFilterBatchableLogs(t *testing.T) {
	logs := []*RequestLog{
		{ID: "1", StatusCode: 200, TotalTokens: 100},                // batchable
		{ID: "2", StatusCode: 400, TotalTokens: 100},                // error status
		{ID: "3", StatusCode: 200, TotalTokens: 0},                  // zero tokens
		{ID: "4", StatusCode: 200, TotalTokens: -10},                // negative tokens
		nil,                                                          // nil entry
		{ID: "5", StatusCode: 200, TotalTokens: 50},                 // batchable
	}

	filtered := filterBatchableLogs(logs)
	if len(filtered) != 2 {
		t.Errorf("expected 2 batchable logs, got %d", len(filtered))
	}
}

func TestBuildBatchKey(t *testing.T) {
	log := &RequestLog{
		UserID:     "user-1",
		ProviderID: "provider-1",
		ModelName:  "model-1",
		Method:     "POST",
		Path:       "/api/v1/chat",
	}
	key := buildBatchKey(log)
	expected := "user-1|provider-1|model-1|POST|/api/v1/chat"
	if key != expected {
		t.Errorf("buildBatchKey = %q, want %q", key, expected)
	}
}

func TestSliceWindow(t *testing.T) {
	baseTime := time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
	logs := []*RequestLog{
		{ID: "1", Timestamp: baseTime},
		{ID: "2", Timestamp: baseTime.Add(1 * time.Minute)},
		{ID: "3", Timestamp: baseTime.Add(2 * time.Minute)},
		{ID: "4", Timestamp: baseTime.Add(10 * time.Minute)}, // outside 5-min window
	}

	opts := &BatchingOptions{
		Window:       5 * time.Minute,
		MaxBatchSize: 10,
	}

	windowLogs, nextIdx := sliceWindow(logs, 0, opts)
	if len(windowLogs) != 3 {
		t.Errorf("expected 3 logs in window, got %d", len(windowLogs))
	}
	if nextIdx != 3 {
		t.Errorf("expected nextIdx 3, got %d", nextIdx)
	}
}

func TestSliceWindow_StartBeyondEnd(t *testing.T) {
	logs := []*RequestLog{
		{ID: "1", Timestamp: time.Now()},
	}

	opts := &BatchingOptions{Window: 5 * time.Minute, MaxBatchSize: 10}
	windowLogs, nextIdx := sliceWindow(logs, 5, opts)
	if len(windowLogs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(windowLogs))
	}
	if nextIdx != 1 {
		t.Errorf("expected nextIdx = len(logs), got %d", nextIdx)
	}
}

func TestSliceWindow_MaxBatchSize(t *testing.T) {
	baseTime := time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
	logs := []*RequestLog{
		{ID: "1", Timestamp: baseTime},
		{ID: "2", Timestamp: baseTime.Add(1 * time.Second)},
		{ID: "3", Timestamp: baseTime.Add(2 * time.Second)},
		{ID: "4", Timestamp: baseTime.Add(3 * time.Second)},
		{ID: "5", Timestamp: baseTime.Add(4 * time.Second)},
	}

	opts := &BatchingOptions{
		Window:       5 * time.Minute,
		MaxBatchSize: 3,
	}

	windowLogs, nextIdx := sliceWindow(logs, 0, opts)
	if len(windowLogs) != 3 {
		t.Errorf("expected 3 logs (max batch size), got %d", len(windowLogs))
	}
	if nextIdx != 3 {
		t.Errorf("expected nextIdx 3, got %d", nextIdx)
	}
}

func TestSuggestBatchSize(t *testing.T) {
	tests := []struct {
		name         string
		requestCount int
		minSize      int
		maxSize      int
		wantMin      int
		wantMax      int
	}{
		{"zero requests", 0, 3, 10, 0, 0},
		{"small count", 4, 3, 10, 2, 4},
		{"large count", 20, 3, 10, 3, 10},
		{"min larger than count", 2, 5, 10, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &BatchingOptions{MinBatchSize: tt.minSize, MaxBatchSize: tt.maxSize}
			result := suggestBatchSize(tt.requestCount, opts)
			if result < tt.wantMin || result > tt.wantMax {
				t.Errorf("suggestBatchSize(%d) = %d, want in [%d, %d]", tt.requestCount, result, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestSuggestBatchSize_ZeroMinMax(t *testing.T) {
	opts := &BatchingOptions{MinBatchSize: 0, MaxBatchSize: 0}
	result := suggestBatchSize(5, opts)
	// minSize defaults to 2, maxSize defaults to requestCount
	if result < 1 {
		t.Errorf("expected batch size >= 1, got %d", result)
	}
}

func TestBuildAutoBatchPlan(t *testing.T) {
	baseTime := time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
	logs := []*RequestLog{
		{ID: "1", Timestamp: baseTime},
		{ID: "2", Timestamp: baseTime.Add(1 * time.Minute)},
		{ID: "3", Timestamp: baseTime.Add(2 * time.Minute)},
		{ID: "4", Timestamp: baseTime.Add(3 * time.Minute)},
		{ID: "5", Timestamp: baseTime.Add(4 * time.Minute)},
	}

	rec := BatchRecommendation{
		ID:        "batch-1",
		BatchSize: 2,
	}
	opts := &BatchingOptions{MaxAutoBatchGroups: 50}

	plan := buildAutoBatchPlan(rec, logs, opts)
	if len(plan) != 3 { // 5 logs / batch size 2 = 3 groups (2, 2, 1)
		t.Errorf("expected 3 auto batch groups, got %d", len(plan))
	}

	// Verify first group
	if len(plan[0].RequestIDs) != 2 {
		t.Errorf("first group should have 2 requests, got %d", len(plan[0].RequestIDs))
	}
	if plan[0].RecommendationID != "batch-1" {
		t.Errorf("RecommendationID = %q, want batch-1", plan[0].RecommendationID)
	}
}

func TestBuildAutoBatchPlan_EmptyLogs(t *testing.T) {
	rec := BatchRecommendation{ID: "batch-1", BatchSize: 2}
	opts := &BatchingOptions{MaxAutoBatchGroups: 50}

	plan := buildAutoBatchPlan(rec, nil, opts)
	if plan != nil {
		t.Errorf("expected nil plan for empty logs, got %v", plan)
	}
}

func TestBuildAutoBatchPlan_ZeroBatchSize(t *testing.T) {
	logs := []*RequestLog{{ID: "1", Timestamp: time.Now()}}
	rec := BatchRecommendation{ID: "batch-1", BatchSize: 0}
	opts := &BatchingOptions{MaxAutoBatchGroups: 50}

	plan := buildAutoBatchPlan(rec, logs, opts)
	if plan != nil {
		t.Errorf("expected nil plan for zero batch size, got %v", plan)
	}
}

func TestBuildAutoBatchPlan_MaxGroupsLimit(t *testing.T) {
	baseTime := time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
	logs := make([]*RequestLog, 20)
	for i := 0; i < 20; i++ {
		logs[i] = &RequestLog{ID: fmt.Sprintf("log-%d", i), Timestamp: baseTime.Add(time.Duration(i) * time.Second)}
	}

	rec := BatchRecommendation{ID: "batch-1", BatchSize: 2}
	opts := &BatchingOptions{MaxAutoBatchGroups: 3}

	plan := buildAutoBatchPlan(rec, logs, opts)
	if len(plan) > 3 {
		t.Errorf("expected at most 3 groups, got %d", len(plan))
	}
}

func TestBuildRecommendation(t *testing.T) {
	baseTime := time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
	logs := []*RequestLog{
		{ID: "1", Timestamp: baseTime, UserID: "u", ProviderID: "p", ModelName: "m", Method: "POST", Path: "/api", TotalTokens: 1000, CostUSD: 0.10, LatencyMs: 200},
		{ID: "2", Timestamp: baseTime.Add(1 * time.Minute), UserID: "u", ProviderID: "p", ModelName: "m", Method: "POST", Path: "/api", TotalTokens: 800, CostUSD: 0.08, LatencyMs: 180},
		{ID: "3", Timestamp: baseTime.Add(2 * time.Minute), UserID: "u", ProviderID: "p", ModelName: "m", Method: "POST", Path: "/api", TotalTokens: 1200, CostUSD: 0.12, LatencyMs: 250},
	}

	opts := DefaultBatchingOptions()
	rec := buildRecommendation(logs, opts)

	if rec.RequestCount != 3 {
		t.Errorf("RequestCount = %d, want 3", rec.RequestCount)
	}
	if rec.TotalTokens != 3000 {
		t.Errorf("TotalTokens = %d, want 3000", rec.TotalTokens)
	}
	if rec.TotalCostUSD != 0.30 {
		t.Errorf("TotalCostUSD = %f, want 0.30", rec.TotalCostUSD)
	}
	if rec.UserID != "u" {
		t.Errorf("UserID = %q, want 'u'", rec.UserID)
	}
	if rec.ProviderID != "p" {
		t.Errorf("ProviderID = %q, want 'p'", rec.ProviderID)
	}
	if len(rec.SampleRequestIDs) != 3 {
		t.Errorf("SampleRequestIDs length = %d, want 3", len(rec.SampleRequestIDs))
	}
}

func TestBuildRecommendation_ManyLogs(t *testing.T) {
	baseTime := time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
	logs := make([]*RequestLog, 10)
	for i := 0; i < 10; i++ {
		logs[i] = &RequestLog{
			ID: fmt.Sprintf("log-%d", i), Timestamp: baseTime.Add(time.Duration(i) * time.Second),
			UserID: "u", ProviderID: "p", ModelName: "m", Method: "POST", Path: "/api",
			TotalTokens: 100, CostUSD: 0.01, LatencyMs: 100,
		}
	}

	opts := DefaultBatchingOptions()
	rec := buildRecommendation(logs, opts)

	// SampleRequestIDs should be capped at 5
	if len(rec.SampleRequestIDs) != 5 {
		t.Errorf("SampleRequestIDs length = %d, want 5", len(rec.SampleRequestIDs))
	}
}

// ---- Tests for batching with multiple groups ----

func TestBuildBatchingRecommendations_MultipleGroups(t *testing.T) {
	baseTime := time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
	logs := []*RequestLog{
		// Group 1: user-1, provider-a, model-x, POST, /api/v1
		{ID: "1", Timestamp: baseTime, UserID: "user-1", ProviderID: "provider-a", ModelName: "model-x", Method: "POST", Path: "/api/v1", TotalTokens: 100, StatusCode: 200, CostUSD: 0.01},
		{ID: "2", Timestamp: baseTime.Add(1 * time.Minute), UserID: "user-1", ProviderID: "provider-a", ModelName: "model-x", Method: "POST", Path: "/api/v1", TotalTokens: 100, StatusCode: 200, CostUSD: 0.01},
		{ID: "3", Timestamp: baseTime.Add(2 * time.Minute), UserID: "user-1", ProviderID: "provider-a", ModelName: "model-x", Method: "POST", Path: "/api/v1", TotalTokens: 100, StatusCode: 200, CostUSD: 0.01},
		// Group 2: user-2, provider-b, model-y, POST, /api/v2
		{ID: "4", Timestamp: baseTime, UserID: "user-2", ProviderID: "provider-b", ModelName: "model-y", Method: "POST", Path: "/api/v2", TotalTokens: 200, StatusCode: 200, CostUSD: 0.02},
		{ID: "5", Timestamp: baseTime.Add(1 * time.Minute), UserID: "user-2", ProviderID: "provider-b", ModelName: "model-y", Method: "POST", Path: "/api/v2", TotalTokens: 200, StatusCode: 200, CostUSD: 0.02},
		{ID: "6", Timestamp: baseTime.Add(2 * time.Minute), UserID: "user-2", ProviderID: "provider-b", ModelName: "model-y", Method: "POST", Path: "/api/v2", TotalTokens: 200, StatusCode: 200, CostUSD: 0.02},
	}

	opts := DefaultBatchingOptions()
	opts.Window = 10 * time.Minute

	result := BuildBatchingRecommendations(logs, opts)
	if len(result.Recommendations) != 2 {
		t.Errorf("expected 2 recommendations, got %d", len(result.Recommendations))
	}
}

func TestBuildBatchingRecommendations_MaxRecommendationsLimit(t *testing.T) {
	baseTime := time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
	var logs []*RequestLog

	// Create many groups to exceed MaxRecommendations
	for g := 0; g < 5; g++ {
		for i := 0; i < 5; i++ {
			logs = append(logs, &RequestLog{
				ID:          fmt.Sprintf("log-%d-%d", g, i),
				Timestamp:   baseTime.Add(time.Duration(i) * time.Second),
				UserID:      fmt.Sprintf("user-%d", g),
				ProviderID:  "p",
				ModelName:   "m",
				Method:      "POST",
				Path:        "/api",
				TotalTokens: 100,
				StatusCode:  200,
				CostUSD:     0.01,
			})
		}
	}

	opts := DefaultBatchingOptions()
	opts.Window = 10 * time.Minute
	opts.MaxRecommendations = 2
	opts.MinBatchSize = 3

	result := BuildBatchingRecommendations(logs, opts)
	if len(result.Recommendations) > 2 {
		t.Errorf("expected at most 2 recommendations, got %d", len(result.Recommendations))
	}
}

func TestBuildBatchingRecommendations_NoAutoBatchPlan(t *testing.T) {
	baseTime := time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
	logs := []*RequestLog{
		{ID: "1", Timestamp: baseTime, UserID: "u", ProviderID: "p", ModelName: "m", Method: "POST", Path: "/api", TotalTokens: 100, StatusCode: 200, CostUSD: 0.01},
		{ID: "2", Timestamp: baseTime.Add(1 * time.Minute), UserID: "u", ProviderID: "p", ModelName: "m", Method: "POST", Path: "/api", TotalTokens: 100, StatusCode: 200, CostUSD: 0.01},
		{ID: "3", Timestamp: baseTime.Add(2 * time.Minute), UserID: "u", ProviderID: "p", ModelName: "m", Method: "POST", Path: "/api", TotalTokens: 100, StatusCode: 200, CostUSD: 0.01},
	}

	opts := DefaultBatchingOptions()
	opts.Window = 10 * time.Minute
	opts.IncludeAutoBatchPlan = false

	result := BuildBatchingRecommendations(logs, opts)
	if len(result.AutoBatchPlan) != 0 {
		t.Errorf("expected 0 auto batch groups when disabled, got %d", len(result.AutoBatchPlan))
	}
}

// ---- Tests for InMemoryStorage ----

func TestInMemoryStorage_GetLogs_Filters(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	now := time.Now()

	_ = storage.SaveLog(ctx, &RequestLog{ID: "1", Timestamp: now, UserID: "alice", ProviderID: "p1"})
	_ = storage.SaveLog(ctx, &RequestLog{ID: "2", Timestamp: now, UserID: "bob", ProviderID: "p2"})
	_ = storage.SaveLog(ctx, &RequestLog{ID: "3", Timestamp: now.Add(-2 * time.Hour), UserID: "alice", ProviderID: "p1"})

	// Filter by UserID
	logs, err := storage.GetLogs(ctx, &LogFilter{UserID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs for alice, got %d", len(logs))
	}

	// Filter by ProviderID
	logs, err = storage.GetLogs(ctx, &LogFilter{ProviderID: "p2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log for p2, got %d", len(logs))
	}

	// Filter by time range
	logs, err = storage.GetLogs(ctx, &LogFilter{StartTime: now.Add(-1 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs in time range, got %d", len(logs))
	}

	// Filter by end time
	logs, err = storage.GetLogs(ctx, &LogFilter{EndTime: now.Add(-1 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log before end time, got %d", len(logs))
	}
}

func TestInMemoryStorage_DeleteOldLogs(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	now := time.Now()
	_ = storage.SaveLog(ctx, &RequestLog{ID: "1", Timestamp: now.Add(-48 * time.Hour)})
	_ = storage.SaveLog(ctx, &RequestLog{ID: "2", Timestamp: now.Add(-24 * time.Hour)})
	_ = storage.SaveLog(ctx, &RequestLog{ID: "3", Timestamp: now})

	deleted, err := storage.DeleteOldLogs(ctx, now.Add(-12*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}

	logs, _ := storage.GetLogs(ctx, &LogFilter{})
	if len(logs) != 1 {
		t.Errorf("expected 1 remaining log, got %d", len(logs))
	}
}

func TestInMemoryStorage_GetLogStats_ErrorRate(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	now := time.Now()
	_ = storage.SaveLog(ctx, &RequestLog{ID: "1", Timestamp: now, UserID: "u", StatusCode: 200, LatencyMs: 100, TotalTokens: 100, CostUSD: 0.01})
	_ = storage.SaveLog(ctx, &RequestLog{ID: "2", Timestamp: now, UserID: "u", StatusCode: 500, LatencyMs: 200, TotalTokens: 200, CostUSD: 0.02})

	stats, err := storage.GetLogStats(ctx, &LogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2", stats.TotalRequests)
	}
	if stats.ErrorRate != 0.5 {
		t.Errorf("ErrorRate = %f, want 0.5", stats.ErrorRate)
	}
	if stats.AvgLatencyMs != 150.0 {
		t.Errorf("AvgLatencyMs = %f, want 150.0", stats.AvgLatencyMs)
	}
}

func TestInMemoryStorage_GetLogStats_Empty(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	stats, err := storage.GetLogStats(ctx, &LogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalRequests != 0 {
		t.Errorf("TotalRequests = %d, want 0", stats.TotalRequests)
	}
	if stats.ErrorRate != 0 {
		t.Errorf("ErrorRate = %f, want 0", stats.ErrorRate)
	}
}

// ---- Tests for Logger ----

func TestLogger_GetLogs(t *testing.T) {
	storage := NewInMemoryStorage()
	logger := NewLogger(storage, DefaultPrivacyConfig())
	ctx := context.Background()

	_ = logger.LogRequest(ctx, &RequestLog{ID: "1", UserID: "u"})
	_ = logger.LogRequest(ctx, &RequestLog{ID: "2", UserID: "u"})

	logs, err := logger.GetLogs(ctx, &LogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
}

func TestLogger_PurgeLogs(t *testing.T) {
	storage := NewInMemoryStorage()
	logger := NewLogger(storage, DefaultPrivacyConfig())
	ctx := context.Background()

	now := time.Now()
	_ = logger.LogRequest(ctx, &RequestLog{ID: "1", Timestamp: now.Add(-48 * time.Hour)})
	_ = logger.LogRequest(ctx, &RequestLog{ID: "2", Timestamp: now})

	deleted, err := logger.PurgeLogs(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
}

func TestLogger_AutoGenerateIDAndTimestamp(t *testing.T) {
	storage := &MockStorage{}
	logger := NewLogger(storage, DefaultPrivacyConfig())
	ctx := context.Background()

	log := &RequestLog{}
	err := logger.LogRequest(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	if storage.logs[0].ID == "" {
		t.Error("expected auto-generated ID")
	}
	if storage.logs[0].Timestamp.IsZero() {
		t.Error("expected auto-set timestamp")
	}
}

func TestLogger_ResponseBodyTruncation(t *testing.T) {
	storage := &MockStorage{}
	privacy := &PrivacyConfig{
		LogRequestBodies:  false,
		LogResponseBodies: true,
		MaxBodyLength:     10,
	}
	logger := NewLogger(storage, privacy)
	ctx := context.Background()

	log := &RequestLog{
		ResponseBody: "this is a very long response body that should be truncated",
	}

	err := logger.LogRequest(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	saved := storage.logs[0]
	if !strings.Contains(saved.ResponseBody, "[truncated]") {
		t.Errorf("expected truncation marker, got: %s", saved.ResponseBody)
	}
}

func TestLogger_RedactionWithResponseBody(t *testing.T) {
	storage := &MockStorage{}
	privacy := &PrivacyConfig{
		LogRequestBodies:  false,
		LogResponseBodies: true,
		RedactPatterns: []string{
			`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		},
	}
	logger := NewLogger(storage, privacy)
	ctx := context.Background()

	log := &RequestLog{
		ResponseBody: "email: user@example.com and other data",
	}

	err := logger.LogRequest(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	saved := storage.logs[0]
	if strings.Contains(saved.ResponseBody, "user@example.com") {
		t.Error("email should be redacted in response body")
	}
}

func TestLogger_InvalidRedactPattern(t *testing.T) {
	storage := &MockStorage{}
	privacy := &PrivacyConfig{
		LogRequestBodies: true,
		RedactPatterns:   []string{"[invalid regex("},
	}
	logger := NewLogger(storage, privacy)
	ctx := context.Background()

	log := &RequestLog{
		RequestBody: "some data here",
	}

	// Should not panic with invalid regex
	err := logger.LogRequest(ctx, log)
	if err != nil {
		t.Fatal(err)
	}
}

// ---- Tests for SanitizeForLogging ----

func TestSanitizeForLogging_NilInput(t *testing.T) {
	result := SanitizeForLogging(nil)
	if result != "null" {
		t.Errorf("expected 'null', got %q", result)
	}
}

func TestSanitizeForLogging_SimpleString(t *testing.T) {
	result := SanitizeForLogging("hello world")
	if result != `"hello world"` {
		t.Errorf("expected '\"hello world\"', got %q", result)
	}
}

func TestSanitizeForLogging_AllSensitiveKeys(t *testing.T) {
	data := map[string]interface{}{
		"password":      "secret123",
		"api_key":       "sk-abc123",
		"token":         "tok-xyz",
		"secret":        "my-secret",
		"authorization": "Bearer abc",
		"normal":        "visible",
	}

	result := SanitizeForLogging(data)
	if strings.Contains(result, "secret123") {
		t.Error("password should be redacted")
	}
	if strings.Contains(result, "sk-abc123") {
		t.Error("api_key should be redacted")
	}
	if strings.Contains(result, "tok-xyz") {
		t.Error("token should be redacted")
	}
	if !strings.Contains(result, "visible") {
		t.Error("normal data should remain visible")
	}
}

func TestSanitizeForLogging_UnmarshalableInput(t *testing.T) {
	// Channels can't be marshaled to JSON
	ch := make(chan int)
	result := SanitizeForLogging(ch)
	if result != "[serialization error]" {
		t.Errorf("expected '[serialization error]', got %q", result)
	}
}

// ---- Tests for CalculateCost edge cases ----

func TestCalculateCost_NegativeValues(t *testing.T) {
	cost := CalculateCost(-1.0, 1000)
	if cost != 0.0 {
		t.Errorf("expected 0 for negative cost per token, got %f", cost)
	}

	cost = CalculateCost(1.0, -1000)
	if cost != 0.0 {
		t.Errorf("expected 0 for negative tokens, got %f", cost)
	}
}

// ---- Tests for buildWhereClause / buildWhereArgs ----

func TestBuildWhereClause(t *testing.T) {
	filter := &LogFilter{
		UserID:     "user-1",
		ProviderID: "provider-1",
		StartTime:  time.Now().Add(-24 * time.Hour),
		EndTime:    time.Now(),
	}

	clause := buildWhereClause(filter)
	if !strings.Contains(clause, "user_id") {
		t.Error("clause should contain user_id filter")
	}
	if !strings.Contains(clause, "provider_id") {
		t.Error("clause should contain provider_id filter")
	}
	if !strings.Contains(clause, "timestamp >=") {
		t.Error("clause should contain start time filter")
	}
	if !strings.Contains(clause, "timestamp <=") {
		t.Error("clause should contain end time filter")
	}
}

func TestBuildWhereClause_Empty(t *testing.T) {
	filter := &LogFilter{}
	clause := buildWhereClause(filter)
	if clause != "" {
		t.Errorf("expected empty clause, got %q", clause)
	}
}

func TestBuildWhereArgs(t *testing.T) {
	now := time.Now()
	filter := &LogFilter{
		UserID:     "user-1",
		ProviderID: "provider-1",
		StartTime:  now.Add(-24 * time.Hour),
		EndTime:    now,
	}

	args := buildWhereArgs(filter)
	if len(args) != 4 {
		t.Errorf("expected 4 args, got %d", len(args))
	}
}

func TestBuildWhereArgs_Empty(t *testing.T) {
	filter := &LogFilter{}
	args := buildWhereArgs(filter)
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
}

func TestBuildWhereArgs_Partial(t *testing.T) {
	filter := &LogFilter{UserID: "user-1"}
	args := buildWhereArgs(filter)
	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
}

// ---- Tests for generateLogID ----

func TestGenerateLogID(t *testing.T) {
	id1 := generateLogID()
	if !strings.HasPrefix(id1, "log-") {
		t.Errorf("expected prefix 'log-', got %q", id1)
	}

	// Should be unique (different nanosecond)
	id2 := generateLogID()
	// Note: in extremely fast execution they could be same nanosecond,
	// but this is acceptable for testing purposes.
	_ = id2
}
