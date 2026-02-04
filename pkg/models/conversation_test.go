package models

import (
	"testing"
	"time"
)

func TestNewConversationContext(t *testing.T) {
	sessionID := "test-session-123"
	beadID := "bead-456"
	projectID := "proj-789"
	duration := 24 * time.Hour

	ctx := NewConversationContext(sessionID, beadID, projectID, duration)

	if ctx.SessionID != sessionID {
		t.Errorf("SessionID mismatch: got %s, want %s", ctx.SessionID, sessionID)
	}
	if ctx.BeadID != beadID {
		t.Errorf("BeadID mismatch: got %s, want %s", ctx.BeadID, beadID)
	}
	if ctx.ProjectID != projectID {
		t.Errorf("ProjectID mismatch: got %s, want %s", ctx.ProjectID, projectID)
	}
	if len(ctx.Messages) != 0 {
		t.Errorf("Expected empty messages, got %d messages", len(ctx.Messages))
	}
	if ctx.TokenCount != 0 {
		t.Errorf("Expected token count 0, got %d", ctx.TokenCount)
	}
	if ctx.Metadata == nil {
		t.Error("Expected metadata to be initialized")
	}
	if ctx.GetSchemaVersion() != ConversationSchemaVersion {
		t.Errorf("Schema version mismatch: got %s, want %s", ctx.GetSchemaVersion(), ConversationSchemaVersion)
	}
}

func TestConversationContext_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "just expired",
			expiresAt: time.Now().Add(-1 * time.Second),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ConversationContext{
				ExpiresAt: tt.expiresAt,
			}
			if got := ctx.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConversationContext_AddMessage(t *testing.T) {
	ctx := NewConversationContext("session-1", "bead-1", "proj-1", 24*time.Hour)
	initialUpdateTime := ctx.UpdatedAt

	// Add first message
	ctx.AddMessage("system", "You are a helpful assistant", 10)

	if len(ctx.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(ctx.Messages))
	}
	if ctx.Messages[0].Role != "system" {
		t.Errorf("Role mismatch: got %s, want system", ctx.Messages[0].Role)
	}
	if ctx.Messages[0].Content != "You are a helpful assistant" {
		t.Errorf("Content mismatch: got %s", ctx.Messages[0].Content)
	}
	if ctx.Messages[0].TokenCount != 10 {
		t.Errorf("TokenCount mismatch: got %d, want 10", ctx.Messages[0].TokenCount)
	}
	if ctx.TokenCount != 10 {
		t.Errorf("Total TokenCount mismatch: got %d, want 10", ctx.TokenCount)
	}
	if !ctx.UpdatedAt.After(initialUpdateTime) {
		t.Error("UpdatedAt should be updated")
	}

	// Add second message
	ctx.AddMessage("user", "Hello", 5)

	if len(ctx.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(ctx.Messages))
	}
	if ctx.TokenCount != 15 {
		t.Errorf("Total TokenCount mismatch: got %d, want 15", ctx.TokenCount)
	}
}

func TestConversationContext_TruncateMessages(t *testing.T) {
	ctx := NewConversationContext("session-1", "bead-1", "proj-1", 24*time.Hour)

	// Add system message (40 chars = ~10 tokens)
	ctx.AddMessage("system", "You are a helpful AI assistant here", 10)

	// Add multiple user/assistant messages
	for i := 0; i < 10; i++ {
		// Each message is 40 chars = ~10 tokens
		ctx.AddMessage("user", "This is a test message for truncation", 10)
		ctx.AddMessage("assistant", "This is a response to your message X", 10)
	}

	// Total: 1 system + 20 user/assistant = 21 messages, 210 tokens
	if len(ctx.Messages) != 21 {
		t.Errorf("Expected 21 messages before truncation, got %d", len(ctx.Messages))
	}

	// Truncate to 50 tokens (system + ~4 recent messages)
	ctx.TruncateMessages(50)

	// Should keep system message + truncation notice + recent messages
	if len(ctx.Messages) < 3 {
		t.Errorf("Expected at least 3 messages after truncation, got %d", len(ctx.Messages))
	}

	// First message should still be system
	if ctx.Messages[0].Role != "system" {
		t.Errorf("First message should be system, got %s", ctx.Messages[0].Role)
	}

	// Should have a truncation notice
	hasNotice := false
	for _, msg := range ctx.Messages {
		if msg.Role == "system" && len(msg.Content) > 20 && msg.Content[:6] == "[Note:" {
			hasNotice = true
			break
		}
	}
	if !hasNotice {
		t.Error("Expected truncation notice in messages")
	}

	// Token count should be under limit
	if ctx.TokenCount > 50 {
		t.Errorf("Token count should be under 50, got %d", ctx.TokenCount)
	}
}

