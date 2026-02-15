package actions

import (
	"errors"
	"testing"
)

func TestParseSimpleJSON_Scope(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "scope", "path": "src"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(env.Actions))
	}
	if env.Actions[0].Type != ActionReadTree {
		t.Errorf("expected read_tree, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Path != "src" {
		t.Errorf("expected path src, got %s", env.Actions[0].Path)
	}
	if env.Actions[0].MaxDepth != 2 {
		t.Errorf("expected depth 2, got %d", env.Actions[0].MaxDepth)
	}
}

func TestParseSimpleJSON_ScopeDefault(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "scope"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Path != "." {
		t.Errorf("expected default path '.', got %s", env.Actions[0].Path)
	}
}

func TestParseSimpleJSON_Tree(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "tree", "path": "internal"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionReadTree {
		t.Errorf("expected read_tree, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].MaxDepth != 3 {
		t.Errorf("expected depth 3, got %d", env.Actions[0].MaxDepth)
	}
}

func TestParseSimpleJSON_Read(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "read", "path": "main.go"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionReadFile {
		t.Errorf("expected read_file, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Path != "main.go" {
		t.Errorf("expected path main.go, got %s", env.Actions[0].Path)
	}
}

func TestParseSimpleJSON_ReadMissingPath(t *testing.T) {
	_, err := ParseSimpleJSON([]byte(`{"action": "read"}`))
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestParseSimpleJSON_Search(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "search", "query": "TODO", "path": "src"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionSearchText {
		t.Errorf("expected search_text, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Query != "TODO" {
		t.Errorf("expected query TODO, got %s", env.Actions[0].Query)
	}
	if env.Actions[0].Path != "src" {
		t.Errorf("expected path src, got %s", env.Actions[0].Path)
	}
}

func TestParseSimpleJSON_SearchMissingQuery(t *testing.T) {
	_, err := ParseSimpleJSON([]byte(`{"action": "search"}`))
	if err == nil {
		t.Fatal("expected error for missing query")
	}
}

func TestParseSimpleJSON_Edit(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "edit", "path": "foo.go", "old": "x := 1", "new": "x := 2"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionEditCode {
		t.Errorf("expected edit_code, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].OldText != "x := 1" {
		t.Errorf("expected old text, got %s", env.Actions[0].OldText)
	}
	if env.Actions[0].NewText != "x := 2" {
		t.Errorf("expected new text, got %s", env.Actions[0].NewText)
	}
}

func TestParseSimpleJSON_EditMissingFields(t *testing.T) {
	_, err := ParseSimpleJSON([]byte(`{"action": "edit", "path": "foo.go"}`))
	if err == nil {
		t.Fatal("expected error for missing old")
	}
	_, err = ParseSimpleJSON([]byte(`{"action": "edit", "old": "x"}`))
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestParseSimpleJSON_Write(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "write", "path": "foo.go", "content": "package foo"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionWriteFile {
		t.Errorf("expected write_file, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Content != "package foo" {
		t.Errorf("expected content, got %s", env.Actions[0].Content)
	}
}

func TestParseSimpleJSON_WriteMissingFields(t *testing.T) {
	_, err := ParseSimpleJSON([]byte(`{"action": "write", "path": "foo.go"}`))
	if err == nil {
		t.Fatal("expected error for missing content")
	}
	_, err = ParseSimpleJSON([]byte(`{"action": "write", "content": "x"}`))
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestParseSimpleJSON_Build(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "build"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionBuildProject {
		t.Errorf("expected build_project, got %s", env.Actions[0].Type)
	}
}

func TestParseSimpleJSON_Test(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "test", "pattern": "TestFoo"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionRunTests {
		t.Errorf("expected run_tests, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].TestPattern != "TestFoo" {
		t.Errorf("expected pattern TestFoo, got %s", env.Actions[0].TestPattern)
	}
}

func TestParseSimpleJSON_Bash(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "bash", "command": "ls -la"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionRunCommand {
		t.Errorf("expected run_command, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Command != "ls -la" {
		t.Errorf("expected command, got %s", env.Actions[0].Command)
	}
}

func TestParseSimpleJSON_BashMissingCommand(t *testing.T) {
	_, err := ParseSimpleJSON([]byte(`{"action": "bash"}`))
	if err == nil {
		t.Fatal("expected error for missing command")
	}
}

func TestParseSimpleJSON_GitCommit(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "git_commit", "message": "fix: bug"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionGitCommit {
		t.Errorf("expected git_commit, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].CommitMessage != "fix: bug" {
		t.Errorf("expected message, got %s", env.Actions[0].CommitMessage)
	}
}

func TestParseSimpleJSON_GitPush(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "git_push"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionGitPush {
		t.Errorf("expected git_push, got %s", env.Actions[0].Type)
	}
}

func TestParseSimpleJSON_GitStatus(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "git_status"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionGitStatus {
		t.Errorf("expected git_status, got %s", env.Actions[0].Type)
	}
}

func TestParseSimpleJSON_Done(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "done", "reason": "all done"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionDone {
		t.Errorf("expected done, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Reason != "all done" {
		t.Errorf("expected reason, got %s", env.Actions[0].Reason)
	}
}

func TestParseSimpleJSON_CloseBead(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "close_bead", "reason": "completed"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionCloseBead {
		t.Errorf("expected close_bead, got %s", env.Actions[0].Type)
	}
}

func TestParseSimpleJSON_Escalate(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "escalate", "reason": "need help"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionEscalateCEO {
		t.Errorf("expected escalate_ceo, got %s", env.Actions[0].Type)
	}
}

func TestParseSimpleJSON_UnknownAction(t *testing.T) {
	_, err := ParseSimpleJSON([]byte(`{"action": "fly_to_moon"}`))
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestParseSimpleJSON_MissingAction(t *testing.T) {
	_, err := ParseSimpleJSON([]byte(`{"path": "foo.go"}`))
	if err == nil {
		t.Fatal("expected error for missing action")
	}
}

func TestParseSimpleJSON_LegacyFormat(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"actions": [{"type": "done", "reason": "completed"}]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(env.Actions))
	}
	if env.Actions[0].Type != ActionDone {
		t.Errorf("expected done, got %s", env.Actions[0].Type)
	}
}

func TestParseSimpleJSON_InvalidJSON(t *testing.T) {
	_, err := ParseSimpleJSON([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseSimpleJSON_WithNotes(t *testing.T) {
	env, err := ParseSimpleJSON([]byte(`{"action": "build", "notes": "checking build"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Notes != "checking build" {
		t.Errorf("expected notes, got %s", env.Notes)
	}
}
