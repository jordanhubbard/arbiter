package project

import (
	"fmt"
	"sync"
	"time"

	"github.com/jordanhubbard/arbiter/pkg/models"
)

// Manager manages projects
type Manager struct {
	projects map[string]*models.Project
	mu       sync.RWMutex
}

// NewManager creates a new project manager
func NewManager() *Manager {
	return &Manager{
		projects: make(map[string]*models.Project),
	}
}

// CreateProject creates a new project
func (m *Manager) CreateProject(name, gitRepo, branch, beadsPath string, context map[string]string) (*models.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate project ID
	projectID := fmt.Sprintf("proj-%d", time.Now().Unix())

	if beadsPath == "" {
		beadsPath = ".beads"
	}

	project := &models.Project{
		ID:        projectID,
		Name:      name,
		GitRepo:   gitRepo,
		Branch:    branch,
		BeadsPath: beadsPath,
		Context:   context,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Agents:    []string{},
	}

	m.projects[projectID] = project

	return project, nil
}

// GetProject retrieves a project by ID
func (m *Manager) GetProject(id string) (*models.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	project, ok := m.projects[id]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", id)
	}

	return project, nil
}

// ListProjects returns all projects
func (m *Manager) ListProjects() []*models.Project {
	m.mu.RLock()
	defer m.mu.RUnlock()

	projects := make([]*models.Project, 0, len(m.projects))
	for _, project := range m.projects {
		projects = append(projects, project)
	}

	return projects
}

// UpdateProject updates a project
func (m *Manager) UpdateProject(id string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	project, ok := m.projects[id]
	if !ok {
		return fmt.Errorf("project not found: %s", id)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		project.Name = name
	}
	if branch, ok := updates["branch"].(string); ok {
		project.Branch = branch
	}
	if context, ok := updates["context"].(map[string]string); ok {
		project.Context = context
	}

	project.UpdatedAt = time.Now()

	return nil
}

// AddAgentToProject adds an agent to a project
func (m *Manager) AddAgentToProject(projectID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	project, ok := m.projects[projectID]
	if !ok {
		return fmt.Errorf("project not found: %s", projectID)
	}

	// Check if agent already in project
	for _, id := range project.Agents {
		if id == agentID {
			return nil // Already added
		}
	}

	project.Agents = append(project.Agents, agentID)
	project.UpdatedAt = time.Now()

	return nil
}

// RemoveAgentFromProject removes an agent from a project
func (m *Manager) RemoveAgentFromProject(projectID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	project, ok := m.projects[projectID]
	if !ok {
		return fmt.Errorf("project not found: %s", projectID)
	}

	// Find and remove agent
	for i, id := range project.Agents {
		if id == agentID {
			project.Agents = append(project.Agents[:i], project.Agents[i+1:]...)
			project.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("agent not found in project: %s", agentID)
}

// DeleteProject deletes a project
func (m *Manager) DeleteProject(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.projects[id]; !ok {
		return fmt.Errorf("project not found: %s", id)
	}

	delete(m.projects, id)

	return nil
}

// LoadProjects loads projects from configuration
func (m *Manager) LoadProjects(projects []models.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, project := range projects {
		// Create a copy
		p := project
		p.CreatedAt = time.Now()
		p.UpdatedAt = time.Now()
		if p.Agents == nil {
			p.Agents = []string{}
		}
		m.projects[p.ID] = &p
	}

	return nil
}
