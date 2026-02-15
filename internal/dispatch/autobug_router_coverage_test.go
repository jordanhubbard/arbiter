package dispatch

import (
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

// --- AnalyzeBugForRouting comprehensive tests ---

func TestAnalyzeBugForRouting_AllRoutes(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name         string
		bead         *models.Bead
		shouldRoute  bool
		personaHint  string
		hasUpdated   bool
		routeReason  string
	}{
		{
			name: "build error via title",
			bead: &models.Bead{
				Title:       "[auto-filed] build failed in production",
				Description: "Exit code 1",
				Tags:        []string{"auto-filed"},
			},
			shouldRoute: true,
			personaHint: "devops-engineer",
			hasUpdated:  true,
		},
		{
			name: "frontend JS error via title",
			bead: &models.Bead{
				Title:       "[auto-filed] TypeError in component",
				Description: "Cannot read property",
				Tags:        []string{"auto-filed"},
			},
			shouldRoute: true,
			personaHint: "web-designer",
			hasUpdated:  true,
		},
		{
			name: "backend Go error via title",
			bead: &models.Bead{
				Title:       "[auto-filed] panic: nil pointer",
				Description: "goroutine 1",
				Tags:        []string{"auto-filed"},
			},
			shouldRoute: true,
			personaHint: "backend-engineer",
			hasUpdated:  true,
		},
		{
			name: "API error via description",
			bead: &models.Bead{
				Title:       "[auto-filed] Server error",
				Description: "HTTP status code 500 returned from api endpoint",
				Tags:        []string{"auto-filed"},
			},
			shouldRoute: true,
			personaHint: "backend-engineer",
			hasUpdated:  true,
		},
		{
			name: "database error via title",
			bead: &models.Bead{
				Title:       "[auto-filed] database connection refused",
				Description: "Cannot connect to postgres",
				Tags:        []string{"auto-filed"},
			},
			shouldRoute: true,
			personaHint: "backend-engineer",
			hasUpdated:  true,
		},
		{
			name: "CSS error via tags",
			bead: &models.Bead{
				Title:       "[auto-filed] Visual glitch",
				Description: "Something looks wrong",
				Tags:        []string{"auto-filed", "css"},
			},
			shouldRoute: true,
			personaHint: "web-designer",
			hasUpdated:  true,
		},
		{
			name: "unclear bug type",
			bead: &models.Bead{
				Title:       "[auto-filed] Unknown issue",
				Description: "Something happened",
				Tags:        []string{"auto-filed"},
			},
			shouldRoute: false,
			personaHint: "",
			hasUpdated:  false,
			routeReason: "Bug type unclear, needs QA triage",
		},
		{
			name: "not auto-filed",
			bead: &models.Bead{
				Title:       "Manual bug report",
				Description: "User reported issue",
				Tags:        []string{"bug"},
			},
			shouldRoute: false,
		},
		{
			name: "already has persona hint",
			bead: &models.Bead{
				Title:       "[backend-engineer] [auto-filed] Fix API",
				Description: "Already triaged",
				Tags:        []string{"auto-filed"},
			},
			shouldRoute: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := router.AnalyzeBugForRouting(tt.bead)
			if info.ShouldRoute != tt.shouldRoute {
				t.Errorf("ShouldRoute = %v, want %v", info.ShouldRoute, tt.shouldRoute)
			}
			if tt.shouldRoute {
				if info.PersonaHint != tt.personaHint {
					t.Errorf("PersonaHint = %q, want %q", info.PersonaHint, tt.personaHint)
				}
				if tt.hasUpdated && info.UpdatedTitle == "" {
					t.Error("Expected UpdatedTitle to be set")
				}
				if info.RoutingReason == "" {
					t.Error("Expected RoutingReason to be set")
				}
			}
			if tt.routeReason != "" && info.RoutingReason != tt.routeReason {
				t.Errorf("RoutingReason = %q, want %q", info.RoutingReason, tt.routeReason)
			}
		})
	}
}

// --- isAutoFiledBug comprehensive ---

