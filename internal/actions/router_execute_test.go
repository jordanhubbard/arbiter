package actions

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jordanhubbard/loom/internal/executor"
	"github.com/jordanhubbard/loom/internal/files"
	"github.com/jordanhubbard/loom/pkg/models"
)

// --- Mock types ---

type mockBeadCloser struct {
	closedIDs []string
	closeErr  error
}

func (m *mockBeadCloser) CloseBead(beadID, reason string) error {
	if m.closeErr != nil {
		return m.closeErr
	}
	m.closedIDs = append(m.closedIDs, beadID)
	return nil
}

type mockBeadEscalator struct {
	escalatedIDs []string
	escalateErr  error
}

func (m *mockBeadEscalator) EscalateBeadToCEO(beadID, reason, returnedTo string) (*models.DecisionBead, error) {
	if m.escalateErr != nil {
		return nil, m.escalateErr
	}
	m.escalatedIDs = append(m.escalatedIDs, beadID)
	return &models.DecisionBead{
		Bead: &models.Bead{ID: "decision-" + beadID},
	}, nil
}

type mockCommandExecutor struct {
	lastReq executor.ExecuteCommandRequest
	result  *executor.ExecuteCommandResult
	err     error
}

func (m *mockCommandExecutor) ExecuteCommand(ctx context.Context, req executor.ExecuteCommandRequest) (*executor.ExecuteCommandResult, error) {
	m.lastReq = req
	if m.err != nil {
		return m.result, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &executor.ExecuteCommandResult{
		ID:       "cmd-1",
		ExitCode: 0,
		Success:  true,
		Stdout:   "ok",
	}, nil
}

type mockFileManager struct {
	readResult   *files.FileResult
	readErr      error
	writeResult  *files.WriteResult
	writeErr     error
	treeResult   []files.TreeEntry
	treeErr      error
	searchResult []files.SearchMatch
	searchErr    error
	patchResult  *files.PatchResult
	patchErr     error
	moveErr      error
	deleteErr    error
	renameErr    error
}

func (m *mockFileManager) ReadFile(ctx context.Context, projectID, path string) (*files.FileResult, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	if m.readResult != nil {
		return m.readResult, nil
	}
	return &files.FileResult{Path: path, Content: "content", Size: 7}, nil
}

func (m *mockFileManager) WriteFile(ctx context.Context, projectID, path, content string) (*files.WriteResult, error) {
	if m.writeErr != nil {
		return nil, m.writeErr
	}
	if m.writeResult != nil {
		return m.writeResult, nil
	}
	return &files.WriteResult{Path: path, BytesWritten: int64(len(content))}, nil
}

func (m *mockFileManager) ReadTree(ctx context.Context, projectID, path string, maxDepth, limit int) ([]files.TreeEntry, error) {
	if m.treeErr != nil {
		return nil, m.treeErr
	}
	return m.treeResult, nil
}

func (m *mockFileManager) SearchText(ctx context.Context, projectID, path, query string, limit int) ([]files.SearchMatch, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.searchResult, nil
}

func (m *mockFileManager) ApplyPatch(ctx context.Context, projectID, patch string) (*files.PatchResult, error) {
	if m.patchErr != nil {
		return m.patchResult, m.patchErr
	}
	if m.patchResult != nil {
		return m.patchResult, nil
	}
	return &files.PatchResult{Applied: true, Output: "applied"}, nil
}

func (m *mockFileManager) MoveFile(ctx context.Context, projectID, sourcePath, targetPath string) error {
	return m.moveErr
}

func (m *mockFileManager) DeleteFile(ctx context.Context, projectID, path string) error {
	return m.deleteErr
}

func (m *mockFileManager) RenameFile(ctx context.Context, projectID, sourcePath, newName string) error {
	return m.renameErr
}

type mockGitOperator struct {
	statusOut string
	statusErr error
	diffOut   string
	diffErr   error
	result    map[string]interface{}
	err       error
}

func (m *mockGitOperator) Status(ctx context.Context, projectID string) (string, error) {
	return m.statusOut, m.statusErr
}
func (m *mockGitOperator) Diff(ctx context.Context, projectID string) (string, error) {
	return m.diffOut, m.diffErr
}
func (m *mockGitOperator) CreateBranch(ctx context.Context, beadID, description, baseBranch string) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) Commit(ctx context.Context, beadID, agentID, message string, f []string, allowAll bool) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) Push(ctx context.Context, beadID, branch string, setUpstream bool) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) GetDiff(ctx context.Context, staged bool) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) CreatePR(ctx context.Context, beadID, title, body, base, branch string, reviewers []string, draft bool) (map[string]interface{}, error) {
	if m.result == nil {
		return map[string]interface{}{"pr_url": "https://github.com/test/pr/1"}, m.err
	}
	return m.result, m.err
}
func (m *mockGitOperator) Merge(ctx context.Context, beadID, sourceBranch, message string, noFF bool) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) Revert(ctx context.Context, beadID string, commitSHAs []string, reason string) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) DeleteBranch(ctx context.Context, branch string, deleteRemote bool) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) Checkout(ctx context.Context, branch string) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) Log(ctx context.Context, branch string, maxCount int) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) Fetch(ctx context.Context) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) ListBranches(ctx context.Context) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) DiffBranches(ctx context.Context, branch1, branch2 string) (map[string]interface{}, error) {
	return m.result, m.err
}
func (m *mockGitOperator) GetBeadCommits(ctx context.Context, beadID string) (map[string]interface{}, error) {
	return m.result, m.err
}

