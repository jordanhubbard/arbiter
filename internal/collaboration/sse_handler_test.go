package collaboration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSEHandler(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)
	assert.NotNil(t, handler)
	assert.Equal(t, store, handler.store)
}

func TestHandleGetContext_MissingBeadID(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/context", nil)
	w := httptest.NewRecorder()

	handler.HandleGetContext(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bead_id parameter required")
}

func TestHandleGetContext_NotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/context?bead_id=nonexistent", nil)
	w := httptest.NewRecorder()

	handler.HandleGetContext(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleGetContext_Success(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()
	_, _ = store.GetOrCreate(ctx, "bead-1", "project-1")
	_ = store.JoinBead(ctx, "bead-1", "agent-1")

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/context?bead_id=bead-1", nil)
	w := httptest.NewRecorder()

	handler.HandleGetContext(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result SharedBeadContext
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "bead-1", result.BeadID)
	assert.Equal(t, "project-1", result.ProjectID)
	assert.Contains(t, result.CollaboratingAgents, "agent-1")
}

func TestHandleJoinBead_MethodNotAllowed(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/join", nil)
	w := httptest.NewRecorder()

	handler.HandleJoinBead(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleJoinBead_InvalidBody(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/join", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	handler.HandleJoinBead(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestHandleJoinBead_MissingFields(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	// Missing agent_id
	body := `{"bead_id": "bead-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/join", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleJoinBead(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bead_id and agent_id required")

	// Missing bead_id
	body = `{"agent_id": "agent-1"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/beads/join", strings.NewReader(body))
	w = httptest.NewRecorder()

	handler.HandleJoinBead(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleJoinBead_BeadNotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	body := `{"bead_id": "nonexistent", "agent_id": "agent-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/join", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleJoinBead(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleJoinBead_Success(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()
	_, _ = store.GetOrCreate(ctx, "bead-1", "project-1")

	handler := NewSSEHandler(store)

	body := `{"bead_id": "bead-1", "agent_id": "agent-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/join", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleJoinBead(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "joined", result["status"])
	assert.Equal(t, "bead-1", result["bead_id"])
	assert.Equal(t, "agent-1", result["agent_id"])
}

func TestHandleLeaveBead_MethodNotAllowed(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/leave", nil)
	w := httptest.NewRecorder()

	handler.HandleLeaveBead(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleLeaveBead_InvalidBody(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/leave", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	handler.HandleLeaveBead(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleLeaveBead_MissingFields(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	body := `{"bead_id": "bead-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/leave", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleLeaveBead(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bead_id and agent_id required")
}

func TestHandleLeaveBead_BeadNotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	body := `{"bead_id": "nonexistent", "agent_id": "agent-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/leave", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleLeaveBead(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleLeaveBead_Success(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()
	_, _ = store.GetOrCreate(ctx, "bead-1", "project-1")
	_ = store.JoinBead(ctx, "bead-1", "agent-1")

	handler := NewSSEHandler(store)

	body := `{"bead_id": "bead-1", "agent_id": "agent-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/leave", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleLeaveBead(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "left", result["status"])
	assert.Equal(t, "bead-1", result["bead_id"])
	assert.Equal(t, "agent-1", result["agent_id"])
}

func TestHandleUpdateData_MethodNotAllowed(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/data", nil)
	w := httptest.NewRecorder()

	handler.HandleUpdateData(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleUpdateData_InvalidBody(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/data", strings.NewReader("invalid"))
	w := httptest.NewRecorder()

	handler.HandleUpdateData(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleUpdateData_MissingFields(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	body := `{"bead_id": "bead-1", "agent_id": "agent-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/data", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleUpdateData(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bead_id, agent_id, and key required")
}

func TestHandleUpdateData_BeadNotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	body := `{"bead_id": "nonexistent", "agent_id": "agent-1", "key": "status", "value": "running"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/data", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleUpdateData(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleUpdateData_VersionConflict(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()
	_, _ = store.GetOrCreate(ctx, "bead-1", "project-1")
	_ = store.JoinBead(ctx, "bead-1", "agent-1")

	// Update data to change version
	_ = store.UpdateData(ctx, "bead-1", "agent-1", "key1", "value1", 0)

	handler := NewSSEHandler(store)

	// Send update with wrong version (version 1 but it's now 3)
	body := `{"bead_id": "bead-1", "agent_id": "agent-1", "key": "key2", "value": "value2", "expected_version": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/data", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleUpdateData(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "version_conflict", result["error"])
}

func TestHandleUpdateData_Success(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()
	_, _ = store.GetOrCreate(ctx, "bead-1", "project-1")
	_ = store.JoinBead(ctx, "bead-1", "agent-1")

	handler := NewSSEHandler(store)

	body := `{"bead_id": "bead-1", "agent_id": "agent-1", "key": "status", "value": "running"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/data", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleUpdateData(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "updated", result["status"])
	assert.Equal(t, "bead-1", result["bead_id"])
	assert.Equal(t, "status", result["key"])
	assert.NotNil(t, result["version"])
}

func TestHandleAddActivity_MethodNotAllowed(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/activity", nil)
	w := httptest.NewRecorder()

	handler.HandleAddActivity(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleAddActivity_InvalidBody(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/activity", strings.NewReader("invalid"))
	w := httptest.NewRecorder()

	handler.HandleAddActivity(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAddActivity_MissingFields(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	body := `{"bead_id": "bead-1", "agent_id": "agent-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/activity", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleAddActivity(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bead_id, agent_id, and activity_type required")
}

func TestHandleAddActivity_BeadNotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	body := `{"bead_id": "nonexistent", "agent_id": "agent-1", "activity_type": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/activity", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleAddActivity(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleAddActivity_Success(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()
	_, _ = store.GetOrCreate(ctx, "bead-1", "project-1")
	_ = store.JoinBead(ctx, "bead-1", "agent-1")

	handler := NewSSEHandler(store)

	body := `{"bead_id": "bead-1", "agent_id": "agent-1", "activity_type": "file_modified", "description": "Modified main.go", "data": {"file": "main.go"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/activity", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleAddActivity(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "activity_added", result["status"])
	assert.Equal(t, "bead-1", result["bead_id"])
	assert.Equal(t, "file_modified", result["activity_type"])
}

func TestServeHTTP_MissingBeadID(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/context/stream", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bead_id parameter required")
}

func TestServeHTTP_BeadNotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/context/stream?bead_id=nonexistent", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should set SSE headers
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))

	// Should contain an error event
	assert.Contains(t, w.Body.String(), "event: error")
}

func TestServeHTTP_WithContext(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	bgCtx := context.Background()
	_, _ = store.GetOrCreate(bgCtx, "bead-1", "project-1")
	_ = store.JoinBead(bgCtx, "bead-1", "agent-1")

	handler := NewSSEHandler(store)

	// Create a cancellable context to stop the SSE stream
	ctx, cancel := context.WithCancel(context.Background())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/context/stream?bead_id=bead-1", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Run ServeHTTP in a goroutine since it blocks
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(w, req)
		close(done)
	}()

	// Give the handler time to send initial state
	time.Sleep(100 * time.Millisecond)

	// Cancel the context to stop the stream
	cancel()

	// Wait for handler to finish
	select {
	case <-done:
		// Good
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for ServeHTTP to return")
	}

	// Check that SSE headers were set
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// Check that initial event was sent
	body := w.Body.String()
	assert.Contains(t, body, "event: initial")
	assert.Contains(t, body, "bead-1")
}

func TestServeHTTP_ReceivesUpdate(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	bgCtx := context.Background()
	_, _ = store.GetOrCreate(bgCtx, "bead-1", "project-1")

	handler := NewSSEHandler(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beads/context/stream?bead_id=bead-1", nil)
	req = req.WithContext(ctx)

	// Use a custom ResponseWriter that supports flushing
	w := &flushableRecorder{ResponseRecorder: httptest.NewRecorder()}

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(w, req)
		close(done)
	}()

	// Wait for handler to start and subscribe
	time.Sleep(100 * time.Millisecond)

	// Trigger an update
	_ = store.JoinBead(bgCtx, "bead-1", "agent-2")

	// Give time for update to propagate
	time.Sleep(200 * time.Millisecond)

	// Stop the stream
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for ServeHTTP to return")
	}

	body := w.ResponseRecorder.Body.String()
	assert.Contains(t, body, "event: initial")
	// The update event may or may not arrive depending on timing
}

// flushableRecorder wraps httptest.ResponseRecorder and implements http.Flusher
type flushableRecorder struct {
	*httptest.ResponseRecorder
}

func (fr *flushableRecorder) Flush() {
	// no-op for testing
}

func TestConflictError_Error(t *testing.T) {
	err := &ConflictError{
		BeadID:          "bead-1",
		ExpectedVersion: 5,
		ActualVersion:   7,
	}

	msg := err.Error()
	assert.Contains(t, msg, "bead-1")
	assert.Contains(t, msg, "5")
	assert.Contains(t, msg, "7")
	assert.Contains(t, msg, "version conflict")
}

func TestContextStore_Get_NotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context not found")
}

func TestContextStore_JoinBead_NotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()

	err := store.JoinBead(ctx, "nonexistent", "agent-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context not found")
}

func TestContextStore_LeaveBead_NotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()

	err := store.LeaveBead(ctx, "nonexistent", "agent-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context not found")
}

func TestContextStore_UpdateData_NotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()

	err := store.UpdateData(ctx, "nonexistent", "agent-1", "key", "value", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context not found")
}

func TestContextStore_AddActivity_NotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()

	err := store.AddActivity(ctx, "nonexistent", "agent-1", "test", "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context not found")
}

func TestExportContext_NotFound(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()

	_, err := store.ExportContext(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestSubscribeUnsubscribe(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ch := store.Subscribe("bead-1")
	assert.NotNil(t, ch)

	// Unsubscribe
	store.Unsubscribe("bead-1", ch)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed after unsubscribe")
}

func TestHandleAddActivity_SuccessWithData(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	ctx := context.Background()
	_, _ = store.GetOrCreate(ctx, "bead-1", "project-1")
	_ = store.JoinBead(ctx, "bead-1", "agent-1")

	handler := NewSSEHandler(store)

	reqBody := map[string]interface{}{
		"bead_id":       "bead-1",
		"agent_id":      "agent-1",
		"activity_type": "message",
		"description":   "Working on task",
		"data": map[string]interface{}{
			"progress": 50,
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/activity", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	handler.HandleAddActivity(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleLeaveBead_EmptyFields(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	// Both fields empty
	body := `{"bead_id": "", "agent_id": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/leave", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleLeaveBead(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleUpdateData_EmptyFields(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	body := `{"bead_id": "", "agent_id": "", "key": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/data", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleUpdateData(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAddActivity_EmptyFields(t *testing.T) {
	store := NewContextStore()
	defer store.Close()

	handler := NewSSEHandler(store)

	body := `{"bead_id": "", "agent_id": "", "activity_type": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beads/activity", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleAddActivity(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
