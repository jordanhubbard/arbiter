package api

import (
	"encoding/json"
	"net/http"

	"github.com/jordanhubbard/agenticorp/internal/auth"
	"github.com/jordanhubbard/agenticorp/internal/cache"
)

// handleGetCacheStats handles GET /api/v1/cache/stats
func (s *Server) handleGetCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Authentication required
	userID := auth.GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get cache stats
	if s.cache == nil {
		http.Error(w, "Cache not initialized", http.StatusInternalServerError)
		return
	}

	stats := s.cache.GetStats(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleGetCacheConfig handles GET /api/v1/cache/config
func (s *Server) handleGetCacheConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Authentication required (admin only for config)
	role := auth.GetRoleFromRequest(r)
	if role != "admin" {
		http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		return
	}

	// Return current cache configuration
	if s.config == nil || s.cache == nil {
		http.Error(w, "Cache not configured", http.StatusInternalServerError)
		return
	}

	cacheConfig := s.config.Cache

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":        cacheConfig.Enabled,
		"default_ttl":    cacheConfig.DefaultTTL.String(),
		"max_size":       cacheConfig.MaxSize,
		"max_memory_mb":  cacheConfig.MaxMemoryMB,
		"cleanup_period": cacheConfig.CleanupPeriod.String(),
	})
}

// handleClearCache handles POST /api/v1/cache/clear
func (s *Server) handleClearCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Authentication required (admin only)
	role := auth.GetRoleFromRequest(r)
	if role != "admin" {
		http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		return
	}

	if s.cache == nil {
		http.Error(w, "Cache not initialized", http.StatusInternalServerError)
		return
	}

	// Clear the cache
	s.cache.Clear(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Cache cleared successfully",
	})
}

// CacheToCacheConfig converts cache.Config to a format suitable for API responses
func CacheToCacheConfig(c *cache.Config) map[string]interface{} {
	return map[string]interface{}{
		"enabled":        c.Enabled,
		"default_ttl":    c.DefaultTTL.String(),
		"max_size":       c.MaxSize,
		"max_memory_mb":  c.MaxMemoryMB,
		"cleanup_period": c.CleanupPeriod.String(),
	}
}
