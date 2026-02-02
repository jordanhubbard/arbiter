package database

import "database/sql"

// migratePatterns creates tables for pattern analysis and optimizations
func migratePatterns(db *sql.DB) error {
	schema := `
	-- Usage patterns table (optional - for caching pattern analysis results)
	CREATE TABLE IF NOT EXISTS usage_patterns (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		group_key TEXT NOT NULL,
		request_count INTEGER NOT NULL,
		total_cost REAL NOT NULL,
		avg_cost REAL NOT NULL,
		total_tokens INTEGER NOT NULL,
		avg_latency REAL NOT NULL,
		error_rate REAL NOT NULL,
		first_seen DATETIME NOT NULL,
		last_seen DATETIME NOT NULL,
		analyzed_at DATETIME NOT NULL,
		metadata_json TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_usage_patterns_type ON usage_patterns(type);
	CREATE INDEX IF NOT EXISTS idx_usage_patterns_group_key ON usage_patterns(group_key);
	CREATE INDEX IF NOT EXISTS idx_usage_patterns_total_cost ON usage_patterns(total_cost DESC);
	CREATE INDEX IF NOT EXISTS idx_usage_patterns_analyzed_at ON usage_patterns(analyzed_at);

	-- Optimizations table (track applied optimizations)
	CREATE TABLE IF NOT EXISTS optimizations (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		pattern_id TEXT,
		recommendation TEXT NOT NULL,
		projected_savings_usd REAL NOT NULL,
		actual_savings_usd REAL,
		applied_at DATETIME,
		applied_by TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		metadata_json TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (pattern_id) REFERENCES usage_patterns(id)
	);

	CREATE INDEX IF NOT EXISTS idx_optimizations_status ON optimizations(status);
	CREATE INDEX IF NOT EXISTS idx_optimizations_type ON optimizations(type);
	CREATE INDEX IF NOT EXISTS idx_optimizations_pattern_id ON optimizations(pattern_id);
	CREATE INDEX IF NOT EXISTS idx_optimizations_created_at ON optimizations(created_at);
	`

	_, err := db.Exec(schema)
	return err
}
