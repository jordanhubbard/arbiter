package motivation

import (
	"context"
	"testing"
	"time"
)

func TestCalendarEvaluator_ScheduledInterval(t *testing.T) {
	eval := &CalendarEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	m := &Motivation{
		Condition: ConditionScheduledInterval,
		// LastTriggeredAt nil means never triggered -> should fire
	}

	triggered, data, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !triggered {
		t.Error("should trigger when never triggered before")
	}
	_ = data
}

func TestCalendarEvaluator_TimeReached(t *testing.T) {
	eval := &CalendarEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	past := time.Now().Add(-1 * time.Hour)
	m := &Motivation{
		Condition:     ConditionTimeReached,
		NextTriggerAt: &past,
	}

	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !triggered {
		t.Error("should trigger when time has passed")
	}

	// Future time should not trigger
	future := time.Now().Add(1 * time.Hour)
	m.NextTriggerAt = &future
	triggered, _, err = eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("should not trigger when time is in future")
	}
}

func TestCalendarEvaluator_DeadlineApproach(t *testing.T) {
	eval := &CalendarEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	// Add upcoming deadlines
	sp.upcomingDeadlines = []BeadDeadlineInfo{
		{BeadID: "bead-1", DueDate: time.Now().Add(3 * 24 * time.Hour)},
	}

	m := &Motivation{
		Condition:  ConditionDeadlineApproach,
		Parameters: map[string]interface{}{"days_threshold": 7},
	}

	triggered, data, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !triggered {
		t.Error("should trigger with upcoming deadlines")
	}
	if data["count"] != 1 {
		t.Errorf("count = %v, want 1", data["count"])
	}
}

func TestCalendarEvaluator_DeadlinePassed(t *testing.T) {
	eval := &CalendarEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	sp.overdueBeads = []BeadDeadlineInfo{
		{BeadID: "overdue-1", DueDate: time.Now().Add(-24 * time.Hour)},
	}

	m := &Motivation{
		Condition: ConditionDeadlinePassed,
	}

	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !triggered {
		t.Error("should trigger with overdue beads")
	}
}

func TestCalendarEvaluator_NoTrigger(t *testing.T) {
	eval := &CalendarEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	// Empty state, no matching condition
	m := &Motivation{
		Condition: ConditionDeadlinePassed,
	}

	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("should not trigger with no overdue beads")
	}
}

func TestEventEvaluator_DecisionPending(t *testing.T) {
	eval := &EventEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	sp.pendingDecisions = []string{"decision-1"}

	m := &Motivation{
		Condition: ConditionDecisionPending,
	}

	triggered, data, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !triggered {
		t.Error("should trigger with pending decisions")
	}
	if data["count"] != 1 {
		t.Errorf("count = %v, want 1", data["count"])
	}
}

func TestThresholdEvaluator_CostExceeded(t *testing.T) {
	eval := &ThresholdEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	sp.currentSpending = 100.0
	sp.budgetThreshold = 80.0

	m := &Motivation{
		Condition:  ConditionCostExceeded,
		Parameters: map[string]interface{}{"threshold_percent": 90.0},
	}

	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	// 100/80 = 125% which is > 90%
	if !triggered {
		t.Error("should trigger when cost exceeds threshold")
	}
}

func TestIdleEvaluator_SystemIdle(t *testing.T) {
	eval := &IdleEvaluator{idleThreshold: 5 * time.Minute}
	ctx := context.Background()
	sp := NewMockStateProvider()

	sp.systemIdle = true

	m := &Motivation{
		Condition: ConditionSystemIdle,
	}

	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !triggered {
		t.Error("should trigger when system is idle")
	}
}

func TestIdleEvaluator_NotIdle(t *testing.T) {
	eval := &IdleEvaluator{idleThreshold: 5 * time.Minute}
	ctx := context.Background()
	sp := NewMockStateProvider()

	sp.systemIdle = false

	m := &Motivation{
		Condition: ConditionSystemIdle,
	}

	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("should not trigger when system is not idle")
	}
}

