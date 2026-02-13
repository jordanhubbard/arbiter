package agent

import (
	"context"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name      string
		maxAgents int
	}{
		{"with limit", 10},
		{"with zero limit", 0},
		{"with negative limit", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(tt.maxAgents)
			if m == nil {
				t.Fatal("NewManager() returned nil")
			}
			if m.maxAgents != tt.maxAgents {
				t.Errorf("maxAgents = %d, want %d", m.maxAgents, tt.maxAgents)
			}
			if m.agents == nil {
				t.Error("agents map not initialized")
			}
		})
	}
}

func TestManager_SpawnAgent(t *testing.T) {
	tests := []struct {
		name        string
		maxAgents   int
		agentName   string
		personaName string
		projectID   string
		persona     *models.Persona
		wantErr     bool
	}{
		{
			name:        "successful spawn",
			maxAgents:   10,
			agentName:   "test-agent",
			personaName: "default/qa-engineer",
			projectID:   "proj-1",
			persona:     &models.Persona{Name: "default/qa-engineer", Description: "QA Engineer"},
			wantErr:     false,
		},
		{
			name:        "spawn with empty name uses persona name",
			maxAgents:   10,
			agentName:   "",
			personaName: "default/cto",
			projectID:   "proj-1",
			persona:     &models.Persona{Name: "default/cto", Description: "CTO"},
			wantErr:     false,
		},
		{
			name:        "max agents reached",
			maxAgents:   0,
			agentName:   "test-agent",
			personaName: "default/qa-engineer",
			projectID:   "proj-1",
			persona:     &models.Persona{Name: "default/qa-engineer"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(tt.maxAgents)
			ctx := context.Background()

			agent, err := m.SpawnAgent(ctx, tt.agentName, tt.personaName, tt.projectID, tt.persona)

			if (err != nil) != tt.wantErr {
				t.Errorf("SpawnAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if agent == nil {
				t.Fatal("SpawnAgent() returned nil agent")
			}

			// Check agent fields
			if agent.ID == "" {
				t.Error("agent.ID is empty")
			}
			if agent.PersonaName != tt.personaName {
				t.Errorf("agent.PersonaName = %v, want %v", agent.PersonaName, tt.personaName)
			}
			if agent.ProjectID != tt.projectID {
				t.Errorf("agent.ProjectID = %v, want %v", agent.ProjectID, tt.projectID)
			}
			if agent.Status != "idle" {
				t.Errorf("agent.Status = %v, want idle", agent.Status)
			}
			if agent.Persona != tt.persona {
				t.Error("agent.Persona doesn't match")
			}

			// If agentName was empty, name should be persona name
			expectedName := tt.agentName
			if expectedName == "" {
				expectedName = tt.personaName
			}
			if agent.Name != expectedName {
				t.Errorf("agent.Name = %v, want %v", agent.Name, expectedName)
			}

			// Verify agent is in the manager's map
			if _, exists := m.agents[agent.ID]; !exists {
				t.Error("agent not found in manager's agents map")
			}
		})
	}
}

func TestManager_GetAgent(t *testing.T) {
	m := NewManager(10)
	ctx := context.Background()

	// Spawn an agent
	persona := &models.Persona{Name: "test-persona"}
	agent, err := m.SpawnAgent(ctx, "test-agent", "test-persona", "proj-1", persona)
	if err != nil {
		t.Fatalf("Failed to spawn agent: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"existing agent", agent.ID, false},
		{"non-existent agent", "invalid-id", true},
		{"empty id", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.GetAgent(tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got == nil {
					t.Fatal("GetAgent() returned nil")
				}
				if got.ID != tt.id {
					t.Errorf("GetAgent() ID = %v, want %v", got.ID, tt.id)
				}
			}
		})
	}
}

func TestManager_ListAgents(t *testing.T) {
	m := NewManager(10)
	ctx := context.Background()

	// Initially empty
	agents := m.ListAgents()
	if len(agents) != 0 {
		t.Errorf("ListAgents() on empty manager = %d, want 0", len(agents))
	}

	// Spawn multiple agents
	persona := &models.Persona{Name: "test-persona"}
	agent1, _ := m.SpawnAgent(ctx, "agent-1", "persona-1", "proj-1", persona)
	agent2, _ := m.SpawnAgent(ctx, "agent-2", "persona-2", "proj-2", persona)

	agents = m.ListAgents()
	if len(agents) != 2 {
		t.Errorf("ListAgents() = %d agents, want 2", len(agents))
	}

	// Verify agents are in the list
	found := make(map[string]bool)
	for _, a := range agents {
		found[a.ID] = true
	}
	if !found[agent1.ID] || !found[agent2.ID] {
		t.Error("ListAgents() missing expected agents")
	}
}

func TestManager_ListAgentsByProject(t *testing.T) {
	m := NewManager(10)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	// Spawn agents in different projects
	agent1, _ := m.SpawnAgent(ctx, "agent-1", "persona-1", "proj-1", persona)
	agent2, _ := m.SpawnAgent(ctx, "agent-2", "persona-2", "proj-1", persona)
	agent3, _ := m.SpawnAgent(ctx, "agent-3", "persona-3", "proj-2", persona)

	tests := []struct {
		name      string
		projectID string
		wantCount int
		wantIDs   []string
	}{
		{"project with 2 agents", "proj-1", 2, []string{agent1.ID, agent2.ID}},
		{"project with 1 agent", "proj-2", 1, []string{agent3.ID}},
		{"project with no agents", "proj-3", 0, []string{}},
		{"empty project ID", "", 0, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := m.ListAgentsByProject(tt.projectID)

			if len(agents) != tt.wantCount {
				t.Errorf("ListAgentsByProject() = %d agents, want %d", len(agents), tt.wantCount)
			}

			// Verify expected agents
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

func TestManager_UpdateAgentStatus(t *testing.T) {
	m := NewManager(10)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	agent, err := m.SpawnAgent(ctx, "test-agent", "test-persona", "proj-1", persona)
	if err != nil {
		t.Fatalf("Failed to spawn agent: %v", err)
	}

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
				return
			}

			if !tt.wantErr {
				// Verify status was updated
				updatedAgent, _ := m.GetAgent(tt.agentID)
				if updatedAgent.Status != tt.newStatus {
					t.Errorf("agent.Status = %v, want %v", updatedAgent.Status, tt.newStatus)
				}

				// Verify LastActive was updated
				if time.Since(updatedAgent.LastActive) > time.Second {
					t.Error("LastActive was not updated")
				}
			}
		})
	}
}

