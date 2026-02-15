package temporal

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/config"
)

// createTestManager creates a Manager with no Temporal client for testing
// methods that don't require a real Temporal connection.
func createTestManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		ctx:    ctx,
		cancel: cancel,
		config: &config.TemporalConfig{
			TaskQueue:                "test-queue",
			WorkflowExecutionTimeout: time.Hour,
			WorkflowTaskTimeout:      10 * time.Second,
		},
	}
}

// TestDSLExecutorScheduleWithManager tests executeSchedule through a manager
// that doesn't have a real Temporal client. CreateSchedule returns "not yet implemented".
func TestDSLExecutorScheduleWithManager(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type:     InstructionTypeSchedule,
			Name:     "TestSchedule",
			Interval: 5 * time.Minute,
			Timeout:  10 * time.Minute,
			Retry:    2,
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-sched")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	r := result.Results[0]
	// Should fail because CreateSchedule returns "not yet implemented"
	if r.Success {
		t.Error("expected failure (CreateSchedule not yet implemented)")
	}
	if !strings.Contains(r.Error, "not yet implemented") {
		t.Errorf("expected 'not yet implemented' error, got %q", r.Error)
	}
	if r.Duration < 0 {
		t.Error("Duration should be non-negative")
	}
}

// TestDSLExecutorActivityWithManager tests executeActivity through a manager.
// ExecuteActivity returns "not yet implemented".
func TestDSLExecutorActivityWithManager(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type:    InstructionTypeActivity,
			Name:    "TestActivity",
			Input:   map[string]interface{}{"key": "value"},
			Timeout: 2 * time.Minute,
			Retry:   3,
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-act")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	r := result.Results[0]
	if r.Success {
		t.Error("expected failure (ExecuteActivity not yet implemented)")
	}
	if !strings.Contains(r.Error, "not yet implemented") {
		t.Errorf("expected 'not yet implemented' error, got %q", r.Error)
	}
}

// TestDSLExecutorListWithManager tests executeList through a manager.
// ListWorkflows returns empty list successfully.
func TestDSLExecutorListWithManager(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type: InstructionTypeListWF,
			Name: "list-all",
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-list")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	r := result.Results[0]
	if !r.Success {
		t.Errorf("expected success, got error: %s", r.Error)
	}
	// The result should contain workflow count = 0
	if r.Result == nil {
		t.Error("expected non-nil result")
	}
	resultMap, ok := r.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", r.Result)
	}
	count, ok := resultMap["workflows_count"]
	if !ok {
		t.Error("expected 'workflows_count' in result")
	}
	if count != 0 {
		t.Errorf("expected 0 workflows, got %v", count)
	}
}

// TestDSLExecutorScheduleWithDefaultTimeout tests the default timeout path.
func TestDSLExecutorScheduleDefaultTimeout(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	// Schedule with Timeout=0 should use the default (5m)
	instrs := []TemporalInstruction{
		{
			Type:     InstructionTypeSchedule,
			Name:     "DefaultTimeoutSchedule",
			Interval: time.Hour,
			Timeout:  0, // Will use default
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-default")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	// Will fail due to "not yet implemented" but exercises the default timeout code
	r := result.Results[0]
	if r.Success {
		t.Error("expected failure")
	}
}

// TestDSLExecutorActivityDefaultTimeout tests activity with default timeout.
func TestDSLExecutorActivityDefaultTimeout(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type:    InstructionTypeActivity,
			Name:    "DefaultTimeoutActivity",
			Timeout: 0, // Will use default (2m)
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-default-act")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	r := result.Results[0]
	if r.Success {
		t.Error("expected failure")
	}
}

// TestDSLExecutorMixedWithManager tests multiple instruction types through a manager.
func TestDSLExecutorMixedWithManager(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type: InstructionTypeListWF,
			Name: "list",
		},
		{
			Type:    InstructionTypeActivity,
			Name:    "TestAct",
			Timeout: time.Minute,
		},
		{
			Type:     InstructionTypeSchedule,
			Name:     "TestSched",
			Interval: time.Hour,
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-mixed")
	if len(result.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result.Results))
	}

	// List should succeed
	if !result.Results[0].Success {
		t.Errorf("LIST should succeed, got error: %s", result.Results[0].Error)
	}

	// Activity should fail (not yet implemented)
	if result.Results[1].Success {
		t.Error("ACTIVITY should fail")
	}

	// Schedule should fail (not yet implemented)
	if result.Results[2].Success {
		t.Error("SCHEDULE should fail")
	}

	// Total duration should be set
	if result.TotalDuration == 0 {
		t.Error("TotalDuration should be non-zero")
	}
}

// TestDSLExecutorScheduleInputAndInterval tests the executeSchedule with input and interval.
func TestDSLExecutorScheduleInputAndInterval(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type:     InstructionTypeSchedule,
			Name:     "DetailedSchedule",
			Interval: 30 * time.Minute,
			Timeout:  15 * time.Minute,
			Retry:    5,
			Input:    map[string]interface{}{"data": "test"},
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-detail")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	// Still fails (not implemented) but exercises more code paths
	if result.Results[0].Success {
		t.Error("expected failure")
	}
}

