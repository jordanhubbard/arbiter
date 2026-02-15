package git

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple lowercase", "hello world", "hello-world"},
		{"uppercase", "Hello World", "hello-world"},
		{"underscores", "hello_world", "hello-world"},
		{"special chars", "hello@world#123", "helloworld123"},
		{"multiple spaces", "hello   world", "hello-world"},
		{"leading trailing spaces", "  hello world  ", "hello-world"},
		{"consecutive hyphens", "hello---world", "hello-world"},
		{"mixed case with numbers", "Feature123 Add Button", "feature123-add-button"},
		{"empty string", "", ""},
		{"only special chars", "@#$%", ""},
		{"unicode chars", "hello\u00e9world", "helloworld"},
		{"already slugified", "hello-world", "hello-world"},
		{"leading hyphens", "---hello", "hello"},
		{"trailing hyphens", "hello---", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsProtectedBranch(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		expected   bool
	}{
		{"main", "main", true},
		{"master", "master", true},
		{"production", "production", true},
		{"release/1.0", "release/1.0", true},
		{"release/2.0.1", "release/2.0.1", true},
		{"hotfix/urgent", "hotfix/urgent", true},
		{"feature/my-feature", "feature/my-feature", false},
		{"agent/bead-123/task", "agent/bead-123/task", false},
		{"develop", "develop", false},
		{"staging", "staging", false},
		{"main-backup", "main-backup", false},
		{"not-main", "not-main", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isProtectedBranch(tt.branchName)
			if got != tt.expected {
				t.Errorf("isProtectedBranch(%q) = %v, want %v", tt.branchName, got, tt.expected)
			}
		})
	}
}

