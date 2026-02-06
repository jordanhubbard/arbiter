package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jordanhubbard/loom/internal/temporal/eventbus"
)

// handleEventStream handles SSE endpoint for real-time event updates
// GET /api/v1/events/stream
func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	eventBus := s.app.GetEventBus()
	if eventBus == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Event bus not available")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get optional filters from query params
	projectID := r.URL.Query().Get("project_id")
	eventType := r.URL.Query().Get("type")

	// Create subscriber with filter
	subscriberID := fmt.Sprintf("sse-%d", time.Now().UnixNano())
	filter := func(event *eventbus.Event) bool {
		if projectID != "" && event.ProjectID != projectID {
			return false
		}
		if eventType != "" && string(event.Type) != eventType {
			return false
		}
		return true
	}

	subscriber := eventBus.Subscribe(subscriberID, filter)
	defer eventBus.Unsubscribe(subscriberID)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\n")
	fmt.Fprintf(w, "data: {\"message\": \"Connected to event stream\"}\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream events to client
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		case event, ok := <-subscriber.Channel:
			if !ok {
				// Channel closed
				return
			}

			// Send event to client
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "event: %s\n", event.Type)
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

// handleGetEvents handles GET requests for recent events
// GET /api/v1/events?project_id=xxx&type=xxx&limit=100
func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// For now, return empty array since we need to implement event storage
	// In a real implementation, you'd query Temporal workflows or a separate event store
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"events":  []interface{}{},
		"message": "Event history not yet implemented - use /api/v1/events/stream for real-time events",
	})
}

// handleGetEventStats returns statistics about events
// GET /api/v1/events/stats
func (s *Server) handleGetEventStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	eventBus := s.app.GetEventBus()
	if eventBus == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Event bus not available")
		return
	}

	// Return basic stats (in a real implementation, track these in the event bus)
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "active",
		"subscribers": 0, // Would need to expose this from EventBus
		"message":     "Event bus is operational",
	})
}
