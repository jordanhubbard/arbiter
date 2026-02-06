package analytics

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDailyBudgetAlert(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Add logs that exceed daily budget
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Use timestamps relative to now (going backwards) to avoid timing issues
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i+1) * time.Minute)
		// Ensure we don't go before start of day
		if ts.Before(startOfDay) {
			ts = startOfDay.Add(time.Duration(i) * time.Minute)
		}
		if err := storage.SaveLog(ctx, &RequestLog{
			ID:        fmt.Sprintf("log-%d", i),
			Timestamp: ts,
			UserID:    "user-test",
			CostUSD:   25.0, // Total: $125, exceeds $100 budget
		}); err != nil {
			t.Fatalf("Failed to save log: %v", err)
		}
	}

	config := &AlertConfig{
		UserID:         "user-test",
		DailyBudgetUSD: 100.0,
	}

	checker := NewAlertChecker(storage, config)
	alerts, err := checker.CheckAlerts(ctx)

	if err != nil {
		t.Fatalf("CheckAlerts failed: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatal("Expected daily budget alert, got none")
	}

	alert := alerts[0]
	if alert.Type != "budget_exceeded" {
		t.Errorf("Expected type 'budget_exceeded', got '%s'", alert.Type)
	}
	if alert.Severity != "warning" {
		t.Errorf("Expected severity 'warning', got '%s'", alert.Severity)
	}
	if alert.CurrentCost != 125.0 {
		t.Errorf("Expected current cost 125.0, got %.2f", alert.CurrentCost)
	}
}

func TestMonthlyBudgetAlert(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	now := time.Now()
	// Add logs in the past few hours (all within current month and before "now")
	// to ensure they're included in the query
	for i := 0; i < 10; i++ {
		if err := storage.SaveLog(ctx, &RequestLog{
			ID:        fmt.Sprintf("log-%d", i),
			Timestamp: now.Add(-time.Duration(i+1) * time.Hour), // Go backwards from now
			UserID:    "user-test",
			CostUSD:   250.0, // Total: $2500, exceeds $2000 budget
		}); err != nil {
			t.Fatalf("Failed to save log: %v", err)
		}
	}

	config := &AlertConfig{
		UserID:           "user-test",
		MonthlyBudgetUSD: 2000.0,
	}

	checker := NewAlertChecker(storage, config)

	// Debug: Check what stats we're getting
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	stats, statsErr := storage.GetLogStats(ctx, &LogFilter{
		UserID:    "user-test",
		StartTime: startOfMonth,
		EndTime:   now,
	})
	if statsErr != nil {
		t.Fatalf("GetLogStats failed: %v", statsErr)
	}
	t.Logf("Stats: TotalRequests=%d, TotalCost=$%.2f, Budget=$%.2f",
		stats.TotalRequests, stats.TotalCostUSD, config.MonthlyBudgetUSD)

	alerts, err := checker.CheckAlerts(ctx)

	if err != nil {
		t.Fatalf("CheckAlerts failed: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatalf("Expected monthly budget alert, got none. Total cost: $%.2f, Budget: $%.2f",
			stats.TotalCostUSD, config.MonthlyBudgetUSD)
	}

	alert := alerts[0]
	if alert.Type != "budget_exceeded" {
		t.Errorf("Expected type 'budget_exceeded', got '%s'", alert.Type)
	}
	if alert.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", alert.Severity)
	}
}

func TestAnomalyDetection(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Add historical logs (last 7 days) with normal spending
	for i := 1; i <= 7; i++ {
		day := startOfToday.Add(-time.Duration(i*24) * time.Hour)
		if err := storage.SaveLog(ctx, &RequestLog{
			ID:        fmt.Sprintf("log-hist-%d", i),
			Timestamp: day,
			UserID:    "user-test",
			CostUSD:   10.0, // $10/day average
		}); err != nil {
			t.Fatalf("Failed to save log: %v", err)
		}
	}

	// Add today's logs with unusual spending (use timestamps before now)
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i+1) * time.Minute)
		// Ensure we don't go before start of day
		if ts.Before(startOfToday) {
			ts = startOfToday.Add(time.Duration(i) * time.Minute)
		}
		if err := storage.SaveLog(ctx, &RequestLog{
			ID:        fmt.Sprintf("log-today-%d", i),
			Timestamp: ts,
			UserID:    "user-test",
			CostUSD:   5.0, // Total: $25 today vs $10 average (2.5x)
		}); err != nil {
			t.Fatalf("Failed to save log: %v", err)
		}
	}

	config := &AlertConfig{
		UserID:           "user-test",
		AnomalyThreshold: 2.0, // Alert if 2x normal
	}

	checker := NewAlertChecker(storage, config)
	alerts, err := checker.CheckAlerts(ctx)

	if err != nil {
		t.Fatalf("CheckAlerts failed: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatal("Expected anomaly alert, got none")
	}

	alert := alerts[0]
	if alert.Type != "anomaly_detected" {
		t.Errorf("Expected type 'anomaly_detected', got '%s'", alert.Type)
	}
	if alert.CurrentCost != 25.0 {
		t.Errorf("Expected current cost 25.0, got %.2f", alert.CurrentCost)
	}
}

