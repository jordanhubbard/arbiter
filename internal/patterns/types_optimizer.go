package patterns

// Optimization represents a recommended optimization
type Optimization struct {
	ID                   string                `json:"id"`
	Type                 string                `json:"type"` // "provider-substitution", "model-substitution", "caching", "batching", "rate-limit"
	Pattern              *UsagePattern         `json:"pattern"`
	Recommendation       string                `json:"recommendation"`
	CurrentCost          float64               `json:"current_cost"`
	ProjectedCost        float64               `json:"projected_cost"`
	ProjectedSavingsUSD  float64               `json:"projected_savings_usd"`
	MonthlySavingsUSD    float64               `json:"monthly_savings_usd"`
	ImpactRating         string                `json:"impact_rating"`  // "high", "medium", "low"
	QualityImpact        string                `json:"quality_impact"` // "none", "minimal", "moderate", "significant"
	AlternativeProviders []ProviderAlternative `json:"alternative_providers,omitempty"`
	AlternativeModels    []ModelAlternative    `json:"alternative_models,omitempty"`
	AutoApplicable       bool                  `json:"auto_applicable"`
	Confidence           float64               `json:"confidence"` // 0-1
}

// ProviderAlternative represents an alternative provider option
type ProviderAlternative struct {
	ProviderID       string  `json:"provider_id"`
	ModelName        string  `json:"model_name"`
	CostPerMToken    float64 `json:"cost_per_mtoken"`
	EstimatedSavings float64 `json:"estimated_savings"`
	QualityScore     float64 `json:"quality_score"`
	LatencyDelta     int64   `json:"latency_delta"`  // ms difference
	CapabilityMatch  bool    `json:"capability_match"`
}

// ModelAlternative represents an alternative model within same provider
type ModelAlternative struct {
	ModelName        string  `json:"model_name"`
	CostPerMToken    float64 `json:"cost_per_mtoken"`
	EstimatedSavings float64 `json:"estimated_savings"`
	QualityScore     float64 `json:"quality_score"`
	CapabilityMatch  bool    `json:"capability_match"`
}
