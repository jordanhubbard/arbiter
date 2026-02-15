package dispatch

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

// --- RecordAction edge cases ---

func TestRecordAction_HistoryTruncation(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-truncate",
		Context: make(map[string]string),
	}

	// Record 55 actions to trigger truncation (limit is 50)
	for i := 0; i < 55; i++ {
		action := ActionRecord{
			Timestamp:  time.Now(),
			AgentID:    "agent-1",
			ActionType: "read_file",
			ActionData: map[string]interface{}{
				"file_path": "file.go",
				"iteration": i,
			},
		}
		if err := ld.RecordAction(bead, action); err != nil {
			t.Fatalf("RecordAction failed at iteration %d: %v", i, err)
		}
	}

	history, err := ld.getActionHistory(bead)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}
	if len(history) != 50 {
		t.Errorf("Expected 50 entries after truncation, got %d", len(history))
	}
}

func TestRecordAction_WithCommandData(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-command",
		Context: make(map[string]string),
	}

	action := ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "bash",
		ActionData: map[string]interface{}{
			"command": "go test ./...",
		},
	}

	err := ld.RecordAction(bead, action)
	if err != nil {
		t.Fatalf("RecordAction failed: %v", err)
	}

	history, err := ld.getActionHistory(bead)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(history))
	}
	if history[0].ProgressKey == "" {
		t.Error("Expected progress key to be generated")
	}
}

func TestRecordAction_WithFilePathData(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-filepath",
		Context: make(map[string]string),
	}

	action := ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "read_file",
		ActionData: map[string]interface{}{
			"file_path": "/path/to/important.go",
		},
	}

	err := ld.RecordAction(bead, action)
	if err != nil {
		t.Fatalf("RecordAction failed: %v", err)
	}

	history, _ := ld.getActionHistory(bead)
	if len(history) == 0 {
		t.Fatal("Expected at least 1 history entry")
	}
	if history[0].ProgressKey == "" {
		t.Error("Expected progress key for file path action")
	}
}

// --- IsStuckInLoop comprehensive cases ---

func TestIsStuckInLoop_WithProgressMetrics(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-progress",
		Context: make(map[string]string),
	}

	// Record enough actions for detection (threshold * 2 = 6)
	for i := 0; i < 8; i++ {
		action := ActionRecord{
			Timestamp:  time.Now(),
			AgentID:    "agent-1",
			ActionType: "read_file",
			ActionData: map[string]interface{}{"file_path": "same.go"},
		}
		_ = ld.RecordAction(bead, action)
	}

	// Since we just recorded actions (recent), there IS progress
	stuck, _ := ld.IsStuckInLoop(bead)
	if stuck {
		t.Error("Expected not stuck when recent progress exists")
	}
}

func TestIsStuckInLoop_NilBead(t *testing.T) {
	ld := NewLoopDetector()
	// Test with a bead that has nil context
	bead := &models.Bead{
		ID:      "b-nil-ctx",
		Context: nil,
	}

	stuck, reason := ld.IsStuckInLoop(bead)
	if stuck {
		t.Errorf("Expected not stuck with nil context, got reason: %s", reason)
	}
}

func TestIsStuckInLoop_EmptyHistory(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID: "b-empty-history",
		Context: map[string]string{
			"action_history": "[]",
		},
	}

	stuck, _ := ld.IsStuckInLoop(bead)
	if stuck {
		t.Error("Expected not stuck with empty history")
	}
}

func TestIsStuckInLoop_InvalidHistoryJSON(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID: "b-bad-json",
		Context: map[string]string{
			"action_history": "not json",
		},
	}

	stuck, _ := ld.IsStuckInLoop(bead)
	if stuck {
		t.Error("Expected not stuck with invalid JSON history")
	}
}

// --- GetProgressSummary edge cases ---

