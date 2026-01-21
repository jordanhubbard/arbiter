package analytics

import (
	"context"
	"testing"
	"time"
)

func TestCostCalculation(t *testing.T) {
	tests := []struct {
		name          string
		costPerMToken float64
		totalTokens   int64
		expectedCost  float64
	}{
		{
			name:          "Standard calculation",
			costPerMToken: 2.0, // $2 per million tokens
			totalTokens:   500000,
			expectedCost:  1.0, // $1
		},
		{
			name:          "Small usage",
			costPerMToken: 0.5,
			totalTokens:   1000,
			expectedCost:  0.0005, // $0.0005
		},
		{
			name:          "Large usage",
			costPerMToken: 5.0,
			totalTokens:   10000000,
			expectedCost:  50.0, // $50
		},
		{
			name:          "Zero tokens",
			costPerMToken: 2.0,
			totalTokens:   0,
			expectedCost:  0.0,
		},
		{
			name:          "Zero cost",
			costPerMToken: 0.0,
			totalTokens:   1000,
			expectedCost:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := CalculateCost(tt.costPerMToken, tt.totalTokens)
			if cost != tt.expectedCost {
				t.Errorf("Expected cost %.6f, got %.6f", tt.expectedCost, cost)
			}
		})
	}
}

func TestCostTrackingPerUser(t *testing.T) {
	storage := NewInMemoryStorage()
	logger := NewLogger(storage, DefaultPrivacyConfig())
	ctx := context.Background()

	// Log requests for different users
	logs := []*RequestLog{
		{
			ID:         "log-1",
			Timestamp:  time.Now(),
			UserID:     "user-alice",
			Method:     "POST",
			Path:       "/api/v1/chat/completions",
			ProviderID: "provider-1",
			ModelName:  "gpt-4",
			TotalTokens: 1000,
			CostUSD:    0.03, // $0.03
		},
		{
			ID:         "log-2",
			Timestamp:  time.Now(),
			UserID:     "user-alice",
			Method:     "POST",
			Path:       "/api/v1/chat/completions",
			ProviderID: "provider-1",
			ModelName:  "gpt-4",
			TotalTokens: 2000,
			CostUSD:    0.06, // $0.06
		},
		{
			ID:         "log-3",
			Timestamp:  time.Now(),
			UserID:     "user-bob",
			Method:     "POST",
			Path:       "/api/v1/chat/completions",
			ProviderID: "provider-2",
			ModelName:  "claude-3",
			TotalTokens: 1500,
			CostUSD:    0.045, // $0.045
		},
	}

	for _, log := range logs {
		if err := logger.LogRequest(ctx, log); err != nil {
			t.Fatalf("Failed to log request: %v", err)
		}
	}

	// Get stats for all users
	stats, err := logger.GetStats(ctx, &LogFilter{})
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Verify total cost
	expectedTotalCost := 0.03 + 0.06 + 0.045
	if stats.TotalCostUSD != expectedTotalCost {
		t.Errorf("Expected total cost %.3f, got %.3f", expectedTotalCost, stats.TotalCostUSD)
	}

	// Verify cost per user
	if stats.CostByUser["user-alice"] != 0.09 {
		t.Errorf("Expected user-alice cost 0.09, got %.3f", stats.CostByUser["user-alice"])
	}
	if stats.CostByUser["user-bob"] != 0.045 {
		t.Errorf("Expected user-bob cost 0.045, got %.3f", stats.CostByUser["user-bob"])
	}
}

func TestCostTrackingPerProvider(t *testing.T) {
	storage := NewInMemoryStorage()
	logger := NewLogger(storage, DefaultPrivacyConfig())
	ctx := context.Background()

	// Log requests for different providers
	logs := []*RequestLog{
		{
			ID:         "log-1",
			Timestamp:  time.Now(),
			UserID:     "user-alice",
			Method:     "POST",
			Path:       "/api/v1/chat/completions",
			ProviderID: "provider-openai",
			ModelName:  "gpt-4",
			TotalTokens: 1000,
			CostUSD:    0.03,
		},
		{
			ID:         "log-2",
			Timestamp:  time.Now(),
			UserID:     "user-alice",
			Method:     "POST",
			Path:       "/api/v1/chat/completions",
			ProviderID: "provider-openai",
			ModelName:  "gpt-4",
			TotalTokens: 2000,
			CostUSD:    0.06,
		},
		{
			ID:         "log-3",
			Timestamp:  time.Now(),
			UserID:     "user-bob",
			Method:     "POST",
			Path:       "/api/v1/chat/completions",
			ProviderID: "provider-anthropic",
			ModelName:  "claude-3",
			TotalTokens: 1500,
			CostUSD:    0.045,
		},
	}

	for _, log := range logs {
		if err := logger.LogRequest(ctx, log); err != nil {
			t.Fatalf("Failed to log request: %v", err)
		}
	}

	// Get stats for all providers
	stats, err := logger.GetStats(ctx, &LogFilter{})
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Verify cost per provider
	if stats.CostByProvider["provider-openai"] != 0.09 {
		t.Errorf("Expected provider-openai cost 0.09, got %.3f", stats.CostByProvider["provider-openai"])
	}
	if stats.CostByProvider["provider-anthropic"] != 0.045 {
		t.Errorf("Expected provider-anthropic cost 0.045, got %.3f", stats.CostByProvider["provider-anthropic"])
	}
}

