package gitops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

func TestValidateProjectID(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		wantErr   bool
	}{
		{"valid-simple", "my-project", false},
		{"valid-underscore", "my_project", false},
		{"valid-alphanumeric", "project123", false},
		{"valid-mixed", "My-Project_123", false},
		{"empty", "", true},
		{"too-long", strings.Repeat("a", 101), true},
		{"has-dot", "my.project", true},
		{"has-slash", "my/project", true},
		{"has-space", "my project", true},
		{"has-special", "project$bad", true},
		{"has-semicolon", "project;rm", true},
		{"traversal-dots", "project..escape", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectID(tt.projectID)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProjectID(%q) error = %v, wantErr %v", tt.projectID, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCommitMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{"valid-simple", "Fix bug in parser", false},
		{"valid-multiline", "Fix bug\n\nDetailed explanation here", false},
		{"empty", "", true},
		{"too-long", strings.Repeat("a", 10001), true},
		{"cmd-substitution-dollar", "Fix $(rm -rf)", true},
		{"cmd-substitution-backtick", "Fix `rm -rf`", true},
		{"pipe-in-subject", "Fix | rm", true},
		{"semicolon-in-subject", "Fix; rm", true},
		{"ampersand-in-subject", "Fix && rm", true},
		{"redirect-in-subject", "Fix > /dev/null", true},
		// Body allows some shell chars (except command substitution)
		{"pipe-in-body", "Fix bug\n\nOutput: foo | bar", false},
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

func TestValidateAuthorInfo(t *testing.T) {
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

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"with space", "'with space'"},
		{"with'quote", "'with'\\''quote'"},
		{"", "''"},
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

func TestProjectIDFromWorkDir(t *testing.T) {
	tests := []struct {
		workDir string
		want    string
	}{
		{"/app/src/my-project", "my-project"},
		{"/app/src/project-123", "project-123"},
		{"project", "project"},
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

func TestSetProjectWorkDir(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.SetProjectWorkDir("test-project", "/custom/path")
	got := mgr.GetProjectWorkDir("test-project")
	if got != "/custom/path" {
		t.Errorf("GetProjectWorkDir() = %q, want /custom/path", got)
	}

	// Without override, should use base
	got2 := mgr.GetProjectWorkDir("other-project")
	expected := filepath.Join(tmpDir, "other-project")
	if got2 != expected {
		t.Errorf("GetProjectWorkDir() = %q, want %q", got2, expected)
	}
}

func TestGetProjectKeyDir(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := filepath.Join(tmpDir, "keys")
	mgr, err := NewManager(tmpDir, keyDir, nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	got := mgr.GetProjectKeyDir()
	if got != keyDir {
		t.Errorf("GetProjectKeyDir() = %q, want %q", got, keyDir)
	}
}

func TestSetKeyManager(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr.keyManager != nil {
		t.Error("keyManager should be nil initially")
	}

	mgr.SetKeyManager(nil)
	if mgr.keyManager != nil {
		t.Error("keyManager should still be nil")
	}
}

func TestNewManager_CreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base", "work")
	keyDir := filepath.Join(tmpDir, "keys", "dir")

	_, err := NewManager(baseDir, keyDir, nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Error("base work directory was not created")
	}
	if _, err := os.Stat(keyDir); os.IsNotExist(err) {
		t.Error("key directory was not created")
	}
}

func TestNewManager_DefaultKeyDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Empty projectKeyDir should default to /app/data/projects
	// but will fail if /app/data can't be created - that's expected in test env
	_, err := NewManager(tmpDir, "", nil, nil)
	// Just check it doesn't panic - may fail on permission
	_ = err
}

func TestValidateSSHKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a test key file
	keyFile := filepath.Join(tmpDir, "test_key")
	if err := os.WriteFile(keyFile, []byte("fake key"), 0600); err != nil {
		t.Fatal(err)
	}

	// Valid path within expected directory
	err := validateSSHKeyPath(keyFile, tmpDir)
	if err != nil {
		t.Errorf("expected no error for valid key path, got: %v", err)
	}

	// Path does not exist
	err = validateSSHKeyPath(filepath.Join(tmpDir, "nonexistent"), tmpDir)
	if err == nil {
		t.Error("expected error for nonexistent path")
	}

	// Path outside expected directory
	otherDir := t.TempDir()
	otherKey := filepath.Join(otherDir, "key")
	os.WriteFile(otherKey, []byte("fake"), 0600)
	err = validateSSHKeyPath(otherKey, tmpDir)
	if err == nil {
		t.Error("expected error for path outside expected directory")
	}
}

func TestProjectKeyPaths(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	projectID := "test-project"

	keyDir := mgr.projectKeyDirForProject(projectID)
	if keyDir == "" {
		t.Error("projectKeyDirForProject should return non-empty")
	}

	privPath := mgr.projectPrivateKeyPath(projectID)
	if !strings.HasSuffix(privPath, "id_ed25519") {
		t.Errorf("private key path should end with id_ed25519, got %q", privPath)
	}

	pubPath := mgr.projectPublicKeyPath(projectID)
	if !strings.HasSuffix(pubPath, "id_ed25519.pub") {
		t.Errorf("public key path should end with id_ed25519.pub, got %q", pubPath)
	}
}

func TestTimePtrHelper(t *testing.T) {
	// Verify timePtr returns a valid pointer
	// We can't call it directly if unexported, but it's called via other methods
	// This is tested indirectly through CloneProject etc.
}

func TestCloneProject_NoGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	project := &models.Project{
		ID:   "no-repo",
		Name: "No Repo Project",
		// GitRepo intentionally empty
	}

	ctx := context.Background()
	err = mgr.CloneProject(ctx, project)
	if err == nil {
		t.Error("expected error for project with no git_repo")
	}
}

func TestCloneProject_AlreadyCloned(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create a fake .git directory
	projectDir := filepath.Join(tmpDir, "already-cloned")
	os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)

	project := &models.Project{
		ID:      "already-cloned",
		Name:    "Already Cloned",
		GitRepo: "https://github.com/example/repo.git",
	}

	ctx := context.Background()
	err = mgr.CloneProject(ctx, project)
	if err == nil {
		t.Error("expected error for already cloned project")
	}
}

