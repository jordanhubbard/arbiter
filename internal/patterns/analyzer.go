package patterns

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/agenticorp/internal/analytics"
)

// Analyzer performs pattern analysis on request logs
type Analyzer struct {
	storage analytics.Storage
	config  *AnalysisConfig
}

// NewAnalyzer creates a new pattern analyzer
func NewAnalyzer(storage analytics.Storage, config *AnalysisConfig) *Analyzer {
	if config == nil {
		config = DefaultAnalysisConfig()
	}
	return &Analyzer{
		storage: storage,
		config:  config,
	}
}

// AnalyzePatterns performs comprehensive pattern analysis
func (a *Analyzer) AnalyzePatterns(ctx context.Context, config *AnalysisConfig) (*PatternReport, error) {
	if config == nil {
		config = a.config
	}

	// Fetch logs within time window
	startTime := time.Now().Add(-config.TimeWindow)
	filter := &analytics.LogFilter{
		StartTime: startTime,
		EndTime:   time.Now(),
		Limit:     100000, // Analyze up to 100K requests
	}

	// Get stats for summary
	stats, err := a.storage.GetLogStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// Get detailed logs for clustering
	logs, err := a.storage.GetLogs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	var allPatterns []*UsagePattern
	clusterSummaries := make(map[string]*ClusterSummary)

	if config.EnableClustering {
		// Apply all clustering strategies
		providerModelPatterns := a.clusterByProviderModel(logs, config)
		allPatterns = append(allPatterns, providerModelPatterns...)
		clusterSummaries["provider-model"] = a.summarizeCluster(providerModelPatterns)

		userPatterns := a.clusterByUser(logs, config)
		allPatterns = append(allPatterns, userPatterns...)
		clusterSummaries["user"] = a.summarizeCluster(userPatterns)

		costPatterns := a.clusterByCost(logs, config)
		allPatterns = append(allPatterns, costPatterns...)
		clusterSummaries["cost-band"] = a.summarizeCluster(costPatterns)

		temporalPatterns := a.clusterByTime(logs, config)
		allPatterns = append(allPatterns, temporalPatterns...)
		clusterSummaries["temporal"] = a.summarizeCluster(temporalPatterns)

		latencyPatterns := a.clusterByLatency(logs, config)
		allPatterns = append(allPatterns, latencyPatterns...)
		clusterSummaries["latency"] = a.summarizeCluster(latencyPatterns)
	}

	// Sort all patterns by total cost descending
	sort.Slice(allPatterns, func(i, j int) bool {
		return allPatterns[i].TotalCost > allPatterns[j].TotalCost
	})

	// Identify expensive patterns (for recommendations)
	expensivePatterns := a.identifyExpensivePatterns(allPatterns, config)

	// Detect anomalies
	anomalies := a.detectAnomalies(logs, config)

	// Generate recommendations based on expensive patterns
	recommendations := a.generateRecommendations(expensivePatterns)

	return &PatternReport{
		AnalyzedAt:       time.Now(),
		TimeWindow:       config.TimeWindow,
		TotalRequests:    stats.TotalRequests,
		TotalCost:        stats.TotalCostUSD,
		Patterns:         allPatterns, // Return all patterns, not just expensive ones
		Anomalies:        anomalies,
		ClusterSummaries: clusterSummaries,
		Recommendations:  recommendations,
	}, nil
}

