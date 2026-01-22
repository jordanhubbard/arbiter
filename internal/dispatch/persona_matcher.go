package dispatch

import (
	"regexp"
	"strings"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

// PersonaMatcher provides fuzzy matching for persona-based routing
type PersonaMatcher struct {
	// Patterns for extracting persona hints from text
	patterns []*regexp.Regexp
}

func NewPersonaMatcher() *PersonaMatcher {
	return &PersonaMatcher{
		patterns: []*regexp.Regexp{
			// "ask the <persona> to ..."
			regexp.MustCompile(`(?i)ask\s+the\s+([a-z][a-z\s\-]+?)\s+to\s+`),
			// "[persona] ..." or "[persona]: ..."
			regexp.MustCompile(`(?i)^\[([a-z][a-z\s\-]+?)\][\s:]`),
			// "persona: ..." (CEO CLI format)
			regexp.MustCompile(`(?i)^([a-z][a-z\s\-]+?):\s+`),
			// "for <persona>:" or "for <persona> agent:"
			regexp.MustCompile(`(?i)for\s+([a-z][a-z\s\-]+?)(?:\s+agent)?[\s:]`),
			// "**FOR <persona> AGENT**" (markdown bold)
			regexp.MustCompile(`(?i)\*\*\s*for\s+([a-z][a-z\s\-]+?)(?:\s+agent)?\s*\*\*`),
		},
	}
}

// ExtractPersonaHint tries to extract persona hints from a bead
func (pm *PersonaMatcher) ExtractPersonaHint(bead *models.Bead) string {
	if bead == nil {
		return ""
	}

	// Check title first
	hint := pm.extractFromText(bead.Title)
	if hint != "" {
		return hint
	}

	// Then check description
	hint = pm.extractFromText(bead.Description)
	if hint != "" {
		return hint
	}

	// Check tags for persona-related tags
	for _, tag := range bead.Tags {
		if strings.Contains(strings.ToLower(tag), "persona") {
			// Extract persona from tags like "web-designer-only" or "for-ceo"
			parts := strings.Split(strings.ToLower(tag), "-")
			if len(parts) >= 2 {
				// Try to build persona name from tag parts
				if parts[0] == "for" {
					return strings.Join(parts[1:], "-")
				}
				if parts[len(parts)-1] == "only" {
					return strings.Join(parts[:len(parts)-1], "-")
				}
			}
		}
	}

	return ""
}

func (pm *PersonaMatcher) extractFromText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// Try each pattern
	for _, pattern := range pm.patterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) >= 2 {
			hint := strings.TrimSpace(matches[1])
			// Normalize the hint
			hint = normalizePersonaHint(hint)
			return hint
		}
	}

	return ""
}

// normalizePersonaHint normalizes a persona hint to match PersonaName format
func normalizePersonaHint(hint string) string {
	hint = strings.TrimSpace(strings.ToLower(hint))
	
	// Remove common suffixes
	hint = strings.TrimSuffix(hint, " agent")
	hint = strings.TrimSuffix(hint, " only")
	
	// Replace spaces with hyphens (persona names use hyphens)
	hint = strings.ReplaceAll(hint, " ", "-")
	
	// Remove extra hyphens
	hint = regexp.MustCompile(`-+`).ReplaceAllString(hint, "-")
	hint = strings.Trim(hint, "-")
	
	return hint
}

// FindAgentByPersonaHint finds the best matching agent for a persona hint
func (pm *PersonaMatcher) FindAgentByPersonaHint(hint string, agents []*models.Agent) *models.Agent {
	if hint == "" || len(agents) == 0 {
		return nil
	}

	hint = strings.ToLower(hint)

	// First pass: exact match on PersonaName (without default/ prefix)
	for _, agent := range agents {
		if agent == nil {
			continue
		}
		personaName := strings.ToLower(agent.PersonaName)
		// Strip "default/" prefix if present
		personaName = strings.TrimPrefix(personaName, "default/")
		
		if personaName == hint {
			return agent
		}
	}

	// Second pass: fuzzy match - check if hint is contained in PersonaName
	for _, agent := range agents {
		if agent == nil {
			continue
		}
		personaName := strings.ToLower(agent.PersonaName)
		personaName = strings.TrimPrefix(personaName, "default/")
		
		// Check if hint matches any part of the persona name
		if strings.Contains(personaName, hint) {
			return agent
		}
		
		// Check if any part of persona name matches the hint
		if strings.Contains(hint, personaName) {
			return agent
		}
	}

	// Third pass: fuzzy match on Role
	for _, agent := range agents {
		if agent == nil {
			continue
		}
		role := strings.ToLower(agent.Role)
		
		if role == hint || strings.Contains(role, hint) || strings.Contains(hint, role) {
			return agent
		}
	}

	// No match found
	return nil
}
