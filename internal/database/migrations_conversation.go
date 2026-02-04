package database

import (
	"log"
)

// migrateConversations creates the conversation_contexts table for
// storing multi-turn conversation sessions
func (d *Database) migrateConversations() error {
	// Conversation contexts table
	// SQLite uses TEXT type for JSON storage (JSONB is PostgreSQL-specific)
	conversationSchema := `
	CREATE TABLE IF NOT EXISTS conversation_contexts (
		session_id TEXT PRIMARY KEY,
		bead_id TEXT NOT NULL,
		project_id TEXT NOT NULL,
		messages TEXT NOT NULL DEFAULT '[]',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL,
		token_count INTEGER NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}'
	);

	CREATE INDEX IF NOT EXISTS idx_conversation_bead ON conversation_contexts(bead_id);
	CREATE INDEX IF NOT EXISTS idx_conversation_expires ON conversation_contexts(expires_at);
	CREATE INDEX IF NOT EXISTS idx_conversation_updated ON conversation_contexts(updated_at);
	CREATE INDEX IF NOT EXISTS idx_conversation_project ON conversation_contexts(project_id);
	`

	if _, err := d.db.Exec(conversationSchema); err != nil {
		return err
	}

	log.Println("Conversation contexts table migrated successfully")
	return nil
}
