package provider

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Scorer: GetCompositeScore
// ---------------------------------------------------------------------------

func TestScorerGetCompositeScore_Exists(t *testing.T) {
	s := NewScorer()
	s.UpdateProviderMetrics("p1", 30, 100, 500, 0)

	score := s.GetCompositeScore("p1")
	if score <= 0 {
		t.Errorf("expected positive composite score, got %f", score)
	}
}

func TestScorerGetCompositeScore_NotExists(t *testing.T) {
	s := NewScorer()
	score := s.GetCompositeScore("nope")
	if score != 0 {
		t.Errorf("expected 0 for nonexistent provider, got %f", score)
	}
}

// ---------------------------------------------------------------------------
// Scorer: SelectBestForComplexity
// ---------------------------------------------------------------------------

func TestScorerSelectBestForComplexity_SimpleTask(t *testing.T) {
	s := NewScorer()
	s.UpdateProviderMetrics("small", 7, 100, 500, 0)
	s.UpdateProviderMetrics("large", 70, 100, 500, 0)

	bestID, score, found := s.SelectBestForComplexity(
		[]string{"small", "large"}, ComplexitySimple,
	)
	if !found {
		t.Fatal("expected to find provider")
	}
	if bestID != "small" {
		t.Errorf("expected small for simple task, got %q", bestID)
	}
	if score < 0 {
		t.Errorf("expected non-negative score, got %f", score)
	}
}

func TestScorerSelectBestForComplexity_ComplexTask(t *testing.T) {
	s := NewScorer()
	s.UpdateProviderMetrics("small", 7, 100, 500, 0)
	s.UpdateProviderMetrics("large", 70, 100, 500, 0)

	bestID, _, found := s.SelectBestForComplexity(
		[]string{"small", "large"}, ComplexityComplex,
	)
	if !found {
		t.Fatal("expected to find provider")
	}
	if bestID != "large" {
		t.Errorf("expected large for complex task, got %q", bestID)
	}
}

func TestScorerSelectBestForComplexity_NoProviders(t *testing.T) {
	s := NewScorer()
	_, _, found := s.SelectBestForComplexity([]string{}, ComplexityMedium)
	if found {
		t.Error("expected found=false with empty list")
	}
}

func TestScorerSelectBestForComplexity_UnknownProviders(t *testing.T) {
	s := NewScorer()
	// Providers without scores should still work (score = 0)
	bestID, score, found := s.SelectBestForComplexity(
		[]string{"a", "b"}, ComplexityMedium,
	)
	if !found {
		t.Fatal("expected to find provider")
	}
	if bestID == "" {
		t.Error("expected non-empty ID")
	}
	if score != 0 {
		t.Errorf("expected score=0 for unknown, got %f", score)
	}
}

// ---------------------------------------------------------------------------
// Scorer: RankProviders with empty and single
// ---------------------------------------------------------------------------

func TestScorerRankProviders_Empty(t *testing.T) {
	s := NewScorer()
	ranked := s.RankProviders([]string{})
	if len(ranked) != 0 {
		t.Errorf("expected empty, got %d", len(ranked))
	}
}

func TestScorerRankProviders_Single(t *testing.T) {
	s := NewScorer()
	s.UpdateProviderMetrics("only", 30, 100, 500, 0)
	ranked := s.RankProviders([]string{"only"})
	if len(ranked) != 1 {
		t.Fatalf("expected 1, got %d", len(ranked))
	}
	if ranked[0] != "only" {
		t.Errorf("expected 'only', got %q", ranked[0])
	}
}

func TestScorerRankProviders_UnknownProviders(t *testing.T) {
	s := NewScorer()
	// Providers without metrics get score 0
	ranked := s.RankProviders([]string{"x", "y", "z"})
	if len(ranked) != 3 {
		t.Errorf("expected 3, got %d", len(ranked))
	}
}

// ---------------------------------------------------------------------------
// Scorer: RankProvidersForComplexity edge cases
// ---------------------------------------------------------------------------