// clusterByProviderModel groups requests by provider and model
func (a *Analyzer) clusterByProviderModel(logs []*analytics.RequestLog, config *AnalysisConfig) []*UsagePattern {
	clusters := make(map[string]*UsagePattern)

	for _, log := range logs {
		key := fmt.Sprintf("%s:%s", log.ProviderID, log.ModelName)

		if cluster, exists := clusters[key]; exists {
			cluster.RequestCount++
			cluster.TotalCost += log.CostUSD
			cluster.TotalTokens += int64(log.TotalTokens)
			cluster.AvgLatency = (cluster.AvgLatency*float64(cluster.RequestCount-1) + float64(log.LatencyMs)) / float64(cluster.RequestCount)

			if log.ErrorMessage != "" {
				cluster.ErrorRate = (cluster.ErrorRate*float64(cluster.RequestCount-1) + 1) / float64(cluster.RequestCount)
			} else {
				cluster.ErrorRate = cluster.ErrorRate * float64(cluster.RequestCount-1) / float64(cluster.RequestCount)
			}

			if log.Timestamp.After(cluster.LastSeen) {
				cluster.LastSeen = log.Timestamp
			}
			if log.Timestamp.Before(cluster.FirstSeen) {
				cluster.FirstSeen = log.Timestamp
			}
		} else {
			errorRate := 0.0
			if log.ErrorMessage != "" {
				errorRate = 1.0
			}

			clusters[key] = &UsagePattern{
				ID:           uuid.New().String(),
				Type:         "provider-model",
				GroupKey:     key,
				ProviderID:   log.ProviderID,
				ModelName:    log.ModelName,
				RequestCount: 1,
				TotalCost:    log.CostUSD,
				TotalTokens:  int64(log.TotalTokens),
				AvgLatency:   float64(log.LatencyMs),
				ErrorRate:    errorRate,
				FirstSeen:    log.Timestamp,
				LastSeen:     log.Timestamp,
				AvgTokens:    int64(log.TotalTokens),
			}
		}
	}

	// Calculate derived metrics
	patterns := make([]*UsagePattern, 0, len(clusters))
	for _, pattern := range clusters {
		pattern.AvgCost = pattern.TotalCost / float64(pattern.RequestCount)
		pattern.AvgTokens = pattern.TotalTokens / pattern.RequestCount

		// Calculate request frequency (requests per day)
		duration := pattern.LastSeen.Sub(pattern.FirstSeen)
		if duration > 0 {
			daysSpan := duration.Hours() / 24
			if daysSpan < 1 {
				daysSpan = 1
			}
			pattern.RequestFrequency = float64(pattern.RequestCount) / daysSpan
		}

		// Filter by minimum thresholds
		if pattern.RequestCount >= int64(config.MinRequests) && pattern.TotalCost >= config.MinCostUSD {
			patterns = append(patterns, pattern)
		}
	}

	return patterns
}

// clusterByUser groups requests by user
func (a *Analyzer) clusterByUser(logs []*analytics.RequestLog, config *AnalysisConfig) []*UsagePattern {
	clusters := make(map[string]*UsagePattern)

	for _, log := range logs {
		key := log.UserID

		if cluster, exists := clusters[key]; exists {
			cluster.RequestCount++
			cluster.TotalCost += log.CostUSD
			cluster.TotalTokens += int64(log.TotalTokens)
			cluster.AvgLatency = (cluster.AvgLatency*float64(cluster.RequestCount-1) + float64(log.LatencyMs)) / float64(cluster.RequestCount)

			if log.Timestamp.After(cluster.LastSeen) {
				cluster.LastSeen = log.Timestamp
			}
			if log.Timestamp.Before(cluster.FirstSeen) {
				cluster.FirstSeen = log.Timestamp
			}
		} else {
			clusters[key] = &UsagePattern{
				ID:           uuid.New().String(),
				Type:         "user",
				GroupKey:     key,
				UserID:       key,
				RequestCount: 1,
				TotalCost:    log.CostUSD,
				TotalTokens:  int64(log.TotalTokens),
				AvgLatency:   float64(log.LatencyMs),
				FirstSeen:    log.Timestamp,
				LastSeen:     log.Timestamp,
			}
		}
	}

	patterns := make([]*UsagePattern, 0, len(clusters))
	for _, pattern := range clusters {
		pattern.AvgCost = pattern.TotalCost / float64(pattern.RequestCount)

		duration := pattern.LastSeen.Sub(pattern.FirstSeen)
		if duration > 0 {
			daysSpan := duration.Hours() / 24
			if daysSpan < 1 {
				daysSpan = 1
			}
			pattern.RequestFrequency = float64(pattern.RequestCount) / daysSpan
		}

		if pattern.RequestCount >= int64(config.MinRequests) {
			patterns = append(patterns, pattern)
		}
	}

	return patterns
}

