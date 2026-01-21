package modelcatalog

import (
	"math"
	"testing"

	internalmodels "github.com/jordanhubbard/agenticorp/internal/models"
)

// TestParseModelName tests extraction of metadata from model names
func TestParseModelName(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedVendor     string
		expectedFamily     string
		expectedTotalB     float64
		expectedActiveB    float64
		expectedPrecision  string
		expectedInstruct   bool
	}{
		{
			name:              "Qwen 480B model with active params",
			input:             "Qwen/Qwen3-Coder-480B-A35B-Instruct",
			expectedVendor:    "Qwen",
			expectedFamily:    "Qwen3",
			expectedTotalB:    480,
			expectedActiveB:   35,
			expectedPrecision: "",
			expectedInstruct:  true,
		},
		{
			name:              "Qwen 30B model with active params",
			input:             "Qwen/Qwen3-Coder-30B-A3B-Instruct",
			expectedVendor:    "Qwen",
			expectedFamily:    "Qwen3",
			expectedTotalB:    30,
			expectedActiveB:   3,
			expectedPrecision: "",
			expectedInstruct:  true,
		},
		{
			name:              "NVIDIA Nemotron with precision",
			input:             "nvidia/NVIDIA-Nemotron-3-Nano-30B-A3B-FP8",
			expectedVendor:    "nvidia",
			expectedFamily:    "NVIDIA",
			expectedTotalB:    30,
			expectedActiveB:   3,
			expectedPrecision: "FP8",
			expectedInstruct:  false,
		},
		{
			name:              "Qwen2.5 without vendor prefix",
			input:             "Qwen2.5-Coder-32B-Instruct",
			expectedVendor:    "",
			expectedFamily:    "Qwen2.5",
			expectedTotalB:    32,
			expectedActiveB:   0,
			expectedPrecision: "",
			expectedInstruct:  true,
		},
		{
			name:              "Qwen2.5 7B model",
			input:             "Qwen2.5-Coder-7B-Instruct",
			expectedVendor:    "",
			expectedFamily:    "Qwen2.5",
			expectedTotalB:    7,
			expectedActiveB:   0,
			expectedPrecision: "",
			expectedInstruct:  true,
		},
		{
			name:              "Model with FP16 precision",
			input:             "TestModel-13B-FP16-Instruct",
			expectedVendor:    "",
			expectedFamily:    "TestModel",
			expectedTotalB:    13,
			expectedActiveB:   0,
			expectedPrecision: "FP16",
			expectedInstruct:  true,
		},
		{
			name:              "Model with INT4 precision",
			input:             "TestModel-7B-INT4",
			expectedVendor:    "",
			expectedFamily:    "TestModel",
			expectedTotalB:    7,
			expectedActiveB:   0,
			expectedPrecision: "INT4",
			expectedInstruct:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := ParseModelName(tt.input)

			if spec.Name != tt.input {
				t.Errorf("Name = %v, want %v", spec.Name, tt.input)
			}
			if spec.Vendor != tt.expectedVendor {
				t.Errorf("Vendor = %v, want %v", spec.Vendor, tt.expectedVendor)
			}
			if spec.Family != tt.expectedFamily {
				t.Errorf("Family = %v, want %v", spec.Family, tt.expectedFamily)
			}
			if spec.TotalParamsB != tt.expectedTotalB {
				t.Errorf("TotalParamsB = %v, want %v", spec.TotalParamsB, tt.expectedTotalB)
			}
			if spec.ActiveParamsB != tt.expectedActiveB {
				t.Errorf("ActiveParamsB = %v, want %v", spec.ActiveParamsB, tt.expectedActiveB)
			}
			if spec.Precision != tt.expectedPrecision {
				t.Errorf("Precision = %v, want %v", spec.Precision, tt.expectedPrecision)
			}
			if spec.Instruct != tt.expectedInstruct {
				t.Errorf("Instruct = %v, want %v", spec.Instruct, tt.expectedInstruct)
			}
		})
	}
}

