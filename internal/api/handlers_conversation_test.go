package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/auth"
	"github.com/jordanhubbard/loom/internal/database"
	"github.com/jordanhubbard/loom/internal/loom"
	"github.com/jordanhubbard/loom/internal/keymanager"
	"github.com/jordanhubbard/loom/pkg/config"
	"github.com/jordanhubbard/loom/pkg/models"
)

func setupConversationTestServer(t *testing.T) (*Server, *database.Database, func()) {
	t.Helper()

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Try to create /app/src and /app/data directories for gitops
	// If this fails (not root), skip the test
	if err := os.MkdirAll("/app/src", 0755); err != nil {
		t.Skipf("Cannot create /app/src: %v - skipping test (requires root or container)", err)
	}
	if err := os.MkdirAll("/app/data/projects", 0755); err != nil {
		t.Skipf("Cannot create /app/data/projects: %v - skipping test (requires root or container)", err)
	}

	// Create minimal config
	cfg := config.DefaultConfig()
	cfg.Database.Path = dbPath

	// Create loom instance
	corp, err := loom.New(cfg)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create loom instance: %v", err)
	}

	// Create key manager and auth manager
	kmPath := filepath.Join(tmpDir, "keys.json")
	km := keymanager.NewKeyManager(kmPath)
	am := auth.NewManager(tmpDir)

	// Create server
	server := NewServer(corp, km, am, cfg)

	cleanup := func() {
		corp.Shutdown()
		db.Close()
	}

	return server, db, cleanup
}

func TestHandleGetConversation(t *testing.T) {
	server, db, cleanup := setupConversationTestServer(t)
	defer cleanup()

	// Create a test conversation
	session := models.NewConversationContext(
		"test-session-123",
		"bead-456",
		"proj-789",
		24*time.Hour,
	)
	session.AddMessage("user", "Hello", 5)
	session.AddMessage("assistant", "Hi there!", 10)

	if err := db.CreateConversationContext(session); err != nil {
		t.Fatalf("Failed to create test conversation: %v", err)
	}

	tests := []struct {
		name           string
		sessionID      string
		expectedStatus int
		expectMessages int
	}{
		{
			name:           "Get existing conversation",
			sessionID:      "test-session-123",
			expectedStatus: http.StatusOK,
			expectMessages: 2,
		},
		{
			name:           "Get non-existent conversation",
			sessionID:      "non-existent",
			expectedStatus: http.StatusNotFound,
			expectMessages: 0,
		},
		{
			name:           "Empty session ID",
			sessionID:      "",
			expectedStatus: http.StatusBadRequest,
			expectMessages: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/conversations/" + tt.sessionID
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			server.handleConversation(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var result models.ConversationContext
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if len(result.Messages) != tt.expectMessages {
					t.Errorf("Expected %d messages, got %d", tt.expectMessages, len(result.Messages))
				}

				if result.SessionID != tt.sessionID {
					t.Errorf("Expected session ID %s, got %s", tt.sessionID, result.SessionID)
				}
			}
		})
	}
}

func TestHandleDeleteConversation(t *testing.T) {
	server, db, cleanup := setupConversationTestServer(t)
	defer cleanup()

	// Create a test conversation
	session := models.NewConversationContext(
		"test-session-delete",
		"bead-456",
		"proj-789",
		24*time.Hour,
	)

	if err := db.CreateConversationContext(session); err != nil {
		t.Fatalf("Failed to create test conversation: %v", err)
	}

	tests := []struct {
		name           string
		sessionID      string
		expectedStatus int
	}{
		{
			name:           "Delete existing conversation",
			sessionID:      "test-session-delete",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Delete non-existent conversation",
			sessionID:      "non-existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/conversations/" + tt.sessionID
			req := httptest.NewRequest(http.MethodDelete, path, nil)
			w := httptest.NewRecorder()

			server.handleConversation(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var result map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if result["session_id"] != tt.sessionID {
					t.Errorf("Expected session ID %s in response, got %v", tt.sessionID, result["session_id"])
				}

				// Verify conversation is actually deleted
				_, err := db.GetConversationContext(tt.sessionID)
				if err == nil {
					t.Error("Conversation should have been deleted from database")
				}
			}
		})
	}
}

