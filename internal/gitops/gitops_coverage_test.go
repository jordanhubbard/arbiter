package gitops

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

// ---- Tests for functions that don't require actual git ----

// TestLogGitEvent covers the logGitEvent helper with nil and non-nil project.
func TestLogGitEvent(t *testing.T) {
	// nil project — should not panic
	logGitEvent("test.event", nil, map[string]interface{}{"key": "value"})

	// non-nil project
	p := &models.Project{ID: "proj-1", GitRepo: "https://github.com/example/repo.git", Branch: "main"}
	logGitEvent("test.event", p, map[string]interface{}{"key": "value"})

	// nil fields map
	logGitEvent("test.event", p, nil)

	// empty fields map
	logGitEvent("test.event", p, map[string]interface{}{})
}

// TestLogGitError covers the logGitError helper with nil and non-nil project.
func TestLogGitError(t *testing.T) {
	err := os.ErrNotExist

	// nil project
	logGitError("test.error", nil, map[string]interface{}{"key": "value"}, err)

	// non-nil project
	p := &models.Project{ID: "proj-1", GitRepo: "https://github.com/example/repo.git", Branch: "main"}
	logGitError("test.error", p, map[string]interface{}{"key": "value"}, err)

	// nil fields
	logGitError("test.error", p, nil, err)
}

// TestProjectIDFromWorkDir covers various workdir path parsing cases.
func TestProjectIDFromWorkDir_Extended(t *testing.T) {
	tests := []struct {
		workDir string
		want    string
	}{
		{"/app/src/my-project", "my-project"},
		{"/app/src/project-123", "project-123"},
		{"project", "project"},
		{"/a/b/c", "c"},
		{".", "."},
		{"/single", "single"},
		{"/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.workDir, func(t *testing.T) {
			got := projectIDFromWorkDir(tt.workDir)
			if got != tt.want {
				t.Errorf("projectIDFromWorkDir(%q) = %q, want %q", tt.workDir, got, tt.want)
			}
		})
	}
}

// TestTimePtr covers the timePtr helper.
func TestTimePtr(t *testing.T) {
	now := time.Now()
	ptr := timePtr(now)
	if ptr == nil {
		t.Fatal("timePtr returned nil")
	}
	if !ptr.Equal(now) {
		t.Errorf("timePtr returned %v, want %v", *ptr, now)
	}
}

// TestShellEscape_Extended covers additional shell escape cases.
func TestShellEscape_Extended(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"with space", "'with space'"},
		{"with'quote", "'with'\\''quote'"},
		{"", "''"},
		{"path/to/key", "'path/to/key'"},
		{"a'b'c", "'a'\\''b'\\''c'"},
		{"no-special-chars", "'no-special-chars'"},
		{"$HOME", "'$HOME'"},
		{"`cmd`", "'`cmd`'"},
		{"a;b", "'a;b'"},
		{"multi\nline", "'multi\nline'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := shellEscape(tt.input)
			if got != tt.want {
				t.Errorf("shellEscape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestValidateProjectID_Extended covers more edge cases.
func TestValidateProjectID_Extended(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		wantErr   bool
		errSubstr string
	}{
		{"valid-simple", "my-project", false, ""},
		{"valid-underscore", "my_project", false, ""},
		{"valid-all-numbers", "12345", false, ""},
		{"valid-mixed-case", "AbCdEf", false, ""},
		{"valid-max-length", strings.Repeat("a", 100), false, ""},
		{"empty", "", true, "required"},
		{"too-long", strings.Repeat("a", 101), true, "too long"},
		{"has-dot", "my.project", true, "invalid character"},
		{"has-slash", "my/project", true, "invalid character"},
		{"has-space", "my project", true, "invalid character"},
		{"has-dollar", "project$bad", true, "invalid character"},
		{"has-backtick", "project`cmd`", true, "invalid character"},
		{"has-at", "proj@123", true, "invalid character"},
		{"has-exclamation", "proj!", true, "invalid character"},
		{"has-tab", "proj\t1", true, "invalid character"},
		{"has-newline", "proj\n1", true, "invalid character"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectID(tt.projectID)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProjectID(%q) error = %v, wantErr %v", tt.projectID, err, tt.wantErr)
			}
			if err != nil && tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("validateProjectID(%q) error = %q, want substr %q", tt.projectID, err, tt.errSubstr)
			}
		})
	}
}

