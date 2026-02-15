package temporal

import (
	"strings"
	"testing"
	"time"
)

func TestParseTemporalDSLEmptyString(t *testing.T) {
	instrs, cleaned, err := ParseTemporalDSL("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instrs != nil {
		t.Errorf("expected nil instructions for empty string, got %v", instrs)
	}
	if cleaned != "" {
		t.Errorf("expected empty cleaned text, got %q", cleaned)
	}
}

func TestParseTemporalDSLScheduleInstruction(t *testing.T) {
	text := `<temporal>
SCHEDULE: DailyReport
  INTERVAL: 24h
  TIMEOUT: 10m
  RETRY: 2
END
</temporal>`

	instrs, _, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}

	instr := instrs[0]
	if instr.Type != InstructionTypeSchedule {
		t.Errorf("expected SCHEDULE, got %s", instr.Type)
	}
	if instr.Name != "DailyReport" {
		t.Errorf("expected DailyReport, got %s", instr.Name)
	}
	if instr.Interval != 24*time.Hour {
		t.Errorf("expected 24h interval, got %v", instr.Interval)
	}
	if instr.Timeout != 10*time.Minute {
		t.Errorf("expected 10m timeout, got %v", instr.Timeout)
	}
	if instr.Retry != 2 {
		t.Errorf("expected retry 2, got %d", instr.Retry)
	}
}

func TestParseTemporalDSLSignalInstruction(t *testing.T) {
	text := `<temporal>
SIGNAL: workflow-abc
  WORKFLOW_ID: wf-123
  NAME: wake-up
  DATA: {"reason": "new_bead"}
  RUN_ID: run-456
END
</temporal>`

	instrs, _, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}

	instr := instrs[0]
	if instr.Type != InstructionTypeSignal {
		t.Errorf("expected SIGNAL, got %s", instr.Type)
	}
	if instr.WorkflowID != "wf-123" {
		t.Errorf("expected workflow_id wf-123, got %s", instr.WorkflowID)
	}
	if instr.SignalName != "wake-up" {
		t.Errorf("expected signal name wake-up, got %s", instr.SignalName)
	}
	if instr.RunID != "run-456" {
		t.Errorf("expected run_id run-456, got %s", instr.RunID)
	}
	if instr.SignalData["reason"] != "new_bead" {
		t.Errorf("expected signal data reason=new_bead, got %v", instr.SignalData["reason"])
	}
}

func TestParseTemporalDSLCancelInstruction(t *testing.T) {
	text := `<temporal>
CANCEL: stop-workflow
  WORKFLOW_ID: wf-789
  RUN_ID: run-abc
END
</temporal>`

	instrs, _, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}

	instr := instrs[0]
	if instr.Type != InstructionTypeCancelWF {
		t.Errorf("expected CANCEL, got %s", instr.Type)
	}
	if instr.WorkflowID != "wf-789" {
		t.Errorf("expected workflow_id wf-789, got %s", instr.WorkflowID)
	}
	if instr.RunID != "run-abc" {
		t.Errorf("expected run_id run-abc, got %s", instr.RunID)
	}
}

func TestParseTemporalDSLActivityInstruction(t *testing.T) {
	text := `<temporal>
ACTIVITY: ProcessData
  INPUT: {"data_source": "db-1"}
  TIMEOUT: 2m
  RETRY: 5
END
</temporal>`

	instrs, _, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}

	instr := instrs[0]
	if instr.Type != InstructionTypeActivity {
		t.Errorf("expected ACTIVITY, got %s", instr.Type)
	}
	if instr.Name != "ProcessData" {
		t.Errorf("expected ProcessData, got %s", instr.Name)
	}
	if instr.Input["data_source"] != "db-1" {
		t.Errorf("expected input data_source=db-1, got %v", instr.Input["data_source"])
	}
	if instr.Retry != 5 {
		t.Errorf("expected retry 5, got %d", instr.Retry)
	}
}