// TestScore tests the scoring logic with deterministic inputs
func TestScore(t *testing.T) {
	catalog := DefaultCatalog()

	tests := []struct {
		name          string
		spec          internalmodels.ModelSpec
		expectedScore float64
		tolerance     float64
	}{
		{
			name: "Fast interactivity bonus",
			spec: internalmodels.ModelSpec{
				Interactivity: "fast",
				TotalParamsB:  10,
				Rank:          1,
			},
			expectedScore: 98.0,  // 100 + 10 (fast) - 10 (log10(11)*10) - 1 (rank)
			tolerance:     1.0,
		},
		{
			name: "Medium interactivity bonus",
			spec: internalmodels.ModelSpec{
				Interactivity: "medium",
				TotalParamsB:  10,
				Rank:          1,
			},
			expectedScore: 93.0,  // 100 + 5 (medium) - 10 (log10(11)*10) - 1 (rank)
			tolerance:     1.0,
		},
		{
			name: "Slow interactivity penalty",
			spec: internalmodels.ModelSpec{
				Interactivity: "slow",
				TotalParamsB:  10,
				Rank:          1,
			},
			expectedScore: 83.0,  // 100 - 5 (slow) - 10 (log10(11)*10) - 1 (rank)
			tolerance:     1.0,
		},
		{
			name: "Large model penalty",
			spec: internalmodels.ModelSpec{
				Interactivity: "fast",
				TotalParamsB:  480,
				Rank:          1,
			},
			expectedScore: 82.0,  // 100 + 10 (fast) - ~27 (log10(481)*10) - 1 (rank)
			tolerance:     1.0,
		},
		{
			name: "Small model bonus",
			spec: internalmodels.ModelSpec{
				Interactivity: "fast",
				TotalParamsB:  7,
				Rank:          5,
			},
			expectedScore: 95.97,  // 100 + 10 (fast) - ~9 (log10(8)*10) - 5 (rank)
			tolerance:     0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := catalog.Score(tt.spec)
			diff := math.Abs(score - tt.expectedScore)
			if diff > tt.tolerance {
				t.Errorf("Score = %.2f, want %.2f ± %.2f (diff: %.2f)",
					score, tt.expectedScore, tt.tolerance, diff)
			}
		})
	}
}

// TestScoreRanking tests that scoring produces correct relative ranking
func TestScoreRanking(t *testing.T) {
	catalog := DefaultCatalog()

	// Fast small model should score higher than slow large model
	fastSmall := internalmodels.ModelSpec{
		Interactivity: "fast",
		TotalParamsB:  7,
		Rank:          5,
	}
	slowLarge := internalmodels.ModelSpec{
		Interactivity: "slow",
		TotalParamsB:  480,
		Rank:          1,
	}

	scoreFastSmall := catalog.Score(fastSmall)
	scoreSlowLarge := catalog.Score(slowLarge)

	if scoreFastSmall <= scoreSlowLarge {
		t.Errorf("Fast small model (%.2f) should score higher than slow large model (%.2f)",
			scoreFastSmall, scoreSlowLarge)
	}
}