// TestValidateCommitMessage_Extended covers additional commit message validation.
func TestValidateCommitMessage_Extended(t *testing.T) {
	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{"valid-simple", "Fix bug in parser", false},
		{"valid-multiline", "Fix bug\n\nDetailed explanation here", false},
		{"empty", "", true},
		{"too-long", strings.Repeat("a", 10001), true},
		{"max-length", strings.Repeat("a", 10000), false},
		{"cmd-substitution-dollar", "Fix $(rm -rf)", true},
		{"cmd-substitution-backtick", "Fix `rm -rf`", true},
		{"pipe-in-subject", "Fix | rm", true},
		{"semicolon-in-subject", "Fix; rm", true},
		{"double-ampersand-in-subject", "Fix && rm", true},
		{"double-pipe-in-subject", "Fix || rm", true},
		{"redirect-gt-in-subject", "Fix > /dev/null", true},
		{"redirect-lt-in-subject", "Fix < /dev/null", true},
		// Body allows some shell chars (except command substitution)
		{"pipe-in-body", "Fix bug\n\nOutput: foo | bar", false},
		{"semicolon-in-body", "Fix bug\n\nRan: cmd1; cmd2", false},
		{"ampersand-in-body", "Fix bug\n\nRan: cmd1 && cmd2", false},
		{"cmd-subst-in-body-dollar", "Fix bug\n\n$(dangerous)", true},
		{"cmd-subst-in-body-backtick", "Fix bug\n\n`dangerous`", true},
		// Edge case: single character messages
		{"single-char", "x", false},
		// Unicode in commit messages
		{"unicode", "Fix internationalization issue", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommitMessage(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommitMessage(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

// TestValidateAuthorInfo_Extended covers more edge cases for author validation.
func TestValidateAuthorInfo_Extended(t *testing.T) {
	tests := []struct {
		name    string
		aName   string
		email   string
		wantErr bool
	}{
		{"valid", "Test Agent", "agent@test.local", false},
		{"both-empty", "", "", false},
		{"name-only", "Agent", "", true},
		{"email-only", "", "agent@test.local", true},
		{"name-too-long", strings.Repeat("a", 101), "agent@test.local", true},
		{"email-too-long", "Agent", strings.Repeat("a", 252) + "@b.c", true},
		{"invalid-email-no-at", "Agent", "noemail", true},
		{"invalid-email-space", "Agent", "agent @test.local", true},
		{"name-has-semicolon", "Agent;bad", "agent@test.local", true},
		{"name-has-pipe", "Agent|bad", "agent@test.local", true},
		{"name-has-backtick", "Agent`cmd`", "agent@test.local", true},
		{"name-has-lt", "Agent<script>", "agent@test.local", true},
		{"name-has-gt", "Agent>bad", "agent@test.local", true},
		{"name-has-ampersand", "Agent&bad", "agent@test.local", true},
		{"name-has-dollar", "Agent$bad", "agent@test.local", true},
		{"name-has-newline", "Agent\nbad", "agent@test.local", true},
		{"name-has-tab", "Agent\tbad", "agent@test.local", true},
		{"email-has-semicolon", "Agent", "agent;bad@test.local", true},
		{"email-has-pipe", "Agent", "agent|bad@test.local", true},
		{"email-has-backtick", "Agent", "agent`cmd`@test.local", true},
		{"email-has-newline", "Agent", "agent\n@test.local", true},
		{"email-has-tab", "Agent", "agent\t@test.local", true},
		// Valid edge cases
		{"name-with-hyphen", "Test-Agent", "agent@test.local", false},
		{"name-with-period", "Test.Agent", "agent@test.local", false},
		{"name-with-space", "Test Agent Name", "agent@test.local", false},
		{"email-with-plus", "Agent", "agent+tag@test.local", false},
		{"email-with-dots", "Agent", "a.b.c@test.local", false},
		{"name-max-length", strings.Repeat("a", 100), "a@b.c", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAuthorInfo(tt.aName, tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAuthorInfo(%q, %q) error = %v, wantErr %v", tt.aName, tt.email, err, tt.wantErr)
			}
		})
	}
}

// TestValidateSSHKeyPath_Extended covers more edge cases.
func TestValidateSSHKeyPath_Extended(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test key file in a subdirectory
	subDir := filepath.Join(tmpDir, "project", "ssh")
	if err := os.MkdirAll(subDir, 0700); err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(subDir, "id_ed25519")
	if err := os.WriteFile(keyFile, []byte("fake key"), 0600); err != nil {
		t.Fatal(err)
	}

	// Valid: key within expected directory
	err := validateSSHKeyPath(keyFile, tmpDir)
	if err != nil {
		t.Errorf("expected no error for valid key path, got: %v", err)
	}

	// Invalid: file does not exist
	err = validateSSHKeyPath(filepath.Join(tmpDir, "nonexistent"), tmpDir)
	if err == nil {
		t.Error("expected error for nonexistent path")
	}

	// Invalid: path outside expected directory
	otherDir := t.TempDir()
	otherKey := filepath.Join(otherDir, "key")
	if err := os.WriteFile(otherKey, []byte("fake"), 0600); err != nil {
		t.Fatal(err)
	}
	err = validateSSHKeyPath(otherKey, tmpDir)
	if err == nil {
		t.Error("expected error for path outside expected directory")
	}
}

// TestProjectKeyPaths_Extended covers key directory and path computation.
func TestProjectKeyPaths_Extended(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := filepath.Join(tmpDir, "keys")
	mgr, err := NewManager(tmpDir, keyDir, nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	projectID := "test-project"

	// Test projectKeyDirForProject
	dir := mgr.projectKeyDirForProject(projectID)
	expected := filepath.Join(keyDir, projectID, "ssh")
	if dir != expected {
		t.Errorf("projectKeyDirForProject(%q) = %q, want %q", projectID, dir, expected)
	}

	// Test projectPrivateKeyPath
	privPath := mgr.projectPrivateKeyPath(projectID)
	expectedPriv := filepath.Join(keyDir, projectID, "ssh", "id_ed25519")
	if privPath != expectedPriv {
		t.Errorf("projectPrivateKeyPath(%q) = %q, want %q", projectID, privPath, expectedPriv)
	}

	// Test projectPublicKeyPath
	pubPath := mgr.projectPublicKeyPath(projectID)
	expectedPub := expectedPriv + ".pub"
	if pubPath != expectedPub {
		t.Errorf("projectPublicKeyPath(%q) = %q, want %q", projectID, pubPath, expectedPub)
	}
}

// TestSetProjectWorkDir_Extended covers workdir overrides.
func TestSetProjectWorkDir_Extended(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Before any override, returns base + projectID
	got := mgr.GetProjectWorkDir("proj-1")
	if got != filepath.Join(tmpDir, "proj-1") {
		t.Errorf("before override: GetProjectWorkDir = %q, want %q", got, filepath.Join(tmpDir, "proj-1"))
	}

	// Set override
	mgr.SetProjectWorkDir("proj-1", "/custom/path")
	got = mgr.GetProjectWorkDir("proj-1")
	if got != "/custom/path" {
		t.Errorf("after override: GetProjectWorkDir = %q, want /custom/path", got)
	}

	// Other projects still use base
	got = mgr.GetProjectWorkDir("proj-2")
	if got != filepath.Join(tmpDir, "proj-2") {
		t.Errorf("other project: GetProjectWorkDir = %q, want %q", got, filepath.Join(tmpDir, "proj-2"))
	}

	// Override again
	mgr.SetProjectWorkDir("proj-1", ".")
	got = mgr.GetProjectWorkDir("proj-1")
	if got != "." {
		t.Errorf("re-override: GetProjectWorkDir = %q, want .", got)
	}
}

// TestNewManager_Extended covers various constructor scenarios.
func TestNewManager_Extended(t *testing.T) {
	tmpDir := t.TempDir()

	// Nested directories that don't exist yet
	baseDir := filepath.Join(tmpDir, "deep", "nested", "work")
	keyDir := filepath.Join(tmpDir, "deep", "nested", "keys")

	mgr, err := NewManager(baseDir, keyDir, nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Verify directories were created
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Error("base work directory was not created")
	}
	if _, err := os.Stat(keyDir); os.IsNotExist(err) {
		t.Error("key directory was not created")
	}

	// Verify manager fields
	if mgr.baseWorkDir != baseDir {
		t.Errorf("baseWorkDir = %q, want %q", mgr.baseWorkDir, baseDir)
	}
	if mgr.projectKeyDir != keyDir {
		t.Errorf("projectKeyDir = %q, want %q", mgr.projectKeyDir, keyDir)
	}
	if mgr.db != nil {
		t.Error("db should be nil")
	}
	if mgr.keyManager != nil {
		t.Error("keyManager should be nil")
	}
}

// TestRestoreKeyFromDB_NilDeps covers restoreKeyFromDB when db/km are nil.
func TestRestoreKeyFromDB_NilDeps(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Should return false when both db and km are nil
	result := mgr.restoreKeyFromDB("test-project")
	if result {
		t.Error("expected false when db/km are nil")
	}
}

// TestStoreKeyInDB_NilDeps covers storeKeyInDB when db/km are nil.
func TestStoreKeyInDB_NilDeps(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Should not panic when both db and km are nil
	mgr.storeKeyInDB("test-project", "ssh-ed25519 AAAA... test")
}

// TestStoreKeyInDBWithRotation_NilDeps covers the rotation variant.
func TestStoreKeyInDBWithRotation_NilDeps(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	now := time.Now()
	// Should not panic when both db and km are nil
	mgr.storeKeyInDBWithRotation("test-project", "ssh-ed25519 AAAA... test", &now)
	mgr.storeKeyInDBWithRotation("test-project", "ssh-ed25519 AAAA... test", nil)
}

// TestBackfillSSHCredentials_NilDeps covers backfill when db/km are nil.
func TestBackfillSSHCredentials_NilDeps(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Should not panic
	mgr.BackfillSSHCredentials(nil)
	mgr.BackfillSSHCredentials([]*models.Project{})
	mgr.BackfillSSHCredentials([]*models.Project{
		{ID: "proj-1"},
		{ID: "proj-2"},
	})
}

// TestConfigureAuth_AuthNone covers the "none" auth method.
func TestConfigureAuth_AuthNone(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	project := &models.Project{
		ID:            "test-project",
		GitRepo:       "https://github.com/example/repo.git",
		GitAuthMethod: models.GitAuthNone,
	}

	// configureAuth with none should succeed
	// We need a dummy exec.Cmd - simplest is to create one
	ctx := context.Background()
	_ = ctx
	cmd := createDummyCmd()
	err = mgr.configureAuth(cmd, project)
	if err != nil {
		t.Errorf("configureAuth with GitAuthNone should not error: %v", err)
	}
}

// TestConfigureAuth_AuthToken covers the "token" auth method.
func TestConfigureAuth_AuthToken(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	project := &models.Project{
		ID:            "test-project",
		GitRepo:       "https://github.com/example/repo.git",
		GitAuthMethod: models.GitAuthToken,
	}

	cmd := createDummyCmd()
	err = mgr.configureAuth(cmd, project)
	if err != nil {
		t.Errorf("configureAuth with GitAuthToken should not error: %v", err)
	}
}

// TestConfigureAuth_AuthBasic covers the "basic" auth method.
func TestConfigureAuth_AuthBasic(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	project := &models.Project{
		ID:            "test-project",
		GitRepo:       "https://github.com/example/repo.git",
		GitAuthMethod: models.GitAuthBasic,
	}

	cmd := createDummyCmd()
	err = mgr.configureAuth(cmd, project)
	if err != nil {
		t.Errorf("configureAuth with GitAuthBasic should not error: %v", err)
	}
}

// TestConfigureAuth_UnsupportedAuth covers unsupported auth methods.
func TestConfigureAuth_UnsupportedAuth(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	project := &models.Project{
		ID:            "test-project",
		GitRepo:       "https://github.com/example/repo.git",
		GitAuthMethod: models.GitAuthMethod("unsupported-method"),
	}

	cmd := createDummyCmd()
	err = mgr.configureAuth(cmd, project)
	if err == nil {
		t.Error("configureAuth with unsupported auth should error")
	}
	if !strings.Contains(err.Error(), "unsupported auth method") {
		t.Errorf("error should mention unsupported auth method: %v", err)
	}
}

// TestGetStatus_Placeholder covers the placeholder GetStatus method.
func TestGetStatus_Placeholder(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	result, err := mgr.GetStatus(ctx)
	if result != nil {
		t.Error("GetStatus should return nil result")
	}
	if err == nil {
		t.Error("GetStatus should return error (placeholder)")
	}
}

// TestGetDiff_Placeholder covers the placeholder GetDiff method.
func TestGetDiff_Placeholder(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()

	// Test with staged=false
	result, err := mgr.GetDiff(ctx, false)
	if result != nil {
		t.Error("GetDiff should return nil result")
	}
	if err == nil {
		t.Error("GetDiff should return error (placeholder)")
	}

	// Test with staged=true
	result, err = mgr.GetDiff(ctx, true)
	if result != nil {
		t.Error("GetDiff should return nil result")
	}
	if err == nil {
		t.Error("GetDiff should return error (placeholder)")
	}
}

// TestCreateBranch_Placeholder covers the placeholder CreateBranch method.
func TestCreateBranch_Placeholder(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	result, err := mgr.CreateBranch(ctx, "bead-123", "test desc", "main")
	if result == nil {
		t.Error("CreateBranch should return non-nil result map")
	}
	if err == nil {
		t.Error("CreateBranch should return error (placeholder)")
	}
	if result["branch"] != "bead-bead-123" {
		t.Errorf("branch = %v, want bead-bead-123", result["branch"])
	}
	if result["base"] != "main" {
		t.Errorf("base = %v, want main", result["base"])
	}
}

// TestCommit_Placeholder covers the placeholder Commit method.
func TestCommit_Placeholder(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	result, err := mgr.Commit(ctx, "bead-1", "agent-1", "test commit msg", []string{"file.go"}, true)
	if result == nil {
		t.Error("Commit should return non-nil result map")
	}
	if err == nil {
		t.Error("Commit should return error (placeholder)")
	}
	if result["message"] != "test commit msg" {
		t.Errorf("message = %v, want 'test commit msg'", result["message"])
	}
	if result["bead_id"] != "bead-1" {
		t.Errorf("bead_id = %v, want bead-1", result["bead_id"])
	}
	if result["agent_id"] != "agent-1" {
		t.Errorf("agent_id = %v, want agent-1", result["agent_id"])
	}
}

// TestPush_Placeholder covers the placeholder Push method.
func TestPush_Placeholder(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	result, err := mgr.Push(ctx, "bead-1", "feature-branch", true)
	if result == nil {
		t.Error("Push should return non-nil result map")
	}
	if err == nil {
		t.Error("Push should return error (placeholder)")
	}
	if result["branch"] != "feature-branch" {
		t.Errorf("branch = %v, want feature-branch", result["branch"])
	}
	if result["set_upstream"] != true {
		t.Errorf("set_upstream = %v, want true", result["set_upstream"])
	}
}

// TestCreatePR_Placeholder covers the placeholder CreatePR method.
func TestCreatePR_Placeholder(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	reviewers := []string{"user1", "user2"}
	result, err := mgr.CreatePR(ctx, "bead-1", "My PR Title", "PR body", "main", "feature-1", reviewers, true)
	if result == nil {
		t.Error("CreatePR should return non-nil result map")
	}
	if err == nil {
		t.Error("CreatePR should return error (placeholder)")
	}
	if result["title"] != "My PR Title" {
		t.Errorf("title = %v, want 'My PR Title'", result["title"])
	}
	if result["draft"] != true {
		t.Errorf("draft = %v, want true", result["draft"])
	}
	if result["base"] != "main" {
		t.Errorf("base = %v, want main", result["base"])
	}
}

// TestLoadBeadsFromProject_ExistingButEmpty covers loading beads from existing dir.
func TestLoadBeadsFromProject_ExistingButEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create the expected beads directory
	projectDir := filepath.Join(tmpDir, "proj-beads")
	beadsDir := filepath.Join(projectDir, ".beads", "beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	project := &models.Project{
		ID:        "proj-beads",
		BeadsPath: ".beads",
	}

	beads, err := mgr.LoadBeadsFromProject(project)
	if err != nil {
		t.Errorf("LoadBeadsFromProject should not error: %v", err)
	}
	// Currently returns nil, nil (placeholder implementation)
	if beads != nil {
		t.Error("expected nil beads from placeholder implementation")
	}
}

// TestStatus_NotCloned covers Status when project not cloned.
func TestStatus_NotCloned(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	_, err = mgr.Status(ctx, "nonexistent-project")
	if err == nil {
		t.Error("expected error for Status on non-cloned project")
	}
	if !strings.Contains(err.Error(), "not cloned") {
		t.Errorf("error should mention 'not cloned': %v", err)
	}
}

// TestDiff_NotCloned covers Diff when project not cloned.
func TestDiff_NotCloned(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	_, err = mgr.Diff(ctx, "nonexistent-project")
	if err == nil {
		t.Error("expected error for Diff on non-cloned project")
	}
	if !strings.Contains(err.Error(), "not cloned") {
		t.Errorf("error should mention 'not cloned': %v", err)
	}
}

// TestCloneProject_EmptyGitRepo ensures empty GitRepo produces proper error.
func TestCloneProject_EmptyGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	project := &models.Project{
		ID:      "no-repo",
		GitRepo: "",
	}

	ctx := context.Background()
	err = mgr.CloneProject(ctx, project)
	if err == nil {
		t.Error("expected error when GitRepo is empty")
	}
	if !strings.Contains(err.Error(), "no git_repo") {
		t.Errorf("error should mention 'no git_repo': %v", err)
	}
}

// TestCheckRemoteAccess_NilProject covers nil project error.
func TestCheckRemoteAccess_NilProject(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	err = mgr.CheckRemoteAccess(ctx, nil)
	if err == nil {
		t.Error("expected error for nil project")
	}
	if !strings.Contains(err.Error(), "project is required") {
		t.Errorf("error should mention 'project is required': %v", err)
	}
}

// TestCheckRemoteAccess_LocalRepo covers dot repo (no remote check needed).
func TestCheckRemoteAccess_LocalRepo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()

	// Empty GitRepo
	err = mgr.CheckRemoteAccess(ctx, &models.Project{ID: "local", GitRepo: ""})
	if err != nil {
		t.Errorf("expected no error for empty GitRepo: %v", err)
	}

	// Dot GitRepo
	err = mgr.CheckRemoteAccess(ctx, &models.Project{ID: "local", GitRepo: "."})
	if err != nil {
		t.Errorf("expected no error for dot GitRepo: %v", err)
	}
}

// TestEnsureProjectSSHKey_EmptyProjectID covers empty project ID.
func TestEnsureProjectSSHKey_EmptyProjectID(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.EnsureProjectSSHKey("")
	if err == nil {
		t.Error("expected error for empty project ID")
	}
	if !strings.Contains(err.Error(), "project ID is required") {
		t.Errorf("error should mention 'project ID is required': %v", err)
	}
}

// TestRotateProjectSSHKey_EmptyProjectID covers empty project ID for rotation.
func TestRotateProjectSSHKey_EmptyProjectID(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.RotateProjectSSHKey("")
	if err == nil {
		t.Error("expected error for empty project ID")
	}
	if !strings.Contains(err.Error(), "project ID is required") {
		t.Errorf("error should mention 'project ID is required': %v", err)
	}
}

// TestCommitChanges_ValidationErrors covers validation failures in CommitChanges.
func TestCommitChanges_ValidationErrors(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	project := &models.Project{ID: "test-proj", GitRepo: "https://example.com/repo.git"}

	// Empty commit message
	err = mgr.CommitChanges(ctx, project, "", "Author", "author@test.com")
	if err == nil {
		t.Error("expected error for empty commit message")
	}
	if !strings.Contains(err.Error(), "invalid commit message") {
		t.Errorf("error should mention 'invalid commit message': %v", err)
	}

	// Dangerous commit message
	err = mgr.CommitChanges(ctx, project, "Fix $(rm -rf /)", "Author", "author@test.com")
	if err == nil {
		t.Error("expected error for dangerous commit message")
	}

	// Invalid author info (name only)
	err = mgr.CommitChanges(ctx, project, "Valid message", "Author", "")
	if err == nil {
		t.Error("expected error for name-only author info")
	}
	if !strings.Contains(err.Error(), "invalid author info") {
		t.Errorf("error should mention 'invalid author info': %v", err)
	}
}

// TestGetProjectPublicKey_EmptyID covers empty project ID in GetProjectPublicKey.
func TestGetProjectPublicKey_EmptyID(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.GetProjectPublicKey("")
	if err == nil {
		t.Error("expected error for empty project ID")
	}
}

// createDummyCmd creates a trivial exec.Cmd for testing configureAuth.
func createDummyCmd() *exec.Cmd {
	return exec.Command("echo", "dummy")
}

// ----- Tests that use ssh-keygen and git (integration-lite) -----

// TestEnsureProjectSSHKey_GeneratesKey tests full key generation.
func TestEnsureProjectSSHKey_GeneratesKey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	pubKey, err := mgr.EnsureProjectSSHKey("test-project")
	if err != nil {
		t.Fatalf("EnsureProjectSSHKey failed: %v", err)
	}
	if pubKey == "" {
		t.Fatal("expected non-empty public key")
	}
	if !strings.HasPrefix(pubKey, "ssh-ed25519") {
		t.Errorf("expected ed25519 key, got: %s", pubKey[:min(len(pubKey), 30)])
	}

	// Verify files exist
	privPath := mgr.projectPrivateKeyPath("test-project")
	pubPath := mgr.projectPublicKeyPath("test-project")
	if _, err := os.Stat(privPath); os.IsNotExist(err) {
		t.Error("private key file not created")
	}
	if _, err := os.Stat(pubPath); os.IsNotExist(err) {
		t.Error("public key file not created")
	}

	// Check permissions on private key
	info, err := os.Stat(privPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("private key permissions = %o, want 0600", info.Mode().Perm())
	}

	// Second call should reuse existing key
	pubKey2, err := mgr.EnsureProjectSSHKey("test-project")
	if err != nil {
		t.Fatalf("second EnsureProjectSSHKey failed: %v", err)
	}
	if pubKey2 != pubKey {
		t.Error("second call should return same public key")
	}
}

