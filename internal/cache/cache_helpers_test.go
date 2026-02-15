package cache

import (
	"context"
	"testing"
	"time"
)

func TestNew_NilConfig(t *testing.T) {
	c := New(nil)
	if c == nil {
		t.Fatal("Expected non-nil cache")
	}
	if c.config == nil {
		t.Fatal("Expected non-nil config")
	}
	if !c.config.Enabled {
		t.Error("Default config should be enabled")
	}
}

func TestNew_DisabledCleanup(t *testing.T) {
	config := &Config{
		Enabled:       true,
		DefaultTTL:    1 * time.Hour,
		MaxSize:       100,
		CleanupPeriod: 0, // No cleanup
	}
	c := New(config)
	if c == nil {
		t.Fatal("Expected non-nil cache")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("Expected non-nil config")
	}
	if !config.Enabled {
		t.Error("Expected enabled=true")
	}
	if config.DefaultTTL != 1*time.Hour {
		t.Errorf("Expected DefaultTTL=1h, got %v", config.DefaultTTL)
	}
	if config.MaxSize != 10000 {
		t.Errorf("Expected MaxSize=10000, got %d", config.MaxSize)
	}
	if config.MaxMemoryMB != 500 {
		t.Errorf("Expected MaxMemoryMB=500, got %d", config.MaxMemoryMB)
	}
	if config.CleanupPeriod != 5*time.Minute {
		t.Errorf("Expected CleanupPeriod=5m, got %v", config.CleanupPeriod)
	}
}

func TestSet_DefaultTTL(t *testing.T) {
	config := &Config{
		Enabled:    true,
		DefaultTTL: 2 * time.Hour,
		MaxSize:    100,
	}
	c := New(config)
	ctx := context.Background()

	// Set with ttl=0 should use default TTL
	err := c.Set(ctx, "default-ttl-key", "value", 0, nil)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	entry, found := c.Get(ctx, "default-ttl-key")
	if !found {
		t.Fatal("Expected cache hit")
	}
	// ExpiresAt should be roughly 2 hours from now
	expectedExpiry := time.Now().Add(2 * time.Hour)
	diff := entry.ExpiresAt.Sub(expectedExpiry)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("ExpiresAt should be about 2h from now, diff=%v", diff)
	}
}

func TestDeleteDisabled(t *testing.T) {
	config := &Config{Enabled: false}
	c := New(config)
	ctx := context.Background()

	// Should not panic when disabled
	c.Delete(ctx, "any-key")
}

func TestClearDisabled(t *testing.T) {
	config := &Config{Enabled: false}
	c := New(config)
	ctx := context.Background()

	// Should not panic when disabled
	c.Clear(ctx)
}

func TestInvalidateByProviderDisabled(t *testing.T) {
	config := &Config{Enabled: false}
	c := New(config)
	ctx := context.Background()

	removed := c.InvalidateByProvider(ctx, "any-provider")
	if removed != 0 {
		t.Errorf("Expected 0 removed when disabled, got %d", removed)
	}
}

func TestInvalidateByModelDisabled(t *testing.T) {
	config := &Config{Enabled: false}
	c := New(config)
	ctx := context.Background()

	removed := c.InvalidateByModel(ctx, "any-model")
	if removed != 0 {
		t.Errorf("Expected 0 removed when disabled, got %d", removed)
	}
}

func TestInvalidateByAgeDisabled(t *testing.T) {
	config := &Config{Enabled: false}
	c := New(config)
	ctx := context.Background()

	removed := c.InvalidateByAge(ctx, 1*time.Hour)
	if removed != 0 {
		t.Errorf("Expected 0 removed when disabled, got %d", removed)
	}
}

func TestInvalidateByPatternDisabled(t *testing.T) {
	config := &Config{Enabled: false}
	c := New(config)
	ctx := context.Background()

	removed := c.InvalidateByPattern(ctx, "any-pattern")
	if removed != 0 {
		t.Errorf("Expected 0 removed when disabled, got %d", removed)
	}
}

func TestGetStringFromMap(t *testing.T) {
	// Nil map
	result := getStringFromMap(nil, "key")
	if result != "" {
		t.Errorf("Expected empty string for nil map, got %q", result)
	}

	// Key not present
	m := map[string]interface{}{"other": "value"}
	result = getStringFromMap(m, "key")
	if result != "" {
		t.Errorf("Expected empty string for missing key, got %q", result)
	}

	// Key present but not a string
	m = map[string]interface{}{"key": 42}
	result = getStringFromMap(m, "key")
	if result != "" {
		t.Errorf("Expected empty string for non-string value, got %q", result)
	}

	// Key present and is a string
	m = map[string]interface{}{"key": "hello"}
	result = getStringFromMap(m, "key")
	if result != "hello" {
		t.Errorf("Expected 'hello', got %q", result)
	}
}