func TestScorerRankProvidersForComplexity_Empty(t *testing.T) {
	s := NewScorer()
	ranked := s.RankProvidersForComplexity([]string{}, ComplexityMedium)
	if len(ranked) != 0 {
		t.Errorf("expected empty, got %d", len(ranked))
	}
}

func TestScorerRankProvidersForComplexity_OnlyUnderqualified(t *testing.T) {
	s := NewScorer()
	s.UpdateProviderMetrics("tiny", 3, 100, 500, 0) // TierSmall

	ranked := s.RankProvidersForComplexity([]string{"tiny"}, ComplexityExtended)
	if len(ranked) != 1 {
		t.Fatalf("expected 1, got %d", len(ranked))
	}
	if ranked[0] != "tiny" {
		t.Errorf("expected tiny, got %q", ranked[0])
	}
}

func TestScorerRankProvidersForComplexity_OverqualifiedPrefersSmaller(t *testing.T) {
	s := NewScorer()
	// Both overqualified for simple task, but medium is "less wasteful"
	s.UpdateProviderMetrics("medium", 32, 100, 500, 0)  // TierMedium
	s.UpdateProviderMetrics("xlarge", 480, 100, 500, 0)  // TierXLarge

	ranked := s.RankProvidersForComplexity(
		[]string{"medium", "xlarge"}, ComplexitySimple,
	)
	// Both overqualified - medium should come first (smaller = less waste)
	if ranked[0] != "medium" {
		t.Errorf("expected medium first (less wasteful), got %q", ranked[0])
	}
}

func TestScorerRankProvidersForComplexity_UnderqualifiedPrefersLarger(t *testing.T) {
	s := NewScorer()
	// Both underqualified for extended task
	s.UpdateProviderMetrics("small", 7, 100, 500, 0)   // TierSmall
	s.UpdateProviderMetrics("medium", 32, 100, 500, 0)  // TierMedium

	ranked := s.RankProvidersForComplexity(
		[]string{"small", "medium"}, ComplexityExtended,
	)
	// Both underqualified - medium should come first (closer to capable)
	if ranked[0] != "medium" {
		t.Errorf("expected medium first (closer to capable), got %q", ranked[0])
	}
}

// ---------------------------------------------------------------------------
// Scorer: SetWeights / GetWeights
// ---------------------------------------------------------------------------

