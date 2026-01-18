package beads

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/jordanhubbard/arbiter/pkg/models"
)

// Manager integrates with the bd (beads) CLI tool
type Manager struct {
	bdPath   string
	mu       sync.RWMutex
	beads    map[string]*models.Bead
	workGraph *models.WorkGraph
}

// NewManager creates a new beads manager
func NewManager(bdPath string) *Manager {
	return &Manager{
		bdPath:   bdPath,
		beads:    make(map[string]*models.Bead),
		workGraph: &models.WorkGraph{
			Beads: make(map[string]*models.Bead),
			Edges: []models.Edge{},
			UpdatedAt: time.Now(),
		},
	}
}

// CreateBead creates a new bead using bd CLI
func (m *Manager) CreateBead(title, description string, priority models.BeadPriority, beadType, projectID string) (*models.Bead, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Build bd command
	args := []string{"create", title, "-p", fmt.Sprintf("%d", priority)}
	
	if description != "" {
		args = append(args, "-d", description)
	}

	// Execute bd command
	cmd := exec.Command(m.bdPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create bead: %w, output: %s", err, string(output))
	}

	// Parse output to get bead ID
	// bd typically outputs: Created bd-xxxxx
	outputStr := string(output)
	beadID := m.extractBeadID(outputStr)
	if beadID == "" {
		return nil, fmt.Errorf("failed to extract bead ID from output: %s", outputStr)
	}

	// Create internal bead representation
	bead := &models.Bead{
		ID:          beadID,
		Type:        beadType,
		Title:       title,
		Description: description,
		Status:      models.BeadStatusOpen,
		Priority:    priority,
		ProjectID:   projectID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.beads[beadID] = bead
	m.workGraph.Beads[beadID] = bead
	m.workGraph.UpdatedAt = time.Now()

	return bead, nil
}

// GetBead retrieves a bead by ID
func (m *Manager) GetBead(id string) (*models.Bead, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bead, ok := m.beads[id]
	if !ok {
		// Try to fetch from bd
		return m.fetchBeadFromBD(id)
	}

	return bead, nil
}

// ListBeads returns all beads, optionally filtered
func (m *Manager) ListBeads(filters map[string]interface{}) ([]*models.Bead, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// In a full implementation, this would call bd with appropriate filters
	// For now, return from cache with basic filtering

	beads := make([]*models.Bead, 0, len(m.beads))
	
	for _, bead := range m.beads {
		if m.matchesFilters(bead, filters) {
			beads = append(beads, bead)
		}
	}

	return beads, nil
}

// UpdateBead updates a bead
func (m *Manager) UpdateBead(id string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bead, ok := m.beads[id]
	if !ok {
		return fmt.Errorf("bead not found: %s", id)
	}

	// Apply updates
	if status, ok := updates["status"].(models.BeadStatus); ok {
		bead.Status = status
	}
	if assignedTo, ok := updates["assigned_to"].(string); ok {
		bead.AssignedTo = assignedTo
	}
	if description, ok := updates["description"].(string); ok {
		bead.Description = description
	}

	bead.UpdatedAt = time.Now()
	m.workGraph.UpdatedAt = time.Now()

	// TODO: Sync with bd CLI

	return nil
}

// ClaimBead assigns a bead to an agent
func (m *Manager) ClaimBead(beadID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bead, ok := m.beads[beadID]
	if !ok {
		return fmt.Errorf("bead not found: %s", beadID)
	}

	if bead.AssignedTo != "" && bead.AssignedTo != agentID {
		return fmt.Errorf("bead already claimed by agent %s", bead.AssignedTo)
	}

	bead.AssignedTo = agentID
	bead.Status = models.BeadStatusInProgress
	bead.UpdatedAt = time.Now()

	return nil
}

// AddDependency adds a dependency between beads
func (m *Manager) AddDependency(childID, parentID, relationship string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	child, ok := m.beads[childID]
	if !ok {
		return fmt.Errorf("child bead not found: %s", childID)
	}

	parent, ok := m.beads[parentID]
	if !ok {
		return fmt.Errorf("parent bead not found: %s", parentID)
	}

	// Update bead relationships
	switch relationship {
	case "blocks":
		child.BlockedBy = append(child.BlockedBy, parentID)
		parent.Blocks = append(parent.Blocks, childID)
		if child.Status == models.BeadStatusInProgress {
			child.Status = models.BeadStatusBlocked
		}
	case "parent":
		child.Parent = parentID
		parent.Children = append(parent.Children, childID)
	case "related":
		child.RelatedTo = append(child.RelatedTo, parentID)
		parent.RelatedTo = append(parent.RelatedTo, childID)
	default:
		return fmt.Errorf("unknown relationship: %s", relationship)
	}

	// Update work graph
	m.workGraph.Edges = append(m.workGraph.Edges, models.Edge{
		From:         childID,
		To:           parentID,
		Relationship: relationship,
	})
	m.workGraph.UpdatedAt = time.Now()

	return nil
}

// GetReadyBeads returns beads with no open blockers
func (m *Manager) GetReadyBeads(projectID string) ([]*models.Bead, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ready := make([]*models.Bead, 0)

	for _, bead := range m.beads {
		if projectID != "" && bead.ProjectID != projectID {
			continue
		}

		if bead.Status != models.BeadStatusOpen {
			continue
		}

		// Check if all blockers are resolved
		allResolved := true
		for _, blockerID := range bead.BlockedBy {
			blocker, ok := m.beads[blockerID]
			if !ok || blocker.Status != models.BeadStatusClosed {
				allResolved = false
				break
			}
		}

		if allResolved {
			ready = append(ready, bead)
		}
	}

	return ready, nil
}

// UnblockBead removes a blocking dependency
func (m *Manager) UnblockBead(beadID, blockerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bead, ok := m.beads[beadID]
	if !ok {
		return fmt.Errorf("bead not found: %s", beadID)
	}

	// Remove blocker
	for i, id := range bead.BlockedBy {
		if id == blockerID {
			bead.BlockedBy = append(bead.BlockedBy[:i], bead.BlockedBy[i+1:]...)
			break
		}
	}

	// If no more blockers, unblock
	if len(bead.BlockedBy) == 0 && bead.Status == models.BeadStatusBlocked {
		bead.Status = models.BeadStatusOpen
	}

	bead.UpdatedAt = time.Now()
	m.workGraph.UpdatedAt = time.Now()

	return nil
}

// GetWorkGraph returns the current work graph
func (m *Manager) GetWorkGraph(projectID string) (*models.WorkGraph, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if projectID == "" {
		return m.workGraph, nil
	}

	// Filter by project
	filteredGraph := &models.WorkGraph{
		Beads:     make(map[string]*models.Bead),
		Edges:     []models.Edge{},
		UpdatedAt: m.workGraph.UpdatedAt,
	}

	for id, bead := range m.workGraph.Beads {
		if bead.ProjectID == projectID {
			filteredGraph.Beads[id] = bead
		}
	}

	for _, edge := range m.workGraph.Edges {
		if _, ok := filteredGraph.Beads[edge.From]; ok {
			if _, ok := filteredGraph.Beads[edge.To]; ok {
				filteredGraph.Edges = append(filteredGraph.Edges, edge)
			}
		}
	}

	return filteredGraph, nil
}

// Helper functions

func (m *Manager) extractBeadID(output string) string {
	// Look for pattern like "bd-xxxxx" or "Created bd-xxxxx"
	parts := strings.Fields(output)
	for _, part := range parts {
		if strings.HasPrefix(part, "bd-") {
			return part
		}
	}
	return ""
}

func (m *Manager) fetchBeadFromBD(id string) (*models.Bead, error) {
	// Execute: bd show <id> --json
	cmd := exec.Command(m.bdPath, "show", id, "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bead: %w", err)
	}

	var bead models.Bead
	if err := json.Unmarshal(output, &bead); err != nil {
		return nil, fmt.Errorf("failed to parse bead: %w", err)
	}

	return &bead, nil
}

func (m *Manager) matchesFilters(bead *models.Bead, filters map[string]interface{}) bool {
	if projectID, ok := filters["project_id"].(string); ok {
		if bead.ProjectID != projectID {
			return false
		}
	}
	
	if status, ok := filters["status"].(models.BeadStatus); ok {
		if bead.Status != status {
			return false
		}
	}
	
	if beadType, ok := filters["type"].(string); ok {
		if bead.Type != beadType {
			return false
		}
	}
	
	return true
}