// clusterByCost groups requests by cost bands
func (a *Analyzer) clusterByCost(logs []*analytics.RequestLog, config *AnalysisConfig) []*UsagePattern {
	clusters := make(map[string]*UsagePattern)

	getCostBand := func(cost float64) string {
		switch {
		case cost < 0.01:
			return "<$0.01"
		case cost < 0.10:
			return "$0.01-$0.10"
		case cost < 1.00:
			return "$0.10-$1.00"
		default:
			return ">$1.00"
		}
	}

	for _, log := range logs {
		key := getCostBand(log.CostUSD)

		if cluster, exists := clusters[key]; exists {
			cluster.RequestCount++
			cluster.TotalCost += log.CostUSD
			cluster.TotalTokens += int64(log.TotalTokens)
			cluster.AvgLatency = (cluster.AvgLatency*float64(cluster.RequestCount-1) + float64(log.LatencyMs)) / float64(cluster.RequestCount)

			if log.Timestamp.After(cluster.LastSeen) {
				cluster.LastSeen = log.Timestamp
			}
			if log.Timestamp.Before(cluster.FirstSeen) {
				cluster.FirstSeen = log.Timestamp
			}
		} else {
			clusters[key] = &UsagePattern{
				ID:           uuid.New().String(),
				Type:         "cost-band",
				GroupKey:     key,
				CostBand:     key,
				RequestCount: 1,
				TotalCost:    log.CostUSD,
				TotalTokens:  int64(log.TotalTokens),
				AvgLatency:   float64(log.LatencyMs),
				FirstSeen:    log.Timestamp,
				LastSeen:     log.Timestamp,
			}
		}
	}

	patterns := make([]*UsagePattern, 0, len(clusters))
	for _, pattern := range clusters {
		pattern.AvgCost = pattern.TotalCost / float64(pattern.RequestCount)

		duration := pattern.LastSeen.Sub(pattern.FirstSeen)
		if duration > 0 {
			daysSpan := duration.Hours() / 24
			if daysSpan < 1 {
				daysSpan = 1
			}
			pattern.RequestFrequency = float64(pattern.RequestCount) / daysSpan
		}

		patterns = append(patterns, pattern)
	}

	return patterns
}

// clusterByTime groups requests by temporal windows
func (a *Analyzer) clusterByTime(logs []*analytics.RequestLog, config *AnalysisConfig) []*UsagePattern {
	clusters := make(map[string]*UsagePattern)

	for _, log := range logs {
		hour := log.Timestamp.Hour()
		var timeWindow string
		switch {
		case hour >= 0 && hour < 6:
			timeWindow = "00:00-06:00"
		case hour >= 6 && hour < 12:
			timeWindow = "06:00-12:00"
		case hour >= 12 && hour < 18:
			timeWindow = "12:00-18:00"
		default:
			timeWindow = "18:00-00:00"
		}

		key := timeWindow

		if cluster, exists := clusters[key]; exists {
			cluster.RequestCount++
			cluster.TotalCost += log.CostUSD
			cluster.TotalTokens += int64(log.TotalTokens)
			cluster.AvgLatency = (cluster.AvgLatency*float64(cluster.RequestCount-1) + float64(log.LatencyMs)) / float64(cluster.RequestCount)

			if log.Timestamp.After(cluster.LastSeen) {
				cluster.LastSeen = log.Timestamp
			}
			if log.Timestamp.Before(cluster.FirstSeen) {
				cluster.FirstSeen = log.Timestamp
			}
		} else {
			clusters[key] = &UsagePattern{
				ID:           uuid.New().String(),
				Type:         "temporal",
				GroupKey:     key,
				RequestCount: 1,
				TotalCost:    log.CostUSD,
				TotalTokens:  int64(log.TotalTokens),
				AvgLatency:   float64(log.LatencyMs),
				FirstSeen:    log.Timestamp,
				LastSeen:     log.Timestamp,
			}
		}
	}

	patterns := make([]*UsagePattern, 0, len(clusters))
	for _, pattern := range clusters {
		pattern.AvgCost = pattern.TotalCost / float64(pattern.RequestCount)

		duration := pattern.LastSeen.Sub(pattern.FirstSeen)
		if duration > 0 {
			daysSpan := duration.Hours() / 24
			if daysSpan < 1 {
				daysSpan = 1
			}
			pattern.RequestFrequency = float64(pattern.RequestCount) / daysSpan
		}

		patterns = append(patterns, pattern)
	}

	return patterns
}

