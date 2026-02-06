package api

import (
	"context"
	"github.com/jordanhubbard/loom/pkg/models"
	"net/http"
	"strings"
)

// handlePersonas handles GET /api/v1/personas
func (s *Server) handlePersonas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	personas, err := s.app.GetPersonaManager().ListPersonas()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Load full persona details
	fullPersonas := make([]*models.Persona, 0, len(personas))
	for _, name := range personas {
		persona, err := s.app.GetPersonaManager().LoadPersona(name)
		if err != nil {
			continue
		}
		fullPersonas = append(fullPersonas, persona)
	}

	s.respondJSON(w, http.StatusOK, fullPersonas)
}

// handlePersona handles GET/PUT /api/v1/personas/{name}
func (s *Server) handlePersona(w http.ResponseWriter, r *http.Request) {
	name := s.extractID(r.URL.Path, "/api/v1/personas")

	switch r.Method {
	case http.MethodGet:
		persona, err := s.app.GetPersonaManager().LoadPersona(name)
		if err != nil {
			s.respondError(w, http.StatusNotFound, "Persona not found")
			return
		}
		s.respondJSON(w, http.StatusOK, persona)

	case http.MethodPut:
		var persona models.Persona
		if err := s.parseJSON(r, &persona); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Load existing persona
		existing, err := s.app.GetPersonaManager().LoadPersona(name)
		if err != nil {
			s.respondError(w, http.StatusNotFound, "Persona not found")
			return
		}

		// Update fields
		persona.Name = existing.Name
		persona.PersonaFile = existing.PersonaFile
		persona.InstructionsFile = existing.InstructionsFile
		persona.CreatedAt = existing.CreatedAt

		// Save
		if err := s.app.GetPersonaManager().SavePersona(&persona); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Invalidate cache
		s.app.GetPersonaManager().InvalidateCache(name)

		s.respondJSON(w, http.StatusOK, &persona)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleAgents handles GET/POST /api/v1/agents
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		agents := s.app.GetAgentManager().ListAgents()
		s.respondJSON(w, http.StatusOK, agents)

	case http.MethodPost:
		var req struct {
			Name        string `json:"name"`
			PersonaName string `json:"persona_name"`
			ProjectID   string `json:"project_id"`
			ProviderID  string `json:"provider_id"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.PersonaName == "" || req.ProjectID == "" {
			s.respondError(w, http.StatusBadRequest, "persona_name and project_id are required")
			return
		}

		// Normalize persona name: prepend "default/" if not a namespaced path
		personaName := req.PersonaName
		if !strings.Contains(personaName, "/") {
			personaName = "default/" + personaName
		}

		agent, err := s.app.SpawnAgent(context.Background(), req.Name, personaName, req.ProjectID, req.ProviderID)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.respondJSON(w, http.StatusCreated, agent)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleAgent handles GET/DELETE /api/v1/agents/{id}
func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	parts := strings.Split(path, "/")
	id := parts[0]

	if len(parts) > 1 {
		action := parts[1]
		s.handleAgentAction(w, r, id, action)
		return
	}

	switch r.Method {
	case http.MethodGet:
		agent, err := s.app.GetAgentManager().GetAgent(id)
		if err != nil {
			s.respondError(w, http.StatusNotFound, "Agent not found")
			return
		}
		s.respondJSON(w, http.StatusOK, agent)

	case http.MethodPut:
		var req struct {
			Name    string          `json:"name"`
			Persona *models.Persona `json:"persona"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		agent, err := s.app.GetAgentManager().GetAgent(id)
		if err != nil {
			s.respondError(w, http.StatusNotFound, "Agent not found")
			return
		}

		// Update agent fields
		if req.Name != "" {
			agent.Name = req.Name
		}
		if req.Persona != nil {
			if agent.Persona == nil {
				agent.Persona = req.Persona
			} else {
				if req.Persona.Mission != "" {
					agent.Persona.Mission = req.Persona.Mission
				}
				if req.Persona.Character != "" {
					agent.Persona.Character = req.Persona.Character
				}
				if req.Persona.Tone != "" {
					agent.Persona.Tone = req.Persona.Tone
				}
				if req.Persona.AutonomyLevel != "" {
					agent.Persona.AutonomyLevel = req.Persona.AutonomyLevel
				}
			}
		}

		// Write-through cache: Persist to database
		// Note: agent is already updated in-memory since GetAgent returns a pointer to the cached object
		if s.app.GetDatabase() != nil {
			if err := s.app.GetDatabase().UpsertAgent(agent); err != nil {
				s.respondError(w, http.StatusInternalServerError, "Failed to update agent in database")
				return
			}
		}

		s.respondJSON(w, http.StatusOK, agent)

	case http.MethodDelete:
		if err := s.app.StopAgent(context.Background(), id); err != nil {
			s.respondError(w, http.StatusNotFound, "Agent not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleAgentAction(w http.ResponseWriter, r *http.Request, id, action string) {
	switch action {
	case "clone":
		s.handleCloneAgent(w, r, id)
	default:
		s.respondError(w, http.StatusNotFound, "Unknown action")
	}
}

func (s *Server) handleCloneAgent(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		NewPersonaName string `json:"new_persona_name"`
		NewAgentName   string `json:"new_agent_name"`
		SourcePersona  string `json:"source_persona"`
		Replace        *bool  `json:"replace"`
	}
	if err := s.parseJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.NewPersonaName == "" {
		s.respondError(w, http.StatusBadRequest, "new_persona_name is required")
		return
	}

	replace := true
	if req.Replace != nil {
		replace = *req.Replace
	}

	agent, err := s.app.CloneAgentPersona(context.Background(), id, req.NewPersonaName, req.NewAgentName, req.SourcePersona, replace)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, agent)
}

// handleProjects handles GET/POST /api/v1/projects
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projects := s.app.GetProjectManager().ListProjects()
		s.respondJSON(w, http.StatusOK, projects)

	case http.MethodPost:
		var req struct {
			Name      string            `json:"name"`
			GitRepo   string            `json:"git_repo"`
			Branch    string            `json:"branch"`
			BeadsPath string            `json:"beads_path"`
			Context   map[string]string `json:"context"`
			IsSticky  *bool             `json:"is_sticky"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Name == "" || req.GitRepo == "" || req.Branch == "" {
			s.respondError(w, http.StatusBadRequest, "name, git_repo, and branch are required")
			return
		}

		project, err := s.app.CreateProject(req.Name, req.GitRepo, req.Branch, req.BeadsPath, req.Context)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if req.IsSticky != nil {
			updates := map[string]interface{}{"is_sticky": *req.IsSticky}
			if err := s.app.GetProjectManager().UpdateProject(project.ID, updates); err == nil {
				s.app.PersistProject(project.ID)
				project, _ = s.app.GetProjectManager().GetProject(project.ID)
			}
		}

		s.respondJSON(w, http.StatusCreated, project)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleProject handles GET /api/v1/projects/{id} and state management endpoints
func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/projects/")
	parts := strings.Split(path, "/")
	id := parts[0]

	// Handle sub-endpoints for project state management
	if len(parts) > 1 {
		action := parts[1]
		if action == "files" {
			s.handleProjectFiles(w, r, id, parts[2:])
			return
		}
		s.handleProjectStateEndpoints(w, r, id, action)
		return
	}

	// Default GET behavior
	switch r.Method {
	case http.MethodGet:
		project, err := s.app.GetProjectManager().GetProject(id)
		if err != nil {
			s.respondError(w, http.StatusNotFound, "Project not found")
			return
		}
		s.respondJSON(w, http.StatusOK, project)

	case http.MethodPut:
		var req struct {
			Name        string            `json:"name"`
			GitRepo     string            `json:"git_repo"`
			Branch      string            `json:"branch"`
			BeadsPath   string            `json:"beads_path"`
			Context     map[string]string `json:"context"`
			Status      string            `json:"status"`
			IsPerpetual *bool             `json:"is_perpetual"`
			IsSticky    *bool             `json:"is_sticky"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		updates := map[string]interface{}{}
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.GitRepo != "" {
			updates["git_repo"] = req.GitRepo
		}
		if req.Branch != "" {
			updates["branch"] = req.Branch
		}
		if req.BeadsPath != "" {
			updates["beads_path"] = req.BeadsPath
		}
		if req.Context != nil {
			updates["context"] = req.Context
		}
		if req.Status != "" {
			updates["status"] = req.Status
		}
		if req.IsPerpetual != nil {
			updates["is_perpetual"] = *req.IsPerpetual
		}
		if req.IsSticky != nil {
			updates["is_sticky"] = *req.IsSticky
		}

		if err := s.app.GetProjectManager().UpdateProject(id, updates); err != nil {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.app.PersistProject(id)
		project, _ := s.app.GetProjectManager().GetProject(id)
		s.respondJSON(w, http.StatusOK, project)

	case http.MethodDelete:
		if err := s.app.DeleteProject(id); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleOrgChart handles GET /api/v1/org-charts/{projectId}
func (s *Server) handleOrgChart(w http.ResponseWriter, r *http.Request) {
	projectID := s.extractID(r.URL.Path, "/api/v1/org-charts")

	switch r.Method {
	case http.MethodGet:
		chart, err := s.app.GetOrgChartManager().GetByProject(projectID)
		if err != nil {
			// If no org chart exists, create one from the project
			project, projErr := s.app.GetProjectManager().GetProject(projectID)
			if projErr != nil {
				s.respondError(w, http.StatusNotFound, "Project not found")
				return
			}
			chart, err = s.app.GetOrgChartManager().CreateForProject(projectID, project.Name)
			if err != nil {
				s.respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		s.respondJSON(w, http.StatusOK, chart)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
