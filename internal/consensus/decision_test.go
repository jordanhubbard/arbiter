package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDecisionManager(t *testing.T) {
	dm := NewDecisionManager()
	assert.NotNil(t, dm)
	defer dm.Close()
}

func TestCreateDecision(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	decision, err := dm.CreateDecision(
		ctx,
		"Refactor Auth Module",
		"Refactoring authentication to improve security",
		"Should we refactor the authentication module?",
		"agent-pm-1",
		[]string{"agent-eng-1", "agent-qa-1", "agent-reviewer-1"},
		time.Now().Add(24*time.Hour),
		0.67,
	)

	require.NoError(t, err)
	assert.NotEmpty(t, decision.ID)
	assert.Equal(t, "Refactor Auth Module", decision.Title)
	assert.Equal(t, StatusPending, decision.Status)
	assert.Len(t, decision.RequiredAgents, 3)
	assert.Equal(t, 0.67, decision.QuorumThreshold)
	assert.Empty(t, decision.Votes)
}

func TestCreateDecision_DefaultQuorum(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	// Pass invalid quorum (0), should default to 0.67
	decision, err := dm.CreateDecision(
		context.Background(),
		"Test Decision",
		"Test",
		"Test question?",
		"agent-1",
		[]string{"agent-2", "agent-3"},
		time.Now().Add(1*time.Hour),
		0, // Invalid - should default
	)

	require.NoError(t, err)
	assert.Equal(t, 0.67, decision.QuorumThreshold)
}

func TestCreateDecision_DefaultDeadline(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	decision, err := dm.CreateDecision(
		context.Background(),
		"Test Decision",
		"Test",
		"Test question?",
		"agent-1",
		[]string{"agent-2"},
		time.Time{}, // Zero time - should default to 24h
		0.5,
	)

	require.NoError(t, err)
	assert.True(t, decision.Deadline.After(time.Now()))
	assert.True(t, decision.Deadline.Before(time.Now().Add(25*time.Hour)))
}

