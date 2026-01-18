package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jordanhubbard/arbiter/pkg/models"
)

// Manager manages agent lifecycle and coordination
type Manager struct {
	agents    map[string]*models.Agent
	mu        sync.RWMutex
	maxAgents int
}

// NewManager creates a new agent manager
func NewManager(maxAgents int) *Manager {
	return &Manager{
		agents:    make(map[string]*models.Agent),
		maxAgents: maxAgents,
	}
}

// SpawnAgent creates and starts a new agent
func (m *Manager) SpawnAgent(ctx context.Context, name, personaName, projectID string, persona *models.Persona) (*models.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we've reached max agents
	if len(m.agents) >= m.maxAgents {
		return nil, fmt.Errorf("maximum number of agents (%d) reached", m.maxAgents)
	}

	// Generate agent ID
	agentID := fmt.Sprintf("agent-%d-%s", time.Now().Unix(), name)

	// Use persona name as agent name if custom name not provided
	if name == "" {
		name = personaName
	}

	agent := &models.Agent{
		ID:          agentID,
		Name:        name,
		PersonaName: personaName,
		Persona:     persona,
		Status:      "idle",
		ProjectID:   projectID,
		StartedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	m.agents[agentID] = agent

	// TODO: Actually spawn the agent process/goroutine
	// This would integrate with an LLM service to run the agent

	return agent, nil
}

// GetAgent retrieves an agent by ID
func (m *Manager) GetAgent(id string) (*models.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, ok := m.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	return agent, nil
}

// ListAgents returns all agents
func (m *Manager) ListAgents() []*models.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*models.Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}

	return agents
}

// ListAgentsByProject returns agents for a specific project
func (m *Manager) ListAgentsByProject(projectID string) []*models.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*models.Agent, 0)
	for _, agent := range m.agents {
		if agent.ProjectID == projectID {
			agents = append(agents, agent)
		}
	}

	return agents
}

// UpdateAgentStatus updates an agent's status
func (m *Manager) UpdateAgentStatus(id, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	agent.Status = status
	agent.LastActive = time.Now()

	return nil
}

// AssignBead assigns a bead to an agent
func (m *Manager) AssignBead(agentID, beadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[agentID]
	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	agent.CurrentBead = beadID
	agent.Status = "working"
	agent.LastActive = time.Now()

	return nil
}

// StopAgent stops and removes an agent
func (m *Manager) StopAgent(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	// TODO: Actually stop the agent process/goroutine

	delete(m.agents, id)

	// Return the agent for logging purposes
	_ = agent

	return nil
}

// UpdateHeartbeat updates an agent's last active time
func (m *Manager) UpdateHeartbeat(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	agent.LastActive = time.Now()

	return nil
}

// GetIdleAgents returns agents that are idle
func (m *Manager) GetIdleAgents() []*models.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*models.Agent, 0)
	for _, agent := range m.agents {
		if agent.Status == "idle" {
			agents = append(agents, agent)
		}
	}

	return agents
}