func TestGetProgressSummary_WithRecentProgress(t *testing.T) {
	ld := NewLoopDetector()

	metrics := ProgressMetrics{
		FilesRead:        10,
		FilesModified:    3,
		TestsRun:         5,
		CommandsExecuted: 2,
		LastProgress:     time.Now().Add(-1 * time.Second),
	}
	data, _ := json.Marshal(metrics)

	bead := &models.Bead{
		ID: "b-recent-progress",
		Context: map[string]string{
			"progress_metrics": string(data),
		},
	}

	result := ld.GetProgressSummary(bead)
	if result == "No progress data" || result == "Invalid progress data" {
		t.Errorf("Expected valid summary, got %q", result)
	}
	if !strings.Contains(result, "Files read: 10") {
		t.Errorf("Expected 'Files read: 10' in summary, got %q", result)
	}
	if !strings.Contains(result, "modified: 3") {
		t.Errorf("Expected 'modified: 3' in summary, got %q", result)
	}
	if !strings.Contains(result, "tests: 5") {
		t.Errorf("Expected 'tests: 5' in summary, got %q", result)
	}
	if !strings.Contains(result, "commands: 2") {
		t.Errorf("Expected 'commands: 2' in summary, got %q", result)
	}
}

func TestGetProgressSummary_EmptyMetrics(t *testing.T) {
	ld := NewLoopDetector()

	bead := &models.Bead{
		ID: "b-empty-metrics",
		Context: map[string]string{
			"progress_metrics": "",
		},
	}

	result := ld.GetProgressSummary(bead)
	if result != "No progress data" {
		t.Errorf("Expected 'No progress data', got %q", result)
	}
}

// --- ResetProgress comprehensive ---

func TestResetProgress_WithData(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-reset-data",
		Context: make(map[string]string),
	}

	// Record some actions
	for i := 0; i < 3; i++ {
		_ = ld.RecordAction(bead, ActionRecord{
			Timestamp:  time.Now(),
			AgentID:    "agent-1",
			ActionType: "read_file",
			ActionData: map[string]interface{}{"file_path": "test.go"},
		})
	}

	// Verify data exists
	if bead.Context["action_history"] == "" {
		t.Error("Expected action_history to be set before reset")
	}
	if bead.Context["progress_metrics"] == "" {
		t.Error("Expected progress_metrics to be set before reset")
	}

	// Reset
	ld.ResetProgress(bead)

	// Verify cleared
	if _, exists := bead.Context["action_history"]; exists {
		t.Error("Expected action_history to be deleted after reset")
	}
	if _, exists := bead.Context["progress_metrics"]; exists {
		t.Error("Expected progress_metrics to be deleted after reset")
	}

	// Other context should remain
	bead.Context["other_key"] = "value"
	ld.ResetProgress(bead)
	if bead.Context["other_key"] != "value" {
		t.Error("Expected other context keys to remain after reset")
	}
}

// --- GetActionHistoryJSON edge cases ---

func TestGetActionHistoryJSON_WithData(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-history-json-data",
		Context: make(map[string]string),
	}

	_ = ld.RecordAction(bead, ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "edit_file",
		ActionData: map[string]interface{}{"file_path": "main.go"},
	})

	result := ld.GetActionHistoryJSON(bead)
	if result == "[]" {
		t.Error("Expected non-empty history JSON")
	}

	var parsed []ActionRecord
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if len(parsed) != 1 {
		t.Errorf("Expected 1 record, got %d", len(parsed))
	}
}

func TestGetActionHistoryJSON_InvalidJSON(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID: "b-bad-history-json",
		Context: map[string]string{
			"action_history": "not valid json",
		},
	}

	result := ld.GetActionHistoryJSON(bead)
	if result != "[]" {
		t.Errorf("Expected '[]' for invalid JSON, got %q", result)
	}
}

// --- SuggestNextSteps comprehensive ---

func TestSuggestNextSteps_EmptyHistory(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-no-history-suggest",
		Context: make(map[string]string),
	}

	suggestions := ld.SuggestNextSteps(bead, "agent did nothing")
	if len(suggestions) < 2 {
		t.Errorf("Expected at least 2 suggestions, got %d", len(suggestions))
	}

	// Should suggest reviewing description and providing context
	foundReview := false
	foundContext := false
	for _, s := range suggestions {
		lower := strings.ToLower(s)
		if strings.Contains(lower, "review") {
			foundReview = true
		}
		if strings.Contains(lower, "context") || strings.Contains(lower, "constraint") {
			foundContext = true
		}
	}
	if !foundReview {
		t.Error("Expected suggestion about reviewing description")
	}
	if !foundContext {
		t.Error("Expected suggestion about providing context")
	}
}