func TestCreateDecision_ValidationErrors(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	tests := []struct {
		name           string
		title          string
		question       string
		requiredAgents []string
		wantErr        string
	}{
		{
			name:           "missing title",
			title:          "",
			question:       "Question?",
			requiredAgents: []string{"agent-1"},
			wantErr:        "title is required",
		},
		{
			name:           "missing question",
			title:          "Title",
			question:       "",
			requiredAgents: []string{"agent-1"},
			wantErr:        "question is required",
		},
		{
			name:           "no required agents",
			title:          "Title",
			question:       "Question?",
			requiredAgents: []string{},
			wantErr:        "at least one required agent must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dm.CreateDecision(
				ctx,
				tt.title,
				"Description",
				tt.question,
				"agent-1",
				tt.requiredAgents,
				time.Now().Add(1*time.Hour),
				0.67,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestCastVote_Approve(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	decision, _ := dm.CreateDecision(
		ctx,
		"Test Decision",
		"Description",
		"Approve?",
		"agent-pm-1",
		[]string{"agent-1", "agent-2", "agent-3"},
		time.Now().Add(1*time.Hour),
		0.67,
	)

	// Cast vote
	err := dm.CastVote(ctx, decision.ID, "agent-1", VoteApprove, "Looks good", 0.9)
	require.NoError(t, err)

	// Verify vote was recorded
	decision, _ = dm.GetDecision(ctx, decision.ID)
	assert.Len(t, decision.Votes, 1)

	vote := decision.Votes["agent-1"]
	assert.Equal(t, VoteApprove, vote.Choice)
	assert.Equal(t, "Looks good", vote.Rationale)
	assert.Equal(t, 0.9, vote.Confidence)
}

func TestCastVote_ReachConsensus(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	decision, _ := dm.CreateDecision(
		ctx,
		"Test Decision",
		"Description",
		"Approve?",
		"agent-pm-1",
		[]string{"agent-1", "agent-2", "agent-3"},
		time.Now().Add(1*time.Hour),
		0.67, // Need 67% approval
	)

	// 3 agents vote - 2 approve, 1 reject
	_ = dm.CastVote(ctx, decision.ID, "agent-1", VoteApprove, "Good", 0.8)
	_ = dm.CastVote(ctx, decision.ID, "agent-2", VoteApprove, "Agree", 0.9)
	_ = dm.CastVote(ctx, decision.ID, "agent-3", VoteReject, "Not yet", 0.7)

	// Decision should be rejected (2/3 = 66.67% < 67% threshold)
	decision, _ = dm.GetDecision(ctx, decision.ID)

	assert.Equal(t, StatusRejected, decision.Status)
	require.NotNil(t, decision.Result)
	assert.Equal(t, 2, decision.Result.ApproveCount)
	assert.Equal(t, 1, decision.Result.RejectCount)
	assert.True(t, decision.Result.QuorumMet)
	assert.InDelta(t, 0.667, decision.Result.ApprovalRate, 0.01)
}

func TestCastVote_ConsensusApproved(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	decision, _ := dm.CreateDecision(
		ctx,
		"Test Decision",
		"Description",
		"Approve?",
		"agent-pm-1",
		[]string{"agent-1", "agent-2", "agent-3"},
		time.Now().Add(1*time.Hour),
		0.67, // 2/3 threshold
	)

	// All 3 agents vote - all approve
	_ = dm.CastVote(ctx, decision.ID, "agent-1", VoteApprove, "Good", 0.8)
	_ = dm.CastVote(ctx, decision.ID, "agent-2", VoteApprove, "Agree", 0.9)
	_ = dm.CastVote(ctx, decision.ID, "agent-3", VoteApprove, "Yes", 0.9)

	// Decision should be approved (100% >= 67%)
	decision, _ = dm.GetDecision(ctx, decision.ID)

	assert.Equal(t, StatusApproved, decision.Status)
	require.NotNil(t, decision.Result)
	assert.Equal(t, 3, decision.Result.ApproveCount)
	assert.Equal(t, 0, decision.Result.RejectCount)
	assert.True(t, decision.Result.QuorumMet)
	assert.Equal(t, 1.0, decision.Result.ApprovalRate)
}

func TestCastVote_ConsensusRejected(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	decision, _ := dm.CreateDecision(
		ctx,
		"Test Decision",
		"Description",
		"Approve?",
		"agent-pm-1",
		[]string{"agent-1", "agent-2", "agent-3"},
		time.Now().Add(1*time.Hour),
		0.67,
	)

	// All 3 agents vote - all reject
	_ = dm.CastVote(ctx, decision.ID, "agent-1", VoteReject, "Not ready", 0.8)
	_ = dm.CastVote(ctx, decision.ID, "agent-2", VoteReject, "Issues", 0.9)
	_ = dm.CastVote(ctx, decision.ID, "agent-3", VoteReject, "Wait", 0.7)

	// Decision should be rejected (0% < 67%)
	decision, _ = dm.GetDecision(ctx, decision.ID)

	assert.Equal(t, StatusRejected, decision.Status)
	require.NotNil(t, decision.Result)
	assert.Equal(t, 0, decision.Result.ApproveCount)
	assert.Equal(t, 3, decision.Result.RejectCount)
	assert.Equal(t, 0.0, decision.Result.ApprovalRate)
}

func TestCastVote_WithAbstain(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	decision, _ := dm.CreateDecision(
		ctx,
		"Test Decision",
		"Description",
		"Approve?",
		"agent-pm-1",
		[]string{"agent-1", "agent-2", "agent-3"},
		time.Now().Add(1*time.Hour),
		0.67,
	)

	// 2 approve, 1 abstain
	_ = dm.CastVote(ctx, decision.ID, "agent-1", VoteApprove, "Good", 0.8)
	_ = dm.CastVote(ctx, decision.ID, "agent-2", VoteApprove, "Agree", 0.9)
	_ = dm.CastVote(ctx, decision.ID, "agent-3", VoteAbstain, "No opinion", 0.0)

	// Decision should be approved (2/(2+0) = 100% >= 67%, abstains don't count)
	decision, _ = dm.GetDecision(ctx, decision.ID)

	assert.Equal(t, StatusApproved, decision.Status)
	assert.Equal(t, 2, decision.Result.ApproveCount)
	assert.Equal(t, 1, decision.Result.AbstainCount)
	assert.Equal(t, 1.0, decision.Result.ApprovalRate) // abstains excluded from rate
}

func TestCastVote_AgentNotInRequiredList(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	decision, _ := dm.CreateDecision(
		ctx,
		"Test Decision",
		"Description",
		"Approve?",
		"agent-pm-1",
		[]string{"agent-1", "agent-2"},
		time.Now().Add(1*time.Hour),
		0.67,
	)

	// Try to vote with agent not in required list
	err := dm.CastVote(ctx, decision.ID, "agent-3", VoteApprove, "Yes", 0.9)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in the required voters list")
}

func TestCastVote_VotingClosed(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	decision, _ := dm.CreateDecision(
		ctx,
		"Test Decision",
		"Description",
		"Approve?",
		"agent-pm-1",
		[]string{"agent-1"},
		time.Now().Add(1*time.Hour),
		0.5,
	)

	// Vote to close decision
	_ = dm.CastVote(ctx, decision.ID, "agent-1", VoteApprove, "Yes", 0.9)

	// Decision is now approved and closed
	// Try to vote again
	err := dm.CastVote(ctx, decision.ID, "agent-1", VoteReject, "Changed mind", 0.8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "voting is closed")
}

func TestCheckTimeout(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	// Create decision with immediate deadline
	decision, _ := dm.CreateDecision(
		ctx,
		"Test Decision",
		"Description",
		"Approve?",
		"agent-pm-1",
		[]string{"agent-1", "agent-2", "agent-3"},
		time.Now().Add(-1*time.Hour), // Already expired
		0.67,
	)

	// Check timeout
	err := dm.CheckTimeout(ctx, decision.ID)
	require.NoError(t, err)

	// Decision should be timed out
	decision, _ = dm.GetDecision(ctx, decision.ID)
	assert.Equal(t, StatusTimeout, decision.Status)
	require.NotNil(t, decision.Result)
	assert.Equal(t, StatusTimeout, decision.Result.FinalStatus)
	assert.False(t, decision.Result.QuorumMet)
}

func TestCancelDecision(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	decision, _ := dm.CreateDecision(
		ctx,
		"Test Decision",
		"Description",
		"Approve?",
		"agent-pm-1",
		[]string{"agent-1", "agent-2"},
		time.Now().Add(1*time.Hour),
		0.67,
	)

	// Cancel decision
	err := dm.CancelDecision(ctx, decision.ID)
	require.NoError(t, err)

	// Verify status
	decision, _ = dm.GetDecision(ctx, decision.ID)
	assert.Equal(t, StatusCancelled, decision.Status)

	// Cannot vote on cancelled decision
	err = dm.CastVote(ctx, decision.ID, "agent-1", VoteApprove, "Yes", 0.9)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "voting is closed")
}

func TestListDecisions(t *testing.T) {
	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	// Create multiple decisions (need multiple agents to avoid immediate resolution)
	d1, _ := dm.CreateDecision(ctx, "Decision 1", "D1", "Q1?", "agent-1", []string{"agent-2", "agent-3"}, time.Now().Add(1*time.Hour), 0.5)
	d2, _ := dm.CreateDecision(ctx, "Decision 2", "D2", "Q2?", "agent-1", []string{"agent-2", "agent-3"}, time.Now().Add(1*time.Hour), 0.5)

	// Approve one (both agents vote to meet quorum)
	_ = dm.CastVote(ctx, d1.ID, "agent-2", VoteApprove, "Yes", 0.9)
	_ = dm.CastVote(ctx, d1.ID, "agent-3", VoteApprove, "Yes", 0.9)

	// List all decisions
	allDecisions := dm.ListDecisions(ctx, "")
	assert.Len(t, allDecisions, 2)

	// List only pending
	pendingDecisions := dm.ListDecisions(ctx, StatusPending)
	assert.Len(t, pendingDecisions, 1)
	assert.Equal(t, d2.ID, pendingDecisions[0].ID)

	// List only approved
	approvedDecisions := dm.ListDecisions(ctx, StatusApproved)
	assert.Len(t, approvedDecisions, 1)
	assert.Equal(t, d1.ID, approvedDecisions[0].ID)
}

func TestConcurrentVoting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	dm := NewDecisionManager()
	defer dm.Close()

	ctx := context.Background()

	agents := []string{"agent-1", "agent-2", "agent-3", "agent-4", "agent-5"}

	decision, _ := dm.CreateDecision(
		ctx,
		"Concurrent Test",
		"Description",
		"Approve?",
		"agent-pm-1",
		agents,
		time.Now().Add(1*time.Hour),
		0.6,
	)

	// Simulate concurrent voting
	done := make(chan bool, len(agents))
	for i, agentID := range agents {
		go func(id string, index int) {
			choice := VoteApprove
			if index%2 == 0 {
				choice = VoteReject
			}
			_ = dm.CastVote(ctx, decision.ID, id, choice, "Test", 0.8)
			done <- true
		}(agentID, i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < len(agents); i++ {
		<-done
	}

	// Verify decision was resolved (quorum met causes early resolution)
	decision, _ = dm.GetDecision(ctx, decision.ID)
	assert.NotEqual(t, StatusPending, decision.Status, "Decision should be resolved")
	require.NotNil(t, decision.Result)

	// Verify at least quorum voted (60% of 5 = 3 votes minimum)
	assert.GreaterOrEqual(t, len(decision.Votes), 3)
	assert.GreaterOrEqual(t, decision.Result.TotalVotes, 3)
	assert.True(t, decision.Result.QuorumMet)
}
