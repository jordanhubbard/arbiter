package api

import "net/http"

// handleSystemStatus handles GET /api/v1/system/status
func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	status := s.app.GetDispatcher().GetSystemStatus()
	s.respondJSON(w, http.StatusOK, status)
}
