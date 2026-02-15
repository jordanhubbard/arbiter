package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/provider"
	"github.com/jordanhubbard/loom/internal/worker"
	"github.com/jordanhubbard/loom/pkg/models"
)

func setupWorkerManager(t *testing.T) *WorkerManager {
	t.Helper()
	providerRegistry := provider.NewRegistry()
	return NewWorkerManager(10, providerRegistry, nil)
}

func TestNewWorkerManager(t *testing.T) {
	providerRegistry := provider.NewRegistry()

	m := NewWorkerManager(10, providerRegistry, nil)

	if m == nil {
		t.Fatal("NewWorkerManager() returned nil")
	}
	if m.maxAgents != 10 {
		t.Errorf("maxAgents = %d, want 10", m.maxAgents)
	}
	if m.agents == nil {
		t.Error("agents map not initialized")
	}
	if m.workerPool == nil {
		t.Error("workerPool not initialized")
	}
	if m.providerRegistry != providerRegistry {
		t.Error("providerRegistry not set")
	}
}

func TestWorkerManager_CreateAgent(t *testing.T) {
	tests := []struct {
		name        string
		agentName   string
		personaName string
		projectID   string
		role        string
		wantStatus  string
	}{
		{
			name:        "create agent with full params",
			agentName:   "Test Agent",
			personaName: "default/qa-engineer",
			projectID:   "proj-1",
			role:        "QA Engineer",
			wantStatus:  "paused",
		},
		{
			name:        "create agent with empty name",
			agentName:   "",
			personaName: "default/web-designer",
			projectID:   "proj-1",
			role:        "Designer",
			wantStatus:  "paused",
		},
		{
			name:        "create agent with empty role",
			agentName:   "My Agent",
			personaName: "default/cto",
			projectID:   "proj-1",
			role:        "",
			wantStatus:  "paused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := setupWorkerManager(t)
			ctx := context.Background()
			persona := &models.Persona{
				Name:        tt.personaName,
				Description: "Test persona",
			}

			agent, err := m.CreateAgent(ctx, tt.agentName, tt.personaName, tt.projectID, tt.role, persona)

			if err != nil {
				t.Fatalf("CreateAgent() error = %v", err)
			}

			if agent == nil {
				t.Fatal("CreateAgent() returned nil agent")
			}

			// Verify agent fields
			if agent.ID == "" {
				t.Error("agent.ID is empty")
			}
			if agent.PersonaName != tt.personaName {
				t.Errorf("agent.PersonaName = %v, want %v", agent.PersonaName, tt.personaName)
			}
			if agent.ProjectID != tt.projectID {
				t.Errorf("agent.ProjectID = %v, want %v", agent.ProjectID, tt.projectID)
			}
			if agent.Status != tt.wantStatus {
				t.Errorf("agent.Status = %v, want %v", agent.Status, tt.wantStatus)
			}
			if agent.ProviderID != "" {
				t.Errorf("agent.ProviderID = %v, want empty", agent.ProviderID)
			}

			// Verify role was set or derived
			if tt.role != "" {
				if agent.Role != tt.role {
					t.Errorf("agent.Role = %v, want %v", agent.Role, tt.role)
				}
			} else {
				if agent.Role == "" {
					t.Error("agent.Role not derived from persona name")
				}
			}

			// Verify agent is in manager's map
			if _, exists := m.agents[agent.ID]; !exists {
				t.Error("agent not found in manager's agents map")
			}
		})
	}
}

func TestWorkerManager_SpawnAgentWorker(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()

	// Register a test provider
	_ = m.providerRegistry.Register(&provider.ProviderConfig{
		ID:       "test",
		Name:     "Test Provider",
		Type:     "custom",
		Endpoint: "http://localhost:8888/v1",
		APIKey:   "test-key",
		Model:    "test-model",
	})

	persona := &models.Persona{
		Name:        "default/qa-engineer",
		Description: "Test persona",
	}

	agent, err := m.SpawnAgentWorker(ctx, "Test Agent", "default/qa-engineer", "proj-1", "test", persona)

	if err != nil {
		t.Fatalf("SpawnAgentWorker() error = %v", err)
	}

	if agent == nil {
		t.Fatal("SpawnAgentWorker() returned nil agent")
	}

	// Verify agent has provider assigned
	if agent.ProviderID != "test" {
		t.Errorf("agent.ProviderID = %v, want test", agent.ProviderID)
	}

	// Verify agent is idle (not paused)
	if agent.Status != "idle" {
		t.Errorf("agent.Status = %v, want idle", agent.Status)
	}

	// Verify worker was spawned
	_, err = m.workerPool.GetWorker(agent.ID)
	if err != nil {
		t.Errorf("Worker not found in pool: %v", err)
	}
}

