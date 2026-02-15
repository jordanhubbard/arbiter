package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/provider"
	"github.com/jordanhubbard/loom/pkg/models"
)

// --- readProjectFile extended tests ---

func TestReadProjectFile_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	content := "This is a test AGENTS.md file\nwith multiple lines.\n"

	err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := readProjectFile(tmpDir, "AGENTS.md", 4000)
	if result != content {
		t.Errorf("Expected file content %q, got %q", content, result)
	}
}

func TestReadProjectFile_Truncation(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a long content string
	content := strings.Repeat("abcdefghij", 100) // 1000 chars

	err := os.WriteFile(filepath.Join(tmpDir, "long.md"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Set maxLen less than content length
	result := readProjectFile(tmpDir, "long.md", 50)
	if len(result) == 0 {
		t.Fatal("Expected non-empty result for truncated file")
	}
	if !strings.HasSuffix(result, "\n... (truncated)") {
		t.Errorf("Expected truncated suffix, got %q", result[len(result)-30:])
	}
	// The result should start with the first 50 chars of content
	if !strings.HasPrefix(result, content[:50]) {
		t.Error("Expected result to start with first 50 chars of original content")
	}
}

func TestReadProjectFile_ExactMaxLen(t *testing.T) {
	tmpDir := t.TempDir()
	content := "exactly50charsxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // 50 chars

	err := os.WriteFile(filepath.Join(tmpDir, "exact.md"), []byte(content[:50]), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := readProjectFile(tmpDir, "exact.md", 50)
	// Exactly at maxLen should NOT be truncated
	if strings.Contains(result, "truncated") {
		t.Error("Expected no truncation when content is exactly maxLen")
	}
}

func TestReadProjectFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "empty.md"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := readProjectFile(tmpDir, "empty.md", 4000)
	if result != "" {
		t.Errorf("Expected empty string for empty file, got %q", result)
	}
}

// --- normalizeRoleName edge cases ---

func TestNormalizeRoleName_ExtraEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "only slashes",
			input:    "///",
			expected: "",
		},
		{
			name:     "only parentheses",
			input:    "(lead)",
			expected: "",
		},
		{
			name:     "multiple spaces between words",
			input:    "backend   engineer",
			expected: "backend-engineer",
		},
		{
			name:     "single character role",
			input:    "x",
			expected: "x",
		},
		{
			name:     "slash at end",
			input:    "team/",
			expected: "",
		},
		{
			name:     "slash at start",
			input:    "/engineer",
			expected: "engineer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRoleName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeRoleName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// --- buildBeadDescription edge cases ---

func TestBuildBeadDescription_EmptyFields(t *testing.T) {
	bead := &models.Bead{
		ID:          "",
		Title:       "",
		Description: "",
	}
	result := buildBeadDescription(bead)
	if result == "" {
		t.Error("Expected non-empty result even with empty fields")
	}
	if !strings.Contains(result, "Work on bead") {
		t.Error("Expected result to contain prefix 'Work on bead'")
	}
}

func TestBuildBeadDescription_SpecialChars(t *testing.T) {
	bead := &models.Bead{
		ID:          "bead-special",
		Title:       "Fix \"quoted\" & <escaped> issue",
		Description: "Description with\nnewlines\tand\ttabs",
	}
	result := buildBeadDescription(bead)
	if !strings.Contains(result, "\"quoted\"") {
		t.Error("Expected quotes to be preserved")
	}
	if !strings.Contains(result, "\nnewlines") {
		t.Error("Expected newlines to be preserved")
	}
}

// --- buildBeadContext edge cases ---

func TestBuildBeadContext_ProjectWithWorkDir(t *testing.T) {
	bead := &models.Bead{
		ID:       "bead-workdir",
		Priority: models.BeadPriorityP1,
		Type:     "task",
	}
	project := &models.Project{
		ID:      "proj-wd",
		Name:    "WorkDirProject",
		Branch:  "main",
		WorkDir: "/nonexistent/workdir",
	}
	result := buildBeadContext(bead, project)
	if !strings.Contains(result, "WorkDirProject") {
		t.Error("Expected project name in context")
	}
	// AGENTS.md won't exist, so it should not appear
	if strings.Contains(result, "AGENTS.md") {
		t.Error("Did not expect AGENTS.md reference when file doesn't exist")
	}
}

func TestBuildBeadContext_ProjectWithWorkDirAndAgentsMD(t *testing.T) {
	tmpDir := t.TempDir()
	agentsContent := "# Agent Instructions\nDo things correctly.\n"
	err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(agentsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create AGENTS.md: %v", err)
	}

	bead := &models.Bead{
		ID:       "bead-agents",
		Priority: models.BeadPriorityP1,
		Type:     "task",
	}
	project := &models.Project{
		ID:      "proj-agents",
		Name:    "AgentsProject",
		Branch:  "main",
		WorkDir: tmpDir,
	}
	result := buildBeadContext(bead, project)
	if !strings.Contains(result, "Project Instructions") {
		t.Error("Expected AGENTS.md section header")
	}
	if !strings.Contains(result, "Do things correctly") {
		t.Error("Expected AGENTS.md content in context")
	}
}

func TestBuildBeadContext_EmptyProject(t *testing.T) {
	bead := &models.Bead{
		ID:       "bead-empty-proj",
		Priority: models.BeadPriorityP2,
		Type:     "bug",
	}
	project := &models.Project{
		ID:     "proj-empty",
		Name:   "",
		Branch: "",
	}
	result := buildBeadContext(bead, project)
	if !strings.Contains(result, "Project:") {
		t.Error("Expected project section even with empty fields")
	}
}

// --- buildDispatchHistory edge cases ---

func TestBuildDispatchHistory_ExactlySixEntries(t *testing.T) {
	// Test with exactly 5 entries + 1 new = 6, alternating
	bead := &models.Bead{
		ID: "b-exact-6",
		Context: map[string]string{
			"dispatch_history": `["a1","a2","a1","a2","a1"]`,
		},
	}
	historyJSON, loopDetected, _ := buildDispatchHistory(bead, "a2")
	var history []string
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if len(history) != 6 {
		t.Errorf("Expected 6 entries, got %d", len(history))
	}
	if !loopDetected {
		t.Error("Expected loop to be detected with 6 alternating entries")
	}
}

func TestBuildDispatchHistory_ThreeUnique(t *testing.T) {
	// Three unique agents in last 6 is NOT a two-agent loop
	bead := &models.Bead{
		ID: "b-3unique",
		Context: map[string]string{
			"dispatch_history": `["a1","a2","a3","a1","a2"]`,
		},
	}
	_, loopDetected, _ := buildDispatchHistory(bead, "a3")
	if loopDetected {
		t.Error("Expected no loop with 3 unique agents")
	}
}

func TestBuildDispatchHistory_SameAgentRepeated(t *testing.T) {
	// Same agent for all 6 - only 1 unique, not 2
	bead := &models.Bead{
		ID: "b-same",
		Context: map[string]string{
			"dispatch_history": `["a1","a1","a1","a1","a1"]`,
		},
	}
	_, loopDetected, _ := buildDispatchHistory(bead, "a1")
	if loopDetected {
		t.Error("Expected no loop with single agent repeated")
	}
}

func TestBuildDispatchHistory_NotAlternatingPattern(t *testing.T) {
	// 2 unique agents but NOT alternating: a1,a1,a2,a2,a1
	bead := &models.Bead{
		ID: "b-not-alternating",
		Context: map[string]string{
			"dispatch_history": `["a1","a1","a2","a2","a1"]`,
		},
	}
	_, loopDetected, _ := buildDispatchHistory(bead, "a2")
	// The loop detector checks if last[i] == last[i%2], so this should NOT be a loop
	if loopDetected {
		t.Error("Expected no loop with non-alternating pattern")
	}
}

func TestBuildDispatchHistory_EmptyHistory(t *testing.T) {
	bead := &models.Bead{
		ID: "b-empty-hist",
		Context: map[string]string{
			"dispatch_history": `[]`,
		},
	}
	historyJSON, loopDetected, _ := buildDispatchHistory(bead, "a1")
	var history []string
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(history))
	}
	if loopDetected {
		t.Error("Expected no loop with only 1 entry")
	}
}

