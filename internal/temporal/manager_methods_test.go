package temporal

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/config"
)

// TestNewManagerNilConfig verifies that NewManager returns an error for nil config.
// (This also covers the first branch in NewManager.)
func TestNewManagerNilConfigDirect(t *testing.T) {
	m, err := NewManager(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
	if m != nil {
		t.Error("expected nil manager for nil config")
	}
	if err != nil && !strings.Contains(err.Error(), "cannot be nil") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestManagerGetClientNil tests GetClient on a manually constructed Manager.
func TestManagerGetClientNil(t *testing.T) {
	m := &Manager{}
	if m.GetClient() != nil {
		t.Error("expected nil client from empty manager")
	}
}

// TestManagerGetEventBusNil tests GetEventBus on a manually constructed Manager.
func TestManagerGetEventBusNil(t *testing.T) {
	m := &Manager{}
	if m.GetEventBus() != nil {
		t.Error("expected nil event bus from empty manager")
	}
}

// TestManagerStopNilFields tests that Stop handles nil fields gracefully.
func TestManagerStopNilFields(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		// worker, eventBus, and client are all nil
	}
	// Should not panic
	m.Stop()
}

// TestManagerExecuteTemporalDSLEmptyText tests ExecuteTemporalDSL with empty text.
func TestManagerExecuteTemporalDSLEmptyText(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	_, err := m.ExecuteTemporalDSL(context.Background(), "agent-1", "")
	if err == nil {
		t.Error("expected error for empty DSL text")
	}
	if err != nil && !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestManagerExecuteTemporalDSLNoBlocks tests ExecuteTemporalDSL with text that has no temporal blocks.
func TestManagerExecuteTemporalDSLNoBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	_, err := m.ExecuteTemporalDSL(context.Background(), "agent-1", "just normal text no blocks")
	if err == nil {
		t.Error("expected error for text with no temporal instructions")
	}
	if err != nil && !strings.Contains(err.Error(), "no temporal instructions") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestManagerExecuteTemporalDSLInvalidDSL tests ExecuteTemporalDSL with invalid DSL syntax.
func TestManagerExecuteTemporalDSLInvalidDSL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	// DSL block with no valid instructions (no colon in instruction header)
	dsl := `<temporal>
INVALID LINE WITHOUT COLON
END
</temporal>`

	_, err := m.ExecuteTemporalDSL(context.Background(), "agent-1", dsl)
	if err == nil {
		t.Error("expected error for DSL with no valid instructions")
	}
	if err != nil && !strings.Contains(err.Error(), "no temporal instructions") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestManagerParseTemporalInstructions tests ParseTemporalInstructions method.
func TestManagerParseTemporalInstructions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	text := `Here is some text.
<temporal>
WORKFLOW: ParseTest
  TIMEOUT: 3m
END
</temporal>
And more text here.`

	instrs, cleanedText, err := m.ParseTemporalInstructions(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}
	if instrs[0].Name != "ParseTest" {
		t.Errorf("expected name ParseTest, got %s", instrs[0].Name)
	}
	if strings.Contains(cleanedText, "<temporal>") {
		t.Error("cleaned text should not contain <temporal> tags")
	}
	if !strings.Contains(cleanedText, "Here is some text") {
		t.Error("cleaned text should contain surrounding text")
	}
}

// TestManagerParseTemporalInstructionsNoBlocks tests with no temporal blocks.
func TestManagerParseTemporalInstructionsNoBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	instrs, cleanedText, err := m.ParseTemporalInstructions("just regular text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 0 {
		t.Errorf("expected 0 instructions, got %d", len(instrs))
	}
	if cleanedText != "just regular text" {
		t.Errorf("expected unchanged text, got %q", cleanedText)
	}
}

// TestManagerParseTemporalInstructionsWithInvalid tests validation logging.
func TestManagerParseTemporalInstructionsWithInvalid(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	// WORKFLOW without a name will fail validation
	text := `<temporal>
WORKFLOW:
  TIMEOUT: 3m
END
</temporal>`

	instrs, _, err := m.ParseTemporalInstructions(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The instruction is still returned even though validation logs a warning
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction (even invalid ones are returned), got %d", len(instrs))
	}
}

// TestManagerStripTemporalDSL tests the StripTemporalDSL method.
func TestManagerStripTemporalDSL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	text := `Please execute:
<temporal>
WORKFLOW: MyWorkflow
  TIMEOUT: 5m
END
</temporal>
That is all.`

	cleaned, err := m.StripTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(cleaned, "<temporal>") {
		t.Error("cleaned text should not contain <temporal> tags")
	}
	if !strings.Contains(cleaned, "Please execute") {
		t.Error("cleaned text should retain surrounding text")
	}
	if !strings.Contains(cleaned, "That is all") {
		t.Error("cleaned text should retain trailing text")
	}
}