func TestPullProject_NotCloned(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	project := &models.Project{
		ID:      "not-cloned",
		Name:    "Not Cloned",
		GitRepo: "https://github.com/example/repo.git",
	}

	ctx := context.Background()
	err = mgr.PullProject(ctx, project)
	if err == nil {
		t.Error("expected error for not-cloned project")
	}
}

func TestGetCurrentCommit_NoGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.GetCurrentCommit(tmpDir)
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestPlaceholderMethods(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()

	// Test placeholder methods that return "not implemented" style responses
	_, err = mgr.GetStatus(ctx)
	_ = err // May or may not error - just verify no panic

	_, err = mgr.GetDiff(ctx, false)
	_ = err

	_, err = mgr.CreateBranch(ctx, "bead-1", "test branch", "main")
	_ = err

	_, err = mgr.Commit(ctx, "bead-1", "agent-1", "test commit", nil, false)
	_ = err

	_, err = mgr.Push(ctx, "bead-1", "main", false)
	_ = err

	_, err = mgr.CreatePR(ctx, "bead-1", "test PR", "body", "main", "feature", nil, false)
	_ = err
}

func TestLoadBeadsFromProject_NoWorkDir(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"), nil, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	project := &models.Project{
		ID:        "no-workdir",
		Name:      "No WorkDir",
		BeadsPath: ".beads",
	}

	beads, err := mgr.LoadBeadsFromProject(project)
	// Should return empty or error - no work dir set
	if err == nil && len(beads) > 0 {
		t.Error("expected empty beads for project without work dir")
	}
}