func TestManager_AssignBead(t *testing.T) {
	m := NewManager(10)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	agent, err := m.SpawnAgent(ctx, "test-agent", "test-persona", "proj-1", persona)
	if err != nil {
		t.Fatalf("Failed to spawn agent: %v", err)
	}

	tests := []struct {
		name    string
		agentID string
		beadID  string
		wantErr bool
	}{
		{"assign bead", agent.ID, "bead-1", false},
		{"assign different bead", agent.ID, "bead-2", false},
		{"non-existent agent", "invalid-id", "bead-1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.AssignBead(tt.agentID, tt.beadID)

			if (err != nil) != tt.wantErr {
				t.Errorf("AssignBead() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify bead was assigned
				updatedAgent, _ := m.GetAgent(tt.agentID)
				if updatedAgent.CurrentBead != tt.beadID {
					t.Errorf("agent.CurrentBead = %v, want %v", updatedAgent.CurrentBead, tt.beadID)
				}
				if updatedAgent.Status != "working" {
					t.Errorf("agent.Status = %v, want working", updatedAgent.Status)
				}
			}
		})
	}
}

func TestManager_StopAgent(t *testing.T) {
	m := NewManager(10)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	agent, err := m.SpawnAgent(ctx, "test-agent", "test-persona", "proj-1", persona)
	if err != nil {
		t.Fatalf("Failed to spawn agent: %v", err)
	}

	tests := []struct {
		name    string
		agentID string
		wantErr bool
	}{
		{"stop existing agent", agent.ID, false},
		{"stop non-existent agent", "invalid-id", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.StopAgent(tt.agentID)

			if (err != nil) != tt.wantErr {
				t.Errorf("StopAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify agent was removed
				_, err := m.GetAgent(tt.agentID)
				if err == nil {
					t.Error("Agent still exists after StopAgent()")
				}

				// Verify agent count decreased
				agents := m.ListAgents()
				if len(agents) != 0 {
					t.Errorf("ListAgents() = %d, want 0", len(agents))
				}
			}
		})
	}
}

