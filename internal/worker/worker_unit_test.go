package worker

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/actions"
	"github.com/jordanhubbard/loom/internal/provider"
	"github.com/jordanhubbard/loom/pkg/models"
)

func makeTestWorker(persona *models.Persona) *Worker {
	agent := &models.Agent{
		ID:          "agent-1",
		Name:        "Test Agent",
		PersonaName: "tester",
		Persona:     persona,
	}
	rp := &provider.RegisteredProvider{
		Config: &provider.ProviderConfig{
			ID:            "prov-1",
			Name:          "Mock",
			Model:         "mock-model",
			ContextWindow: 32768,
		},
	}
	return NewWorker("worker-1", agent, rp)
}

func TestNewWorker(t *testing.T) {
	w := makeTestWorker(nil)
	if w.id != "worker-1" {
		t.Errorf("id = %q, want worker-1", w.id)
	}
	if w.status != WorkerStatusIdle {
		t.Errorf("status = %q, want idle", w.status)
	}
	if w.agent.ID != "agent-1" {
		t.Errorf("agent.ID = %q, want agent-1", w.agent.ID)
	}
}

func TestWorker_StartStop(t *testing.T) {
	w := makeTestWorker(nil)

	if err := w.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if w.GetStatus() != WorkerStatusIdle {
		t.Errorf("status after start = %q, want idle", w.GetStatus())
	}

	w.Stop()
	if w.GetStatus() != WorkerStatusStopped {
		t.Errorf("status after stop = %q, want stopped", w.GetStatus())
	}
}

func TestWorker_GetInfo(t *testing.T) {
	w := makeTestWorker(&models.Persona{Character: "A tester"})
	info := w.GetInfo()

	if info.ID != "worker-1" {
		t.Errorf("info.ID = %q", info.ID)
	}
	if info.AgentName != "Test Agent" {
		t.Errorf("info.AgentName = %q", info.AgentName)
	}
	if info.PersonaName != "tester" {
		t.Errorf("info.PersonaName = %q", info.PersonaName)
	}
	if info.ProviderID != "prov-1" {
		t.Errorf("info.ProviderID = %q", info.ProviderID)
	}
	if info.Status != WorkerStatusIdle {
		t.Errorf("info.Status = %q", info.Status)
	}
	if info.StartedAt.IsZero() {
		t.Error("info.StartedAt is zero")
	}
}

func TestWorker_buildSystemPrompt_NilPersona(t *testing.T) {
	w := makeTestWorker(nil)
	prompt := w.buildSystemPrompt()

	if !strings.Contains(prompt, "Test Agent") {
		t.Error("prompt should contain agent name when no persona")
	}
	if !strings.Contains(prompt, "Your Role") {
		t.Error("prompt should contain role section")
	}
}

func TestWorker_buildSystemPrompt_WithPersona(t *testing.T) {
	w := makeTestWorker(&models.Persona{
		Character: "A skilled Go developer",
		Mission:   "Write clean code",
	})
	prompt := w.buildSystemPrompt()

	if !strings.Contains(prompt, "A skilled Go developer") {
		t.Error("prompt should contain character")
	}
	if !strings.Contains(prompt, "Write clean code") {
		t.Error("prompt should contain mission")
	}
}

func TestWorker_buildSystemPrompt_PersonaNoCharacter(t *testing.T) {
	w := makeTestWorker(&models.Persona{
		Mission: "Help with tasks",
	})
	prompt := w.buildSystemPrompt()

	if !strings.Contains(prompt, "Test Agent") {
		t.Error("should fall back to agent name when no character")
	}
	if !strings.Contains(prompt, "Help with tasks") {
		t.Error("prompt should contain mission")
	}
}

