package dispatch

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

func TestGetAgentCommitRange(t *testing.T) {
	ld := NewLoopDetector()

	tests := []struct {
		name          string
		bead          *models.Bead
		expectedFirst string
		expectedLast  string
		expectedCount int
	}{
		{
			name:          "nil bead",
			bead:          nil,
			expectedFirst: "",
			expectedLast:  "",
			expectedCount: 0,
		},
		{
			name: "nil context",
			bead: &models.Bead{
				ID:      "b-1",
				Context: nil,
			},
			expectedFirst: "",
			expectedLast:  "",
			expectedCount: 0,
		},
		{
			name: "empty context",
			bead: &models.Bead{
				ID:      "b-2",
				Context: map[string]string{},
			},
			expectedFirst: "",
			expectedLast:  "",
			expectedCount: 0,
		},
		{
			name: "with commit range",
			bead: &models.Bead{
				ID: "b-3",
				Context: map[string]string{
					"agent_first_commit_sha": "abc123",
					"agent_last_commit_sha":  "def456",
					"agent_commit_count":     "5",
				},
			},
			expectedFirst: "abc123",
			expectedLast:  "def456",
			expectedCount: 5,
		},
		{
			name: "partial commit info - only first sha",
			bead: &models.Bead{
				ID: "b-4",
				Context: map[string]string{
					"agent_first_commit_sha": "abc123",
				},
			},
			expectedFirst: "abc123",
			expectedLast:  "",
			expectedCount: 0,
		},
		{
			name: "invalid commit count",
			bead: &models.Bead{
				ID: "b-5",
				Context: map[string]string{
					"agent_first_commit_sha": "abc123",
					"agent_last_commit_sha":  "def456",
					"agent_commit_count":     "invalid",
				},
			},
			expectedFirst: "abc123",
			expectedLast:  "def456",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, last, count := ld.GetAgentCommitRange(tt.bead)
			if first != tt.expectedFirst {
				t.Errorf("firstSHA = %q, want %q", first, tt.expectedFirst)
			}
			if last != tt.expectedLast {
				t.Errorf("lastSHA = %q, want %q", last, tt.expectedLast)
			}
			if count != tt.expectedCount {
				t.Errorf("count = %d, want %d", count, tt.expectedCount)
			}
		})
	}
}

