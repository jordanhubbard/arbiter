package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// ConversationSchemaVersion is the current schema version
const ConversationSchemaVersion SchemaVersion = "1.0"

// EntityTypeConversation identifies conversation entities
const EntityTypeConversation EntityType = "conversation"

// ConversationContext represents a persistent conversation session for a bead.
// It stores the complete message history across multiple agent dispatches,
// enabling iterative problem-solving and context retention.
type ConversationContext struct {
	SessionID  string       `json:"session_id" db:"session_id"`
	BeadID     string       `json:"bead_id" db:"bead_id"`
	ProjectID  string       `json:"project_id" db:"project_id"`
	Messages   []ChatMessage `json:"messages" db:"messages"` // Stored as JSON in SQLite
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at" db:"updated_at"`
	ExpiresAt  time.Time    `json:"expires_at" db:"expires_at"`
	TokenCount int          `json:"token_count" db:"token_count"` // Cumulative token usage
	Metadata   map[string]string `json:"metadata" db:"metadata"` // Stored as JSON in SQLite

	// Entity versioning
	EntityMetadata `json:"entity_metadata,omitempty"`
}

// ChatMessage represents a single message in the conversation history.
// This extends the basic provider.ChatMessage with additional fields needed
// for conversation tracking.
type ChatMessage struct {
	Role       string    `json:"role"`       // "system", "user", "assistant"
	Content    string    `json:"content"`    // Message content
	Timestamp  time.Time `json:"timestamp"`  // When this message was created
	TokenCount int       `json:"token_count,omitempty"` // Tokens in this message (0 if not counted)
}

// NewConversationContext creates a new conversation context with default values
func NewConversationContext(sessionID, beadID, projectID string, expirationDuration time.Duration) *ConversationContext {
	now := time.Now()
	return &ConversationContext{
		SessionID:      sessionID,
		BeadID:         beadID,
		ProjectID:      projectID,
		Messages:       []ChatMessage{},
		CreatedAt:      now,
		UpdatedAt:      now,
		ExpiresAt:      now.Add(expirationDuration),
		TokenCount:     0,
		Metadata:       make(map[string]string),
		EntityMetadata: NewEntityMetadata(ConversationSchemaVersion),
	}
}

// IsExpired checks if the conversation session has expired
func (c *ConversationContext) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// AddMessage appends a new message to the conversation history
func (c *ConversationContext) AddMessage(role, content string, tokenCount int) {
	c.Messages = append(c.Messages, ChatMessage{
		Role:       role,
		Content:    content,
		Timestamp:  time.Now(),
		TokenCount: tokenCount,
	})
	c.TokenCount += tokenCount
	c.UpdatedAt = time.Now()
}

// TruncateMessages implements a sliding window strategy to keep conversation
// within token limits. Keeps the system message and the most recent messages
// that fit within maxTokens.
func (c *ConversationContext) TruncateMessages(maxTokens int) {
	if len(c.Messages) == 0 || c.TokenCount <= maxTokens {
		return
	}

	// Always keep the first message (usually system message)
	systemMsg := c.Messages[0]
	systemTokens := c.estimateTokens(systemMsg.Content)

	// Find how many recent messages fit
	keepMessages := []ChatMessage{systemMsg}
	totalTokens := systemTokens

	// Work backwards from the most recent messages
	for i := len(c.Messages) - 1; i > 0; i-- {
		msgTokens := c.estimateTokens(c.Messages[i].Content)
		if totalTokens+msgTokens > maxTokens {
			// Add truncation notice if we're dropping messages
			truncatedCount := i
			noticeMsg := ChatMessage{
				Role:      "system",
				Content:   fmt.Sprintf("[Note: %d older messages truncated to stay within token limit]", truncatedCount),
				Timestamp: time.Now(),
			}
			keepMessages = append([]ChatMessage{systemMsg, noticeMsg}, keepMessages[1:]...)
			break
		}
		totalTokens += msgTokens
		keepMessages = append([]ChatMessage{c.Messages[i]}, keepMessages...)
	}

	c.Messages = keepMessages
	c.TokenCount = totalTokens
	c.UpdatedAt = time.Now()
}

// estimateTokens provides a rough token count estimate.
// Uses the approximation: 1 token ≈ 4 characters
func (c *ConversationContext) estimateTokens(text string) int {
	return len(text) / 4
}

// MessagesJSON returns messages as JSON bytes for database storage
func (c *ConversationContext) MessagesJSON() ([]byte, error) {
	if len(c.Messages) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(c.Messages)
}

// SetMessagesFromJSON parses JSON bytes into messages
func (c *ConversationContext) SetMessagesFromJSON(data []byte) error {
	if len(data) == 0 || string(data) == "[]" || string(data) == "null" {
		c.Messages = []ChatMessage{}
		return nil
	}
	return json.Unmarshal(data, &c.Messages)
}

// MetadataJSON returns metadata as JSON bytes for database storage
func (c *ConversationContext) MetadataJSON() ([]byte, error) {
	if len(c.Metadata) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(c.Metadata)
}

// SetMetadataFromJSON parses JSON bytes into metadata
func (c *ConversationContext) SetMetadataFromJSON(data []byte) error {
	if len(data) == 0 || string(data) == "{}" || string(data) == "null" {
		c.Metadata = make(map[string]string)
		return nil
	}
	return json.Unmarshal(data, &c.Metadata)
}

// VersionedEntity interface implementation

// GetEntityType returns the entity type for conversation contexts
func (c *ConversationContext) GetEntityType() EntityType {
	return EntityTypeConversation
}

// GetSchemaVersion returns the current schema version
func (c *ConversationContext) GetSchemaVersion() SchemaVersion {
	return c.EntityMetadata.SchemaVersion
}

// SetSchemaVersion updates the schema version
func (c *ConversationContext) SetSchemaVersion(version SchemaVersion) {
	c.EntityMetadata.SchemaVersion = version
}

// GetEntityMetadata returns the entity's metadata
func (c *ConversationContext) GetEntityMetadata() *EntityMetadata {
	return &c.EntityMetadata
}

// GetID returns the session ID as the unique identifier
func (c *ConversationContext) GetID() string {
	return c.SessionID
}
