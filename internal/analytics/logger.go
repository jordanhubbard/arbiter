package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// RequestLog represents a logged API request
type RequestLog struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	UserID          string    `json:"user_id"`
	Method          string    `json:"method"`
	Path            string    `json:"path"`
	ProviderID      string    `json:"provider_id"`
	ModelName       string    `json:"model_name"`
	PromptTokens    int64     `json:"prompt_tokens"`
	CompletionTokens int64    `json:"completion_tokens"`
	TotalTokens     int64     `json:"total_tokens"`
	LatencyMs       int64     `json:"latency_ms"`
	StatusCode      int       `json:"status_code"`
	CostUSD         float64   `json:"cost_usd"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	RequestBody     string    `json:"request_body,omitempty"`      // Redacted if privacy enabled
	ResponseBody    string    `json:"response_body,omitempty"`     // Redacted if privacy enabled
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// PrivacyConfig controls what data is logged
type PrivacyConfig struct {
	LogRequestBodies  bool     // Log full request bodies
	LogResponseBodies bool     // Log full response bodies
	RedactPatterns    []string // Regex patterns to redact (emails, tokens, etc.)
	MaxBodyLength     int      // Max length of logged bodies (0 = unlimited)
}

// DefaultPrivacyConfig provides GDPR-compliant defaults
func DefaultPrivacyConfig() *PrivacyConfig {
	return &PrivacyConfig{
		LogRequestBodies:  false, // Don't log by default (privacy-first)
		LogResponseBodies: false,
		RedactPatterns: []string{
			// Email addresses
			`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
			// API keys/tokens (common patterns)
			`(?i)(api[_-]?key|token|secret|password)["\s:=]+([a-zA-Z0-9_-]{20,})`,
			// Credit card numbers
			`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`,
			// SSN
			`\b\d{3}-\d{2}-\d{4}\b`,
		},
		MaxBodyLength: 10000, // 10KB max
	}
}

// Logger handles request/response logging with privacy controls
type Logger struct {
	storage Storage
	privacy *PrivacyConfig
}

// Storage interface for persisting logs
type Storage interface {
	SaveLog(ctx context.Context, log *RequestLog) error
	GetLogs(ctx context.Context, filter *LogFilter) ([]*RequestLog, error)
	GetLogStats(ctx context.Context, filter *LogFilter) (*LogStats, error)
	DeleteOldLogs(ctx context.Context, before time.Time) (int64, error)
}

// LogFilter for querying logs
type LogFilter struct {
	UserID     string
	ProviderID string
	StartTime  time.Time
	EndTime    time.Time
	Limit      int
	Offset     int
}

// LogStats provides aggregate statistics
type LogStats struct {
	TotalRequests    int64   `json:"total_requests"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalCostUSD     float64 `json:"total_cost_usd"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	ErrorRate        float64 `json:"error_rate"`
	RequestsByUser   map[string]int64 `json:"requests_by_user"`
	RequestsByProvider map[string]int64 `json:"requests_by_provider"`
	CostByProvider   map[string]float64 `json:"cost_by_provider"`
	CostByUser       map[string]float64 `json:"cost_by_user"` // Cost tracking per user
}

// NewLogger creates a new request logger
func NewLogger(storage Storage, privacy *PrivacyConfig) *Logger {
	if privacy == nil {
		privacy = DefaultPrivacyConfig()
	}
	return &Logger{
		storage: storage,
		privacy: privacy,
	}
}

// LogRequest logs an API request with privacy controls
func (l *Logger) LogRequest(ctx context.Context, log *RequestLog) error {
	// Apply privacy filters
	if !l.privacy.LogRequestBodies {
		log.RequestBody = "" // Don't log request bodies
	} else if l.privacy.MaxBodyLength > 0 && len(log.RequestBody) > l.privacy.MaxBodyLength {
		log.RequestBody = log.RequestBody[:l.privacy.MaxBodyLength] + "... [truncated]"
	}

	if !l.privacy.LogResponseBodies {
		log.ResponseBody = "" // Don't log response bodies
	} else if l.privacy.MaxBodyLength > 0 && len(log.ResponseBody) > l.privacy.MaxBodyLength {
		log.ResponseBody = log.ResponseBody[:l.privacy.MaxBodyLength] + "... [truncated]"
	}

	// Redact sensitive patterns
	if log.RequestBody != "" {
		log.RequestBody = l.redactSensitiveData(log.RequestBody)
	}
	if log.ResponseBody != "" {
		log.ResponseBody = l.redactSensitiveData(log.ResponseBody)
	}

	// Generate ID if not provided
	if log.ID == "" {
		log.ID = generateLogID()
	}

	// Set timestamp if not provided
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	return l.storage.SaveLog(ctx, log)
}

// GetLogs retrieves logs with filtering
func (l *Logger) GetLogs(ctx context.Context, filter *LogFilter) ([]*RequestLog, error) {
	return l.storage.GetLogs(ctx, filter)
}

// GetStats retrieves aggregate statistics
func (l *Logger) GetStats(ctx context.Context, filter *LogFilter) (*LogStats, error) {
	return l.storage.GetLogStats(ctx, filter)
}

// PurgeLogs deletes logs older than the specified time
func (l *Logger) PurgeLogs(ctx context.Context, before time.Time) (int64, error) {
	return l.storage.DeleteOldLogs(ctx, before)
}

// redactSensitiveData applies privacy redaction patterns
func (l *Logger) redactSensitiveData(data string) string {
	for _, pattern := range l.privacy.RedactPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		data = re.ReplaceAllString(data, "[REDACTED]")
	}
	return data
}

// generateLogID creates a unique log ID
func generateLogID() string {
	return fmt.Sprintf("log-%d", time.Now().UnixNano())
}

// CalculateCost computes cost based on token usage and provider pricing
func CalculateCost(providerCostPerMToken float64, totalTokens int64) float64 {
	if providerCostPerMToken <= 0 || totalTokens <= 0 {
		return 0.0
	}
	// Cost = (tokens / 1,000,000) * cost_per_million_tokens
	return (float64(totalTokens) / 1000000.0) * providerCostPerMToken
}

// SanitizeForLogging removes sensitive data from arbitrary JSON
func SanitizeForLogging(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "[serialization error]"
	}

	str := string(bytes)
	
	// Redact common sensitive fields
	sensitiveKeys := []string{"password", "api_key", "token", "secret", "authorization"}
	for _, key := range sensitiveKeys {
		// Match "key":"value" or 'key':'value' patterns
		re := regexp.MustCompile(fmt.Sprintf(`(?i)"%s"\s*:\s*"[^"]*"`, key))
		str = re.ReplaceAllString(str, fmt.Sprintf(`"%s":"[REDACTED]"`, key))
	}

	return str
}