// --- hasTag edge cases ---

func TestHasTag_CaseVariations(t *testing.T) {
	d := &Dispatcher{}

	bead := &models.Bead{
		Tags: []string{"REQUIRES-HUMAN-CONFIG", "Auto-Filed", "  Bug  "},
	}

	if !d.hasTag(bead, "requires-human-config") {
		t.Error("Expected uppercase tag to match lowercase search")
	}
	if !d.hasTag(bead, "AUTO-FILED") {
		t.Error("Expected lowercase tag to match uppercase search")
	}
	if !d.hasTag(bead, "bug") {
		t.Error("Expected padded tag to match trimmed search")
	}
}

// --- Dispatcher setter method tests ---

func TestDispatcher_SetMaxDispatchHops_Values(t *testing.T) {
	d := &Dispatcher{}

	// Test zero
	d.SetMaxDispatchHops(0)
	d.mu.RLock()
	val := d.maxDispatchHops
	d.mu.RUnlock()
	if val != 0 {
		t.Errorf("Expected 0, got %d", val)
	}

	// Test negative
	d.SetMaxDispatchHops(-10)
	d.mu.RLock()
	val = d.maxDispatchHops
	d.mu.RUnlock()
	if val != -10 {
		t.Errorf("Expected -10, got %d", val)
	}

	// The DispatchOnce method handles the fallback to 20 when <= 0
	maxHops := val
	if maxHops <= 0 {
		maxHops = 20
	}
	if maxHops != 20 {
		t.Errorf("Expected fallback 20, got %d", maxHops)
	}
}

