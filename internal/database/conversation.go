package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

// CreateConversationContext inserts a new conversation context
func (d *Database) CreateConversationContext(ctx *models.ConversationContext) error {
	messagesJSON, err := ctx.MessagesJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	metadataJSON, err := ctx.MetadataJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO conversation_contexts (
			session_id, bead_id, project_id, messages,
			created_at, updated_at, expires_at, token_count, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = d.db.Exec(query,
		ctx.SessionID,
		ctx.BeadID,
		ctx.ProjectID,
		messagesJSON,
		ctx.CreatedAt,
		ctx.UpdatedAt,
		ctx.ExpiresAt,
		ctx.TokenCount,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create conversation context: %w", err)
	}
	return nil
}

// GetConversationContext retrieves a conversation context by session ID
func (d *Database) GetConversationContext(sessionID string) (*models.ConversationContext, error) {
	query := `
		SELECT session_id, bead_id, project_id, messages,
			   created_at, updated_at, expires_at, token_count, metadata
		FROM conversation_contexts
		WHERE session_id = ?
	`

	ctx := &models.ConversationContext{}
	var messagesJSON, metadataJSON []byte

	err := d.db.QueryRow(query, sessionID).Scan(
		&ctx.SessionID,
		&ctx.BeadID,
		&ctx.ProjectID,
		&messagesJSON,
		&ctx.CreatedAt,
		&ctx.UpdatedAt,
		&ctx.ExpiresAt,
		&ctx.TokenCount,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conversation context not found: %s", sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation context: %w", err)
	}

	// Unmarshal JSON fields
	if err := ctx.SetMessagesFromJSON(messagesJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}
	if err := ctx.SetMetadataFromJSON(metadataJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return ctx, nil
}

// GetConversationContextByBeadID retrieves the conversation context for a specific bead
func (d *Database) GetConversationContextByBeadID(beadID string) (*models.ConversationContext, error) {
	query := `
		SELECT session_id, bead_id, project_id, messages,
			   created_at, updated_at, expires_at, token_count, metadata
		FROM conversation_contexts
		WHERE bead_id = ?
		ORDER BY updated_at DESC
		LIMIT 1
	`

	ctx := &models.ConversationContext{}
	var messagesJSON, metadataJSON []byte

	err := d.db.QueryRow(query, beadID).Scan(
		&ctx.SessionID,
		&ctx.BeadID,
		&ctx.ProjectID,
		&messagesJSON,
		&ctx.CreatedAt,
		&ctx.UpdatedAt,
		&ctx.ExpiresAt,
		&ctx.TokenCount,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conversation context not found for bead: %s", beadID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation context: %w", err)
	}

	// Unmarshal JSON fields
	if err := ctx.SetMessagesFromJSON(messagesJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}
	if err := ctx.SetMetadataFromJSON(metadataJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return ctx, nil
}

// UpdateConversationContext updates an existing conversation context
func (d *Database) UpdateConversationContext(ctx *models.ConversationContext) error {
	messagesJSON, err := ctx.MessagesJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	metadataJSON, err := ctx.MetadataJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE conversation_contexts
		SET messages = ?, updated_at = ?, token_count = ?, metadata = ?
		WHERE session_id = ?
	`

	result, err := d.db.Exec(query,
		messagesJSON,
		ctx.UpdatedAt,
		ctx.TokenCount,
		metadataJSON,
		ctx.SessionID,
	)

	if err != nil {
		return fmt.Errorf("failed to update conversation context: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("conversation context not found: %s", ctx.SessionID)
	}

	return nil
}

// DeleteConversationContext deletes a conversation context
func (d *Database) DeleteConversationContext(sessionID string) error {
	query := `
		DELETE FROM conversation_contexts
		WHERE session_id = ?
	`

	result, err := d.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation context: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("conversation context not found: %s", sessionID)
	}

	return nil
}

// DeleteExpiredConversationContexts removes all expired conversation contexts
// This should be called periodically (e.g., hourly cron job)
func (d *Database) DeleteExpiredConversationContexts() (int64, error) {
	query := `
		DELETE FROM conversation_contexts
		WHERE expires_at < ?
	`

	result, err := d.db.Exec(query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired conversations: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rows, nil
}

// ListConversationContextsByProject retrieves all conversation contexts for a project
func (d *Database) ListConversationContextsByProject(projectID string, limit int) ([]*models.ConversationContext, error) {
	query := `
		SELECT session_id, bead_id, project_id, messages,
			   created_at, updated_at, expires_at, token_count, metadata
		FROM conversation_contexts
		WHERE project_id = ?
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := d.db.Query(query, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversation contexts: %w", err)
	}
	defer rows.Close()

	var contexts []*models.ConversationContext
	for rows.Next() {
		ctx := &models.ConversationContext{}
		var messagesJSON, metadataJSON []byte

		err := rows.Scan(
			&ctx.SessionID,
			&ctx.BeadID,
			&ctx.ProjectID,
			&messagesJSON,
			&ctx.CreatedAt,
			&ctx.UpdatedAt,
			&ctx.ExpiresAt,
			&ctx.TokenCount,
			&metadataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation context: %w", err)
		}

		// Unmarshal JSON fields
		if err := ctx.SetMessagesFromJSON(messagesJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
		}
		if err := ctx.SetMetadataFromJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		contexts = append(contexts, ctx)
	}

	return contexts, nil
}

// ResetConversationMessages clears the conversation history but keeps the session
// Optionally keeps the first system message
func (d *Database) ResetConversationMessages(sessionID string, keepSystemMessage bool) error {
	// First get the current context to check for system message
	ctx, err := d.GetConversationContext(sessionID)
	if err != nil {
		return err
	}

	// Prepare new messages array
	var newMessages []models.ChatMessage
	if keepSystemMessage && len(ctx.Messages) > 0 && ctx.Messages[0].Role == "system" {
		newMessages = []models.ChatMessage{ctx.Messages[0]}
	}

	// Update the context
	ctx.Messages = newMessages
	ctx.TokenCount = 0
	for _, msg := range newMessages {
		ctx.TokenCount += msg.TokenCount
	}
	ctx.UpdatedAt = time.Now()

	return d.UpdateConversationContext(ctx)
}