func TestConversationContext_TruncateMessages_NoTruncationNeeded(t *testing.T) {
	ctx := NewConversationContext("session-1", "bead-1", "proj-1", 24*time.Hour)

	ctx.AddMessage("system", "System", 2)
	ctx.AddMessage("user", "Hello", 2)

	initialMessageCount := len(ctx.Messages)
	ctx.TruncateMessages(100)

	// Should not truncate when under limit
	if len(ctx.Messages) != initialMessageCount {
		t.Errorf("Messages should not be truncated, expected %d, got %d", initialMessageCount, len(ctx.Messages))
	}
}

func TestConversationContext_MessagesJSON(t *testing.T) {
	ctx := NewConversationContext("session-1", "bead-1", "proj-1", 24*time.Hour)
	ctx.AddMessage("system", "System message", 5)
	ctx.AddMessage("user", "User message", 5)

	// Test marshaling
	jsonData, err := ctx.MessagesJSON()
	if err != nil {
		t.Fatalf("Failed to marshal messages: %v", err)
	}

	// Test unmarshaling
	newCtx := &ConversationContext{}
	if err := newCtx.SetMessagesFromJSON(jsonData); err != nil {
		t.Fatalf("Failed to unmarshal messages: %v", err)
	}

	if len(newCtx.Messages) != 2 {
		t.Errorf("Expected 2 messages after unmarshal, got %d", len(newCtx.Messages))
	}
	if newCtx.Messages[0].Role != "system" {
		t.Errorf("First message role mismatch: got %s, want system", newCtx.Messages[0].Role)
	}
	if newCtx.Messages[1].Content != "User message" {
		t.Errorf("Second message content mismatch: got %s", newCtx.Messages[1].Content)
	}
}

func TestConversationContext_MetadataJSON(t *testing.T) {
	ctx := NewConversationContext("session-1", "bead-1", "proj-1", 24*time.Hour)
	ctx.Metadata["agent_id"] = "agent-123"
	ctx.Metadata["provider_id"] = "provider-456"

	// Test marshaling
	jsonData, err := ctx.MetadataJSON()
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}

	// Test unmarshaling
	newCtx := &ConversationContext{}
	if err := newCtx.SetMetadataFromJSON(jsonData); err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}

	if newCtx.Metadata["agent_id"] != "agent-123" {
		t.Errorf("agent_id mismatch: got %s, want agent-123", newCtx.Metadata["agent_id"])
	}
	if newCtx.Metadata["provider_id"] != "provider-456" {
		t.Errorf("provider_id mismatch: got %s, want provider-456", newCtx.Metadata["provider_id"])
	}
}

func TestConversationContext_VersionedEntityInterface(t *testing.T) {
	ctx := NewConversationContext("session-1", "bead-1", "proj-1", 24*time.Hour)

	// Test interface implementation
	var _ VersionedEntity = ctx

	if ctx.GetEntityType() != EntityTypeConversation {
		t.Errorf("EntityType mismatch: got %s, want %s", ctx.GetEntityType(), EntityTypeConversation)
	}

	if ctx.GetID() != "session-1" {
		t.Errorf("ID mismatch: got %s, want session-1", ctx.GetID())
	}

	newVersion := SchemaVersion("2.0")
	ctx.SetSchemaVersion(newVersion)
	if ctx.GetSchemaVersion() != newVersion {
		t.Errorf("Schema version mismatch after set: got %s, want %s", ctx.GetSchemaVersion(), newVersion)
	}

	meta := ctx.GetEntityMetadata()
	if meta == nil {
		t.Error("EntityMetadata should not be nil")
	}
}
