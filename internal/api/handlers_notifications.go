package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jordanhubbard/loom/internal/notifications"
)

// handleGetNotifications handles GET requests for user notifications
// GET /api/v1/notifications?status=unread&limit=50
func (s *Server) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	notificationMgr := s.app.GetNotificationManager()
	if notificationMgr == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Notification manager not available")
		return
	}

	// Get user from context (set by auth middleware)
	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	notifs, err := notificationMgr.GetNotifications(user.ID, status, limit, offset)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get notifications: %v", err))
		return
	}

	// Filter by priority if specified
	if priority != "" {
		filtered := make([]*notifications.Notification, 0)
		for _, n := range notifs {
			if n.Priority == priority {
				filtered = append(filtered, n)
			}
		}
		notifs = filtered
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": notifs,
		"count":         len(notifs),
		"limit":         limit,
		"offset":        offset,
	})
}

// handleNotificationStream handles SSE endpoint for real-time user notifications
// GET /api/v1/notifications/stream
func (s *Server) handleNotificationStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	notificationMgr := s.app.GetNotificationManager()
	if notificationMgr == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Notification manager not available")
		return
	}

	// Get user from context
	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create subscriber
	subscriberID := fmt.Sprintf("notification-sse-%d", time.Now().UnixNano())
	subscriber := notificationMgr.Subscribe(user.ID, subscriberID)
	defer notificationMgr.Unsubscribe(user.ID, subscriberID)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\n")
	fmt.Fprintf(w, "data: {\"message\": \"Connected to notification stream\"}\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream notifications to client
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		case notification, ok := <-subscriber:
			if !ok {
				// Channel closed
				return
			}

			// Send notification to client
			data, err := json.Marshal(notification)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "event: notification\n")
			fmt.Fprintf(w, "data: %s\n\n", data)

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-time.After(30 * time.Second):
			// Send keepalive ping
			fmt.Fprintf(w, ": keepalive\n\n")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// handleNotificationActions handles notification action requests
// POST /api/v1/notifications/{id}/read
func (s *Server) handleNotificationActions(w http.ResponseWriter, r *http.Request) {
	notificationMgr := s.app.GetNotificationManager()
	if notificationMgr == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Notification manager not available")
		return
	}

	// Get user from context
	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse notification ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/notifications/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		s.respondError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	notificationID := parts[0]
	action := parts[1]

	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	switch action {
	case "read":
		if err := notificationMgr.MarkRead(notificationID); err != nil {
			s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to mark notification as read: %v", err))
			return
		}
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Notification marked as read",
		})

	default:
		s.respondError(w, http.StatusBadRequest, "Invalid action")
	}
}

// handleMarkAllRead handles marking all notifications as read
// POST /api/v1/notifications/mark-all-read
func (s *Server) handleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	notificationMgr := s.app.GetNotificationManager()
	if notificationMgr == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Notification manager not available")
		return
	}

	// Get user from context
	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := notificationMgr.MarkAllRead(user.ID); err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to mark all notifications as read: %v", err))
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "All notifications marked as read",
	})
}

// handleNotificationPreferences handles notification preferences requests
// GET /api/v1/notifications/preferences
// PATCH /api/v1/notifications/preferences
func (s *Server) handleNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	notificationMgr := s.app.GetNotificationManager()
	if notificationMgr == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Notification manager not available")
		return
	}

	// Get user from context
	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		prefs, err := notificationMgr.GetPreferences(user.ID)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get preferences: %v", err))
			return
		}
		s.respondJSON(w, http.StatusOK, prefs)

	case http.MethodPatch:
		// Parse request body
		var updates notifications.NotificationPreferences
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		// Get existing preferences
		prefs, err := notificationMgr.GetPreferences(user.ID)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get preferences: %v", err))
			return
		}

		// Apply updates (only update fields that are present in request)
		// This is a simplified approach - in production you'd want to use JSON Patch or similar
		if updates.EnableInApp != prefs.EnableInApp {
			prefs.EnableInApp = updates.EnableInApp
		}
		if updates.EnableEmail != prefs.EnableEmail {
			prefs.EnableEmail = updates.EnableEmail
		}
		if updates.EnableWebhook != prefs.EnableWebhook {
			prefs.EnableWebhook = updates.EnableWebhook
		}
		if len(updates.SubscribedEvents) > 0 {
			prefs.SubscribedEvents = updates.SubscribedEvents
		}
		if updates.DigestMode != "" {
			prefs.DigestMode = updates.DigestMode
		}
		if updates.QuietHoursStart != "" {
			prefs.QuietHoursStart = updates.QuietHoursStart
		}
		if updates.QuietHoursEnd != "" {
			prefs.QuietHoursEnd = updates.QuietHoursEnd
		}
		if len(updates.ProjectFilters) > 0 {
			prefs.ProjectFilters = updates.ProjectFilters
		}
		if updates.MinPriority != "" {
			prefs.MinPriority = updates.MinPriority
		}

		// Save updates
		if err := notificationMgr.UpdatePreferences(prefs); err != nil {
			s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update preferences: %v", err))
			return
		}

		s.respondJSON(w, http.StatusOK, prefs)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