func TestWorkerManager_GetAgent(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	agent, _ := m.CreateAgent(ctx, "test-agent", "test-persona", "proj-1", "Test", persona)

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"existing agent", agent.ID, false},
		{"non-existent agent", "invalid-id", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.GetAgent(tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAgent() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && got.ID != tt.id {
				t.Errorf("GetAgent() ID = %v, want %v", got.ID, tt.id)
			}
		})
	}
}

func TestWorkerManager_ListAgents(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	// Initially empty
	if len(m.ListAgents()) != 0 {
		t.Error("ListAgents() should be empty initially")
	}

	// Create agents
	agent1, _ := m.CreateAgent(ctx, "agent-1", "persona-1", "proj-1", "Role1", persona)
	agent2, _ := m.CreateAgent(ctx, "agent-2", "persona-2", "proj-2", "Role2", persona)

	agents := m.ListAgents()
	if len(agents) != 2 {
		t.Errorf("ListAgents() = %d, want 2", len(agents))
	}

	// Verify agents are in list
	found := make(map[string]bool)
	for _, a := range agents {
		found[a.ID] = true
	}
	if !found[agent1.ID] || !found[agent2.ID] {
		t.Error("ListAgents() missing expected agents")
	}
}

func TestWorkerManager_ListAgentsByProject(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	agent1, _ := m.CreateAgent(ctx, "agent-1", "persona-1", "proj-1", "Role1", persona)
	agent2, _ := m.CreateAgent(ctx, "agent-2", "persona-2", "proj-1", "Role2", persona)
	agent3, _ := m.CreateAgent(ctx, "agent-3", "persona-3", "proj-2", "Role3", persona)

	tests := []struct {
		name      string
		projectID string
		wantCount int
		wantIDs   []string
	}{
		{"project with 2 agents", "proj-1", 2, []string{agent1.ID, agent2.ID}},
		{"project with 1 agent", "proj-2", 1, []string{agent3.ID}},
		{"non-existent project", "proj-3", 0, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := m.ListAgentsByProject(tt.projectID)

			if len(agents) != tt.wantCount {
				t.Errorf("ListAgentsByProject() = %d, want %d", len(agents), tt.wantCount)
			}

			found := make(map[string]bool)
			for _, a := range agents {
				found[a.ID] = true
			}
			for _, id := range tt.wantIDs {
				if !found[id] {
					t.Errorf("ListAgentsByProject() missing agent %s", id)
				}
			}
		})
	}
}

func TestWorkerManager_UpdateAgentStatus(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	agent, _ := m.CreateAgent(ctx, "test-agent", "test-persona", "proj-1", "Test", persona)

	tests := []struct {
		name      string
		agentID   string
		newStatus string
		wantErr   bool
	}{
		{"update to working", agent.ID, "working", false},
		{"update to idle", agent.ID, "idle", false},
		{"update to paused", agent.ID, "paused", false},
		{"non-existent agent", "invalid-id", "working", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.UpdateAgentStatus(tt.agentID, tt.newStatus)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateAgentStatus() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				updatedAgent, _ := m.GetAgent(tt.agentID)
				if updatedAgent.Status != tt.newStatus {
					t.Errorf("agent.Status = %v, want %v", updatedAgent.Status, tt.newStatus)
				}
			}
		})
	}
}

func TestWorkerManager_GetIdleAgentsByProject(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	// Create agents with different statuses
	agent1, _ := m.CreateAgent(ctx, "agent-1", "persona-1", "proj-1", "Role1", persona) // paused
	agent2, _ := m.CreateAgent(ctx, "agent-2", "persona-2", "proj-1", "Role2", persona) // paused
	_, _ = m.CreateAgent(ctx, "agent-3", "persona-3", "proj-2", "Role3", persona)       // paused

	// Update some to idle and working
	m.UpdateAgentStatus(agent1.ID, "idle")
	m.UpdateAgentStatus(agent2.ID, "working")
	// agent3 stays paused

	tests := []struct {
		name      string
		projectID string
		wantCount int
	}{
		{"proj-1 idle/paused agents", "proj-1", 1}, // agent1 is idle, agent2 is working (excluded)
		{"proj-2 idle/paused agents", "proj-2", 1}, // agent3 is paused (included)
		{"all projects", "", 2},                    // agent1 (idle) + agent3 (paused)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := m.GetIdleAgentsByProject(tt.projectID)

			if len(agents) != tt.wantCount {
				t.Errorf("GetIdleAgentsByProject() = %d, want %d", len(agents), tt.wantCount)
			}
		})
	}
}

