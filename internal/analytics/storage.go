package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// DatabaseStorage implements Storage using SQLite
type DatabaseStorage struct {
	db *sql.DB
}

// NewDatabaseStorage creates a new database-backed storage
func NewDatabaseStorage(db *sql.DB) (*DatabaseStorage, error) {
	storage := &DatabaseStorage{db: db}
	if err := storage.initSchema(); err != nil {
		return nil, err
	}
	return storage, nil
}

// initSchema creates the request_logs table
func (s *DatabaseStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS request_logs (
		id TEXT PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		user_id TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		provider_id TEXT,
		model_name TEXT,
		prompt_tokens INTEGER,
		completion_tokens INTEGER,
		total_tokens INTEGER,
		latency_ms INTEGER,
		status_code INTEGER,
		cost_usd REAL,
		error_message TEXT,
		request_body TEXT,
		response_body TEXT,
		metadata_json TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_request_logs_user_id ON request_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_request_logs_provider_id ON request_logs(provider_id);
	CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveLog persists a request log
func (s *DatabaseStorage) SaveLog(ctx context.Context, log *RequestLog) error {
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO request_logs (
			id, timestamp, user_id, method, path, provider_id, model_name,
			prompt_tokens, completion_tokens, total_tokens, latency_ms,
			status_code, cost_usd, error_message, request_body, response_body,
			metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		log.ID,
		log.Timestamp,
		log.UserID,
		log.Method,
		log.Path,
		log.ProviderID,
		log.ModelName,
		log.PromptTokens,
		log.CompletionTokens,
		log.TotalTokens,
		log.LatencyMs,
		log.StatusCode,
		log.CostUSD,
		log.ErrorMessage,
		log.RequestBody,
		log.ResponseBody,
		string(metadataJSON),
	)

	return err
}

// GetLogs retrieves logs with filtering
func (s *DatabaseStorage) GetLogs(ctx context.Context, filter *LogFilter) ([]*RequestLog, error) {
	query := `
		SELECT 
			id, timestamp, user_id, method, path, provider_id, model_name,
			prompt_tokens, completion_tokens, total_tokens, latency_ms,
			status_code, cost_usd, error_message, request_body, response_body,
			metadata_json
		FROM request_logs
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}

	if filter.ProviderID != "" {
		query += " AND provider_id = ?"
		args = append(args, filter.ProviderID)
	}

	if !filter.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.StartTime)
	}

	if !filter.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.EndTime)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*RequestLog
	for rows.Next() {
		log := &RequestLog{}
		var metadataJSON string

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.UserID,
			&log.Method,
			&log.Path,
			&log.ProviderID,
			&log.ModelName,
			&log.PromptTokens,
			&log.CompletionTokens,
			&log.TotalTokens,
			&log.LatencyMs,
			&log.StatusCode,
			&log.CostUSD,
			&log.ErrorMessage,
			&log.RequestBody,
			&log.ResponseBody,
			&metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != "" {
			json.Unmarshal([]byte(metadataJSON), &log.Metadata)
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetLogStats computes aggregate statistics
func (s *DatabaseStorage) GetLogStats(ctx context.Context, filter *LogFilter) (*LogStats, error) {
	baseQuery := `
		SELECT 
			COUNT(*) as total_requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(cost_usd), 0) as total_cost,
			COALESCE(AVG(latency_ms), 0) as avg_latency,
			COALESCE(SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END), 0) as error_count
		FROM request_logs
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.UserID != "" {
		baseQuery += " AND user_id = ?"
		args = append(args, filter.UserID)
	}

	if filter.ProviderID != "" {
		baseQuery += " AND provider_id = ?"
		args = append(args, filter.ProviderID)
	}

	if !filter.StartTime.IsZero() {
		baseQuery += " AND timestamp >= ?"
		args = append(args, filter.StartTime)
	}

	if !filter.EndTime.IsZero() {
		baseQuery += " AND timestamp <= ?"
		args = append(args, filter.EndTime)
	}

	stats := &LogStats{
		RequestsByUser:   make(map[string]int64),
		RequestsByProvider: make(map[string]int64),
		CostByProvider:   make(map[string]float64),
		CostByUser:       make(map[string]float64),
	}

	var errorCount int64
	row := s.db.QueryRowContext(ctx, baseQuery, args...)
	err := row.Scan(
		&stats.TotalRequests,
		&stats.TotalTokens,
		&stats.TotalCostUSD,
		&stats.AvgLatencyMs,
		&errorCount,
	)
	if err != nil {
		return nil, err
	}

	if stats.TotalRequests > 0 {
		stats.ErrorRate = float64(errorCount) / float64(stats.TotalRequests)
	}

	// Get per-user stats (requests and costs)
	userQuery := fmt.Sprintf(`
		SELECT user_id, COUNT(*) as count, COALESCE(SUM(cost_usd), 0) as cost
		FROM request_logs
		WHERE 1=1 %s AND user_id IS NOT NULL AND user_id != ''
		GROUP BY user_id
	`, buildWhereClause(filter))
	
	rows, err := s.db.QueryContext(ctx, userQuery, buildWhereArgs(filter)...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var userID string
			var count int64
			var cost float64
			if err := rows.Scan(&userID, &count, &cost); err == nil {
				stats.RequestsByUser[userID] = count
				stats.CostByUser[userID] = cost
			}
		}
	}

	// Get per-provider stats
	providerQuery := fmt.Sprintf(`
		SELECT provider_id, COUNT(*) as count, COALESCE(SUM(cost_usd), 0) as cost
		FROM request_logs
		WHERE 1=1 %s AND provider_id IS NOT NULL AND provider_id != ''
		GROUP BY provider_id
	`, buildWhereClause(filter))
	
	rows, err = s.db.QueryContext(ctx, providerQuery, buildWhereArgs(filter)...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var providerID string
			var count int64
			var cost float64
			if err := rows.Scan(&providerID, &count, &cost); err == nil {
				stats.RequestsByProvider[providerID] = count
				stats.CostByProvider[providerID] = cost
			}
		}
	}

	return stats, nil
}

// DeleteOldLogs removes logs older than the specified time
func (s *DatabaseStorage) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM request_logs WHERE timestamp < ?", before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Helper functions for building queries
func buildWhereClause(filter *LogFilter) string {
	where := ""
	if filter.UserID != "" {
		where += " AND user_id = ?"
	}
	if filter.ProviderID != "" {
		where += " AND provider_id = ?"
	}
	if !filter.StartTime.IsZero() {
		where += " AND timestamp >= ?"
	}
	if !filter.EndTime.IsZero() {
		where += " AND timestamp <= ?"
	}
	return where
}

func buildWhereArgs(filter *LogFilter) []interface{} {
	args := []interface{}{}
	if filter.UserID != "" {
		args = append(args, filter.UserID)
	}
	if filter.ProviderID != "" {
		args = append(args, filter.ProviderID)
	}
	if !filter.StartTime.IsZero() {
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		args = append(args, filter.EndTime)
	}
	return args
}
