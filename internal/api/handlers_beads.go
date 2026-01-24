package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

// handleBeads handles GET/POST /api/v1/beads
func (s *Server) handleBeads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Parse query parameters
		projectID := r.URL.Query().Get("project_id")
		statusStr := r.URL.Query().Get("status")
		beadType := r.URL.Query().Get("type")
		assignedTo := r.URL.Query().Get("assigned_to")

		filters := make(map[string]interface{})
		if projectID != "" {
			filters["project_id"] = projectID
		}
		if statusStr != "" {
			filters["status"] = models.BeadStatus(statusStr)
		}
		if beadType != "" {
			filters["type"] = beadType
		}
		if assignedTo != "" {
			if strings.Contains(assignedTo, ",") {
				parts := strings.Split(assignedTo, ",")
				values := make([]string, 0, len(parts))
				for _, part := range parts {
					value := strings.TrimSpace(part)
					if value != "" {
						values = append(values, value)
					}
				}
				if len(values) > 0 {
					filters["assigned_to"] = values
				}
			} else {
				filters["assigned_to"] = assignedTo
			}
		}

		beads, err := s.agenticorp.GetBeadsManager().ListBeads(filters)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.respondJSON(w, http.StatusOK, beads)

	case http.MethodPost:
		var req struct {
			Type        string            `json:"type"`
			Title       string            `json:"title"`
			Description string            `json:"description"`
			Priority    int               `json:"priority"`
			ProjectID   string            `json:"project_id"`
			Parent      string            `json:"parent"`
			Tags        []string          `json:"tags"`
			Context     map[string]string `json:"context"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Title == "" || req.ProjectID == "" {
			s.respondError(w, http.StatusBadRequest, "title and project_id are required")
			return
		}

		if req.Type == "" {
			req.Type = "task"
		}
		if req.Priority == 0 {
			req.Priority = 2
		}

		bead, err := s.agenticorp.CreateBead(req.Title, req.Description, models.BeadPriority(req.Priority), req.Type, req.ProjectID)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.respondJSON(w, http.StatusCreated, bead)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleBead handles GET/PATCH /api/v1/beads/{id} and POST /api/v1/beads/{id}/claim
func (s *Server) handleBead(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/beads/")
	parts := strings.Split(path, "/")
	id := parts[0]

	// Handle /claim endpoint
	if len(parts) > 1 && parts[1] == "claim" {
		if r.Method != http.MethodPost {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req struct {
			AgentID string `json:"agent_id"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.AgentID == "" {
			s.respondError(w, http.StatusBadRequest, "agent_id is required")
			return
		}

		if err := s.agenticorp.ClaimBead(id, req.AgentID); err != nil {
			if strings.Contains(err.Error(), "already claimed") {
				s.respondError(w, http.StatusConflict, err.Error())
			} else {
				s.respondError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		s.respondJSON(w, http.StatusOK, map[string]string{"status": "claimed"})
		return
	}

	// Handle /redispatch endpoint
	if len(parts) > 1 && parts[1] == "redispatch" {
		if r.Method != http.MethodPost {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req struct {
			Reason string `json:"reason"`
		}
		_ = s.parseJSON(r, &req)

		updates := map[string]interface{}{
			"status": models.BeadStatusOpen,
			"context": map[string]string{
				"redispatch_requested":    "true",
				"redispatch_requested_at": time.Now().UTC().Format(time.RFC3339),
				"redispatch_reason":       req.Reason,
			},
		}

		bead, err := s.agenticorp.UpdateBead(id, updates)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, bead)
		return
	}

	// Handle /escalate endpoint (human-in-the-loop)
	if len(parts) > 1 && parts[1] == "escalate" {
		if r.Method != http.MethodPost {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req struct {
			Reason     string `json:"reason"`
			ReturnedTo string `json:"returned_to"`
		}
		_ = s.parseJSON(r, &req)

		decision, err := s.agenticorp.EscalateBeadToCEO(id, req.Reason, req.ReturnedTo)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, decision)
		return
	}

	// Handle regular bead operations
	switch r.Method {
	case http.MethodGet:
		bead, err := s.agenticorp.GetBeadsManager().GetBead(id)
		if err != nil {
			s.respondError(w, http.StatusNotFound, "Bead not found")
			return
		}
		s.respondJSON(w, http.StatusOK, bead)

	case http.MethodPatch:
		var req struct {
			Title       *string            `json:"title"`
			Type        *string            `json:"type"`
			Status      *string            `json:"status"`
			Priority    *int               `json:"priority"`
			ProjectID   *string            `json:"project_id"`
			AssignedTo  *string            `json:"assigned_to"`
			Description *string            `json:"description"`
			Parent      *string            `json:"parent"`
			Tags        *[]string          `json:"tags"`
			BlockedBy   *[]string          `json:"blocked_by"`
			Blocks      *[]string          `json:"blocks"`
			RelatedTo   *[]string          `json:"related_to"`
			Children    *[]string          `json:"children"`
			Context     map[string]string `json:"context"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		updates := make(map[string]interface{})
		if req.Title != nil {
			updates["title"] = *req.Title
		}
		if req.Type != nil {
			updates["type"] = *req.Type
		}
		if req.Status != nil {
			updates["status"] = models.BeadStatus(*req.Status)
		}
		if req.Priority != nil {
			updates["priority"] = models.BeadPriority(*req.Priority)
		}
		if req.ProjectID != nil {
			updates["project_id"] = *req.ProjectID
		}
		if req.AssignedTo != nil {
			updates["assigned_to"] = *req.AssignedTo
		}
		if req.Description != nil {
			updates["description"] = *req.Description
		}
		if req.Parent != nil {
			updates["parent"] = *req.Parent
		}
		if req.Tags != nil {
			updates["tags"] = *req.Tags
		}
		if req.BlockedBy != nil {
			updates["blocked_by"] = *req.BlockedBy
		}
		if req.Blocks != nil {
			updates["blocks"] = *req.Blocks
		}
		if req.RelatedTo != nil {
			updates["related_to"] = *req.RelatedTo
		}
		if req.Children != nil {
			updates["children"] = *req.Children
		}
		if req.Context != nil {
			updates["context"] = req.Context
		}

		bead, err := s.agenticorp.UpdateBead(id, updates)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, bead)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleDecisions handles GET /api/v1/decisions
func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse query parameters
	statusStr := r.URL.Query().Get("status")
	priorityStr := r.URL.Query().Get("priority")

	filters := make(map[string]interface{})
	if statusStr != "" {
		filters["status"] = models.BeadStatus(statusStr)
	}
	if priorityStr != "" {
		if priority, err := strconv.Atoi(priorityStr); err == nil {
			filters["priority"] = models.BeadPriority(priority)
		}
	}

	decisions, err := s.agenticorp.GetDecisionManager().ListDecisions(filters)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, decisions)
}

// handleDecision handles GET /api/v1/decisions/{id} and POST /api/v1/decisions/{id}/decide
func (s *Server) handleDecision(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/decisions/")
	parts := strings.Split(path, "/")
	id := parts[0]

	// Handle /decide endpoint
	if len(parts) > 1 && parts[1] == "decide" {
		if r.Method != http.MethodPost {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req struct {
			DeciderID string `json:"decider_id"`
			Decision  string `json:"decision"`
			Rationale string `json:"rationale"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Decision == "" || req.Rationale == "" {
			s.respondError(w, http.StatusBadRequest, "decision and rationale are required")
			return
		}

		if err := s.agenticorp.MakeDecision(id, req.DeciderID, req.Decision, req.Rationale); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.respondJSON(w, http.StatusOK, map[string]string{"status": "decided"})
		return
	}

	// Handle regular decision operations
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	decision, err := s.agenticorp.GetDecisionManager().GetDecision(id)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Decision not found")
		return
	}
	s.respondJSON(w, http.StatusOK, decision)
}

// handleFileLocks handles GET/POST /api/v1/file-locks
func (s *Server) handleFileLocks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projectID := r.URL.Query().Get("project_id")

		var locks []*models.FileLock
		if projectID != "" {
			locks = s.agenticorp.GetFileLockManager().ListLocksByProject(projectID)
		} else {
			locks = s.agenticorp.GetFileLockManager().ListLocks()
		}

		s.respondJSON(w, http.StatusOK, locks)

	case http.MethodPost:
		var req struct {
			FilePath  string `json:"file_path"`
			ProjectID string `json:"project_id"`
			AgentID   string `json:"agent_id"`
			BeadID    string `json:"bead_id"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.FilePath == "" || req.ProjectID == "" || req.AgentID == "" {
			s.respondError(w, http.StatusBadRequest, "file_path, project_id, and agent_id are required")
			return
		}

		lock, err := s.agenticorp.RequestFileAccess(req.ProjectID, req.FilePath, req.AgentID, req.BeadID)
		if err != nil {
			if strings.Contains(err.Error(), "already locked") {
				s.respondError(w, http.StatusConflict, err.Error())
			} else {
				s.respondError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		s.respondJSON(w, http.StatusCreated, lock)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleFileLock handles DELETE /api/v1/file-locks/{project_id}/{path}
func (s *Server) handleFileLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/file-locks/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) < 2 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	projectID := parts[0]
	filePath := parts[1]

	// Get agent ID from request (could be from body or query)
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		s.respondError(w, http.StatusBadRequest, "agent_id is required")
		return
	}

	if err := s.agenticorp.ReleaseFileAccess(projectID, filePath, agentID); err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleWorkGraph handles GET /api/v1/work-graph
func (s *Server) handleWorkGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	projectID := r.URL.Query().Get("project_id")

	graph, err := s.agenticorp.GetWorkGraph(projectID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, graph)
}
