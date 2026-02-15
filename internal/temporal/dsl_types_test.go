package temporal

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTemporalInstructionTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      TemporalInstructionType
		expected string
	}{
		{"workflow", InstructionTypeWorkflow, "WORKFLOW"},
		{"schedule", InstructionTypeSchedule, "SCHEDULE"},
		{"query", InstructionTypeQuery, "QUERY"},
		{"signal", InstructionTypeSignal, "SIGNAL"},
		{"activity", InstructionTypeActivity, "ACTIVITY"},
		{"cancel", InstructionTypeCancelWF, "CANCEL"},
		{"list", InstructionTypeListWF, "LIST"},
		{"motivation", InstructionTypeMotivation, "MOTIVATION"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.got))
			}
		})
	}
}

func TestTemporalInstructionJSON(t *testing.T) {
	instr := TemporalInstruction{
		Type:           InstructionTypeWorkflow,
		Name:           "TestWorkflow",
		WorkflowID:     "wf-123",
		Input:          map[string]interface{}{"key": "value"},
		Timeout:        5 * time.Minute,
		Retry:          3,
		Wait:           true,
		Interval:       10 * time.Second,
		QueryType:      "status",
		SignalName:     "my-signal",
		SignalData:     map[string]interface{}{"data": "payload"},
		RunID:          "run-456",
		Priority:       5,
		IdempotencyKey: "idem-789",
		Description:    "Test instruction",
		AgentRole:      "developer",
		Condition:      "idle",
		Enabled:        true,
		CooldownMinutes: 30,
	}

	data, err := json.Marshal(instr)
	if err != nil {
		t.Fatalf("failed to marshal TemporalInstruction: %v", err)
	}

	var decoded TemporalInstruction
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal TemporalInstruction: %v", err)
	}

	if decoded.Type != instr.Type {
		t.Errorf("Type: expected %q, got %q", instr.Type, decoded.Type)
	}
	if decoded.Name != instr.Name {
		t.Errorf("Name: expected %q, got %q", instr.Name, decoded.Name)
	}
	if decoded.WorkflowID != instr.WorkflowID {
		t.Errorf("WorkflowID: expected %q, got %q", instr.WorkflowID, decoded.WorkflowID)
	}
	if decoded.Retry != instr.Retry {
		t.Errorf("Retry: expected %d, got %d", instr.Retry, decoded.Retry)
	}
	if decoded.Wait != instr.Wait {
		t.Errorf("Wait: expected %v, got %v", instr.Wait, decoded.Wait)
	}
	if decoded.Priority != instr.Priority {
		t.Errorf("Priority: expected %d, got %d", instr.Priority, decoded.Priority)
	}
	if decoded.IdempotencyKey != instr.IdempotencyKey {
		t.Errorf("IdempotencyKey: expected %q, got %q", instr.IdempotencyKey, decoded.IdempotencyKey)
	}
	if decoded.Description != instr.Description {
		t.Errorf("Description: expected %q, got %q", instr.Description, decoded.Description)
	}
	if decoded.AgentRole != instr.AgentRole {
		t.Errorf("AgentRole: expected %q, got %q", instr.AgentRole, decoded.AgentRole)
	}
	if decoded.Condition != instr.Condition {
		t.Errorf("Condition: expected %q, got %q", instr.Condition, decoded.Condition)
	}
	if decoded.Enabled != instr.Enabled {
		t.Errorf("Enabled: expected %v, got %v", instr.Enabled, decoded.Enabled)
	}
	if decoded.CooldownMinutes != instr.CooldownMinutes {
		t.Errorf("CooldownMinutes: expected %d, got %d", instr.CooldownMinutes, decoded.CooldownMinutes)
	}
}