type mockWorkflowOperator struct {
	advanceErr error
}

func (m *mockWorkflowOperator) AdvanceWorkflowWithCondition(beadID, agentID, condition string, resultData map[string]string) error {
	return m.advanceErr
}
func (m *mockWorkflowOperator) StartDevelopment(ctx context.Context, workflow string, requireReviews bool, projectPath string) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockWorkflowOperator) WhatsNext(ctx context.Context, userInput, ctxStr, conversationSummary string, recentMessages []map[string]string) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockWorkflowOperator) ProceedToPhase(ctx context.Context, targetPhase, reviewState, reason string) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockWorkflowOperator) ConductReview(ctx context.Context, targetPhase string) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockWorkflowOperator) ResumeWorkflow(ctx context.Context, includeSystemPrompt bool) (map[string]interface{}, error) {
	return nil, nil
}

type mockLSPOperator struct {
	refResult  map[string]interface{}
	defResult  map[string]interface{}
	implResult map[string]interface{}
	err        error
}

func (m *mockLSPOperator) FindReferences(ctx context.Context, file string, line, column int, symbol string) (map[string]interface{}, error) {
	return m.refResult, m.err
}
func (m *mockLSPOperator) GoToDefinition(ctx context.Context, file string, line, column int, symbol string) (map[string]interface{}, error) {
	return m.defResult, m.err
}
func (m *mockLSPOperator) FindImplementations(ctx context.Context, file string, line, column int, symbol string) (map[string]interface{}, error) {
	return m.implResult, m.err
}

type mockActionLogger struct {
	logged []Action
}

func (m *mockActionLogger) LogAction(ctx context.Context, actx ActionContext, action Action, result Result) {
	m.logged = append(m.logged, action)
}

// --- Tests ---

func TestRouter_Execute_NilEnvelope(t *testing.T) {
	r := &Router{}
	_, err := r.Execute(context.Background(), nil, ActionContext{})
	if err == nil {
		t.Fatal("expected error for nil envelope")
	}
}

