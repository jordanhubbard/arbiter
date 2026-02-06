package api

import (
	"encoding/json"
	"net/http"

	"github.com/jordanhubbard/loom/internal/routing"
)

// handleSelectProvider handles provider selection with routing policy
func (s *Server) handleSelectProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Policy       string                        `json:"policy"`       // minimize_cost, minimize_latency, maximize_quality, balanced
		Requirements *routing.ProviderRequirements `json:"requirements"` // Optional requirements
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default policy
	if req.Policy == "" {
		req.Policy = "balanced"
	}

	provider, err := s.app.SelectProvider(r.Context(), req.Requirements, req.Policy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(provider); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetRoutingPolicies handles listing available routing policies
func (s *Server) handleGetRoutingPolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	policies := []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}{
		{
			ID:          "minimize_cost",
			Name:        "Minimize Cost",
			Description: "Select the cheapest provider that meets requirements",
		},
		{
			ID:          "minimize_latency",
			Name:        "Minimize Latency",
			Description: "Select the fastest provider with lowest response time",
		},
		{
			ID:          "maximize_quality",
			Name:        "Maximize Quality",
			Description: "Select the highest quality provider with best capabilities",
		},
		{
			ID:          "balanced",
			Name:        "Balanced",
			Description: "Balance cost, latency, and quality (default)",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policies); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