func TestSuggestNextSteps_NoReads(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-no-reads",
		Context: make(map[string]string),
	}

	// Record only bash commands, no reads
	for i := 0; i < 3; i++ {
		_ = ld.RecordAction(bead, ActionRecord{
			Timestamp:  time.Now(),
			AgentID:    "agent-1",
			ActionType: "bash",
			ActionData: map[string]interface{}{"command": "echo hello"},
		})
	}

	suggestions := ld.SuggestNextSteps(bead, "no file exploration")
	foundExplore := false
	for _, s := range suggestions {
		lower := strings.ToLower(s)
		if strings.Contains(lower, "explore") || strings.Contains(lower, "file") || strings.Contains(lower, "entry point") {
			foundExplore = true
			break
		}
	}
	if !foundExplore {
		t.Errorf("Expected suggestion about exploring codebase, got: %v", suggestions)
	}
}

func TestSuggestNextSteps_OnlyEditsNoTests(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-edits-only",
		Context: make(map[string]string),
	}

	_ = ld.RecordAction(bead, ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "read_file",
		ActionData: map[string]interface{}{"file_path": "a.go"},
	})
	_ = ld.RecordAction(bead, ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "read_file",
		ActionData: map[string]interface{}{"file_path": "b.go"},
	})
	_ = ld.RecordAction(bead, ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "edit_file",
		ActionData: map[string]interface{}{"file_path": "a.go"},
	})

	suggestions := ld.SuggestNextSteps(bead, "made edits but no tests")
	foundTest := false
	for _, s := range suggestions {
		if strings.Contains(strings.ToLower(s), "test") {
			foundTest = true
			break
		}
	}
	if !foundTest {
		t.Errorf("Expected suggestion about running tests, got: %v", suggestions)
	}
}

func TestSuggestNextSteps_FullWorkflowStillStuck(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-full-stuck",
		Context: make(map[string]string),
	}

	// Record a full set of action types
	actions := []ActionRecord{
		{Timestamp: time.Now(), AgentID: "a1", ActionType: "read_file", ActionData: map[string]interface{}{"file_path": "a.go"}},
		{Timestamp: time.Now(), AgentID: "a1", ActionType: "read_file", ActionData: map[string]interface{}{"file_path": "b.go"}},
		{Timestamp: time.Now(), AgentID: "a1", ActionType: "edit_file", ActionData: map[string]interface{}{"file_path": "a.go"}},
		{Timestamp: time.Now(), AgentID: "a1", ActionType: "run_tests", ActionData: map[string]interface{}{"command": "go test"}},
		{Timestamp: time.Now(), AgentID: "a1", ActionType: "bash", ActionData: map[string]interface{}{"command": "make build"}},
	}

	for _, action := range actions {
		_ = ld.RecordAction(bead, action)
	}

	suggestions := ld.SuggestNextSteps(bead, "tried everything")

	// Should have the general suggestions appended
	if len(suggestions) < 2 {
		t.Errorf("Expected at least 2 suggestions, got %d", len(suggestions))
	}

	// Should suggest breaking down or providing examples
	foundBreakdown := false
	foundExamples := false
	for _, s := range suggestions {
		lower := strings.ToLower(s)
		if strings.Contains(lower, "break down") || strings.Contains(lower, "subtask") {
			foundBreakdown = true
		}
		if strings.Contains(lower, "example") || strings.Contains(lower, "reference") {
			foundExamples = true
		}
	}
	if !foundBreakdown {
		t.Errorf("Expected suggestion about breaking down tasks, got: %v", suggestions)
	}
	if !foundExamples {
		t.Errorf("Expected suggestion about examples/references, got: %v", suggestions)
	}
}

// --- SetRepeatThreshold edge cases ---