func TestExternalEvaluator_NoEvents(t *testing.T) {
	eval := &ExternalEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()
	sp.externalEvents = make(map[string][]ExternalEvent)

	m := &Motivation{
		Condition: ConditionGitHubIssueOpened,
	}

	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("should not trigger with no events")
	}
}

func TestExternalEvaluator_WithEvents(t *testing.T) {
	eval := &ExternalEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()
	sp.externalEvents = map[string][]ExternalEvent{
		"github_issue_opened": {{Type: "github_issue_opened", Data: map[string]interface{}{"title": "bug"}}},
	}

	m := &Motivation{
		Condition: ConditionGitHubIssueOpened,
	}

	triggered, data, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !triggered {
		t.Error("should trigger with events")
	}
	if data["count"] != 1 {
		t.Errorf("count = %v, want 1", data["count"])
	}
}

func TestExternalEvaluator_UnknownCondition(t *testing.T) {
	eval := &ExternalEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	m := &Motivation{
		Condition: "unknown_condition",
	}

	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("should not trigger for unknown condition")
	}
}

func TestEventEvaluator_BeadCreated(t *testing.T) {
	eval := &EventEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	m := &Motivation{Condition: ConditionBeadCreated}
	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("event-driven conditions should not trigger via polling")
	}
}

func TestEventEvaluator_ReleasePublished(t *testing.T) {
	eval := &EventEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	m := &Motivation{Condition: ConditionReleasePublished}
	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("should not trigger without release tracking")
	}
}

func TestThresholdEvaluator_CoverageDropped(t *testing.T) {
	eval := &ThresholdEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	m := &Motivation{Condition: ConditionCoverageDropped}
	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("should not trigger without coverage integration")
	}
}

func TestThresholdEvaluator_TestFailure(t *testing.T) {
	eval := &ThresholdEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	m := &Motivation{Condition: ConditionTestFailure}
	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("should not trigger without CI/CD integration")
	}
}

func TestThresholdEvaluator_VelocityDrop(t *testing.T) {
	eval := &ThresholdEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	m := &Motivation{Condition: ConditionVelocityDrop}
	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("should not trigger without velocity tracking")
	}
}

func TestIdleEvaluator_ProjectIdle(t *testing.T) {
	eval := &IdleEvaluator{idleThreshold: 5 * time.Minute}
	ctx := context.Background()
	sp := NewMockStateProvider()
	sp.projectIdle = map[string]bool{"proj1": true}

	m := &Motivation{
		Condition: ConditionProjectIdle,
		ProjectID: "proj1",
	}

	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !triggered {
		t.Error("should trigger when project is idle")
	}
}

func TestIdleEvaluator_AgentIdle(t *testing.T) {
	eval := &IdleEvaluator{idleThreshold: 5 * time.Minute}
	ctx := context.Background()
	sp := NewMockStateProvider()
	sp.idleAgents = []string{"agent-1", "agent-2"}
	sp.agentsByRole = map[string][]string{"coder": {"agent-1", "agent-2"}}

	m := &Motivation{
		Condition: ConditionAgentIdle,
		AgentRole: "coder",
	}

	triggered, data, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !triggered {
		t.Error("should trigger when agents are idle")
	}
	if data["role"] != "coder" {
		t.Errorf("role = %v, want coder", data["role"])
	}
}

func TestCalendarEvaluator_ScheduledIntervalWithLastTrigger(t *testing.T) {
	eval := &CalendarEvaluator{}
	ctx := context.Background()
	sp := NewMockStateProvider()

	past := time.Now().Add(-2 * time.Hour)
	m := &Motivation{
		Condition:      ConditionScheduledInterval,
		LastTriggeredAt: &past,
		CooldownPeriod: 1 * time.Hour,
	}

	triggered, _, err := eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !triggered {
		t.Error("should trigger when interval has passed")
	}

	// Recently triggered - should not fire
	recent := time.Now().Add(-5 * time.Minute)
	m.LastTriggeredAt = &recent
	triggered, _, err = eval.Evaluate(ctx, m, sp)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("should not trigger when recently triggered")
	}
}