func TestGetInt64FromMap(t *testing.T) {
	// Nil map
	result := getInt64FromMap(nil, "key")
	if result != 0 {
		t.Errorf("Expected 0 for nil map, got %d", result)
	}

	// Key not present
	m := map[string]interface{}{"other": int64(42)}
	result = getInt64FromMap(m, "key")
	if result != 0 {
		t.Errorf("Expected 0 for missing key, got %d", result)
	}

	// int64 value
	m = map[string]interface{}{"key": int64(42)}
	result = getInt64FromMap(m, "key")
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	// int value
	m = map[string]interface{}{"key": 100}
	result = getInt64FromMap(m, "key")
	if result != 100 {
		t.Errorf("Expected 100 from int, got %d", result)
	}

	// float64 value
	m = map[string]interface{}{"key": 3.14}
	result = getInt64FromMap(m, "key")
	if result != 3 {
		t.Errorf("Expected 3 from float64, got %d", result)
	}

	// Unsupported type
	m = map[string]interface{}{"key": "not-a-number"}
	result = getInt64FromMap(m, "key")
	if result != 0 {
		t.Errorf("Expected 0 for unsupported type, got %d", result)
	}
}

func TestGenerateKey_ErrorPath(t *testing.T) {
	// Channels cannot be marshaled to JSON
	request := make(chan int)
	_, err := GenerateKey("provider", "model", request)
	if err == nil {
		t.Fatal("Expected error for unmarshalable request")
	}
}

func TestCleanup(t *testing.T) {
	config := &Config{
		Enabled:    true,
		DefaultTTL: 1 * time.Hour,
		MaxSize:    100,
	}
	c := New(config)
	ctx := context.Background()

	// Add some entries
	c.Set(ctx, "key1", "value1", 1*time.Hour, nil)
	c.Set(ctx, "key2", "value2", 1*time.Hour, nil)

	// Manually expire one entry
	c.mu.Lock()
	if entry, ok := c.entries["key1"]; ok {
		entry.ExpiresAt = time.Now().Add(-1 * time.Minute)
	}
	c.mu.Unlock()

	// Run cleanup
	c.cleanup()

	// key1 should be removed
	c.mu.RLock()
	_, exists1 := c.entries["key1"]
	_, exists2 := c.entries["key2"]
	c.mu.RUnlock()

	if exists1 {
		t.Error("Expected key1 to be cleaned up")
	}
	if !exists2 {
		t.Error("Expected key2 to still exist")
	}
}

func TestEvictOldest(t *testing.T) {
	config := &Config{
		Enabled:    true,
		DefaultTTL: 1 * time.Hour,
		MaxSize:    100,
	}
	c := New(config)
	ctx := context.Background()

	// Add entries with different timestamps
	c.Set(ctx, "oldest", "value", 1*time.Hour, nil)
	time.Sleep(10 * time.Millisecond)
	c.Set(ctx, "middle", "value", 1*time.Hour, nil)
	time.Sleep(10 * time.Millisecond)
	c.Set(ctx, "newest", "value", 1*time.Hour, nil)

	initialEvictions := c.stats.Evictions

	c.mu.Lock()
	c.evictOldest()
	c.mu.Unlock()

	c.mu.RLock()
	_, existsOldest := c.entries["oldest"]
	_, existsMiddle := c.entries["middle"]
	_, existsNewest := c.entries["newest"]
	c.mu.RUnlock()

	if existsOldest {
		t.Error("Expected oldest entry to be evicted")
	}
	if !existsMiddle {
		t.Error("Expected middle entry to remain")
	}
	if !existsNewest {
		t.Error("Expected newest entry to remain")
	}
	if c.stats.Evictions != initialEvictions+1 {
		t.Errorf("Expected evictions to increment by 1")
	}
}

func TestEvictOldest_EmptyCache(t *testing.T) {
	config := &Config{
		Enabled:    true,
		DefaultTTL: 1 * time.Hour,
		MaxSize:    100,
	}
	c := New(config)

	initialEvictions := c.stats.Evictions

	c.mu.Lock()
	c.evictOldest()
	c.mu.Unlock()

	// Should not panic and evictions should not change
	if c.stats.Evictions != initialEvictions {
		t.Error("Evictions should not change for empty cache")
	}
}

