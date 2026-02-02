package patterns

import "github.com/jordanhubbard/agenticorp/internal/cache"

// ComprehensiveReport combines all analysis results
type ComprehensiveReport struct {
	PatternAnalysis        *PatternReport           `json:"pattern_analysis"`
	CacheOpportunities     []*cache.CacheOpportunity `json:"cache_opportunities"`
	BatchingOpportunities  []BatchOpportunity       `json:"batching_opportunities"`
	Optimizations          []*Optimization          `json:"optimizations"`
	TotalSavingsUSD        float64                  `json:"total_savings_usd"`
	MonthlySavingsUSD      float64                  `json:"monthly_savings_usd"`
}

// BatchOpportunity represents a batching opportunity
// This is a placeholder for future batching analysis
type BatchOpportunity struct {
	ID               string  `json:"id"`
	Pattern          string  `json:"pattern"`
	RequestCount     int64   `json:"request_count"`
	PotentialSavings float64 `json:"potential_savings"`
	TimeWindow       string  `json:"time_window"`
}
