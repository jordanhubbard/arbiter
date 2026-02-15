package git

import (
	"testing"
)

func TestParseCommitMetadata(t *testing.T) {
	tests := []struct {
		name      string
		commitMsg string
		checkFn   func(t *testing.T, meta *CommitMetadata)
	}{
		{
			name: "full metadata",
			commitMsg: `Fix authentication bug

Bead: loom-abc123
Agent: agent-456
Project: myapp
Dispatch: 5
Progress: files_modified=3, tests_run=2`,
			checkFn: func(t *testing.T, meta *CommitMetadata) {
				if meta.Subject != "Fix authentication bug" {
					t.Errorf("Subject: expected 'Fix authentication bug', got %q", meta.Subject)
				}
				if meta.BeadID != "loom-abc123" {
					t.Errorf("BeadID: expected 'loom-abc123', got %q", meta.BeadID)
				}
				if meta.AgentID != "agent-456" {
					t.Errorf("AgentID: expected 'agent-456', got %q", meta.AgentID)
				}
				if meta.ProjectID != "myapp" {
					t.Errorf("ProjectID: expected 'myapp', got %q", meta.ProjectID)
				}
				if meta.Dispatch != 5 {
					t.Errorf("Dispatch: expected 5, got %d", meta.Dispatch)
				}
				if meta.Progress["files_modified"] != 3 {
					t.Errorf("Progress[files_modified]: expected 3, got %d", meta.Progress["files_modified"])
				}
				if meta.Progress["tests_run"] != 2 {
					t.Errorf("Progress[tests_run]: expected 2, got %d", meta.Progress["tests_run"])
				}
			},
		},
		{
			name: "partial metadata - only bead",
			commitMsg: `Update config

Bead: bead-only`,
			checkFn: func(t *testing.T, meta *CommitMetadata) {
				if meta.Subject != "Update config" {
					t.Errorf("Subject: expected 'Update config', got %q", meta.Subject)
				}
				if meta.BeadID != "bead-only" {
					t.Errorf("BeadID: expected 'bead-only', got %q", meta.BeadID)
				}
				if meta.AgentID != "" {
					t.Errorf("AgentID should be empty, got %q", meta.AgentID)
				}
				if meta.ProjectID != "" {
					t.Errorf("ProjectID should be empty, got %q", meta.ProjectID)
				}
			},
		},
		{
			name:      "no metadata",
			commitMsg: "Simple commit message\nWith a body but no trailers",
			checkFn: func(t *testing.T, meta *CommitMetadata) {
				if meta.Subject != "Simple commit message" {
					t.Errorf("Subject: expected 'Simple commit message', got %q", meta.Subject)
				}
				if meta.BeadID != "" {
					t.Errorf("BeadID should be empty, got %q", meta.BeadID)
				}
				if meta.AgentID != "" {
					t.Errorf("AgentID should be empty, got %q", meta.AgentID)
				}
			},
		},
		{
			name:      "empty message",
			commitMsg: "",
			checkFn: func(t *testing.T, meta *CommitMetadata) {
				if meta.Subject != "" {
					t.Errorf("Subject should be empty, got %q", meta.Subject)
				}
			},
		},
		{
			name:      "single line",
			commitMsg: "One liner",
			checkFn: func(t *testing.T, meta *CommitMetadata) {
				if meta.Subject != "One liner" {
					t.Errorf("Subject: expected 'One liner', got %q", meta.Subject)
				}
			},
		},
		{
			name: "dispatch with non-numeric value",
			commitMsg: `Fix bug
Dispatch: invalid`,
			checkFn: func(t *testing.T, meta *CommitMetadata) {
				if meta.Dispatch != 0 {
					t.Errorf("Dispatch should be 0 for non-numeric, got %d", meta.Dispatch)
				}
			},
		},
		{
			name: "progress with mixed valid and invalid",
			commitMsg: `Fix
Progress: valid=10, invalid=abc, another=20`,
			checkFn: func(t *testing.T, meta *CommitMetadata) {
				if meta.Progress["valid"] != 10 {
					t.Errorf("Progress[valid]: expected 10, got %d", meta.Progress["valid"])
				}
				if _, ok := meta.Progress["invalid"]; ok {
					t.Error("Progress[invalid] should not exist")
				}
				if meta.Progress["another"] != 20 {
					t.Errorf("Progress[another]: expected 20, got %d", meta.Progress["another"])
				}
			},
		},
		{
			name: "whitespace in trailers",
			commitMsg: `Commit subject
  Bead:   loom-padded
  Agent:   agent-padded  `,
			checkFn: func(t *testing.T, meta *CommitMetadata) {
				if meta.BeadID != "loom-padded" {
					t.Errorf("BeadID: expected 'loom-padded', got %q", meta.BeadID)
				}
				if meta.AgentID != "agent-padded" {
					t.Errorf("AgentID: expected 'agent-padded', got %q", meta.AgentID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := ParseCommitMetadata(tt.commitMsg)
			if meta == nil {
				t.Fatal("expected non-nil metadata")
			}
			tt.checkFn(t, meta)
		})
	}
}

func TestExtractTrailer(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		key      string
		expected string
	}{
		{"matching trailer", "Bead: loom-123", "Bead:", "loom-123"},
		{"non-matching trailer", "Agent: agent-1", "Bead:", ""},
		{"empty line", "", "Bead:", ""},
		{"key without value", "Bead:", "Bead:", ""},
		{"key with spaces in value", "Bead:   spaced-value  ", "Bead:", "spaced-value"},
		{"partial key match", "BeadID: not-this", "Bead:", ""},
		{"case sensitive", "bead: lower", "Bead:", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTrailer(tt.line, tt.key)
			if got != tt.expected {
				t.Errorf("extractTrailer(%q, %q) = %q, want %q", tt.line, tt.key, got, tt.expected)
			}
		})
	}
}

