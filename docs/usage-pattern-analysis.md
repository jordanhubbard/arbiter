# Usage Pattern Analysis Engine (ac-a87)

## Overview

The usage pattern analysis engine identifies usage patterns, detects expensive operations, and generates optimization opportunities across multiple dimensions.

## Architecture

```
RequestLogs (Analytics) → Pattern Analyzer → Optimization Engine → Recommendations
                              ↓                     ↓
                        Pattern Clusters    [Cache, Substitution, Batching, etc.]
```

## Features

### 1. Multi-dimensional Pattern Clustering

The analyzer groups requests across five dimensions:

#### Provider-Model Clustering
- Groups by: `provider_id + model_name`
- Metrics: total_requests, total_cost, avg_latency, error_rate
- Use case: Compare costs across providers for same workload

#### User Clustering
- Groups by: `user_id`
- Metrics: requests_per_day, cost_per_user, favorite_models
- Use case: Identify high-cost users, detect anomalies

#### Cost Clustering
- Groups by: cost_per_request bands (<$0.01, $0.01-$0.10, $0.10-$1.00, >$1.00)
- Metrics: request_count, total_cost, common_patterns
- Use case: Find expensive outliers

#### Temporal Clustering
- Groups by: hour_of_day (6-hour windows)
- Metrics: request_volume, peak_cost_times
- Use case: Identify usage patterns for capacity planning

#### Latency Clustering
- Groups by: latency bands (<100ms, 100-500ms, 500-2000ms, >2000ms)
- Metrics: request_count, avg_cost, providers_used
- Use case: Find performance bottlenecks

### 2. Anomaly Detection

Statistical anomaly detection identifies:
- **Cost spikes**: Requests with unusually high cost (>2σ from mean)
- **Latency spikes**: Requests with unusually high latency (>2σ from mean)
- **Error spikes**: High error rates (>5%)

### 3. Optimization Recommendations

The optimizer generates actionable recommendations:

#### Provider Substitution
- Identifies expensive provider-model combinations
- Suggests cheaper alternatives with similar capabilities
- Estimates potential savings

#### Model Substitution
- Suggests cheaper models within the same provider
- Estimates quality trade-offs

#### Rate Limiting
- Identifies high-frequency, low-value patterns
- Recommends throttling for excessive usage

#### Cache Integration
- References existing cache analyzer's duplicate detection
- Includes cache opportunities in unified view

## API Endpoints

### GET /api/v1/patterns/analysis
Full pattern analysis across all dimensions.

**Query Parameters:**
- `time_window` (int): Hours to analyze (default: 168 = 7 days)
- `min_requests` (int): Minimum requests to form a pattern (default: 10)
- `min_cost` (float): Minimum cost to flag as expensive (default: 1.0)

**Example:**
```bash
curl http://localhost:8080/api/v1/patterns/analysis?time_window=168
```

**Response:**
```json
{
  "analyzed_at": "2026-02-01T...",
  "time_window": 604800000000000,
  "total_requests": 1500,
  "total_cost": 125.50,
  "patterns": [...],
  "anomalies": [...],
  "cluster_summaries": {...},
  "recommendations": [...]
}
```

### GET /api/v1/patterns/expensive
Top N most expensive patterns.

**Query Parameters:**
- `limit` (int): Number of patterns to return (default: 10)
- `min_cost_usd` (float): Minimum cost threshold

**Example:**
```bash
curl http://localhost:8080/api/v1/patterns/expensive?limit=5
```

### GET /api/v1/patterns/anomalies
Detect anomalies in usage patterns.

**Query Parameters:**
- `threshold` (float): Standard deviations for anomaly detection
- `type` (string): Filter by anomaly type (cost-spike, latency-spike, error-spike)

**Example:**
```bash
curl http://localhost:8080/api/v1/patterns/anomalies
```

### GET /api/v1/optimizations
Unified view of all optimization opportunities (patterns + caching + batching).

**Query Parameters:**
- `type` (string): Filter by optimization type (provider-substitution, model-substitution, rate-limit)
- `min_savings` (float): Minimum monthly savings in USD
- `impact_rating` (string): Filter by impact (high, medium, low)

**Example:**
```bash
curl 'http://localhost:8080/api/v1/optimizations?min_savings=100'
```

