package analytics

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNewDatabaseStorage(t *testing.T) {
	db := newTestDB(t)
	storage, err := NewDatabaseStorage(db)
	if err != nil {
		t.Fatalf("NewDatabaseStorage failed: %v", err)
	}
	if storage == nil {
		t.Fatal("expected non-nil storage")
	}
}

func TestDatabaseStorage_SaveAndGetLogs(t *testing.T) {
	db := newTestDB(t)
	storage, err := NewDatabaseStorage(db)
	if err != nil {
		t.Fatalf("NewDatabaseStorage failed: %v", err)
	}

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	log1 := &RequestLog{
		ID:               "log-1",
		Timestamp:        now,
		UserID:           "user-alice",
		Method:           "POST",
		Path:             "/api/v1/chat",
		ProviderID:       "openai",
		ModelName:        "gpt-4",
		PromptTokens:     500,
		CompletionTokens: 200,
		TotalTokens:      700,
		LatencyMs:        150,
		StatusCode:       200,
		CostUSD:          0.05,
		Metadata:         map[string]string{"key1": "val1"},
	}

	err = storage.SaveLog(ctx, log1)
	if err != nil {
		t.Fatalf("SaveLog failed: %v", err)
	}

	log2 := &RequestLog{
		ID:          "log-2",
		Timestamp:   now.Add(-1 * time.Hour),
		UserID:      "user-bob",
		Method:      "POST",
		Path:        "/api/v1/chat",
		ProviderID:  "anthropic",
		ModelName:   "claude-3",
		TotalTokens: 500,
		LatencyMs:   200,
		StatusCode:  200,
		CostUSD:     0.03,
	}

	err = storage.SaveLog(ctx, log2)
	if err != nil {
		t.Fatalf("SaveLog failed: %v", err)
	}

	// Get all logs
	logs, err := storage.GetLogs(ctx, &LogFilter{})
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}

	// Filter by UserID
	logs, err = storage.GetLogs(ctx, &LogFilter{UserID: "user-alice"})
	if err != nil {
		t.Fatalf("GetLogs filtered failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log for alice, got %d", len(logs))
	}
	if logs[0].ID != "log-1" {
		t.Errorf("expected log-1, got %s", logs[0].ID)
	}

	// Filter by ProviderID
	logs, err = storage.GetLogs(ctx, &LogFilter{ProviderID: "anthropic"})
	if err != nil {
		t.Fatalf("GetLogs by provider failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log for anthropic, got %d", len(logs))
	}

	// Filter by time range
	logs, err = storage.GetLogs(ctx, &LogFilter{
		StartTime: now.Add(-30 * time.Minute),
		EndTime:   now.Add(1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("GetLogs by time range failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log in time range, got %d", len(logs))
	}

	// Test limit and offset
	logs, err = storage.GetLogs(ctx, &LogFilter{Limit: 1})
	if err != nil {
		t.Fatalf("GetLogs with limit failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log with limit, got %d", len(logs))
	}

	logs, err = storage.GetLogs(ctx, &LogFilter{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("GetLogs with offset failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log with offset, got %d", len(logs))
	}
}

func TestDatabaseStorage_GetLogStats(t *testing.T) {
	db := newTestDB(t)
	storage, err := NewDatabaseStorage(db)
	if err != nil {
		t.Fatalf("NewDatabaseStorage failed: %v", err)
	}

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert logs with different users and providers
	logs := []*RequestLog{
		{ID: "s1", Timestamp: now, UserID: "alice", ProviderID: "openai", TotalTokens: 100, CostUSD: 0.01, LatencyMs: 100, StatusCode: 200, Method: "POST", Path: "/api"},
		{ID: "s2", Timestamp: now, UserID: "alice", ProviderID: "openai", TotalTokens: 200, CostUSD: 0.02, LatencyMs: 200, StatusCode: 200, Method: "POST", Path: "/api"},
		{ID: "s3", Timestamp: now, UserID: "bob", ProviderID: "anthropic", TotalTokens: 300, CostUSD: 0.03, LatencyMs: 300, StatusCode: 500, Method: "POST", Path: "/api"},
	}

	for _, log := range logs {
		if err := storage.SaveLog(ctx, log); err != nil {
			t.Fatalf("SaveLog failed: %v", err)
		}
	}

	stats, err := storage.GetLogStats(ctx, &LogFilter{})
	if err != nil {
		t.Fatalf("GetLogStats failed: %v", err)
	}

	if stats.TotalRequests != 3 {
		t.Errorf("TotalRequests = %d, want 3", stats.TotalRequests)
	}
	if stats.TotalTokens != 600 {
		t.Errorf("TotalTokens = %d, want 600", stats.TotalTokens)
	}
	if stats.TotalCostUSD != 0.06 {
		t.Errorf("TotalCostUSD = %f, want 0.06", stats.TotalCostUSD)
	}

	// Error rate: 1 out of 3
	expectedErrorRate := 1.0 / 3.0
	if stats.ErrorRate < expectedErrorRate-0.01 || stats.ErrorRate > expectedErrorRate+0.01 {
		t.Errorf("ErrorRate = %f, want ~%f", stats.ErrorRate, expectedErrorRate)
	}

	// Per-user stats
	if stats.RequestsByUser["alice"] != 2 {
		t.Errorf("alice requests = %d, want 2", stats.RequestsByUser["alice"])
	}
	if stats.RequestsByUser["bob"] != 1 {
		t.Errorf("bob requests = %d, want 1", stats.RequestsByUser["bob"])
	}
	if stats.CostByUser["alice"] != 0.03 {
		t.Errorf("alice cost = %f, want 0.03", stats.CostByUser["alice"])
	}

	// Per-provider stats
	if stats.RequestsByProvider["openai"] != 2 {
		t.Errorf("openai requests = %d, want 2", stats.RequestsByProvider["openai"])
	}
	if stats.RequestsByProvider["anthropic"] != 1 {
		t.Errorf("anthropic requests = %d, want 1", stats.RequestsByProvider["anthropic"])
	}
	if stats.CostByProvider["openai"] != 0.03 {
		t.Errorf("openai cost = %f, want 0.03", stats.CostByProvider["openai"])
	}
}

func TestDatabaseStorage_GetLogStats_Filtered(t *testing.T) {
	db := newTestDB(t)
	storage, err := NewDatabaseStorage(db)
	if err != nil {
		t.Fatalf("NewDatabaseStorage failed: %v", err)
	}

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	_ = storage.SaveLog(ctx, &RequestLog{ID: "f1", Timestamp: now, UserID: "alice", ProviderID: "p1", TotalTokens: 100, CostUSD: 0.01, StatusCode: 200, Method: "POST", Path: "/api"})
	_ = storage.SaveLog(ctx, &RequestLog{ID: "f2", Timestamp: now, UserID: "bob", ProviderID: "p2", TotalTokens: 200, CostUSD: 0.02, StatusCode: 200, Method: "POST", Path: "/api"})

	// Filter by user
	stats, err := storage.GetLogStats(ctx, &LogFilter{UserID: "alice"})
	if err != nil {
		t.Fatalf("GetLogStats filtered failed: %v", err)
	}
	if stats.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want 1", stats.TotalRequests)
	}

	// Filter by provider
	stats, err = storage.GetLogStats(ctx, &LogFilter{ProviderID: "p2"})
	if err != nil {
		t.Fatalf("GetLogStats by provider failed: %v", err)
	}
	if stats.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want 1", stats.TotalRequests)
	}
}