func TestWorker_buildSingleShotMessages(t *testing.T) {
	w := makeTestWorker(nil)

	t.Run("without context", func(t *testing.T) {
		task := &Task{ID: "t1", Description: "Do something"}
		msgs := w.buildSingleShotMessages(task)
		if len(msgs) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(msgs))
		}
		if msgs[0].Role != "system" {
			t.Error("first message should be system")
		}
		if msgs[1].Role != "user" {
			t.Error("second message should be user")
		}
		if msgs[1].Content != "Do something" {
			t.Errorf("user content = %q", msgs[1].Content)
		}
	})

	t.Run("with context", func(t *testing.T) {
		task := &Task{ID: "t2", Description: "Do something", Context: "extra info"}
		msgs := w.buildSingleShotMessages(task)
		if !strings.Contains(msgs[1].Content, "extra info") {
			t.Error("user message should include context")
		}
		if !strings.Contains(msgs[1].Content, "Context:") {
			t.Error("should have context separator")
		}
	})
}

func TestWorker_messageExists(t *testing.T) {
	w := makeTestWorker(nil)

	msgs := []models.ChatMessage{
		{Role: "system", Content: "sys prompt"},
		{Role: "user", Content: "hello"},
	}

	if !w.messageExists(msgs, "hello") {
		t.Error("should find existing message")
	}
	if w.messageExists(msgs, "world") {
		t.Error("should not find non-existing message")
	}
	if w.messageExists(nil, "anything") {
		t.Error("should handle nil messages")
	}
}

func TestWorker_getModelTokenLimit(t *testing.T) {
	t.Run("with context window", func(t *testing.T) {
		w := makeTestWorker(nil)
		if w.getModelTokenLimit() != 32768 {
			t.Errorf("expected 32768, got %d", w.getModelTokenLimit())
		}
	})

	t.Run("default", func(t *testing.T) {
		w := makeTestWorker(nil)
		w.provider.Config.ContextWindow = 0
		if w.getModelTokenLimit() != 32768 {
			t.Errorf("expected default 32768, got %d", w.getModelTokenLimit())
		}
	})
}

// --- Pure function tests ---

func TestIsConversationalResponse(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"What would you like me to do?", true},
		{"Shall I proceed with the implementation?", true},
		{"Would you like me to continue?", true},
		{"Let me know if you need anything else.", true},
		{"Please let me know how to proceed.", true},
		{"Do you want me to fix this?", true},
		{"How should I proceed with this?", true},
		{"Awaiting your instructions.", true},
		{"Could you clarify the requirements?", true},
		{`{"action": "done", "reason": "task completed"}`, false},
		{`{"actions": [{"type": "read_code", "path": "main.go"}]}`, false},
		{"The file contains 50 lines of Go code.", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isConversationalResponse(tt.input)
		if got != tt.want {
			t.Errorf("isConversationalResponse(%q) = %v, want %v", tt.input[:min(len(tt.input), 40)], got, tt.want)
		}
	}
}