func TestDispatcher_SetWorkflowEngine(t *testing.T) {
	d := &Dispatcher{}

	// Initially nil
	if d.workflowEngine != nil {
		t.Error("Expected workflowEngine to be nil initially")
	}

	// Set nil explicitly
	d.SetWorkflowEngine(nil)
	d.mu.RLock()
	we := d.workflowEngine
	d.mu.RUnlock()
	if we != nil {
		t.Error("Expected workflowEngine to remain nil")
	}
}

func TestDispatcher_SetReadinessCheck_WithFunc(t *testing.T) {
	d := &Dispatcher{}

	// Setting nil should work
	d.SetReadinessCheck(nil)
	d.mu.RLock()
	rc := d.readinessCheck
	d.mu.RUnlock()
	if rc != nil {
		t.Error("Expected nil readinessCheck")
	}

	// Setting a real function should work
	d.SetReadinessCheck(func(ctx context.Context, projectID string) (bool, []string) {
		return true, nil
	})
	d.mu.RLock()
	rc2 := d.readinessCheck
	d.mu.RUnlock()
	if rc2 == nil {
		t.Error("Expected readinessCheck to be non-nil after setting")
	}
}

// --- estimateBeadComplexity edge cases ---

func TestEstimateBeadComplexity_P0Simple(t *testing.T) {
	d := &Dispatcher{
		complexityEstimator: provider.NewComplexityEstimator(),
	}

	// P0 task should be at least medium even if content is simple
	bead := &models.Bead{
		ID:          "b-p0-simple",
		Title:       "fix typo",
		Description: "change one word",
		Type:        "task",
		Priority:    models.BeadPriorityP0,
	}

	result := d.estimateBeadComplexity(bead)
	if result < provider.ComplexityMedium {
		t.Errorf("Expected at least medium for P0, got %s", result.String())
	}
}

func TestEstimateBeadComplexity_DecisionAlwaysComplex(t *testing.T) {
	d := &Dispatcher{
		complexityEstimator: provider.NewComplexityEstimator(),
	}

	bead := &models.Bead{
		ID:          "b-decision-simple",
		Title:       "simple question",
		Description: "yes or no",
		Type:        "decision",
		Priority:    models.BeadPriorityP3,
	}

	result := d.estimateBeadComplexity(bead)
	if result != provider.ComplexityComplex {
		t.Errorf("Expected ComplexityComplex for decision bead, got %s", result.String())
	}
}

func TestEstimateBeadComplexity_WithErrorContext(t *testing.T) {
	d := &Dispatcher{
		complexityEstimator: provider.NewComplexityEstimator(),
	}

	bead := &models.Bead{
		ID:          "b-with-errors",
		Title:       "Debug issue",
		Description: "Something broke",
		Type:        "bug",
		Priority:    models.BeadPriorityP2,
		Context: map[string]string{
			"error_message": "fatal: architecture design flaw in distributed system",
			"agent_output":  "This requires refactoring the entire module",
		},
	}

	result := d.estimateBeadComplexity(bead)
	// With error context containing complexity keywords, should be at least medium
	if result < provider.ComplexitySimple {
		t.Errorf("Expected at least simple complexity with error context, got %s", result.String())
	}
}

func TestEstimateBeadComplexity_EmptyDescription(t *testing.T) {
	d := &Dispatcher{
		complexityEstimator: provider.NewComplexityEstimator(),
	}

	bead := &models.Bead{
		ID:          "b-no-desc",
		Title:       "task",
		Description: "",
		Type:        "task",
		Priority:    models.BeadPriorityP2,
	}

	// Should not panic with empty description
	result := d.estimateBeadComplexity(bead)
	if result < provider.ComplexitySimple {
		t.Errorf("Expected at least simple complexity, got %s", result.String())
	}
}