func TestWorkerManager_AssignBead(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	agent, _ := m.CreateAgent(ctx, "test-agent", "test-persona", "proj-1", "Test", persona)

	err := m.AssignBead(agent.ID, "bead-123")
	if err != nil {
		t.Fatalf("AssignBead() error = %v", err)
	}

	updatedAgent, _ := m.GetAgent(agent.ID)
	if updatedAgent.CurrentBead != "bead-123" {
		t.Errorf("agent.CurrentBead = %v, want bead-123", updatedAgent.CurrentBead)
	}
	if updatedAgent.Status != "working" {
		t.Errorf("agent.Status = %v, want working", updatedAgent.Status)
	}
}

func TestWorkerManager_UpdateHeartbeat(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	agent, _ := m.CreateAgent(ctx, "test-agent", "test-persona", "proj-1", "Test", persona)

	initialAgent, _ := m.GetAgent(agent.ID)
	initialLastActive := initialAgent.LastActive

	time.Sleep(10 * time.Millisecond)

	err := m.UpdateHeartbeat(agent.ID)
	if err != nil {
		t.Fatalf("UpdateHeartbeat() error = %v", err)
	}

	updatedAgent, _ := m.GetAgent(agent.ID)
	if !updatedAgent.LastActive.After(initialLastActive) {
		t.Error("LastActive was not updated")
	}
}

func TestWorkerManager_StopAgent(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()

	// Register test provider
	_ = m.providerRegistry.Register(&provider.ProviderConfig{
		ID:       "test",
		Name:     "Test Provider",
		Type:     "custom",
		Endpoint: "http://localhost:8888/v1",
		APIKey:   "test-key",
		Model:    "test-model",
	})

	persona := &models.Persona{Name: "test-persona"}
	agent, _ := m.SpawnAgentWorker(ctx, "test-agent", "test-persona", "proj-1", "test", persona)

	err := m.StopAgent(agent.ID)
	if err != nil {
		t.Fatalf("StopAgent() error = %v", err)
	}

	// Verify agent was removed
	_, err = m.GetAgent(agent.ID)
	if err == nil {
		t.Error("Agent still exists after StopAgent()")
	}

	// Verify worker was stopped
	_, err = m.workerPool.GetWorker(agent.ID)
	if err == nil {
		t.Error("Worker still exists after StopAgent()")
	}
}

func TestWorkerManager_StopAll(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	// Create multiple agents
	m.CreateAgent(ctx, "agent-1", "persona-1", "proj-1", "Role1", persona)
	m.CreateAgent(ctx, "agent-2", "persona-2", "proj-1", "Role2", persona)

	m.StopAll()

	agents := m.ListAgents()
	if len(agents) != 0 {
		t.Errorf("ListAgents() after StopAll() = %d, want 0", len(agents))
	}
}

func TestWorkerManager_UpdateAgentProject(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	agent, _ := m.CreateAgent(ctx, "test-agent", "test-persona", "proj-1", "Test", persona)

	err := m.UpdateAgentProject(agent.ID, "proj-2")
	if err != nil {
		t.Fatalf("UpdateAgentProject() error = %v", err)
	}

	updatedAgent, _ := m.GetAgent(agent.ID)
	if updatedAgent.ProjectID != "proj-2" {
		t.Errorf("agent.ProjectID = %v, want proj-2", updatedAgent.ProjectID)
	}
}

func TestWorkerManager_ResetStuckAgents(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	// Create agents with different statuses
	agent1, _ := m.CreateAgent(ctx, "agent-1", "persona-1", "proj-1", "Role1", persona)
	agent2, _ := m.CreateAgent(ctx, "agent-2", "persona-2", "proj-1", "Role2", persona)

	// Set one to working
	m.UpdateAgentStatus(agent1.ID, "working")
	m.AssignBead(agent1.ID, "bead-123")

	// Manually set LastActive to old time
	m.mu.Lock()
	m.agents[agent1.ID].LastActive = time.Now().Add(-2 * time.Hour)
	m.mu.Unlock()

	// Reset stuck agents (max duration 1 hour)
	count := m.ResetStuckAgents(1 * time.Hour)

	if count != 1 {
		t.Errorf("ResetStuckAgents() = %d, want 1", count)
	}

	// Verify agent1 was reset
	updatedAgent1, _ := m.GetAgent(agent1.ID)
	if updatedAgent1.Status != "idle" {
		t.Errorf("agent1.Status = %v, want idle", updatedAgent1.Status)
	}
	if updatedAgent1.CurrentBead != "" {
		t.Errorf("agent1.CurrentBead = %v, want empty", updatedAgent1.CurrentBead)
	}

	// Verify agent2 was not affected
	updatedAgent2, _ := m.GetAgent(agent2.ID)
	if updatedAgent2.Status != "paused" {
		t.Errorf("agent2.Status = %v, want paused", updatedAgent2.Status)
	}
}

