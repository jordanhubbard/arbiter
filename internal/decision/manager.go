package decision

import (
	"fmt"
	"sync"
	"time"

	"github.com/jordanhubbard/arbiter/pkg/models"
)

// Manager manages decision beads and decision-making
type Manager struct {
	decisions map[string]*models.DecisionBead
	mu        sync.RWMutex
}

// NewManager creates a new decision manager
func NewManager() *Manager {
	return &Manager{
		decisions: make(map[string]*models.DecisionBead),
	}
}

// CreateDecision creates a new decision bead
func (m *Manager) CreateDecision(question, parentBeadID, requesterID string, options []string, recommendation string, priority models.BeadPriority, projectID string) (*models.DecisionBead, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate decision ID
	decisionID := fmt.Sprintf("bd-dec-%d", time.Now().Unix())

	// Create base bead
	bead := &models.Bead{
		ID:          decisionID,
		Type:        "decision",
		Title:       fmt.Sprintf("Decision: %s", question),
		Description: question,
		Status:      models.BeadStatusOpen,
		Priority:    priority,
		ProjectID:   projectID,
		Parent:      parentBeadID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	decision := &models.DecisionBead{
		Bead:           bead,
		Question:       question,
		Options:        options,
		Recommendation: recommendation,
		RequesterID:    requesterID,
	}

	m.decisions[decisionID] = decision

	return decision, nil
}

// GetDecision retrieves a decision by ID
func (m *Manager) GetDecision(id string) (*models.DecisionBead, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	decision, ok := m.decisions[id]
	if !ok {
		return nil, fmt.Errorf("decision not found: %s", id)
	}

	return decision, nil
}

// ListDecisions returns all decisions, optionally filtered
func (m *Manager) ListDecisions(filters map[string]interface{}) ([]*models.DecisionBead, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	decisions := make([]*models.DecisionBead, 0, len(m.decisions))

	for _, decision := range m.decisions {
		if m.matchesFilters(decision, filters) {
			decisions = append(decisions, decision)
		}
	}

	return decisions, nil
}

// ClaimDecision assigns a decision to a decider
func (m *Manager) ClaimDecision(decisionID, deciderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	decision, ok := m.decisions[decisionID]
	if !ok {
		return fmt.Errorf("decision not found: %s", decisionID)
	}

	if decision.DeciderID != "" && decision.DeciderID != deciderID {
		return fmt.Errorf("decision already claimed by %s", decision.DeciderID)
	}

	decision.DeciderID = deciderID
	decision.Status = models.BeadStatusInProgress
	decision.UpdatedAt = time.Now()

	return nil
}

// MakeDecision resolves a decision
func (m *Manager) MakeDecision(decisionID, deciderID, decisionText, rationale string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	decision, ok := m.decisions[decisionID]
	if !ok {
		return fmt.Errorf("decision not found: %s", decisionID)
	}

	// Verify decider
	if decision.DeciderID != "" && decision.DeciderID != deciderID {
		return fmt.Errorf("decision claimed by different agent: %s", decision.DeciderID)
	}

	now := time.Now()
	decision.DeciderID = deciderID
	decision.Decision = decisionText
	decision.Rationale = rationale
	decision.DecidedAt = &now
	decision.Status = models.BeadStatusClosed
	decision.ClosedAt = &now
	decision.UpdatedAt = now

	return nil
}

// EscalateDecision escalates a decision to P0
func (m *Manager) EscalateDecision(decisionID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	decision, ok := m.decisions[decisionID]
	if !ok {
		return fmt.Errorf("decision not found: %s", decisionID)
	}

	decision.Priority = models.BeadPriorityP0
	decision.Description = fmt.Sprintf("%s\n\nESCALATED: %s", decision.Description, reason)
	decision.UpdatedAt = time.Now()

	return nil
}

// GetPendingDecisions returns decisions that need to be resolved
func (m *Manager) GetPendingDecisions(priority *models.BeadPriority) ([]*models.DecisionBead, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	decisions := make([]*models.DecisionBead, 0)

	for _, decision := range m.decisions {
		if decision.Status == models.BeadStatusOpen || decision.Status == models.BeadStatusInProgress {
			if priority == nil || decision.Priority == *priority {
				decisions = append(decisions, decision)
			}
		}
	}

	return decisions, nil
}

// GetP0Decisions returns all P0 decisions requiring human intervention
func (m *Manager) GetP0Decisions() ([]*models.DecisionBead, error) {
	p0 := models.BeadPriorityP0
	return m.GetPendingDecisions(&p0)
}

// GetDecisionsByProject returns decisions for a specific project
func (m *Manager) GetDecisionsByProject(projectID string) ([]*models.DecisionBead, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	decisions := make([]*models.DecisionBead, 0)

	for _, decision := range m.decisions {
		if decision.ProjectID == projectID {
			decisions = append(decisions, decision)
		}
	}

	return decisions, nil
}

// GetDecisionsByRequester returns decisions filed by a specific agent
func (m *Manager) GetDecisionsByRequester(requesterID string) ([]*models.DecisionBead, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	decisions := make([]*models.DecisionBead, 0)

	for _, decision := range m.decisions {
		if decision.RequesterID == requesterID {
			decisions = append(decisions, decision)
		}
	}

	return decisions, nil
}

// GetBlockedBeads returns beads blocked by a decision
func (m *Manager) GetBlockedBeads(decisionID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	decision, ok := m.decisions[decisionID]
	if !ok {
		return []string{}
	}

	return decision.Blocks
}

// CanAutoDecide determines if a decision can be made autonomously
func (m *Manager) CanAutoDecide(decisionID string, deciderAutonomy models.AutonomyLevel) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	decision, ok := m.decisions[decisionID]
	if !ok {
		return false, "decision not found"
	}

	// P0 decisions always require human intervention
	if decision.Priority == models.BeadPriorityP0 {
		return false, "P0 decisions require human intervention"
	}

	// Supervised agents cannot make any decisions
	if deciderAutonomy == models.AutonomySupervised {
		return false, "supervised agents cannot make decisions"
	}

	// Semi-autonomous can make routine decisions (P2, P3)
	if deciderAutonomy == models.AutonomySemi {
		if decision.Priority <= models.BeadPriorityP1 {
			return false, "semi-autonomous agents cannot make P0/P1 decisions"
		}
	}

	// Full autonomy can decide all non-P0
	return true, ""
}

// matchesFilters checks if a decision matches the given filters
func (m *Manager) matchesFilters(decision *models.DecisionBead, filters map[string]interface{}) bool {
	if status, ok := filters["status"].(models.BeadStatus); ok {
		if decision.Status != status {
			return false
		}
	}

	if priority, ok := filters["priority"].(models.BeadPriority); ok {
		if decision.Priority != priority {
			return false
		}
	}

	if projectID, ok := filters["project_id"].(string); ok {
		if decision.ProjectID != projectID {
			return false
		}
	}

	if requesterID, ok := filters["requester_id"].(string); ok {
		if decision.RequesterID != requesterID {
			return false
		}
	}

	return true
}

// UpdateDecisionContext adds context to a decision
func (m *Manager) UpdateDecisionContext(decisionID string, context map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	decision, ok := m.decisions[decisionID]
	if !ok {
		return fmt.Errorf("decision not found: %s", decisionID)
	}

	if decision.Context == nil {
		decision.Context = make(map[string]string)
	}

	for key, value := range context {
		decision.Context[key] = value
	}

	decision.UpdatedAt = time.Now()

	return nil
}