func TestHasSecrets(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "no secrets",
			content:  "This is normal code\nvar x = 42\n",
			expected: false,
		},
		{
			name:     "api key pattern with equals",
			content:  `api_key="abcdefghijklmnopqrstuvwxyz"`,
			expected: true,
		},
		{
			name:     "api key pattern with colon",
			content:  `api_key: 'abcdefghijklmnopqrstuvwxyz'`,
			expected: true,
		},
		{
			name:     "secret key pattern",
			content:  `secret_key="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef"`,
			expected: true,
		},
		{
			name:     "token pattern",
			content:  `token="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef"`,
			expected: true,
		},
		{
			name:     "aws access key",
			content:  `aws_access_key_id="AKIAIOSFODNN7EXAMPLE"`,
			expected: true,
		},
		{
			name:     "rsa private key",
			content:  "-----BEGIN RSA PRIVATE KEY-----\nsome data\n-----END RSA PRIVATE KEY-----",
			expected: true,
		},
		{
			name:     "ec private key",
			content:  "-----BEGIN EC PRIVATE KEY-----\nsome data\n-----END EC PRIVATE KEY-----",
			expected: true,
		},
		{
			name:     "openssh private key",
			content:  "-----BEGIN OPENSSH PRIVATE KEY-----\nsome data\n-----END OPENSSH PRIVATE KEY-----",
			expected: true,
		},
		{
			name:     "short api key value (not matched)",
			content:  `api_key = "short"`,
			expected: false,
		},
		{
			name:     "empty content",
			content:  "",
			expected: false,
		},
		{
			name:     "regular go code",
			content:  `func main() {\n\tfmt.Println("hello")\n}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasSecrets([]byte(tt.content))
			if got != tt.expected {
				t.Errorf("hasSecrets() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidateBranchNameWithPrefix(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		prefix     string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid agent branch",
			branchName: "agent/bead-123/fix-bug",
			prefix:     "agent/",
			wantErr:    false,
		},
		{
			name:       "wrong prefix",
			branchName: "feature/my-feature",
			prefix:     "agent/",
			wantErr:    true,
			errMsg:     "must start with",
		},
		{
			name:       "branch name too long",
			branchName: "agent/" + string(make([]byte, 70)),
			prefix:     "agent/",
			wantErr:    true,
			errMsg:     "too long",
		},
		{
			name:       "branch with spaces",
			branchName: "agent/bad branch",
			prefix:     "agent/",
			wantErr:    true,
			errMsg:     "whitespace",
		},
		{
			name:       "branch with tab",
			branchName: "agent/bad\tbranch",
			prefix:     "agent/",
			wantErr:    true,
			errMsg:     "whitespace",
		},
		{
			name:       "branch with newline",
			branchName: "agent/bad\nbranch",
			prefix:     "agent/",
			wantErr:    true,
			errMsg:     "whitespace",
		},
		{
			name:       "exact max length",
			branchName: "agent/" + string(make([]byte, 66)), // 72 total
			prefix:     "agent/",
			wantErr:    false,
		},
		{
			name:       "custom prefix",
			branchName: "custom/branch-1",
			prefix:     "custom/",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill any byte arrays with valid chars
			cleaned := make([]byte, len(tt.branchName))
			for i := range tt.branchName {
				if tt.branchName[i] == 0 {
					cleaned[i] = 'a'
				} else {
					cleaned[i] = tt.branchName[i]
				}
			}
			err := validateBranchNameWithPrefix(string(cleaned), tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBranchNameWithPrefix() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnsureCommitMetadata(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		beadID   string
		agentID  string
		checkFn  func(t *testing.T, result string)
	}{
		{
			name:    "empty message gets default",
			message: "",
			beadID:  "bead-1",
			agentID: "agent-1",
			checkFn: func(t *testing.T, result string) {
				if result == "" {
					t.Error("expected non-empty result")
				}
				if !containsStr(result, "Update from agent") {
					t.Error("expected default message")
				}
				if !containsStr(result, "Bead: bead-1") {
					t.Error("expected bead trailer")
				}
				if !containsStr(result, "Agent: agent-1") {
					t.Error("expected agent trailer")
				}
			},
		},
		{
			name:    "appends bead and agent trailers",
			message: "Fix authentication bug",
			beadID:  "bead-abc",
			agentID: "agent-xyz",
			checkFn: func(t *testing.T, result string) {
				if !containsStr(result, "Bead: bead-abc") {
					t.Error("expected bead trailer")
				}
				if !containsStr(result, "Agent: agent-xyz") {
					t.Error("expected agent trailer")
				}
			},
		},
		{
			name:    "skips bead if already present",
			message: "Fix bug\n\nBead: bead-existing",
			beadID:  "bead-existing",
			agentID: "agent-1",
			checkFn: func(t *testing.T, result string) {
				// Should have Agent trailer
				if !containsStr(result, "Agent: agent-1") {
					t.Error("expected agent trailer")
				}
			},
		},
		{
			name:    "skips agent if already present",
			message: "Fix bug\n\nAgent: existing-agent",
			beadID:  "bead-1",
			agentID: "agent-new",
			checkFn: func(t *testing.T, result string) {
				// Should have Bead trailer but not duplicate Agent
				if !containsStr(result, "Bead: bead-1") {
					t.Error("expected bead trailer")
				}
			},
		},
		{
			name:    "truncates long first line",
			message: "This is a very long commit message that exceeds the seventy-two character limit that we enforce for readability",
			beadID:  "",
			agentID: "",
			checkFn: func(t *testing.T, result string) {
				lines := splitLines(result)
				if len(lines[0]) > 72 {
					t.Errorf("first line too long: %d chars", len(lines[0]))
				}
				if len(lines[0]) < 60 {
					t.Errorf("first line too short after truncation: %d chars", len(lines[0]))
				}
			},
		},
		{
			name:    "empty bead and agent IDs",
			message: "Simple commit",
			beadID:  "",
			agentID: "",
			checkFn: func(t *testing.T, result string) {
				if containsStr(result, "Bead:") {
					t.Error("should not have bead trailer with empty beadID")
				}
				if containsStr(result, "Agent:") {
					t.Error("should not have agent trailer with empty agentID")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureCommitMetadata(tt.message, tt.beadID, tt.agentID)
			tt.checkFn(t, result)
		})
	}
}

func TestExtractPRNumberComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{"standard url", "https://github.com/owner/repo/pull/42", 42},
		{"large number", "https://github.com/owner/repo/pull/12345", 12345},
		{"with trailing slash", "https://github.com/owner/repo/pull/99/", 99},
		{"with query params", "https://github.com/owner/repo/pull/7?tab=files", 7},
		{"no pull path", "https://github.com/owner/repo", 0},
		{"issues url", "https://github.com/owner/repo/issues/10", 0},
		{"empty url", "", 0},
		{"not github", "https://gitlab.com/owner/repo/merge_requests/1", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPRNumber(tt.url)
			if got != tt.expected {
				t.Errorf("extractPRNumber(%q) = %d, want %d", tt.url, got, tt.expected)
			}
		})
	}
}

func TestIsGhCLIAvailableDoesNotPanic(t *testing.T) {
	// Just verify it doesn't panic
	_ = isGhCLIAvailable()
}

func TestGenerateBranchName(t *testing.T) {
	svc := &GitService{
		branchPrefix: "agent/",
	}

	tests := []struct {
		name        string
		beadID      string
		description string
		checkFn     func(t *testing.T, result string)
	}{
		{
			name:        "simple description",
			beadID:      "bead-123",
			description: "Fix authentication",
			checkFn: func(t *testing.T, result string) {
				if result != "agent/bead-123/fix-authentication" {
					t.Errorf("expected 'agent/bead-123/fix-authentication', got %q", result)
				}
			},
		},
		{
			name:        "long description gets truncated",
			beadID:      "bead-456",
			description: "This is an extremely long description that should be truncated to forty characters maximum for the slug portion",
			checkFn: func(t *testing.T, result string) {
				// Slug portion should be <= 40 chars
				expectedPrefix := "agent/bead-456/"
				if len(result) > len(expectedPrefix)+40 {
					t.Errorf("branch name too long: %q (%d chars after prefix)", result, len(result)-len(expectedPrefix))
				}
			},
		},
		{
			name:        "special characters in description",
			beadID:      "bead-789",
			description: "Fix @#$ special! chars",
			checkFn: func(t *testing.T, result string) {
				if !containsStr(result, "agent/bead-789/") {
					t.Errorf("expected prefix 'agent/bead-789/', got %q", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.generateBranchName(tt.beadID, tt.description)
			tt.checkFn(t, result)
		})
	}
}

func TestSetBranchPrefix(t *testing.T) {
	svc := &GitService{
		branchPrefix: "agent/",
	}

	// Setting non-empty prefix should work
	svc.SetBranchPrefix("custom/")
	if svc.branchPrefix != "custom/" {
		t.Errorf("expected prefix 'custom/', got %q", svc.branchPrefix)
	}

	// Setting empty prefix should be ignored
	svc.SetBranchPrefix("")
	if svc.branchPrefix != "custom/" {
		t.Errorf("expected prefix to remain 'custom/', got %q", svc.branchPrefix)
	}
}

func TestCreateBranchRequestStruct(t *testing.T) {
	req := CreateBranchRequest{
		BeadID:      "bead-1",
		Description: "My feature",
		BaseBranch:  "main",
	}

	if req.BeadID != "bead-1" {
		t.Errorf("BeadID: expected bead-1, got %s", req.BeadID)
	}
	if req.Description != "My feature" {
		t.Errorf("Description: expected 'My feature', got %s", req.Description)
	}
	if req.BaseBranch != "main" {
		t.Errorf("BaseBranch: expected main, got %s", req.BaseBranch)
	}
}

func TestCreateBranchResultStruct(t *testing.T) {
	result := CreateBranchResult{
		BranchName: "agent/bead-1/my-feature",
		Created:    true,
		Existed:    false,
	}

	if result.BranchName != "agent/bead-1/my-feature" {
		t.Errorf("BranchName: expected 'agent/bead-1/my-feature', got %s", result.BranchName)
	}
	if !result.Created {
		t.Error("expected Created=true")
	}
	if result.Existed {
		t.Error("expected Existed=false")
	}
}

func TestCommitRequestStruct(t *testing.T) {
	req := CommitRequest{
		BeadID:   "bead-1",
		AgentID:  "agent-1",
		Message:  "Fix bug",
		Files:    []string{"file1.go", "file2.go"},
		AllowAll: false,
	}

	if req.BeadID != "bead-1" {
		t.Errorf("BeadID: expected bead-1, got %s", req.BeadID)
	}
	if len(req.Files) != 2 {
		t.Errorf("Files: expected 2, got %d", len(req.Files))
	}
}

func TestCommitResultStruct(t *testing.T) {
	result := CommitResult{
		CommitSHA:    "abc123",
		FilesChanged: 3,
		Insertions:   42,
		Deletions:    10,
		Files:        []string{"a.go", "b.go", "c.go"},
	}

	if result.CommitSHA != "abc123" {
		t.Errorf("CommitSHA: expected abc123, got %s", result.CommitSHA)
	}
	if result.FilesChanged != 3 {
		t.Errorf("FilesChanged: expected 3, got %d", result.FilesChanged)
	}
	if result.Insertions != 42 {
		t.Errorf("Insertions: expected 42, got %d", result.Insertions)
	}
	if result.Deletions != 10 {
		t.Errorf("Deletions: expected 10, got %d", result.Deletions)
	}
}

func TestPushRequestStruct(t *testing.T) {
	req := PushRequest{
		BeadID:      "bead-1",
		Branch:      "agent/bead-1/fix",
		SetUpstream: true,
		Force:       false,
	}

	if req.BeadID != "bead-1" {
		t.Errorf("BeadID: expected bead-1, got %s", req.BeadID)
	}
	if !req.SetUpstream {
		t.Error("expected SetUpstream=true")
	}
	if req.Force {
		t.Error("expected Force=false")
	}
}

func TestPushResultStruct(t *testing.T) {
	result := PushResult{
		Branch:  "agent/bead-1/fix",
		Remote:  "origin",
		Success: true,
	}

	if result.Branch != "agent/bead-1/fix" {
		t.Errorf("Branch: expected agent/bead-1/fix, got %s", result.Branch)
	}
	if result.Remote != "origin" {
		t.Errorf("Remote: expected origin, got %s", result.Remote)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
}

func TestLogEntryStruct(t *testing.T) {
	entry := LogEntry{
		SHA:     "abc123def456",
		Author:  "Test Author",
		Date:    "2025-01-15T10:30:00-08:00",
		Subject: "Fix critical bug",
	}

	if entry.SHA != "abc123def456" {
		t.Errorf("SHA: expected abc123def456, got %s", entry.SHA)
	}
	if entry.Author != "Test Author" {
		t.Errorf("Author: expected Test Author, got %s", entry.Author)
	}
}

func TestBranchInfoStruct(t *testing.T) {
	info := BranchInfo{
		Name:       "agent/bead-1/fix",
		IsCurrent:  true,
		IsRemote:   false,
		LastCommit: "abc123",
	}

	if info.Name != "agent/bead-1/fix" {
		t.Errorf("Name: expected agent/bead-1/fix, got %s", info.Name)
	}
	if !info.IsCurrent {
		t.Error("expected IsCurrent=true")
	}
	if info.IsRemote {
		t.Error("expected IsRemote=false")
	}
}

func TestMergeRequestStruct(t *testing.T) {
	req := MergeRequest{
		SourceBranch: "agent/bead-1/feature",
		Message:      "Merge agent work",
		NoFF:         true,
		BeadID:       "bead-1",
	}

	if req.SourceBranch != "agent/bead-1/feature" {
		t.Errorf("SourceBranch: expected agent/bead-1/feature, got %s", req.SourceBranch)
	}
	if !req.NoFF {
		t.Error("expected NoFF=true")
	}
}

func TestMergeResultStruct(t *testing.T) {
	result := MergeResult{
		MergedBranch: "agent/bead-1/feature",
		CommitSHA:    "abc123",
		Success:      true,
	}

	if result.MergedBranch != "agent/bead-1/feature" {
		t.Errorf("MergedBranch: expected agent/bead-1/feature, got %s", result.MergedBranch)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
}

func TestRevertRequestStruct(t *testing.T) {
	req := RevertRequest{
		CommitSHAs: []string{"abc", "def"},
		BeadID:     "bead-1",
		Reason:     "broken build",
	}

	if len(req.CommitSHAs) != 2 {
		t.Errorf("CommitSHAs: expected 2, got %d", len(req.CommitSHAs))
	}
	if req.Reason != "broken build" {
		t.Errorf("Reason: expected 'broken build', got %s", req.Reason)
	}
}

func TestRevertResultStruct(t *testing.T) {
	result := RevertResult{
		RevertedSHAs: []string{"abc", "def"},
		NewCommitSHA: "ghi789",
		Success:      true,
	}

	if len(result.RevertedSHAs) != 2 {
		t.Errorf("RevertedSHAs: expected 2, got %d", len(result.RevertedSHAs))
	}
	if result.NewCommitSHA != "ghi789" {
		t.Errorf("NewCommitSHA: expected ghi789, got %s", result.NewCommitSHA)
	}
}

func TestDeleteBranchRequestStruct(t *testing.T) {
	req := DeleteBranchRequest{
		Branch:       "agent/bead-1/fix",
		DeleteRemote: true,
	}

	if req.Branch != "agent/bead-1/fix" {
		t.Errorf("Branch: expected agent/bead-1/fix, got %s", req.Branch)
	}
	if !req.DeleteRemote {
		t.Error("expected DeleteRemote=true")
	}
}

func TestDeleteBranchResultStruct(t *testing.T) {
	result := DeleteBranchResult{
		Branch:        "agent/bead-1/fix",
		DeletedLocal:  true,
		DeletedRemote: true,
	}

	if !result.DeletedLocal {
		t.Error("expected DeletedLocal=true")
	}
	if !result.DeletedRemote {
		t.Error("expected DeletedRemote=true")
	}
}

func TestCheckoutRequestStruct(t *testing.T) {
	req := CheckoutRequest{
		Branch: "agent/bead-1/feature",
	}

	if req.Branch != "agent/bead-1/feature" {
		t.Errorf("Branch: expected agent/bead-1/feature, got %s", req.Branch)
	}
}

func TestCheckoutResultStruct(t *testing.T) {
	result := CheckoutResult{
		Branch:         "agent/bead-1/feature",
		PreviousBranch: "main",
	}

	if result.Branch != "agent/bead-1/feature" {
		t.Errorf("Branch: expected agent/bead-1/feature, got %s", result.Branch)
	}
	if result.PreviousBranch != "main" {
		t.Errorf("PreviousBranch: expected main, got %s", result.PreviousBranch)
	}
}

func TestLogRequestStruct(t *testing.T) {
	req := LogRequest{
		Branch:   "main",
		MaxCount: 50,
	}

	if req.Branch != "main" {
		t.Errorf("Branch: expected main, got %s", req.Branch)
	}
	if req.MaxCount != 50 {
		t.Errorf("MaxCount: expected 50, got %d", req.MaxCount)
	}
}

func TestDiffBranchesRequestStruct(t *testing.T) {
	req := DiffBranchesRequest{
		Branch1: "main",
		Branch2: "agent/bead-1/feature",
	}

	if req.Branch1 != "main" {
		t.Errorf("Branch1: expected main, got %s", req.Branch1)
	}
	if req.Branch2 != "agent/bead-1/feature" {
		t.Errorf("Branch2: expected agent/bead-1/feature, got %s", req.Branch2)
	}
}

func TestCreatePRResultStruct(t *testing.T) {
	result := CreatePRResult{
		Number: 42,
		URL:    "https://github.com/owner/repo/pull/42",
		Branch: "agent/bead-1/fix",
		Base:   "main",
	}

	if result.Number != 42 {
		t.Errorf("Number: expected 42, got %d", result.Number)
	}
	if result.URL != "https://github.com/owner/repo/pull/42" {
		t.Errorf("URL: expected pull/42, got %s", result.URL)
	}
}

// Helper function used in tests
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	if len(lines) == 0 {
		lines = append(lines, s)
	}
	return lines
}
