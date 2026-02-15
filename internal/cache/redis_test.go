package cache

import (
	"context"
	"testing"
	"time"
)

// These tests cover RedisCache methods using a disabled config,
// which exercise the early-return paths without requiring a Redis connection.

func newDisabledRedisCache() *RedisCache {
	return &RedisCache{
		config: &Config{Enabled: false},
		stats:  &Stats{},
	}
}

func TestRedisCache_Get_Disabled(t *testing.T) {
	rc := newDisabledRedisCache()
	ctx := context.Background()

	entry, found := rc.Get(ctx, "any-key")
	if found {
		t.Error("Expected no result when disabled")
	}
	if entry != nil {
		t.Error("Expected nil entry when disabled")
	}
}

func TestRedisCache_Set_Disabled(t *testing.T) {
	rc := newDisabledRedisCache()
	ctx := context.Background()

	err := rc.Set(ctx, "key", "value", 1*time.Hour, nil)
	if err != nil {
		t.Errorf("Expected nil error when disabled, got %v", err)
	}
}

func TestRedisCache_Delete_Disabled(t *testing.T) {
	rc := newDisabledRedisCache()
	ctx := context.Background()

	// Should not panic
	rc.Delete(ctx, "any-key")
}

func TestRedisCache_Clear_Disabled(t *testing.T) {
	rc := newDisabledRedisCache()
	ctx := context.Background()

	// Should not panic
	rc.Clear(ctx)
}

func TestRedisCache_InvalidateByProvider_Disabled(t *testing.T) {
	rc := newDisabledRedisCache()
	ctx := context.Background()

	removed := rc.InvalidateByProvider(ctx, "provider-1")
	if removed != 0 {
		t.Errorf("Expected 0 removed when disabled, got %d", removed)
	}
}

func TestRedisCache_InvalidateByModel_Disabled(t *testing.T) {
	rc := newDisabledRedisCache()
	ctx := context.Background()

	removed := rc.InvalidateByModel(ctx, "model-1")
	if removed != 0 {
		t.Errorf("Expected 0 removed when disabled, got %d", removed)
	}
}

func TestRedisCache_InvalidateByAge_Disabled(t *testing.T) {
	rc := newDisabledRedisCache()
	ctx := context.Background()

	removed := rc.InvalidateByAge(ctx, 1*time.Hour)
	if removed != 0 {
		t.Errorf("Expected 0 removed when disabled, got %d", removed)
	}
}

func TestRedisCache_InvalidateByPattern_Disabled(t *testing.T) {
	rc := newDisabledRedisCache()
	ctx := context.Background()

	removed := rc.InvalidateByPattern(ctx, "pattern*")
	if removed != 0 {
		t.Errorf("Expected 0 removed when disabled, got %d", removed)
	}
}

func TestRedisCache_Set_DefaultTTL(t *testing.T) {
	// Disabled, so won't actually talk to Redis
	rc := &RedisCache{
		config: &Config{
			Enabled:    false,
			DefaultTTL: 30 * time.Minute,
		},
		stats: &Stats{},
	}
	ctx := context.Background()

	err := rc.Set(ctx, "key", "value", 0, nil)
	if err != nil {
		t.Errorf("Expected nil error when disabled, got %v", err)
	}
}

func TestNewRedisCache_InvalidURL(t *testing.T) {
	_, err := NewRedisCache("not-a-valid-url", nil)
	if err == nil {
		t.Fatal("Expected error for invalid Redis URL")
	}
}
