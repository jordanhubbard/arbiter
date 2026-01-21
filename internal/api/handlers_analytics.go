package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jordanhubbard/agenticorp/internal/analytics"
	"github.com/jordanhubbard/agenticorp/internal/auth"
)

// handleGetLogs handles GET /api/v1/analytics/logs
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filter := &analytics.LogFilter{
		UserID: userID, // Users can only see their own logs
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
	json.NewEncoder(w).Encode(logs)
}

// handleGetLogStats handles GET /api/v1/analytics/stats
func (s *Server) handleGetLogStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filter := &analytics.LogFilter{
		UserID: userID, // Users can only see their own stats
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
	json.NewEncoder(w).Encode(stats)
}

// handleExportLogs handles GET /api/v1/analytics/export
func (s *Server) handleExportLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserIDFromRequest(r)
	if userID == "" {
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
		json.NewEncoder(w).Encode(logs)
	}
}

// handleGetCostReport handles GET /api/v1/analytics/costs
func (s *Server) handleGetCostReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filter := &analytics.LogFilter{
		UserID: userID, // Users can only see their own costs by default
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
	json.NewEncoder(w).Encode(costReport)
}

// exportLogsAsCSV exports logs in CSV format
func exportLogsAsCSV(w http.ResponseWriter, logs []*analytics.RequestLog) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"logs.csv\"")

	// Write CSV header
	w.Write([]byte("timestamp,user_id,method,path,provider_id,model,tokens,latency_ms,cost_usd,status\n"))

	// Write data rows
	for _, log := range logs {
		w.Write([]byte(log.Timestamp.Format(time.RFC3339)))
		w.Write([]byte(","))
		w.Write([]byte(log.UserID))
		w.Write([]byte(","))
		w.Write([]byte(log.Method))
		w.Write([]byte(","))
		w.Write([]byte(log.Path))
		w.Write([]byte(","))
		w.Write([]byte(log.ProviderID))
		w.Write([]byte(","))
		w.Write([]byte(log.ModelName))
		w.Write([]byte(","))
		w.Write([]byte(string(rune(log.TotalTokens))))
		w.Write([]byte(","))
		w.Write([]byte(string(rune(log.LatencyMs))))
		w.Write([]byte(","))
		w.Write([]byte(string(rune(int(log.CostUSD * 100)))))
		w.Write([]byte(","))
		w.Write([]byte(string(rune(log.StatusCode))))
		w.Write([]byte("\n"))
	}
}