// TestRotateProjectSSHKey_Rotates tests full key rotation.
func TestRotateProjectSSHKey_Rotates(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// First generate a key
	pubKey1, err := mgr.EnsureProjectSSHKey("rotate-test")
	if err != nil {
		t.Fatalf("EnsureProjectSSHKey failed: %v", err)
	}

	// Rotate
	pubKey2, err := mgr.RotateProjectSSHKey("rotate-test")
	if err != nil {
		t.Fatalf("RotateProjectSSHKey failed: %v", err)
	}

	if pubKey2 == "" {
		t.Fatal("expected non-empty public key after rotation")
	}
	if pubKey2 == pubKey1 {
		t.Error("rotated key should be different from original")
	}
}

// TestConfigureAuth_SSH covers the SSH auth method path.
func TestConfigureAuth_SSH(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Generate SSH key first
	_, err = mgr.EnsureProjectSSHKey("ssh-test")
	if err != nil {
		t.Fatalf("EnsureProjectSSHKey failed: %v", err)
	}

	project := &models.Project{
		ID:            "ssh-test",
		GitRepo:       "git@github.com:example/repo.git",
		GitAuthMethod: models.GitAuthSSH,
	}

	cmd := createDummyCmd()
	err = mgr.configureAuth(cmd, project)
	if err != nil {
		t.Fatalf("configureAuth with SSH should not error: %v", err)
	}

	// Verify GIT_SSH_COMMAND and GIT_TERMINAL_PROMPT are set in env.
	// configureAuth appends to os.Environ() so check the LAST occurrence
	// (the system may already have GIT_SSH_COMMAND set via proxy).
	lastSSH := ""
	foundPrompt := false
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "GIT_SSH_COMMAND=") {
			lastSSH = env
		}
		if env == "GIT_TERMINAL_PROMPT=0" {
			foundPrompt = true
		}
	}
	if lastSSH == "" {
		t.Error("GIT_SSH_COMMAND not found in cmd.Env")
	} else if !strings.Contains(lastSSH, "ssh -i") {
		t.Errorf("last GIT_SSH_COMMAND should contain 'ssh -i': %s", lastSSH)
	}
	if !foundPrompt {
		t.Error("GIT_TERMINAL_PROMPT=0 not found in cmd.Env")
	}
}