func TestCheckTerminalCondition(t *testing.T) {
	tests := []struct {
		name    string
		env     *actions.ActionEnvelope
		results []actions.Result
		want    string
	}{
		{
			name: "close_bead success",
			env:  &actions.ActionEnvelope{Actions: []actions.Action{{Type: actions.ActionCloseBead}}},
			results: []actions.Result{{ActionType: actions.ActionCloseBead, Status: "executed"}},
			want: "completed",
		},
		{
			name: "close_bead failed",
			env:  &actions.ActionEnvelope{Actions: []actions.Action{{Type: actions.ActionCloseBead}}},
			results: []actions.Result{{ActionType: actions.ActionCloseBead, Status: "error"}},
			want: "",
		},
		{
			name:    "done action",
			env:     &actions.ActionEnvelope{Actions: []actions.Action{{Type: actions.ActionDone}}},
			results: []actions.Result{{ActionType: actions.ActionDone, Status: "executed"}},
			want:    "completed",
		},
		{
			name:    "escalate action",
			env:     &actions.ActionEnvelope{Actions: []actions.Action{{Type: actions.ActionEscalateCEO}}},
			results: []actions.Result{{ActionType: actions.ActionEscalateCEO, Status: "executed"}},
			want:    "escalated",
		},
		{
			name:    "non-terminal action",
			env:     &actions.ActionEnvelope{Actions: []actions.Action{{Type: actions.ActionReadCode}}},
			results: []actions.Result{{ActionType: actions.ActionReadCode, Status: "executed"}},
			want:    "",
		},
		{
			name:    "empty actions",
			env:     &actions.ActionEnvelope{Actions: []actions.Action{}},
			results: []actions.Result{},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkTerminalCondition(tt.env, tt.results)
			if got != tt.want {
				t.Errorf("checkTerminalCondition() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateForLesson(t *testing.T) {
	short := "short string"
	if truncateForLesson(short) != short {
		t.Error("short strings should not be truncated")
	}

	long := strings.Repeat("x", 600)
	result := truncateForLesson(long)
	if len(result) != 500 {
		t.Errorf("expected 500 chars, got %d", len(result))
	}

	if truncateForLesson("") != "" {
		t.Error("empty string should stay empty")
	}

	exact := strings.Repeat("a", 500)
	if truncateForLesson(exact) != exact {
		t.Error("500 char string should not be truncated")
	}
}

func TestTruncateMessages(t *testing.T) {
	t.Run("2 or fewer messages pass through", func(t *testing.T) {
		msgs := []provider.ChatMessage{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hi"},
		}
		result := truncateMessages(msgs, 0.5)
		if len(result) != 2 {
			t.Errorf("expected 2, got %d", len(result))
		}
	})

	t.Run("single message", func(t *testing.T) {
		msgs := []provider.ChatMessage{{Role: "system", Content: "sys"}}
		result := truncateMessages(msgs, 0.5)
		if len(result) != 1 {
			t.Errorf("expected 1, got %d", len(result))
		}
	})

	t.Run("keeps fraction of middle messages", func(t *testing.T) {
		msgs := []provider.ChatMessage{
			{Role: "system", Content: "system prompt"},
			{Role: "user", Content: "msg1"},
			{Role: "assistant", Content: "resp1"},
			{Role: "user", Content: "msg2"},
			{Role: "assistant", Content: "resp2"},
			{Role: "user", Content: "msg3"},
			{Role: "assistant", Content: "resp3"},
			{Role: "user", Content: "final question"},
		}

		result := truncateMessages(msgs, 0.5)
		// System + notice + kept middle + last
		if result[0].Role != "system" {
			t.Error("first should be system")
		}
		if result[len(result)-1].Content != "final question" {
			t.Error("last should be the final message")
		}
	})

	t.Run("fraction 0 drops all middle", func(t *testing.T) {
		msgs := []provider.ChatMessage{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "old"},
			{Role: "assistant", Content: "old resp"},
			{Role: "user", Content: "latest"},
		}
		result := truncateMessages(msgs, 0.0)
		// system + notice + latest
		if len(result) != 3 {
			t.Errorf("expected 3, got %d", len(result))
		}
		if result[0].Role != "system" {
			t.Error("first should be system")
		}
		if !strings.Contains(result[1].Content, "dropped") {
			t.Error("second should be drop notice")
		}
		if result[2].Content != "latest" {
			t.Error("last should be latest message")
		}
	})
}

func TestHashActions(t *testing.T) {
	acts1 := []actions.Action{
		{Type: actions.ActionReadCode, Path: "main.go"},
		{Type: actions.ActionWriteFile, Path: "out.go"},
	}
	acts2 := []actions.Action{
		{Type: actions.ActionReadCode, Path: "main.go"},
		{Type: actions.ActionWriteFile, Path: "out.go"},
	}
	acts3 := []actions.Action{
		{Type: actions.ActionReadCode, Path: "other.go"},
	}

	h1 := hashActions(acts1)
	h2 := hashActions(acts2)
	h3 := hashActions(acts3)

	if h1 != h2 {
		t.Error("same actions should produce same hash")
	}
	if h1 == h3 {
		t.Error("different actions should produce different hash")
	}
	if len(h1) != 16 { // hex of 8 bytes
		t.Errorf("hash length = %d, want 16", len(h1))
	}

	// Empty actions
	empty := hashActions(nil)
	if empty == "" {
		t.Error("empty actions should still produce a hash")
	}
}

func TestFlattenActionLog(t *testing.T) {
	log := []ActionLogEntry{
		{
			Iteration: 1,
			Results: []actions.Result{
				{ActionType: actions.ActionReadCode, Status: "executed", Message: "ok", Metadata: map[string]interface{}{"path": "main.go"}},
				{ActionType: actions.ActionWriteFile, Status: "executed", Message: "written"},
			},
		},
		{
			Iteration: 2,
			Results: []actions.Result{
				{ActionType: actions.ActionBuildProject, Status: "error", Message: "build failed"},
			},
		},
	}

	entries := flattenActionLog(log)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].ActionType != string(actions.ActionReadCode) {
		t.Errorf("entry[0].ActionType = %q", entries[0].ActionType)
	}
	if entries[0].Path != "main.go" {
		t.Errorf("entry[0].Path = %q", entries[0].Path)
	}
	if entries[0].Iteration != 1 {
		t.Errorf("entry[0].Iteration = %d", entries[0].Iteration)
	}

	// Entry without path metadata
	if entries[1].Path != "" {
		t.Errorf("entry without path metadata should have empty path, got %q", entries[1].Path)
	}

	if entries[2].Status != "error" {
		t.Errorf("entry[2].Status = %q", entries[2].Status)
	}
}

func TestFlattenActionLog_Empty(t *testing.T) {
	entries := flattenActionLog(nil)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- Tests that need a mock provider (for ExecuteTask paths) ---

func TestWorker_ExecuteTask_NotIdle(t *testing.T) {
	mockProv := &MockConversationProvider{responseContent: "ok", tokenCount: 5}
	rp := &provider.RegisteredProvider{
		Config:   &provider.ProviderConfig{ID: "p1", Name: "P", Model: "m"},
		Protocol: mockProv,
	}
	agent := &models.Agent{ID: "a1", Name: "A"}
	w := NewWorker("w1", agent, rp)
	_ = w.Start()

	// Manually set to working
	w.mu.Lock()
	w.status = WorkerStatusWorking
	w.mu.Unlock()

	_, err := w.ExecuteTask(t.Context(), &Task{ID: "t1", Description: "test"})
	if err == nil {
		t.Error("expected error for non-idle worker")
	}
	if !strings.Contains(err.Error(), "not idle") {
		t.Errorf("error = %q, want 'not idle'", err.Error())
	}

	// Reset for cleanup
	w.mu.Lock()
	w.status = WorkerStatusIdle
	w.mu.Unlock()
}

func TestWorker_ExecuteTask_WithConversationSession(t *testing.T) {
	mockProv := &MockConversationProvider{responseContent: `{"result": "done"}`, tokenCount: 10}
	rp := &provider.RegisteredProvider{
		Config:   &provider.ProviderConfig{ID: "p1", Name: "P", Model: "m"},
		Protocol: mockProv,
	}
	agent := &models.Agent{ID: "a1", Name: "Agent", Persona: &models.Persona{Character: "helper"}}
	w := NewWorker("w1", agent, rp)
	_ = w.Start()

	convCtx := models.NewConversationContext("sess-1", "bead-1", "proj-1", 24*3600*1e9)
	task := &Task{
		ID:                  "t1",
		Description:         "do task",
		ConversationSession: convCtx,
	}

	result, err := w.ExecuteTask(t.Context(), task)
	if err != nil {
		t.Fatalf("ExecuteTask error = %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Response != `{"result": "done"}` {
		t.Errorf("response = %q", result.Response)
	}

	// Conversation context gets system message added during buildConversationMessages.
	// Without db, the assistant response is NOT written back to the context.
	if len(convCtx.Messages) < 1 {
		t.Errorf("expected at least 1 message in conversation (system), got %d", len(convCtx.Messages))
	}
}

func TestWorker_buildEnhancedSystemPrompt(t *testing.T) {
	t.Run("nil persona", func(t *testing.T) {
		w := makeTestWorker(nil)
		prompt := w.buildEnhancedSystemPrompt(nil, "proj-1", "")
		if !strings.Contains(prompt, "Test Agent") {
			t.Error("should contain agent name")
		}
	})

	t.Run("with persona", func(t *testing.T) {
		w := makeTestWorker(&models.Persona{
			Character: "Expert coder",
			Mission:   "Ship fast",
		})
		prompt := w.buildEnhancedSystemPrompt(nil, "proj-1", "")
		if !strings.Contains(prompt, "Expert coder") {
			t.Error("should contain character")
		}
		if !strings.Contains(prompt, "Ship fast") {
			t.Error("should contain mission")
		}
	})

	t.Run("text mode", func(t *testing.T) {
		w := makeTestWorker(nil)
		w.textMode = true
		prompt := w.buildEnhancedSystemPrompt(nil, "proj-1", "some progress")
		if prompt == "" {
			t.Error("prompt should not be empty")
		}
	})

	t.Run("with lessons provider", func(t *testing.T) {
		w := makeTestWorker(nil)
		lp := &mockLessonsProvider{lessonsText: "Lesson: always run tests"}
		prompt := w.buildEnhancedSystemPrompt(lp, "proj-1", "building feature")
		_ = prompt // Just verify it doesn't panic
	})
}

// mockLessonsProvider implements LessonsProvider for testing
type mockLessonsProvider struct {
	lessonsText string
}

func (m *mockLessonsProvider) GetLessonsForPrompt(projectID string) string {
	return m.lessonsText
}

func (m *mockLessonsProvider) GetRelevantLessons(projectID, taskContext string, topK int) string {
	return m.lessonsText
}

func (m *mockLessonsProvider) RecordLesson(projectID, category, title, detail, beadID, agentID string) error {
	return nil
}

func TestWorker_recordBuildLessons(t *testing.T) {
	w := makeTestWorker(nil)
	lp := &mockLessonsProvider{}

	config := &LoopConfig{
		LessonsProvider: lp,
		ActionContext:   actions.ActionContext{ProjectID: "p1", BeadID: "b1"},
	}

	// Build failure
	env := &actions.ActionEnvelope{Actions: []actions.Action{{Type: actions.ActionBuildProject}}}
	results := []actions.Result{{ActionType: actions.ActionBuildProject, Status: "error", Message: "compile error"}}
	w.recordBuildLessons(config, env, results)

	// Test failure
	env2 := &actions.ActionEnvelope{Actions: []actions.Action{{Type: actions.ActionRunTests}}}
	results2 := []actions.Result{{ActionType: actions.ActionRunTests, Status: "error", Message: "test failed"}}
	w.recordBuildLessons(config, env2, results2)

	// Edit failure
	env3 := &actions.ActionEnvelope{Actions: []actions.Action{{Type: actions.ActionEditCode}}}
	results3 := []actions.Result{{ActionType: actions.ActionEditCode, Status: "error", Message: "patch failed"}}
	w.recordBuildLessons(config, env3, results3)

	// No lessons provider - should not panic
	config2 := &LoopConfig{LessonsProvider: nil}
	w.recordBuildLessons(config2, env, results)
}

func TestWorker_SetDatabase(t *testing.T) {
	w := makeTestWorker(nil)
	if w.db != nil {
		t.Error("db should be nil initially")
	}
	w.SetDatabase(nil)
	if w.db != nil {
		t.Error("db should still be nil")
	}
}

// --- ExecuteTaskWithLoop tests ---

// sequenceMockProvider returns different responses on successive calls
type sequenceMockProvider struct {
	responses []string
	callCount int
}

func (m *sequenceMockProvider) CreateChatCompletion(ctx context.Context, req *provider.ChatCompletionRequest) (*provider.ChatCompletionResponse, error) {
	idx := m.callCount
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	}
	m.callCount++
	return &provider.ChatCompletionResponse{
		ID: "resp",
		Choices: []struct {
			Index   int                  `json:"index"`
			Message provider.ChatMessage `json:"message"`
			Finish  string               `json:"finish_reason"`
		}{
			{Index: 0, Message: provider.ChatMessage{Role: "assistant", Content: m.responses[idx]}, Finish: "stop"},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{PromptTokens: 50, CompletionTokens: 20, TotalTokens: 70},
	}, nil
}

func (m *sequenceMockProvider) GetModels(ctx context.Context) ([]provider.Model, error) {
	return nil, nil
}

func TestWorker_ExecuteTaskWithLoop_DoneAction(t *testing.T) {
	mock := &sequenceMockProvider{
		responses: []string{`{"action": "done", "reason": "task completed"}`},
	}
	rp := &provider.RegisteredProvider{
		Config:   &provider.ProviderConfig{ID: "p1", Name: "P", Model: "m"},
		Protocol: mock,
	}
	agent := &models.Agent{ID: "a1", Name: "Agent"}
	w := NewWorker("w1", agent, rp)
	_ = w.Start()

	task := &Task{ID: "t1", Description: "do something"}
	config := &LoopConfig{
		MaxIterations: 5,
		Router:        &actions.Router{},
		ActionContext: actions.ActionContext{ProjectID: "p1", BeadID: "b1"},
		TextMode:      true,
	}

	result, err := w.ExecuteTaskWithLoop(context.Background(), task, config)
	if err != nil {
		t.Fatalf("ExecuteTaskWithLoop error = %v", err)
	}
	if result.TerminalReason != "completed" {
		t.Errorf("TerminalReason = %q, want completed", result.TerminalReason)
	}
	if result.Iterations != 1 {
		t.Errorf("Iterations = %d, want 1", result.Iterations)
	}
}

func TestWorker_ExecuteTaskWithLoop_ParseFailure(t *testing.T) {
	mock := &sequenceMockProvider{
		responses: []string{
			"This is not valid JSON at all",
			"Still not valid JSON",
		},
	}
	rp := &provider.RegisteredProvider{
		Config:   &provider.ProviderConfig{ID: "p1", Name: "P", Model: "m"},
		Protocol: mock,
	}
	agent := &models.Agent{ID: "a1", Name: "Agent"}
	w := NewWorker("w1", agent, rp)
	_ = w.Start()

	task := &Task{ID: "t1", Description: "do something"}
	config := &LoopConfig{
		MaxIterations: 10,
		Router:        &actions.Router{},
		ActionContext: actions.ActionContext{ProjectID: "p1", BeadID: "b1"},
		TextMode:      true,
	}

	result, err := w.ExecuteTaskWithLoop(context.Background(), task, config)
	if err != nil {
		t.Fatalf("ExecuteTaskWithLoop error = %v", err)
	}
	if result.TerminalReason != "parse_failures" {
		t.Errorf("TerminalReason = %q, want parse_failures", result.TerminalReason)
	}
}

func TestWorker_ExecuteTaskWithLoop_EmptyActions(t *testing.T) {
	mock := &sequenceMockProvider{
		responses: []string{`{"actions": []}`},
	}
	rp := &provider.RegisteredProvider{
		Config:   &provider.ProviderConfig{ID: "p1", Name: "P", Model: "m"},
		Protocol: mock,
	}
	agent := &models.Agent{ID: "a1", Name: "Agent"}
	w := NewWorker("w1", agent, rp)
	_ = w.Start()

	task := &Task{ID: "t1", Description: "do something"}
	config := &LoopConfig{
		MaxIterations: 5,
		Router:        &actions.Router{},
		ActionContext: actions.ActionContext{},
		TextMode:      false, // Legacy parser handles {"actions": []} as empty
	}

	result, err := w.ExecuteTaskWithLoop(context.Background(), task, config)
	if err != nil {
		t.Fatalf("ExecuteTaskWithLoop error = %v", err)
	}
	// Empty actions array triggers validation failures (parsers require at least one action)
	if result.TerminalReason != "validation_failures" {
		t.Errorf("TerminalReason = %q, want validation_failures", result.TerminalReason)
	}
}

func TestWorker_ExecuteTaskWithLoop_NotIdle(t *testing.T) {
	mock := &sequenceMockProvider{responses: []string{`{"actions":[]}`}}
	rp := &provider.RegisteredProvider{
		Config:   &provider.ProviderConfig{ID: "p1", Name: "P", Model: "m"},
		Protocol: mock,
	}
	agent := &models.Agent{ID: "a1", Name: "Agent"}
	w := NewWorker("w1", agent, rp)
	w.status = WorkerStatusWorking

	task := &Task{ID: "t1", Description: "do something"}
	config := &LoopConfig{MaxIterations: 5, Router: &actions.Router{}}

	_, err := w.ExecuteTaskWithLoop(context.Background(), task, config)
	if err == nil {
		t.Error("expected error for non-idle worker")
	}
}

func TestWorker_ExecuteTaskWithLoop_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Already canceled

	mock := &sequenceMockProvider{responses: []string{`{"actions":[]}`}}
	rp := &provider.RegisteredProvider{
		Config:   &provider.ProviderConfig{ID: "p1", Name: "P", Model: "m"},
		Protocol: mock,
	}
	agent := &models.Agent{ID: "a1", Name: "Agent"}
	w := NewWorker("w1", agent, rp)
	_ = w.Start()

	task := &Task{ID: "t1", Description: "do something"}
	config := &LoopConfig{MaxIterations: 5, Router: &actions.Router{}}

	result, _ := w.ExecuteTaskWithLoop(ctx, task, config)
	if result.TerminalReason != "context_canceled" {
		t.Errorf("TerminalReason = %q, want context_canceled", result.TerminalReason)
	}
}

func TestWorker_ExecuteTaskWithLoop_ConversationalSlip(t *testing.T) {
	mock := &sequenceMockProvider{
		responses: []string{
			"What would you like me to do next?",                              // conversational slip
			`{"actions": [{"type": "done", "reason": "ok"}]}`, // proper legacy format
		},
	}
	rp := &provider.RegisteredProvider{
		Config:   &provider.ProviderConfig{ID: "p1", Name: "P", Model: "m"},
		Protocol: mock,
	}
	agent := &models.Agent{ID: "a1", Name: "Agent"}
	w := NewWorker("w1", agent, rp)
	_ = w.Start()

	task := &Task{ID: "t1", Description: "do something"}
	config := &LoopConfig{
		MaxIterations: 10,
		Router:        &actions.Router{},
		ActionContext: actions.ActionContext{},
		TextMode:      false, // Use legacy parser which produces parse errors for non-JSON
	}

	result, err := w.ExecuteTaskWithLoop(context.Background(), task, config)
	if err != nil {
		t.Fatalf("ExecuteTaskWithLoop error = %v", err)
	}
	if result.TerminalReason != "completed" {
		t.Errorf("TerminalReason = %q, want completed", result.TerminalReason)
	}
}

func TestWorker_ExecuteTaskWithLoop_WithConversationSession(t *testing.T) {
	mock := &sequenceMockProvider{
		responses: []string{`{"action": "done", "reason": "done"}`},
	}
	rp := &provider.RegisteredProvider{
		Config:   &provider.ProviderConfig{ID: "p1", Name: "P", Model: "m"},
		Protocol: mock,
	}
	agent := &models.Agent{ID: "a1", Name: "Agent"}
	w := NewWorker("w1", agent, rp)
	_ = w.Start()

	convCtx := models.NewConversationContext("sess-1", "b1", "p1", time.Duration(24*time.Hour))
	task := &Task{
		ID:                  "t1",
		Description:         "do task",
		ConversationSession: convCtx,
	}
	config := &LoopConfig{
		MaxIterations: 5,
		Router:        &actions.Router{},
		ActionContext: actions.ActionContext{ProjectID: "p1", BeadID: "b1"},
		TextMode:      true,
	}

	result, err := w.ExecuteTaskWithLoop(context.Background(), task, config)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.TerminalReason != "completed" {
		t.Errorf("TerminalReason = %q, want completed", result.TerminalReason)
	}
}

func TestWorker_ExecuteTaskWithLoop_DefaultMaxIterations(t *testing.T) {
	mock := &sequenceMockProvider{
		responses: []string{`{"action": "done", "reason": "done"}`},
	}
	rp := &provider.RegisteredProvider{
		Config:   &provider.ProviderConfig{ID: "p1", Name: "P", Model: "m"},
		Protocol: mock,
	}
	agent := &models.Agent{ID: "a1", Name: "Agent"}
	w := NewWorker("w1", agent, rp)
	_ = w.Start()

	task := &Task{ID: "t1", Description: "do something"}
	config := &LoopConfig{
		MaxIterations: 0, // Should default to 25
		Router:        &actions.Router{},
		TextMode:      true,
	}

	result, err := w.ExecuteTaskWithLoop(context.Background(), task, config)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.TerminalReason != "completed" {
		t.Errorf("TerminalReason = %q, want completed", result.TerminalReason)
	}
}