func TestCostTrackingTimeRangeFiltering(t *testing.T) {
	storage := NewInMemoryStorage()
	logger := NewLogger(storage, DefaultPrivacyConfig())
	ctx := context.Background()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	// Log requests at different times
	logs := []*RequestLog{
		{
			ID:         "log-old",
			Timestamp:  lastWeek,
			UserID:     "user-alice",
			ProviderID: "provider-1",
			TotalTokens: 1000,
			CostUSD:    0.01,
		},
		{
			ID:         "log-yesterday",
			Timestamp:  yesterday,
			UserID:     "user-alice",
			ProviderID: "provider-1",
			TotalTokens: 2000,
			CostUSD:    0.02,
		},
		{
			ID:         "log-today",
			Timestamp:  now,
			UserID:     "user-alice",
			ProviderID: "provider-1",
			TotalTokens: 3000,
			CostUSD:    0.03,
		},
	}

	for _, log := range logs {
		if err := logger.LogRequest(ctx, log); err != nil {
			t.Fatalf("Failed to log request: %v", err)
		}
	}

	// Get stats for last 24 hours
	stats, err := logger.GetStats(ctx, &LogFilter{
		StartTime: yesterday.Add(-1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Should only include yesterday and today logs
	expectedCost := 0.02 + 0.03
	if stats.TotalCostUSD != expectedCost {
		t.Errorf("Expected cost %.3f for last 24h, got %.3f", expectedCost, stats.TotalCostUSD)
	}
	if stats.TotalRequests != 2 {
		t.Errorf("Expected 2 requests for last 24h, got %d", stats.TotalRequests)
	}
}

// InMemoryStorage is a simple in-memory implementation for testing
type InMemoryStorage struct {
	logs []*RequestLog
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		logs: make([]*RequestLog, 0),
	}
}

func (s *InMemoryStorage) SaveLog(ctx context.Context, log *RequestLog) error {
	s.logs = append(s.logs, log)
	return nil
}

func (s *InMemoryStorage) GetLogs(ctx context.Context, filter *LogFilter) ([]*RequestLog, error) {
	filtered := make([]*RequestLog, 0)
	for _, log := range s.logs {
		if filter.UserID != "" && log.UserID != filter.UserID {
			continue
		}
		if filter.ProviderID != "" && log.ProviderID != filter.ProviderID {
			continue
		}
		if !filter.StartTime.IsZero() && log.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && log.Timestamp.After(filter.EndTime) {
			continue
		}
		filtered = append(filtered, log)
	}
	return filtered, nil
}

func (s *InMemoryStorage) GetLogStats(ctx context.Context, filter *LogFilter) (*LogStats, error) {
	logs, err := s.GetLogs(ctx, filter)
	if err != nil {
		return nil, err
	}

	stats := &LogStats{
		RequestsByUser:     make(map[string]int64),
		RequestsByProvider: make(map[string]int64),
		CostByProvider:     make(map[string]float64),
		CostByUser:         make(map[string]float64),
	}

	var totalLatency int64
	var errorCount int64

	for _, log := range logs {
		stats.TotalRequests++
		stats.TotalTokens += log.TotalTokens
		stats.TotalCostUSD += log.CostUSD
		totalLatency += log.LatencyMs

		if log.StatusCode >= 400 {
			errorCount++
		}

		if log.UserID != "" {
			stats.RequestsByUser[log.UserID]++
			stats.CostByUser[log.UserID] += log.CostUSD
		}

		if log.ProviderID != "" {
			stats.RequestsByProvider[log.ProviderID]++
			stats.CostByProvider[log.ProviderID] += log.CostUSD
		}
	}

	if stats.TotalRequests > 0 {
		stats.AvgLatencyMs = float64(totalLatency) / float64(stats.TotalRequests)
		stats.ErrorRate = float64(errorCount) / float64(stats.TotalRequests)
	}

	return stats, nil
}

func (s *InMemoryStorage) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	newLogs := make([]*RequestLog, 0)
	deleted := int64(0)
	for _, log := range s.logs {
		if log.Timestamp.Before(before) {
			deleted++
		} else {
			newLogs = append(newLogs, log)
		}
	}
	s.logs = newLogs
	return deleted, nil
}