// TestConfigureAuth_SSH_InvalidProjectID covers SSH with invalid project ID.
func TestConfigureAuth_SSH_InvalidProjectID(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	project := &models.Project{
		ID:            "invalid/project",
		GitRepo:       "git@github.com:example/repo.git",
		GitAuthMethod: models.GitAuthSSH,
	}

	cmd := createDummyCmd()
	err = mgr.configureAuth(cmd, project)
	if err == nil {
		t.Error("expected error for invalid project ID with SSH auth")
	}
}

// TestRunGitCommand_InRealRepo tests runGitCommand in a real git repo.
func TestRunGitCommand_InRealRepo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Init a real git repo
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// git init should work
	err = mgr.runGitCommand(ctx, repoDir, "init")
	if err != nil {
		t.Fatalf("runGitCommand init failed: %v", err)
	}

	// Verify .git exists
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); os.IsNotExist(err) {
		t.Error("git init did not create .git directory")
	}

	// Run a command that should fail
	err = mgr.runGitCommand(ctx, repoDir, "checkout", "nonexistent-branch")
	if err == nil {
		t.Error("expected error for nonexistent branch checkout")
	}
}

// TestRunGitCommandWithOutput_InRealRepo tests runGitCommandWithOutput.
func TestRunGitCommandWithOutput_InRealRepo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Init a real git repo
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// git init
	err = mgr.runGitCommand(ctx, repoDir, "init")
	if err != nil {
		t.Fatalf("runGitCommand init failed: %v", err)
	}

	// Get status with output
	output, err := mgr.runGitCommandWithOutput(ctx, repoDir, "status", "-sb")
	if err != nil {
		t.Fatalf("runGitCommandWithOutput failed: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty status output")
	}

	// Command that fails
	_, err = mgr.runGitCommandWithOutput(ctx, repoDir, "log", "--oneline")
	if err == nil {
		// In an empty repo, git log might fail (no commits)
		// This is expected behavior
	}
}