func TestGetActionHistory_EmptyCases(t *testing.T) {
	ld := NewLoopDetector()

	tests := []struct {
		name     string
		bead     *models.Bead
		expected int
	}{
		{
			name: "nil context",
			bead: &models.Bead{
				ID:      "b-1",
				Context: nil,
			},
			expected: 0,
		},
		{
			name: "empty action_history",
			bead: &models.Bead{
				ID: "b-2",
				Context: map[string]string{
					"action_history": "",
				},
			},
			expected: 0,
		},
		{
			name: "valid empty array",
			bead: &models.Bead{
				ID: "b-3",
				Context: map[string]string{
					"action_history": "[]",
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history, err := ld.getActionHistory(tt.bead)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(history) != tt.expected {
				t.Errorf("Expected %d entries, got %d", tt.expected, len(history))
			}
		})
	}
}

func TestGetActionHistory_InvalidJSON(t *testing.T) {
	ld := NewLoopDetector()

	bead := &models.Bead{
		ID: "b-invalid",
		Context: map[string]string{
			"action_history": "not valid json",
		},
	}

	_, err := ld.getActionHistory(bead)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestHasRecentProgress(t *testing.T) {
	ld := NewLoopDetector()

	tests := []struct {
		name     string
		bead     *models.Bead
		expected bool
	}{
		{
			name: "nil context",
			bead: &models.Bead{
				ID:      "b-1",
				Context: nil,
			},
			expected: false,
		},
		{
			name: "empty progress_metrics",
			bead: &models.Bead{
				ID: "b-2",
				Context: map[string]string{
					"progress_metrics": "",
				},
			},
			expected: false,
		},
		{
			name: "invalid JSON in progress_metrics",
			bead: &models.Bead{
				ID: "b-3",
				Context: map[string]string{
					"progress_metrics": "invalid",
				},
			},
			expected: false,
		},
		{
			name: "zero time in metrics",
			bead: &models.Bead{
				ID: "b-4",
				Context: map[string]string{
					"progress_metrics": `{"files_read":1,"files_modified":0,"tests_run":0,"commands_executed":0}`,
				},
			},
			expected: false,
		},
		{
			name: "recent progress",
			bead: func() *models.Bead {
				metrics := ProgressMetrics{
					FilesRead:    5,
					LastProgress: time.Now(),
				}
				data, _ := json.Marshal(metrics)
				return &models.Bead{
					ID: "b-5",
					Context: map[string]string{
						"progress_metrics": string(data),
					},
				}
			}(),
			expected: true,
		},
		{
			name: "old progress",
			bead: func() *models.Bead {
				metrics := ProgressMetrics{
					FilesRead:    5,
					LastProgress: time.Now().Add(-10 * time.Minute),
				}
				data, _ := json.Marshal(metrics)
				return &models.Bead{
					ID: "b-6",
					Context: map[string]string{
						"progress_metrics": string(data),
					},
				}
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ld.hasRecentProgress(tt.bead)
			if result != tt.expected {
				t.Errorf("hasRecentProgress() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindRepeatedPattern(t *testing.T) {
	ld := NewLoopDetector()

	tests := []struct {
		name          string
		history       []ActionRecord
		expectedCount int
		hasPattern    bool
	}{
		{
			name:          "empty history",
			history:       []ActionRecord{},
			expectedCount: 0,
			hasPattern:    false,
		},
		{
			name: "short history - below threshold",
			history: []ActionRecord{
				{ProgressKey: "key1"},
				{ProgressKey: "key2"},
			},
			expectedCount: 0,
			hasPattern:    false,
		},
		{
			name: "repeated pattern at threshold",
			history: []ActionRecord{
				{ProgressKey: "key1"},
				{ProgressKey: "key1"},
				{ProgressKey: "key1"},
			},
			expectedCount: 3,
			hasPattern:    true,
		},
		{
			name: "mixed keys - no repeating pattern",
			history: []ActionRecord{
				{ProgressKey: "key1"},
				{ProgressKey: "key2"},
				{ProgressKey: "key3"},
				{ProgressKey: "key4"},
			},
			expectedCount: 0,
			hasPattern:    false,
		},
		{
			name: "alternating keys - no consecutive repeat",
			history: []ActionRecord{
				{ProgressKey: "key1"},
				{ProgressKey: "key2"},
				{ProgressKey: "key1"},
				{ProgressKey: "key2"},
				{ProgressKey: "key1"},
				{ProgressKey: "key2"},
			},
			expectedCount: 0,
			hasPattern:    false,
		},
		{
			name: "consecutive repeats exceed threshold",
			history: []ActionRecord{
				{ProgressKey: "key1"},
				{ProgressKey: "key2"},
				{ProgressKey: "key2"},
				{ProgressKey: "key2"},
				{ProgressKey: "key2"},
			},
			expectedCount: 4,
			hasPattern:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, count := ld.findRepeatedPattern(tt.history)
			if tt.hasPattern {
				if count < ld.repeatThreshold {
					t.Errorf("Expected count >= %d, got %d", ld.repeatThreshold, count)
				}
				if count != tt.expectedCount {
					t.Errorf("Expected count %d, got %d", tt.expectedCount, count)
				}
				if pattern == "" {
					t.Error("Expected non-empty pattern")
				}
			} else {
				if count >= ld.repeatThreshold {
					t.Errorf("Expected count < %d, got %d", ld.repeatThreshold, count)
				}
			}
		})
	}
}

func TestUpdateProgressMetrics(t *testing.T) {
	ld := NewLoopDetector()

	tests := []struct {
		name       string
		actionType string
		checkField string
	}{
		{name: "read_file", actionType: "read_file", checkField: "files_read"},
		{name: "glob", actionType: "glob", checkField: "files_read"},
		{name: "grep", actionType: "grep", checkField: "files_read"},
		{name: "edit_file", actionType: "edit_file", checkField: "files_modified"},
		{name: "write_file", actionType: "write_file", checkField: "files_modified"},
		{name: "run_tests", actionType: "run_tests", checkField: "tests_run"},
		{name: "test", actionType: "test", checkField: "tests_run"},
		{name: "bash", actionType: "bash", checkField: "commands_executed"},
		{name: "execute", actionType: "execute", checkField: "commands_executed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bead := &models.Bead{
				ID:      "b-metrics-" + tt.name,
				Context: make(map[string]string),
			}

			action := ActionRecord{
				Timestamp:  time.Now(),
				AgentID:    "agent-1",
				ActionType: tt.actionType,
				ActionData: map[string]interface{}{},
			}

			ld.updateProgressMetrics(bead, action)

			metricsJSON := bead.Context["progress_metrics"]
			if metricsJSON == "" {
				t.Fatal("Expected progress_metrics to be set")
			}

			var metrics ProgressMetrics
			if err := json.Unmarshal([]byte(metricsJSON), &metrics); err != nil {
				t.Fatalf("Failed to parse metrics: %v", err)
			}

			switch tt.checkField {
			case "files_read":
				if metrics.FilesRead == 0 {
					t.Error("Expected FilesRead > 0")
				}
			case "files_modified":
				if metrics.FilesModified == 0 {
					t.Error("Expected FilesModified > 0")
				}
			case "tests_run":
				if metrics.TestsRun == 0 {
					t.Error("Expected TestsRun > 0")
				}
			case "commands_executed":
				if metrics.CommandsExecuted == 0 {
					t.Error("Expected CommandsExecuted > 0")
				}
			}

			if metrics.LastProgress.IsZero() {
				t.Error("Expected LastProgress to be set")
			}
		})
	}
}

func TestUpdateProgressMetrics_NilContext(t *testing.T) {
	ld := NewLoopDetector()

	bead := &models.Bead{
		ID:      "b-nil-ctx",
		Context: nil,
	}

	action := ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "read_file",
		ActionData: map[string]interface{}{},
	}

	// Should not panic
	ld.updateProgressMetrics(bead, action)

	if bead.Context == nil {
		t.Error("Expected context to be initialized")
	}

	if bead.Context["progress_metrics"] == "" {
		t.Error("Expected progress_metrics to be set")
	}
}

func TestUpdateProgressMetrics_UnknownAction(t *testing.T) {
	ld := NewLoopDetector()

	bead := &models.Bead{
		ID:      "b-unknown",
		Context: make(map[string]string),
	}

	action := ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "unknown_action",
		ActionData: map[string]interface{}{},
	}

	ld.updateProgressMetrics(bead, action)

	metricsJSON := bead.Context["progress_metrics"]
	if metricsJSON == "" {
		t.Fatal("Expected progress_metrics to be set even for unknown action")
	}

	var metrics ProgressMetrics
	if err := json.Unmarshal([]byte(metricsJSON), &metrics); err != nil {
		t.Fatalf("Failed to parse metrics: %v", err)
	}

	// Unknown action should not increment any counters
	if metrics.FilesRead != 0 || metrics.FilesModified != 0 ||
		metrics.TestsRun != 0 || metrics.CommandsExecuted != 0 {
		t.Error("Unknown action should not increment any counters")
	}

	// LastProgress should not be set for unknown action
	if !metrics.LastProgress.IsZero() {
		t.Error("LastProgress should not be set for unknown action type")
	}
}

func TestRecordAction_NilContext(t *testing.T) {
	ld := NewLoopDetector()

	bead := &models.Bead{
		ID:      "b-nil-ctx-record",
		Context: nil,
	}

	action := ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "read_file",
		ActionData: map[string]interface{}{"file_path": "test.go"},
	}

	err := ld.RecordAction(bead, action)
	if err != nil {
		t.Fatalf("RecordAction failed: %v", err)
	}

	if bead.Context == nil {
		t.Error("Expected context to be initialized")
	}
}

func TestGetProgressSummary_InvalidJSON(t *testing.T) {
	ld := NewLoopDetector()

	bead := &models.Bead{
		ID: "b-invalid-summary",
		Context: map[string]string{
			"progress_metrics": "invalid json",
		},
	}

	result := ld.GetProgressSummary(bead)
	if result != "Invalid progress data" {
		t.Errorf("Expected 'Invalid progress data', got %q", result)
	}
}

func TestGetProgressSummary_NilContext(t *testing.T) {
	ld := NewLoopDetector()

	bead := &models.Bead{
		ID:      "b-nil-summary",
		Context: nil,
	}

	result := ld.GetProgressSummary(bead)
	if result != "No progress data" {
		t.Errorf("Expected 'No progress data', got %q", result)
	}
}

func TestGetProgressSummary_ZeroLastProgress(t *testing.T) {
	ld := NewLoopDetector()

	metrics := ProgressMetrics{
		FilesRead:     3,
		FilesModified: 1,
		TestsRun:      2,
	}
	data, _ := json.Marshal(metrics)

	bead := &models.Bead{
		ID: "b-zero-progress",
		Context: map[string]string{
			"progress_metrics": string(data),
		},
	}

	result := ld.GetProgressSummary(bead)
	if result == "No progress data" || result == "Invalid progress data" {
		t.Errorf("Expected valid summary, got %q", result)
	}
	if result == "" {
		t.Error("Expected non-empty summary")
	}
}

func TestResetProgress_NilContext(t *testing.T) {
	ld := NewLoopDetector()

	bead := &models.Bead{
		ID:      "b-nil-reset",
		Context: nil,
	}

	// Should not panic
	ld.ResetProgress(bead)
}

func TestGetActionHistoryJSON_NilContext(t *testing.T) {
	ld := NewLoopDetector()

	bead := &models.Bead{
		ID:      "b-nil-history-json",
		Context: nil,
	}

	result := ld.GetActionHistoryJSON(bead)
	if result != "[]" {
		t.Errorf("Expected '[]', got %q", result)
	}
}

func TestSuggestNextSteps_NoCommands(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "b-no-commands",
		Context: make(map[string]string),
	}

	// Record reads and edits but no bash/execute commands
	actions := []ActionRecord{
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "read_file", ActionData: map[string]interface{}{"file_path": "a.go"}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "read_file", ActionData: map[string]interface{}{"file_path": "b.go"}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "edit_file", ActionData: map[string]interface{}{"file_path": "a.go"}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "run_tests", ActionData: map[string]interface{}{"command": "go test"}},
	}

	for _, action := range actions {
		_ = ld.RecordAction(bead, action)
	}

	suggestions := ld.SuggestNextSteps(bead, "no bash commands")

	// Should suggest providing build/test/debug commands
	found := false
	for _, s := range suggestions {
		if containsAny(s, []string{"command", "build", "debug"}) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected suggestion about commands, got: %v", suggestions)
	}
}