func TestHandleResetConversation(t *testing.T) {
	server, db, cleanup := setupConversationTestServer(t)
	defer cleanup()

	// Create a test conversation with messages
	session := models.NewConversationContext(
		"test-session-reset",
		"bead-456",
		"proj-789",
		24*time.Hour,
	)
	session.AddMessage("system", "You are a helpful assistant", 10)
	session.AddMessage("user", "Hello", 5)
	session.AddMessage("assistant", "Hi there!", 10)

	if err := db.CreateConversationContext(session); err != nil {
		t.Fatalf("Failed to create test conversation: %v", err)
	}

	tests := []struct {
		name              string
		sessionID         string
		keepSystemMessage bool
		expectedStatus    int
		expectedMessages  int
	}{
		{
			name:              "Reset with system message",
			sessionID:         "test-session-reset",
			keepSystemMessage: true,
			expectedStatus:    http.StatusOK,
			expectedMessages:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"keep_system_message": tt.keepSystemMessage,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			path := fmt.Sprintf("/api/v1/conversations/%s/reset", tt.sessionID)
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleConversation(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				// Verify conversation was reset in database
				result, err := db.GetConversationContext(tt.sessionID)
				if err != nil {
					t.Fatalf("Failed to get conversation after reset: %v", err)
				}

				if len(result.Messages) != tt.expectedMessages {
					t.Errorf("Expected %d messages after reset, got %d", tt.expectedMessages, len(result.Messages))
				}

				if tt.keepSystemMessage && len(result.Messages) > 0 {
					if result.Messages[0].Role != "system" {
						t.Error("Expected first message to be system message")
					}
				}
			}
		})
	}
}

func TestHandleBeadConversation(t *testing.T) {
	server, db, cleanup := setupConversationTestServer(t)
	defer cleanup()

	// Create a test conversation linked to a bead
	session := models.NewConversationContext(
		"test-session-bead",
		"bead-123",
		"proj-789",
		24*time.Hour,
	)
	session.AddMessage("user", "Test message", 5)

	if err := db.CreateConversationContext(session); err != nil {
		t.Fatalf("Failed to create test conversation: %v", err)
	}

	tests := []struct {
		name           string
		beadID         string
		expectedStatus int
	}{
		{
			name:           "Get conversation for existing bead",
			beadID:         "bead-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get conversation for non-existent bead",
			beadID:         "non-existent-bead",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/api/v1/beads/%s/conversation", tt.beadID)
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			server.handleBeadConversation(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var result models.ConversationContext
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if result.BeadID != tt.beadID {
					t.Errorf("Expected bead ID %s, got %s", tt.beadID, result.BeadID)
				}
			}
		})
	}
}

func TestHandleConversationsList(t *testing.T) {
	server, db, cleanup := setupConversationTestServer(t)
	defer cleanup()

	projectID := "proj-list-test"

	// Create multiple test conversations for the same project
	for i := 0; i < 3; i++ {
		session := models.NewConversationContext(
			fmt.Sprintf("session-%d", i),
			fmt.Sprintf("bead-%d", i),
			projectID,
			24*time.Hour,
		)
		if err := db.CreateConversationContext(session); err != nil {
			t.Fatalf("Failed to create test conversation %d: %v", i, err)
		}
	}

	tests := []struct {
		name           string
		projectID      string
		limit          string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "List conversations for project",
			projectID:      projectID,
			limit:          "10",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "List with limit",
			projectID:      projectID,
			limit:          "2",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "Missing project_id",
			projectID:      "",
			limit:          "10",
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:           "Invalid limit",
			projectID:      projectID,
			limit:          "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/conversations"
			if tt.projectID != "" || tt.limit != "" {
				path += "?"
				if tt.projectID != "" {
					path += "project_id=" + tt.projectID
				}
				if tt.limit != "" {
					if tt.projectID != "" {
						path += "&"
					}
					path += "limit=" + tt.limit
				}
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			server.handleConversationsList(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var result map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				conversations, ok := result["conversations"].([]interface{})
				if !ok {
					t.Fatal("Expected conversations to be an array")
				}

				if len(conversations) != tt.expectedCount {
					t.Errorf("Expected %d conversations, got %d", tt.expectedCount, len(conversations))
				}
			}
		})
	}
}

func TestHandleConversation_MethodNotAllowed(t *testing.T) {
	server, _, cleanup := setupConversationTestServer(t)
	defer cleanup()

	// Test unsupported methods
	methods := []string{http.MethodPut, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/conversations/test-session", nil)
			w := httptest.NewRecorder()

			server.handleConversation(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s method, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}