func TestWorkerManager_RestoreAgentWorker(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()

	// Register test provider
	_ = m.providerRegistry.Register(&provider.ProviderConfig{
		ID:       "test",
		Name:     "Test Provider",
		Type:     "custom",
		Endpoint: "http://localhost:8888/v1",
		APIKey:   "test-key",
		Model:    "test-model",
	})

	persona := &models.Persona{
		Name:        "default/qa-engineer",
		Description: "Test persona",
	}

	// Test restoring new agent
	agent := &models.Agent{
		ID:          "agent-restore-1",
		Name:        "Restored Agent",
		PersonaName: "default/qa-engineer",
		Persona:     persona,
		ProviderID:  "test",
		Status:      "idle",
		ProjectID:   "proj-1",
		StartedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	restored, err := m.RestoreAgentWorker(ctx, agent)
	if err != nil {
		t.Fatalf("RestoreAgentWorker() error = %v", err)
	}

	if restored == nil {
		t.Fatal("RestoreAgentWorker() returned nil")
	}

	// Verify agent was added
	foundAgent, err := m.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Agent not found after restore: %v", err)
	}

	if foundAgent.ID != agent.ID {
		t.Errorf("foundAgent.ID = %v, want %v", foundAgent.ID, agent.ID)
	}

	// Test restoring existing agent (should update)
	agent.Status = "working"
	restored, err = m.RestoreAgentWorker(ctx, agent)
	if err != nil {
		t.Fatalf("RestoreAgentWorker() on existing agent error = %v", err)
	}

	updatedAgent, _ := m.GetAgent(agent.ID)
	if updatedAgent.Status != "working" {
		t.Errorf("updatedAgent.Status = %v, want working", updatedAgent.Status)
	}
}

func Test_deriveRoleFromPersonaName(t *testing.T) {
	tests := []struct {
		personaName string
		want        string
	}{
		{"default/qa-engineer", "qa-engineer"},
		{"default/cto", "cto"},
		{"projects/myproj/web-designer", "web-designer"},
		{"custom/architect", "architect"},
		{"simple", "simple"},
		{"", ""},
		{"  default/test  ", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.personaName, func(t *testing.T) {
			got := deriveRoleFromPersonaName(tt.personaName)
			if got != tt.want {
				t.Errorf("deriveRoleFromPersonaName(%q) = %v, want %v", tt.personaName, got, tt.want)
			}
		})
	}
}

func Test_deriveDisplayName(t *testing.T) {
	tests := []struct {
		personaName string
		wantContain []string
	}{
		{"default/qa-engineer", []string{"Qa", "Engineer", "(Default)"}},
		{"default/web-designer", []string{"Web", "Designer", "(Default)"}},
		{"projects/myproj/cto", []string{"Cto", "(myproj)"}},
		{"custom/architect", []string{"Architect", "(Custom)"}},
	}

	for _, tt := range tests {
		t.Run(tt.personaName, func(t *testing.T) {
			got := deriveDisplayName(tt.personaName)
			for _, substr := range tt.wantContain {
				if !strings.Contains(got, substr) {
					t.Errorf("deriveDisplayName(%q) = %v, should contain %v", tt.personaName, got, substr)
				}
			}
		})
	}
}