**Response:**
```json
{
  "optimizations": [...],
  "count": 5,
  "total_savings_usd": 250.75,
  "monthly_savings_usd": 1075.00,
  "cache_opportunities": 12,
  "batching_opportunities": 3
}
```

### GET /api/v1/optimizations/substitutions
Provider and model substitution recommendations.

**Query Parameters:**
- `pattern_id` (string): Filter by specific pattern
- `provider_id` (string): Filter by provider

**Example:**
```bash
curl http://localhost:8080/api/v1/optimizations/substitutions
```

### POST /api/v1/optimizations/{id}/apply
Apply an optimization recommendation (placeholder for future implementation).

**Request Body:**
```json
{
  "confirmation": true,
  "options": {...}
}
```

## Configuration

Default configuration (7 days, minimum 10 requests, $1.00 minimum cost):

```go
config := patterns.DefaultAnalysisConfig()
// Override defaults:
config.TimeWindow = 30 * 24 * time.Hour  // 30 days
config.MinRequests = 5
config.MinCostUSD = 0.50
config.ExpensivePercentile = 0.1  // Top 10%
```

## Usage Example

### Programmatic Usage

```go
import (
    "github.com/jordanhubbard/agenticorp/internal/patterns"
    "github.com/jordanhubbard/agenticorp/internal/analytics"
)

// Initialize storage
storage, _ := analytics.NewDatabaseStorage(db)

// Create pattern manager
manager := patterns.NewManager(storage, nil)

// Run comprehensive analysis
report, err := manager.AnalyzeAll(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total Savings: $%.2f\n", report.TotalSavingsUSD)
fmt.Printf("Monthly Projection: $%.2f\n", report.MonthlySavingsUSD)

// Get specific optimizations
for _, opt := range report.Optimizations {
    if opt.ImpactRating == "high" {
        fmt.Printf("HIGH IMPACT: %s\n", opt.Recommendation)
        fmt.Printf("  Savings: $%.2f/month\n", opt.MonthlySavingsUSD)
    }
}
```

### CLI Usage

```bash
# Get pattern analysis
curl http://localhost:8080/api/v1/patterns/analysis | jq '.patterns | length'

# Find expensive patterns
curl http://localhost:8080/api/v1/patterns/expensive?limit=5 | jq '.patterns[0]'

# Get all optimizations with >$100 monthly savings
curl 'http://localhost:8080/api/v1/optimizations?min_savings=100' \
  | jq '.optimizations[] | {type, savings: .monthly_savings_usd, recommendation}'
```

## Database Schema

### usage_patterns Table
Stores analyzed patterns (optional - for caching results):

```sql
CREATE TABLE usage_patterns (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    group_key TEXT NOT NULL,
    request_count INTEGER NOT NULL,
    total_cost REAL NOT NULL,
    avg_cost REAL NOT NULL,
    analyzed_at DATETIME NOT NULL,
    metadata_json TEXT
);
```

### optimizations Table
Tracks applied optimizations:

```sql
CREATE TABLE optimizations (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    pattern_id TEXT,
    recommendation TEXT NOT NULL,
    projected_savings_usd REAL NOT NULL,
    actual_savings_usd REAL,
    applied_at DATETIME,
    status TEXT DEFAULT 'pending'
);
```

## Performance Considerations

- Analysis runs on up to 100K requests per query
- Uses in-memory aggregation (no persistent pattern storage by default)
- Results are computed on-demand (no background jobs)
- Consider caching analysis results for frequently-accessed data

## Integration Points

- **Analytics**: Reads from `request_logs` table
- **Cache Analyzer**: Includes cache opportunities in unified report
- **Provider Routing**: Uses routing system to find cheaper alternatives
- **Model Catalog**: Uses catalog for model comparisons

## Future Enhancements

- Machine learning-based clustering (k-means, DBSCAN)
- Semantic similarity analysis using embeddings
- Predictive cost modeling
- Automated A/B testing of substitutions
- Real-time pattern streaming
- Budget enforcement and alerts
- Multi-tenant cost allocation

## Related Beads

- **ac-o1c**: Prompt optimization (uses pattern analysis to identify verbose prompts)
- **ac-9lm**: Provider substitutions (built into optimizer)
- **ac-2uj**: Team usage analytics (uses user clustering for team views)
- **ac-fuh**: Cache analyzer (integrated into comprehensive report)
