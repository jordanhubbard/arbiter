package cache

import (
	"context"
	"testing"
	"time"
)

// mockBackend implements CacheBackend for testing
type mockBackend struct {
	entries  map[string]*Entry
	stats    *Stats
	getCount int
	setCount int
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		entries: make(map[string]*Entry),
		stats:   &Stats{},
	}
}

func (m *mockBackend) Get(ctx context.Context, key string) (*Entry, bool) {
	m.getCount++
	entry, ok := m.entries[key]
	if ok {
		m.stats.Hits++
	} else {
		m.stats.Misses++
	}
	return entry, ok
}

func (m *mockBackend) Set(ctx context.Context, key string, response interface{}, ttl time.Duration, metadata map[string]interface{}) error {
	m.setCount++
	m.entries[key] = &Entry{
		Key:       key,
		Response:  response,
		Metadata:  metadata,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (m *mockBackend) Delete(ctx context.Context, key string) {
	delete(m.entries, key)
}

func (m *mockBackend) Clear(ctx context.Context) {
	m.entries = make(map[string]*Entry)
}

func (m *mockBackend) GetStats(ctx context.Context) *Stats {
	return m.stats
}

func (m *mockBackend) InvalidateByProvider(ctx context.Context, providerID string) int {
	return 0
}

func (m *mockBackend) InvalidateByModel(ctx context.Context, modelName string) int {
	return 0
}

func (m *mockBackend) InvalidateByAge(ctx context.Context, maxAge time.Duration) int {
	return 0
}

func (m *mockBackend) InvalidateByPattern(ctx context.Context, pattern string) int {
	return 0
}

func TestCacheWithBackend_Get(t *testing.T) {
	backend := newMockBackend()
	config := DefaultConfig()

	c := &Cache{
		backend: backend,
		config:  config,
		entries: make(map[string]*Entry),
		stats:   &Stats{},
	}
	ctx := context.Background()

	// Backend miss
	_, found := c.Get(ctx, "nonexistent")
	if found {
		t.Error("Expected miss from backend")
	}
	if backend.getCount != 1 {
		t.Errorf("Expected 1 backend get call, got %d", backend.getCount)
	}

	// Add to backend and get hit
	backend.entries["key1"] = &Entry{
		Key:      "key1",
		Response: "test",
	}

	entry, found := c.Get(ctx, "key1")
	if !found {
		t.Fatal("Expected hit from backend")
	}
	if entry.Key != "key1" {
		t.Errorf("Expected key 'key1', got %q", entry.Key)
	}
	if backend.getCount != 2 {
		t.Errorf("Expected 2 backend get calls, got %d", backend.getCount)
	}
}

func TestNewFromRedis(t *testing.T) {
	// Create a minimal RedisCache struct (without real connection)
	// to test that NewFromRedis wires it up correctly
	rc := &RedisCache{
		config: &Config{
			Enabled:    true,
			DefaultTTL: 30 * time.Minute,
			MaxSize:    5000,
		},
		stats: &Stats{
			Hits:   10,
			Misses: 5,
		},
	}

	c := NewFromRedis(rc)
	if c == nil {
		t.Fatal("Expected non-nil cache")
	}
	if c.backend != rc {
		t.Error("Expected backend to be the RedisCache")
	}
	if c.config != rc.config {
		t.Error("Expected config to match RedisCache config")
	}
	if c.stats != rc.stats {
		t.Error("Expected stats to match RedisCache stats")
	}
}