func TestRouter_Execute_WithLogger(t *testing.T) {
	logger := &mockActionLogger{}
	r := &Router{Logger: logger}
	env := &ActionEnvelope{
		Actions: []Action{{Type: ActionDone, Reason: "test"}},
	}
	results, err := r.Execute(context.Background(), env, ActionContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(logger.logged) != 1 {
		t.Errorf("expected 1 logged action, got %d", len(logger.logged))
	}
}

func TestRouter_Execute_WithProjectID(t *testing.T) {
	r := &Router{}
	env := &ActionEnvelope{
		Actions: []Action{{Type: ActionDone}},
	}
	actx := ActionContext{ProjectID: "proj-1"}
	results, err := r.Execute(context.Background(), env, actx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result")
	}
}

func TestRouter_AskFollowup_WithBeads(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads}
	result := r.executeAction(context.Background(), Action{Type: ActionAskFollowup, Question: "What next?"}, ActionContext{ProjectID: "p1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_AskFollowup_NoBeads(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionAskFollowup, Question: "What next?"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_ReadCode_WithFiles(t *testing.T) {
	fm := &mockFileManager{
		readResult: &files.FileResult{Path: "foo.go", Content: "package foo", Size: 11},
	}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionReadCode, Path: "foo.go"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
	if result.Metadata["path"] != "foo.go" {
		t.Errorf("expected path foo.go")
	}
}

func TestRouter_ReadCode_Error(t *testing.T) {
	fm := &mockFileManager{readErr: errors.New("not found")}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionReadCode, Path: "missing.go"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_ReadCode_NoFiles(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads}
	result := r.executeAction(context.Background(), Action{Type: ActionReadCode, Path: "foo.go"}, ActionContext{ProjectID: "p1"})
	if result.Status != "executed" {
		t.Errorf("expected fallback bead creation, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_EditCode_TextBased(t *testing.T) {
	fm := &mockFileManager{
		readResult: &files.FileResult{Path: "foo.go", Content: "x := 1", Size: 6},
	}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionEditCode, Path: "foo.go", OldText: "x := 1", NewText: "x := 2"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
	if result.Metadata["match_strategy"] != "exact" {
		t.Errorf("expected exact match, got %v", result.Metadata["match_strategy"])
	}
}

func TestRouter_EditCode_TextBased_ReadError(t *testing.T) {
	fm := &mockFileManager{readErr: errors.New("cannot read")}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionEditCode, Path: "foo.go", OldText: "x", NewText: "y"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_EditCode_TextBased_NoMatch(t *testing.T) {
	fm := &mockFileManager{
		readResult: &files.FileResult{Path: "foo.go", Content: "a := 1", Size: 6},
	}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionEditCode, Path: "foo.go", OldText: "x := 1", NewText: "x := 2"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
	if !containsStr(result.Message, "OLD text not found") {
		t.Errorf("expected match failure message, got %s", result.Message)
	}
}

func TestRouter_EditCode_TextBased_WriteError(t *testing.T) {
	fm := &mockFileManager{
		readResult: &files.FileResult{Path: "foo.go", Content: "x := 1", Size: 6},
		writeErr:   errors.New("write failed"),
	}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionEditCode, Path: "foo.go", OldText: "x := 1", NewText: "x := 2"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_EditCode_PatchBased(t *testing.T) {
	fm := &mockFileManager{
		patchResult: &files.PatchResult{Applied: true, Output: "applied"},
	}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionEditCode, Path: "foo.go", Patch: "diff content"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_EditCode_PatchBased_Error(t *testing.T) {
	fm := &mockFileManager{
		patchErr:    errors.New("patch failed"),
		patchResult: &files.PatchResult{Output: "hunk 1 failed"},
	}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionEditCode, Path: "foo.go", Patch: "bad patch"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_EditCode_NoFiles(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads}
	result := r.executeAction(context.Background(), Action{Type: ActionEditCode, Path: "foo.go", Patch: "diff"}, ActionContext{ProjectID: "p1"})
	if result.ActionType != ActionCreateBead {
		t.Errorf("expected bead creation fallback, got %s", result.ActionType)
	}
}

func TestRouter_WriteFile(t *testing.T) {
	fm := &mockFileManager{}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionWriteFile, Path: "new.go", Content: "package new"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_WriteFile_Error(t *testing.T) {
	fm := &mockFileManager{writeErr: errors.New("write denied")}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionWriteFile, Path: "new.go", Content: "pkg"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_WriteFile_NoFiles(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads}
	result := r.executeAction(context.Background(), Action{Type: ActionWriteFile, Path: "f.go", Content: "pkg"}, ActionContext{ProjectID: "p1"})
	if result.ActionType != ActionCreateBead {
		t.Errorf("expected bead creation fallback")
	}
}

func TestRouter_ReadFile(t *testing.T) {
	fm := &mockFileManager{}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionReadFile, Path: "main.go"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_ReadFile_NoFiles(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionReadFile, Path: "main.go"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_ReadTree(t *testing.T) {
	fm := &mockFileManager{
		treeResult: []files.TreeEntry{{Path: "src/", Type: "dir", Depth: 0}},
	}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionReadTree, Path: "."}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_ReadTree_EmptyPath(t *testing.T) {
	fm := &mockFileManager{}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionReadTree}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_ReadTree_NoFiles(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionReadTree, Path: "."}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_SearchText(t *testing.T) {
	fm := &mockFileManager{
		searchResult: []files.SearchMatch{{Path: "foo.go", Line: 10, Text: "TODO"}},
	}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionSearchText, Query: "TODO"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_SearchText_EmptyPath(t *testing.T) {
	fm := &mockFileManager{}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionSearchText, Query: "foo"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_ApplyPatch(t *testing.T) {
	fm := &mockFileManager{}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionApplyPatch, Patch: "diff"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_ApplyPatch_Error(t *testing.T) {
	fm := &mockFileManager{patchErr: errors.New("fail")}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionApplyPatch, Patch: "bad"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitStatus(t *testing.T) {
	git := &mockGitOperator{statusOut: "clean"}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitStatus}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
	if result.Metadata["output"] != "clean" {
		t.Errorf("expected clean, got %v", result.Metadata["output"])
	}
}

func TestRouter_GitStatus_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitStatus}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitDiff(t *testing.T) {
	git := &mockGitOperator{diffOut: "+new line"}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitDiff}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitCommit(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"commit_sha": "abc123"}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitCommit, CommitMessage: "fix bug"}, ActionContext{BeadID: "bead-1", AgentID: "agent-1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_GitCommit_AutoMessage(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"commit_sha": "abc123"}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitCommit}, ActionContext{BeadID: "bead-1", AgentID: "agent-1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitCommit_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("nothing to commit")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitCommit}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitPush(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"success": true}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitPush, Branch: "main", SetUpstream: true}, ActionContext{BeadID: "bead-1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_CreatePR(t *testing.T) {
	git := &mockGitOperator{}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionCreatePR, PRTitle: "Feature", PRBody: "Desc", PRBase: "develop", Branch: "feature"}, ActionContext{BeadID: "bead-1", AgentID: "agent-1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_CreatePR_Defaults(t *testing.T) {
	git := &mockGitOperator{}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionCreatePR}, ActionContext{BeadID: "bead-1", AgentID: "agent-1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_GitMerge(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"success": true}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitMerge, SourceBranch: "feature"}, ActionContext{BeadID: "bead-1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitRevert(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"success": true}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitRevert, CommitSHAs: []string{"abc"}}, ActionContext{BeadID: "bead-1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitRevert_SingleSHA(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"success": true}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitRevert, CommitSHA: "abc"}, ActionContext{BeadID: "bead-1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitBranchDelete(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"success": true}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitBranchDelete, Branch: "old", DeleteRemote: true}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitCheckout(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"branch": "develop"}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitCheckout, Branch: "develop"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
	if !containsStr(result.Message, "develop") {
		t.Errorf("expected branch name in message")
	}
}

func TestRouter_GitLog(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"entries": []interface{}{}}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitLog, Branch: "main", MaxCount: 10}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitFetch(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"success": true}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitFetch}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitListBranches(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"branches": []interface{}{"main", "develop"}}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitListBranches}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitDiffBranches(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"diff": "some diff"}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitDiffBranches, SourceBranch: "a", TargetBranch: "b"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitBeadCommits(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"commits": []interface{}{}}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitBeadCommits, BeadID: "bead-1"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_GitBeadCommits_FromContext(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"commits": []interface{}{}}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitBeadCommits}, ActionContext{BeadID: "bead-2"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_RunCommand(t *testing.T) {
	cmd := &mockCommandExecutor{}
	r := &Router{Commands: cmd}
	result := r.executeAction(context.Background(), Action{Type: ActionRunCommand, Command: "ls"}, ActionContext{AgentID: "a", BeadID: "b", ProjectID: "p"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_RunCommand_Error(t *testing.T) {
	cmd := &mockCommandExecutor{err: errors.New("permission denied")}
	r := &Router{Commands: cmd}
	result := r.executeAction(context.Background(), Action{Type: ActionRunCommand, Command: "rm -rf /"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_RunCommand_NoExecutor(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads}
	result := r.executeAction(context.Background(), Action{Type: ActionRunCommand, Command: "ls"}, ActionContext{ProjectID: "p1"})
	if result.ActionType != ActionCreateBead {
		t.Errorf("expected bead fallback, got %s", result.ActionType)
	}
}

func TestRouter_CreateBead(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads}
	result := r.executeAction(context.Background(), Action{
		Type: ActionCreateBead,
		Bead: &BeadPayload{Title: "New task", ProjectID: "p1", Priority: 2, Type: "bug"},
	}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_CreateBead_DefaultType(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads, BeadType: "feature"}
	result := r.executeAction(context.Background(), Action{
		Type: ActionCreateBead,
		Bead: &BeadPayload{Title: "task", ProjectID: "p1"},
	}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_CreateBead_NoPayload(t *testing.T) {
	r := &Router{Beads: &mockBeadCreator{}}
	result := r.executeAction(context.Background(), Action{Type: ActionCreateBead}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_CreateBead_NoBeads(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{
		Type: ActionCreateBead,
		Bead: &BeadPayload{Title: "task", ProjectID: "p1"},
	}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_CloseBead(t *testing.T) {
	closer := &mockBeadCloser{}
	r := &Router{Closer: closer}
	result := r.executeAction(context.Background(), Action{Type: ActionCloseBead, BeadID: "bead-1", Reason: "done"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
	if len(closer.closedIDs) != 1 || closer.closedIDs[0] != "bead-1" {
		t.Errorf("expected bead-1 closed")
	}
}

func TestRouter_CloseBead_Error(t *testing.T) {
	closer := &mockBeadCloser{closeErr: errors.New("not found")}
	r := &Router{Closer: closer}
	result := r.executeAction(context.Background(), Action{Type: ActionCloseBead, BeadID: "bead-1"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_CloseBead_NoCloser(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionCloseBead, BeadID: "bead-1"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_EscalateCEO(t *testing.T) {
	esc := &mockBeadEscalator{}
	r := &Router{Escalator: esc}
	result := r.executeAction(context.Background(), Action{Type: ActionEscalateCEO, BeadID: "bead-1", Reason: "blocked"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_EscalateCEO_Error(t *testing.T) {
	esc := &mockBeadEscalator{escalateErr: errors.New("CEO unavailable")}
	r := &Router{Escalator: esc}
	result := r.executeAction(context.Background(), Action{Type: ActionEscalateCEO, BeadID: "bead-1", Reason: "blocked"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_EscalateCEO_NoEscalator(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionEscalateCEO, BeadID: "bead-1"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_ApproveBead(t *testing.T) {
	wf := &mockWorkflowOperator{}
	r := &Router{Workflow: wf}
	result := r.executeAction(context.Background(), Action{Type: ActionApproveBead, BeadID: "bead-1", Reason: "LGTM"}, ActionContext{AgentID: "agent-1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_ApproveBead_NoWorkflow(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionApproveBead, BeadID: "bead-1"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_RejectBead(t *testing.T) {
	wf := &mockWorkflowOperator{}
	r := &Router{Workflow: wf}
	result := r.executeAction(context.Background(), Action{Type: ActionRejectBead, BeadID: "bead-1", Reason: "poor quality"}, ActionContext{AgentID: "agent-1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_RejectBead_Error(t *testing.T) {
	wf := &mockWorkflowOperator{advanceErr: errors.New("workflow error")}
	r := &Router{Workflow: wf}
	result := r.executeAction(context.Background(), Action{Type: ActionRejectBead, BeadID: "bead-1", Reason: "bad"}, ActionContext{AgentID: "agent-1"})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_WorkflowMCPActions(t *testing.T) {
	r := &Router{}
	tests := []struct {
		actionType string
		action     Action
		wantStatus string
	}{
		{ActionStartDev, Action{Type: ActionStartDev, Workflow: "epcc"}, "mcp_required"},
		{ActionWhatsNext, Action{Type: ActionWhatsNext}, "mcp_required"},
		{ActionProceedToPhase, Action{Type: ActionProceedToPhase, TargetPhase: "impl", ReviewState: "done"}, "mcp_required"},
		{ActionConductReview, Action{Type: ActionConductReview, TargetPhase: "design"}, "mcp_required"},
		{ActionResumeWorkflow, Action{Type: ActionResumeWorkflow}, "mcp_required"},
	}

	for _, tt := range tests {
		t.Run(tt.actionType, func(t *testing.T) {
			result := r.executeAction(context.Background(), tt.action, ActionContext{})
			if result.Status != tt.wantStatus {
				t.Errorf("expected status %s, got %s", tt.wantStatus, result.Status)
			}
		})
	}
}

func TestRouter_LSP_FindReferences(t *testing.T) {
	lsp := &mockLSPOperator{refResult: map[string]interface{}{"count": 3}}
	r := &Router{LSP: lsp}
	result := r.executeAction(context.Background(), Action{Type: ActionFindReferences, Path: "foo.go", Symbol: "MyFunc"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_LSP_FindReferences_NoLSP(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionFindReferences, Path: "foo.go", Symbol: "x"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_LSP_GoToDefinition(t *testing.T) {
	lsp := &mockLSPOperator{defResult: map[string]interface{}{"found": true, "file": "bar.go", "line": 10, "column": 5}}
	r := &Router{LSP: lsp}
	result := r.executeAction(context.Background(), Action{Type: ActionGoToDefinition, Path: "foo.go", Symbol: "MyType"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
	if !containsStr(result.Message, "Definition found") {
		t.Errorf("expected found message, got %s", result.Message)
	}
}

func TestRouter_LSP_GoToDefinition_NotFound(t *testing.T) {
	lsp := &mockLSPOperator{defResult: map[string]interface{}{"found": false}}
	r := &Router{LSP: lsp}
	result := r.executeAction(context.Background(), Action{Type: ActionGoToDefinition, Path: "foo.go", Symbol: "Missing"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
	if !containsStr(result.Message, "not found") {
		t.Errorf("expected not found message, got %s", result.Message)
	}
}

func TestRouter_LSP_FindImplementations(t *testing.T) {
	lsp := &mockLSPOperator{implResult: map[string]interface{}{"count": 2}}
	r := &Router{LSP: lsp}
	result := r.executeAction(context.Background(), Action{Type: ActionFindImplementations, Path: "iface.go", Symbol: "Reader"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_RefactoringActions(t *testing.T) {
	r := &Router{}
	tests := []struct {
		name   string
		action Action
	}{
		{"extract_method", Action{Type: ActionExtractMethod, Path: "f.go", MethodName: "doStuff", StartLine: 1, EndLine: 10}},
		{"rename_symbol", Action{Type: ActionRenameSymbol, Path: "f.go", Symbol: "old", NewName: "new"}},
		{"inline_variable", Action{Type: ActionInlineVariable, Path: "f.go", VariableName: "tmp"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.executeAction(context.Background(), tt.action, ActionContext{})
			if result.Status != "executed" {
				t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
			}
		})
	}
}

func TestRouter_FileManagementActions(t *testing.T) {
	fm := &mockFileManager{}
	r := &Router{Files: fm}
	tests := []struct {
		name   string
		action Action
	}{
		{"move_file", Action{Type: ActionMoveFile, SourcePath: "a.go", TargetPath: "b.go"}},
		{"delete_file", Action{Type: ActionDeleteFile, Path: "old.go"}},
		{"rename_file", Action{Type: ActionRenameFile, SourcePath: "a.go", NewName: "b.go"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.executeAction(context.Background(), tt.action, ActionContext{})
			if result.Status != "executed" {
				t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
			}
		})
	}
}

func TestRouter_FileManagement_Errors(t *testing.T) {
	testErr := errors.New("operation failed")
	fm := &mockFileManager{moveErr: testErr, deleteErr: testErr, renameErr: testErr}
	r := &Router{Files: fm}

	tests := []struct {
		name   string
		action Action
	}{
		{"move_file error", Action{Type: ActionMoveFile, SourcePath: "a", TargetPath: "b"}},
		{"delete_file error", Action{Type: ActionDeleteFile, Path: "a"}},
		{"rename_file error", Action{Type: ActionRenameFile, SourcePath: "a", NewName: "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.executeAction(context.Background(), tt.action, ActionContext{})
			if result.Status != "error" {
				t.Errorf("expected error, got %s", result.Status)
			}
		})
	}
}

func TestRouter_FileManagement_NoFileManager(t *testing.T) {
	r := &Router{}
	tests := []struct {
		name   string
		action Action
	}{
		{"move_file", Action{Type: ActionMoveFile, SourcePath: "a", TargetPath: "b"}},
		{"delete_file", Action{Type: ActionDeleteFile, Path: "a"}},
		{"rename_file", Action{Type: ActionRenameFile, SourcePath: "a", NewName: "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.executeAction(context.Background(), tt.action, ActionContext{})
			if result.Status != "error" {
				t.Errorf("expected error, got %s", result.Status)
			}
		})
	}
}

func TestRouter_DebuggingActions(t *testing.T) {
	r := &Router{}
	tests := []struct {
		name   string
		action Action
	}{
		{"add_log", Action{Type: ActionAddLog, Path: "f.go", Line: 42, LogMessage: "debug", LogLevel: "info"}},
		{"add_breakpoint", Action{Type: ActionAddBreakpoint, Path: "f.go", Line: 42, Condition: "x > 0"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.executeAction(context.Background(), tt.action, ActionContext{})
			if result.Status != "executed" {
				t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
			}
		})
	}
}

func TestRouter_GenerateDocs(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGenerateDocs, Path: "f.go", DocFormat: "godoc"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_Done(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionDone, Reason: "all done"}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
	if result.Metadata["reason"] != "all done" {
		t.Errorf("expected reason")
	}
}

func TestRouter_UnknownAction(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: "fly_to_moon"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
	if !containsStr(result.Message, "unsupported") {
		t.Errorf("expected unsupported message, got %s", result.Message)
	}
}

func TestTruncateContent(t *testing.T) {
	short := "hello"
	if truncateContent(short, 10) != "hello" {
		t.Error("short string should be unchanged")
	}

	long := "hello world"
	result := truncateContent(long, 8)
	if result != "hello..." {
		t.Errorf("expected truncated, got %q", result)
	}
}

func TestRouter_CreateBeadFromAction(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads, BeadType: "feature", DefaultP0: true}
	result := r.createBeadFromAction("Title", "Detail", ActionContext{ProjectID: "p1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
	if len(beads.createdBeads) != 1 {
		t.Fatal("expected 1 bead created")
	}
	if beads.createdBeads[0].Priority != models.BeadPriority(0) {
		t.Errorf("expected P0 priority with DefaultP0")
	}
}

func TestRouter_CreateBeadFromAction_DefaultPriority(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads}
	result := r.createBeadFromAction("Title", "Detail", ActionContext{ProjectID: "p1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
	if beads.createdBeads[0].Priority != models.BeadPriority(2) {
		t.Errorf("expected P2 default priority")
	}
}

func TestRouter_CreateBeadFromAction_Error(t *testing.T) {
	beads := &mockBeadCreator{createError: errors.New("creation failed")}
	r := &Router{Beads: beads}
	result := r.createBeadFromAction("Title", "Detail", ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_AutoFileParseFailure(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads}
	result := r.AutoFileParseFailure(context.Background(), ActionContext{ProjectID: "p1"}, fmt.Errorf("parse error"), "raw response")
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestRouter_AutoFileParseFailure_DefaultP0(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads, DefaultP0: true}
	result := r.AutoFileParseFailure(context.Background(), ActionContext{ProjectID: "p1"}, fmt.Errorf("err"), "raw")
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
	if beads.createdBeads[0].Priority != models.BeadPriority(0) {
		t.Errorf("expected P0 for DefaultP0")
	}
}

func TestRouter_AutoFileParseFailure_NoBeads(t *testing.T) {
	r := &Router{}
	result := r.AutoFileParseFailure(context.Background(), ActionContext{}, fmt.Errorf("err"), "raw")
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_AutoFileParseFailure_WithLogger(t *testing.T) {
	beads := &mockBeadCreator{}
	logger := &mockActionLogger{}
	r := &Router{Beads: beads, Logger: logger}
	_ = r.AutoFileParseFailure(context.Background(), ActionContext{ProjectID: "p1"}, fmt.Errorf("err"), "raw")
	if len(logger.logged) != 1 {
		t.Errorf("expected 1 logged action, got %d", len(logger.logged))
	}
}

func TestRouter_AutoFileParseFailure_BeadCreateError(t *testing.T) {
	beads := &mockBeadCreator{createError: errors.New("db error")}
	r := &Router{Beads: beads}
	result := r.AutoFileParseFailure(context.Background(), ActionContext{}, fmt.Errorf("parse err"), "raw")
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_ReadFile_Error(t *testing.T) {
	fm := &mockFileManager{readErr: errors.New("access denied")}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionReadFile, Path: "secret.go"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_ReadTree_Error(t *testing.T) {
	fm := &mockFileManager{treeErr: errors.New("path not found")}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionReadTree, Path: "missing/"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_SearchText_Error(t *testing.T) {
	fm := &mockFileManager{searchErr: errors.New("search failed")}
	r := &Router{Files: fm}
	result := r.executeAction(context.Background(), Action{Type: ActionSearchText, Query: "TODO"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_SearchText_NoFiles(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionSearchText, Query: "TODO"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_ApplyPatch_NoFiles(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionApplyPatch, Patch: "diff"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitStatus_Error(t *testing.T) {
	git := &mockGitOperator{statusErr: errors.New("not a git repo")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitStatus}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitDiff_Error(t *testing.T) {
	git := &mockGitOperator{diffErr: errors.New("diff failed")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitDiff}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitDiff_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitDiff}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitPush_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("push rejected")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitPush, Branch: "main"}, ActionContext{BeadID: "bead-1"})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitPush_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitPush}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_CreatePR_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("PR creation failed")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionCreatePR, PRTitle: "test"}, ActionContext{BeadID: "b1", AgentID: "a1"})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_CreatePR_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionCreatePR}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitMerge_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitMerge, SourceBranch: "f"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitMerge_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("conflict")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitMerge, SourceBranch: "f"}, ActionContext{BeadID: "b1"})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitRevert_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitRevert, CommitSHAs: []string{"abc"}}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitRevert_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("revert failed")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitRevert, CommitSHAs: []string{"abc"}}, ActionContext{BeadID: "b1"})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitBranchDelete_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitBranchDelete, Branch: "old"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitBranchDelete_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("branch not found")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitBranchDelete, Branch: "old"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitCheckout_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitCheckout, Branch: "main"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitCheckout_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("checkout failed")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitCheckout, Branch: "main"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitLog_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitLog}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitLog_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("log failed")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitLog}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitFetch_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitFetch}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitFetch_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("fetch failed")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitFetch}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitListBranches_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitListBranches}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitListBranches_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("list failed")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitListBranches}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitDiffBranches_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitDiffBranches, SourceBranch: "a", TargetBranch: "b"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitDiffBranches_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("diff failed")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitDiffBranches, SourceBranch: "a", TargetBranch: "b"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitBeadCommits_NoGit(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGitBeadCommits, BeadID: "b1"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitBeadCommits_Error(t *testing.T) {
	git := &mockGitOperator{err: errors.New("commits failed")}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitBeadCommits, BeadID: "b1"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_LSP_FindReferences_Error(t *testing.T) {
	lsp := &mockLSPOperator{err: errors.New("lsp error")}
	r := &Router{LSP: lsp}
	result := r.executeAction(context.Background(), Action{Type: ActionFindReferences, Path: "f.go", Symbol: "x"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_LSP_GoToDefinition_Error(t *testing.T) {
	lsp := &mockLSPOperator{err: errors.New("lsp error")}
	r := &Router{LSP: lsp}
	result := r.executeAction(context.Background(), Action{Type: ActionGoToDefinition, Path: "f.go", Symbol: "x"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_LSP_GoToDefinition_NoLSP(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionGoToDefinition, Path: "f.go", Symbol: "x"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_LSP_FindImplementations_Error(t *testing.T) {
	lsp := &mockLSPOperator{err: errors.New("lsp error")}
	r := &Router{LSP: lsp}
	result := r.executeAction(context.Background(), Action{Type: ActionFindImplementations, Path: "f.go", Symbol: "x"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_LSP_FindImplementations_NoLSP(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionFindImplementations, Path: "f.go", Symbol: "x"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_ApproveBead_Error(t *testing.T) {
	wf := &mockWorkflowOperator{advanceErr: errors.New("workflow error")}
	r := &Router{Workflow: wf}
	result := r.executeAction(context.Background(), Action{Type: ActionApproveBead, BeadID: "bead-1"}, ActionContext{AgentID: "a1"})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_CreateBead_Error(t *testing.T) {
	beads := &mockBeadCreator{createError: errors.New("db error")}
	r := &Router{Beads: beads}
	result := r.executeAction(context.Background(), Action{
		Type: ActionCreateBead,
		Bead: &BeadPayload{Title: "task", ProjectID: "p1"},
	}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_CreateBead_DefaultTypeFromRouter(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads}
	result := r.executeAction(context.Background(), Action{
		Type: ActionCreateBead,
		Bead: &BeadPayload{Title: "task", ProjectID: "p1"},
	}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_RunTests_NoRunner(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionRunTests}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_RunTests_Error(t *testing.T) {
	tests := &mockTestRunner{
		runFunc: func(ctx context.Context, projectPath, testPattern, framework string, timeoutSeconds int) (map[string]interface{}, error) {
			return nil, errors.New("test crash")
		},
	}
	r := &Router{Tests: tests}
	result := r.executeAction(context.Background(), Action{Type: ActionRunTests}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_RunLinter_NoLinter(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionRunLinter}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_BuildProject_NoBuilder(t *testing.T) {
	r := &Router{}
	result := r.executeAction(context.Background(), Action{Type: ActionBuildProject}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_GitMerge_WithNoFF(t *testing.T) {
	git := &mockGitOperator{result: map[string]interface{}{"success": true}}
	r := &Router{Git: git}
	result := r.executeAction(context.Background(), Action{Type: ActionGitMerge, SourceBranch: "feature", NoFF: true}, ActionContext{BeadID: "b1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
}

func TestRouter_CreateBeadFromAction_NoBeads(t *testing.T) {
	r := &Router{}
	result := r.createBeadFromAction("Title", "Detail", ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestRouter_CreateBeadFromAction_DefaultType(t *testing.T) {
	beads := &mockBeadCreator{}
	r := &Router{Beads: beads}
	result := r.createBeadFromAction("Title", "Detail", ActionContext{ProjectID: "p1"})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s", result.Status)
	}
	// When BeadType is empty, default is "task"
}
