package dispatch

import (
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

func TestNewPersonaMatcher(t *testing.T) {
	pm := NewPersonaMatcher()
	if pm == nil {
		t.Fatal("Expected PersonaMatcher to be created")
	}
	if len(pm.patterns) == 0 {
		t.Error("Expected PersonaMatcher to have patterns")
	}
}

func TestExtractPersonaHint(t *testing.T) {
	pm := NewPersonaMatcher()

	tests := []struct {
		name     string
		bead     *models.Bead
		expected string
	}{
		{
			name:     "nil bead returns empty",
			bead:     nil,
			expected: "",
		},
		{
			name: "no hint returns empty",
			bead: &models.Bead{
				Title:       "Fix login bug",
				Description: "Users cannot login after update",
			},
			expected: "",
		},
		{
			name: "hint in title - ask the pattern",
			bead: &models.Bead{
				Title:       "ask the backend-engineer to fix this",
				Description: "",
			},
			expected: "backend-engineer",
		},
		{
			name: "hint in title - bracket pattern",
			bead: &models.Bead{
				Title:       "[web-designer] Fix CSS layout",
				Description: "",
			},
			expected: "web-designer",
		},
		{
			name: "hint in title - colon pattern",
			bead: &models.Bead{
				Title:       "backend-engineer: Fix API endpoint",
				Description: "",
			},
			expected: "backend-engineer",
		},
		{
			name: "hint in description when title has no hint",
			bead: &models.Bead{
				Title:       "Fix an issue",
				Description: "ask the devops-engineer to deploy",
			},
			expected: "devops-engineer",
		},
		{
			name: "hint with 'for' pattern",
			bead: &models.Bead{
				Title:       "for qa-engineer: review this",
				Description: "",
			},
			expected: "for-qa-engineer", // colon pattern matches first
		},
		{
			name: "hint with 'for agent' pattern",
			bead: &models.Bead{
				Title:       "for backend agent: fix this",
				Description: "",
			},
			expected: "for-backend", // colon pattern matches first, strips "agent" suffix
		},
		{
			name: "hint with markdown bold pattern",
			bead: &models.Bead{
				Title:       "**FOR web-designer AGENT** Fix layout",
				Description: "",
			},
			expected: "web-designer",
		},
		{
			name: "persona tag with for- prefix",
			bead: &models.Bead{
				Title:       "Some task",
				Description: "Some description",
				Tags:        []string{"persona-for-ceo"},
			},
			expected: "", // parts[0]="persona" not "for", so no match
		},
		{
			name: "persona tag with -only suffix",
			bead: &models.Bead{
				Title:       "Some task",
				Description: "Some description",
				Tags:        []string{"persona-web-designer-only"},
			},
			expected: "persona-web-designer", // includes "persona" prefix in join
		},
		{
			name: "title hint takes priority over description",
			bead: &models.Bead{
				Title:       "[ceo] Review strategy",
				Description: "ask the backend-engineer to do something",
			},
			expected: "ceo",
		},
		{
			name: "no tags with persona prefix",
			bead: &models.Bead{
				Title:       "Some task",
				Description: "Some description",
				Tags:        []string{"bug", "frontend"},
			},
			expected: "",
		},
		{
			name: "empty title and description",
			bead: &models.Bead{
				Title:       "",
				Description: "",
			},
			expected: "",
		},
		{
			name: "agent suffix stripped from hint",
			bead: &models.Bead{
				Title:       "ask the backend engineer agent to fix this",
				Description: "",
			},
			expected: "backend-engineer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pm.ExtractPersonaHint(tt.bead)
			if result != tt.expected {
				t.Errorf("ExtractPersonaHint() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractFromText(t *testing.T) {
	pm := NewPersonaMatcher()

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "empty string",
			text:     "",
			expected: "",
		},
		{
			name:     "whitespace only",
			text:     "   ",
			expected: "",
		},
		{
			name:     "ask the pattern",
			text:     "ask the web-designer to fix the CSS",
			expected: "web-designer",
		},
		{
			name:     "bracket pattern",
			text:     "[backend-engineer] Fix API",
			expected: "backend-engineer",
		},
		{
			name:     "bracket with colon pattern",
			text:     "[ceo]: Review this",
			expected: "ceo",
		},
		{
			name:     "colon pattern",
			text:     "qa-engineer: Test this feature",
			expected: "qa-engineer",
		},
		{
			name:     "for persona pattern",
			text:     "for devops-engineer: deploy this",
			expected: "for-devops-engineer",
		},
		{
			name:     "for persona agent pattern",
			text:     "for backend agent: handle this",
			expected: "for-backend",
		},
		{
			name:     "markdown bold for pattern",
			text:     "**FOR web-designer AGENT** Fix layout",
			expected: "web-designer",
		},
		{
			name:     "no match",
			text:     "Fix the broken login page",
			expected: "",
		},
		{
			name:     "case insensitive ask",
			text:     "Ask The Backend-Engineer To fix this",
			expected: "backend-engineer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pm.extractFromText(tt.text)
			if result != tt.expected {
				t.Errorf("extractFromText(%q) = %q, want %q", tt.text, result, tt.expected)
			}
		})
	}
}

func TestNormalizePersonaHint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "backend-engineer",
			expected: "backend-engineer",
		},
		{
			name:     "uppercase",
			input:    "BACKEND-ENGINEER",
			expected: "backend-engineer",
		},
		{
			name:     "spaces to hyphens",
			input:    "backend engineer",
			expected: "backend-engineer",
		},
		{
			name:     "strip agent suffix",
			input:    "backend engineer agent",
			expected: "backend-engineer",
		},
		{
			name:     "strip only suffix",
			input:    "backend engineer only",
			expected: "backend-engineer",
		},
		{
			name:     "extra hyphens",
			input:    "backend--engineer",
			expected: "backend-engineer",
		},
		{
			name:     "leading/trailing hyphens",
			input:    "-backend-engineer-",
			expected: "backend-engineer",
		},
		{
			name:     "leading/trailing whitespace",
			input:    "  web-designer  ",
			expected: "web-designer",
		},
		{
			name:     "spaces and agent suffix",
			input:    "  Web Designer Agent  ",
			expected: "web-designer",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "mixed case with spaces",
			input:    "QA Engineer",
			expected: "qa-engineer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePersonaHint(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePersonaHint(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindAgentByPersonaHint(t *testing.T) {
	pm := NewPersonaMatcher()

	agents := []*models.Agent{
		{ID: "a1", Name: "BackendBot", PersonaName: "default/backend-engineer", Role: "Backend Engineer"},
		{ID: "a2", Name: "DesignBot", PersonaName: "default/web-designer", Role: "Web Designer"},
		{ID: "a3", Name: "OpsBot", PersonaName: "devops-engineer", Role: "DevOps Engineer"},
		{ID: "a4", Name: "QABot", PersonaName: "qa-engineer", Role: "QA Engineer"},
		nil, // nil agents should be skipped
	}

	tests := []struct {
		name       string
		hint       string
		agents     []*models.Agent
		expectedID string
	}{
		{
			name:       "empty hint returns nil",
			hint:       "",
			agents:     agents,
			expectedID: "",
		},
		{
			name:       "nil agents returns nil",
			hint:       "backend-engineer",
			agents:     nil,
			expectedID: "",
		},
		{
			name:       "empty agents returns nil",
			hint:       "backend-engineer",
			agents:     []*models.Agent{},
			expectedID: "",
		},
		{
			name:       "exact match on persona name without prefix",
			hint:       "backend-engineer",
			agents:     agents,
			expectedID: "a1",
		},
		{
			name:       "exact match on persona name no prefix",
			hint:       "devops-engineer",
			agents:     agents,
			expectedID: "a3",
		},
		{
			name:       "fuzzy match - hint contained in persona name",
			hint:       "backend",
			agents:     agents,
			expectedID: "a1",
		},
		{
			name:       "fuzzy match - persona name contained in hint",
			hint:       "web-designer-special",
			agents:     agents,
			expectedID: "a2",
		},
		{
			name:       "role match fallback",
			hint:       "web designer",
			agents:     agents,
			expectedID: "a2",
		},
		{
			name:       "no match returns nil",
			hint:       "nonexistent-persona",
			agents:     agents,
			expectedID: "",
		},
		{
			name:       "case insensitive match",
			hint:       "Backend-Engineer",
			agents:     agents,
			expectedID: "a1",
		},
		{
			name:       "all nil agents returns nil",
			hint:       "backend-engineer",
			agents:     []*models.Agent{nil, nil, nil},
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pm.FindAgentByPersonaHint(tt.hint, tt.agents)
			if tt.expectedID == "" {
				if result != nil {
					t.Errorf("Expected nil, got agent %s", result.ID)
				}
			} else {
				if result == nil {
					t.Errorf("Expected agent %s, got nil", tt.expectedID)
				} else if result.ID != tt.expectedID {
					t.Errorf("Expected agent %s, got %s", tt.expectedID, result.ID)
				}
			}
		})
	}
}
