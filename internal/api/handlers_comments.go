package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// handleBeadComments handles comment operations for a specific bead
// GET /api/v1/beads/{id}/comments - Get all comments
// POST /api/v1/beads/{id}/comments - Create comment
func (s *Server) handleBeadComments(w http.ResponseWriter, r *http.Request) {
	commentsMgr := s.app.GetCommentsManager()
	if commentsMgr == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Comments manager not available")
		return
	}

	// Extract bead ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/beads/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "comments" {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	beadID := parts[0]

	switch r.Method {
	case http.MethodGet:
		s.handleGetComments(w, r, beadID, commentsMgr)
	case http.MethodPost:
		s.handleCreateComment(w, r, beadID, commentsMgr)
	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetComments retrieves all comments for a bead
func (s *Server) handleGetComments(w http.ResponseWriter, r *http.Request, beadID string, commentsMgr interface{}) {
	// Type assert to access GetComments method
	type CommentsGetter interface {
		GetComments(beadID string) (interface{}, error)
	}

	getter, ok := commentsMgr.(CommentsGetter)
	if !ok {
		s.respondError(w, http.StatusInternalServerError, "Invalid comments manager")
		return
	}

	comments, err := getter.GetComments(beadID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get comments: %v", err))
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"bead_id":  beadID,
		"comments": comments,
	})
}

// handleCreateComment creates a new comment
func (s *Server) handleCreateComment(w http.ResponseWriter, r *http.Request, beadID string, commentsMgr interface{}) {
	// Get user from context
	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req struct {
		Content  string `json:"content"`
		ParentID string `json:"parent_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.Content == "" {
		s.respondError(w, http.StatusBadRequest, "Content is required")
		return
	}

	// Type assert to access CreateComment method
	type CommentCreator interface {
		CreateComment(beadID, authorID, authorUsername, content, parentID string) (interface{}, error)
	}

	creator, ok := commentsMgr.(CommentCreator)
	if !ok {
		s.respondError(w, http.StatusInternalServerError, "Invalid comments manager")
		return
	}

	comment, err := creator.CreateComment(beadID, user.ID, user.Username, req.Content, req.ParentID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create comment: %v", err))
		return
	}

	s.respondJSON(w, http.StatusCreated, comment)
}

// handleComment handles operations on a specific comment
// PATCH /api/v1/comments/{id} - Update comment
// DELETE /api/v1/comments/{id} - Delete comment
func (s *Server) handleComment(w http.ResponseWriter, r *http.Request) {
	commentsMgr := s.app.GetCommentsManager()
	if commentsMgr == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Comments manager not available")
		return
	}

	// Get user from context
	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract comment ID from path
	commentID := strings.TrimPrefix(r.URL.Path, "/api/v1/comments/")
	if commentID == "" {
		s.respondError(w, http.StatusBadRequest, "Comment ID is required")
		return
	}

	switch r.Method {
	case http.MethodPatch:
		s.handleUpdateComment(w, r, commentID, user.ID, commentsMgr)
	case http.MethodDelete:
		s.handleDeleteComment(w, r, commentID, user.ID, commentsMgr)
	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleUpdateComment updates a comment
func (s *Server) handleUpdateComment(w http.ResponseWriter, r *http.Request, commentID, userID string, commentsMgr interface{}) {
	// Parse request body
	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.Content == "" {
		s.respondError(w, http.StatusBadRequest, "Content is required")
		return
	}

	// Type assert to access UpdateComment method
	type CommentUpdater interface {
		UpdateComment(commentID, authorID, content string) error
	}

	updater, ok := commentsMgr.(CommentUpdater)
	if !ok {
		s.respondError(w, http.StatusInternalServerError, "Invalid comments manager")
		return
	}

	if err := updater.UpdateComment(commentID, userID, req.Content); err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			s.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			s.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update comment: %v", err))
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Comment updated successfully",
	})
}

// handleDeleteComment deletes a comment
func (s *Server) handleDeleteComment(w http.ResponseWriter, r *http.Request, commentID, userID string, commentsMgr interface{}) {
	// Type assert to access DeleteComment method
	type CommentDeleter interface {
		DeleteComment(commentID, authorID string) error
	}

	deleter, ok := commentsMgr.(CommentDeleter)
	if !ok {
		s.respondError(w, http.StatusInternalServerError, "Invalid comments manager")
		return
	}

	if err := deleter.DeleteComment(commentID, userID); err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			s.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			s.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete comment: %v", err))
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Comment deleted successfully",
	})
}
