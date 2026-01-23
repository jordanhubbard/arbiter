package logging

import (
	"container/ring"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	// MaxBufferSize is the maximum number of log entries to keep in memory
	MaxBufferSize = 10000

	// LogLevelDebug represents debug-level logs
	LogLevelDebug = "debug"
	// LogLevelInfo represents info-level logs
	LogLevelInfo = "info"
	// LogLevelWarn represents warning-level logs
	LogLevelWarn = "warn"
	// LogLevelError represents error-level logs
	LogLevelError = "error"
)

// LogEntry represents a single log entry
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Source    string                 `json:"source"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Manager handles log collection, buffering, and persistence
type Manager struct {
	mu       sync.RWMutex
	buffer   *ring.Ring
	db       *sql.DB
	handlers []func(LogEntry)
}

// NewManager creates a new logging manager
func NewManager(db *sql.DB) *Manager {
	m := &Manager{
		buffer:   ring.New(MaxBufferSize),
		db:       db,
		handlers: make([]func(LogEntry), 0),
	}

	// Initialize database schema
	if err := m.initSchema(); err != nil {
		log.Printf("Warning: Failed to initialize logging schema: %v", err)
	}

	return m
}

// initSchema creates the logs table if it doesn't exist
func (m *Manager) initSchema() error {
	if m.db == nil {
		return nil
	}

	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			level TEXT NOT NULL,
			source TEXT NOT NULL,
			message TEXT NOT NULL,
			metadata_json TEXT,
			agent_id TEXT,
			bead_id TEXT,
			project_id TEXT,
			provider_id TEXT,
			FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL,
			FOREIGN KEY (bead_id) REFERENCES beads(id) ON DELETE SET NULL,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL,
			FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE SET NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create logs table: %w", err)
	}

	// Create indexes for common queries
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level)",
		"CREATE INDEX IF NOT EXISTS idx_logs_source ON logs(source)",
		"CREATE INDEX IF NOT EXISTS idx_logs_agent_id ON logs(agent_id)",
		"CREATE INDEX IF NOT EXISTS idx_logs_bead_id ON logs(bead_id)",
	}

	for _, indexSQL := range indexes {
		if _, err := m.db.Exec(indexSQL); err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}

	return nil
}

// Log adds a log entry to the buffer and optionally persists it
func (m *Manager) Log(level, source, message string, metadata map[string]interface{}) {
	entry := LogEntry{
		ID:        fmt.Sprintf("log-%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Level:     level,
		Source:    source,
		Message:   message,
		Metadata:  metadata,
	}

	m.mu.Lock()
	m.buffer.Value = entry
	m.buffer = m.buffer.Next()
	m.mu.Unlock()

	// Notify handlers (for SSE streaming)
	for _, handler := range m.handlers {
		go handler(entry)
	}

	// Persist to database asynchronously
	go m.persistLog(entry)
}

// persistLog saves a log entry to the database
func (m *Manager) persistLog(entry LogEntry) {
	if m.db == nil {
		return
	}

	var metadataJSON *string
	if entry.Metadata != nil && len(entry.Metadata) > 0 {
		data, err := json.Marshal(entry.Metadata)
		if err == nil {
			jsonStr := string(data)
			metadataJSON = &jsonStr
		}
	}

	// Extract common entity IDs from metadata
	var agentID, beadID, projectID, providerID *string
	if entry.Metadata != nil {
		if val, ok := entry.Metadata["agent_id"].(string); ok && val != "" {
			agentID = &val
		}
		if val, ok := entry.Metadata["bead_id"].(string); ok && val != "" {
			beadID = &val
		}
		if val, ok := entry.Metadata["project_id"].(string); ok && val != "" {
			projectID = &val
		}
		if val, ok := entry.Metadata["provider_id"].(string); ok && val != "" {
			providerID = &val
		}
	}

	_, err := m.db.Exec(`
		INSERT INTO logs (id, timestamp, level, source, message, metadata_json, agent_id, bead_id, project_id, provider_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.Timestamp, entry.Level, entry.Source, entry.Message, metadataJSON, agentID, beadID, projectID, providerID)

	if err != nil {
		log.Printf("Failed to persist log entry: %v", err)
	}
}

// GetRecent returns the most recent log entries from the buffer
func (m *Manager) GetRecent(limit int, levelFilter, sourceFilter string) []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > MaxBufferSize {
		limit = 100
	}

	logs := make([]LogEntry, 0, limit)
	count := 0

	// Iterate backwards through the ring buffer
	m.buffer.Do(func(v interface{}) {
		if count >= limit {
			return
		}
		if v == nil {
			return
		}

		entry, ok := v.(LogEntry)
		if !ok {
			return
		}

		// Apply filters
		if levelFilter != "" && entry.Level != levelFilter {
			return
		}
		if sourceFilter != "" && entry.Source != sourceFilter {
			return
		}

		logs = append(logs, entry)
		count++
	})

	// Reverse to get newest first
	for i := 0; i < len(logs)/2; i++ {
		logs[i], logs[len(logs)-1-i] = logs[len(logs)-1-i], logs[i]
	}

	return logs
}

// Query returns log entries from the database based on filters
func (m *Manager) Query(limit int, levelFilter, sourceFilter string, since time.Time) ([]LogEntry, error) {
	if m.db == nil {
		return m.GetRecent(limit, levelFilter, sourceFilter), nil
	}

	query := `SELECT id, timestamp, level, source, message, metadata_json FROM logs WHERE 1=1`
	args := make([]interface{}, 0)

	if !since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, since)
	}
	if levelFilter != "" {
		query += " AND level = ?"
		args = append(args, levelFilter)
	}
	if sourceFilter != "" {
		query += " AND source = ?"
		args = append(args, sourceFilter)
	}

	query += " ORDER BY timestamp DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	logs := make([]LogEntry, 0)
	for rows.Next() {
		var entry LogEntry
		var metadataJSON *string

		err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.Level, &entry.Source, &entry.Message, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}

		if metadataJSON != nil && *metadataJSON != "" {
			if err := json.Unmarshal([]byte(*metadataJSON), &entry.Metadata); err != nil {
				log.Printf("Warning: Failed to unmarshal log metadata: %v", err)
			}
		}

		logs = append(logs, entry)
	}

	return logs, nil
}

// AddHandler registers a handler to be called for each new log entry (for SSE)
func (m *Manager) AddHandler(handler func(LogEntry)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// Debug logs a debug-level message
func (m *Manager) Debug(source, message string, metadata map[string]interface{}) {
	m.Log(LogLevelDebug, source, message, metadata)
}

// Info logs an info-level message
func (m *Manager) Info(source, message string, metadata map[string]interface{}) {
	m.Log(LogLevelInfo, source, message, metadata)
}

// Warn logs a warning-level message
func (m *Manager) Warn(source, message string, metadata map[string]interface{}) {
	m.Log(LogLevelWarn, source, message, metadata)
}

// Error logs an error-level message
func (m *Manager) Error(source, message string, metadata map[string]interface{}) {
	m.Log(LogLevelError, source, message, metadata)
}
