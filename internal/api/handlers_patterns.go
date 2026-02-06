package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanhubbard/loom/internal/patterns"
)

// handlePatternAnalysis handles GET /api/v1/patterns/analysis
func (s *Server) handlePatternAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	config := patterns.DefaultAnalysisConfig()

	if timeWindowStr := r.URL.Query().Get("time_window"); timeWindowStr != "" {
		if hours, err := strconv.Atoi(timeWindowStr); err == nil {
			config.TimeWindow = time.Duration(hours) * time.Hour
		}
	}

	if minRequests := r.URL.Query().Get("min_requests"); minRequests != "" {
		if val, err := strconv.Atoi(minRequests); err == nil {
			config.MinRequests = val
		}
	}

	if minCost := r.URL.Query().Get("min_cost"); minCost != "" {
		if val, err := strconv.ParseFloat(minCost, 64); err == nil {
			config.MinCostUSD = val
		}
	}

	// Get pattern manager
	patternManager := s.app.GetPatternManager()
	if patternManager == nil {
		http.Error(w, "Pattern analysis not available", http.StatusServiceUnavailable)
		return
	}

	// Run analysis
	report, err := patternManager.AnalyzePatterns(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(report); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleExpensivePatterns handles GET /api/v1/patterns/expensive
func (s *Server) handleExpensivePatterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse limit parameter
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	// Get pattern manager
	patternManager := s.app.GetPatternManager()
	if patternManager == nil {
		http.Error(w, "Pattern analysis not available", http.StatusServiceUnavailable)
		return
	}

	// Get expensive patterns
	patterns, err := patternManager.GetExpensivePatterns(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"patterns": patterns,
		"count":    len(patterns),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleAnomalies handles GET /api/v1/patterns/anomalies
func (s *Server) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get pattern manager
	patternManager := s.app.GetPatternManager()
	if patternManager == nil {
		http.Error(w, "Pattern analysis not available", http.StatusServiceUnavailable)
		return
	}

	// Get anomalies
	anomalies, err := patternManager.GetAnomalies(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"anomalies": anomalies,
		"count":     len(anomalies),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleOptimizations handles GET /api/v1/optimizations
func (s *Server) handleOptimizations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get pattern manager
	patternManager := s.app.GetPatternManager()
	if patternManager == nil {
		http.Error(w, "Pattern analysis not available", http.StatusServiceUnavailable)
		return
	}

	// Get comprehensive report
	report, err := patternManager.AnalyzeAll(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter by type if specified
	optimizationType := r.URL.Query().Get("type")
	minSavings := 0.0
	if minSavingsStr := r.URL.Query().Get("min_savings"); minSavingsStr != "" {
		if val, err := strconv.ParseFloat(minSavingsStr, 64); err == nil {
			minSavings = val
		}
	}

	filteredOptimizations := report.Optimizations
	if optimizationType != "" || minSavings > 0 {
		filtered := make([]*patterns.Optimization, 0)
		for _, opt := range report.Optimizations {
			if optimizationType != "" && opt.Type != optimizationType {
				continue
			}
			if minSavings > 0 && opt.ProjectedSavingsUSD < minSavings {
				continue
			}
			filtered = append(filtered, opt)
		}
		filteredOptimizations = filtered
	}

	response := map[string]interface{}{
		"optimizations":          filteredOptimizations,
		"count":                  len(filteredOptimizations),
		"total_savings_usd":      report.TotalSavingsUSD,
		"monthly_savings_usd":    report.MonthlySavingsUSD,
		"cache_opportunities":    len(report.CacheOpportunities),
		"batching_opportunities": len(report.BatchingOpportunities),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleSubstitutions handles GET /api/v1/optimizations/substitutions
func (s *Server) handleSubstitutions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get pattern manager
	patternManager := s.app.GetPatternManager()
	if patternManager == nil {
		http.Error(w, "Pattern analysis not available", http.StatusServiceUnavailable)
		return
	}

	// Get all optimizations
	optimizations, err := patternManager.GetOptimizations(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter for substitution optimizations
	substitutions := make([]*patterns.Optimization, 0)
	for _, opt := range optimizations {
		if opt.Type == "provider-substitution" || opt.Type == "model-substitution" {
			substitutions = append(substitutions, opt)
		}
	}

	response := map[string]interface{}{
		"substitutions": substitutions,
		"count":         len(substitutions),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleOptimizationActions handles POST /api/v1/optimizations/{id}/apply
func (s *Server) handleOptimizationActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract optimization ID from path
	// This is a placeholder - actual implementation would parse the path
	// and apply the optimization

	response := map[string]interface{}{
		"status":  "pending",
		"message": "Optimization application is not yet implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handlePromptAnalysis handles GET /api/v1/prompts/analysis
func (s *Server) handlePromptAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get pattern manager
	patternManager := s.app.GetPatternManager()
	if patternManager == nil {
		http.Error(w, "Pattern analysis not available", http.StatusServiceUnavailable)
		return
	}

	// Run prompt analysis
	report, err := patternManager.AnalyzePrompts(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(report); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handlePromptOptimizations handles GET /api/v1/prompts/optimizations
func (s *Server) handlePromptOptimizations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse limit parameter
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	// Get pattern manager
	patternManager := s.app.GetPatternManager()
	if patternManager == nil {
		http.Error(w, "Pattern analysis not available", http.StatusServiceUnavailable)
		return
	}

	// Get prompt optimizations
	optimizations, err := patternManager.GetPromptOptimizations(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"optimizations": optimizations,
		"count":         len(optimizations),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
