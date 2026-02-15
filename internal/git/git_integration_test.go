package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestGitRepo creates a temporary git repository for testing.
func setupTestGitRepo(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "loom-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	// Initialize git repo
	if err := execGit(dir, "init"); err != nil {
		cleanup()
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	if err := execGit(dir, "config", "user.email", "test@loom.dev"); err != nil {
		cleanup()
		t.Fatalf("failed to config email: %v", err)
	}
	if err := execGit(dir, "config", "user.name", "Loom Test"); err != nil {
		cleanup()
		t.Fatalf("failed to config name: %v", err)
	}

	// Create initial commit
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644); err != nil {
		cleanup()
		t.Fatalf("failed to write README: %v", err)
	}
	if err := execGit(dir, "add", "README.md"); err != nil {
		cleanup()
		t.Fatalf("failed to git add: %v", err)
	}
	if err := execGit(dir, "commit", "-m", "Initial commit"); err != nil {
		cleanup()
		t.Fatalf("failed to git commit: %v", err)
	}

	return dir, cleanup
}

// createTestGitService creates a GitService with an AuditLogger that uses temp dir for logging.
func createTestGitService(t *testing.T, repoDir string) *GitService {
	t.Helper()
	logDir, err := os.MkdirTemp("", "loom-audit-*")
	if err != nil {
		t.Fatalf("failed to create audit log dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(logDir) })

	logger := &AuditLogger{
		projectID: "test-project",
		logPath:   filepath.Join(logDir, "git_audit.log"),
	}

	return &GitService{
		projectPath:   repoDir,
		projectID:     "test-project",
		projectKeyDir: filepath.Join(logDir, "keys"),
		branchPrefix:  "agent/",
		auditLogger:   logger,
	}
}

func execGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &testGitError{args: args, output: string(output), err: err}
	}
	return nil
}

type testGitError struct {
	args   []string
	output string
	err    error
}

func (e *testGitError) Error() string {
	return e.err.Error() + ": " + e.output
}

func TestIsGitRepo(t *testing.T) {
	t.Run("valid git repo", func(t *testing.T) {
		dir, cleanup := setupTestGitRepo(t)
		defer cleanup()

		if !isGitRepo(dir) {
			t.Error("expected isGitRepo to return true for valid repo")
		}
	})

	t.Run("non-git directory", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "non-git-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(dir)

		if isGitRepo(dir) {
			t.Error("expected isGitRepo to return false for non-git directory")
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		if isGitRepo("/nonexistent/path/12345") {
			t.Error("expected isGitRepo to return false for nonexistent path")
		}
	})
}

func TestNewGitServiceNonGitDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "non-git-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	_, err = NewGitService(dir, "test-project")
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestNewAuditLoggerWithTempDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "audit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	logger := &AuditLogger{
		projectID: "test-project-logger",
		logPath:   filepath.Join(dir, "audit.log"),
	}
	if logger.projectID != "test-project-logger" {
		t.Errorf("projectID: expected test-project-logger, got %s", logger.projectID)
	}
	if logger.logPath == "" {
		t.Error("logPath should not be empty")
	}
}

func TestAuditLoggerLogOperation(t *testing.T) {
	dir, err := os.MkdirTemp("", "audit-log-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	logger := &AuditLogger{
		projectID: "test-project",
		logPath:   filepath.Join(dir, "audit.log"),
	}

	// Log an operation
	logger.LogOperation("test_op", "bead-1", "ref-1", true, nil)

	// Verify log file was created
	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logStr := string(content)
	if !strings.Contains(logStr, "test_op") {
		t.Error("log should contain operation name")
	}
	if !strings.Contains(logStr, "bead-1") {
		t.Error("log should contain bead ID")
	}
	if !strings.Contains(logStr, "test-project") {
		t.Error("log should contain project ID")
	}
}

func TestAuditLoggerLogOperationWithError(t *testing.T) {
	dir, err := os.MkdirTemp("", "audit-log-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	logger := &AuditLogger{
		projectID: "test-project",
		logPath:   filepath.Join(dir, "audit.log"),
	}

	testErr := &testGitError{err: os.ErrPermission, output: "permission denied"}
	logger.LogOperation("failed_op", "bead-2", "ref-2", false, testErr)

	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logStr := string(content)
	if !strings.Contains(logStr, "failed_op") {
		t.Error("log should contain operation name")
	}
	if !strings.Contains(logStr, `"success":false`) {
		t.Error("log should show success=false")
	}
}

func TestAuditLoggerLogOperationWithDuration(t *testing.T) {
	dir, err := os.MkdirTemp("", "audit-log-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	logger := &AuditLogger{
		projectID: "test-project",
		logPath:   filepath.Join(dir, "audit.log"),
	}

	logger.LogOperationWithDuration("timed_op", "bead-3", "ref-3", true, nil, 1500000000) // 1.5s

	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logStr := string(content)
	if !strings.Contains(logStr, "timed_op") {
		t.Error("log should contain operation name")
	}
	if !strings.Contains(logStr, "duration_ms") {
		t.Error("log should contain duration_ms")
	}
}