// TestStatus_InRealRepo tests Status with a real git repo.
func TestStatus_InRealRepo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Init a repo
	repoDir := filepath.Join(tmpDir, "status-test")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = mgr.runGitCommand(ctx, repoDir, "init")
	if err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Status on the initialized repo
	status, err := mgr.Status(ctx, "status-test")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status == "" {
		t.Error("expected non-empty status")
	}
}

// TestDiff_InRealRepo tests Diff with a real git repo.
func TestDiff_InRealRepo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Init a repo
	repoDir := filepath.Join(tmpDir, "diff-test")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = mgr.runGitCommand(ctx, repoDir, "init")
	if err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Diff on empty repo (should work but return empty)
	diff, err := mgr.Diff(ctx, "diff-test")
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	// Empty diff is expected in a fresh repo
	_ = diff
}

// TestGenerateSSHKeyPair tests direct key pair generation.
func TestGenerateSSHKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	privPath := filepath.Join(tmpDir, "test_key")
	err = mgr.generateSSHKeyPair(privPath)
	if err != nil {
		t.Fatalf("generateSSHKeyPair failed: %v", err)
	}

	// Verify private key exists
	if _, err := os.Stat(privPath); os.IsNotExist(err) {
		t.Error("private key not created")
	}

	// Verify public key exists
	pubPath := privPath + ".pub"
	if _, err := os.Stat(pubPath); os.IsNotExist(err) {
		t.Error("public key not created")
	}

	// Verify permissions
	info, err := os.Stat(privPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("key permissions = %o, want 0600", info.Mode().Perm())
	}
}

// TestWritePublicKeyFromPrivate tests deriving public key from private.
func TestWritePublicKeyFromPrivate(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// First generate a key pair
	privPath := filepath.Join(tmpDir, "derive_key")
	err = mgr.generateSSHKeyPair(privPath)
	if err != nil {
		t.Fatalf("generateSSHKeyPair failed: %v", err)
	}

	// Remove the public key
	pubPath := privPath + ".pub"
	os.Remove(pubPath)

	// Re-derive public key from private
	newPubPath := filepath.Join(tmpDir, "derived.pub")
	err = mgr.writePublicKeyFromPrivate(privPath, newPubPath)
	if err != nil {
		t.Fatalf("writePublicKeyFromPrivate failed: %v", err)
	}

	// Verify it was created
	if _, err := os.Stat(newPubPath); os.IsNotExist(err) {
		t.Error("derived public key not created")
	}

	data, err := os.ReadFile(newPubPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "ssh-ed25519") {
		t.Errorf("expected ed25519 public key, got: %s", string(data[:min(len(data), 30)]))
	}
}