func TestIsAutoFiledBug_AllPatterns(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		bead     *models.Bead
		expected bool
	}{
		{
			name: "in title lowercase",
			bead: &models.Bead{
				Title: "[auto-filed] bug",
				Tags:  []string{},
			},
			expected: true,
		},
		{
			name: "in title uppercase",
			bead: &models.Bead{
				Title: "[AUTO-FILED] bug",
				Tags:  []string{},
			},
			expected: true,
		},
		{
			name: "in title mixed case",
			bead: &models.Bead{
				Title: "[Auto-Filed] BUG",
				Tags:  []string{},
			},
			expected: true,
		},
		{
			name: "in tags only",
			bead: &models.Bead{
				Title: "Bug report",
				Tags:  []string{"auto-filed"},
			},
			expected: true,
		},
		{
			name: "in tags uppercase",
			bead: &models.Bead{
				Title: "Bug report",
				Tags:  []string{"AUTO-FILED"},
			},
			expected: true,
		},
		{
			name: "not present",
			bead: &models.Bead{
				Title: "Normal bug",
				Tags:  []string{"bug", "frontend"},
			},
			expected: false,
		},
		{
			name: "partial match in title - should match (contains)",
			bead: &models.Bead{
				Title: "This is an [auto-filed] error in production",
				Tags:  []string{},
			},
			expected: true,
		},
		{
			name: "empty title and tags",
			bead: &models.Bead{
				Title: "",
				Tags:  []string{},
			},
			expected: false,
		},
		{
			name: "nil tags",
			bead: &models.Bead{
				Title: "Normal bug",
				Tags:  nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isAutoFiledBug(tt.bead)
			if result != tt.expected {
				t.Errorf("isAutoFiledBug() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// --- hasPersonaHint comprehensive ---

func TestHasPersonaHint_AllPersonas(t *testing.T) {
	router := NewAutoBugRouter()

	personas := []string{"web-designer", "backend-engineer", "devops-engineer", "qa-engineer", "ceo", "cfo"}

	for _, persona := range personas {
		t.Run(persona, func(t *testing.T) {
			bead := &models.Bead{
				Title: "[" + persona + "] Some bug",
			}
			if !router.hasPersonaHint(bead) {
				t.Errorf("Expected hasPersonaHint=true for [%s]", persona)
			}
		})
	}
}

func TestHasPersonaHint_Unknown(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		Title: "[unknown-role] Some bug",
	}
	if router.hasPersonaHint(bead) {
		t.Error("Expected hasPersonaHint=false for unknown persona")
	}
}

func TestHasPersonaHint_NoSquareBrackets(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		Title: "web-designer Fix bug",
	}
	if router.hasPersonaHint(bead) {
		t.Error("Expected hasPersonaHint=false without square brackets")
	}
}

// --- Detection method tests with tag-only triggers ---

func TestIsFrontendJSError_TagOnly(t *testing.T) {
	router := NewAutoBugRouter()

	tagTests := []struct {
		name string
		tags map[string]bool
	}{
		{"frontend tag", map[string]bool{"frontend": true}},
		{"javascript tag", map[string]bool{"javascript": true}},
		{"js_error tag", map[string]bool{"js_error": true}},
	}

	for _, tt := range tagTests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isFrontendJSError("generic error", "generic desc", tt.tags)
			if !result {
				t.Errorf("Expected true for %s", tt.name)
			}
		})
	}
}

func TestIsBackendGoError_TagOnly(t *testing.T) {
	router := NewAutoBugRouter()

	tagTests := []struct {
		name string
		tags map[string]bool
	}{
		{"backend tag", map[string]bool{"backend": true}},
		{"golang tag", map[string]bool{"golang": true}},
		{"go_error tag", map[string]bool{"go_error": true}},
	}

	for _, tt := range tagTests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isBackendGoError("generic error", "generic desc", tt.tags)
			if !result {
				t.Errorf("Expected true for %s", tt.name)
			}
		})
	}
}

func TestIsAPIError_TagOnly(t *testing.T) {
	router := NewAutoBugRouter()

	tagTests := []struct {
		name string
		tags map[string]bool
	}{
		{"api tag", map[string]bool{"api": true}},
		{"api_error tag", map[string]bool{"api_error": true}},
		{"http tag", map[string]bool{"http": true}},
	}

	for _, tt := range tagTests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isAPIError("generic error", "generic desc", tt.tags)
			if !result {
				t.Errorf("Expected true for %s", tt.name)
			}
		})
	}
}

func TestIsDatabaseError_TagOnly(t *testing.T) {
	router := NewAutoBugRouter()

	tagTests := []struct {
		name string
		tags map[string]bool
	}{
		{"database tag", map[string]bool{"database": true}},
		{"sql tag", map[string]bool{"sql": true}},
		{"db_error tag", map[string]bool{"db_error": true}},
	}

	for _, tt := range tagTests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isDatabaseError("generic error", "generic desc", tt.tags)
			if !result {
				t.Errorf("Expected true for %s", tt.name)
			}
		})
	}
}