func TestNoAlertsWhenWithinBudget(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Add logs within budget
	if err := storage.SaveLog(ctx, &RequestLog{
		ID:        "log-1",
		Timestamp: startOfDay,
		UserID:    "user-test",
		CostUSD:   50.0, // Under $100 budget
	}); err != nil {
		t.Fatalf("Failed to save log: %v", err)
	}

	config := &AlertConfig{
		UserID:         "user-test",
		DailyBudgetUSD: 100.0,
	}

	checker := NewAlertChecker(storage, config)
	alerts, err := checker.CheckAlerts(ctx)

	if err != nil {
		t.Fatalf("CheckAlerts failed: %v", err)
	}

	if len(alerts) > 0 {
		t.Errorf("Expected no alerts, got %d", len(alerts))
	}
}

func TestSMTPConfigLoading(t *testing.T) {
	// Test with no SMTP configuration
	config := loadSMTPConfigFromEnv()
	if config != nil {
		t.Error("Expected nil config when SMTP_HOST not set")
	}

	// Test with SMTP configuration
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USERNAME", "test@example.com")
	t.Setenv("SMTP_PASSWORD", "password123")
	t.Setenv("SMTP_FROM", "noreply@loom.dev")
	t.Setenv("SMTP_USE_TLS", "true")

	config = loadSMTPConfigFromEnv()
	if config == nil {
		t.Fatal("Expected config to be loaded")
	}

	if config.Host != "smtp.example.com" {
		t.Errorf("Expected host smtp.example.com, got %s", config.Host)
	}
	if config.Port != 587 {
		t.Errorf("Expected port 587, got %d", config.Port)
	}
	if config.Username != "test@example.com" {
		t.Errorf("Expected username test@example.com, got %s", config.Username)
	}
	if !config.UseTLS {
		t.Error("Expected UseTLS to be true")
	}
}

func TestEmailBodyGeneration(t *testing.T) {
	alert := &Alert{
		ID:          "alert-test-123",
		UserID:      "user-test",
		Type:        "budget_exceeded",
		Severity:    "warning",
		Message:     "Daily budget exceeded: $150.00 / $100.00 (150%)",
		CurrentCost: 150.0,
		Threshold:   100.0,
		TriggeredAt: time.Now(),
	}

	body := buildEmailBody(alert)

	// Check that key elements are present in the HTML
	if !containsString(body, "Loom Alert") {
		t.Error("Email body missing title")
	}
	if !containsString(body, "warning") {
		t.Error("Email body missing severity")
	}
	if !containsString(body, "budget_exceeded") {
		t.Error("Email body missing alert type")
	}
	if !containsString(body, "$150.00") {
		t.Error("Email body missing current cost")
	}
	if !containsString(body, "$100.00") {
		t.Error("Email body missing threshold")
	}
	if !containsString(body, "alert-test-123") {
		t.Error("Email body missing alert ID")
	}

	// Check HTML structure
	if !containsString(body, "<!DOCTYPE html>") {
		t.Error("Email body missing HTML doctype")
	}
	if !containsString(body, "</html>") {
		t.Error("Email body missing closing HTML tag")
	}
}

func TestEmailNotificationDisabled(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Add logs that exceed budget
	for i := 0; i < 3; i++ {
		ts := now.Add(-time.Duration(i+1) * time.Minute)
		if ts.Before(startOfDay) {
			ts = startOfDay.Add(time.Duration(i) * time.Minute)
		}
		_ = storage.SaveLog(ctx, &RequestLog{
			ID:        fmt.Sprintf("log-%d", i),
			Timestamp: ts,
			UserID:    "user-test",
			CostUSD:   50.0, // Total: $150
		})
	}

	config := &AlertConfig{
		UserID:            "user-test",
		DailyBudgetUSD:    100.0,
		EnableEmailAlerts: false, // Email alerts disabled
		EmailAddress:      "test@example.com",
	}

	checker := NewAlertChecker(storage, config)
	alerts, err := checker.CheckAlerts(ctx)

	if err != nil {
		t.Fatalf("CheckAlerts failed: %v", err)
	}

	// Should still detect alert, but not send email (test passes if no crash)
	if len(alerts) == 0 {
		t.Fatal("Expected budget alert to be detected")
	}
}

// Helper function to check if string contains substring
func containsString(str, substr string) bool {
	return len(str) > 0 && len(substr) > 0 &&
		(str == substr || len(str) >= len(substr) &&
			(str[:len(substr)] == substr ||
				str[len(str)-len(substr):] == substr ||
				findInString(str, substr)))
}

func findInString(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