func TestParseProgressTrailer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]int
	}{
		{
			name:  "standard format",
			input: "files_modified=3, tests_run=2",
			expected: map[string]int{
				"files_modified": 3,
				"tests_run":      2,
			},
		},
		{
			name:  "single entry",
			input: "count=42",
			expected: map[string]int{
				"count": 42,
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: map[string]int{},
		},
		{
			name:     "non-numeric values ignored",
			input:    "valid=10, invalid=abc",
			expected: map[string]int{"valid": 10},
		},
		{
			name:     "no equals sign",
			input:    "just-text",
			expected: map[string]int{},
		},
		{
			name:  "extra whitespace",
			input: "  key1 = 100 , key2 = 200  ",
			expected: map[string]int{
				"key1": 100,
				"key2": 200,
			},
		},
		{
			name:     "zero values",
			input:    "count=0",
			expected: map[string]int{"count": 0},
		},
		{
			name:     "negative values",
			input:    "diff=-5",
			expected: map[string]int{"diff": -5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseProgressTrailer(tt.input)
			for key, expectedVal := range tt.expected {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("expected key %q in result", key)
					continue
				}
				if gotVal != expectedVal {
					t.Errorf("key %q: expected %d, got %d", key, expectedVal, gotVal)
				}
			}
			for key := range got {
				if _, ok := tt.expected[key]; !ok {
					t.Errorf("unexpected key %q in result", key)
				}
			}
		})
	}
}

func TestCommitMetadataStruct(t *testing.T) {
	meta := CommitMetadata{
		SHA:       "abc123",
		BeadID:    "bead-1",
		AgentID:   "agent-1",
		ProjectID: "project-1",
		Dispatch:  3,
		Progress:  map[string]int{"files": 5},
		Subject:   "Test commit",
	}

	if meta.SHA != "abc123" {
		t.Errorf("SHA: expected abc123, got %s", meta.SHA)
	}
	if meta.BeadID != "bead-1" {
		t.Errorf("BeadID: expected bead-1, got %s", meta.BeadID)
	}
	if meta.AgentID != "agent-1" {
		t.Errorf("AgentID: expected agent-1, got %s", meta.AgentID)
	}
	if meta.ProjectID != "project-1" {
		t.Errorf("ProjectID: expected project-1, got %s", meta.ProjectID)
	}
	if meta.Dispatch != 3 {
		t.Errorf("Dispatch: expected 3, got %d", meta.Dispatch)
	}
	if meta.Progress["files"] != 5 {
		t.Errorf("Progress[files]: expected 5, got %d", meta.Progress["files"])
	}
	if meta.Subject != "Test commit" {
		t.Errorf("Subject: expected 'Test commit', got %s", meta.Subject)
	}
}

func TestAuditLoggerStruct(t *testing.T) {
	logger := &AuditLogger{
		projectID: "test-project",
		logPath:   "/tmp/test.log",
	}

	if logger.projectID != "test-project" {
		t.Errorf("projectID: expected test-project, got %s", logger.projectID)
	}
	if logger.logPath != "/tmp/test.log" {
		t.Errorf("logPath: expected /tmp/test.log, got %s", logger.logPath)
	}
}

func TestGitServiceStruct(t *testing.T) {
	svc := &GitService{
		projectPath:   "/path/to/project",
		projectID:     "project-1",
		projectKeyDir: "/keys",
		branchPrefix:  "agent/",
	}

	if svc.projectPath != "/path/to/project" {
		t.Errorf("projectPath: expected /path/to/project, got %s", svc.projectPath)
	}
	if svc.projectID != "project-1" {
		t.Errorf("projectID: expected project-1, got %s", svc.projectID)
	}
	if svc.projectKeyDir != "/keys" {
		t.Errorf("projectKeyDir: expected /keys, got %s", svc.projectKeyDir)
	}
	if svc.branchPrefix != "agent/" {
		t.Errorf("branchPrefix: expected agent/, got %s", svc.branchPrefix)
	}
}