// clusterByLatency groups requests by latency bands
func (a *Analyzer) clusterByLatency(logs []*analytics.RequestLog, config *AnalysisConfig) []*UsagePattern {
	clusters := make(map[string]*UsagePattern)

	getLatencyBand := func(latencyMs int64) string {
		switch {
		case latencyMs < 100:
			return "<100ms"
		case latencyMs < 500:
			return "100-500ms"
		case latencyMs < 2000:
			return "500-2000ms"
		default:
			return ">2000ms"
		}
	}

	for _, log := range logs {
		key := getLatencyBand(log.LatencyMs)

		if cluster, exists := clusters[key]; exists {
			cluster.RequestCount++
			cluster.TotalCost += log.CostUSD
			cluster.TotalTokens += int64(log.TotalTokens)
			cluster.AvgLatency = (cluster.AvgLatency*float64(cluster.RequestCount-1) + float64(log.LatencyMs)) / float64(cluster.RequestCount)

			if log.Timestamp.After(cluster.LastSeen) {
				cluster.LastSeen = log.Timestamp
			}
			if log.Timestamp.Before(cluster.FirstSeen) {
				cluster.FirstSeen = log.Timestamp
			}
		} else {
			clusters[key] = &UsagePattern{
				ID:           uuid.New().String(),
				Type:         "latency",
				GroupKey:     key,
				LatencyBand:  key,
				RequestCount: 1,
				TotalCost:    log.CostUSD,
				TotalTokens:  int64(log.TotalTokens),
				AvgLatency:   float64(log.LatencyMs),
				FirstSeen:    log.Timestamp,
				LastSeen:     log.Timestamp,
			}
		}
	}

	patterns := make([]*UsagePattern, 0, len(clusters))
	for _, pattern := range clusters {
		pattern.AvgCost = pattern.TotalCost / float64(pattern.RequestCount)

		duration := pattern.LastSeen.Sub(pattern.FirstSeen)
		if duration > 0 {
			daysSpan := duration.Hours() / 24
			if daysSpan < 1 {
				daysSpan = 1
			}
			pattern.RequestFrequency = float64(pattern.RequestCount) / daysSpan
		}

		patterns = append(patterns, pattern)
	}

	return patterns
}

// identifyExpensivePatterns finds patterns in the top N% by cost
func (a *Analyzer) identifyExpensivePatterns(patterns []*UsagePattern, config *AnalysisConfig) []*UsagePattern {
	if len(patterns) == 0 {
		return nil
	}

	// Calculate cost percentiles
	costs := make([]float64, len(patterns))
	for i, p := range patterns {
		costs[i] = p.TotalCost
	}
	sort.Float64s(costs)

	// Find threshold at the specified percentile
	percentileIndex := int(float64(len(costs)) * (1.0 - config.ExpensivePercentile))
	if percentileIndex >= len(costs) {
		percentileIndex = len(costs) - 1
	}
	threshold := costs[percentileIndex]

	// Filter patterns above threshold
	var expensive []*UsagePattern
	for _, p := range patterns {
		if p.TotalCost >= threshold {
			expensive = append(expensive, p)
		}
	}

	// Sort by total cost descending
	sort.Slice(expensive, func(i, j int) bool {
		return expensive[i].TotalCost > expensive[j].TotalCost
	})

	return expensive
}