func TestWorkerManager_ExecuteTask(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()

	// Register test provider
	_ = m.providerRegistry.Register(&provider.ProviderConfig{
		ID:       "test",
		Name:     "Test Provider",
		Type:     "custom",
		Endpoint: "http://localhost:8888/v1",
		APIKey:   "test-key",
		Model:    "test-model",
	})

	persona := &models.Persona{Name: "test-persona", Description: "Test"}

	// Spawn agent with worker
	agent, err := m.SpawnAgentWorker(ctx, "test-agent", "test-persona", "proj-1", "test", persona)
	if err != nil {
		t.Fatalf("Failed to spawn agent: %v", err)
	}

	// Create a task
	task := &worker.Task{
		ID:          "task-1",
		Description: "Test task",
		ProjectID:   "proj-1",
		BeadID:      "bead-1",
	}

	// Execute task (will fail because no actual LLM, but should not panic)
	result, err := m.ExecuteTask(ctx, agent.ID, task)

	// We expect an error since there's no actual LLM endpoint
	if err == nil {
		t.Log("ExecuteTask() succeeded unexpectedly (test provider may have succeeded)")
	}

	// Should return a result object even on error
	if result == nil {
		t.Log("ExecuteTask() returned nil result (acceptable for failed task)")
	}

	// Verify agent status was updated back to idle after execution
	updatedAgent, _ := m.GetAgent(agent.ID)
	if updatedAgent.Status != "idle" {
		t.Errorf("agent.Status after task = %v, want idle", updatedAgent.Status)
	}
}

func TestWorkerManager_GetPoolStats(t *testing.T) {
	m := setupWorkerManager(t)

	stats := m.GetPoolStats()

	// Just verify we can get stats without panicking
	if stats.TotalWorkers < 0 {
		t.Error("TotalWorkers should not be negative")
	}
	if stats.MaxWorkers < 0 {
		t.Error("MaxWorkers should not be negative")
	}
}

func TestWorkerManager_SettersAndGetters(t *testing.T) {
	m := setupWorkerManager(t)

	// Test SetActionLoopEnabled
	m.SetActionLoopEnabled(true)
	if !m.actionLoopEnabled {
		t.Error("SetActionLoopEnabled(true) failed")
	}

	// Test SetMaxLoopIterations
	m.SetMaxLoopIterations(20)
	if m.maxLoopIterations != 20 {
		t.Errorf("SetMaxLoopIterations(20) = %d", m.maxLoopIterations)
	}

	// Test GetWorkerPool
	pool := m.GetWorkerPool()
	if pool == nil {
		t.Error("GetWorkerPool() returned nil")
	}

	// Test SetAgentPersister
	m.SetAgentPersister(nil)
	if m.agentPersister != nil {
		t.Error("SetAgentPersister(nil) should set nil")
	}

	// Test SetActionRouter
	m.SetActionRouter(nil)
	if m.actionRouter != nil {
		t.Error("SetActionRouter(nil) should set nil")
	}

	// Test SetAnalyticsLogger
	m.SetAnalyticsLogger(nil)
	if m.analyticsLogger != nil {
		t.Error("SetAnalyticsLogger(nil) should set nil")
	}

	// Test SetLessonsProvider
	m.SetLessonsProvider(nil)
	if m.lessonsProvider != nil {
		t.Error("SetLessonsProvider(nil) should set nil")
	}

	// Test SetDatabase
	m.SetDatabase(nil)
	if m.db != nil {
		t.Error("SetDatabase(nil) should set nil")
	}
}

func TestWorkerManager_GetIdleAgents(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	// No agents - should return empty
	idle := m.GetIdleAgents()
	if len(idle) != 0 {
		t.Errorf("GetIdleAgents() = %d, want 0", len(idle))
	}

	// Create an agent and set it to idle
	agent, err := m.CreateAgent(ctx, "idle-agent", "coder", "proj1", "coder", persona)
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}
	agent.Status = "idle"

	idle = m.GetIdleAgents()
	if len(idle) != 1 {
		t.Errorf("GetIdleAgents() = %d, want 1", len(idle))
	}

	// Create a working agent - should not appear in idle
	agent2, err := m.CreateAgent(ctx, "working-agent", "reviewer", "proj1", "reviewer", persona)
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}
	agent2.Status = "working"

	idle = m.GetIdleAgents()
	if len(idle) != 1 {
		t.Errorf("GetIdleAgents() after working agent = %d, want 1", len(idle))
	}
}

func TestWorkerManager_PersistAgent(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	// persistAgent with nil agent should not panic
	m.persistAgent(nil)

	// persistAgent without persister should not panic
	agent, _ := m.CreateAgent(ctx, "test-agent", "coder", "proj1", "coder", persona)
	m.persistAgent(agent)
}

func TestWorkerManager_ExecuteTask_ErrorCases(t *testing.T) {
	m := setupWorkerManager(t)
	ctx := context.Background()

	// Non-existent agent
	task := &worker.Task{
		ID:          "task-1",
		Description: "do something",
	}
	_, err := m.ExecuteTask(ctx, "nonexistent", task)
	if err == nil {
		t.Error("ExecuteTask with nonexistent agent should fail")
	}
}