func TestTemporalInstructionResultJSON(t *testing.T) {
	result := TemporalInstructionResult{
		Instruction: TemporalInstruction{
			Type: InstructionTypeWorkflow,
			Name: "TestWF",
		},
		Success:    true,
		Result:     map[string]interface{}{"workflow_id": "wf-1"},
		Error:      "",
		ExecutedAt: time.Now().UTC().Truncate(time.Second),
		Duration:   2 * time.Second,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded TemporalInstructionResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Success != result.Success {
		t.Errorf("Success: expected %v, got %v", result.Success, decoded.Success)
	}
	if decoded.Error != result.Error {
		t.Errorf("Error: expected %q, got %q", result.Error, decoded.Error)
	}
	if decoded.Instruction.Name != "TestWF" {
		t.Errorf("Instruction.Name: expected TestWF, got %s", decoded.Instruction.Name)
	}
}

func TestTemporalDSLExecutionJSON(t *testing.T) {
	exec := TemporalDSLExecution{
		AgentID: "agent-1",
		Instructions: []TemporalInstruction{
			{Type: InstructionTypeWorkflow, Name: "WF1"},
			{Type: InstructionTypeActivity, Name: "Act1"},
		},
		Results: []TemporalInstructionResult{
			{Success: true},
			{Success: false, Error: "failed"},
		},
		CleanedText:    "cleaned output",
		ExecutionError: "",
		TotalDuration:  5 * time.Second,
		ExecutedAt:     time.Now().UTC().Truncate(time.Second),
	}

	data, err := json.Marshal(exec)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded TemporalDSLExecution
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.AgentID != "agent-1" {
		t.Errorf("AgentID: expected agent-1, got %s", decoded.AgentID)
	}
	if len(decoded.Instructions) != 2 {
		t.Errorf("Instructions: expected 2, got %d", len(decoded.Instructions))
	}
	if len(decoded.Results) != 2 {
		t.Errorf("Results: expected 2, got %d", len(decoded.Results))
	}
	if decoded.CleanedText != "cleaned output" {
		t.Errorf("CleanedText: expected 'cleaned output', got %q", decoded.CleanedText)
	}
}

func TestWorkflowOptionsJSON(t *testing.T) {
	opts := WorkflowOptions{
		ID:             "wf-1",
		Name:           "TestWorkflow",
		Input:          map[string]string{"key": "value"},
		Timeout:        10 * time.Minute,
		Retry:          3,
		Wait:           true,
		Priority:       5,
		IdempotencyKey: "idem-1",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded WorkflowOptions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != opts.ID {
		t.Errorf("ID: expected %q, got %q", opts.ID, decoded.ID)
	}
	if decoded.Name != opts.Name {
		t.Errorf("Name: expected %q, got %q", opts.Name, decoded.Name)
	}
	if decoded.Retry != opts.Retry {
		t.Errorf("Retry: expected %d, got %d", opts.Retry, decoded.Retry)
	}
	if decoded.Wait != opts.Wait {
		t.Errorf("Wait: expected %v, got %v", opts.Wait, decoded.Wait)
	}
}

func TestActivityOptionsJSON(t *testing.T) {
	opts := ActivityOptions{
		Name:    "TestActivity",
		Input:   "input-data",
		Timeout: 2 * time.Minute,
		Retry:   5,
		Wait:    true,
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ActivityOptions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Name != opts.Name {
		t.Errorf("Name: expected %q, got %q", opts.Name, decoded.Name)
	}
	if decoded.Wait != opts.Wait {
		t.Errorf("Wait: expected %v, got %v", opts.Wait, decoded.Wait)
	}
}

func TestScheduleOptionsJSON(t *testing.T) {
	opts := ScheduleOptions{
		Name:     "daily-check",
		Workflow: "CheckWorkflow",
		Input:    "input",
		Interval: 24 * time.Hour,
		Timeout:  5 * time.Minute,
		Retry:    2,
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ScheduleOptions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Name != opts.Name {
		t.Errorf("Name: expected %q, got %q", opts.Name, decoded.Name)
	}
	if decoded.Workflow != opts.Workflow {
		t.Errorf("Workflow: expected %q, got %q", opts.Workflow, decoded.Workflow)
	}
}

func TestQueryOptionsJSON(t *testing.T) {
	opts := QueryOptions{
		WorkflowID: "wf-1",
		RunID:      "run-1",
		QueryType:  "getStatus",
		Args:       []interface{}{"arg1", 42},
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded QueryOptions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.WorkflowID != opts.WorkflowID {
		t.Errorf("WorkflowID: expected %q, got %q", opts.WorkflowID, decoded.WorkflowID)
	}
	if decoded.QueryType != opts.QueryType {
		t.Errorf("QueryType: expected %q, got %q", opts.QueryType, decoded.QueryType)
	}
}

func TestSignalOptionsJSON(t *testing.T) {
	opts := SignalOptions{
		WorkflowID: "wf-1",
		RunID:      "run-1",
		Name:       "my-signal",
		Data:       map[string]string{"key": "value"},
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded SignalOptions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.WorkflowID != opts.WorkflowID {
		t.Errorf("WorkflowID: expected %q, got %q", opts.WorkflowID, decoded.WorkflowID)
	}
	if decoded.Name != opts.Name {
		t.Errorf("Name: expected %q, got %q", opts.Name, decoded.Name)
	}
}

func TestCancelOptionsJSON(t *testing.T) {
	opts := CancelOptions{
		WorkflowID: "wf-1",
		RunID:      "run-1",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded CancelOptions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.WorkflowID != opts.WorkflowID {
		t.Errorf("WorkflowID: expected %q, got %q", opts.WorkflowID, decoded.WorkflowID)
	}
	if decoded.RunID != opts.RunID {
		t.Errorf("RunID: expected %q, got %q", opts.RunID, decoded.RunID)
	}
}

func TestMotivationOptionsJSON(t *testing.T) {
	opts := MotivationOptions{
		Name:            "idle-check",
		Type:            "idle",
		Condition:       "system_idle > 30m",
		AgentRole:       "ceo",
		Enabled:         true,
		CooldownMinutes: 60,
		Parameters:      map[string]interface{}{"threshold": 30},
		CreateBead:      true,
		WakeAgent:       true,
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded MotivationOptions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Name != opts.Name {
		t.Errorf("Name: expected %q, got %q", opts.Name, decoded.Name)
	}
	if decoded.Type != opts.Type {
		t.Errorf("Type: expected %q, got %q", opts.Type, decoded.Type)
	}
	if decoded.AgentRole != opts.AgentRole {
		t.Errorf("AgentRole: expected %q, got %q", opts.AgentRole, decoded.AgentRole)
	}
	if decoded.CooldownMinutes != opts.CooldownMinutes {
		t.Errorf("CooldownMinutes: expected %d, got %d", opts.CooldownMinutes, decoded.CooldownMinutes)
	}
	if decoded.CreateBead != opts.CreateBead {
		t.Errorf("CreateBead: expected %v, got %v", opts.CreateBead, decoded.CreateBead)
	}
	if decoded.WakeAgent != opts.WakeAgent {
		t.Errorf("WakeAgent: expected %v, got %v", opts.WakeAgent, decoded.WakeAgent)
	}
}