func TestIsBuildError_TagOnly(t *testing.T) {
	router := NewAutoBugRouter()

	tagTests := []struct {
		name string
		tags map[string]bool
	}{
		{"build tag", map[string]bool{"build": true}},
		{"deployment tag", map[string]bool{"deployment": true}},
		{"docker tag", map[string]bool{"docker": true}},
	}

	for _, tt := range tagTests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isBuildError("generic error", "generic desc", tt.tags)
			if !result {
				t.Errorf("Expected true for %s", tt.name)
			}
		})
	}
}

func TestIsStylingError_TagOnly(t *testing.T) {
	router := NewAutoBugRouter()

	tagTests := []struct {
		name string
		tags map[string]bool
	}{
		{"css tag", map[string]bool{"css": true}},
		{"styling tag", map[string]bool{"styling": true}},
		{"ui tag", map[string]bool{"ui": true}},
	}

	for _, tt := range tagTests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isStylingError("generic error", "generic desc", tt.tags)
			if !result {
				t.Errorf("Expected true for %s", tt.name)
			}
		})
	}
}

// --- Detection method tests with description-only triggers ---

func TestIsFrontendJSError_DescOnly(t *testing.T) {
	router := NewAutoBugRouter()

	descs := []string{
		"javascript error in module",
		"syntaxerror unexpected token",
		"referenceerror: x is not defined",
		"cannot read property of null",
		"cannot access before initialization",
	}

	for _, desc := range descs {
		t.Run(desc, func(t *testing.T) {
			result := router.isFrontendJSError("generic", desc, map[string]bool{})
			if !result {
				t.Errorf("Expected true for desc %q", desc)
			}
		})
	}
}

func TestIsBackendGoError_DescOnly(t *testing.T) {
	router := NewAutoBugRouter()

	descs := []string{
		"runtime error: index out of range",
		"nil pointer dereference in handler",
		"invalid memory address",
		"undefined: myFunc",
		"cannot use string as int",
	}

	for _, desc := range descs {
		t.Run(desc, func(t *testing.T) {
			result := router.isBackendGoError("generic", desc, map[string]bool{})
			if !result {
				t.Errorf("Expected true for desc %q", desc)
			}
		})
	}
}

func TestIsAPIError_DescOnly(t *testing.T) {
	router := NewAutoBugRouter()

	descs := []string{
		"api error in request",
		"api request failed with timeout",
		"got http error",
		"received status code 502",
		"route not found for endpoint",
		"returned 405 method not allowed",
		"server returned 404",
	}

	for _, desc := range descs {
		t.Run(desc, func(t *testing.T) {
			result := router.isAPIError("generic", desc, map[string]bool{})
			if !result {
				t.Errorf("Expected true for desc %q", desc)
			}
		})
	}
}

func TestIsDatabaseError_DescOnly(t *testing.T) {
	router := NewAutoBugRouter()

	descs := []string{
		"database timeout",
		"sql syntax error near",
		"query execution failed",
		"connection refused to postgres",
		"sqlite database is locked",
		"deadlock detected between transactions",
		"constraint violation on insert",
		"foreign key constraint failed",
	}

	for _, desc := range descs {
		t.Run(desc, func(t *testing.T) {
			result := router.isDatabaseError("generic", desc, map[string]bool{})
			if !result {
				t.Errorf("Expected true for desc %q", desc)
			}
		})
	}
}

func TestIsBuildError_DescOnly(t *testing.T) {
	router := NewAutoBugRouter()

	descs := []string{
		"docker image failed to build",
		"dockerfile has syntax error",
		"compile error in module",
		"deployment failed with error",
		"makefile target not found",
		"ci/cd pipeline failed",
		"container crashed on startup",
	}

	for _, desc := range descs {
		t.Run(desc, func(t *testing.T) {
			result := router.isBuildError("generic", desc, map[string]bool{})
			if !result {
				t.Errorf("Expected true for desc %q", desc)
			}
		})
	}
}

func TestIsStylingError_DescOnly(t *testing.T) {
	router := NewAutoBugRouter()

	descs := []string{
		"css not loading",
		"style sheet missing",
		"layout shifted after update",
		"rendering incorrectly on mobile",
		"display none not working",
		"flexbox alignment issue",
		"grid columns overlapping",
		"responsive breakpoint broken",
	}

	for _, desc := range descs {
		t.Run(desc, func(t *testing.T) {
			result := router.isStylingError("generic", desc, map[string]bool{})
			if !result {
				t.Errorf("Expected true for desc %q", desc)
			}
		})
	}
}

// --- No match cases for all detection methods ---

