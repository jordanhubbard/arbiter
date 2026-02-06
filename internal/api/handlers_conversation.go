package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jordanhubbard/loom/internal/database"
)

// handleConversation handles operations on a specific conversation session
// GET /api/v1/conversations/{id} - Get full conversation
// DELETE /api/v1/conversations/{id} - Delete session
// POST /api/v1/conversations/{id}/reset - Reset conversation history
func (s *Server) handleConversation(w http.ResponseWriter, r *http.Request) {
	db := s.app.GetDatabase()
	if db == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Extract session ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/conversations/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		s.respondError(w, http.StatusBadRequest, "Session ID is required")
		return
	}

	sessionID := parts[0]

	// Check for reset endpoint
	if len(parts) == 2 && parts[1] == "reset" {
		if r.Method != http.MethodPost {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handleResetConversation(w, r, sessionID, db)
		return
	}

	// Handle main conversation operations
	switch r.Method {
	case http.MethodGet:
		s.handleGetConversation(w, r, sessionID, db)
	case http.MethodDelete:
		s.handleDeleteConversation(w, r, sessionID, db)
	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetConversation retrieves a full conversation session
func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request, sessionID string, db *database.Database) {
	session, err := db.GetConversationContext(sessionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.respondError(w, http.StatusNotFound, fmt.Sprintf("Conversation session not found: %s", sessionID))
			return
		}
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get conversation: %v", err))
		return
	}

	s.respondJSON(w, http.StatusOK, session)
}

// handleDeleteConversation deletes a conversation session
func (s *Server) handleDeleteConversation(w http.ResponseWriter, r *http.Request, sessionID string, db *database.Database) {
	err := db.DeleteConversationContext(sessionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.respondError(w, http.StatusNotFound, fmt.Sprintf("Conversation session not found: %s", sessionID))
			return
		}
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete conversation: %v", err))
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Conversation deleted successfully",
		"session_id": sessionID,
	})
}

// handleResetConversation clears conversation history but keeps the session
func (s *Server) handleResetConversation(w http.ResponseWriter, r *http.Request, sessionID string, db *database.Database) {
	// Parse request body for options
	var req struct {
		KeepSystemMessage bool `json:"keep_system_message"`
	}

	// Default to keeping system message
	req.KeepSystemMessage = true

	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Ignore decode errors - use defaults
			_ = err
		}
	}

	err := db.ResetConversationMessages(sessionID, req.KeepSystemMessage)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.respondError(w, http.StatusNotFound, fmt.Sprintf("Conversation session not found: %s", sessionID))
			return
		}
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reset conversation: %v", err))
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":              "Conversation reset successfully",
		"session_id":           sessionID,
		"keep_system_message":  req.KeepSystemMessage,
	})
}

// handleBeadConversation retrieves the conversation for a specific bead
// GET /api/v1/beads/{id}/conversation - Get conversation for bead
func (s *Server) handleBeadConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	db := s.app.GetDatabase()
	if db == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Extract bead ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/beads/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "conversation" {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	beadID := parts[0]

	session, err := db.GetConversationContextByBeadID(beadID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.respondError(w, http.StatusNotFound, fmt.Sprintf("No conversation found for bead: %s", beadID))
			return
		}
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get conversation: %v", err))
		return
	}

	s.respondJSON(w, http.StatusOK, session)
}

// handleConversationsList lists conversations with optional filters
// GET /api/v1/conversations?project_id=<id>&limit=<n>
func (s *Server) handleConversationsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	db := s.app.GetDatabase()
	if db == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Get query parameters
	projectID := r.URL.Query().Get("project_id")
	limitStr := r.URL.Query().Get("limit")

	limit := 50 // Default limit
	if limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid limit parameter")
			return
		}
		if limit < 1 || limit > 1000 {
			s.respondError(w, http.StatusBadRequest, "Limit must be between 1 and 1000")
			return
		}
	}

	// If no project_id specified, require it
	if projectID == "" {
		s.respondError(w, http.StatusBadRequest, "project_id parameter is required")
		return
	}

	conversations, err := db.ListConversationContextsByProject(projectID, limit)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list conversations: %v", err))
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"project_id":    projectID,
		"limit":         limit,
		"conversations": conversations,
	})
}