func TestParseTemporalDSLListInstruction(t *testing.T) {
	text := `<temporal>
LIST: all-workflows
END
</temporal>`

	instrs, _, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}

	instr := instrs[0]
	if instr.Type != InstructionTypeListWF {
		t.Errorf("expected LIST, got %s", instr.Type)
	}
}

func TestParseTemporalDSLWithPriorityAndIdempotency(t *testing.T) {
	text := `<temporal>
WORKFLOW: HighPriorityTask
  INPUT: {"task": "urgent"}
  PRIORITY: 10
  IDEMPOTENCY_KEY: unique-key-123
  DESCRIPTION: An urgent processing task
  WAIT: false
END
</temporal>`

	instrs, _, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}

	instr := instrs[0]
	if instr.Priority != 10 {
		t.Errorf("expected priority 10, got %d", instr.Priority)
	}
	if instr.IdempotencyKey != "unique-key-123" {
		t.Errorf("expected idempotency_key unique-key-123, got %s", instr.IdempotencyKey)
	}
	if instr.Description != "An urgent processing task" {
		t.Errorf("expected description 'An urgent processing task', got %s", instr.Description)
	}
	if instr.Wait {
		t.Error("expected Wait=false")
	}
}

func TestParseTemporalDSLMultipleBlocks(t *testing.T) {
	text := `First block here.

<temporal>
WORKFLOW: FirstWorkflow
  TIMEOUT: 1m
END
</temporal>

Some middle text.

<temporal>
WORKFLOW: SecondWorkflow
  TIMEOUT: 2m
END
</temporal>

Final text.`

	instrs, cleaned, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(instrs))
	}

	if instrs[0].Name != "FirstWorkflow" {
		t.Errorf("expected first workflow name FirstWorkflow, got %s", instrs[0].Name)
	}
	if instrs[1].Name != "SecondWorkflow" {
		t.Errorf("expected second workflow name SecondWorkflow, got %s", instrs[1].Name)
	}

	if strings.Contains(cleaned, "<temporal>") {
		t.Error("cleaned text still contains <temporal> tags")
	}
	if !strings.Contains(cleaned, "First block here") {
		t.Error("cleaned text missing first block text")
	}
	if !strings.Contains(cleaned, "Some middle text") {
		t.Error("cleaned text missing middle text")
	}
	if !strings.Contains(cleaned, "Final text") {
		t.Error("cleaned text missing final text")
	}
}

func TestParseTemporalDSLInvalidJSONInput(t *testing.T) {
	text := `<temporal>
WORKFLOW: TestWorkflow
  INPUT: {invalid json}
END
</temporal>`

	instrs, _, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should parse instruction but with empty input due to bad JSON
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}
	// Input should still be empty map (initialized but JSON parse failed)
	if len(instrs[0].Input) != 0 {
		t.Errorf("expected empty input map after bad JSON, got %v", instrs[0].Input)
	}
}

func TestParseTemporalDSLInstructionWithoutColon(t *testing.T) {
	text := `<temporal>
INVALID LINE WITHOUT COLON
END
</temporal>`

	instrs, _, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should skip the invalid instruction
	if len(instrs) != 0 {
		t.Errorf("expected 0 instructions for invalid input, got %d", len(instrs))
	}
}

func TestParseTemporalDSLCaseInsensitiveKeys(t *testing.T) {
	text := `<temporal>
workflow: MyWorkflow
  timeout: 5m
  retry: 2
  wait: true
END
</temporal>`

	instrs, _, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}

	instr := instrs[0]
	if instr.Type != InstructionTypeWorkflow {
		t.Errorf("expected WORKFLOW type, got %s", instr.Type)
	}
	if instr.Timeout != 5*time.Minute {
		t.Errorf("expected 5m timeout, got %v", instr.Timeout)
	}
	if instr.Retry != 2 {
		t.Errorf("expected retry 2, got %d", instr.Retry)
	}
	if !instr.Wait {
		t.Error("expected Wait=true")
	}
}