func TestGitServiceGetStatus(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	status, err := svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status == "" {
		t.Error("expected non-empty status")
	}
}

func TestGitServiceGetDiff(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// No changes - empty diff
	diff, err := svc.GetDiff(ctx, false)
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff for clean repo, got %q", diff)
	}

	// Make a change
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Changed\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Unstaged diff should show changes
	diff, err = svc.GetDiff(ctx, false)
	if err != nil {
		t.Fatalf("GetDiff (unstaged) failed: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff after modification")
	}

	// Staged diff should be empty (nothing staged yet)
	diff, err = svc.GetDiff(ctx, true)
	if err != nil {
		t.Fatalf("GetDiff (staged) failed: %v", err)
	}
	if diff != "" {
		t.Error("expected empty staged diff before staging")
	}
}

func TestGitServiceBranchOperations(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create a branch
	result, err := svc.CreateBranch(ctx, CreateBranchRequest{
		BeadID:      "bead-test",
		Description: "Test Feature",
	})
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Created {
		t.Error("expected Created=true")
	}
	if result.Existed {
		t.Error("expected Existed=false")
	}
	if !strings.HasPrefix(result.BranchName, "agent/") {
		t.Errorf("expected branch name to start with 'agent/', got %q", result.BranchName)
	}

	// Creating same branch should return Existed=true
	// First, go back to a different branch
	if err := execGit(dir, "checkout", "-"); err != nil {
		t.Fatalf("failed to checkout back: %v", err)
	}
	result2, err := svc.CreateBranch(ctx, CreateBranchRequest{
		BeadID:      "bead-test",
		Description: "Test Feature",
	})
	if err != nil {
		t.Fatalf("CreateBranch (second time) failed: %v", err)
	}
	if result2.Created {
		t.Error("expected Created=false on second call")
	}
	if !result2.Existed {
		t.Error("expected Existed=true on second call")
	}
}

func TestGitServiceCurrentBranch(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	branch, err := svc.getCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("getCurrentBranch failed: %v", err)
	}
	if branch == "" {
		t.Error("expected non-empty branch name")
	}
}

func TestGitServiceBranchExists(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Current branch should exist
	currentBranch, _ := svc.getCurrentBranch(ctx)
	exists, err := svc.branchExists(ctx, currentBranch)
	if err != nil {
		t.Fatalf("branchExists failed: %v", err)
	}
	if !exists {
		t.Error("expected current branch to exist")
	}

	// Non-existent branch
	exists, err = svc.branchExists(ctx, "nonexistent-branch-12345")
	if err != nil {
		t.Fatalf("branchExists failed: %v", err)
	}
	if exists {
		t.Error("expected non-existent branch to not exist")
	}
}

func TestGitServiceLog(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	entries, err := svc.Log(ctx, LogRequest{MaxCount: 10})
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one log entry")
	}
	if len(entries) > 0 {
		if entries[0].SHA == "" {
			t.Error("expected non-empty SHA in log entry")
		}
		if entries[0].Subject != "Initial commit" {
			t.Errorf("expected 'Initial commit', got %q", entries[0].Subject)
		}
		if entries[0].Author != "Loom Test" {
			t.Errorf("expected author 'Loom Test', got %q", entries[0].Author)
		}
	}
}

func TestGitServiceLogMaxCount(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Zero max count should default to 20
	entries, err := svc.Log(ctx, LogRequest{MaxCount: 0})
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one entry with default max count")
	}

	// Negative max count should default to 20
	entries, err = svc.Log(ctx, LogRequest{MaxCount: -5})
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one entry with negative max count")
	}
}

func TestGitServiceLogWithBranch(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Get current branch name
	branch, err := svc.getCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("getCurrentBranch failed: %v", err)
	}

	entries, err := svc.Log(ctx, LogRequest{MaxCount: 10, Branch: branch})
	if err != nil {
		t.Fatalf("Log with branch failed: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one entry")
	}
}

func TestGitServiceListBranches(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	branches, err := svc.ListBranches(ctx)
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	if len(branches) == 0 {
		t.Error("expected at least one branch")
	}

	hasCurrent := false
	for _, b := range branches {
		if b.IsCurrent {
			hasCurrent = true
			break
		}
	}
	if !hasCurrent {
		t.Error("expected at least one branch to be current")
	}
}

