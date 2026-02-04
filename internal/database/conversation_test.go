package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

func TestConversationContext_CRUD(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create conversation context
	ctx := models.NewConversationContext(
		"session-test-123",
		"bead-456",
		"proj-789",
		24*time.Hour,
	)
	ctx.AddMessage("system", "You are a helpful assistant", 10)
	ctx.AddMessage("user", "Hello, world!", 5)
	ctx.Metadata["agent_id"] = "agent-test"

	t.Run("Create", func(t *testing.T) {
		err := db.CreateConversationContext(ctx)
		if err != nil {
			t.Fatalf("Failed to create conversation context: %v", err)
		}
	})

	t.Run("Get by SessionID", func(t *testing.T) {
		retrieved, err := db.GetConversationContext(ctx.SessionID)
		if err != nil {
			t.Fatalf("Failed to get conversation context: %v", err)
		}

		if retrieved.SessionID != ctx.SessionID {
			t.Errorf("SessionID mismatch: got %s, want %s", retrieved.SessionID, ctx.SessionID)
		}
		if retrieved.BeadID != ctx.BeadID {
			t.Errorf("BeadID mismatch: got %s, want %s", retrieved.BeadID, ctx.BeadID)
		}
		if retrieved.ProjectID != ctx.ProjectID {
			t.Errorf("ProjectID mismatch: got %s, want %s", retrieved.ProjectID, ctx.ProjectID)
		}
		if len(retrieved.Messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(retrieved.Messages))
		}
		if retrieved.TokenCount != 15 {
			t.Errorf("TokenCount mismatch: got %d, want 15", retrieved.TokenCount)
		}
		if retrieved.Metadata["agent_id"] != "agent-test" {
			t.Errorf("Metadata agent_id mismatch: got %s, want agent-test", retrieved.Metadata["agent_id"])
		}
	})

	t.Run("Get by BeadID", func(t *testing.T) {
		retrieved, err := db.GetConversationContextByBeadID(ctx.BeadID)
		if err != nil {
			t.Fatalf("Failed to get conversation context by bead ID: %v", err)
		}

		if retrieved.SessionID != ctx.SessionID {
			t.Errorf("SessionID mismatch: got %s, want %s", retrieved.SessionID, ctx.SessionID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		// Add more messages
		ctx.AddMessage("assistant", "Hello! How can I help?", 8)

		err := db.UpdateConversationContext(ctx)
		if err != nil {
			t.Fatalf("Failed to update conversation context: %v", err)
		}

		// Verify update
		retrieved, err := db.GetConversationContext(ctx.SessionID)
		if err != nil {
			t.Fatalf("Failed to get updated conversation context: %v", err)
		}

		if len(retrieved.Messages) != 3 {
			t.Errorf("Expected 3 messages after update, got %d", len(retrieved.Messages))
		}
		if retrieved.TokenCount != 23 {
			t.Errorf("TokenCount mismatch after update: got %d, want 23", retrieved.TokenCount)
		}
	})

	t.Run("List by Project", func(t *testing.T) {
		// Create another conversation for the same project
		ctx2 := models.NewConversationContext(
			"session-test-456",
			"bead-789",
			"proj-789",
			24*time.Hour,
		)
		ctx2.AddMessage("system", "Another conversation", 5)

		err := db.CreateConversationContext(ctx2)
		if err != nil {
			t.Fatalf("Failed to create second conversation context: %v", err)
		}

		// List conversations for project
		conversations, err := db.ListConversationContextsByProject("proj-789", 10)
		if err != nil {
			t.Fatalf("Failed to list conversations: %v", err)
		}

		if len(conversations) != 2 {
			t.Errorf("Expected 2 conversations, got %d", len(conversations))
		}

		// Should be ordered by updated_at DESC, so ctx2 might be first
		// Just check that both are present
		foundCtx1 := false
		foundCtx2 := false
		for _, c := range conversations {
			if c.SessionID == ctx.SessionID {
				foundCtx1 = true
			}
			if c.SessionID == ctx2.SessionID {
				foundCtx2 = true
			}
		}
		if !foundCtx1 || !foundCtx2 {
			t.Error("Both conversations should be in the list")
		}
	})

	t.Run("Reset Messages", func(t *testing.T) {
		// Reset keeping system message
		err := db.ResetConversationMessages(ctx.SessionID, true)
		if err != nil {
			t.Fatalf("Failed to reset conversation messages: %v", err)
		}

		retrieved, err := db.GetConversationContext(ctx.SessionID)
		if err != nil {
			t.Fatalf("Failed to get conversation after reset: %v", err)
		}

		if len(retrieved.Messages) != 1 {
			t.Errorf("Expected 1 message after reset (system message), got %d", len(retrieved.Messages))
		}
		if retrieved.Messages[0].Role != "system" {
			t.Errorf("Expected system message after reset, got %s", retrieved.Messages[0].Role)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := db.DeleteConversationContext(ctx.SessionID)
		if err != nil {
			t.Fatalf("Failed to delete conversation context: %v", err)
		}

		// Verify deletion
		_, err = db.GetConversationContext(ctx.SessionID)
		if err == nil {
			t.Error("Expected error when getting deleted conversation, got nil")
		}
	})
}

func TestDeleteExpiredConversationContexts(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create expired conversation
	expiredCtx := models.NewConversationContext(
		"expired-session",
		"bead-1",
		"proj-1",
		-1*time.Hour, // Already expired
	)
	expiredCtx.AddMessage("system", "Expired conversation", 5)

	err = db.CreateConversationContext(expiredCtx)
	if err != nil {
		t.Fatalf("Failed to create expired conversation: %v", err)
	}

	// Create active conversation
	activeCtx := models.NewConversationContext(
		"active-session",
		"bead-2",
		"proj-1",
		24*time.Hour,
	)
	activeCtx.AddMessage("system", "Active conversation", 5)

	err = db.CreateConversationContext(activeCtx)
	if err != nil {
		t.Fatalf("Failed to create active conversation: %v", err)
	}

	// Delete expired conversations
	deletedCount, err := db.DeleteExpiredConversationContexts()
	if err != nil {
		t.Fatalf("Failed to delete expired conversations: %v", err)
	}

	if deletedCount != 1 {
		t.Errorf("Expected 1 deleted conversation, got %d", deletedCount)
	}

	// Verify expired is gone
	_, err = db.GetConversationContext(expiredCtx.SessionID)
	if err == nil {
		t.Error("Expected error when getting expired conversation, got nil")
	}

	// Verify active still exists
	retrieved, err := db.GetConversationContext(activeCtx.SessionID)
	if err != nil {
		t.Errorf("Active conversation should still exist: %v", err)
	}
	if retrieved.SessionID != activeCtx.SessionID {
		t.Errorf("SessionID mismatch: got %s, want %s", retrieved.SessionID, activeCtx.SessionID)
	}
}

func TestResetConversationMessages_NoSystemMessage(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create conversation without system message
	ctx := models.NewConversationContext(
		"session-no-system",
		"bead-1",
		"proj-1",
		24*time.Hour,
	)
	ctx.AddMessage("user", "First message", 5)
	ctx.AddMessage("assistant", "Response", 5)

	err = db.CreateConversationContext(ctx)
	if err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Reset without keeping system message
	err = db.ResetConversationMessages(ctx.SessionID, true)
	if err != nil {
		t.Fatalf("Failed to reset messages: %v", err)
	}

	retrieved, err := db.GetConversationContext(ctx.SessionID)
	if err != nil {
		t.Fatalf("Failed to get conversation after reset: %v", err)
	}

	// Should be completely empty since first message wasn't system
	if len(retrieved.Messages) != 0 {
		t.Errorf("Expected 0 messages after reset (no system message), got %d", len(retrieved.Messages))
	}
	if retrieved.TokenCount != 0 {
		t.Errorf("Expected token count 0 after reset, got %d", retrieved.TokenCount)
	}
}