// TestSelectBest tests negotiation/selection logic
func TestSelectBest(t *testing.T) {
	catalog := DefaultCatalog()

	tests := []struct {
		name          string
		available     []string
		expectedModel string
		shouldFind    bool
	}{
		{
			name: "Single recommended model available",
			available: []string{
				"Qwen/Qwen3-Coder-30B-A3B-Instruct",
			},
			expectedModel: "Qwen/Qwen3-Coder-30B-A3B-Instruct",
			shouldFind:    true,
		},
		{
			name: "Multiple recommended models - picks highest score",
			available: []string{
				"Qwen2.5-Coder-7B-Instruct",          // Rank 5 but high score (fast+small)
				"Qwen/Qwen3-Coder-30B-A3B-Instruct",  // Rank 2, medium score
				"nvidia/NVIDIA-Nemotron-3-Nano-30B-A3B-FP8", // Rank 3, good score (fast)
			},
			expectedModel: "Qwen2.5-Coder-7B-Instruct",  // Highest score wins (fast+small beats rank)
			shouldFind:    true,
		},
		{
			name: "Mix of recommended and non-recommended models",
			available: []string{
				"unknown-model-1",
				"Qwen/Qwen3-Coder-30B-A3B-Instruct",
				"unknown-model-2",
			},
			expectedModel: "Qwen/Qwen3-Coder-30B-A3B-Instruct",
			shouldFind:    true,
		},
		{
			name: "Only non-recommended models available",
			available: []string{
				"unknown-model-1",
				"unknown-model-2",
			},
			expectedModel: "",
			shouldFind:    false,
		},
		{
			name:          "No models available",
			available:     []string{},
			expectedModel: "",
			shouldFind:    false,
		},
		{
			name: "Case insensitive matching",
			available: []string{
				"qwen/qwen3-coder-30b-a3b-instruct",  // lowercase
			},
			expectedModel: "Qwen/Qwen3-Coder-30B-A3B-Instruct",  // Should match despite case difference
			shouldFind:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, score, found := catalog.SelectBest(tt.available)

			if found != tt.shouldFind {
				t.Errorf("SelectBest found = %v, want %v", found, tt.shouldFind)
			}

			if tt.shouldFind {
				if model == nil {
					t.Error("SelectBest returned nil model but found=true")
					return
				}
				if model.Name != tt.expectedModel {
					t.Errorf("SelectBest model = %v, want %v", model.Name, tt.expectedModel)
				}
				if score <= 0 {
					t.Errorf("SelectBest score = %.2f, want > 0", score)
				}
			} else {
				if model != nil {
					t.Errorf("SelectBest model = %v, want nil", model.Name)
				}
				if score != 0 {
					t.Errorf("SelectBest score = %.2f, want 0", score)
				}
			}
		})
	}
}

// TestDefaultCatalog verifies the default catalog contains expected models
func TestDefaultCatalog(t *testing.T) {
	catalog := DefaultCatalog()

	if catalog == nil {
		t.Fatal("DefaultCatalog returned nil")
	}

	models := catalog.List()
	if len(models) == 0 {
		t.Fatal("DefaultCatalog has no models")
	}

	// Verify expected models are present
	expectedModels := []string{
		"Qwen/Qwen3-Coder-480B-A35B-Instruct",
		"Qwen/Qwen3-Coder-30B-A3B-Instruct",
		"nvidia/NVIDIA-Nemotron-3-Nano-30B-A3B-FP8",
	}

	for _, expected := range expectedModels {
		found := false
		for _, model := range models {
			if model.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected model %s not found in default catalog", expected)
		}
	}

	// Verify models are sorted by rank
	for i := 1; i < len(models); i++ {
		if models[i].Rank < models[i-1].Rank {
			t.Errorf("Models not sorted by rank: %d comes before %d",
				models[i].Rank, models[i-1].Rank)
		}
	}
}

// TestValidate tests catalog validation
func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		catalog     *Catalog
		shouldError bool
	}{
		{
			name:        "Nil catalog",
			catalog:     nil,
			shouldError: true,
		},
		{
			name:        "Empty catalog",
			catalog:     NewCatalog([]internalmodels.ModelSpec{}),
			shouldError: true,
		},
		{
			name: "Valid catalog",
			catalog: NewCatalog([]internalmodels.ModelSpec{
				{Name: "TestModel-7B"},
			}),
			shouldError: false,
		},
		{
			name: "Catalog with empty name",
			catalog: NewCatalog([]internalmodels.ModelSpec{
				{Name: ""},
			}),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.catalog.Validate()
			if tt.shouldError && err == nil {
				t.Error("Validate() expected error but got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

// TestCatalogReplace tests replacing catalog models
func TestCatalogReplace(t *testing.T) {
	catalog := NewCatalog([]internalmodels.ModelSpec{
		{Name: "OldModel-7B"},
	})

	newModels := []internalmodels.ModelSpec{
		{Name: "NewModel-13B-Instruct"},
	}

	catalog.Replace(newModels)
	models := catalog.List()

	if len(models) != 1 {
		t.Fatalf("Expected 1 model after replace, got %d", len(models))
	}

	if models[0].Name != "NewModel-13B-Instruct" {
		t.Errorf("Expected NewModel-13B-Instruct, got %s", models[0].Name)
	}

	// Verify parsing happened during replace
	if models[0].TotalParamsB != 13 {
		t.Errorf("Expected TotalParamsB=13 after replace, got %.0f", models[0].TotalParamsB)
	}
	if !models[0].Instruct {
		t.Error("Expected Instruct=true after replace")
	}
}
