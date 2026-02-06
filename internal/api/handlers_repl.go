package api

import (
	"context"
	"net/http"
	"time"
)

// handleRepl handles POST /api/v1/repl for CEO REPL queries.
func (s *Server) handleRepl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Message    string `json:"message"`
		TimeoutSec int    `json:"timeout_sec"`
	}
	if err := s.parseJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Message == "" {
		s.respondError(w, http.StatusBadRequest, "message is required")
		return
	}

	timeout := 3 * time.Minute
	if req.TimeoutSec > 0 {
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := s.app.RunReplQuery(ctx, req.Message)
	if err != nil {
		s.respondError(w, http.StatusBadGateway, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, result)
}