// TestManagerStripTemporalDSLEmpty tests StripTemporalDSL with empty text.
func TestManagerStripTemporalDSLEmpty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	cleaned, err := m.StripTemporalDSL("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleaned != "" {
		t.Errorf("expected empty cleaned text, got %q", cleaned)
	}
}

// TestManagerExecuteActivity tests the ExecuteActivity stub that returns "not implemented".
func TestManagerExecuteActivity(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	result, err := m.ExecuteActivity(context.Background(), ActivityOptions{
		Name:    "TestActivity",
		Input:   "test-input",
		Timeout: time.Minute,
	})
	if err == nil {
		t.Error("expected error from unimplemented ExecuteActivity")
	}
	if result != nil {
		t.Error("expected nil result from unimplemented method")
	}
	if err != nil && !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestManagerCreateSchedule tests the CreateSchedule stub.
func TestManagerCreateSchedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	id, err := m.CreateSchedule(context.Background(), ScheduleOptions{
		Name:     "test-schedule",
		Workflow: "TestWorkflow",
		Interval: 24 * time.Hour,
	})
	if err == nil {
		t.Error("expected error from unimplemented CreateSchedule")
	}
	if id != "" {
		t.Error("expected empty id from unimplemented method")
	}
	if err != nil && !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestManagerListWorkflows tests the ListWorkflows stub.
func TestManagerListWorkflows(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	workflows, err := m.ListWorkflows(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workflows == nil {
		t.Error("expected non-nil workflows slice")
	}
	if len(workflows) != 0 {
		t.Errorf("expected empty workflow list, got %d", len(workflows))
	}
}

// TestManagerStructFields tests direct field access on Manager.
func TestManagerStructFields(t *testing.T) {
	cfg := &config.TemporalConfig{
		TaskQueue:                "my-queue",
		WorkflowExecutionTimeout: time.Hour,
		WorkflowTaskTimeout:      10 * time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := &Manager{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	if m.config.TaskQueue != "my-queue" {
		t.Errorf("TaskQueue: expected my-queue, got %s", m.config.TaskQueue)
	}
	if m.config.WorkflowExecutionTimeout != time.Hour {
		t.Errorf("WorkflowExecutionTimeout: expected 1h, got %v", m.config.WorkflowExecutionTimeout)
	}
}

// TestManagerExecuteTemporalDSLOnlyTextNoTags tests DSL with text but no temporal tags.
func TestManagerExecuteTemporalDSLOnlyTextNoTags(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	_, err := m.ExecuteTemporalDSL(context.Background(), "agent-1", "just plain text with no temporal tags at all")
	if err == nil {
		t.Error("expected error for text without temporal blocks")
	}
	if err != nil && !strings.Contains(err.Error(), "no temporal instructions") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestManagerParseTemporalInstructionsMultipleTypes tests parsing different instruction types.
func TestManagerParseTemporalInstructionsMultipleTypes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	text := `<temporal>
WORKFLOW: WF1
  TIMEOUT: 1m
END
SCHEDULE: Sched1
  INTERVAL: 1h
  TIMEOUT: 5m
END
QUERY: Q1
  WORKFLOW_ID: wf-123
  TYPE: status
END
SIGNAL: Sig1
  WORKFLOW_ID: wf-456
  NAME: wake
END
ACTIVITY: Act1
  TIMEOUT: 30s
END
CANCEL: Cancel1
  WORKFLOW_ID: wf-789
END
LIST: List1
END
</temporal>`

	instrs, _, err := m.ParseTemporalInstructions(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 7 {
		t.Fatalf("expected 7 instructions, got %d", len(instrs))
	}

	expectedTypes := []TemporalInstructionType{
		InstructionTypeWorkflow,
		InstructionTypeSchedule,
		InstructionTypeQuery,
		InstructionTypeSignal,
		InstructionTypeActivity,
		InstructionTypeCancelWF,
		InstructionTypeListWF,
	}

	for i, expected := range expectedTypes {
		if instrs[i].Type != expected {
			t.Errorf("instruction %d: expected type %s, got %s", i, expected, instrs[i].Type)
		}
	}
}

// TestManagerStripTemporalDSLMultipleBlocks tests stripping multiple blocks.
func TestManagerStripTemporalDSLMultipleBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{TaskQueue: "test-queue"},
	}

	text := `Before.
<temporal>
WORKFLOW: W1
END
</temporal>
Middle.
<temporal>
WORKFLOW: W2
END
</temporal>
After.`

	cleaned, err := m.StripTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(cleaned, "<temporal>") {
		t.Error("should not contain temporal tags")
	}
	if !strings.Contains(cleaned, "Before") {
		t.Error("should contain Before")
	}
	if !strings.Contains(cleaned, "Middle") {
		t.Error("should contain Middle")
	}
	if !strings.Contains(cleaned, "After") {
		t.Error("should contain After")
	}
}