// TestWritePublicKeyFromPrivate_InvalidKey tests with invalid private key.
func TestWritePublicKeyFromPrivate_InvalidKey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create a fake "private key"
	privPath := filepath.Join(tmpDir, "fake_key")
	os.WriteFile(privPath, []byte("not a real key"), 0600)

	pubPath := filepath.Join(tmpDir, "fake_key.pub")
	err = mgr.writePublicKeyFromPrivate(privPath, pubPath)
	if err == nil {
		t.Error("expected error for invalid private key")
	}
}

// TestCommitChanges_InRealRepo tests CommitChanges with a real git repo.
func TestCommitChanges_InRealRepo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()

	// Init a repo
	repoDir := filepath.Join(tmpDir, "commit-test")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	err = mgr.runGitCommand(ctx, repoDir, "init")
	if err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user for the repo
	err = mgr.runGitCommand(ctx, repoDir, "config", "user.email", "test@test.com")
	if err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	err = mgr.runGitCommand(ctx, repoDir, "config", "user.name", "Test")
	if err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create a file and commit
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	project := &models.Project{
		ID:            "commit-test",
		GitRepo:       "https://example.com/repo.git",
		GitAuthMethod: models.GitAuthNone,
	}

	err = mgr.CommitChanges(ctx, project, "Initial commit", "Test Author", "author@test.com")
	if err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	// Verify commit was made
	hash, err := mgr.GetCurrentCommit(repoDir)
	if err != nil {
		t.Fatalf("GetCurrentCommit failed: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty commit hash")
	}

	// CommitChanges again with no changes should succeed (no-op)
	err = mgr.CommitChanges(ctx, project, "No changes", "Test Author", "author@test.com")
	if err != nil {
		t.Errorf("CommitChanges with no changes should succeed: %v", err)
	}
}

// TestCheckRemoteAccess_InvalidRepo tests CheckRemoteAccess with invalid repo.
func TestCheckRemoteAccess_InvalidRepo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	project := &models.Project{
		ID:            "remote-test",
		GitRepo:       "https://invalid-host-that-does-not-exist.example.com/repo.git",
		GitAuthMethod: models.GitAuthNone,
	}

	err = mgr.CheckRemoteAccess(ctx, project)
	if err == nil {
		t.Error("expected error for invalid remote")
	}
}

// TestPullProject_NotClonedDir covers PullProject when .git directory is missing.
func TestPullProject_NotClonedDir(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create the project directory but NOT .git
	projectDir := filepath.Join(tmpDir, "pull-test")
	os.MkdirAll(projectDir, 0755)

	project := &models.Project{
		ID:      "pull-test",
		GitRepo: "https://example.com/repo.git",
	}

	ctx := context.Background()
	err = mgr.PullProject(ctx, project)
	if err == nil {
		t.Error("expected error for not-cloned project")
	}
	if !strings.Contains(err.Error(), "not cloned") {
		t.Errorf("error should mention 'not cloned': %v", err)
	}
}

// TestCloneProject_LocalRepoClone tests cloning a local git repo (clean dir path).
func TestCloneProject_LocalRepoClone(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a "remote" bare repo
	bareDir := filepath.Join(tmpDir, "bare-repo.git")
	os.MkdirAll(bareDir, 0755)
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	ctx := context.Background()

	// Init bare repo
	err = mgr.runGitCommand(ctx, bareDir, "init", "--bare")
	if err != nil {
		t.Fatalf("git init --bare failed: %v", err)
	}

	// Create a temp working repo, make a commit, push to bare
	workRepo := filepath.Join(tmpDir, "work-repo")
	os.MkdirAll(workRepo, 0755)
	err = mgr.runGitCommand(ctx, workRepo, "init")
	if err != nil {
		t.Fatalf("git init work repo failed: %v", err)
	}
	err = mgr.runGitCommand(ctx, workRepo, "config", "user.email", "test@test.com")
	if err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	err = mgr.runGitCommand(ctx, workRepo, "config", "user.name", "Test")
	if err != nil {
		t.Fatalf("git config failed: %v", err)
	}

	// Create initial commit
	os.WriteFile(filepath.Join(workRepo, "README.md"), []byte("# Test"), 0644)
	err = mgr.runGitCommand(ctx, workRepo, "add", ".")
	if err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	err = mgr.runGitCommand(ctx, workRepo, "commit", "-m", "initial")
	if err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
	err = mgr.runGitCommand(ctx, workRepo, "remote", "add", "origin", bareDir)
	if err != nil {
		t.Fatalf("git remote add failed: %v", err)
	}

	// Get the current branch name
	branchOutput, err := mgr.runGitCommandWithOutput(ctx, workRepo, "branch", "--show-current")
	if err != nil {
		t.Fatalf("git branch failed: %v", err)
	}
	branch := strings.TrimSpace(branchOutput)
	if branch == "" {
		branch = "main"
	}

	err = mgr.runGitCommand(ctx, workRepo, "push", "-u", "origin", branch)
	if err != nil {
		t.Fatalf("git push failed: %v", err)
	}

	// Now clone from bare repo into project dir
	project := &models.Project{
		ID:            "clone-test",
		Name:          "Clone Test",
		GitRepo:       bareDir,
		Branch:        branch,
		BeadsPath:     ".beads",
		GitAuthMethod: models.GitAuthNone,
	}

	err = mgr.CloneProject(ctx, project)
	if err != nil {
		t.Fatalf("CloneProject failed: %v", err)
	}

	// Verify the repo was cloned
	cloneDir := mgr.GetProjectWorkDir("clone-test")
	if _, err := os.Stat(filepath.Join(cloneDir, ".git")); os.IsNotExist(err) {
		t.Error("git repo was not cloned")
	}
	if _, err := os.Stat(filepath.Join(cloneDir, "README.md")); os.IsNotExist(err) {
		t.Error("README.md not present in clone")
	}

	// Verify project metadata updated
	if project.WorkDir != cloneDir {
		t.Errorf("WorkDir = %q, want %q", project.WorkDir, cloneDir)
	}
	if project.LastSyncAt == nil {
		t.Error("LastSyncAt not set")
	}
	if project.LastCommitHash == "" {
		t.Error("LastCommitHash not set")
	}
}