func TestSetRepeatThreshold_BoundaryValues(t *testing.T) {
	ld := NewLoopDetector()

	// Test clamping for values below minimum
	ld.SetRepeatThreshold(-100)
	if ld.repeatThreshold != 2 {
		t.Errorf("Expected threshold clamped to 2 for -100, got %d", ld.repeatThreshold)
	}

	ld.SetRepeatThreshold(0)
	if ld.repeatThreshold != 2 {
		t.Errorf("Expected threshold clamped to 2 for 0, got %d", ld.repeatThreshold)
	}

	ld.SetRepeatThreshold(1)
	if ld.repeatThreshold != 2 {
		t.Errorf("Expected threshold clamped to 2 for 1, got %d", ld.repeatThreshold)
	}

	// Exactly at minimum
	ld.SetRepeatThreshold(2)
	if ld.repeatThreshold != 2 {
		t.Errorf("Expected threshold 2 for 2, got %d", ld.repeatThreshold)
	}

	// Above minimum
	ld.SetRepeatThreshold(100)
	if ld.repeatThreshold != 100 {
		t.Errorf("Expected threshold 100, got %d", ld.repeatThreshold)
	}
}

// --- findRepeatedPattern edge cases ---

func TestFindRepeatedPattern_LongHistory(t *testing.T) {
	ld := NewLoopDetector()

	// Create history with 20 entries, only last 15 should be checked
	history := make([]ActionRecord, 20)
	for i := 0; i < 20; i++ {
		key := "unique-key"
		if i >= 5 {
			key = "repeated-key" // last 15 entries have same key
		}
		history[i] = ActionRecord{ProgressKey: key}
	}

	pattern, count := ld.findRepeatedPattern(history)
	// Should detect the repeated pattern in last 15 entries
	if count < ld.repeatThreshold {
		t.Errorf("Expected count >= %d, got %d (pattern: %s)", ld.repeatThreshold, count, pattern)
	}
}

func TestFindRepeatedPattern_MultiplePatterns(t *testing.T) {
	ld := NewLoopDetector()

	// Create history with two repeated patterns
	history := []ActionRecord{
		{ProgressKey: "key1"},
		{ProgressKey: "key1"},
		{ProgressKey: "key1"},
		{ProgressKey: "key2"},
		{ProgressKey: "key2"},
		{ProgressKey: "key2"},
		{ProgressKey: "key2"},
		{ProgressKey: "key2"},
	}

	pattern, count := ld.findRepeatedPattern(history)
	// Should return the highest count pattern
	if count != 5 {
		t.Errorf("Expected count 5 for key2, got %d (pattern: %s)", count, pattern)
	}
}

// --- generateProgressKey comprehensive ---

func TestGenerateProgressKey_SameActionSameKey(t *testing.T) {
	ld := NewLoopDetector()

	action1 := ActionRecord{
		ActionType: "read_file",
		ActionData: map[string]interface{}{"file_path": "test.go"},
	}
	action2 := ActionRecord{
		ActionType: "read_file",
		ActionData: map[string]interface{}{"file_path": "test.go"},
	}

	key1 := ld.generateProgressKey(action1)
	key2 := ld.generateProgressKey(action2)

	if key1 != key2 {
		t.Errorf("Same actions should produce same key: %s != %s", key1, key2)
	}
}

func TestGenerateProgressKey_DifferentFilesDifferentKeys(t *testing.T) {
	ld := NewLoopDetector()

	action1 := ActionRecord{
		ActionType: "read_file",
		ActionData: map[string]interface{}{"file_path": "a.go"},
	}
	action2 := ActionRecord{
		ActionType: "read_file",
		ActionData: map[string]interface{}{"file_path": "b.go"},
	}

	key1 := ld.generateProgressKey(action1)
	key2 := ld.generateProgressKey(action2)

	if key1 == key2 {
		t.Error("Different files should produce different keys")
	}
}

func TestGenerateProgressKey_CommandPriority(t *testing.T) {
	ld := NewLoopDetector()

	// When both file_path and command are present, file_path takes priority
	action := ActionRecord{
		ActionType: "bash",
		ActionData: map[string]interface{}{
			"file_path": "test.go",
			"command":   "go test",
		},
	}

	key := ld.generateProgressKey(action)
	if key == "" {
		t.Error("Expected non-empty key")
	}
	if len(key) != 16 {
		t.Errorf("Expected 16 char hex key, got %d chars: %s", len(key), key)
	}
}

// --- UpdateProgressMetrics with existing metrics ---

