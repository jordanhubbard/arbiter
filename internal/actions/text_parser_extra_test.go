package actions

import (
	"errors"
	"testing"
)

func TestParseTextAction_Write(t *testing.T) {
	response := "ACTION: WRITE config.json\n<<<\n{\"key\": \"value\"}\n>>>"
	env, err := ParseTextAction(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionWriteFile {
		t.Errorf("expected write_file, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Path != "config.json" {
		t.Errorf("expected path config.json, got %s", env.Actions[0].Path)
	}
	if env.Actions[0].Content == "" {
		t.Error("expected content to be non-empty")
	}
}

func TestParseTextAction_Write_NoBlock(t *testing.T) {
	response := "ACTION: WRITE config.json\nsome plain content"
	env, err := ParseTextAction(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Content == "" {
		t.Error("expected plain content fallback")
	}
}

func TestParseTextAction_Write_EmptyContent(t *testing.T) {
	response := "ACTION: WRITE config.json"
	_, err := ParseTextAction(response)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestParseTextAction_Write_MissingPathExplicit(t *testing.T) {
	// When WRITE has no args at all and no body content
	response := "ACTION: WRITE"
	_, err := ParseTextAction(response)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Logf("got error type %T: %v", err, err)
	}
}

func TestParseTextAction_Build(t *testing.T) {
	env, err := ParseTextAction("ACTION: BUILD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionBuildProject {
		t.Errorf("expected build_project, got %s", env.Actions[0].Type)
	}
}

func TestParseTextAction_Test_WithPattern(t *testing.T) {
	env, err := ParseTextAction("ACTION: TEST TestCalculate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionRunTests {
		t.Errorf("expected run_tests, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].TestPattern != "TestCalculate" {
		t.Errorf("expected pattern TestCalculate, got %s", env.Actions[0].TestPattern)
	}
}

func TestParseTextAction_Test_NoPattern(t *testing.T) {
	env, err := ParseTextAction("ACTION: TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].TestPattern != "" {
		t.Errorf("expected empty pattern, got %s", env.Actions[0].TestPattern)
	}
}

func TestParseTextAction_Tree(t *testing.T) {
	env, err := ParseTextAction("ACTION: TREE src/internal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionReadTree {
		t.Errorf("expected read_tree, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Path != "src/internal" {
		t.Errorf("expected path src/internal, got %s", env.Actions[0].Path)
	}
	if env.Actions[0].MaxDepth != 3 {
		t.Errorf("expected depth 3 for TREE, got %d", env.Actions[0].MaxDepth)
	}
}

func TestParseTextAction_TreeDefault(t *testing.T) {
	env, err := ParseTextAction("ACTION: TREE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Path != "." {
		t.Errorf("expected default path '.', got %s", env.Actions[0].Path)
	}
}

func TestParseTextAction_ScopeDefault(t *testing.T) {
	env, err := ParseTextAction("ACTION: SCOPE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Path != "." {
		t.Errorf("expected default path '.', got %s", env.Actions[0].Path)
	}
	if env.Actions[0].MaxDepth != 2 {
		t.Errorf("expected depth 2 for SCOPE, got %d", env.Actions[0].MaxDepth)
	}
}

func TestParseTextAction_CloseBead(t *testing.T) {
	env, err := ParseTextAction("ACTION: CLOSE_BEAD work complete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionCloseBead {
		t.Errorf("expected close_bead, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Reason != "work complete" {
		t.Errorf("expected reason, got %s", env.Actions[0].Reason)
	}
}

func TestParseTextAction_Escalate(t *testing.T) {
	env, err := ParseTextAction("ACTION: ESCALATE need CEO decision")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionEscalateCEO {
		t.Errorf("expected escalate_ceo, got %s", env.Actions[0].Type)
	}
}

func TestParseTextAction_GitCommit(t *testing.T) {
	env, err := ParseTextAction("ACTION: GIT_COMMIT fix: resolve import issue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionGitCommit {
		t.Errorf("expected git_commit, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].CommitMessage != "fix: resolve import issue" {
		t.Errorf("expected commit message, got %s", env.Actions[0].CommitMessage)
	}
}

func TestParseTextAction_GitPush(t *testing.T) {
	env, err := ParseTextAction("ACTION: GIT_PUSH")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionGitPush {
		t.Errorf("expected git_push, got %s", env.Actions[0].Type)
	}
}

func TestParseTextAction_GitStatus(t *testing.T) {
	env, err := ParseTextAction("ACTION: GIT_STATUS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionGitStatus {
		t.Errorf("expected git_status, got %s", env.Actions[0].Type)
	}
}

func TestParseTextAction_UnknownCommand(t *testing.T) {
	_, err := ParseTextAction("ACTION: FLY_TO_MOON")
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestParseTextAction_SearchNoArgs(t *testing.T) {
	_, err := ParseTextAction("ACTION: SEARCH")
	if err == nil {
		t.Fatal("expected error for empty search")
	}
}

func TestParseTextAction_ReadNoArgs(t *testing.T) {
	_, err := ParseTextAction("ACTION: READ")
	if err == nil {
		t.Fatal("expected error for empty read")
	}
}

func TestParseTextAction_BashNoArgs(t *testing.T) {
	_, err := ParseTextAction("ACTION: BASH")
	if err == nil {
		t.Fatal("expected error for empty bash")
	}
}

func TestParseTextAction_Edit_MissingBlocks(t *testing.T) {
	_, err := ParseTextAction("ACTION: EDIT foo.go\nOLD: something")
	if err == nil {
		t.Fatal("expected error for missing blocks")
	}
}

func TestParseTextAction_NotesExtraction(t *testing.T) {
	response := "Here is my analysis of the code.\n\nACTION: READ main.go"
	env, err := ParseTextAction(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Notes == "" {
		t.Error("expected notes to be extracted")
	}
}

func TestParseTextAction_FirstActionWins(t *testing.T) {
	// When multiple ACTION lines exist, FindStringSubmatch returns the first match
	response := "ACTION: READ first.go\nSome analysis\nACTION: READ second.go"
	env, err := ParseTextAction(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The parser uses FindStringSubmatch which returns the first match
	if env.Actions[0].Path != "first.go" {
		t.Errorf("expected first action (first.go), got %s", env.Actions[0].Path)
	}
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  []string
	}{
		{"", 2, nil},
		{"hello", 2, []string{"hello"}},
		{"hello world", 2, []string{"hello", "world"}},
		{"a b c d", 2, []string{"a", "b c d"}},
		{"a b c", 0, []string{"a", "b", "c"}},
		{"  spaces  everywhere  ", 2, []string{"spaces", "everywhere"}},
	}

	for _, tt := range tests {
		result := splitArgs(tt.input, tt.max)
		if len(result) != len(tt.want) {
			t.Errorf("splitArgs(%q, %d) len = %d, want %d", tt.input, tt.max, len(result), len(tt.want))
			continue
		}
		for i := range result {
			if result[i] != tt.want[i] {
				t.Errorf("splitArgs(%q, %d)[%d] = %q, want %q", tt.input, tt.max, i, result[i], tt.want[i])
			}
		}
	}
}

func TestBuildUnifiedPatch(t *testing.T) {
	patch := buildUnifiedPatch("foo.go", "old line", "new line")
	if patch == "" {
		t.Fatal("expected non-empty patch")
	}
	if !containsStr(patch, "--- a/foo.go") {
		t.Error("expected old file header")
	}
	if !containsStr(patch, "+++ b/foo.go") {
		t.Error("expected new file header")
	}
	if !containsStr(patch, "-old line") {
		t.Error("expected removed line")
	}
	if !containsStr(patch, "+new line") {
		t.Error("expected added line")
	}
}

func TestSimpleError(t *testing.T) {
	err := errorf("test %s error", "message")
	if err.Error() != "test message error" {
		t.Errorf("expected formatted error, got %q", err.Error())
	}
}

func TestErrorfNoArgs(t *testing.T) {
	err := errorf("simple error")
	if err.Error() != "simple error" {
		t.Errorf("expected simple error, got %q", err.Error())
	}
}

func TestReplaceArgs(t *testing.T) {
	result := replaceArgs("hello %s world %s", "big", "wide")
	if result != "hello big world wide" {
		t.Errorf("expected 'hello big world wide', got %q", result)
	}
}

func TestReplaceArgs_NonString(t *testing.T) {
	result := replaceArgs("value: %s", 42)
	if result != "value: <?>" {
		t.Errorf("expected placeholder for non-string, got %q", result)
	}
}

func TestReplaceArgs_NoArgs(t *testing.T) {
	result := replaceArgs("no placeholders")
	if result != "no placeholders" {
		t.Errorf("expected unchanged, got %q", result)
	}
}