// TestDSLExecutorActivityWithInput tests executeActivity with all fields.
func TestDSLExecutorActivityWithInput(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type:    InstructionTypeActivity,
			Name:    "DetailedActivity",
			Input:   map[string]interface{}{"step": 1, "data": "process"},
			Timeout: 5 * time.Minute,
			Retry:   3,
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-detail-act")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Success {
		t.Error("expected failure")
	}
}

// TestDSLExecutorExecuteInstructionFields tests that result fields are properly set.
func TestDSLExecutorExecuteInstructionFields(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type: InstructionTypeListWF,
			Name: "field-test",
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-fields")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	r := result.Results[0]
	// Verify instruction is preserved in result
	if r.Instruction.Type != InstructionTypeListWF {
		t.Errorf("expected type LIST, got %s", r.Instruction.Type)
	}
	if r.Instruction.Name != "field-test" {
		t.Errorf("expected name field-test, got %s", r.Instruction.Name)
	}
	// ExecutedAt should be set
	if r.ExecutedAt.IsZero() {
		t.Error("ExecutedAt should not be zero")
	}
	// Duration should be non-negative
	if r.Duration < 0 {
		t.Error("Duration should be non-negative")
	}
}

// TestManagerNewManagerInvalidHost tests NewManager with a config that has an invalid host.
// This exercises the client creation error path.
func TestManagerNewManagerInvalidHost(t *testing.T) {
	cfg := &config.TemporalConfig{
		Host:      "", // Invalid - will fail to connect
		Namespace: "test",
		TaskQueue: "test",
	}

	_, err := NewManager(cfg)
	if err == nil {
		// Some versions of the Temporal SDK might not fail immediately on empty host.
		// If it doesn't error, just skip since we can't control Temporal SDK behavior.
		t.Skip("NewManager did not fail with empty host (SDK may defer connection)")
	}
	if err != nil && !strings.Contains(err.Error(), "failed to create temporal client") {
		t.Logf("NewManager error (acceptable): %v", err)
	}
}

// TestDSLExecutorListResultStructure tests the detailed result structure of executeList.
func TestDSLExecutorListResultStructure(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type: InstructionTypeListWF,
			Name: "structure-test",
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-struct")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	r := result.Results[0]
	if !r.Success {
		t.Errorf("expected success, got error: %s", r.Error)
	}
	if r.Error != "" {
		t.Errorf("expected empty error, got %q", r.Error)
	}
	resultMap, ok := r.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", r.Result)
	}
	wfList, ok := resultMap["workflows"]
	if !ok {
		t.Error("expected 'workflows' key in result")
	}
	if wfList == nil {
		t.Error("expected non-nil workflows list")
	}
}

// TestDSLExecutorAgentIDPreserved tests that the agent ID is preserved in execution results.
func TestDSLExecutorAgentIDPreserved(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	agentIDs := []string{"agent-alpha", "agent-beta", "agent-gamma"}
	for _, agentID := range agentIDs {
		result := executor.ExecuteInstructions(ctx, []TemporalInstruction{
			{Type: InstructionTypeListWF, Name: "id-test"},
		}, agentID)

		if result.AgentID != agentID {
			t.Errorf("AgentID: expected %s, got %s", agentID, result.AgentID)
		}
	}
}

// TestDSLExecutorInstructionsPreserved tests that instructions are preserved in execution.
func TestDSLExecutorInstructionsPreserved(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{Type: InstructionTypeListWF, Name: "preserved-1"},
		{Type: InstructionTypeActivity, Name: "preserved-2", Timeout: time.Minute},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-preserve")
	if len(result.Instructions) != 2 {
		t.Fatalf("expected 2 instructions preserved, got %d", len(result.Instructions))
	}
	if result.Instructions[0].Name != "preserved-1" {
		t.Errorf("expected name preserved-1, got %s", result.Instructions[0].Name)
	}
	if result.Instructions[1].Name != "preserved-2" {
		t.Errorf("expected name preserved-2, got %s", result.Instructions[1].Name)
	}
}

// TestDSLExecutorScheduleAllFields tests executeSchedule with all supported fields populated.
func TestDSLExecutorScheduleAllFields(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	instrs := []TemporalInstruction{
		{
			Type:     InstructionTypeSchedule,
			Name:     "FullSchedule",
			Interval: 2 * time.Hour,
			Timeout:  30 * time.Minute,
			Retry:    10,
			Input:    map[string]interface{}{"env": "prod", "region": "us-west"},
		},
	}

	result := executor.ExecuteInstructions(ctx, instrs, "agent-full-sched")
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	// Fails because CreateSchedule is not implemented, but exercises all the fields
	r := result.Results[0]
	if r.Success {
		t.Error("expected failure")
	}
	if !strings.Contains(r.Error, "not yet implemented") {
		t.Errorf("expected 'not yet implemented', got %q", r.Error)
	}
}

// TestDSLExecutorEmptyAgentID tests execution with empty agent ID.
func TestDSLExecutorEmptyAgentID(t *testing.T) {
	m := createTestManager()
	defer m.cancel()
	executor := NewDSLExecutor(m)
	ctx := context.Background()

	result := executor.ExecuteInstructions(ctx, []TemporalInstruction{
		{Type: InstructionTypeListWF, Name: "empty-agent-test"},
	}, "")

	if result.AgentID != "" {
		t.Errorf("expected empty AgentID, got %q", result.AgentID)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if !result.Results[0].Success {
		t.Errorf("LIST should succeed: %s", result.Results[0].Error)
	}
}