func TestUpdateProgressMetrics_Accumulation(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-accumulate",
		Context: make(map[string]string),
	}

	// Record multiple file reads
	for i := 0; i < 5; i++ {
		ld.updateProgressMetrics(bead, ActionRecord{
			ActionType: "read_file",
			ActionData: map[string]interface{}{},
		})
	}

	var metrics ProgressMetrics
	if err := json.Unmarshal([]byte(bead.Context["progress_metrics"]), &metrics); err != nil {
		t.Fatalf("Failed to parse metrics: %v", err)
	}
	if metrics.FilesRead != 5 {
		t.Errorf("Expected FilesRead=5, got %d", metrics.FilesRead)
	}

	// Add some edits
	for i := 0; i < 3; i++ {
		ld.updateProgressMetrics(bead, ActionRecord{
			ActionType: "edit_file",
			ActionData: map[string]interface{}{},
		})
	}

	if err := json.Unmarshal([]byte(bead.Context["progress_metrics"]), &metrics); err != nil {
		t.Fatalf("Failed to parse metrics: %v", err)
	}
	if metrics.FilesRead != 5 {
		t.Errorf("Expected FilesRead to remain 5, got %d", metrics.FilesRead)
	}
	if metrics.FilesModified != 3 {
		t.Errorf("Expected FilesModified=3, got %d", metrics.FilesModified)
	}
}

func TestUpdateProgressMetrics_ExistingInvalidJSON(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID: "b-bad-existing",
		Context: map[string]string{
			"progress_metrics": "not valid json",
		},
	}

	// Should handle invalid existing metrics gracefully
	ld.updateProgressMetrics(bead, ActionRecord{
		ActionType: "read_file",
		ActionData: map[string]interface{}{},
	})

	var metrics ProgressMetrics
	if err := json.Unmarshal([]byte(bead.Context["progress_metrics"]), &metrics); err != nil {
		t.Fatalf("Expected valid JSON after update, got error: %v", err)
	}
	if metrics.FilesRead != 1 {
		t.Errorf("Expected FilesRead=1 after fresh start, got %d", metrics.FilesRead)
	}
}

// --- GetAgentCommitRange edge cases ---

func TestGetAgentCommitRange_ZeroCount(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID: "b-zero-count",
		Context: map[string]string{
			"agent_first_commit_sha": "abc123",
			"agent_last_commit_sha":  "def456",
			"agent_commit_count":     "0",
		},
	}

	first, last, count := ld.GetAgentCommitRange(bead)
	if first != "abc123" {
		t.Errorf("Expected firstSHA 'abc123', got %q", first)
	}
	if last != "def456" {
		t.Errorf("Expected lastSHA 'def456', got %q", last)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

func TestGetAgentCommitRange_LargeCount(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID: "b-large-count",
		Context: map[string]string{
			"agent_first_commit_sha": "start",
			"agent_last_commit_sha":  "end",
			"agent_commit_count":     "9999",
		},
	}

	_, _, count := ld.GetAgentCommitRange(bead)
	if count != 9999 {
		t.Errorf("Expected count 9999, got %d", count)
	}
}

// --- hasRecentProgress edge cases ---

func TestHasRecentProgress_ExactlyFiveMinutes(t *testing.T) {
	ld := NewLoopDetector()

	metrics := ProgressMetrics{
		FilesRead:    1,
		LastProgress: time.Now().Add(-5*time.Minute - 1*time.Second), // just over 5 min
	}
	data, _ := json.Marshal(metrics)

	bead := &models.Bead{
		ID: "b-exact-5min",
		Context: map[string]string{
			"progress_metrics": string(data),
		},
	}

	result := ld.hasRecentProgress(bead)
	if result {
		t.Error("Expected no recent progress when exactly at 5 minute boundary")
	}
}

func TestHasRecentProgress_JustUnderFiveMinutes(t *testing.T) {
	ld := NewLoopDetector()

	metrics := ProgressMetrics{
		FilesRead:    1,
		LastProgress: time.Now().Add(-4*time.Minute - 59*time.Second), // just under 5 min
	}
	data, _ := json.Marshal(metrics)

	bead := &models.Bead{
		ID: "b-under-5min",
		Context: map[string]string{
			"progress_metrics": string(data),
		},
	}

	result := ld.hasRecentProgress(bead)
	if !result {
		t.Error("Expected recent progress when just under 5 minutes")
	}
}