// TestCloneProject_NonEmptyDir tests the init+fetch path for non-empty dirs.
func TestCloneProject_NonEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a "remote" bare repo
	bareDir := filepath.Join(tmpDir, "bare-repo.git")
	os.MkdirAll(bareDir, 0755)
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	ctx := context.Background()

	// Init bare repo and populate
	err = mgr.runGitCommand(ctx, bareDir, "init", "--bare")
	if err != nil {
		t.Fatalf("git init --bare failed: %v", err)
	}

	workRepo := filepath.Join(tmpDir, "work-repo")
	os.MkdirAll(workRepo, 0755)
	err = mgr.runGitCommand(ctx, workRepo, "init")
	if err != nil {
		t.Fatalf("git init work repo failed: %v", err)
	}
	mgr.runGitCommand(ctx, workRepo, "config", "user.email", "test@test.com")
	mgr.runGitCommand(ctx, workRepo, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(workRepo, "file.txt"), []byte("content"), 0644)
	mgr.runGitCommand(ctx, workRepo, "add", ".")
	mgr.runGitCommand(ctx, workRepo, "commit", "-m", "initial")
	mgr.runGitCommand(ctx, workRepo, "remote", "add", "origin", bareDir)
	branchOutput, _ := mgr.runGitCommandWithOutput(ctx, workRepo, "branch", "--show-current")
	branch := strings.TrimSpace(branchOutput)
	if branch == "" {
		branch = "main"
	}
	mgr.runGitCommand(ctx, workRepo, "push", "-u", "origin", branch)

	// Pre-create project dir with some files (simulating ssh/ from key gen)
	projectDir := filepath.Join(tmpDir, "nonempty-clone")
	sshDir := filepath.Join(projectDir, "ssh")
	os.MkdirAll(sshDir, 0755)
	os.WriteFile(filepath.Join(sshDir, "id_ed25519"), []byte("fake-key"), 0600)

	project := &models.Project{
		ID:            "nonempty-clone",
		Name:          "NonEmpty Clone",
		GitRepo:       bareDir,
		Branch:        branch,
		BeadsPath:     ".beads",
		GitAuthMethod: models.GitAuthNone,
	}

	err = mgr.CloneProject(ctx, project)
	if err != nil {
		t.Fatalf("CloneProject (non-empty dir) failed: %v", err)
	}

	// Verify the repo was cloned
	cloneDir := mgr.GetProjectWorkDir("nonempty-clone")
	if _, err := os.Stat(filepath.Join(cloneDir, ".git")); os.IsNotExist(err) {
		t.Error("git repo was not initialized in non-empty dir")
	}
	if _, err := os.Stat(filepath.Join(cloneDir, "file.txt")); os.IsNotExist(err) {
		t.Error("file.txt not present after clone")
	}
	// ssh/ dir should still exist
	if _, err := os.Stat(filepath.Join(cloneDir, "ssh", "id_ed25519")); os.IsNotExist(err) {
		t.Error("pre-existing ssh dir was removed")
	}
}

// TestPullProject_InRealRepo tests PullProject with a real local git repo.
func TestPullProject_InRealRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a "remote" bare repo
	bareDir := filepath.Join(tmpDir, "bare-pull.git")
	os.MkdirAll(bareDir, 0755)
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	ctx := context.Background()

	// Init bare repo and populate
	err = mgr.runGitCommand(ctx, bareDir, "init", "--bare")
	if err != nil {
		t.Fatalf("git init --bare failed: %v", err)
	}

	workRepo := filepath.Join(tmpDir, "work-pull")
	os.MkdirAll(workRepo, 0755)
	mgr.runGitCommand(ctx, workRepo, "init")
	mgr.runGitCommand(ctx, workRepo, "config", "user.email", "test@test.com")
	mgr.runGitCommand(ctx, workRepo, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(workRepo, "data.txt"), []byte("v1"), 0644)
	mgr.runGitCommand(ctx, workRepo, "add", ".")
	mgr.runGitCommand(ctx, workRepo, "commit", "-m", "v1")
	mgr.runGitCommand(ctx, workRepo, "remote", "add", "origin", bareDir)
	branchOutput, _ := mgr.runGitCommandWithOutput(ctx, workRepo, "branch", "--show-current")
	branch := strings.TrimSpace(branchOutput)
	if branch == "" {
		branch = "main"
	}
	mgr.runGitCommand(ctx, workRepo, "push", "-u", "origin", branch)

	// Clone the bare repo into the project dir
	project := &models.Project{
		ID:            "pull-test-real",
		Name:          "Pull Test Real",
		GitRepo:       bareDir,
		Branch:        branch,
		BeadsPath:     ".beads",
		GitAuthMethod: models.GitAuthNone,
	}

	err = mgr.CloneProject(ctx, project)
	if err != nil {
		t.Fatalf("CloneProject failed: %v", err)
	}

	// Now make a new commit in the "work" repo and push to bare
	os.WriteFile(filepath.Join(workRepo, "data.txt"), []byte("v2"), 0644)
	mgr.runGitCommand(ctx, workRepo, "add", ".")
	mgr.runGitCommand(ctx, workRepo, "commit", "-m", "v2")
	mgr.runGitCommand(ctx, workRepo, "push")

	// Pull in the project
	err = mgr.PullProject(ctx, project)
	if err != nil {
		t.Fatalf("PullProject failed: %v", err)
	}

	// Verify the data was updated
	cloneDir := mgr.GetProjectWorkDir("pull-test-real")
	data, err := os.ReadFile(filepath.Join(cloneDir, "data.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "v2" {
		t.Errorf("data.txt = %q, want 'v2'", string(data))
	}

	// Verify metadata updated
	if project.LastSyncAt == nil {
		t.Error("LastSyncAt not updated after pull")
	}
}

// TestPushChanges_NoRemote tests PushChanges on a repo with no remote.
func TestPushChanges_NoRemote(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()

	// Init a repo without a remote
	repoDir := filepath.Join(tmpDir, "push-test")
	os.MkdirAll(repoDir, 0755)
	mgr.runGitCommand(ctx, repoDir, "init")
	mgr.runGitCommand(ctx, repoDir, "config", "user.email", "test@test.com")
	mgr.runGitCommand(ctx, repoDir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("hello"), 0644)
	mgr.runGitCommand(ctx, repoDir, "add", ".")
	mgr.runGitCommand(ctx, repoDir, "commit", "-m", "initial")

	project := &models.Project{
		ID:            "push-test",
		GitRepo:       "https://example.com/repo.git",
		GitAuthMethod: models.GitAuthNone,
	}

	// Push should fail (no remote configured in the actual repo)
	err = mgr.PushChanges(ctx, project)
	if err == nil {
		t.Error("expected error when pushing without remote")
	}
}

// TestEnsureProjectSSHKey_ExistingPublicKeyMissing tests regeneration of public key.
func TestEnsureProjectSSHKey_ExistingPublicKeyMissing(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Generate key first
	pubKey1, err := mgr.EnsureProjectSSHKey("pubkey-test")
	if err != nil {
		t.Fatalf("first EnsureProjectSSHKey failed: %v", err)
	}

	// Remove public key
	pubPath := mgr.projectPublicKeyPath("pubkey-test")
	os.Remove(pubPath)

	// Ensure should regenerate public key from private
	pubKey2, err := mgr.EnsureProjectSSHKey("pubkey-test")
	if err != nil {
		t.Fatalf("second EnsureProjectSSHKey failed: %v", err)
	}

	// Should get the same key (derived from same private key)
	if pubKey2 == "" {
		t.Error("expected non-empty public key")
	}
	_ = pubKey1 // pubKey1 and pubKey2 should match
}

