package actions

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeLenient_StrictPassthrough(t *testing.T) {
	payload := `{"actions": [{"type": "done", "reason": "finished"}]}`
	env, err := DecodeLenient([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionDone {
		t.Errorf("expected done, got %s", env.Actions[0].Type)
	}
}

func TestDecodeLenient_MarkdownFences(t *testing.T) {
	payload := "```json\n{\"actions\": [{\"type\": \"done\"}]}\n```"
	env, err := DecodeLenient([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionDone {
		t.Errorf("expected done, got %s", env.Actions[0].Type)
	}
}

func TestDecodeLenient_ThinkTags(t *testing.T) {
	payload := "<think>Let me think about this...</think>{\"actions\": [{\"type\": \"done\"}]}"
	env, err := DecodeLenient([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionDone {
		t.Errorf("expected done, got %s", env.Actions[0].Type)
	}
}

func TestDecodeLenient_ThinkTagsNoOpen(t *testing.T) {
	payload := "Some reasoning text here</think>{\"actions\": [{\"type\": \"done\"}]}"
	env, err := DecodeLenient([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionDone {
		t.Errorf("expected done, got %s", env.Actions[0].Type)
	}
}

func TestDecodeLenient_ExtraTextBeforeJSON(t *testing.T) {
	payload := "Here's the action:\n{\"actions\": [{\"type\": \"done\"}]}"
	env, err := DecodeLenient([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionDone {
		t.Errorf("expected done, got %s", env.Actions[0].Type)
	}
}

func TestDecodeLenient_TotalGarbage(t *testing.T) {
	_, err := DecodeLenient([]byte("no json here at all"))
	if err == nil {
		t.Fatal("expected error for garbage input")
	}
}

func TestStripCodeFences_NoFences(t *testing.T) {
	input := `{"actions": [{"type": "done"}]}`
	result := stripCodeFences([]byte(input))
	if string(result) != input {
		t.Errorf("expected unchanged input, got %q", string(result))
	}
}

func TestStripCodeFences_WithLanguage(t *testing.T) {
	input := "```json\n{\"foo\": true}\n```"
	result := stripCodeFences([]byte(input))
	if strings.Contains(string(result), "```") {
		t.Errorf("expected fences removed, got %q", string(result))
	}
	if !strings.Contains(string(result), `"foo"`) {
		t.Errorf("expected content preserved, got %q", string(result))
	}
}

func TestStripCodeFences_SingleLine(t *testing.T) {
	input := "```json```"
	result := stripCodeFences([]byte(input))
	// Single line should be returned as-is since there's nothing between fences
	_ = result // Just checking it doesn't panic
}

func TestStripThinkTags_PairedTags(t *testing.T) {
	input := "<think>reasoning</think>actual content"
	result := stripThinkTags([]byte(input))
	if strings.Contains(string(result), "reasoning") {
		t.Error("expected think content removed")
	}
	if !strings.Contains(string(result), "actual content") {
		t.Error("expected actual content preserved")
	}
}

func TestStripThinkTags_MultiplePairs(t *testing.T) {
	input := "<think>first</think>mid<think>second</think>end"
	result := stripThinkTags([]byte(input))
	if strings.Contains(string(result), "first") || strings.Contains(string(result), "second") {
		t.Error("expected all think blocks removed")
	}
	if !strings.Contains(string(result), "mid") || !strings.Contains(string(result), "end") {
		t.Error("expected non-think content preserved")
	}
}

func TestStripThinkTags_UnclosedTag(t *testing.T) {
	input := "<think>reasoning without close tag"
	result := stripThinkTags([]byte(input))
	if strings.Contains(string(result), "reasoning") {
		t.Error("expected unclosed think content removed")
	}
}

func TestStripThinkTags_CloseOnly(t *testing.T) {
	input := "some thinking output here</think>{\"actions\": []}"
	result := stripThinkTags([]byte(input))
	if strings.Contains(string(result), "some thinking") {
		t.Error("expected pre-close content removed")
	}
	if !strings.Contains(string(result), "actions") {
		t.Error("expected post-close content preserved")
	}
}

func TestStripThinkTags_NoTags(t *testing.T) {
	input := "just normal content"
	result := stripThinkTags([]byte(input))
	if string(result) != input {
		t.Errorf("expected unchanged, got %q", string(result))
	}
}

func TestExtractJSONObject_Simple(t *testing.T) {
	input := `some text {"key": "value"} more text`
	result, err := extractJSONObject([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != `{"key": "value"}` {
		t.Errorf("expected JSON object, got %q", string(result))
	}
}

func TestExtractJSONObject_Nested(t *testing.T) {
	input := `prefix {"outer": {"inner": "value"}} suffix`
	result, err := extractJSONObject([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != `{"outer": {"inner": "value"}}` {
		t.Errorf("expected nested JSON, got %q", string(result))
	}
}

func TestExtractJSONObject_WithStrings(t *testing.T) {
	input := `text {"key": "value with { braces }"} more`
	result, err := extractJSONObject([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(result), "braces") {
		t.Error("expected string content preserved")
	}
}

func TestExtractJSONObject_WithEscapedQuotes(t *testing.T) {
	input := `prefix {"key": "value with \" escaped"} suffix`
	result, err := extractJSONObject([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(result), "escaped") {
		t.Error("expected escaped content preserved")
	}
}

func TestExtractJSONObject_NoJSON(t *testing.T) {
	_, err := extractJSONObject([]byte("no json here"))
	if err == nil {
		t.Fatal("expected error for no JSON")
	}
}

func TestValidate_NilEnvelope(t *testing.T) {
	err := Validate(nil)
	if err == nil || !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestValidate_EmptyActions(t *testing.T) {
	err := Validate(&ActionEnvelope{})
	if err == nil || !strings.Contains(err.Error(), "at least one action") {
		t.Errorf("expected empty actions error, got %v", err)
	}
}

func TestValidate_MissingType(t *testing.T) {
	err := Validate(&ActionEnvelope{Actions: []Action{{}}})
	if err == nil || !strings.Contains(err.Error(), "missing type") {
		t.Errorf("expected missing type error, got %v", err)
	}
}

func TestValidate_UnknownActionType(t *testing.T) {
	err := Validate(&ActionEnvelope{Actions: []Action{{Type: "fly_to_moon"}}})
	if err == nil || !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("expected unknown action error, got %v", err)
	}
}

func TestDecodeStrict_ValidJSON(t *testing.T) {
	payload := `{"actions": [{"type": "git_status"}]}`
	env, err := DecodeStrict([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(env.Actions))
	}
}

func TestDecodeStrict_UnknownField(t *testing.T) {
	payload := `{"actions": [{"type": "done"}], "bogus_field": true}`
	_, err := DecodeStrict([]byte(payload))
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestDecodeStrict_TrailingTokens(t *testing.T) {
	payload := `{"actions": [{"type": "done"}]}{"extra": true}`
	_, err := DecodeStrict([]byte(payload))
	if err == nil {
		t.Fatal("expected error for trailing tokens")
	}
}

func TestDecodeStrict_ValidationError(t *testing.T) {
	payload := `{"actions": [{"type": "read_file"}]}`
	_, err := DecodeStrict([]byte(payload))
	if err == nil {
		t.Fatal("expected validation error for missing path")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	ve := &ValidationError{Err: inner}
	if ve.Error() != "inner error" {
		t.Errorf("expected 'inner error', got %q", ve.Error())
	}
	if ve.Unwrap() != inner {
		t.Error("expected unwrap to return inner error")
	}
}

func TestValidateAction_ExtendedGit(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		{
			name:    "git_merge valid",
			action:  Action{Type: ActionGitMerge, SourceBranch: "feature"},
			wantErr: false,
		},
		{
			name:    "git_merge missing source",
			action:  Action{Type: ActionGitMerge},
			wantErr: true,
		},
		{
			name:    "git_revert with sha",
			action:  Action{Type: ActionGitRevert, CommitSHA: "abc123"},
			wantErr: false,
		},
		{
			name:    "git_revert with shas",
			action:  Action{Type: ActionGitRevert, CommitSHAs: []string{"abc", "def"}},
			wantErr: false,
		},
		{
			name:    "git_revert missing both",
			action:  Action{Type: ActionGitRevert},
			wantErr: true,
		},
		{
			name:    "git_branch_delete valid",
			action:  Action{Type: ActionGitBranchDelete, Branch: "old-branch"},
			wantErr: false,
		},
		{
			name:    "git_branch_delete missing branch",
			action:  Action{Type: ActionGitBranchDelete},
			wantErr: true,
		},
		{
			name:    "git_checkout valid",
			action:  Action{Type: ActionGitCheckout, Branch: "main"},
			wantErr: false,
		},
		{
			name:    "git_checkout missing branch",
			action:  Action{Type: ActionGitCheckout},
			wantErr: true,
		},
		{
			name:    "git_log no fields",
			action:  Action{Type: ActionGitLog},
			wantErr: false,
		},
		{
			name:    "git_fetch no fields",
			action:  Action{Type: ActionGitFetch},
			wantErr: false,
		},
		{
			name:    "git_list_branches no fields",
			action:  Action{Type: ActionGitListBranches},
			wantErr: false,
		},
		{
			name:    "git_diff_branches valid",
			action:  Action{Type: ActionGitDiffBranches, SourceBranch: "a", TargetBranch: "b"},
			wantErr: false,
		},
		{
			name:    "git_diff_branches missing target",
			action:  Action{Type: ActionGitDiffBranches, SourceBranch: "a"},
			wantErr: true,
		},
		{
			name:    "git_bead_commits no fields",
			action:  Action{Type: ActionGitBeadCommits},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAction_BeadActions(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		{
			name:    "close_bead valid",
			action:  Action{Type: ActionCloseBead, BeadID: "bead-1"},
			wantErr: false,
		},
		{
			name:    "close_bead missing id",
			action:  Action{Type: ActionCloseBead},
			wantErr: true,
		},
		{
			name:    "escalate_ceo valid",
			action:  Action{Type: ActionEscalateCEO, BeadID: "bead-1"},
			wantErr: false,
		},
		{
			name:    "escalate_ceo missing id",
			action:  Action{Type: ActionEscalateCEO},
			wantErr: true,
		},
		{
			name:    "approve_bead valid",
			action:  Action{Type: ActionApproveBead, BeadID: "bead-1"},
			wantErr: false,
		},
		{
			name:    "approve_bead missing id",
			action:  Action{Type: ActionApproveBead},
			wantErr: true,
		},
		{
			name:    "reject_bead valid",
			action:  Action{Type: ActionRejectBead, BeadID: "bead-1", Reason: "quality"},
			wantErr: false,
		},
		{
			name:    "reject_bead missing reason",
			action:  Action{Type: ActionRejectBead, BeadID: "bead-1"},
			wantErr: true,
		},
		{
			name:    "reject_bead missing id",
			action:  Action{Type: ActionRejectBead, Reason: "quality"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// PR review and agent communication action types are validated at the router level,
// not by validateAction (which treats them as unknown). Test that they parse as valid JSON
// and that the router handles them correctly (covered in router_pr_review_test.go and
// router_agent_message_test.go / router_delegate_test.go).

func TestValidateAction_PRActions_UnknownToValidator(t *testing.T) {
	// These action types are handled by the router but not recognized by validateAction
	unknownToValidator := []string{
		ActionFetchPR, ActionReviewCode, ActionAddPRComment, ActionSubmitReview, ActionRequestReview,
		ActionSendAgentMessage, ActionDelegateTask,
	}
	for _, actionType := range unknownToValidator {
		err := validateAction(Action{Type: actionType})
		if err == nil {
			t.Errorf("expected validateAction to reject %s (handled by router instead)", actionType)
		}
	}
}

func TestValidateAction_WorkflowActions(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		{
			name:    "start_dev valid",
			action:  Action{Type: ActionStartDev, Workflow: "epcc"},
			wantErr: false,
		},
		{
			name:    "start_dev missing workflow",
			action:  Action{Type: ActionStartDev},
			wantErr: true,
		},
		{
			name:    "whats_next no fields",
			action:  Action{Type: ActionWhatsNext},
			wantErr: false,
		},
		{
			name:    "proceed_to_phase valid",
			action:  Action{Type: ActionProceedToPhase, TargetPhase: "impl", ReviewState: "performed"},
			wantErr: false,
		},
		{
			name:    "proceed_to_phase missing phase",
			action:  Action{Type: ActionProceedToPhase, ReviewState: "performed"},
			wantErr: true,
		},
		{
			name:    "proceed_to_phase missing review_state",
			action:  Action{Type: ActionProceedToPhase, TargetPhase: "impl"},
			wantErr: true,
		},
		{
			name:    "conduct_review valid",
			action:  Action{Type: ActionConductReview, TargetPhase: "design"},
			wantErr: false,
		},
		{
			name:    "conduct_review missing phase",
			action:  Action{Type: ActionConductReview},
			wantErr: true,
		},
		{
			name:    "resume_workflow no fields",
			action:  Action{Type: ActionResumeWorkflow},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAction_LSPActions(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		{
			name:    "find_references with symbol",
			action:  Action{Type: ActionFindReferences, Path: "f.go", Symbol: "MyFunc"},
			wantErr: false,
		},
		{
			name:    "find_references with position",
			action:  Action{Type: ActionFindReferences, Path: "f.go", Line: 10, Column: 5},
			wantErr: false,
		},
		{
			name:    "find_references missing path",
			action:  Action{Type: ActionFindReferences, Symbol: "MyFunc"},
			wantErr: true,
		},
		{
			name:    "find_references missing symbol and position",
			action:  Action{Type: ActionFindReferences, Path: "f.go"},
			wantErr: true,
		},
		{
			name:    "go_to_definition with symbol",
			action:  Action{Type: ActionGoToDefinition, Path: "f.go", Symbol: "MyType"},
			wantErr: false,
		},
		{
			name:    "go_to_definition missing path",
			action:  Action{Type: ActionGoToDefinition, Symbol: "MyType"},
			wantErr: true,
		},
		{
			name:    "go_to_definition missing symbol and position",
			action:  Action{Type: ActionGoToDefinition, Path: "f.go"},
			wantErr: true,
		},
		{
			name:    "find_implementations with symbol",
			action:  Action{Type: ActionFindImplementations, Path: "f.go", Symbol: "Reader"},
			wantErr: false,
		},
		{
			name:    "find_implementations missing path",
			action:  Action{Type: ActionFindImplementations, Symbol: "Reader"},
			wantErr: true,
		},
		{
			name:    "find_implementations missing symbol and position",
			action:  Action{Type: ActionFindImplementations, Path: "f.go"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAction_RefactoringActions(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		{
			name:    "extract_method valid",
			action:  Action{Type: ActionExtractMethod, Path: "f.go", MethodName: "doWork", StartLine: 10, EndLine: 20},
			wantErr: false,
		},
		{
			name:    "extract_method missing path",
			action:  Action{Type: ActionExtractMethod, MethodName: "doWork", StartLine: 10, EndLine: 20},
			wantErr: true,
		},
		{
			name:    "extract_method missing method_name",
			action:  Action{Type: ActionExtractMethod, Path: "f.go", StartLine: 10, EndLine: 20},
			wantErr: true,
		},
		{
			name:    "extract_method missing lines",
			action:  Action{Type: ActionExtractMethod, Path: "f.go", MethodName: "doWork"},
			wantErr: true,
		},
		{
			name:    "rename_symbol valid",
			action:  Action{Type: ActionRenameSymbol, Path: "f.go", Symbol: "old", NewName: "new"},
			wantErr: false,
		},
		{
			name:    "rename_symbol missing path",
			action:  Action{Type: ActionRenameSymbol, Symbol: "old", NewName: "new"},
			wantErr: true,
		},
		{
			name:    "rename_symbol missing symbol",
			action:  Action{Type: ActionRenameSymbol, Path: "f.go", NewName: "new"},
			wantErr: true,
		},
		{
			name:    "rename_symbol missing new_name",
			action:  Action{Type: ActionRenameSymbol, Path: "f.go", Symbol: "old"},
			wantErr: true,
		},
		{
			name:    "inline_variable valid",
			action:  Action{Type: ActionInlineVariable, Path: "f.go", VariableName: "tmp"},
			wantErr: false,
		},
		{
			name:    "inline_variable missing path",
			action:  Action{Type: ActionInlineVariable, VariableName: "tmp"},
			wantErr: true,
		},
		{
			name:    "inline_variable missing variable_name",
			action:  Action{Type: ActionInlineVariable, Path: "f.go"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAction_FileManagementActions(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		{
			name:    "move_file valid",
			action:  Action{Type: ActionMoveFile, SourcePath: "a.go", TargetPath: "b.go"},
			wantErr: false,
		},
		{
			name:    "move_file missing source",
			action:  Action{Type: ActionMoveFile, TargetPath: "b.go"},
			wantErr: true,
		},
		{
			name:    "move_file missing target",
			action:  Action{Type: ActionMoveFile, SourcePath: "a.go"},
			wantErr: true,
		},
		{
			name:    "delete_file valid",
			action:  Action{Type: ActionDeleteFile, Path: "old.go"},
			wantErr: false,
		},
		{
			name:    "delete_file missing path",
			action:  Action{Type: ActionDeleteFile},
			wantErr: true,
		},
		{
			name:    "rename_file valid",
			action:  Action{Type: ActionRenameFile, SourcePath: "a.go", NewName: "b.go"},
			wantErr: false,
		},
		{
			name:    "rename_file missing source",
			action:  Action{Type: ActionRenameFile, NewName: "b.go"},
			wantErr: true,
		},
		{
			name:    "rename_file missing new_name",
			action:  Action{Type: ActionRenameFile, SourcePath: "a.go"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAction_DebuggingActions(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		{
			name:    "add_log valid",
			action:  Action{Type: ActionAddLog, Path: "f.go", Line: 42, LogMessage: "debug value"},
			wantErr: false,
		},
		{
			name:    "add_log missing path",
			action:  Action{Type: ActionAddLog, Line: 42, LogMessage: "debug"},
			wantErr: true,
		},
		{
			name:    "add_log missing line",
			action:  Action{Type: ActionAddLog, Path: "f.go", LogMessage: "debug"},
			wantErr: true,
		},
		{
			name:    "add_log missing message",
			action:  Action{Type: ActionAddLog, Path: "f.go", Line: 42},
			wantErr: true,
		},
		{
			name:    "add_breakpoint valid",
			action:  Action{Type: ActionAddBreakpoint, Path: "f.go", Line: 42},
			wantErr: false,
		},
		{
			name:    "add_breakpoint missing path",
			action:  Action{Type: ActionAddBreakpoint, Line: 42},
			wantErr: true,
		},
		{
			name:    "add_breakpoint missing line",
			action:  Action{Type: ActionAddBreakpoint, Path: "f.go"},
			wantErr: true,
		},
		{
			name:    "generate_docs valid",
			action:  Action{Type: ActionGenerateDocs, Path: "f.go"},
			wantErr: false,
		},
		{
			name:    "generate_docs missing path",
			action:  Action{Type: ActionGenerateDocs},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAction_CoreActions(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		{
			name:    "ask_followup valid",
			action:  Action{Type: ActionAskFollowup, Question: "What next?"},
			wantErr: false,
		},
		{
			name:    "ask_followup missing question",
			action:  Action{Type: ActionAskFollowup},
			wantErr: true,
		},
		{
			name:    "read_code valid",
			action:  Action{Type: ActionReadCode, Path: "f.go"},
			wantErr: false,
		},
		{
			name:    "read_code missing path",
			action:  Action{Type: ActionReadCode},
			wantErr: true,
		},
		{
			name:    "edit_code valid",
			action:  Action{Type: ActionEditCode, Path: "f.go", Patch: "diff"},
			wantErr: false,
		},
		{
			name:    "edit_code missing path",
			action:  Action{Type: ActionEditCode, Patch: "diff"},
			wantErr: true,
		},
		{
			name:    "edit_code missing patch",
			action:  Action{Type: ActionEditCode, Path: "f.go"},
			wantErr: true,
		},
		{
			name:    "write_file valid",
			action:  Action{Type: ActionWriteFile, Path: "f.go", Content: "pkg"},
			wantErr: false,
		},
		{
			name:    "write_file missing path",
			action:  Action{Type: ActionWriteFile, Content: "pkg"},
			wantErr: true,
		},
		{
			name:    "write_file missing content",
			action:  Action{Type: ActionWriteFile, Path: "f.go"},
			wantErr: true,
		},
		{
			name:    "read_file valid",
			action:  Action{Type: ActionReadFile, Path: "f.go"},
			wantErr: false,
		},
		{
			name:    "read_file missing path",
			action:  Action{Type: ActionReadFile},
			wantErr: true,
		},
		{
			name:    "read_tree valid",
			action:  Action{Type: ActionReadTree, Path: "."},
			wantErr: false,
		},
		{
			name:    "read_tree missing path",
			action:  Action{Type: ActionReadTree},
			wantErr: true,
		},
		{
			name:    "search_text valid",
			action:  Action{Type: ActionSearchText, Query: "TODO"},
			wantErr: false,
		},
		{
			name:    "search_text missing query",
			action:  Action{Type: ActionSearchText},
			wantErr: true,
		},
		{
			name:    "apply_patch valid",
			action:  Action{Type: ActionApplyPatch, Patch: "diff"},
			wantErr: false,
		},
		{
			name:    "apply_patch missing patch",
			action:  Action{Type: ActionApplyPatch},
			wantErr: true,
		},
		{
			name:    "run_command valid",
			action:  Action{Type: ActionRunCommand, Command: "ls"},
			wantErr: false,
		},
		{
			name:    "run_command missing command",
			action:  Action{Type: ActionRunCommand},
			wantErr: true,
		},
		{
			name:    "done no fields",
			action:  Action{Type: ActionDone},
			wantErr: false,
		},
		{
			name:    "git_status no fields",
			action:  Action{Type: ActionGitStatus},
			wantErr: false,
		},
		{
			name:    "git_diff no fields",
			action:  Action{Type: ActionGitDiff},
			wantErr: false,
		},
		{
			name:    "git_commit no fields",
			action:  Action{Type: ActionGitCommit},
			wantErr: false,
		},
		{
			name:    "git_push no fields",
			action:  Action{Type: ActionGitPush},
			wantErr: false,
		},
		{
			name:    "create_pr no fields",
			action:  Action{Type: ActionCreatePR},
			wantErr: false,
		},
		{
			name:    "run_tests no fields",
			action:  Action{Type: ActionRunTests},
			wantErr: false,
		},
		{
			name:    "run_linter no fields",
			action:  Action{Type: ActionRunLinter},
			wantErr: false,
		},
		{
			name:    "build_project no fields",
			action:  Action{Type: ActionBuildProject},
			wantErr: false,
		},
		{
			name:    "create_bead valid",
			action:  Action{Type: ActionCreateBead, Bead: &BeadPayload{Title: "task", ProjectID: "p1"}},
			wantErr: false,
		},
		{
			name:    "create_bead nil payload",
			action:  Action{Type: ActionCreateBead},
			wantErr: true,
		},
		{
			name:    "create_bead missing title",
			action:  Action{Type: ActionCreateBead, Bead: &BeadPayload{ProjectID: "p1"}},
			wantErr: true,
		},
		{
			name:    "create_bead missing project_id",
			action:  Action{Type: ActionCreateBead, Bead: &BeadPayload{Title: "task"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_MultipleActions(t *testing.T) {
	env := &ActionEnvelope{
		Actions: []Action{
			{Type: ActionDone},
			{Type: ActionReadFile, Path: "f.go"},
		},
	}
	err := Validate(env)
	if err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestValidate_SecondActionInvalid(t *testing.T) {
	env := &ActionEnvelope{
		Actions: []Action{
			{Type: ActionDone},
			{Type: ActionReadFile}, // missing path
		},
	}
	err := Validate(env)
	if err == nil {
		t.Error("expected error for invalid second action")
	}
	if !strings.Contains(err.Error(), "action[1]") {
		t.Errorf("expected action index in error, got %v", err)
	}
}

func TestDecodeLenient_StrictPassthroughWithValidation(t *testing.T) {
	// Strict-decodable but with validation error
	payload := `{"actions": [{"type": "read_file"}]}`
	_, err := DecodeLenient([]byte(payload))
	if err == nil {
		t.Fatal("expected validation error for missing path")
	}
}

func TestExtractJSONObject_UnbalancedBraces(t *testing.T) {
	_, err := extractJSONObject([]byte("{ unclosed"))
	if err == nil {
		t.Fatal("expected error for unbalanced braces")
	}
}

func TestExtractJSONObject_ClosingBraceBeforeOpening(t *testing.T) {
	// A closing brace before any opening brace should be skipped
	result, err := extractJSONObject([]byte("} {\"key\": \"val\"}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(result), "key") {
		t.Error("expected valid JSON object extracted")
	}
}