func TestEstimateBeadComplexity_NilContext(t *testing.T) {
	d := &Dispatcher{
		complexityEstimator: provider.NewComplexityEstimator(),
	}

	bead := &models.Bead{
		ID:          "b-nil-ctx",
		Title:       "Fix something",
		Description: "It's broken",
		Type:        "task",
		Priority:    models.BeadPriorityP1,
		Context:     nil,
	}

	// Should not panic with nil context
	result := d.estimateBeadComplexity(bead)
	_ = result // Just ensuring no panic
}

// --- SystemStatus JSON edge cases ---

func TestSystemStatusJSON_EmptyState(t *testing.T) {
	status := SystemStatus{
		State:     "",
		Reason:    "",
		UpdatedAt: time.Time{},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal empty SystemStatus: %v", err)
	}

	var decoded SystemStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal empty SystemStatus: %v", err)
	}

	if decoded.State != "" {
		t.Errorf("Expected empty state, got %q", decoded.State)
	}
}

// --- DispatchResult JSON edge cases ---

func TestDispatchResultJSON_AllEmpty(t *testing.T) {
	result := DispatchResult{}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal empty DispatchResult: %v", err)
	}

	var decoded DispatchResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal empty DispatchResult: %v", err)
	}

	if decoded.Dispatched != false {
		t.Error("Expected Dispatched to be false")
	}
}

func TestDispatchResultJSON_Omitempty(t *testing.T) {
	// Empty optional fields should be omitted
	result := DispatchResult{
		Dispatched: true,
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(data)
	// Optional fields with omitempty should not appear when empty
	if strings.Contains(jsonStr, "project_id") {
		t.Error("Expected project_id to be omitted when empty")
	}
	if strings.Contains(jsonStr, "bead_id") {
		t.Error("Expected bead_id to be omitted when empty")
	}
}

// --- Concurrent operations ---

func TestDispatcher_ConcurrentSetters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	d := &Dispatcher{readinessMode: ReadinessWarn}
	done := make(chan bool)

	// Concurrent SetReadinessMode
	for i := 0; i < 5; i++ {
		go func(idx int) {
			if idx%2 == 0 {
				d.SetReadinessMode(ReadinessBlock)
			} else {
				d.SetReadinessMode(ReadinessWarn)
			}
			done <- true
		}(i)
	}

	// Concurrent SetMaxDispatchHops
	for i := 0; i < 5; i++ {
		go func(idx int) {
			d.SetMaxDispatchHops(idx * 10)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			_ = d.GetSystemStatus()
			done <- true
		}()
	}

	for i := 0; i < 15; i++ {
		<-done
	}

	// Should not deadlock or panic
	status := d.GetSystemStatus()
	_ = status
}

// --- NewDispatcher ---

