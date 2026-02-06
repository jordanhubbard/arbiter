package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanhubbard/loom/internal/analytics"
	"github.com/jordanhubbard/loom/internal/auth"
)

// handleGetLogs handles GET /api/v1/analytics/logs
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserIDFromRequest(r)
	// If auth is disabled, allow access with empty userID (show all logs)
	if userID == "" && s.config.Security.EnableAuth {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filter := &analytics.LogFilter{
		UserID: userID, // Users can only see their own logs (or all if auth disabled)
		Limit:  100,    // Default limit
	}

	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		filter.ProviderID = providerID
	}

	if startTime := r.URL.Query().Get("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}

	if endTime := r.URL.Query().Get("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}

	logs, err := s.analyticsLogger.GetLogs(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetLogStats handles GET /api/v1/analytics/stats
func (s *Server) handleGetLogStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserIDFromRequest(r)
	// If auth is disabled, allow access with empty userID (show all stats)
	if userID == "" && s.config.Security.EnableAuth {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filter := &analytics.LogFilter{
		UserID: userID, // Users can only see their own stats (or all if auth disabled)
	}

	if startTime := r.URL.Query().Get("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}

	if endTime := r.URL.Query().Get("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}

	// Admin can see all stats
	role := auth.GetRoleFromRequest(r)
	if role == "admin" {
		filter.UserID = "" // Remove user filter for admins
		if queryUserID := r.URL.Query().Get("user_id"); queryUserID != "" {
			filter.UserID = queryUserID
		}
	}

	stats, err := s.analyticsLogger.GetStats(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleExportLogs handles GET /api/v1/analytics/export
func (s *Server) handleExportLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserIDFromRequest(r)
	// If auth is disabled, allow access with empty userID (export all logs)
	if userID == "" && s.config.Security.EnableAuth {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filter := &analytics.LogFilter{
		UserID: userID,
		Limit:  10000, // Higher limit for exports
	}

	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		filter.ProviderID = providerID
	}

	if startTime := r.URL.Query().Get("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}

	if endTime := r.URL.Query().Get("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}

	logs, err := s.analyticsLogger.GetLogs(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Default to JSON export
	format := r.URL.Query().Get("format")
	switch format {
	case "csv":
		exportLogsAsCSV(w, logs)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=\"logs.json\"")
		if err := json.NewEncoder(w).Encode(logs); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

// handleGetCostReport handles GET /api/v1/analytics/costs
func (s *Server) handleGetCostReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserIDFromRequest(r)
	// If auth is disabled, allow access with empty userID (show all costs)
	if userID == "" && s.config.Security.EnableAuth {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filter := &analytics.LogFilter{
		UserID: userID, // Users can only see their own costs by default (or all if auth disabled)
	}

	if startTime := r.URL.Query().Get("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}

	if endTime := r.URL.Query().Get("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}

	// Admin can see all costs
	role := auth.GetRoleFromRequest(r)
	if role == "admin" {
		filter.UserID = "" // Remove user filter for admins
		if queryUserID := r.URL.Query().Get("user_id"); queryUserID != "" {
			filter.UserID = queryUserID
		}
	}

	stats, err := s.analyticsLogger.GetStats(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build cost report
	costReport := map[string]interface{}{
		"total_cost_usd": stats.TotalCostUSD,
		"total_requests": stats.TotalRequests,
		"total_tokens":   stats.TotalTokens,
		"cost_per_request": func() float64 {
			if stats.TotalRequests > 0 {
				return stats.TotalCostUSD / float64(stats.TotalRequests)
			}
			return 0.0
		}(),
		"cost_per_1k_tokens": func() float64 {
			if stats.TotalTokens > 0 {
				return (stats.TotalCostUSD / float64(stats.TotalTokens)) * 1000
			}
			return 0.0
		}(),
		"cost_by_provider": stats.CostByProvider,
		"cost_by_user":     stats.CostByUser,
		"time_range": map[string]interface{}{
			"start": filter.StartTime,
			"end":   filter.EndTime,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(costReport); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetBatchingRecommendations handles GET /api/v1/analytics/batching
func (s *Server) handleGetBatchingRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.analyticsLogger == nil {
		http.Error(w, "Analytics unavailable", http.StatusServiceUnavailable)
		return
	}

	userID := auth.GetUserIDFromRequest(r)
	if userID == "" && s.config.Security.EnableAuth {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	filter := &analytics.LogFilter{
		UserID: userID,
		Limit:  2000,
	}

	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		filter.ProviderID = providerID
	}

	if startTime := r.URL.Query().Get("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}

	if endTime := r.URL.Query().Get("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}

	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			if parsed > 10000 {
				parsed = 10000
			}
			filter.Limit = parsed
		}
	}

	role := auth.GetRoleFromRequest(r)
	if role == "admin" {
		filter.UserID = ""
		if queryUserID := r.URL.Query().Get("user_id"); queryUserID != "" {
			filter.UserID = queryUserID
		}
	}

	logs, err := s.analyticsLogger.GetLogs(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	options := analytics.DefaultBatchingOptions()
	if maxRecommendations := r.URL.Query().Get("max_recommendations"); maxRecommendations != "" {
		if parsed, err := strconv.Atoi(maxRecommendations); err == nil && parsed > 0 {
			options.MaxRecommendations = parsed
		}
	}
	if windowMinutes := r.URL.Query().Get("window_minutes"); windowMinutes != "" {
		if parsed, err := strconv.Atoi(windowMinutes); err == nil && parsed > 0 {
			options.Window = time.Duration(parsed) * time.Minute
		}
	}
	if autoBatch := r.URL.Query().Get("auto_batch"); autoBatch != "" {
		options.IncludeAutoBatchPlan = autoBatch == "true" || autoBatch == "1"
	}

	recommendations := analytics.BuildBatchingRecommendations(logs, options)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(recommendations); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleExportStats handles GET /api/v1/analytics/export-stats
func (s *Server) handleExportStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserIDFromRequest(r)
	// If auth is disabled, allow access with empty userID (export all stats)
	if userID == "" && s.config.Security.EnableAuth {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filter := &analytics.LogFilter{
		UserID: userID,
	}

	if startTime := r.URL.Query().Get("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}

	if endTime := r.URL.Query().Get("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}

	// Admin can see all stats
	role := auth.GetRoleFromRequest(r)
	if role == "admin" {
		filter.UserID = ""
		if queryUserID := r.URL.Query().Get("user_id"); queryUserID != "" {
			filter.UserID = queryUserID
		}
	}

	stats, err := s.analyticsLogger.GetStats(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Export format
	format := r.URL.Query().Get("format")
	switch format {
	case "csv":
		exportStatsAsCSV(w, stats, filter)
	default:
		// JSON export
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=\"loom-stats-"+time.Now().Format("2006-01-02")+".json\"")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"exported_at": time.Now().Format(time.RFC3339),
			"time_range": map[string]string{
				"start": filter.StartTime.Format(time.RFC3339),
				"end":   filter.EndTime.Format(time.RFC3339),
			},
			"summary": map[string]interface{}{
				"total_requests": stats.TotalRequests,
				"total_tokens":   stats.TotalTokens,
				"total_cost_usd": stats.TotalCostUSD,
				"avg_latency_ms": stats.AvgLatencyMs,
				"error_rate":     stats.ErrorRate,
			},
			"cost_by_provider":     stats.CostByProvider,
			"cost_by_user":         stats.CostByUser,
			"requests_by_provider": stats.RequestsByProvider,
			"requests_by_user":     stats.RequestsByUser,
		}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

// exportStatsAsCSV exports stats summary in CSV format
func exportStatsAsCSV(w http.ResponseWriter, stats *analytics.LogStats, filter *analytics.LogFilter) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"loom-stats-"+time.Now().Format("2006-01-02")+".csv\"")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Summary section
	_ = writer.Write([]string{"Summary", "", "", ""})
	_ = writer.Write([]string{"Metric", "Value", "", ""})
	_ = writer.Write([]string{"Total Requests", fmt.Sprintf("%d", stats.TotalRequests), "", ""})
	_ = writer.Write([]string{"Total Tokens", fmt.Sprintf("%d", stats.TotalTokens), "", ""})
	_ = writer.Write([]string{"Total Cost (USD)", fmt.Sprintf("%.4f", stats.TotalCostUSD), "", ""})
	_ = writer.Write([]string{"Avg Latency (ms)", fmt.Sprintf("%.2f", stats.AvgLatencyMs), "", ""})
	_ = writer.Write([]string{"Error Rate", fmt.Sprintf("%.2f%%", stats.ErrorRate*100), "", ""})
	_ = writer.Write([]string{""})

	// Cost by Provider
	_ = writer.Write([]string{"Cost by Provider", "", "", ""})
	_ = writer.Write([]string{"Provider ID", "Requests", "Cost (USD)", ""})
	for provider, cost := range stats.CostByProvider {
		requests := stats.RequestsByProvider[provider]
		_ = writer.Write([]string{provider, fmt.Sprintf("%d", requests), fmt.Sprintf("%.4f", cost), ""})
	}
	_ = writer.Write([]string{""})

	// Cost by User
	_ = writer.Write([]string{"Cost by User", "", "", ""})
	_ = writer.Write([]string{"User ID", "Requests", "Cost (USD)", ""})
	for user, cost := range stats.CostByUser {
		requests := stats.RequestsByUser[user]
		_ = writer.Write([]string{user, fmt.Sprintf("%d", requests), fmt.Sprintf("%.4f", cost), ""})
	}
}

// exportLogsAsCSV exports logs in CSV format
func exportLogsAsCSV(w http.ResponseWriter, logs []*analytics.RequestLog) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"loom-logs-"+time.Now().Format("2006-01-02")+".csv\"")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write CSV header
	_ = writer.Write([]string{
		"Timestamp",
		"User ID",
		"Method",
		"Path",
		"Provider ID",
		"Model",
		"Prompt Tokens",
		"Completion Tokens",
		"Total Tokens",
		"Latency (ms)",
		"Status Code",
		"Cost (USD)",
		"Error Message",
	})

	// Write data rows
	for _, log := range logs {
		_ = writer.Write([]string{
			log.Timestamp.Format(time.RFC3339),
			log.UserID,
			log.Method,
			log.Path,
			log.ProviderID,
			log.ModelName,
			fmt.Sprintf("%d", log.PromptTokens),
			fmt.Sprintf("%d", log.CompletionTokens),
			fmt.Sprintf("%d", log.TotalTokens),
			fmt.Sprintf("%d", log.LatencyMs),
			fmt.Sprintf("%d", log.StatusCode),
			fmt.Sprintf("%.4f", log.CostUSD),
			log.ErrorMessage,
		})
	}
}
