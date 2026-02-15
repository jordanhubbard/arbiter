package temporal

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewDSLExecutor(t *testing.T) {
	// Test with nil manager
	exec := NewDSLExecutor(nil)
	if exec == nil {
		t.Fatal("expected non-nil executor")
	}
	if exec.manager != nil {
		t.Error("expected nil manager in executor")
	}
}

func TestDSLExecutorExecuteInstructionsEmpty(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	result := exec.ExecuteInstructions(ctx, nil, "agent-1")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.AgentID != "agent-1" {
		t.Errorf("AgentID: expected agent-1, got %s", result.AgentID)
	}
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Results))
	}
	if result.ExecutedAt.IsZero() {
		t.Error("ExecutedAt should not be zero")
	}
}

func TestDSLExecutorExecuteInstructionsValidationFailure(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	// An instruction with missing required fields should produce a validation error
	instrs := []TemporalInstruction{
		{
			Type: InstructionTypeWorkflow,
			Name: "", // Missing name
		},
	}

	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Success {
		t.Error("expected validation failure (Success=false)")
	}
	if result.Results[0].Error == "" {
		t.Error("expected non-empty error for validation failure")
	}
}

func TestDSLExecutorNilManagerWorkflow(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type: InstructionTypeWorkflow,
			Name: "TestWorkflow",
		},
	}

	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	r := result.Results[0]
	if r.Success {
		t.Error("expected failure with nil manager")
	}
	if !strings.Contains(r.Error, "not initialized") {
		t.Errorf("expected 'not initialized' error, got %q", r.Error)
	}
}

func TestDSLExecutorNilManagerSchedule(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type:     InstructionTypeSchedule,
			Name:     "TestSchedule",
			Interval: 5 * time.Minute,
		},
	}

	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	r := result.Results[0]
	if r.Success {
		t.Error("expected failure with nil manager")
	}
	if !strings.Contains(r.Error, "not initialized") {
		t.Errorf("expected 'not initialized' error, got %q", r.Error)
	}
}

func TestDSLExecutorNilManagerQuery(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type:       InstructionTypeQuery,
			WorkflowID: "wf-123",
			QueryType:  "status",
		},
	}

	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Success {
		t.Error("expected failure with nil manager")
	}
}

func TestDSLExecutorNilManagerSignal(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type:       InstructionTypeSignal,
			WorkflowID: "wf-123",
			SignalName: "my-signal",
		},
	}

	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Success {
		t.Error("expected failure with nil manager")
	}
}

func TestDSLExecutorNilManagerActivity(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type: InstructionTypeActivity,
			Name: "TestActivity",
		},
	}

	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Success {
		t.Error("expected failure with nil manager")
	}
}

func TestDSLExecutorNilManagerCancel(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type:       InstructionTypeCancelWF,
			WorkflowID: "wf-123",
		},
	}

	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Success {
		t.Error("expected failure with nil manager")
	}
}

func TestDSLExecutorNilManagerList(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type: InstructionTypeListWF,
			Name: "list-all",
		},
	}

	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Success {
		t.Error("expected failure with nil manager")
	}
}

func TestDSLExecutorUnknownInstructionType(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type: "UNKNOWN_TYPE",
			Name: "test",
		},
	}

	// This should fail validation since UNKNOWN_TYPE is not recognized
	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Success {
		t.Error("expected failure for unknown instruction type")
	}
}

func TestDSLExecutorMultipleInstructionsMixed(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type: InstructionTypeWorkflow,
			Name: "", // Invalid - missing name
		},
		{
			Type: InstructionTypeWorkflow,
			Name: "ValidWorkflow", // Valid but nil manager
		},
		{
			Type: InstructionTypeListWF,
			Name: "list", // Valid but nil manager
		},
	}

	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if len(result.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result.Results))
	}

	// First should be validation failure
	if result.Results[0].Success {
		t.Error("first instruction should have failed validation")
	}

	// Second should fail due to nil manager
	if result.Results[1].Success {
		t.Error("second instruction should have failed (nil manager)")
	}

	// Third should fail due to nil manager
	if result.Results[2].Success {
		t.Error("third instruction should have failed (nil manager)")
	}

	// TotalDuration should be set
	if result.TotalDuration == 0 {
		t.Error("TotalDuration should be non-zero")
	}
}

func TestDSLExecutorResultDuration(t *testing.T) {
	exec := NewDSLExecutor(nil)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type: InstructionTypeWorkflow,
			Name: "TestWorkflow",
		},
	}

	result := exec.ExecuteInstructions(ctx, instrs, "agent-1")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	// Duration should be set (non-negative)
	if result.Results[0].Duration < 0 {
		t.Error("Duration should be non-negative")
	}

	// ExecutedAt should be set
	if result.Results[0].ExecutedAt.IsZero() {
		t.Error("ExecutedAt should not be zero")
	}
}