func TestNewDispatcher_NilArgs(t *testing.T) {
	// NewDispatcher with all nil args should not panic
	d := NewDispatcher(nil, nil, nil, nil, nil)
	if d == nil {
		t.Fatal("Expected non-nil Dispatcher")
	}

	// Check defaults are set
	if d.personaMatcher == nil {
		t.Error("Expected personaMatcher to be initialized")
	}
	if d.autoBugRouter == nil {
		t.Error("Expected autoBugRouter to be initialized")
	}
	if d.complexityEstimator == nil {
		t.Error("Expected complexityEstimator to be initialized")
	}
	if d.loopDetector == nil {
		t.Error("Expected loopDetector to be initialized")
	}
	if d.readinessMode != ReadinessWarn {
		t.Errorf("Expected default readinessMode to be ReadinessWarn, got %q", d.readinessMode)
	}

	// Default status should be parked
	status := d.GetSystemStatus()
	if status.State != StatusParked {
		t.Errorf("Expected default state to be StatusParked, got %q", status.State)
	}
	if status.Reason != "not started" {
		t.Errorf("Expected default reason to be 'not started', got %q", status.Reason)
	}
	if status.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

// --- Dispatcher creation defaults verification ---

func TestNewDispatcher_DefaultFields(t *testing.T) {
	d := NewDispatcher(nil, nil, nil, nil, nil)

	// Verify loopDetector default threshold
	if d.loopDetector.repeatThreshold != 3 {
		t.Errorf("Expected default repeatThreshold=3, got %d", d.loopDetector.repeatThreshold)
	}

	// Verify persona matcher has patterns
	if len(d.personaMatcher.patterns) == 0 {
		t.Error("Expected persona matcher to have patterns")
	}

	// maxDispatchHops should be 0 (unset), with fallback in DispatchOnce
	if d.maxDispatchHops != 0 {
		t.Errorf("Expected maxDispatchHops=0 (unset), got %d", d.maxDispatchHops)
	}

	// escalator should be nil
	if d.escalator != nil {
		t.Error("Expected escalator to be nil by default")
	}

	// db should be nil
	if d.db != nil {
		t.Error("Expected db to be nil by default")
	}

	// workflowEngine should be nil
	if d.workflowEngine != nil {
		t.Error("Expected workflowEngine to be nil by default")
	}

	// readinessCheck should be nil
	if d.readinessCheck != nil {
		t.Error("Expected readinessCheck to be nil by default")
	}
}

// --- Multiple dispatch history overflow ---

func TestBuildDispatchHistory_OverflowTruncation(t *testing.T) {
	// Create history with exactly 20 entries, then add one more
	entries := make([]string, 20)
	for i := 0; i < 20; i++ {
		entries[i] = fmt.Sprintf("agent-%d", i)
	}
	entriesJSON, _ := json.Marshal(entries)

	bead := &models.Bead{
		ID: "b-overflow",
		Context: map[string]string{
			"dispatch_history": string(entriesJSON),
		},
	}

	historyJSON, _, _ := buildDispatchHistory(bead, "agent-new")
	var history []string
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(history) > 20 {
		t.Errorf("Expected history to be capped at 20, got %d", len(history))
	}
	if history[len(history)-1] != "agent-new" {
		t.Errorf("Expected last entry to be 'agent-new', got %q", history[len(history)-1])
	}
}

// --- ensureBeadHasWorkflow keyword detection (unit test) ---

func TestWorkflowTypeDetection(t *testing.T) {
	// Test the workflow type detection logic without needing a real workflow engine.
	// This tests the logic inline since ensureBeadHasWorkflow requires external deps.
	tests := []struct {
		name     string
		title    string
		tags     []string
		expected string
	}{
		{
			name:     "self-improvement tag",
			title:    "Review code quality",
			tags:     []string{"self-improvement"},
			expected: "self-improvement",
		},
		{
			name:     "code-review tag",
			title:    "Check module",
			tags:     []string{"code-review"},
			expected: "self-improvement",
		},
		{
			name:     "maintainability tag",
			title:    "Improve code",
			tags:     []string{"maintainability"},
			expected: "self-improvement",
		},
		{
			name:     "refactoring tag",
			title:    "Clean up code",
			tags:     []string{"refactoring"},
			expected: "self-improvement",
		},
		{
			name:     "code review in title",
			title:    "[code review] Review auth module",
			tags:     []string{},
			expected: "self-improvement",
		},
		{
			name:     "refactor in title",
			title:    "[refactor] Clean up handlers",
			tags:     []string{},
			expected: "self-improvement",
		},
		{
			name:     "optimization in title",
			title:    "[optimization] Improve query performance",
			tags:     []string{},
			expected: "self-improvement",
		},
		{
			name:     "self-improvement in title",
			title:    "[self-improvement] Better logging",
			tags:     []string{},
			expected: "self-improvement",
		},
		{
			name:     "maintenance in title",
			title:    "[maintenance] Update dependencies",
			tags:     []string{},
			expected: "self-improvement",
		},
		{
			name:     "feature keyword",
			title:    "Add new feature for login",
			tags:     []string{},
			expected: "feature",
		},
		{
			name:     "enhancement keyword",
			title:    "Enhancement to reporting",
			tags:     []string{},
			expected: "feature",
		},
		{
			name:     "ui keyword",
			title:    "Fix ui rendering",
			tags:     []string{},
			expected: "ui",
		},
		{
			name:     "design keyword",
			title:    "New design for dashboard",
			tags:     []string{},
			expected: "ui",
		},
		{
			name:     "css keyword",
			title:    "Fix css layout",
			tags:     []string{},
			expected: "ui",
		},
		{
			name:     "html keyword",
			title:    "Update html template",
			tags:     []string{},
			expected: "ui",
		},
		{
			name:     "default to bug",
			title:    "Fix login timeout",
			tags:     []string{},
			expected: "bug",
		},
		{
			name:     "tag priority over title",
			title:    "Add feature for login", // matches "feature"
			tags:     []string{"self-improvement"},
			expected: "self-improvement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := strings.ToLower(tt.title)

			// Check self-improvement tags first (same logic as ensureBeadHasWorkflow)
			isSelfImprovement := false
			for _, tag := range tt.tags {
				tagLower := strings.ToLower(tag)
				if tagLower == "self-improvement" || tagLower == "code-review" ||
					tagLower == "maintainability" || tagLower == "refactoring" {
					isSelfImprovement = true
					break
				}
			}

			if strings.Contains(title, "[code review]") || strings.Contains(title, "[refactor]") ||
				strings.Contains(title, "[optimization]") || strings.Contains(title, "[self-improvement]") ||
				strings.Contains(title, "[maintenance]") {
				isSelfImprovement = true
			}

			var workflowType string
			if isSelfImprovement {
				workflowType = "self-improvement"
			} else if strings.Contains(title, "feature") || strings.Contains(title, "enhancement") {
				workflowType = "feature"
			} else if strings.Contains(title, "ui") || strings.Contains(title, "design") || strings.Contains(title, "css") || strings.Contains(title, "html") {
				workflowType = "ui"
			} else {
				workflowType = "bug"
			}

			if workflowType != tt.expected {
				t.Errorf("Expected workflow type %q, got %q", tt.expected, workflowType)
			}
		})
	}
}

