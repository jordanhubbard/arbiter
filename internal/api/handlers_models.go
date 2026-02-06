package api

import "net/http"

// handleRecommendedModels handles GET /api/v1/models/recommended
func (s *Server) handleRecommendedModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	models := s.app.ListModelCatalog()
	s.respondJSON(w, http.StatusOK, map[string]interface{}{"models": models})
}
