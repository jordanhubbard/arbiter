package dispatch

import (
	"fmt"
	"strings"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

// AutoBugRouter handles intelligent routing of auto-filed bugs to appropriate coding agents
type AutoBugRouter struct{}

func NewAutoBugRouter() *AutoBugRouter {
	return &AutoBugRouter{}
}

// BugRouteInfo contains information about how to route a bug
type BugRouteInfo struct {
	ShouldRoute   bool   // Whether this bug should be auto-dispatched
	PersonaHint   string // Which persona/role should handle it
	UpdatedTitle  string // Title with persona hint added
	RoutingReason string // Why this routing was chosen
}

// AnalyzeBugForRouting analyzes an auto-filed bug and determines routing
func (r *AutoBugRouter) AnalyzeBugForRouting(bead *models.Bead) *BugRouteInfo {
	if bead == nil {
		return &BugRouteInfo{ShouldRoute: false}
	}

	// Check if this is an auto-filed bug
	if !r.isAutoFiledBug(bead) {
		return &BugRouteInfo{ShouldRoute: false}
	}

	// Check if already has a persona hint (already triaged)
	if r.hasPersonaHint(bead) {
		return &BugRouteInfo{ShouldRoute: false}
	}

	// Analyze bug type and determine routing
	info := &BugRouteInfo{ShouldRoute: true}

	title := strings.ToLower(bead.Title)
	desc := strings.ToLower(bead.Description)
	tags := r.getTagsLower(bead)

	// Build/deployment errors (check first - more specific than backend errors)
	if r.isBuildError(title, desc, tags) {
		info.PersonaHint = "devops-engineer"
		info.RoutingReason = "Build or deployment error detected"
		info.UpdatedTitle = fmt.Sprintf("[devops-engineer] %s", bead.Title)
		return info
	}

	// Frontend JavaScript errors
	if r.isFrontendJSError(title, desc, tags) {
		info.PersonaHint = "web-designer"
		info.RoutingReason = "Frontend JavaScript error detected"
		info.UpdatedTitle = fmt.Sprintf("[web-designer] %s", bead.Title)
		return info
	}

	// Backend Go errors
	if r.isBackendGoError(title, desc, tags) {
		info.PersonaHint = "backend-engineer"
		info.RoutingReason = "Backend Go compilation or runtime error detected"
		info.UpdatedTitle = fmt.Sprintf("[backend-engineer] %s", bead.Title)
		return info
	}

	// API/HTTP errors
	if r.isAPIError(title, desc, tags) {
		info.PersonaHint = "backend-engineer"
		info.RoutingReason = "API/HTTP error detected"
		info.UpdatedTitle = fmt.Sprintf("[backend-engineer] %s", bead.Title)
		return info
	}

	// Database errors
	if r.isDatabaseError(title, desc, tags) {
		info.PersonaHint = "backend-engineer"
		info.RoutingReason = "Database error detected"
		info.UpdatedTitle = fmt.Sprintf("[backend-engineer] %s", bead.Title)
		return info
	}

	// CSS/styling errors
	if r.isStylingError(title, desc, tags) {
		info.PersonaHint = "web-designer"
		info.RoutingReason = "CSS or styling error detected"
		info.UpdatedTitle = fmt.Sprintf("[web-designer] %s", bead.Title)
		return info
	}

	// Default to QA Engineer for triage if type unclear
	info.ShouldRoute = false // Keep with QA for now
	info.RoutingReason = "Bug type unclear, needs QA triage"
	return info
}

func (r *AutoBugRouter) isAutoFiledBug(bead *models.Bead) bool {
	if strings.Contains(strings.ToLower(bead.Title), "[auto-filed]") {
		return true
	}
	for _, tag := range bead.Tags {
		if strings.ToLower(tag) == "auto-filed" {
			return true
		}
	}
	return false
}

func (r *AutoBugRouter) hasPersonaHint(bead *models.Bead) bool {
	// Check for existing persona hints in title
	title := strings.ToLower(bead.Title)
	personas := []string{"web-designer", "backend-engineer", "devops-engineer", "qa-engineer", "ceo", "cfo"}
	for _, persona := range personas {
		if strings.Contains(title, fmt.Sprintf("[%s]", persona)) {
			return true
		}
	}
	return false
}

func (r *AutoBugRouter) getTagsLower(bead *models.Bead) map[string]bool {
	tags := make(map[string]bool)
	for _, tag := range bead.Tags {
		tags[strings.ToLower(tag)] = true
	}
	return tags
}

func (r *AutoBugRouter) isFrontendJSError(title, desc string, tags map[string]bool) bool {
	// Check for JavaScript error patterns
	jsIndicators := []string{
		"javascript", "js_error", "syntaxerror", "referenceerror", "typeerror",
		"uncaught", "undefined", "is not a function", "is not defined",
		"cannot read", "cannot access", "ui error",
	}

	for _, indicator := range jsIndicators {
		if strings.Contains(title, indicator) || strings.Contains(desc, indicator) {
			return true
		}
	}

	return tags["frontend"] || tags["javascript"] || tags["js_error"]
}

func (r *AutoBugRouter) isBackendGoError(title, desc string, tags map[string]bool) bool {
	// Check for Go error patterns
	goIndicators := []string{
		"panic", "runtime error", "nil pointer", "invalid memory",
		"undefined:", "cannot use", "go build", "compilation error",
	}

	for _, indicator := range goIndicators {
		if strings.Contains(title, indicator) || strings.Contains(desc, indicator) {
			return true
		}
	}

	return tags["backend"] || tags["golang"] || tags["go_error"]
}

func (r *AutoBugRouter) isAPIError(title, desc string, tags map[string]bool) bool {
	// Check for API error patterns
	apiIndicators := []string{
		"api error", "api request failed", "http", "status code",
		"endpoint", "route not found", "405", "404", "500", "502", "503",
	}

	for _, indicator := range apiIndicators {
		if strings.Contains(title, indicator) || strings.Contains(desc, indicator) {
			return true
		}
	}

	return tags["api"] || tags["api_error"] || tags["http"]
}

func (r *AutoBugRouter) isDatabaseError(title, desc string, tags map[string]bool) bool {
	// Check for database error patterns
	dbIndicators := []string{
		"database", "sql", "query", "connection refused", "postgres", "sqlite",
		"deadlock", "constraint", "foreign key",
	}

	for _, indicator := range dbIndicators {
		if strings.Contains(title, indicator) || strings.Contains(desc, indicator) {
			return true
		}
	}

	return tags["database"] || tags["sql"] || tags["db_error"]
}

func (r *AutoBugRouter) isBuildError(title, desc string, tags map[string]bool) bool {
	// Check for build/deployment error patterns
	buildIndicators := []string{
		"build failed", "docker", "dockerfile", "compile", "deployment",
		"makefile", "ci/cd", "pipeline", "container",
	}

	for _, indicator := range buildIndicators {
		if strings.Contains(title, indicator) || strings.Contains(desc, indicator) {
			return true
		}
	}

	return tags["build"] || tags["deployment"] || tags["docker"]
}

func (r *AutoBugRouter) isStylingError(title, desc string, tags map[string]bool) bool {
	// Check for CSS/styling error patterns
	styleIndicators := []string{
		"css", "style", "layout", "rendering", "display",
		"flexbox", "grid", "responsive",
	}

	for _, indicator := range styleIndicators {
		if strings.Contains(title, indicator) || strings.Contains(desc, indicator) {
			return true
		}
	}

	return tags["css"] || tags["styling"] || tags["ui"]
}
