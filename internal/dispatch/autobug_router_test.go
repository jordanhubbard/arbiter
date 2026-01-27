package dispatch

import (
	"testing"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

func TestAutoBugRouter_FrontendJSError(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		ID:          "ac-001",
		Title:       "[auto-filed] [frontend] UI Error: ReferenceError: apiCall is not defined",
		Description: "JavaScript error occurred in the UI",
		Tags:        []string{"auto-filed", "frontend", "js_error"},
		Type:        "bug",
		Priority:    models.BeadPriority(1),
	}

	info := router.AnalyzeBugForRouting(bead)

	if !info.ShouldRoute {
		t.Error("Expected ShouldRoute to be true for frontend JS error")
	}

	if info.PersonaHint != "web-designer" {
		t.Errorf("Expected PersonaHint 'web-designer', got '%s'", info.PersonaHint)
	}

	if info.UpdatedTitle != "[web-designer] [auto-filed] [frontend] UI Error: ReferenceError: apiCall is not defined" {
		t.Errorf("UpdatedTitle mismatch: got '%s'", info.UpdatedTitle)
	}

	if info.RoutingReason == "" {
		t.Error("Expected RoutingReason to be set")
	}
}

func TestAutoBugRouter_BackendGoError(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		ID:          "bd-002",
		Title:       "[auto-filed] [backend] Runtime Error: panic in handler",
		Description: "Go panic occurred: runtime error: invalid memory address",
		Tags:        []string{"auto-filed", "backend"},
		Type:        "bug",
		Priority:    models.BeadPriority(0),
	}

	info := router.AnalyzeBugForRouting(bead)

	if !info.ShouldRoute {
		t.Error("Expected ShouldRoute to be true for backend Go error")
	}

	if info.PersonaHint != "backend-engineer" {
		t.Errorf("Expected PersonaHint 'backend-engineer', got '%s'", info.PersonaHint)
	}
}

func TestAutoBugRouter_BuildError(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		ID:          "bd-003",
		Title:       "[auto-filed] Build failed - Docker compilation error",
		Description: "Docker build failed with exit code 1",
		Tags:        []string{"auto-filed", "build", "docker"},
		Type:        "bug",
		Priority:    models.BeadPriority(0),
	}

	info := router.AnalyzeBugForRouting(bead)

	if !info.ShouldRoute {
		t.Error("Expected ShouldRoute to be true for build error")
	}

	if info.PersonaHint != "devops-engineer" {
		t.Errorf("Expected PersonaHint 'devops-engineer', got '%s'", info.PersonaHint)
	}
}

func TestAutoBugRouter_APIError(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		ID:          "ac-004",
		Title:       "[auto-filed] [frontend] API Error: Error: API request failed",
		Description: "HTTP 500 error from /api/v1/beads endpoint",
		Tags:        []string{"auto-filed", "api_error"},
		Type:        "bug",
		Priority:    models.BeadPriority(1),
	}

	info := router.AnalyzeBugForRouting(bead)

	if !info.ShouldRoute {
		t.Error("Expected ShouldRoute to be true for API error")
	}

	if info.PersonaHint != "backend-engineer" {
		t.Errorf("Expected PersonaHint 'backend-engineer', got '%s'", info.PersonaHint)
	}
}

func TestAutoBugRouter_NonAutoFiledBug(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		ID:          "bd-005",
		Title:       "Regular bug report",
		Description: "This is not an auto-filed bug",
		Tags:        []string{"bug"},
		Type:        "bug",
		Priority:    models.BeadPriority(2),
	}

	info := router.AnalyzeBugForRouting(bead)

	if info.ShouldRoute {
		t.Error("Expected ShouldRoute to be false for non-auto-filed bug")
	}
}

func TestAutoBugRouter_AlreadyHasPersonaHint(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		ID:          "ac-006",
		Title:       "[backend-engineer] [auto-filed] Already triaged bug",
		Description: "This bug already has a persona hint",
		Tags:        []string{"auto-filed", "backend"},
		Type:        "bug",
		Priority:    models.BeadPriority(1),
	}

	info := router.AnalyzeBugForRouting(bead)

	if info.ShouldRoute {
		t.Error("Expected ShouldRoute to be false for bug with existing persona hint")
	}
}

func TestAutoBugRouter_UnclearBugType(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		ID:          "ac-007",
		Title:       "[auto-filed] Generic error",
		Description: "Something went wrong but unclear what",
		Tags:        []string{"auto-filed"},
		Type:        "bug",
		Priority:    models.BeadPriority(2),
	}

	info := router.AnalyzeBugForRouting(bead)

	if info.ShouldRoute {
		t.Error("Expected ShouldRoute to be false for unclear bug type (should stay with QA)")
	}

	if info.RoutingReason == "" {
		t.Error("Expected RoutingReason to explain why bug wasn't routed")
	}
}

func TestAutoBugRouter_IsAutoFiledBug(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		bead     *models.Bead
		expected bool
	}{
		{
			name: "auto-filed in title",
			bead: &models.Bead{
				Title: "[auto-filed] Bug report",
				Tags:  []string{},
			},
			expected: true,
		},
		{
			name: "auto-filed in tags",
			bead: &models.Bead{
				Title: "Bug report",
				Tags:  []string{"auto-filed", "frontend"},
			},
			expected: true,
		},
		{
			name: "not auto-filed",
			bead: &models.Bead{
				Title: "Regular bug",
				Tags:  []string{"bug"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isAutoFiledBug(tt.bead)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAutoBugRouter_DatabaseError(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		ID:          "bd-008",
		Title:       "[auto-filed] Database connection refused",
		Description: "PostgreSQL connection error: connection refused on port 5432",
		Tags:        []string{"auto-filed", "database"},
		Type:        "bug",
		Priority:    models.BeadPriority(0),
	}

	info := router.AnalyzeBugForRouting(bead)

	if !info.ShouldRoute {
		t.Error("Expected ShouldRoute to be true for database error")
	}

	if info.PersonaHint != "backend-engineer" {
		t.Errorf("Expected PersonaHint 'backend-engineer', got '%s'", info.PersonaHint)
	}
}

func TestAutoBugRouter_CSSError(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		ID:          "ac-009",
		Title:       "[auto-filed] [frontend] CSS layout broken",
		Description: "Flexbox layout not rendering correctly",
		Tags:        []string{"auto-filed", "css", "ui"},
		Type:        "bug",
		Priority:    models.BeadPriority(2),
	}

	info := router.AnalyzeBugForRouting(bead)

	if !info.ShouldRoute {
		t.Error("Expected ShouldRoute to be true for CSS error")
	}

	if info.PersonaHint != "web-designer" {
		t.Errorf("Expected PersonaHint 'web-designer', got '%s'", info.PersonaHint)
	}
}