func TestUpdateStats(t *testing.T) {
	config := &Config{
		Enabled:    true,
		DefaultTTL: 1 * time.Hour,
		MaxSize:    100,
	}
	c := New(config)

	// Cache hit
	c.updateStats(true, 50, 0)
	c.mu.RLock()
	if c.stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", c.stats.Hits)
	}
	if c.stats.TokensSaved != 50 {
		t.Errorf("Expected 50 tokens saved, got %d", c.stats.TokensSaved)
	}
	c.mu.RUnlock()

	// Cache miss
	c.updateStats(false, 0, 0)
	c.mu.RLock()
	if c.stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", c.stats.Misses)
	}
	c.mu.RUnlock()

	// Multiple hits
	c.updateStats(true, 100, 0)
	c.mu.RLock()
	if c.stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", c.stats.Hits)
	}
	if c.stats.TokensSaved != 150 {
		t.Errorf("Expected 150 tokens saved, got %d", c.stats.TokensSaved)
	}
	c.mu.RUnlock()
}

func TestGetStats_HitRateCalculation(t *testing.T) {
	config := &Config{
		Enabled:    true,
		DefaultTTL: 1 * time.Hour,
		MaxSize:    100,
	}
	c := New(config)
	ctx := context.Background()

	// No requests - hit rate should be 0
	stats := c.GetStats(ctx)
	if stats.HitRate != 0 {
		t.Errorf("Expected 0 hit rate with no requests, got %f", stats.HitRate)
	}

	// Add some hits and misses
	c.Set(ctx, "key1", "val", 1*time.Hour, nil)
	c.Get(ctx, "key1") // hit
	c.Get(ctx, "key1") // hit
	c.Get(ctx, "miss") // miss

	stats = c.GetStats(ctx)
	expectedRate := 2.0 / 3.0
	if stats.HitRate < expectedRate-0.01 || stats.HitRate > expectedRate+0.01 {
		t.Errorf("Expected hit rate ~%.2f, got %.2f", expectedRate, stats.HitRate)
	}
	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 entry, got %d", stats.TotalEntries)
	}
}

func TestInvalidateByProvider_NoMatch(t *testing.T) {
	c := New(DefaultConfig())
	ctx := context.Background()

	metadata := map[string]interface{}{
		"provider_id": "provider-a",
		"model_name":  "gpt-4",
	}
	c.Set(ctx, "key1", "val", 1*time.Hour, metadata)

	removed := c.InvalidateByProvider(ctx, "nonexistent-provider")
	if removed != 0 {
		t.Errorf("Expected 0 removed for nonexistent provider, got %d", removed)
	}
}

func TestInvalidateByModel_NoMatch(t *testing.T) {
	c := New(DefaultConfig())
	ctx := context.Background()

	metadata := map[string]interface{}{
		"provider_id": "provider-a",
		"model_name":  "gpt-4",
	}
	c.Set(ctx, "key1", "val", 1*time.Hour, metadata)

	removed := c.InvalidateByModel(ctx, "nonexistent-model")
	if removed != 0 {
		t.Errorf("Expected 0 removed for nonexistent model, got %d", removed)
	}
}

func TestInvalidateByAge_NoneOldEnough(t *testing.T) {
	c := New(DefaultConfig())
	ctx := context.Background()

	c.Set(ctx, "key1", "val", 1*time.Hour, nil)
	c.Set(ctx, "key2", "val", 1*time.Hour, nil)

	removed := c.InvalidateByAge(ctx, 24*time.Hour)
	if removed != 0 {
		t.Errorf("Expected 0 removed (nothing old enough), got %d", removed)
	}
}

func TestInvalidateByPattern_NoMatch(t *testing.T) {
	c := New(DefaultConfig())
	ctx := context.Background()

	c.Set(ctx, "alpha-1", "val", 1*time.Hour, nil)
	c.Set(ctx, "alpha-2", "val", 1*time.Hour, nil)

	removed := c.InvalidateByPattern(ctx, "beta")
	if removed != 0 {
		t.Errorf("Expected 0 removed for non-matching pattern, got %d", removed)
	}
}

func TestSetWithNilMetadata(t *testing.T) {
	c := New(DefaultConfig())
	ctx := context.Background()

	err := c.Set(ctx, "nil-meta", "value", 1*time.Hour, nil)
	if err != nil {
		t.Fatalf("Set with nil metadata should not error: %v", err)
	}

	entry, found := c.Get(ctx, "nil-meta")
	if !found {
		t.Fatal("Expected cache hit")
	}
	if entry.ProviderID != "" {
		t.Errorf("Expected empty provider_id for nil metadata, got %q", entry.ProviderID)
	}
	if entry.ModelName != "" {
		t.Errorf("Expected empty model_name for nil metadata, got %q", entry.ModelName)
	}
	if entry.TokensSaved != 0 {
		t.Errorf("Expected 0 tokens saved for nil metadata, got %d", entry.TokensSaved)
	}
}