func TestGitServiceStageFiles(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// No files and allowAll=false should return error
	err := svc.stageFiles(ctx, nil, false)
	if err == nil {
		t.Error("expected error when no files and allowAll=false")
	}

	// Create a test file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Stage specific file
	err = svc.stageFiles(ctx, []string{"test.txt"}, false)
	if err != nil {
		t.Fatalf("stageFiles with specific file failed: %v", err)
	}

	// Stage all
	testFile2 := filepath.Join(dir, "test2.txt")
	if err := os.WriteFile(testFile2, []byte("test content 2"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	err = svc.stageFiles(ctx, nil, true)
	if err != nil {
		t.Fatalf("stageFiles with allowAll failed: %v", err)
	}
}

func TestGitServiceGetLastCommitSHA(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	sha, err := svc.getLastCommitSHA(ctx)
	if err != nil {
		t.Fatalf("getLastCommitSHA failed: %v", err)
	}
	if sha == "" {
		t.Error("expected non-empty SHA")
	}
	if len(sha) < 7 {
		t.Errorf("SHA seems too short: %q", sha)
	}
}

func TestGitServiceCheckForSecretsNoSecrets(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create and stage a clean file
	testFile := filepath.Join(dir, "clean.go")
	if err := os.WriteFile(testFile, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	if err := execGit(dir, "add", "clean.go"); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	err := svc.checkForSecrets(ctx)
	if err != nil {
		t.Fatalf("checkForSecrets should not error for clean files: %v", err)
	}
}

func TestGitServiceCheckForSecretsWithSecrets(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create and stage a file with a secret
	testFile := filepath.Join(dir, "config.go")
	content := "package config\napi_key=\"abcdefghijklmnopqrstuvwxyz\"\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	if err := execGit(dir, "add", "config.go"); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	err := svc.checkForSecrets(ctx)
	if err == nil {
		t.Error("expected error for file containing secrets")
	}
}

func TestGitServiceBuildEnv(t *testing.T) {
	svc := &GitService{
		projectPath: "/tmp/test",
		projectID:   "test-project",
	}

	env := svc.buildEnv()
	if len(env) == 0 {
		t.Error("expected non-empty environment")
	}
}

func TestGitServiceCommitWithMetadata(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create a file to commit
	testFile := filepath.Join(dir, "feature.go")
	if err := os.WriteFile(testFile, []byte("package feature\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result, err := svc.Commit(ctx, CommitRequest{
		BeadID:   "bead-commit-test",
		AgentID:  "agent-commit-test",
		Message:  "Add feature module",
		Files:    []string{"feature.go"},
		AllowAll: false,
	})
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil commit result")
	}
	if result.CommitSHA == "" {
		t.Error("expected non-empty commit SHA")
	}
}

func TestGitServiceGetCommitStats(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	sha, err := svc.getLastCommitSHA(ctx)
	if err != nil {
		t.Fatalf("getLastCommitSHA failed: %v", err)
	}

	stats, err := svc.getCommitStats(ctx, sha)
	if err != nil {
		t.Fatalf("getCommitStats failed: %v", err)
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.CommitSHA != sha {
		t.Errorf("CommitSHA: expected %s, got %s", sha, stats.CommitSHA)
	}
}

func TestGitServicePushForceBlocked(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	_, err := svc.Push(ctx, PushRequest{
		BeadID: "bead-1",
		Branch: "agent/bead-1/fix",
		Force:  true,
	})
	if err == nil {
		t.Error("expected error for force push")
	}
	if !strings.Contains(err.Error(), "force push is not allowed") {
		t.Errorf("expected 'force push is not allowed', got %q", err.Error())
	}
}

func TestGitServiceDiffBranches(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Get current branch
	currentBranch, _ := svc.getCurrentBranch(ctx)

	// Create a new branch with changes
	if err := execGit(dir, "checkout", "-b", "agent/test/diff-branch"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new-file.txt"), []byte("new content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := execGit(dir, "add", "new-file.txt"); err != nil {
		t.Fatalf("failed to stage: %v", err)
	}
	if err := execGit(dir, "commit", "-m", "Add new file"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	diff, err := svc.DiffBranches(ctx, DiffBranchesRequest{
		Branch1: currentBranch,
		Branch2: "agent/test/diff-branch",
	})
	if err != nil {
		t.Fatalf("DiffBranches failed: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff between branches")
	}
}

func TestGitServiceStashSaveAndPop(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Make changes
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Modified\n"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Stash save
	err := svc.StashSave(ctx, "test stash")
	if err != nil {
		t.Fatalf("StashSave failed: %v", err)
	}

	// Verify working tree is clean
	status, _ := svc.GetStatus(ctx)
	if strings.Contains(status, "modified") {
		t.Error("working tree should be clean after stash")
	}

	// Stash pop
	err = svc.StashPop(ctx)
	if err != nil {
		t.Fatalf("StashPop failed: %v", err)
	}
}

func TestGitServiceCheckoutClean(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create a branch to switch to
	if err := execGit(dir, "branch", "agent/test-checkout"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	result, err := svc.Checkout(ctx, CheckoutRequest{Branch: "agent/test-checkout"})
	if err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Branch != "agent/test-checkout" {
		t.Errorf("Branch: expected agent/test-checkout, got %s", result.Branch)
	}
}

func TestGitServiceCheckoutDirtyFails(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create a branch
	if err := execGit(dir, "branch", "agent/dirty-test"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Make uncommitted changes
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Dirty\n"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	_, err := svc.Checkout(ctx, CheckoutRequest{Branch: "agent/dirty-test"})
	if err == nil {
		t.Error("expected error for checkout with dirty working tree")
	}
}

func TestGitServiceFetch(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Fetch should fail for repo without remote (but not panic)
	err := svc.Fetch(ctx)
	if err == nil {
		// No remote configured, this could succeed or fail depending on git version
		t.Log("Fetch succeeded (no remote configured)")
	} else {
		// Expected - no remote
		t.Logf("Fetch failed as expected without remote: %v", err)
	}
}

func TestGitServiceMerge(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Ensure we start on a non-protected branch
	if err := execGit(dir, "checkout", "-b", "agent/merge-target"); err != nil {
		t.Fatalf("failed to create target branch: %v", err)
	}

	// Create a source branch with changes
	if err := execGit(dir, "checkout", "-b", "agent/merge-source"); err != nil {
		t.Fatalf("failed to create source branch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "merge-file.txt"), []byte("merge content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := execGit(dir, "add", "merge-file.txt"); err != nil {
		t.Fatalf("failed to stage: %v", err)
	}
	if err := execGit(dir, "commit", "-m", "Add merge file"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Switch back to target branch
	if err := execGit(dir, "checkout", "agent/merge-target"); err != nil {
		t.Fatalf("failed to checkout target: %v", err)
	}

	// Merge source into target
	result, err := svc.Merge(ctx, MergeRequest{
		SourceBranch: "agent/merge-source",
		Message:      "Merge source into target",
		NoFF:         true,
		BeadID:       "bead-merge",
	})
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil merge result")
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.MergedBranch != "agent/merge-source" {
		t.Errorf("expected MergedBranch=agent/merge-source, got %s", result.MergedBranch)
	}
	if result.CommitSHA == "" {
		t.Error("expected non-empty CommitSHA")
	}
}

func TestGitServiceMergeNonExistentBranch(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Switch to non-protected branch
	if err := execGit(dir, "checkout", "-b", "agent/merge-base"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	_, err := svc.Merge(ctx, MergeRequest{
		SourceBranch: "nonexistent-branch-xyz",
		BeadID:       "bead-1",
	})
	if err == nil {
		t.Error("expected error for merging non-existent branch")
	}
}

func TestGitServiceMergeIntoProtectedBranch(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Rename current branch to "main" (protected)
	currentBranch, _ := svc.getCurrentBranch(ctx)
	if currentBranch != "main" && currentBranch != "master" {
		if err := execGit(dir, "branch", "-m", currentBranch, "main"); err != nil {
			t.Fatalf("failed to rename branch to main: %v", err)
		}
	}

	// Create a source branch
	if err := execGit(dir, "branch", "agent/source-branch"); err != nil {
		t.Fatalf("failed to create source branch: %v", err)
	}

	// Try to merge into main (protected) - should fail
	_, err := svc.Merge(ctx, MergeRequest{
		SourceBranch: "agent/source-branch",
		BeadID:       "bead-protected",
	})
	if err == nil {
		t.Error("expected error for merging into protected branch")
	}
	if err != nil && !strings.Contains(err.Error(), "protected branch") {
		t.Errorf("expected 'protected branch' error, got: %v", err)
	}
}

func TestGitServiceMergeWithoutNoFF(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create a non-protected base branch
	if err := execGit(dir, "checkout", "-b", "agent/ff-target"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Create source branch with changes
	if err := execGit(dir, "checkout", "-b", "agent/ff-source"); err != nil {
		t.Fatalf("failed to create source branch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ff-file.txt"), []byte("ff content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := execGit(dir, "add", "ff-file.txt"); err != nil {
		t.Fatalf("failed to stage: %v", err)
	}
	if err := execGit(dir, "commit", "-m", "Add ff file"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Switch back and merge with fast-forward (NoFF=false)
	if err := execGit(dir, "checkout", "agent/ff-target"); err != nil {
		t.Fatalf("failed to checkout: %v", err)
	}

	result, err := svc.Merge(ctx, MergeRequest{
		SourceBranch: "agent/ff-source",
		NoFF:         false, // allow fast-forward
		BeadID:       "bead-ff",
	})
	if err != nil {
		t.Fatalf("fast-forward Merge failed: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
}

func TestGitServiceRevert(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Make a commit to revert
	if err := os.WriteFile(filepath.Join(dir, "revert-file.txt"), []byte("to be reverted"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := execGit(dir, "add", "revert-file.txt"); err != nil {
		t.Fatalf("failed to stage: %v", err)
	}
	if err := execGit(dir, "commit", "-m", "Add file to revert"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Get the SHA of the commit to revert
	sha, err := svc.getLastCommitSHA(ctx)
	if err != nil {
		t.Fatalf("failed to get SHA: %v", err)
	}

	// Revert it
	result, err := svc.Revert(ctx, RevertRequest{
		CommitSHAs: []string{sha},
		BeadID:     "bead-revert",
		Reason:     "testing revert",
	})
	if err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if len(result.RevertedSHAs) != 1 {
		t.Errorf("expected 1 reverted SHA, got %d", len(result.RevertedSHAs))
	}
	if result.RevertedSHAs[0] != sha {
		t.Errorf("expected reverted SHA %s, got %s", sha, result.RevertedSHAs[0])
	}
	if result.NewCommitSHA == "" {
		t.Error("expected non-empty NewCommitSHA")
	}

	// Verify the file is gone
	if _, err := os.Stat(filepath.Join(dir, "revert-file.txt")); err == nil {
		t.Error("expected revert-file.txt to be removed after revert")
	}
}

func TestGitServiceRevertEmptySHAs(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	_, err := svc.Revert(ctx, RevertRequest{
		CommitSHAs: nil,
		BeadID:     "bead-empty",
	})
	if err == nil {
		t.Error("expected error for empty commit SHAs")
	}
	if err != nil && !strings.Contains(err.Error(), "no commit SHAs") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGitServiceRevertInvalidSHA(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	_, err := svc.Revert(ctx, RevertRequest{
		CommitSHAs: []string{"0000000000000000000000000000000000000000"},
		BeadID:     "bead-invalid",
	})
	if err == nil {
		t.Error("expected error for invalid SHA")
	}
}

func TestGitServiceDeleteBranch(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create a branch to delete
	if err := execGit(dir, "branch", "agent/to-delete"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	result, err := svc.DeleteBranch(ctx, DeleteBranchRequest{
		Branch: "agent/to-delete",
	})
	if err != nil {
		t.Fatalf("DeleteBranch failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.DeletedLocal {
		t.Error("expected DeletedLocal=true")
	}
	if result.Branch != "agent/to-delete" {
		t.Errorf("expected Branch=agent/to-delete, got %s", result.Branch)
	}

	// Verify branch no longer exists
	exists, _ := svc.branchExists(ctx, "agent/to-delete")
	if exists {
		t.Error("branch should no longer exist after deletion")
	}
}

func TestGitServiceDeleteBranchProtected(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	_, err := svc.DeleteBranch(ctx, DeleteBranchRequest{
		Branch: "main",
	})
	if err == nil {
		t.Error("expected error for deleting protected branch")
	}
	if err != nil && !strings.Contains(err.Error(), "protected branch") {
		t.Errorf("expected 'protected branch' error, got: %v", err)
	}
}

func TestGitServiceDeleteCurrentBranch(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Switch to a non-protected branch first
	if err := execGit(dir, "checkout", "-b", "agent/current-branch"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Try to delete the current branch
	_, err := svc.DeleteBranch(ctx, DeleteBranchRequest{
		Branch: "agent/current-branch",
	})
	if err == nil {
		t.Error("expected error for deleting current branch")
	}
	if err != nil && !strings.Contains(err.Error(), "cannot delete current branch") {
		t.Errorf("expected 'cannot delete current branch' error, got: %v", err)
	}
}

func TestGitServiceDeleteBranchWithRemote(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create a branch to delete
	if err := execGit(dir, "branch", "agent/remote-delete"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Delete with remote flag - remote delete will fail (no remote) but local delete should succeed
	result, err := svc.DeleteBranch(ctx, DeleteBranchRequest{
		Branch:       "agent/remote-delete",
		DeleteRemote: true,
	})
	if err != nil {
		t.Fatalf("DeleteBranch failed: %v", err)
	}
	if !result.DeletedLocal {
		t.Error("expected DeletedLocal=true")
	}
	// Remote delete should fail silently (no remote configured)
	if result.DeletedRemote {
		t.Error("expected DeletedRemote=false (no remote)")
	}
}

func TestGitServiceGetBeadCommits(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create a commit with bead metadata
	beadID := "bead-search-test"
	if err := os.WriteFile(filepath.Join(dir, "bead-file.txt"), []byte("bead content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := execGit(dir, "add", "bead-file.txt"); err != nil {
		t.Fatalf("failed to stage: %v", err)
	}
	commitMsg := "Add bead file\n\nBead: " + beadID + "\nAgent: agent-search"
	if err := execGit(dir, "commit", "-m", commitMsg); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Search for commits with this bead ID
	commits, err := svc.GetBeadCommits(ctx, beadID)
	if err != nil {
		t.Fatalf("GetBeadCommits failed: %v", err)
	}
	if len(commits) == 0 {
		t.Error("expected at least one commit for bead")
	}
	if len(commits) > 0 {
		if commits[0].BeadID != beadID {
			t.Errorf("expected BeadID=%s, got %s", beadID, commits[0].BeadID)
		}
		if commits[0].AgentID != "agent-search" {
			t.Errorf("expected AgentID=agent-search, got %s", commits[0].AgentID)
		}
		if commits[0].SHA == "" {
			t.Error("expected non-empty SHA")
		}
	}
}

func TestGitServiceGetBeadCommitsNoResults(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	commits, err := svc.GetBeadCommits(ctx, "nonexistent-bead-xyz")
	if err != nil {
		t.Fatalf("GetBeadCommits failed: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits, got %d", len(commits))
	}
}

func TestNewAuditLoggerWithTempHome(t *testing.T) {
	// Create a temp dir to act as HOME
	tmpHome, err := os.MkdirTemp("", "loom-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Override HOME environment variable
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	logger, err := NewAuditLogger("test-audit-project")
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	if logger.projectID != "test-audit-project" {
		t.Errorf("projectID: expected test-audit-project, got %s", logger.projectID)
	}
	if !strings.Contains(logger.logPath, "test-audit-project") {
		t.Errorf("logPath should contain project ID, got %s", logger.logPath)
	}
	if !strings.Contains(logger.logPath, "git_audit.log") {
		t.Errorf("logPath should contain git_audit.log, got %s", logger.logPath)
	}

	// Verify the directory was created
	logDir := filepath.Dir(logger.logPath)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("log directory should have been created")
	}
}

func TestNewGitServiceWithTempHome(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Create a temp dir to act as HOME
	tmpHome, err := os.MkdirTemp("", "loom-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Override HOME so NewAuditLogger can create dirs
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	svc, err := NewGitService(dir, "test-project-new")
	if err != nil {
		t.Fatalf("NewGitService failed: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.projectPath != dir {
		t.Errorf("projectPath: expected %s, got %s", dir, svc.projectPath)
	}
	if svc.projectID != "test-project-new" {
		t.Errorf("projectID: expected test-project-new, got %s", svc.projectID)
	}
	if svc.branchPrefix != "agent/" {
		t.Errorf("branchPrefix: expected agent/, got %s", svc.branchPrefix)
	}
	if svc.auditLogger == nil {
		t.Error("expected non-nil auditLogger")
	}
}

func TestNewGitServiceWithCustomKeyDir(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Create a temp dir to act as HOME
	tmpHome, err := os.MkdirTemp("", "loom-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	customKeyDir := filepath.Join(tmpHome, "custom-keys")
	svc, err := NewGitService(dir, "test-project-keys", customKeyDir)
	if err != nil {
		t.Fatalf("NewGitService failed: %v", err)
	}
	if svc.projectKeyDir != customKeyDir {
		t.Errorf("projectKeyDir: expected %s, got %s", customKeyDir, svc.projectKeyDir)
	}
}

func TestNewGitServiceDefaultKeyDir(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Create a temp dir to act as HOME
	tmpHome, err := os.MkdirTemp("", "loom-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	svc, err := NewGitService(dir, "test-project-default")
	if err != nil {
		t.Fatalf("NewGitService failed: %v", err)
	}
	// Default keyDir should be /app/data/projects
	if svc.projectKeyDir != filepath.Join("/app/data", "projects") {
		t.Errorf("expected default key dir /app/data/projects, got %s", svc.projectKeyDir)
	}
}

func TestConfigureSSHKeyNotFound(t *testing.T) {
	svc := &GitService{
		projectPath:   "/tmp/test-repo",
		projectID:     "test-project",
		projectKeyDir: "/nonexistent/key/dir",
		branchPrefix:  "agent/",
	}

	err := svc.configureSSH()
	if err == nil {
		t.Error("expected error when SSH key does not exist")
	}
	if err != nil && !strings.Contains(err.Error(), "SSH key not found") {
		t.Errorf("expected 'SSH key not found' error, got: %v", err)
	}
}

func TestConfigureSSHWithKeyFile(t *testing.T) {
	// Create a temp directory structure matching the expected key path
	tmpDir, err := os.MkdirTemp("", "ssh-key-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectID := "test-ssh-project"
	keyDir := filepath.Join(tmpDir, projectID, "ssh")
	if err := os.MkdirAll(keyDir, 0755); err != nil {
		t.Fatalf("failed to create key dir: %v", err)
	}

	// Create a dummy key file
	keyPath := filepath.Join(keyDir, "id_ed25519")
	if err := os.WriteFile(keyPath, []byte("dummy-key"), 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	svc := &GitService{
		projectPath:   "/tmp/test-repo",
		projectID:     projectID,
		projectKeyDir: tmpDir,
		branchPrefix:  "agent/",
	}

	err = svc.configureSSH()
	if err != nil {
		t.Fatalf("configureSSH failed: %v", err)
	}

	// Verify GIT_SSH_COMMAND was set
	sshCmd := os.Getenv("GIT_SSH_COMMAND")
	if sshCmd == "" {
		t.Error("expected GIT_SSH_COMMAND to be set")
	}
	if !strings.Contains(sshCmd, "id_ed25519") {
		t.Errorf("expected GIT_SSH_COMMAND to contain key path, got %s", sshCmd)
	}

	// Clean up env
	os.Unsetenv("GIT_SSH_COMMAND")
}

func TestGitServiceRunPrePushTestsNoIndicator(t *testing.T) {
	// Create an empty dir with no go.mod, package.json, or Makefile
	tmpDir, err := os.MkdirTemp("", "no-tests-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	svc := &GitService{
		projectPath: tmpDir,
	}

	// Should succeed (no test infrastructure found, allow push)
	err = svc.runPrePushTests(context.Background())
	if err != nil {
		t.Errorf("expected nil error for project without test infrastructure, got: %v", err)
	}
}

func TestGitServiceLogMaxCountCapped(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// MaxCount > 100 should be capped to 100
	entries, err := svc.Log(ctx, LogRequest{MaxCount: 200})
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	// We only have 1 commit, so just verify it didn't error
	if len(entries) == 0 {
		t.Error("expected at least one log entry")
	}
}

func TestGitServiceCommitAllowAll(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create files to commit
	if err := os.WriteFile(filepath.Join(dir, "file1.go"), []byte("package file1\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file2.go"), []byte("package file2\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	result, err := svc.Commit(ctx, CommitRequest{
		BeadID:   "bead-allow-all",
		AgentID:  "agent-allow-all",
		Message:  "Add multiple files",
		AllowAll: true,
	})
	if err != nil {
		t.Fatalf("Commit with AllowAll failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.CommitSHA == "" {
		t.Error("expected non-empty CommitSHA")
	}
}

func TestGitServiceCommitNoFilesNoAllowAll(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	_, err := svc.Commit(ctx, CommitRequest{
		BeadID:   "bead-no-files",
		AgentID:  "agent-no-files",
		Message:  "Should fail",
		Files:    nil,
		AllowAll: false,
	})
	if err == nil {
		t.Error("expected error when no files and allowAll=false")
	}
}

func TestGitServicePushRunsPrePushTests(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Push without force should reach the pre-push test phase
	// Since the repo has no go.mod etc., pre-push tests pass but configureSSH will fail
	_, err := svc.Push(ctx, PushRequest{
		BeadID: "bead-push-test",
		Branch: "agent/test/push-branch",
		Force:  false,
	})
	// Should fail at configureSSH (SSH key not found) since pre-push tests pass
	if err == nil {
		t.Error("expected error (SSH config should fail)")
	}
}

func TestGitServiceCreateBranchWithBaseBranch(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Get current branch as base
	currentBranch, _ := svc.getCurrentBranch(ctx)

	result, err := svc.CreateBranch(ctx, CreateBranchRequest{
		BeadID:      "bead-base",
		Description: "Feature from base",
		BaseBranch:  currentBranch,
	})
	if err != nil {
		t.Fatalf("CreateBranch with base failed: %v", err)
	}
	if !result.Created {
		t.Error("expected Created=true")
	}
}

func TestGitServiceStashSaveWithoutMessage(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Make changes
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Stash test\n"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Stash save without message
	err := svc.StashSave(ctx, "")
	if err != nil {
		t.Fatalf("StashSave without message failed: %v", err)
	}

	// Pop to restore
	err = svc.StashPop(ctx)
	if err != nil {
		t.Fatalf("StashPop failed: %v", err)
	}
}

func TestGitServiceMultipleCommitsAndRevert(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Make two commits
	for i := 1; i <= 2; i++ {
		fname := filepath.Join(dir, fmt.Sprintf("multi-file-%d.txt", i))
		if err := os.WriteFile(fname, []byte(fmt.Sprintf("content %d", i)), 0644); err != nil {
			t.Fatalf("failed to write file %d: %v", i, err)
		}
		if err := execGit(dir, "add", fmt.Sprintf("multi-file-%d.txt", i)); err != nil {
			t.Fatalf("failed to stage file %d: %v", i, err)
		}
		if err := execGit(dir, "commit", "-m", fmt.Sprintf("Add file %d", i)); err != nil {
			t.Fatalf("failed to commit %d: %v", i, err)
		}
	}

	// Get the SHA of the last commit
	sha, _ := svc.getLastCommitSHA(ctx)

	// Revert just the last commit
	result, err := svc.Revert(ctx, RevertRequest{
		CommitSHAs: []string{sha},
		BeadID:     "bead-multi-revert",
	})
	if err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
}

func TestGitServiceDeleteUnmergedBranch(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create a branch with unique commits (unmerged)
	if err := execGit(dir, "checkout", "-b", "agent/unmerged"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unmerged.txt"), []byte("unmerged"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := execGit(dir, "add", "unmerged.txt"); err != nil {
		t.Fatalf("failed to stage: %v", err)
	}
	if err := execGit(dir, "commit", "-m", "Unmerged commit"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Switch back to original branch
	if err := execGit(dir, "checkout", "-"); err != nil {
		t.Fatalf("failed to checkout: %v", err)
	}

	// Delete unmerged branch - should still succeed (force delete)
	result, err := svc.DeleteBranch(ctx, DeleteBranchRequest{
		Branch: "agent/unmerged",
	})
	if err != nil {
		t.Fatalf("DeleteBranch (unmerged) failed: %v", err)
	}
	if !result.DeletedLocal {
		t.Error("expected DeletedLocal=true")
	}
}

func TestGitServiceMergeWithMessage(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create non-protected base branch
	if err := execGit(dir, "checkout", "-b", "agent/msg-target"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Create source branch with changes
	if err := execGit(dir, "checkout", "-b", "agent/msg-source"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "msg-file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := execGit(dir, "add", "msg-file.txt"); err != nil {
		t.Fatalf("failed to stage: %v", err)
	}
	if err := execGit(dir, "commit", "-m", "Add msg file"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Switch to target and merge with a custom message
	if err := execGit(dir, "checkout", "agent/msg-target"); err != nil {
		t.Fatalf("failed to checkout: %v", err)
	}

	result, err := svc.Merge(ctx, MergeRequest{
		SourceBranch: "agent/msg-source",
		Message:      "Custom merge message",
		NoFF:         true,
		BeadID:       "bead-msg",
	})
	if err != nil {
		t.Fatalf("Merge with message failed: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
}

func TestGitServicePushCurrentBranch(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Push without specifying branch (should use current branch)
	// Will fail at pre-push tests or SSH config but exercises getCurrentBranch path
	_, err := svc.Push(ctx, PushRequest{
		BeadID: "bead-current-push",
		Branch: "", // empty = use current
		Force:  false,
	})
	// Should fail but not panic
	if err == nil {
		t.Log("Push succeeded unexpectedly (probably no remote)")
	}
}

func TestGitServiceCommitWithSecrets(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create a file with secrets
	secretFile := filepath.Join(dir, "secrets.go")
	content := "package config\nvar apiKey = `api_key=\"abcdefghijklmnopqrstuvwxyz\"`\n"
	if err := os.WriteFile(secretFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	_, err := svc.Commit(ctx, CommitRequest{
		BeadID:  "bead-secret",
		AgentID: "agent-secret",
		Message: "Add secrets",
		Files:   []string{"secrets.go"},
	})
	if err == nil {
		t.Error("expected error for commit with secrets")
	}
	if err != nil && !strings.Contains(err.Error(), "secret detected") {
		t.Errorf("expected 'secret detected' error, got: %v", err)
	}
}

func TestGitServiceLogFormatParsing(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	// Create multiple commits to test log parsing
	for i := 1; i <= 3; i++ {
		fname := filepath.Join(dir, fmt.Sprintf("log-file-%d.txt", i))
		if err := os.WriteFile(fname, []byte(fmt.Sprintf("content %d", i)), 0644); err != nil {
			t.Fatalf("failed to write: %v", err)
		}
		if err := execGit(dir, "add", fmt.Sprintf("log-file-%d.txt", i)); err != nil {
			t.Fatalf("failed to stage: %v", err)
		}
		if err := execGit(dir, "commit", "-m", fmt.Sprintf("Commit number %d", i)); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}
	}

	entries, err := svc.Log(ctx, LogRequest{MaxCount: 5})
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	// Should have 4 commits (initial + 3)
	if len(entries) != 4 {
		t.Errorf("expected 4 log entries, got %d", len(entries))
	}
	// Most recent first
	if len(entries) > 0 && entries[0].Subject != "Commit number 3" {
		t.Errorf("expected most recent commit first, got %q", entries[0].Subject)
	}
	// Each entry should have all fields
	for i, entry := range entries {
		if entry.SHA == "" {
			t.Errorf("entry %d: missing SHA", i)
		}
		if entry.Author == "" {
			t.Errorf("entry %d: missing Author", i)
		}
		if entry.Date == "" {
			t.Errorf("entry %d: missing Date", i)
		}
		if entry.Subject == "" {
			t.Errorf("entry %d: missing Subject", i)
		}
	}
}

func TestGitServiceGetBeadCommitsMultiple(t *testing.T) {
	dir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	svc := createTestGitService(t, dir)
	ctx := context.Background()

	beadID := "bead-multi-search"

	// Create multiple commits for the same bead
	for i := 1; i <= 3; i++ {
		fname := filepath.Join(dir, fmt.Sprintf("bead-multi-%d.txt", i))
		if err := os.WriteFile(fname, []byte(fmt.Sprintf("content %d", i)), 0644); err != nil {
			t.Fatalf("failed to write: %v", err)
		}
		if err := execGit(dir, "add", fmt.Sprintf("bead-multi-%d.txt", i)); err != nil {
			t.Fatalf("failed to stage: %v", err)
		}
		msg := fmt.Sprintf("Commit %d\n\nBead: %s\nAgent: agent-%d", i, beadID, i)
		if err := execGit(dir, "commit", "-m", msg); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}
	}

	commits, err := svc.GetBeadCommits(ctx, beadID)
	if err != nil {
		t.Fatalf("GetBeadCommits failed: %v", err)
	}
	if len(commits) != 3 {
		t.Errorf("expected 3 commits, got %d", len(commits))
	}
}