func TestParseDurationVariants(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"1h30m", 90 * time.Minute, false},
		{"500ms", 500 * time.Millisecond, false},
		{"now", 0, false},
		{"immediate", 0, false},
		{"default", 5 * time.Minute, false},
		{"120", 120 * time.Second, false},
		{"  30s  ", 30 * time.Second, false},
		{"invalid-duration", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseJSONInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		keys    []string
	}{
		{
			name:    "valid json object",
			input:   `{"key": "value", "num": 42}`,
			wantErr: false,
			keys:    []string{"key", "num"},
		},
		{
			name:    "empty json object",
			input:   `{}`,
			wantErr: false,
			keys:    []string{},
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "json with whitespace",
			input:   `  {"trimmed": "yes"}  `,
			wantErr: false,
			keys:    []string{"trimmed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := make(map[string]interface{})
			err := parseJSONInput(tt.input, target)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSONInput() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				for _, key := range tt.keys {
					if _, ok := target[key]; !ok {
						t.Errorf("expected key %q in parsed result", key)
					}
				}
			}
		})
	}
}

func TestParseJSONInputMerge(t *testing.T) {
	target := map[string]interface{}{
		"existing": "value",
	}
	err := parseJSONInput(`{"new_key": "new_value"}`, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target["existing"] != "value" {
		t.Error("existing key was lost")
	}
	if target["new_key"] != "new_value" {
		t.Error("new key was not merged")
	}
}

func TestValidateInstructionComprehensive(t *testing.T) {
	tests := []struct {
		name    string
		instr   TemporalInstruction
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid workflow",
			instr:   TemporalInstruction{Type: InstructionTypeWorkflow, Name: "WF"},
			wantErr: false,
		},
		{
			name:    "workflow without name",
			instr:   TemporalInstruction{Type: InstructionTypeWorkflow},
			wantErr: true,
			errMsg:  "requires NAME",
		},
		{
			name:    "valid schedule",
			instr:   TemporalInstruction{Type: InstructionTypeSchedule, Name: "Sched", Interval: time.Minute},
			wantErr: false,
		},
		{
			name:    "schedule without name",
			instr:   TemporalInstruction{Type: InstructionTypeSchedule, Interval: time.Minute},
			wantErr: true,
			errMsg:  "requires NAME",
		},
		{
			name:    "schedule without interval",
			instr:   TemporalInstruction{Type: InstructionTypeSchedule, Name: "Sched"},
			wantErr: true,
			errMsg:  "requires INTERVAL",
		},
		{
			name:    "valid query",
			instr:   TemporalInstruction{Type: InstructionTypeQuery, WorkflowID: "wf-1", QueryType: "status"},
			wantErr: false,
		},
		{
			name:    "query without workflow_id",
			instr:   TemporalInstruction{Type: InstructionTypeQuery, QueryType: "status"},
			wantErr: true,
			errMsg:  "requires WORKFLOW_ID",
		},
		{
			name:    "query without type",
			instr:   TemporalInstruction{Type: InstructionTypeQuery, WorkflowID: "wf-1"},
			wantErr: true,
			errMsg:  "requires TYPE",
		},
		{
			name:    "valid signal",
			instr:   TemporalInstruction{Type: InstructionTypeSignal, WorkflowID: "wf-1", SignalName: "sig"},
			wantErr: false,
		},
		{
			name:    "signal without workflow_id",
			instr:   TemporalInstruction{Type: InstructionTypeSignal, SignalName: "sig"},
			wantErr: true,
			errMsg:  "requires WORKFLOW_ID",
		},
		{
			name:    "signal without name",
			instr:   TemporalInstruction{Type: InstructionTypeSignal, WorkflowID: "wf-1"},
			wantErr: true,
			errMsg:  "requires NAME",
		},
		{
			name:    "valid activity",
			instr:   TemporalInstruction{Type: InstructionTypeActivity, Name: "Act"},
			wantErr: false,
		},
		{
			name:    "activity without name",
			instr:   TemporalInstruction{Type: InstructionTypeActivity},
			wantErr: true,
			errMsg:  "requires NAME",
		},
		{
			name:    "valid cancel",
			instr:   TemporalInstruction{Type: InstructionTypeCancelWF, WorkflowID: "wf-1"},
			wantErr: false,
		},
		{
			name:    "cancel without workflow_id",
			instr:   TemporalInstruction{Type: InstructionTypeCancelWF},
			wantErr: true,
			errMsg:  "requires WORKFLOW_ID",
		},
		{
			name:    "valid list",
			instr:   TemporalInstruction{Type: InstructionTypeListWF},
			wantErr: false,
		},
		{
			name:    "unknown type",
			instr:   TemporalInstruction{Type: "UNKNOWN"},
			wantErr: true,
			errMsg:  "unknown instruction type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInstruction(tt.instr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstruction() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestFormatDSLComprehensive(t *testing.T) {
	t.Run("workflow with all fields", func(t *testing.T) {
		instr := TemporalInstruction{
			Type:       InstructionTypeWorkflow,
			Name:       "FullWorkflow",
			Timeout:    10 * time.Minute,
			Retry:      5,
			Wait:       true,
			Interval:   30 * time.Second,
			Input:      map[string]interface{}{"key": "value"},
			QueryType:  "status",
			SignalName: "sig",
			SignalData: map[string]interface{}{"data": "payload"},
		}

		formatted := FormatDSL(instr)

		expectations := []string{
			"<temporal>",
			"</temporal>",
			"WORKFLOW: FullWorkflow",
			"TIMEOUT:",
			"RETRY: 5",
			"WAIT: true",
			"INTERVAL:",
			"INPUT:",
			"TYPE: status",
			"NAME: sig",
			"DATA:",
			"END",
		}

		for _, exp := range expectations {
			if !strings.Contains(formatted, exp) {
				t.Errorf("formatted DSL missing %q", exp)
			}
		}
	})

	t.Run("minimal instruction", func(t *testing.T) {
		instr := TemporalInstruction{
			Type: InstructionTypeListWF,
			Name: "ListAll",
		}

		formatted := FormatDSL(instr)

		if !strings.Contains(formatted, "<temporal>") {
			t.Error("missing opening tag")
		}
		if !strings.Contains(formatted, "</temporal>") {
			t.Error("missing closing tag")
		}
		if !strings.Contains(formatted, "LIST: ListAll") {
			t.Error("missing instruction header")
		}
		// Should not contain optional fields
		if strings.Contains(formatted, "TIMEOUT:") {
			t.Error("should not contain TIMEOUT for zero value")
		}
		if strings.Contains(formatted, "RETRY:") {
			t.Error("should not contain RETRY for zero value")
		}
		if strings.Contains(formatted, "WAIT:") {
			t.Error("should not contain WAIT for false value")
		}
	})
}

func TestParseTemporalBlockEmpty(t *testing.T) {
	instrs, err := parseTemporalBlock("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 0 {
		t.Errorf("expected 0 instructions, got %d", len(instrs))
	}
}

func TestParseTemporalBlockMultipleENDs(t *testing.T) {
	block := `WORKFLOW: First
  TIMEOUT: 1m
END
WORKFLOW: Second
  TIMEOUT: 2m
END`

	instrs, err := parseTemporalBlock(block)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(instrs))
	}
	if instrs[0].Name != "First" {
		t.Errorf("expected first instruction name 'First', got %q", instrs[0].Name)
	}
	if instrs[1].Name != "Second" {
		t.Errorf("expected second instruction name 'Second', got %q", instrs[1].Name)
	}
}

func TestParseTemporalDSLWhitespaceHandling(t *testing.T) {
	text := `  <temporal>
  WORKFLOW: SpacedWorkflow
    TIMEOUT: 3m
    WAIT: true
  END
  </temporal>  `

	instrs, _, err := ParseTemporalDSL(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}
	if instrs[0].Name != "SpacedWorkflow" {
		t.Errorf("expected SpacedWorkflow, got %q", instrs[0].Name)
	}
}
