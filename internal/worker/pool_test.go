package worker

import (
	"context"
	"fmt"
	"testing"

	"github.com/jordanhubbard/loom/internal/provider"
	"github.com/jordanhubbard/loom/pkg/models"
)

func registerMockProvider(t *testing.T, registry *provider.Registry, id string) {
	t.Helper()
	err := registry.Upsert(&provider.ProviderConfig{
		ID:   id,
		Name: id,
		Type: "mock",
	})
	if err != nil {
		t.Fatalf("failed to register mock provider: %v", err)
	}
}

func TestNewPool(t *testing.T) {
	registry := provider.NewRegistry()
	pool := NewPool(registry, 5)

	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
	if pool.maxWorkers != 5 {
		t.Errorf("maxWorkers = %d, want 5", pool.maxWorkers)
	}
	if pool.workers == nil {
		t.Error("workers map should be initialized")
	}
}

func TestPool_SetDatabase(t *testing.T) {
	registry := provider.NewRegistry()
	pool := NewPool(registry, 5)

	// Set nil db
	pool.SetDatabase(nil)
	if pool.db != nil {
		t.Error("db should be nil")
	}
}

func TestPool_GetPoolStats_Empty(t *testing.T) {
	registry := provider.NewRegistry()
	pool := NewPool(registry, 10)

	stats := pool.GetPoolStats()

	if stats.TotalWorkers != 0 {
		t.Errorf("TotalWorkers = %d, want 0", stats.TotalWorkers)
	}
	if stats.MaxWorkers != 10 {
		t.Errorf("MaxWorkers = %d, want 10", stats.MaxWorkers)
	}
	if stats.IdleWorkers != 0 {
		t.Errorf("IdleWorkers = %d, want 0", stats.IdleWorkers)
	}
}

func TestPool_ListWorkers_Empty(t *testing.T) {
	registry := provider.NewRegistry()
	pool := NewPool(registry, 5)

	workers := pool.ListWorkers()
	if len(workers) != 0 {
		t.Errorf("len(workers) = %d, want 0", len(workers))
	}
}

func TestPool_GetIdleWorkers_Empty(t *testing.T) {
	registry := provider.NewRegistry()
	pool := NewPool(registry, 5)

	workers := pool.GetIdleWorkers()
	if len(workers) != 0 {
		t.Errorf("len(workers) = %d, want 0", len(workers))
	}
}

func TestPool_GetWorker_NotFound(t *testing.T) {
	registry := provider.NewRegistry()
	pool := NewPool(registry, 5)

	_, err := pool.GetWorker("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent worker")
	}
}

func TestPool_StopWorker_NotFound(t *testing.T) {
	registry := provider.NewRegistry()
	pool := NewPool(registry, 5)

	err := pool.StopWorker("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent worker")
	}
}

func TestPool_ExecuteTask_NotFound(t *testing.T) {
	registry := provider.NewRegistry()
	pool := NewPool(registry, 5)

	task := &Task{
		ID:          "task-1",
		Description: "test task",
	}

	_, err := pool.ExecuteTask(context.Background(), task, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent worker")
	}
}

func TestPool_SpawnWorker_NoProvider(t *testing.T) {
	registry := provider.NewRegistry()
	pool := NewPool(registry, 5)

	agent := &models.Agent{
		ID:   "agent-1",
		Name: "Test Agent",
	}

	_, err := pool.SpawnWorker(agent, "nonexistent-provider")
	if err == nil {
		t.Error("expected error for nonexistent provider")
	}
}

func TestPool_SpawnWorker_MaxReached(t *testing.T) {
	registry := provider.NewRegistry()
	pool := NewPool(registry, 0) // Max 0 workers

	agent := &models.Agent{
		ID:   "agent-1",
		Name: "Test Agent",
	}

	_, err := pool.SpawnWorker(agent, "some-provider")
	if err == nil {
		t.Error("expected error when max workers reached")
	}
}

