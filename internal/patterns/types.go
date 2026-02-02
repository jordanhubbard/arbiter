package patterns

import "time"

// UsagePattern represents a detected pattern in API usage
type UsagePattern struct {
	ID               string    `json:"id"`
	Type             string    `json:"type"` // "provider-model", "user", "cost-band", "temporal", "latency"
	GroupKey         string    `json:"group_key"`
	RequestCount     int64     `json:"request_count"`
	TotalCost        float64   `json:"total_cost"`
	AvgCost          float64   `json:"avg_cost"`
	TotalTokens      int64     `json:"total_tokens"`
	AvgLatency       float64   `json:"avg_latency"`
	ErrorRate        float64   `json:"error_rate"`
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
	RequestFrequency float64   `json:"request_frequency"` // requests per day
	CostTrend        string    `json:"cost_trend"`        // "increasing", "stable", "decreasing"

	// Additional context for certain pattern types
	ProviderID       string  `json:"provider_id,omitempty"`
	ModelName        string  `json:"model_name,omitempty"`
	UserID           string  `json:"user_id,omitempty"`
	CostBand         string  `json:"cost_band,omitempty"`
	LatencyBand      string  `json:"latency_band,omitempty"`
	AvgContextWindow int     `json:"avg_context_window,omitempty"`
	UsesFunction     bool    `json:"uses_function,omitempty"`
	UsesVision       bool    `json:"uses_vision,omitempty"`
	AvgTokens        int64   `json:"avg_tokens,omitempty"`
}

// ClusterSummary summarizes a cluster of patterns
type ClusterSummary struct {
	Type          string  `json:"type"`
	ClusterCount  int     `json:"cluster_count"`
	TotalRequests int64   `json:"total_requests"`
	TotalCost     float64 `json:"total_cost"`
	AvgClusterSize int    `json:"avg_cluster_size"`
}

// PatternAnomaly represents an anomalous usage pattern
type PatternAnomaly struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "cost-spike", "latency-spike", "error-spike"
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // "low", "medium", "high", "critical"
	DetectedAt  time.Time `json:"detected_at"`
	Pattern     *UsagePattern `json:"pattern,omitempty"`
	Baseline    float64   `json:"baseline"`
	Actual      float64   `json:"actual"`
	Deviation   float64   `json:"deviation"` // standard deviations from baseline
}

// PatternReport contains the results of pattern analysis
type PatternReport struct {
	AnalyzedAt       time.Time           `json:"analyzed_at"`
	TimeWindow       time.Duration       `json:"time_window"`
	TotalRequests    int64               `json:"total_requests"`
	TotalCost        float64             `json:"total_cost"`
	Patterns         []*UsagePattern     `json:"patterns"`
	Anomalies        []*PatternAnomaly   `json:"anomalies"`
	ClusterSummaries map[string]*ClusterSummary `json:"cluster_summaries"`
	Recommendations  []string            `json:"recommendations"`
}

// AnalysisConfig configures pattern analysis behavior
type AnalysisConfig struct {
	TimeWindow          time.Duration `json:"time_window"`
	MinRequests         int           `json:"min_requests"`          // Minimum requests to form a pattern
	MinCostUSD          float64       `json:"min_cost_usd"`          // Minimum cost to flag as expensive
	ExpensivePercentile float64       `json:"expensive_percentile"`  // Top N% considered expensive
	AnomalyThreshold    float64       `json:"anomaly_threshold"`     // Std deviations for anomaly
	EnableClustering    bool          `json:"enable_clustering"`
	EnableSubstitutions bool          `json:"enable_substitutions"`
	EnableRateLimiting  bool          `json:"enable_rate_limiting"`
	RateLimitThreshold  float64       `json:"rate_limit_threshold"`  // Requests per day
}

// DefaultAnalysisConfig returns default configuration
func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		TimeWindow:          7 * 24 * time.Hour, // 7 days
		MinRequests:         10,
		MinCostUSD:          1.0,
		ExpensivePercentile: 0.2,
		AnomalyThreshold:    2.0,
		EnableClustering:    true,
		EnableSubstitutions: true,
		EnableRateLimiting:  true,
		RateLimitThreshold:  1000, // 1000 req/day
	}
}

// ProviderRequirements specifies requirements for provider alternatives
type ProviderRequirements struct {
	MinContextWindow int
	RequiresFunction bool
	RequiresVision   bool
}
