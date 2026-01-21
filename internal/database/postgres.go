package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// NewPostgres creates a PostgreSQL database connection.
func NewPostgres(dsn string) (*Database, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	d := &Database{
		db:         db,
		dbType:     "postgres",
		supportsHA: true,
	}

	// Initialize schema
	if err := d.initSchemaPostgres(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Run migrations
	if err := d.migrateProviderOwnership(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate provider ownership: %w", err)
	}

	if err := d.migrateProviderRouting(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate provider routing: %w", err)
	}

	return d, nil
}

// initSchemaPostgres creates PostgreSQL-specific tables.
func (d *Database) initSchemaPostgres() error {
	schema := `
	-- Global configuration key-value store
	CREATE TABLE IF NOT EXISTS config_kv (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Distributed locks table for HA
	CREATE TABLE IF NOT EXISTS distributed_locks (
		lock_name TEXT PRIMARY KEY,
		instance_id TEXT NOT NULL,
		acquired_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL,
		heartbeat_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Instance registry for tracking active instances
	CREATE TABLE IF NOT EXISTS instances (
		instance_id TEXT PRIMARY KEY,
		hostname TEXT NOT NULL,
		started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_heartbeat TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		status TEXT NOT NULL DEFAULT 'active',
		metadata JSONB
	);

	-- Global providers (shared across all projects)
	CREATE TABLE IF NOT EXISTS providers (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		endpoint TEXT NOT NULL,
		model TEXT,
		configured_model TEXT,
		selected_model TEXT,
		selection_reason TEXT,
		model_score REAL,
		selected_gpu TEXT,
		gpu_constraints_json TEXT,
		description TEXT,
		requires_key BOOLEAN NOT NULL DEFAULT false,
		key_id TEXT,
		owner_id TEXT,
		is_shared BOOLEAN NOT NULL DEFAULT true,
		status TEXT NOT NULL DEFAULT 'active',
		last_heartbeat_at TIMESTAMP,
		last_heartbeat_latency_ms INTEGER,
		last_heartbeat_error TEXT,
		metrics_json TEXT,
		schema_version TEXT NOT NULL DEFAULT '1.0',
		attributes_json TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		cost_per_mtoken REAL,
		context_window INTEGER,
		supports_function BOOLEAN DEFAULT false,
		supports_vision BOOLEAN DEFAULT false,
		supports_streaming BOOLEAN DEFAULT false,
		tags TEXT[]
	);

	-- Request logs for analytics
	CREATE TABLE IF NOT EXISTS request_logs (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		user_id TEXT,
		provider_id TEXT,
		model TEXT,
		endpoint TEXT,
		method TEXT,
		status_code INTEGER,
		latency_ms INTEGER,
		prompt_tokens INTEGER,
		completion_tokens INTEGER,
		total_tokens INTEGER,
		cost_usd REAL,
		error_message TEXT,
		request_body_hash TEXT,
		ip_address TEXT
	);

	-- Create indexes for performance
	CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_request_logs_user_id ON request_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_request_logs_provider_id ON request_logs(provider_id);
	CREATE INDEX IF NOT EXISTS idx_distributed_locks_expires_at ON distributed_locks(expires_at);
	CREATE INDEX IF NOT EXISTS idx_instances_last_heartbeat ON instances(last_heartbeat);
	`

	_, err := d.db.Exec(schema)
	return err
}
