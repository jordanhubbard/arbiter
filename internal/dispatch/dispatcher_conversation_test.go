package dispatch

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jordanhubbard/agenticorp/internal/database"
	"github.com/jordanhubbard/agenticorp/pkg/models"
)

func TestDispatcher_getOrCreateConversationSession(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create dispatcher with database
	d := &Dispatcher{
		db: db,
	}

	// Test bead without conversation session
	bead := &models.Bead{
		ID:        "bead-test-123",
		ProjectID: "proj-test-456",
		Context:   make(map[string]string),
	}

	projectID := "proj-test-456"

	t.Run("Create new session", func(t *testing.T) {
		// Create new conversation session
		session, err := d.getOrCreateConversationSession(bead, projectID)
		if err != nil {
			t.Fatalf("Failed to create conversation session: %v", err)
		}

		if session == nil {
			t.Fatal("Expected session to be created, got nil")
		}

		if session.BeadID != bead.ID {
			t.Errorf("Session bead ID mismatch: got %s, want %s", session.BeadID, bead.ID)
		}

		if session.ProjectID != projectID {
			t.Errorf("Session project ID mismatch: got %s, want %s", session.ProjectID, projectID)
		}

		if session.IsExpired() {
			t.Error("Newly created session should not be expired")
		}

		// Verify session ID was stored in bead context
		sessionID := bead.Context["conversation_session_id"]
		if sessionID == "" {
			t.Error("Session ID should be stored in bead context")
		}

		if sessionID != session.SessionID {
			t.Errorf("Stored session ID mismatch: got %s, want %s", sessionID, session.SessionID)
		}

		// Verify session exists in database
		dbSession, err := db.GetConversationContext(session.SessionID)
		if err != nil {
			t.Fatalf("Failed to retrieve session from database: %v", err)
		}

		if dbSession.SessionID != session.SessionID {
			t.Errorf("Database session ID mismatch: got %s, want %s", dbSession.SessionID, session.SessionID)
		}
	})

	t.Run("Resume existing session", func(t *testing.T) {
		// Get the session ID from previous test
		existingSessionID := bead.Context["conversation_session_id"]

		// Try to get session again - should return existing one
		session, err := d.getOrCreateConversationSession(bead, projectID)
		if err != nil {
			t.Fatalf("Failed to get conversation session: %v", err)
		}

		if session == nil {
			t.Fatal("Expected existing session, got nil")
		}

		if session.SessionID != existingSessionID {
			t.Errorf("Should have resumed existing session: got %s, want %s", session.SessionID, existingSessionID)
		}
	})

	t.Run("Create new session when expired", func(t *testing.T) {
		// Create an expired session
		expiredBead := &models.Bead{
			ID:        "bead-expired-789",
			ProjectID: "proj-test-456",
			Context:   make(map[string]string),
		}

		// Create session with negative expiration
		expiredSession := models.NewConversationContext(
			"expired-session-id",
			expiredBead.ID,
			projectID,
			-1*time.Hour, // Already expired
		)

		// Save expired session
		if err := db.CreateConversationContext(expiredSession); err != nil {
			t.Fatalf("Failed to create expired session: %v", err)
		}

		// Store expired session ID in bead
		expiredBead.Context["conversation_session_id"] = expiredSession.SessionID

		// Try to get session - should create new one because old one is expired
		newSession, err := d.getOrCreateConversationSession(expiredBead, projectID)
		if err != nil {
			t.Fatalf("Failed to create new session after expiration: %v", err)
		}

		if newSession == nil {
			t.Fatal("Expected new session to be created, got nil")
		}

		if newSession.SessionID == expiredSession.SessionID {
			t.Error("Should have created new session, but got the expired one")
		}

		if newSession.IsExpired() {
			t.Error("New session should not be expired")
		}

		// Verify new session ID was stored in bead context
		newSessionID := expiredBead.Context["conversation_session_id"]
		if newSessionID == expiredSession.SessionID {
			t.Error("Bead context should have been updated with new session ID")
		}

		if newSessionID != newSession.SessionID {
			t.Errorf("Stored session ID mismatch: got %s, want %s", newSessionID, newSession.SessionID)
		}
	})

	t.Run("Handle missing session gracefully", func(t *testing.T) {
		// Create bead with non-existent session ID
		missingBead := &models.Bead{
			ID:        "bead-missing-999",
			ProjectID: "proj-test-456",
			Context: map[string]string{
				"conversation_session_id": "non-existent-session-id",
			},
		}

		// Should create new session when existing one can't be found
		session, err := d.getOrCreateConversationSession(missingBead, projectID)
		if err != nil {
			t.Fatalf("Failed to handle missing session: %v", err)
		}

		if session == nil {
			t.Fatal("Expected new session to be created, got nil")
		}

		if session.SessionID == "non-existent-session-id" {
			t.Error("Should have created new session, not returned the non-existent one")
		}
	})

	t.Run("No database available", func(t *testing.T) {
		// Dispatcher without database
		dNoDb := &Dispatcher{
			db: nil,
		}

		bead := &models.Bead{
			ID:        "bead-no-db",
			ProjectID: "proj-test",
		}

		// Should return nil without error when no database
		session, err := dNoDb.getOrCreateConversationSession(bead, "proj-test")
		if err != nil {
			t.Errorf("Should not error when no database: %v", err)
		}

		if session != nil {
			t.Error("Should return nil session when no database")
		}
	})
}

func TestDispatcher_ConversationSessionWithMetadata(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	d := &Dispatcher{
		db: db,
	}

	// Create bead with agent/provider context
	bead := &models.Bead{
		ID:        "bead-with-metadata",
		ProjectID: "proj-test",
		Context: map[string]string{
			"agent_id":    "agent-123",
			"provider_id": "provider-456",
		},
	}

	session, err := d.getOrCreateConversationSession(bead, "proj-test")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify metadata was copied from bead context
	if session.Metadata["agent_id"] != "agent-123" {
		t.Errorf("Agent ID not copied to metadata: got %s, want agent-123", session.Metadata["agent_id"])
	}

	if session.Metadata["provider_id"] != "provider-456" {
		t.Errorf("Provider ID not copied to metadata: got %s, want provider-456", session.Metadata["provider_id"])
	}
}

func TestDispatcher_ConversationSessionIntegration(t *testing.T) {
	// This test simulates a full dispatch cycle but only tests the session management part
	// We don't actually execute the task, just verify the session is created/passed correctly

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	d := &Dispatcher{
		db: db,
	}

	ctx := context.Background()
	_ = ctx // For future use

	// Simulate first dispatch
	bead := &models.Bead{
		ID:        "bead-integration-test",
		ProjectID: "proj-test",
		Context:   make(map[string]string),
	}

	session1, err := d.getOrCreateConversationSession(bead, "proj-test")
	if err != nil {
		t.Fatalf("Failed to create session on first dispatch: %v", err)
	}

	if session1 == nil {
		t.Fatal("Expected session on first dispatch")
	}

	sessionID1 := session1.SessionID

	// Simulate second dispatch (redispatch) - should resume same session
	session2, err := d.getOrCreateConversationSession(bead, "proj-test")
	if err != nil {
		t.Fatalf("Failed to resume session on second dispatch: %v", err)
	}

	if session2 == nil {
		t.Fatal("Expected session on second dispatch")
	}

	if session2.SessionID != sessionID1 {
		t.Errorf("Second dispatch should resume same session: got %s, want %s", session2.SessionID, sessionID1)
	}

	// Verify both sessions point to same underlying data
	if session1.BeadID != session2.BeadID {
		t.Error("Sessions should have same bead ID")
	}
}