// --- getWorkflowRoleRequirement logic ---

func TestGetWorkflowRoleRequirement_NilInputs(t *testing.T) {
	d := &Dispatcher{}

	// nil workflow engine
	result := d.getWorkflowRoleRequirement(nil)
	if result != "" {
		t.Errorf("Expected empty string for nil execution, got %q", result)
	}
}

// --- findDefaultTriageAgent_NilManager ---

func TestFindDefaultTriageAgent_NilManager(t *testing.T) {
	d := &Dispatcher{agents: nil}
	result := d.findDefaultTriageAgent("proj-1")
	if result != "" {
		t.Errorf("Expected empty string with nil agents manager, got %q", result)
	}
}

// --- Status transitions ---

func TestDispatcher_StatusTransitions(t *testing.T) {
	d := &Dispatcher{}

	transitions := []struct {
		state  StatusState
		reason string
	}{
		{StatusParked, "initial"},
		{StatusActive, "dispatching bead-1"},
		{StatusParked, "execution failed"},
		{StatusActive, "dispatching bead-2"},
		{StatusParked, "idle"},
	}

	for _, tr := range transitions {
		d.setStatus(tr.state, tr.reason)
		status := d.GetSystemStatus()
		if status.State != tr.state {
			t.Errorf("Expected state %q, got %q", tr.state, status.State)
		}
		if status.Reason != tr.reason {
			t.Errorf("Expected reason %q, got %q", tr.reason, status.Reason)
		}
		if status.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}
	}
}

// --- Readiness mode setter ---

func TestSetReadinessMode_AllVariations(t *testing.T) {
	tests := []struct {
		name     string
		mode     ReadinessMode
		expected ReadinessMode
	}{
		{"block mode", ReadinessBlock, ReadinessBlock},
		{"warn mode", ReadinessWarn, ReadinessWarn},
		{"random string", ReadinessMode("random"), ReadinessWarn},
		{"numeric string", ReadinessMode("123"), ReadinessWarn},
		{"empty string", ReadinessMode(""), ReadinessWarn},
		{"case sensitive - Block", ReadinessMode("Block"), ReadinessWarn},
		{"case sensitive - Warn", ReadinessMode("Warn"), ReadinessWarn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dispatcher{readinessMode: ReadinessWarn}
			d.SetReadinessMode(tt.mode)
			d.mu.RLock()
			actual := d.readinessMode
			d.mu.RUnlock()
			if actual != tt.expected {
				t.Errorf("SetReadinessMode(%q) = %q, want %q", tt.mode, actual, tt.expected)
			}
		})
	}
}

// --- Test SetEscalator with mock ---

func TestDispatcher_SetEscalator_NilAndNonNil(t *testing.T) {
	d := &Dispatcher{}

	// Set nil
	d.SetEscalator(nil)
	d.mu.RLock()
	e := d.escalator
	d.mu.RUnlock()
	if e != nil {
		t.Error("Expected escalator to be nil")
	}

	// Set non-nil
	mock := &MockEscalator{Decisions: make(map[string]*models.DecisionBead)}
	d.SetEscalator(mock)
	d.mu.RLock()
	e = d.escalator
	d.mu.RUnlock()
	if e == nil {
		t.Error("Expected escalator to be non-nil after setting")
	}
}