func TestScorerSetGetWeights(t *testing.T) {
	s := NewScorer()
	custom := ScoringWeights{
		ModelSize:      10,
		RoundTrip:      20,
		RequestLatency: 30,
		Cost:           40,
	}
	s.SetWeights(custom)
	got := s.GetWeights()
	if got.ModelSize != 10 || got.RoundTrip != 20 || got.RequestLatency != 30 || got.Cost != 40 {
		t.Errorf("weights not set correctly: %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Scorer: GetScore
// ---------------------------------------------------------------------------

func TestScorerGetScore_Exists(t *testing.T) {
	s := NewScorer()
	s.UpdateProviderMetrics("p", 30, 100, 500, 1.0)

	score, ok := s.GetScore("p")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if score == nil {
		t.Fatal("expected non-nil score")
	}
	if score.ProviderID != "p" {
		t.Errorf("ProviderID = %q, want %q", score.ProviderID, "p")
	}
	if score.ModelParamsB != 30 {
		t.Errorf("ModelParamsB = %f, want 30", score.ModelParamsB)
	}
	if score.HeartbeatLatencyMs != 100 {
		t.Errorf("HeartbeatLatencyMs = %d, want 100", score.HeartbeatLatencyMs)
	}
	if score.AvgRequestLatencyMs != 500 {
		t.Errorf("AvgRequestLatencyMs = %f, want 500", score.AvgRequestLatencyMs)
	}
	if score.CostPerMToken != 1.0 {
		t.Errorf("CostPerMToken = %f, want 1.0", score.CostPerMToken)
	}
	if score.LastUpdated.IsZero() {
		t.Error("LastUpdated should not be zero")
	}
}

func TestScorerGetScore_NotExists(t *testing.T) {
	s := NewScorer()
	_, ok := s.GetScore("nope")
	if ok {
		t.Error("expected ok=false")
	}
}

// ---------------------------------------------------------------------------
// Scorer: RemoveProvider that doesn't exist (should not panic)
// ---------------------------------------------------------------------------

func TestScorerRemoveProvider_NonExistent(t *testing.T) {
	s := NewScorer()
	// Should not panic
	s.RemoveProvider("nope")
}

// ---------------------------------------------------------------------------
// Scorer: Normalization bounds update
// ---------------------------------------------------------------------------

func TestScorerNormalizationBoundsUpdate(t *testing.T) {
	s := NewScorer()

	// First: within default bounds
	s.UpdateProviderMetrics("p1", 30, 100, 500, 1.0)

	// Second: exceeds default bounds
	score := s.UpdateProviderMetrics("p2", 1000, 10000, 60000, 20.0)

	// The score should still compute without errors
	if score.CompositeScore < 0 {
		t.Errorf("unexpected negative composite: %f", score.CompositeScore)
	}
}

// ---------------------------------------------------------------------------
// Scorer: Request latency scoring
// ---------------------------------------------------------------------------

func TestScorerRequestLatencyScoring(t *testing.T) {
	s := NewScorer()

	// Zero latency = 100
	score0 := s.UpdateProviderMetrics("fast", 30, 100, 0, 0)
	if score0.RequestLatencyScore != 100 {
		t.Errorf("zero latency should score 100, got %f", score0.RequestLatencyScore)
	}

	// Max latency = 0
	scoreMax := s.UpdateProviderMetrics("slow", 30, 100, 30000, 0)
	if scoreMax.RequestLatencyScore != 0 {
		t.Errorf("max latency should score 0, got %f", scoreMax.RequestLatencyScore)
	}
}

// ---------------------------------------------------------------------------
// clamp helper
// ---------------------------------------------------------------------------

func TestClamp(t *testing.T) {
	tests := []struct {
		value, min, max, want float64
	}{
		{50, 0, 100, 50},
		{-10, 0, 100, 0},
		{150, 0, 100, 100},
		{0, 0, 100, 0},
		{100, 0, 100, 100},
	}

	for _, tt := range tests {
		got := clamp(tt.value, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("clamp(%f, %f, %f) = %f, want %f", tt.value, tt.min, tt.max, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Scorer: Multiple updates for same provider
// ---------------------------------------------------------------------------

func TestScorerMultipleUpdates(t *testing.T) {
	s := NewScorer()

	// First update
	s.UpdateProviderMetrics("p", 30, 1000, 5000, 2.0)

	// Second update should replace
	score := s.UpdateProviderMetrics("p", 70, 500, 2000, 1.0)

	if score.ModelParamsB != 70 {
		t.Errorf("ModelParamsB = %f, want 70", score.ModelParamsB)
	}
	if score.HeartbeatLatencyMs != 500 {
		t.Errorf("HeartbeatLatencyMs = %d, want 500", score.HeartbeatLatencyMs)
	}

	// Composite should reflect improved metrics
	if score.CompositeScore <= 0 {
		t.Error("expected positive composite after update")
	}
}

// ---------------------------------------------------------------------------
// Scorer: Composite calculation verifies weight contributions
// ---------------------------------------------------------------------------

func TestScorerCompositeCalculation(t *testing.T) {
	s := NewScorer()

	// All zeros except model size
	s.SetWeights(ScoringWeights{
		ModelSize:      100,
		RoundTrip:      0,
		RequestLatency: 0,
		Cost:           0,
	})

	score := s.UpdateProviderMetrics("p", 500, 0, 0, 0)

	// With only model size weight, composite should equal model size weight * score/100
	expectedComposite := 100 * (score.ModelSizeScore / 100)
	if score.CompositeScore != expectedComposite {
		t.Errorf("composite = %f, want %f", score.CompositeScore, expectedComposite)
	}
}

// ---------------------------------------------------------------------------
// RequiredModelTier default case
// ---------------------------------------------------------------------------

func TestRequiredModelTier_Default(t *testing.T) {
	// An unknown complexity level should default to TierMedium
	tier := RequiredModelTier(ComplexityLevel(99))
	if tier != TierMedium {
		t.Errorf("expected TierMedium for unknown complexity, got %d", tier)
	}
}