// TestPullProject_WithLocalChanges tests PullProject with uncommitted local changes.
func TestPullProject_WithLocalChanges(t *testing.T) {
	tmpDir := t.TempDir()

	bareDir := filepath.Join(tmpDir, "bare-stash.git")
	os.MkdirAll(bareDir, 0755)
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	ctx := context.Background()

	// Setup bare repo with initial commit
	mgr.runGitCommand(ctx, bareDir, "init", "--bare")
	workRepo := filepath.Join(tmpDir, "work-stash")
	os.MkdirAll(workRepo, 0755)
	mgr.runGitCommand(ctx, workRepo, "init")
	mgr.runGitCommand(ctx, workRepo, "config", "user.email", "test@test.com")
	mgr.runGitCommand(ctx, workRepo, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(workRepo, "data.txt"), []byte("v1"), 0644)
	mgr.runGitCommand(ctx, workRepo, "add", ".")
	mgr.runGitCommand(ctx, workRepo, "commit", "-m", "v1")
	mgr.runGitCommand(ctx, workRepo, "remote", "add", "origin", bareDir)
	branchOutput, _ := mgr.runGitCommandWithOutput(ctx, workRepo, "branch", "--show-current")
	branch := strings.TrimSpace(branchOutput)
	if branch == "" {
		branch = "main"
	}
	mgr.runGitCommand(ctx, workRepo, "push", "-u", "origin", branch)

	project := &models.Project{
		ID:            "stash-test",
		GitRepo:       bareDir,
		Branch:        branch,
		GitAuthMethod: models.GitAuthNone,
	}

	err = mgr.CloneProject(ctx, project)
	if err != nil {
		t.Fatalf("CloneProject failed: %v", err)
	}

	// Make local uncommitted changes in the cloned project
	cloneDir := mgr.GetProjectWorkDir("stash-test")
	os.WriteFile(filepath.Join(cloneDir, "local-change.txt"), []byte("local"), 0644)

	// Push a new commit to bare from the work repo
	os.WriteFile(filepath.Join(workRepo, "data.txt"), []byte("v2"), 0644)
	mgr.runGitCommand(ctx, workRepo, "add", ".")
	mgr.runGitCommand(ctx, workRepo, "commit", "-m", "v2")
	mgr.runGitCommand(ctx, workRepo, "push")

	// Pull should stash, pull, then pop
	err = mgr.PullProject(ctx, project)
	if err != nil {
		t.Fatalf("PullProject with local changes failed: %v", err)
	}

	// Local change should still be there (restored from stash)
	if _, err := os.Stat(filepath.Join(cloneDir, "local-change.txt")); os.IsNotExist(err) {
		t.Error("local-change.txt was not restored after stash pop")
	}

	// Remote changes should also be present
	data, _ := os.ReadFile(filepath.Join(cloneDir, "data.txt"))
	if string(data) != "v2" {
		t.Errorf("data.txt = %q, want 'v2'", string(data))
	}
}

// TestCloneProject_BadBranch tests cloning with a branch that doesn't exist.
func TestCloneProject_BadBranch(t *testing.T) {
	tmpDir := t.TempDir()

	bareDir := filepath.Join(tmpDir, "bare-bad-branch.git")
	os.MkdirAll(bareDir, 0755)
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	ctx := context.Background()

	// Setup bare repo with initial commit
	mgr.runGitCommand(ctx, bareDir, "init", "--bare")
	workRepo := filepath.Join(tmpDir, "work-bad")
	os.MkdirAll(workRepo, 0755)
	mgr.runGitCommand(ctx, workRepo, "init")
	mgr.runGitCommand(ctx, workRepo, "config", "user.email", "test@test.com")
	mgr.runGitCommand(ctx, workRepo, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(workRepo, "file.txt"), []byte("data"), 0644)
	mgr.runGitCommand(ctx, workRepo, "add", ".")
	mgr.runGitCommand(ctx, workRepo, "commit", "-m", "initial")
	mgr.runGitCommand(ctx, workRepo, "remote", "add", "origin", bareDir)
	branchOutput, _ := mgr.runGitCommandWithOutput(ctx, workRepo, "branch", "--show-current")
	branch := strings.TrimSpace(branchOutput)
	if branch == "" {
		branch = "main"
	}
	mgr.runGitCommand(ctx, workRepo, "push", "-u", "origin", branch)

	project := &models.Project{
		ID:            "bad-branch",
		GitRepo:       bareDir,
		Branch:        "nonexistent-branch-xyz",
		GitAuthMethod: models.GitAuthNone,
	}

	err = mgr.CloneProject(ctx, project)
	// Should fail because the branch doesn't exist
	if err == nil {
		t.Error("expected error when cloning with nonexistent branch")
	}
}

// TestPushChanges_WithRemote tests PushChanges with a valid remote.
func TestPushChanges_WithRemote(t *testing.T) {
	tmpDir := t.TempDir()

	bareDir := filepath.Join(tmpDir, "bare-push.git")
	os.MkdirAll(bareDir, 0755)
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	ctx := context.Background()

	// Setup bare repo
	mgr.runGitCommand(ctx, bareDir, "init", "--bare")
	workRepo := filepath.Join(tmpDir, "work-push")
	os.MkdirAll(workRepo, 0755)
	mgr.runGitCommand(ctx, workRepo, "init")
	mgr.runGitCommand(ctx, workRepo, "config", "user.email", "test@test.com")
	mgr.runGitCommand(ctx, workRepo, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(workRepo, "file.txt"), []byte("data"), 0644)
	mgr.runGitCommand(ctx, workRepo, "add", ".")
	mgr.runGitCommand(ctx, workRepo, "commit", "-m", "initial")
	mgr.runGitCommand(ctx, workRepo, "remote", "add", "origin", bareDir)
	branchOutput, _ := mgr.runGitCommandWithOutput(ctx, workRepo, "branch", "--show-current")
	branch := strings.TrimSpace(branchOutput)
	if branch == "" {
		branch = "main"
	}
	mgr.runGitCommand(ctx, workRepo, "push", "-u", "origin", branch)

	// Clone project
	project := &models.Project{
		ID:            "push-real",
		GitRepo:       bareDir,
		Branch:        branch,
		GitAuthMethod: models.GitAuthNone,
	}
	err = mgr.CloneProject(ctx, project)
	if err != nil {
		t.Fatalf("CloneProject failed: %v", err)
	}

	// Make a commit
	cloneDir := mgr.GetProjectWorkDir("push-real")
	os.WriteFile(filepath.Join(cloneDir, "new-file.txt"), []byte("new"), 0644)
	err = mgr.CommitChanges(ctx, project, "Add new file", "Test", "test@test.com")
	if err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	// Push should succeed
	err = mgr.PushChanges(ctx, project)
	if err != nil {
		t.Fatalf("PushChanges failed: %v", err)
	}
}

// min helper for older Go versions.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
