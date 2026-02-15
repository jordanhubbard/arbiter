package agent

import "testing"

// TestDeriveRoleFromPersonaName tests role inference from persona names
func TestDeriveRoleFromPersonaName(t *testing.T) {
	tests := []struct {
		name         string
		personaName  string
		expectedRole string
	}{
		// QA roles
		{
			name:         "qa-engineer persona",
			personaName:  "default/qa-engineer",
			expectedRole: "QA",
		},
		{
			name:         "qa keyword",
			personaName:  "custom/qa",
			expectedRole: "QA",
		},
		{
			name:         "quality-assurance keyword",
			personaName:  "default/quality-assurance",
			expectedRole: "QA",
		},

		// Engineering Manager roles
		{
			name:         "engineering-manager persona",
			personaName:  "default/engineering-manager",
			expectedRole: "Engineering Manager",
		},
		{
			name:         "eng-manager keyword",
			personaName:  "custom/eng-manager",
			expectedRole: "Engineering Manager",
		},

		// Product Manager roles
		{
			name:         "product-manager persona",
			personaName:  "default/product-manager",
			expectedRole: "Product Manager",
		},
		{
			name:         "pm keyword",
			personaName:  "custom/pm",
			expectedRole: "Product Manager",
		},

		// Web Designer roles
		{
			name:         "web-designer persona",
			personaName:  "default/web-designer",
			expectedRole: "Web Designer",
		},
		{
			name:         "designer keyword",
			personaName:  "custom/designer",
			expectedRole: "Web Designer",
		},

		// Backend Engineer roles
		{
			name:         "backend-engineer persona",
			personaName:  "default/backend-engineer",
			expectedRole: "Backend Engineer",
		},
		{
			name:         "backend keyword",
			personaName:  "custom/backend",
			expectedRole: "Backend Engineer",
		},

		// Frontend Engineer roles
		{
			name:         "frontend-engineer persona",
			personaName:  "default/frontend-engineer",
			expectedRole: "Frontend Engineer",
		},
		{
			name:         "frontend keyword",
			personaName:  "custom/frontend",
			expectedRole: "Frontend Engineer",
		},

		// Code Reviewer roles
		{
			name:         "code-reviewer persona",
			personaName:  "default/code-reviewer",
			expectedRole: "Code Reviewer",
		},
		{
			name:         "reviewer keyword",
			personaName:  "custom/reviewer",
			expectedRole: "Code Reviewer",
		},

		// CEO role
		{
			name:         "ceo persona",
			personaName:  "default/ceo",
			expectedRole: "CEO",
		},

		// Project-scoped personas
		{
			name:         "project-scoped qa-engineer",
			personaName:  "projects/myproject/qa-engineer",
			expectedRole: "QA",
		},
		{
			name:         "project-scoped engineering-manager",
			personaName:  "projects/loom/engineering-manager",
			expectedRole: "Engineering Manager",
		},

		// Case insensitivity
		{
			name:         "uppercase QA",
			personaName:  "default/QA-ENGINEER",
			expectedRole: "QA",
		},
		{
			name:         "mixed case",
			personaName:  "default/Engineering-Manager",
			expectedRole: "Engineering Manager",
		},

		// Fallback behavior (no match)
		{
			name:         "unknown persona with default prefix",
			personaName:  "default/unknown-role",
			expectedRole: "unknown-role",
		},
		{
			name:         "unknown persona with custom prefix",
			personaName:  "custom/unknown-role",
			expectedRole: "unknown-role",
		},
		{
			name:         "unknown persona with projects prefix",
			personaName:  "projects/myproject/unknown",
			expectedRole: "unknown",
		},
		{
			name:         "plain persona name",
			personaName:  "generic-agent",
			expectedRole: "generic-agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveRoleFromPersonaName(tt.personaName)
			if got != tt.expectedRole {
				t.Errorf("deriveRoleFromPersonaName(%q) = %q, want %q",
					tt.personaName, got, tt.expectedRole)
			}
		})
	}
}

// TestDeriveRoleFromPersonaName_KeywordPriority tests that more specific keywords match first
func TestDeriveRoleFromPersonaName_KeywordPriority(t *testing.T) {
	tests := []struct {
		name         string
		personaName  string
		expectedRole string
		reason       string
	}{
		{
			name:         "qa-engineer over qa",
			personaName:  "default/qa-engineer",
			expectedRole: "QA",
			reason:       "Both 'qa-engineer' and 'qa' keywords match, should return QA",
		},
		{
			name:         "engineering-manager specific match",
			personaName:  "default/engineering-manager",
			expectedRole: "Engineering Manager",
			reason:       "Should match engineering-manager keyword",
		},
		{
			name:         "backend-engineer over backend",
			personaName:  "custom/backend-engineer",
			expectedRole: "Backend Engineer",
			reason:       "Both 'backend-engineer' and 'backend' match, should return Backend Engineer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveRoleFromPersonaName(tt.personaName)
			if got != tt.expectedRole {
				t.Errorf("deriveRoleFromPersonaName(%q) = %q, want %q (reason: %s)",
					tt.personaName, got, tt.expectedRole, tt.reason)
			}
		})
	}
}

// TestDeriveRoleFromPersonaName_EdgeCases tests edge cases
func TestDeriveRoleFromPersonaName_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		personaName  string
		expectedRole string
	}{
		{
			name:         "empty string",
			personaName:  "",
			expectedRole: "",
		},
		{
			name:         "whitespace only",
			personaName:  "   ",
			expectedRole: "",
		},
		{
			name:         "just slash",
			personaName:  "/",
			expectedRole: "",
		},
		{
			name:         "multiple slashes",
			personaName:  "a/b/c/d/qa-engineer",
			expectedRole: "QA",
		},
		{
			name:         "trailing slash",
			personaName:  "default/qa-engineer/",
			expectedRole: "QA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveRoleFromPersonaName(tt.personaName)
			if got != tt.expectedRole {
				t.Errorf("deriveRoleFromPersonaName(%q) = %q, want %q",
					tt.personaName, got, tt.expectedRole)
			}
		})
	}
}

// TestDeriveRoleFromPersonaName_WorkflowCompatibility tests compatibility with workflow role names
func TestDeriveRoleFromPersonaName_WorkflowCompatibility(t *testing.T) {
	// These are the exact role names used in workflow YAML files
	workflowRoles := []string{
		"QA",
		"Engineering Manager",
		"Product Manager",
		"Web Designer",
		"Backend Engineer",
		"Frontend Engineer",
		"Code Reviewer",
		"CEO",
	}

	// Test that each workflow role can be derived from a corresponding persona
	personas := []string{
		"default/qa-engineer",
		"default/engineering-manager",
		"default/product-manager",
		"default/web-designer",
		"default/backend-engineer",
		"default/frontend-engineer",
		"default/code-reviewer",
		"default/ceo",
	}

	for i, persona := range personas {
		expectedRole := workflowRoles[i]
		t.Run(expectedRole, func(t *testing.T) {
			got := deriveRoleFromPersonaName(persona)
			if got != expectedRole {
				t.Errorf("deriveRoleFromPersonaName(%q) = %q, want %q (workflow role)",
					persona, got, expectedRole)
			}
		})
	}
}
