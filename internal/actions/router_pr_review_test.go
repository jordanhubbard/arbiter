package actions

import (
	"context"
	"testing"

	"github.com/jordanhubbard/loom/internal/executor"
)

func TestHandleFetchPR_NoPRNumber(t *testing.T) {
	r := &Router{}
	result := r.handleFetchPR(context.Background(), Action{Type: ActionFetchPR, PRNumber: 0}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
	if !containsStr(result.Message, "pr_number") {
		t.Errorf("expected pr_number error, got %s", result.Message)
	}
}

func TestHandleFetchPR_NoCommands(t *testing.T) {
	r := &Router{}
	result := r.handleFetchPR(context.Background(), Action{Type: ActionFetchPR, PRNumber: 42}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestHandleFetchPR_Success(t *testing.T) {
	cmd := &mockCommandExecutor{
		result: &executor.ExecuteCommandResult{
			Success: true,
			Stdout:  `{"number": 42, "title": "Test PR", "state": "open"}`,
		},
	}
	r := &Router{Commands: cmd}
	result := r.handleFetchPR(context.Background(), Action{Type: ActionFetchPR, PRNumber: 42}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestHandleFetchPR_WithDiff(t *testing.T) {
	callCount := 0
	cmd := &mockCommandExecutor{}
	cmd.result = &executor.ExecuteCommandResult{
		Success: true,
		Stdout:  `{"number": 42, "title": "Test PR"}`,
	}
	// Override with a function-like approach: first call returns PR data, second returns diff
	origExec := cmd
	_ = origExec
	r := &Router{Commands: &mockCommandExecutorFunc{
		fn: func(ctx context.Context, req executor.ExecuteCommandRequest) (*executor.ExecuteCommandResult, error) {
			callCount++
			if callCount == 1 {
				return &executor.ExecuteCommandResult{
					Success: true,
					Stdout:  `{"number": 42, "title": "Test PR"}`,
				}, nil
			}
			return &executor.ExecuteCommandResult{
				Success: true,
				Stdout:  "+added line\n-removed line",
			}, nil
		},
	}}
	result := r.handleFetchPR(context.Background(), Action{
		Type:        ActionFetchPR,
		PRNumber:    42,
		IncludeDiff: true,
	}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestHandleReviewCode_NoPRNumber(t *testing.T) {
	r := &Router{}
	result := r.handleReviewCode(context.Background(), Action{Type: ActionReviewCode, PRNumber: 0}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestHandleReviewCode_Success(t *testing.T) {
	cmd := &mockCommandExecutorFunc{
		fn: func(ctx context.Context, req executor.ExecuteCommandRequest) (*executor.ExecuteCommandResult, error) {
			return &executor.ExecuteCommandResult{
				Success: true,
				Stdout:  `{"number": 42, "title": "Test PR", "files": []}`,
			}, nil
		},
	}
	r := &Router{Commands: cmd}
	result := r.handleReviewCode(context.Background(), Action{
		Type:           ActionReviewCode,
		PRNumber:       42,
		ReviewCriteria: []string{"quality", "security"},
	}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestHandleAddPRComment_NoPRNumber(t *testing.T) {
	r := &Router{}
	result := r.handleAddPRComment(context.Background(), Action{Type: ActionAddPRComment}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestHandleAddPRComment_NoBody(t *testing.T) {
	r := &Router{}
	result := r.handleAddPRComment(context.Background(), Action{Type: ActionAddPRComment, PRNumber: 42}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestHandleAddPRComment_General(t *testing.T) {
	cmd := &mockCommandExecutor{
		result: &executor.ExecuteCommandResult{Success: true, Stdout: "comment added"},
	}
	r := &Router{Commands: cmd}
	result := r.handleAddPRComment(context.Background(), Action{
		Type:        ActionAddPRComment,
		PRNumber:    42,
		CommentBody: "LGTM",
	}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
	if !containsStr(result.Message, "general") {
		t.Errorf("expected general comment type, got %s", result.Message)
	}
}

func TestHandleAddPRComment_Inline(t *testing.T) {
	cmd := &mockCommandExecutor{
		result: &executor.ExecuteCommandResult{Success: true, Stdout: ""},
	}
	r := &Router{Commands: cmd}
	result := r.handleAddPRComment(context.Background(), Action{
		Type:        ActionAddPRComment,
		PRNumber:    42,
		CommentBody: "Issue here",
		CommentPath: "foo.go",
		CommentLine: 10,
	}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
	if !containsStr(result.Message, "inline") {
		t.Errorf("expected inline comment type, got %s", result.Message)
	}
}

func TestHandleSubmitReview_NoPRNumber(t *testing.T) {
	r := &Router{}
	result := r.handleSubmitReview(context.Background(), Action{Type: ActionSubmitReview}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestHandleSubmitReview_NoEvent(t *testing.T) {
	r := &Router{}
	result := r.handleSubmitReview(context.Background(), Action{Type: ActionSubmitReview, PRNumber: 42}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error for missing event, got %s", result.Status)
	}
}

func TestHandleSubmitReview_NoBody(t *testing.T) {
	r := &Router{}
	result := r.handleSubmitReview(context.Background(), Action{Type: ActionSubmitReview, PRNumber: 42, ReviewEvent: "APPROVE"}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error for missing body, got %s", result.Status)
	}
}

func TestHandleSubmitReview_InvalidEvent(t *testing.T) {
	r := &Router{Commands: &mockCommandExecutor{}}
	result := r.handleSubmitReview(context.Background(), Action{
		Type:        ActionSubmitReview,
		PRNumber:    42,
		ReviewEvent: "INVALID",
		CommentBody: "text",
	}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error for invalid event, got %s", result.Status)
	}
}

func TestHandleSubmitReview_Success(t *testing.T) {
	cmd := &mockCommandExecutor{
		result: &executor.ExecuteCommandResult{Success: true, Stdout: "reviewed"},
	}
	r := &Router{Commands: cmd}
	result := r.handleSubmitReview(context.Background(), Action{
		Type:        ActionSubmitReview,
		PRNumber:    42,
		ReviewEvent: "APPROVE",
		CommentBody: "LGTM",
	}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

func TestHandleRequestReview_NoPRNumber(t *testing.T) {
	r := &Router{}
	result := r.handleRequestReview(context.Background(), Action{Type: ActionRequestReview}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error, got %s", result.Status)
	}
}

func TestHandleRequestReview_NoReviewer(t *testing.T) {
	r := &Router{}
	result := r.handleRequestReview(context.Background(), Action{Type: ActionRequestReview, PRNumber: 42}, ActionContext{})
	if result.Status != "error" {
		t.Errorf("expected error for missing reviewer, got %s", result.Status)
	}
}

func TestHandleRequestReview_Success(t *testing.T) {
	cmd := &mockCommandExecutor{
		result: &executor.ExecuteCommandResult{Success: true, Stdout: "requested"},
	}
	r := &Router{Commands: cmd}
	result := r.handleRequestReview(context.Background(), Action{
		Type:     ActionRequestReview,
		PRNumber: 42,
		Reviewer: "user1",
	}, ActionContext{})
	if result.Status != "executed" {
		t.Errorf("expected executed, got %s: %s", result.Status, result.Message)
	}
}

// mockCommandExecutorFunc allows function-based mocking for multi-call scenarios
type mockCommandExecutorFunc struct {
	fn func(ctx context.Context, req executor.ExecuteCommandRequest) (*executor.ExecuteCommandResult, error)
}

func (m *mockCommandExecutorFunc) ExecuteCommand(ctx context.Context, req executor.ExecuteCommandRequest) (*executor.ExecuteCommandResult, error) {
	return m.fn(ctx, req)
}
