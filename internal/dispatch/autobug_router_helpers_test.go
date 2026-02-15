package dispatch

import (
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

func TestAutoBugRouter_NilBead(t *testing.T) {
	router := NewAutoBugRouter()
	info := router.AnalyzeBugForRouting(nil)
	if info.ShouldRoute {
		t.Error("Expected ShouldRoute to be false for nil bead")
	}
}

func TestAutoBugRouter_GetTagsLower(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		bead     *models.Bead
		expected map[string]bool
	}{
		{
			name: "mixed case tags",
			bead: &models.Bead{
				Tags: []string{"Frontend", "AUTO-FILED", "Bug"},
			},
			expected: map[string]bool{
				"frontend":   true,
				"auto-filed": true,
				"bug":        true,
			},
		},
		{
			name: "empty tags",
			bead: &models.Bead{
				Tags: []string{},
			},
			expected: map[string]bool{},
		},
		{
			name: "nil tags",
			bead: &models.Bead{
				Tags: nil,
			},
			expected: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.getTagsLower(tt.bead)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tags, got %d", len(tt.expected), len(result))
			}
			for key, val := range tt.expected {
				if result[key] != val {
					t.Errorf("Expected tag %q=%v, got %v", key, val, result[key])
				}
			}
		})
	}
}

func TestAutoBugRouter_HasPersonaHint(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		bead     *models.Bead
		expected bool
	}{
		{
			name: "has web-designer hint",
			bead: &models.Bead{
				Title: "[web-designer] Fix CSS",
			},
			expected: true,
		},
		{
			name: "has backend-engineer hint",
			bead: &models.Bead{
				Title: "[backend-engineer] Fix API",
			},
			expected: true,
		},
		{
			name: "has devops-engineer hint",
			bead: &models.Bead{
				Title: "[devops-engineer] Fix deploy",
			},
			expected: true,
		},
		{
			name: "has qa-engineer hint",
			bead: &models.Bead{
				Title: "[qa-engineer] Test this",
			},
			expected: true,
		},
		{
			name: "has ceo hint",
			bead: &models.Bead{
				Title: "[ceo] Review strategy",
			},
			expected: true,
		},
		{
			name: "has cfo hint",
			bead: &models.Bead{
				Title: "[cfo] Budget review",
			},
			expected: true,
		},
		{
			name: "no persona hint",
			bead: &models.Bead{
				Title: "[auto-filed] Some bug",
			},
			expected: false,
		},
		{
			name: "case sensitive - uppercase not matched",
			bead: &models.Bead{
				Title: "[WEB-DESIGNER] Fix CSS",
			},
			expected: true, // title is lowercased before comparison
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.hasPersonaHint(tt.bead)
			if result != tt.expected {
				t.Errorf("hasPersonaHint() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAutoBugRouter_IsFrontendJSError(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		title    string
		desc     string
		tags     map[string]bool
		expected bool
	}{
		{
			name:     "typeerror in title",
			title:    "typeerror: cannot read property",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "uncaught in description",
			title:    "error",
			desc:     "uncaught exception in module",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "javascript tag",
			title:    "some error",
			desc:     "some desc",
			tags:     map[string]bool{"javascript": true},
			expected: true,
		},
		{
			name:     "frontend tag",
			title:    "error",
			desc:     "desc",
			tags:     map[string]bool{"frontend": true},
			expected: true,
		},
		{
			name:     "js_error tag",
			title:    "error",
			desc:     "desc",
			tags:     map[string]bool{"js_error": true},
			expected: true,
		},
		{
			name:     "not a function",
			title:    "foo is not a function",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "not defined",
			title:    "bar is not defined",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "ui error",
			title:    "ui error occurred",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "no match",
			title:    "database timeout",
			desc:     "connection refused",
			tags:     map[string]bool{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isFrontendJSError(tt.title, tt.desc, tt.tags)
			if result != tt.expected {
				t.Errorf("isFrontendJSError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAutoBugRouter_IsBackendGoError(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		title    string
		desc     string
		tags     map[string]bool
		expected bool
	}{
		{
			name:     "panic in title",
			title:    "panic: runtime error",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "nil pointer in description",
			title:    "error",
			desc:     "nil pointer dereference",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "go build in title",
			title:    "go build failed",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "backend tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"backend": true},
			expected: true,
		},
		{
			name:     "golang tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"golang": true},
			expected: true,
		},
		{
			name:     "go_error tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"go_error": true},
			expected: true,
		},
		{
			name:     "compilation error",
			title:    "compilation error",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "no match",
			title:    "css layout broken",
			desc:     "flexbox issue",
			tags:     map[string]bool{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isBackendGoError(tt.title, tt.desc, tt.tags)
			if result != tt.expected {
				t.Errorf("isBackendGoError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAutoBugRouter_IsAPIError(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		title    string
		desc     string
		tags     map[string]bool
		expected bool
	}{
		{
			name:     "api error in title",
			title:    "api error occurred",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "status code in description",
			title:    "error",
			desc:     "received status code 503",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "endpoint in title",
			title:    "endpoint not found",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "api tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"api": true},
			expected: true,
		},
		{
			name:     "api_error tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"api_error": true},
			expected: true,
		},
		{
			name:     "http tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"http": true},
			expected: true,
		},
		{
			name:     "500 error",
			title:    "got 500 response",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "no match",
			title:    "css broken",
			desc:     "layout issue",
			tags:     map[string]bool{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isAPIError(tt.title, tt.desc, tt.tags)
			if result != tt.expected {
				t.Errorf("isAPIError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAutoBugRouter_IsDatabaseError(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		title    string
		desc     string
		tags     map[string]bool
		expected bool
	}{
		{
			name:     "database in title",
			title:    "database error",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "sql in description",
			title:    "error",
			desc:     "sql syntax error",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "deadlock",
			title:    "deadlock detected",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "foreign key violation",
			title:    "foreign key constraint failed",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "database tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"database": true},
			expected: true,
		},
		{
			name:     "sql tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"sql": true},
			expected: true,
		},
		{
			name:     "db_error tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"db_error": true},
			expected: true,
		},
		{
			name:     "postgres in description",
			title:    "error",
			desc:     "postgres connection timeout",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "no match",
			title:    "css broken",
			desc:     "layout issue",
			tags:     map[string]bool{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isDatabaseError(tt.title, tt.desc, tt.tags)
			if result != tt.expected {
				t.Errorf("isDatabaseError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAutoBugRouter_IsBuildError(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		title    string
		desc     string
		tags     map[string]bool
		expected bool
	}{
		{
			name:     "build failed in title",
			title:    "build failed",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "docker in description",
			title:    "error",
			desc:     "docker image build error",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "dockerfile in title",
			title:    "dockerfile syntax error",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "pipeline",
			title:    "pipeline failed",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "makefile",
			title:    "makefile error",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "container in desc",
			title:    "error",
			desc:     "container failed to start",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "build tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"build": true},
			expected: true,
		},
		{
			name:     "deployment tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"deployment": true},
			expected: true,
		},
		{
			name:     "docker tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"docker": true},
			expected: true,
		},
		{
			name:     "no match",
			title:    "css broken",
			desc:     "layout issue",
			tags:     map[string]bool{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isBuildError(tt.title, tt.desc, tt.tags)
			if result != tt.expected {
				t.Errorf("isBuildError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAutoBugRouter_IsStylingError(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		title    string
		desc     string
		tags     map[string]bool
		expected bool
	}{
		{
			name:     "css in title",
			title:    "css layout broken",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "flexbox in description",
			title:    "error",
			desc:     "flexbox not working",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "responsive in title",
			title:    "responsive design broken",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "grid in description",
			title:    "error",
			desc:     "css grid layout issue",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "style in title",
			title:    "style not applying",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "layout in title",
			title:    "layout problem",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "rendering in title",
			title:    "rendering issue",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "display in title",
			title:    "display not correct",
			desc:     "",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			name:     "css tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"css": true},
			expected: true,
		},
		{
			name:     "styling tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"styling": true},
			expected: true,
		},
		{
			name:     "ui tag",
			title:    "error",
			desc:     "error",
			tags:     map[string]bool{"ui": true},
			expected: true,
		},
		{
			name:     "no match",
			title:    "api error",
			desc:     "500 server error",
			tags:     map[string]bool{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.isStylingError(tt.title, tt.desc, tt.tags)
			if result != tt.expected {
				t.Errorf("isStylingError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAutoBugRouter_RoutingPriority(t *testing.T) {
	router := NewAutoBugRouter()

	// Build errors should take priority over backend errors
	bead := &models.Bead{
		Title:       "[auto-filed] build failed with panic",
		Description: "go build panicked during docker build",
		Tags:        []string{"auto-filed"},
		Type:        "bug",
	}

	info := router.AnalyzeBugForRouting(bead)
	if !info.ShouldRoute {
		t.Fatal("Expected ShouldRoute to be true")
	}
	if info.PersonaHint != "devops-engineer" {
		t.Errorf("Expected devops-engineer for build error priority, got %q", info.PersonaHint)
	}
}

func TestAutoBugRouter_StylingFallback(t *testing.T) {
	router := NewAutoBugRouter()

	// CSS error that is auto-filed
	bead := &models.Bead{
		Title:       "[auto-filed] CSS rendering issue",
		Description: "The grid layout is not responsive",
		Tags:        []string{"auto-filed", "css"},
		Type:        "bug",
	}

	info := router.AnalyzeBugForRouting(bead)
	if !info.ShouldRoute {
		t.Fatal("Expected ShouldRoute to be true")
	}
	if info.PersonaHint != "web-designer" {
		t.Errorf("Expected web-designer for CSS error, got %q", info.PersonaHint)
	}
}

func TestNewAutoBugRouter(t *testing.T) {
	router := NewAutoBugRouter()
	if router == nil {
		t.Fatal("Expected non-nil router")
	}
}

func TestAutoBugRouter_IsAutoFiledBug_CaseInsensitive(t *testing.T) {
	router := NewAutoBugRouter()

	tests := []struct {
		name     string
		bead     *models.Bead
		expected bool
	}{
		{
			name: "lowercase auto-filed in title",
			bead: &models.Bead{
				Title: "[auto-filed] bug",
			},
			expected: true,
		},
		{
			name: "mixed case auto-filed in title",
			bead: &models.Bead{
				Title: "[Auto-Filed] bug",
			},
			expected: true,
		},
		{
			name: "uppercase auto-filed in tags",
			bead: &models.Bead{
				Title: "bug",
				Tags:  []string{"Auto-Filed"},
			},
			expected: true,
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