func containsAny(s string, substrings []string) bool {
	lower := toLower(s)
	for _, sub := range substrings {
		if containsLower(lower, sub) {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := range s {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

func containsLower(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestGenerateProgressKey_EdgeCases(t *testing.T) {
	ld := NewLoopDetector()

	tests := []struct {
		name   string
		action ActionRecord
	}{
		{
			name: "empty action data",
			action: ActionRecord{
				ActionType: "read_file",
				ActionData: map[string]interface{}{},
			},
		},
		{
			name: "nil action data",
			action: ActionRecord{
				ActionType: "bash",
				ActionData: nil,
			},
		},
		{
			name: "non-string file_path",
			action: ActionRecord{
				ActionType: "read_file",
				ActionData: map[string]interface{}{"file_path": 123},
			},
		},
		{
			name: "non-string command",
			action: ActionRecord{
				ActionType: "bash",
				ActionData: map[string]interface{}{"command": 456},
			},
		},
		{
			name: "both file_path and command",
			action: ActionRecord{
				ActionType: "bash",
				ActionData: map[string]interface{}{
					"file_path": "test.go",
					"command":   "go test",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := ld.generateProgressKey(tt.action)
			if key == "" {
				t.Error("Expected non-empty progress key")
			}
			// Key should be 16 hex characters (8 bytes)
			if len(key) != 16 {
				t.Errorf("Expected key length 16, got %d: %s", len(key), key)
			}
		})
	}
}