// detectAnomalies identifies anomalous patterns using statistical methods
func (a *Analyzer) detectAnomalies(logs []*analytics.RequestLog, config *AnalysisConfig) []*PatternAnomaly {
	if len(logs) == 0 {
		return nil
	}

	var anomalies []*PatternAnomaly

	// Calculate baseline statistics
	var costs []float64
	var latencies []float64
	errorCount := 0

	for _, log := range logs {
		costs = append(costs, log.CostUSD)
		latencies = append(latencies, float64(log.LatencyMs))
		if log.ErrorMessage != "" {
			errorCount++
		}
	}

	// Cost anomalies
	costMean, costStdDev := calculateStats(costs)
	for _, log := range logs {
		deviation := math.Abs(log.CostUSD-costMean) / costStdDev
		if deviation >= config.AnomalyThreshold {
			severity := getSeverity(deviation)
			anomalies = append(anomalies, &PatternAnomaly{
				ID:          uuid.New().String(),
				Type:        "cost-spike",
				Description: fmt.Sprintf("Unusually high cost: $%.4f (%.1f std devs from mean)", log.CostUSD, deviation),
				Severity:    severity,
				DetectedAt:  time.Now(),
				Baseline:    costMean,
				Actual:      log.CostUSD,
				Deviation:   deviation,
			})
		}
	}

	// Latency anomalies
	latencyMean, latencyStdDev := calculateStats(latencies)
	for _, log := range logs {
		deviation := math.Abs(float64(log.LatencyMs)-latencyMean) / latencyStdDev
		if deviation >= config.AnomalyThreshold {
			severity := getSeverity(deviation)
			anomalies = append(anomalies, &PatternAnomaly{
				ID:          uuid.New().String(),
				Type:        "latency-spike",
				Description: fmt.Sprintf("Unusually high latency: %dms (%.1f std devs from mean)", log.LatencyMs, deviation),
				Severity:    severity,
				DetectedAt:  time.Now(),
				Baseline:    latencyMean,
				Actual:      float64(log.LatencyMs),
				Deviation:   deviation,
			})
		}
	}

	// Error rate anomaly
	errorRate := float64(errorCount) / float64(len(logs))
	if errorRate > 0.05 { // More than 5% error rate
		anomalies = append(anomalies, &PatternAnomaly{
			ID:          uuid.New().String(),
			Type:        "error-spike",
			Description: fmt.Sprintf("High error rate: %.1f%%", errorRate*100),
			Severity:    getSeverity(errorRate * 20), // Scale to severity
			DetectedAt:  time.Now(),
			Baseline:    0.01, // Assume 1% baseline
			Actual:      errorRate,
			Deviation:   errorRate / 0.01,
		})
	}

	return anomalies
}

// generateRecommendations creates high-level recommendations
func (a *Analyzer) generateRecommendations(patterns []*UsagePattern) []string {
	var recommendations []string

	// Find most expensive pattern
	if len(patterns) > 0 {
		mostExpensive := patterns[0]
		recommendations = append(recommendations,
			fmt.Sprintf("Consider optimizing %s (%s) - costing $%.2f across %d requests",
				mostExpensive.GroupKey, mostExpensive.Type, mostExpensive.TotalCost, mostExpensive.RequestCount))
	}

	// Check for high-frequency patterns
	for _, pattern := range patterns {
		if pattern.RequestFrequency > a.config.RateLimitThreshold {
			recommendations = append(recommendations,
				fmt.Sprintf("High frequency detected in %s (%.0f req/day) - consider caching or rate limiting",
					pattern.GroupKey, pattern.RequestFrequency))
			break
		}
	}

	return recommendations
}

// summarizeCluster creates a summary of a cluster
func (a *Analyzer) summarizeCluster(patterns []*UsagePattern) *ClusterSummary {
	if len(patterns) == 0 {
		return &ClusterSummary{
			Type:         "",
			ClusterCount: 0,
		}
	}

	var totalRequests int64
	var totalCost float64

	for _, p := range patterns {
		totalRequests += p.RequestCount
		totalCost += p.TotalCost
	}

	avgSize := 0
	if len(patterns) > 0 {
		avgSize = int(totalRequests) / len(patterns)
	}

	return &ClusterSummary{
		Type:           patterns[0].Type,
		ClusterCount:   len(patterns),
		TotalRequests:  totalRequests,
		TotalCost:      totalCost,
		AvgClusterSize: avgSize,
	}
}

// Helper functions

func calculateStats(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	// Calculate standard deviation
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values))
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

func getSeverity(deviation float64) string {
	switch {
	case deviation >= 4.0:
		return "critical"
	case deviation >= 3.0:
		return "high"
	case deviation >= 2.0:
		return "medium"
	default:
		return "low"
	}
}