func TestDetectionMethods_NoMatch(t *testing.T) {
	router := NewAutoBugRouter()
	emptyTags := map[string]bool{}

	if router.isFrontendJSError("generic title", "generic desc", emptyTags) {
		t.Error("Expected no frontend JS error for generic text")
	}
	if router.isBackendGoError("generic title", "generic desc", emptyTags) {
		t.Error("Expected no backend Go error for generic text")
	}
	if router.isAPIError("generic title", "generic desc", emptyTags) {
		t.Error("Expected no API error for generic text")
	}
	if router.isDatabaseError("generic title", "generic desc", emptyTags) {
		t.Error("Expected no database error for generic text")
	}
	if router.isBuildError("generic title", "generic desc", emptyTags) {
		t.Error("Expected no build error for generic text")
	}
	if router.isStylingError("generic title", "generic desc", emptyTags) {
		t.Error("Expected no styling error for generic text")
	}
}

// --- Updated title format tests ---

func TestAnalyzeBugForRouting_UpdatedTitleFormat(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		Title:       "[auto-filed] JavaScript TypeError",
		Description: "typeerror in the frontend",
		Tags:        []string{"auto-filed"},
	}

	info := router.AnalyzeBugForRouting(bead)
	if !info.ShouldRoute {
		t.Fatal("Expected ShouldRoute to be true")
	}

	// Updated title should have persona hint prepended
	expected := "[web-designer] [auto-filed] JavaScript TypeError"
	if info.UpdatedTitle != expected {
		t.Errorf("UpdatedTitle = %q, want %q", info.UpdatedTitle, expected)
	}
}

func TestAnalyzeBugForRouting_BuildErrorTitle(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		Title:       "[auto-filed] Docker build failed",
		Description: "Exit code 1",
		Tags:        []string{"auto-filed"},
	}

	info := router.AnalyzeBugForRouting(bead)
	expected := "[devops-engineer] [auto-filed] Docker build failed"
	if info.UpdatedTitle != expected {
		t.Errorf("UpdatedTitle = %q, want %q", info.UpdatedTitle, expected)
	}
}

func TestAnalyzeBugForRouting_BackendErrorTitle(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		Title:       "[auto-filed] nil pointer dereference",
		Description: "panic in handler",
		Tags:        []string{"auto-filed"},
	}

	info := router.AnalyzeBugForRouting(bead)
	expected := "[backend-engineer] [auto-filed] nil pointer dereference"
	if info.UpdatedTitle != expected {
		t.Errorf("UpdatedTitle = %q, want %q", info.UpdatedTitle, expected)
	}
}

// --- getTagsLower edge cases ---

func TestGetTagsLower_SpecialChars(t *testing.T) {
	router := NewAutoBugRouter()

	bead := &models.Bead{
		Tags: []string{"Tag-With-Hyphens", "TAG_WITH_UNDERSCORES", "  spaced  ", "UPPER"},
	}

	tags := router.getTagsLower(bead)
	if !tags["tag-with-hyphens"] {
		t.Error("Expected 'tag-with-hyphens' in lowered tags")
	}
	if !tags["tag_with_underscores"] {
		t.Error("Expected 'tag_with_underscores' in lowered tags")
	}
	if !tags["  spaced  "] {
		t.Error("Expected '  spaced  ' in lowered tags (preserves spaces)")
	}
	if !tags["upper"] {
		t.Error("Expected 'upper' in lowered tags")
	}
}

// --- Priority/routing order tests ---

func TestRoutingPriority_BuildOverBackend(t *testing.T) {
	router := NewAutoBugRouter()

	// Both build and backend indicators present - build should win (checked first)
	bead := &models.Bead{
		Title:       "[auto-filed] go build panic in docker container",
		Description: "compilation error during docker build",
		Tags:        []string{"auto-filed"},
	}

	info := router.AnalyzeBugForRouting(bead)
	if !info.ShouldRoute {
		t.Fatal("Expected ShouldRoute=true")
	}
	if info.PersonaHint != "devops-engineer" {
		t.Errorf("Expected devops-engineer (build priority), got %q", info.PersonaHint)
	}
}

func TestRoutingPriority_FrontendOverAPI(t *testing.T) {
	router := NewAutoBugRouter()

	// Frontend JS indicator takes priority over API
	bead := &models.Bead{
		Title:       "[auto-filed] TypeError in api call",
		Description: "uncaught typeerror when calling endpoint",
		Tags:        []string{"auto-filed"},
	}

	info := router.AnalyzeBugForRouting(bead)
	if !info.ShouldRoute {
		t.Fatal("Expected ShouldRoute=true")
	}
	// Frontend JS checked before API, so should match web-designer
	if info.PersonaHint != "web-designer" {
		t.Errorf("Expected web-designer (frontend priority), got %q", info.PersonaHint)
	}
}
