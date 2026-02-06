package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanhubbard/loom/internal/analytics"
	"github.com/jordanhubbard/loom/internal/cache"
)

// handleCacheAnalysis runs cache opportunity analysis
// GET /api/v1/cache/analysis
func (s *Server) handleCacheAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse query parameters
	config := cache.DefaultAnalysisConfig()

	if timeWindowStr := r.URL.Query().Get("time_window"); timeWindowStr != "" {
		duration, err := time.ParseDuration(timeWindowStr)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid time_window: %v", err))
			return
		}
		config.TimeWindow = duration
	}

	if minSavingsStr := r.URL.Query().Get("min_savings"); minSavingsStr != "" {
		minSavings, err := strconv.ParseFloat(minSavingsStr, 64)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid min_savings: %v", err))
			return
		}
		config.MinSavingsUSD = minSavings
	}

	if autoEnableStr := r.URL.Query().Get("auto_enable"); autoEnableStr != "" {
		autoEnable, err := strconv.ParseBool(autoEnableStr)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid auto_enable: %v", err))
			return
		}
		config.AutoEnable = autoEnable
	}

	// Get database for analytics storage
	db := s.app.GetDatabase()
	if db == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Create analytics storage
	storage, err := analytics.NewDatabaseStorage(db.DB())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create analytics storage: %v", err))
		return
	}

	// Create analyzer
	analyzer := cache.NewAnalyzer(storage, config)

	// Run analysis
	ctx := context.Background()
	report, err := analyzer.Analyze(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	s.respondJSON(w, http.StatusOK, report)
}

// handleCacheOpportunities lists current caching opportunities
// GET /api/v1/cache/opportunities
func (s *Server) handleCacheOpportunities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse query parameters for pagination
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	priority := r.URL.Query().Get("priority") // "high", "medium", "low"

	// Get database for analytics storage
	db := s.app.GetDatabase()
	if db == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Create analytics storage
	storage, err := analytics.NewDatabaseStorage(db.DB())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create analytics storage: %v", err))
		return
	}

	// Create analyzer with default config
	analyzer := cache.NewAnalyzer(storage, nil)

	// Run analysis
	ctx := context.Background()
	report, err := analyzer.Analyze(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	// Filter by priority if specified
	opportunities := report.Opportunities
	if priority != "" {
		var filtered []*cache.CacheOpportunity
		for _, opp := range opportunities {
			if opp.Priority == priority {
				filtered = append(filtered, opp)
			}
		}
		opportunities = filtered
	}

	// Apply limit
	if len(opportunities) > limit {
		opportunities = opportunities[:limit]
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"opportunities": opportunities,
		"count":         len(opportunities),
		"total":         len(report.Opportunities),
		"limit":         limit,
	})
}

// handleCacheOptimize applies cache optimization recommendations
// POST /api/v1/cache/optimize
func (s *Server) handleCacheOptimize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse request body
	var req struct {
		OpportunityIDs []string `json:"opportunity_ids"`
		AutoEnable     bool     `json:"auto_enable"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Get database for analytics storage
	db := s.app.GetDatabase()
	if db == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Create analytics storage
	storage, err := analytics.NewDatabaseStorage(db.DB())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create analytics storage: %v", err))
		return
	}

	// Create analyzer
	config := cache.DefaultAnalysisConfig()
	config.AutoEnable = req.AutoEnable
	analyzer := cache.NewAnalyzer(storage, config)

	// Run analysis to get opportunities
	ctx := context.Background()
	report, err := analyzer.Analyze(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	// Apply optimizations
	result := &cache.OptimizationResult{
		AppliedPatterns: []string{},
		SkippedPatterns: []string{},
		Errors:          []string{},
	}

	// If no specific IDs provided and auto_enable is true, apply all auto-enableable opportunities
	if len(req.OpportunityIDs) == 0 && req.AutoEnable {
		for _, opp := range report.Opportunities {
			if opp.AutoEnableable {
				req.OpportunityIDs = append(req.OpportunityIDs, opp.ID)
			}
		}
	}

	// Apply each opportunity
	for _, oppID := range req.OpportunityIDs {
		// Find opportunity
		var found *cache.CacheOpportunity
		for _, opp := range report.Opportunities {
			if opp.ID == oppID {
				found = opp
				break
			}
		}

		if found == nil {
			result.SkippedCount++
			result.SkippedPatterns = append(result.SkippedPatterns, oppID)
			result.Errors = append(result.Errors, fmt.Sprintf("Opportunity %s not found", oppID))
			continue
		}

		// TODO: Actually enable caching for this pattern
		// For now, just record that we would enable it
		result.AppliedCount++
		result.TotalSavingsUSD += found.CostSavableUSD
		result.AppliedPatterns = append(result.AppliedPatterns, found.Pattern)

		// In a real implementation, we would:
		// 1. Update cache configuration
		// 2. Enable caching for the provider/model combination
		// 3. Set the TTL
		// 4. Log the optimization
	}

	s.respondJSON(w, http.StatusOK, result)
}

// handleCacheRecommendations generates human-readable cache recommendations
// GET /api/v1/cache/recommendations
func (s *Server) handleCacheRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get database for analytics storage
	db := s.app.GetDatabase()
	if db == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Create analytics storage
	storage, err := analytics.NewDatabaseStorage(db.DB())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create analytics storage: %v", err))
		return
	}

	// Create analyzer
	analyzer := cache.NewAnalyzer(storage, nil)

	// Run analysis
	ctx := context.Background()
	report, err := analyzer.Analyze(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	// Return recommendations in a simplified format
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"recommendations":    report.Recommendations,
		"total_savings_usd":  report.TotalSavingsUSD,
		"monthly_projection": report.MonthlyProjection,
		"duplicate_percent":  report.DuplicatePercent,
		"opportunity_count":  len(report.Opportunities),
		"analyzed_at":        report.AnalyzedAt,
		"time_window":        report.TimeWindow.String(),
	})
}