func TestDatabaseStorage_GetLogStats_Empty(t *testing.T) {
	db := newTestDB(t)
	storage, err := NewDatabaseStorage(db)
	if err != nil {
		t.Fatalf("NewDatabaseStorage failed: %v", err)
	}

	ctx := context.Background()
	stats, err := storage.GetLogStats(ctx, &LogFilter{})
	if err != nil {
		t.Fatalf("GetLogStats on empty DB failed: %v", err)
	}
	if stats.TotalRequests != 0 {
		t.Errorf("TotalRequests = %d, want 0", stats.TotalRequests)
	}
}

func TestDatabaseStorage_DeleteOldLogs(t *testing.T) {
	db := newTestDB(t)
	storage, err := NewDatabaseStorage(db)
	if err != nil {
		t.Fatalf("NewDatabaseStorage failed: %v", err)
	}

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	_ = storage.SaveLog(ctx, &RequestLog{ID: "d1", Timestamp: now.Add(-48 * time.Hour), UserID: "u", Method: "POST", Path: "/api"})
	_ = storage.SaveLog(ctx, &RequestLog{ID: "d2", Timestamp: now.Add(-24 * time.Hour), UserID: "u", Method: "POST", Path: "/api"})
	_ = storage.SaveLog(ctx, &RequestLog{ID: "d3", Timestamp: now, UserID: "u", Method: "POST", Path: "/api"})

	deleted, err := storage.DeleteOldLogs(ctx, now.Add(-12*time.Hour))
	if err != nil {
		t.Fatalf("DeleteOldLogs failed: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}

	// Verify only 1 remains
	logs, err := storage.GetLogs(ctx, &LogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 remaining log, got %d", len(logs))
	}
}

func TestDatabaseStorage_SaveLog_NilMetadata(t *testing.T) {
	db := newTestDB(t)
	storage, err := NewDatabaseStorage(db)
	if err != nil {
		t.Fatalf("NewDatabaseStorage failed: %v", err)
	}

	ctx := context.Background()
	log := &RequestLog{
		ID:        "nil-meta",
		Timestamp: time.Now(),
		UserID:    "u",
		Method:    "POST",
		Path:      "/api",
		Metadata:  nil,
	}

	err = storage.SaveLog(ctx, log)
	if err != nil {
		t.Fatalf("SaveLog with nil metadata failed: %v", err)
	}

	// Verify we can read it back
	logs, err := storage.GetLogs(ctx, &LogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
}

func TestDatabaseStorage_SaveLog_WithMetadata(t *testing.T) {
	db := newTestDB(t)
	storage, err := NewDatabaseStorage(db)
	if err != nil {
		t.Fatalf("NewDatabaseStorage failed: %v", err)
	}

	ctx := context.Background()
	log := &RequestLog{
		ID:        "with-meta",
		Timestamp: time.Now(),
		UserID:    "u",
		Method:    "POST",
		Path:      "/api",
		Metadata:  map[string]string{"project": "loom", "version": "1.0"},
	}

	err = storage.SaveLog(ctx, log)
	if err != nil {
		t.Fatalf("SaveLog with metadata failed: %v", err)
	}

	logs, err := storage.GetLogs(ctx, &LogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Metadata == nil {
		t.Error("expected metadata to be preserved")
	} else {
		if logs[0].Metadata["project"] != "loom" {
			t.Errorf("metadata project = %q, want loom", logs[0].Metadata["project"])
		}
	}
}

func TestDatabaseStorage_GetLogStats_TokensByUser(t *testing.T) {
	db := newTestDB(t)
	storage, err := NewDatabaseStorage(db)
	if err != nil {
		t.Fatalf("NewDatabaseStorage failed: %v", err)
	}

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	_ = storage.SaveLog(ctx, &RequestLog{ID: "t1", Timestamp: now, UserID: "alice", ProviderID: "p1", TotalTokens: 1000, CostUSD: 0.10, LatencyMs: 100, StatusCode: 200, Method: "POST", Path: "/api"})
	_ = storage.SaveLog(ctx, &RequestLog{ID: "t2", Timestamp: now, UserID: "alice", ProviderID: "p1", TotalTokens: 2000, CostUSD: 0.20, LatencyMs: 150, StatusCode: 200, Method: "POST", Path: "/api"})

	stats, err := storage.GetLogStats(ctx, &LogFilter{})
	if err != nil {
		t.Fatalf("GetLogStats failed: %v", err)
	}

	if stats.TokensByUser["alice"] != 3000 {
		t.Errorf("alice tokens = %d, want 3000", stats.TokensByUser["alice"])
	}
	if stats.TokensByProvider["p1"] != 3000 {
		t.Errorf("p1 tokens = %d, want 3000", stats.TokensByProvider["p1"])
	}
}