func TestPool_SpawnWorker_WithMockProvider(t *testing.T) {
	registry := provider.NewRegistry()
	registerMockProvider(t, registry, "mock-1")
	pool := NewPool(registry, 5)

	agent := &models.Agent{
		ID:          "agent-1",
		Name:        "Test Agent",
		PersonaName: "test-persona",
	}

	worker, err := pool.SpawnWorker(agent, "mock-1")
	if err != nil {
		t.Fatalf("SpawnWorker() error = %v", err)
	}
	if worker == nil {
		t.Fatal("expected non-nil worker")
	}

	// Verify stats
	stats := pool.GetPoolStats()
	if stats.TotalWorkers != 1 {
		t.Errorf("TotalWorkers = %d, want 1", stats.TotalWorkers)
	}

	// Verify we can get the worker
	w, err := pool.GetWorker("agent-1")
	if err != nil {
		t.Fatalf("GetWorker() error = %v", err)
	}
	if w != worker {
		t.Error("expected same worker")
	}

	// List should include it
	workers := pool.ListWorkers()
	if len(workers) != 1 {
		t.Errorf("len(ListWorkers()) = %d, want 1", len(workers))
	}
}

func TestPool_SpawnWorker_Duplicate(t *testing.T) {
	registry := provider.NewRegistry()
	registerMockProvider(t, registry, "mock-1")
	pool := NewPool(registry, 5)

	agent := &models.Agent{
		ID:          "agent-1",
		Name:        "Test Agent",
		PersonaName: "test-persona",
	}

	_, err := pool.SpawnWorker(agent, "mock-1")
	if err != nil {
		t.Fatalf("First SpawnWorker() error = %v", err)
	}

	// Second spawn should fail
	_, err = pool.SpawnWorker(agent, "mock-1")
	if err == nil {
		t.Error("expected error for duplicate worker")
	}
}

func TestPool_StopWorker(t *testing.T) {
	registry := provider.NewRegistry()
	registerMockProvider(t, registry, "mock-1")
	pool := NewPool(registry, 5)

	agent := &models.Agent{
		ID:          "agent-1",
		Name:        "Test Agent",
		PersonaName: "test-persona",
	}

	_, err := pool.SpawnWorker(agent, "mock-1")
	if err != nil {
		t.Fatalf("SpawnWorker() error = %v", err)
	}

	// Stop the worker
	err = pool.StopWorker("agent-1")
	if err != nil {
		t.Fatalf("StopWorker() error = %v", err)
	}

	// Should be gone
	_, err = pool.GetWorker("agent-1")
	if err == nil {
		t.Error("expected error after stopping worker")
	}

	stats := pool.GetPoolStats()
	if stats.TotalWorkers != 0 {
		t.Errorf("TotalWorkers = %d, want 0 after stop", stats.TotalWorkers)
	}
}

func TestPool_StopAll(t *testing.T) {
	registry := provider.NewRegistry()
	registerMockProvider(t, registry, "mock-1")
	pool := NewPool(registry, 5)

	// Spawn multiple workers
	for i := 0; i < 3; i++ {
		agent := &models.Agent{
			ID:          fmt.Sprintf("agent-%d", i),
			Name:        fmt.Sprintf("Agent %d", i),
			PersonaName: "test-persona",
		}
		_, err := pool.SpawnWorker(agent, "mock-1")
		if err != nil {
			t.Fatalf("SpawnWorker(%d) error = %v", i, err)
		}
	}

	if stats := pool.GetPoolStats(); stats.TotalWorkers != 3 {
		t.Errorf("TotalWorkers = %d, want 3", stats.TotalWorkers)
	}

	pool.StopAll()

	if stats := pool.GetPoolStats(); stats.TotalWorkers != 0 {
		t.Errorf("TotalWorkers = %d, want 0 after StopAll", stats.TotalWorkers)
	}
}

func TestPool_GetIdleWorkers(t *testing.T) {
	registry := provider.NewRegistry()
	registerMockProvider(t, registry, "mock-1")
	pool := NewPool(registry, 5)

	agent := &models.Agent{
		ID:          "agent-idle",
		Name:        "Idle Agent",
		PersonaName: "test-persona",
	}

	_, err := pool.SpawnWorker(agent, "mock-1")
	if err != nil {
		t.Fatalf("SpawnWorker() error = %v", err)
	}

	// New worker should be idle
	idle := pool.GetIdleWorkers()
	if len(idle) != 1 {
		t.Errorf("len(GetIdleWorkers()) = %d, want 1", len(idle))
	}
}
