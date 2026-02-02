package patterns

import (
	"context"

	"github.com/jordanhubbard/agenticorp/internal/analytics"
	"github.com/jordanhubbard/agenticorp/internal/cache"
)

// Manager coordinates pattern analysis and optimization
type Manager struct {
	storage         analytics.Storage
	cacheAnalyzer   *cache.Analyzer
	patternAnalyzer *Analyzer
	optimizer       *Optimizer
	config          *AnalysisConfig
}

// NewManager creates a new pattern manager
func NewManager(storage analytics.Storage, config *AnalysisConfig) *Manager {
	if config == nil {
		config = DefaultAnalysisConfig()
	}

	return &Manager{
		storage:         storage,
		cacheAnalyzer:   cache.NewAnalyzer(storage, nil),
		patternAnalyzer: NewAnalyzer(storage, config),
		optimizer:       NewOptimizer(config),
		config:          config,
	}
}

// AnalyzeAll performs comprehensive analysis across all dimensions
func (m *Manager) AnalyzeAll(ctx context.Context) (*ComprehensiveReport, error) {
	// Run pattern analysis
	patternReport, err := m.patternAnalyzer.AnalyzePatterns(ctx, m.config)
	if err != nil {
		return nil, err
	}

	// Run cache analysis (existing)
	cacheReport, err := m.cacheAnalyzer.Analyze(ctx)
	if err != nil {
		// Log error but continue - cache analysis is optional
		cacheReport = &cache.AnalysisReport{
			Opportunities: []*cache.CacheOpportunity{},
		}
	}

	// Generate optimizations from patterns
	optimizations := m.optimizer.GenerateRecommendations(patternReport.Patterns)

	// Calculate total savings
	totalSavings := 0.0
	monthlySavings := 0.0
	for _, opt := range optimizations {
		totalSavings += opt.ProjectedSavingsUSD
		monthlySavings += opt.MonthlySavingsUSD
	}

	// Add cache savings
	totalSavings += cacheReport.TotalSavingsUSD
	monthlySavings += cacheReport.MonthlyProjection

	// Placeholder for future batching analysis
	batchingOpportunities := []BatchOpportunity{}

	return &ComprehensiveReport{
		PatternAnalysis:       patternReport,
		CacheOpportunities:    cacheReport.Opportunities,
		BatchingOpportunities: batchingOpportunities,
		Optimizations:         optimizations,
		TotalSavingsUSD:       totalSavings,
		MonthlySavingsUSD:     monthlySavings,
	}, nil
}

// AnalyzePatterns runs only pattern analysis
func (m *Manager) AnalyzePatterns(ctx context.Context) (*PatternReport, error) {
	return m.patternAnalyzer.AnalyzePatterns(ctx, m.config)
}

// GetOptimizations gets optimization recommendations for patterns
func (m *Manager) GetOptimizations(ctx context.Context) ([]*Optimization, error) {
	patternReport, err := m.patternAnalyzer.AnalyzePatterns(ctx, m.config)
	if err != nil {
		return nil, err
	}

	return m.optimizer.GenerateRecommendations(patternReport.Patterns), nil
}

// GetExpensivePatterns returns the most expensive patterns
func (m *Manager) GetExpensivePatterns(ctx context.Context, limit int) ([]*UsagePattern, error) {
	patternReport, err := m.patternAnalyzer.AnalyzePatterns(ctx, m.config)
	if err != nil {
		return nil, err
	}

	patterns := patternReport.Patterns
	if limit > 0 && limit < len(patterns) {
		patterns = patterns[:limit]
	}

	return patterns, nil
}

// GetAnomalies returns detected anomalies
func (m *Manager) GetAnomalies(ctx context.Context) ([]*PatternAnomaly, error) {
	patternReport, err := m.patternAnalyzer.AnalyzePatterns(ctx, m.config)
	if err != nil {
		return nil, err
	}

	return patternReport.Anomalies, nil
}