func TestManager_UpdateHeartbeat(t *testing.T) {
	m := NewManager(10)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	agent, err := m.SpawnAgent(ctx, "test-agent", "test-persona", "proj-1", persona)
	if err != nil {
		t.Fatalf("Failed to spawn agent: %v", err)
	}

	// Get initial LastActive time
	initialAgent, _ := m.GetAgent(agent.ID)
	initialLastActive := initialAgent.LastActive

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	tests := []struct {
		name    string
		agentID string
		wantErr bool
	}{
		{"update existing agent", agent.ID, false},
		{"update non-existent agent", "invalid-id", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.UpdateHeartbeat(tt.agentID)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateHeartbeat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify LastActive was updated
				updatedAgent, _ := m.GetAgent(tt.agentID)
				if !updatedAgent.LastActive.After(initialLastActive) {
					t.Error("LastActive was not updated")
				}
			}
		})
	}
}

func TestManager_GetIdleAgents(t *testing.T) {
	m := NewManager(10)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	// Spawn agents with different statuses
	agent1, _ := m.SpawnAgent(ctx, "agent-1", "persona-1", "proj-1", persona)
	agent2, _ := m.SpawnAgent(ctx, "agent-2", "persona-2", "proj-1", persona)
	agent3, _ := m.SpawnAgent(ctx, "agent-3", "persona-3", "proj-1", persona)

	// Set different statuses
	m.UpdateAgentStatus(agent1.ID, "idle")
	m.UpdateAgentStatus(agent2.ID, "working")
	m.UpdateAgentStatus(agent3.ID, "idle")

	idleAgents := m.GetIdleAgents()

	// Should have 2 idle agents
	if len(idleAgents) != 2 {
		t.Errorf("GetIdleAgents() = %d, want 2", len(idleAgents))
	}

	// Verify correct agents are returned
	found := make(map[string]bool)
	for _, a := range idleAgents {
		found[a.ID] = true
	}
	if !found[agent1.ID] || !found[agent3.ID] {
		t.Error("GetIdleAgents() returned wrong agents")
	}
	if found[agent2.ID] {
		t.Error("GetIdleAgents() returned working agent")
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	m := NewManager(100)
	ctx := context.Background()
	persona := &models.Persona{Name: "test-persona"}

	// Spawn initial agents concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			_, _ = m.SpawnAgent(ctx, "test", "persona", "proj-1", persona)
			done <- true
		}(i)
	}

	// Wait for all spawns to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify agents were created
	agents := m.ListAgents()
	if len(agents) == 0 {
		t.Error("No agents were created during concurrent spawn")
	}

	// Concurrent reads and writes
	for i := 0; i < 5; i++ {
		go func() {
			_ = m.ListAgents()
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		go func() {
			agents := m.ListAgents()
			if len(agents) > 0 {
				_ = m.UpdateAgentStatus(agents[0].ID, "working")
			}
			done <- true
		}()
	}

	// Wait for all operations
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test should complete without deadlocks or panics
}
