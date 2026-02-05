# Consensus Decision Making

The Consensus Decision Making system enables multiple agents to collaboratively vote on decisions, ensuring that important changes have support from the team rather than being made unilaterally.

## Overview

Consensus decisions allow teams of agents to:
- Vote on architectural changes, refactorings, or major features
- Reach agreement through democratic voting (approve/reject/abstain)
- Set quorum thresholds to ensure sufficient participation
- Handle timeouts gracefully when deadlines pass
- Track voting history and rationales

## Key Concepts

### Vote Choices

- **Approve**: Agent agrees with the decision
- **Reject**: Agent disagrees with the decision
- **Abstain**: Agent has no strong opinion (doesn't count toward approval rate)

### Decision Status

- **Pending**: Waiting for votes
- **Approved**: Consensus reached - decision approved
- **Rejected**: Consensus reached - decision rejected
- **Timeout**: Deadline passed without reaching quorum
- **Cancelled**: Decision was cancelled before completion

### Quorum Rules

- **Quorum Threshold**: Percentage of required agents who must vote (default: 0.67 = 2/3)
- **Approval Threshold**: Percentage of approve/(approve+reject) needed (default: same as quorum, 0.67)
- **Participation Rate**: Actual votes / required agents
- **Approval Rate**: Approve votes / (approve + reject votes) - abstains excluded

## Usage

### Creating a Decision

```go
import "github.com/jordanhubbard/agenticorp/internal/consensus"

dm := consensus.NewDecisionManager()
defer dm.Close()

decision, err := dm.CreateDecision(
    ctx,
    "Refactor Authentication Module",                    // Title
    "Refactoring authentication to improve security",    // Description
    "Should we refactor the authentication module?",      // Question
    "agent-pm-1",                                         // Created by
    []string{"agent-eng-1", "agent-qa-1", "agent-reviewer-1"}, // Required voters
    time.Now().Add(24 * time.Hour),                      // Deadline (24h from now)
    0.67,                                                 // Quorum threshold (2/3)
)
```

### Casting Votes

```go
// Agent approves
err := dm.CastVote(
    ctx,
    decision.ID,
    "agent-eng-1",
    consensus.VoteApprove,
    "Authentication module has technical debt that needs addressing",
    0.85, // Confidence: 0.0-1.0
)

// Agent rejects
err := dm.CastVote(
    ctx,
    decision.ID,
    "agent-qa-1",
    consensus.VoteReject,
    "Not enough test coverage for safe refactoring",
    0.75,
)

// Agent abstains
err := dm.CastVote(
    ctx,
    decision.ID,
    "agent-reviewer-1",
    consensus.VoteAbstain,
    "I don't have enough context to vote",
    0.0,
)
```

### Checking Decision Status

```go
decision, err := dm.GetDecision(ctx, decisionID)

if decision.Status == consensus.StatusApproved {
    fmt.Println("Decision approved!")
    fmt.Printf("Approval rate: %.1f%%\n", decision.Result.ApprovalRate * 100)
    fmt.Printf("Votes: %d approve, %d reject, %d abstain\n",
        decision.Result.ApproveCount,
        decision.Result.RejectCount,
        decision.Result.AbstainCount)
}
```

### Listing Decisions

```go
// Get all pending decisions
pending := dm.ListDecisions(ctx, consensus.StatusPending)

// Get all approved decisions
approved := dm.ListDecisions(ctx, consensus.StatusApproved)

// Get all decisions (any status)
all := dm.ListDecisions(ctx, "")
```

### Cancelling a Decision

```go
err := dm.CancelDecision(ctx, decisionID)
```

## Resolution Logic

A decision is automatically resolved when:

1. **Quorum is met** AND
2. **Approval threshold is met** (approved) OR **not met** (rejected)

### Resolution Examples

**Example 1: 3 Required Agents, 67% Threshold**

```
Scenario: All 3 agents vote
- Agent 1: Approve
- Agent 2: Approve
- Agent 3: Reject

Participation: 3/3 = 100% ≥ 67% ✓ (quorum met)
Approval: 2/3 = 66.7% < 67% ✗ (threshold not met)
Result: REJECTED
```

**Example 2: 3 Required Agents, 67% Threshold**

```
Scenario: All 3 agents vote
- Agent 1: Approve
- Agent 2: Approve
- Agent 3: Approve

Participation: 3/3 = 100% ≥ 67% ✓ (quorum met)
Approval: 3/3 = 100% ≥ 67% ✓ (threshold met)
Result: APPROVED
```

**Example 3: 3 Required Agents, 67% Threshold, With Abstain**

```
Scenario: All 3 agents vote
- Agent 1: Approve
- Agent 2: Approve
- Agent 3: Abstain

Participation: 3/3 = 100% ≥ 67% ✓ (quorum met)
Approval: 2/2 = 100% ≥ 67% ✓ (abstains don't count)
Result: APPROVED
```

**Example 4: 5 Required Agents, 67% Threshold**

```
Scenario: Only 3 agents vote
- Agent 1: Approve
- Agent 2: Reject
- Agent 3: (no vote)
- Agent 4: (no vote)
- Agent 5: (no vote)

Participation: 3/5 = 60% < 67% ✗ (quorum not met)
Result: PENDING (waiting for more votes or timeout)
```

## Timeout Handling

When a decision deadline passes:

1. Automatic timeout monitor checks every minute
2. Decision status changes to `StatusTimeout`
3. Result includes partial vote counts
4. No additional votes can be cast

```go
// Manually check timeout
err := dm.CheckTimeout(ctx, decisionID)

// Decision will be marked as timeout if deadline passed
decision, _ := dm.GetDecision(ctx, decisionID)
if decision.Status == consensus.StatusTimeout {
    fmt.Printf("Decision timed out with %d/%d votes\n",
        decision.Result.TotalVotes,
        len(decision.RequiredAgents))
}
```

## Integration with Agent Messaging

Consensus decisions can be combined with agent messaging:

```go
// 1. Create decision
decision, _ := dm.CreateDecision(ctx, "Refactor Auth", "...", "Approve refactor?",
    "agent-pm-1",
    []string{"agent-eng-1", "agent-qa-1", "agent-reviewer-1"},
    time.Now().Add(24*time.Hour),
    0.67)

// 2. Notify agents via message bus
for _, agentID := range decision.RequiredAgents {
    messageBus.Send(ctx, &messaging.AgentMessage{
        Type:        messaging.MessageTypeConsensusRequest,
        FromAgentID: "agent-pm-1",
        ToAgentID:   agentID,
        Subject:     decision.Title,
        Body:        decision.Question,
        Context: map[string]interface{}{
            "decision_id": decision.ID,
            "deadline": decision.Deadline,
        },
    })
}

// 3. Agents vote
// ... agents cast votes via CastVote() ...

// 4. Decision automatically resolves when quorum met

// 5. Notify all agents of result
if decision.Status == consensus.StatusApproved {
    for _, agentID := range decision.RequiredAgents {
        messageBus.Send(ctx, &messaging.AgentMessage{
            Type:        messaging.MessageTypeNotification,
            FromAgentID: "agent-pm-1",
            ToAgentID:   agentID,
            Subject:     "Decision Approved: " + decision.Title,
            Body:        fmt.Sprintf("Consensus reached: %d approve, %d reject",
                decision.Result.ApproveCount, decision.Result.RejectCount),
        })
    }
}
```

## Use Cases

### 1. Architectural Decisions

```go
decision, _ := dm.CreateDecision(
    ctx,
    "Adopt GraphQL API",
    "Replace REST API with GraphQL for better client flexibility",
    "Should we migrate from REST to GraphQL?",
    "agent-architect-1",
    []string{"agent-frontend-1", "agent-backend-1", "agent-mobile-1"},
    time.Now().Add(48 * time.Hour), // 2 days
    0.67,
)
```

### 2. Major Refactoring

```go
decision, _ := dm.CreateDecision(
    ctx,
    "Refactor Database Layer",
    "Refactor database abstraction layer for better testability",
    "Approve database layer refactoring?",
    "agent-tech-lead-1",
    []string{"agent-eng-1", "agent-eng-2", "agent-qa-1"},
    time.Now().Add(24 * time.Hour),
    0.67,
)
```

### 3. Dependency Upgrades

```go
decision, _ := dm.CreateDecision(
    ctx,
    "Upgrade to Go 1.26",
    "Upgrade Go version from 1.25 to 1.26 (includes breaking changes)",
    "Approve Go 1.26 upgrade?",
    "agent-devops-1",
    []string{"agent-eng-1", "agent-eng-2", "agent-qa-1", "agent-ci-1"},
    time.Now().Add(72 * time.Hour), // 3 days
    0.75, // Higher threshold for breaking changes
)
```

### 4. Breaking API Changes

```go
decision, _ := dm.CreateDecision(
    ctx,
    "Deprecate Legacy Endpoint",
    "Remove /api/v1/legacy endpoint (breaking change for old clients)",
    "Approve removal of legacy endpoint?",
    "agent-api-owner-1",
    []string{"agent-backend-1", "agent-frontend-1", "agent-mobile-1", "agent-support-1"},
    time.Now().Add(7 * 24 * time.Hour), // 1 week
    0.75, // Higher threshold for breaking changes
)
```

## Best Practices

### 1. Choose Appropriate Thresholds

```go
// Simple decisions (50% = simple majority)
threshold := 0.5

// Important decisions (67% = 2/3 majority)
threshold := 0.67

// Critical decisions (75% = 3/4 majority)
threshold := 0.75

// Unanimous decisions (100%)
threshold := 1.0
```

### 2. Set Realistic Deadlines

```go
// Quick decisions (1 day)
deadline := time.Now().Add(24 * time.Hour)

// Normal decisions (2-3 days)
deadline := time.Now().Add(48 * time.Hour)

// Strategic decisions (1 week)
deadline := time.Now().Add(7 * 24 * time.Hour)
```

### 3. Provide Context in Vote Rationales

```go
// Good - specific and actionable
dm.CastVote(ctx, decisionID, agentID, consensus.VoteReject,
    "Current test coverage is 45%, should be >80% before refactoring", 0.9)

// Bad - vague
dm.CastVote(ctx, decisionID, agentID, consensus.VoteReject,
    "Not ready", 0.5)
```

### 4. Include All Stakeholders

```go
// Good - include all affected teams
requiredAgents := []string{
    "agent-backend-eng",
    "agent-frontend-eng",
    "agent-qa",
    "agent-devops",
    "agent-security",
}

// Bad - missing key stakeholders
requiredAgents := []string{
    "agent-backend-eng", // Security team should review too!
}
```

### 5. Use Confidence Scores

```go
// High confidence (0.9-1.0): Expert knowledge
dm.CastVote(ctx, decisionID, agentID, consensus.VoteApprove,
    "I designed this module and am confident in the refactoring approach", 0.95)

// Medium confidence (0.6-0.8): General knowledge
dm.CastVote(ctx, decisionID, agentID, consensus.VoteApprove,
    "Looks good based on code review, but not my area of expertise", 0.7)

// Low confidence (0.3-0.5): Uncertain
dm.CastVote(ctx, decisionID, agentID, consensus.VoteAbstain,
    "Not familiar enough with this codebase to vote", 0.3)
```

## API Reference

### ConsensusDecision

```go
type ConsensusDecision struct {
    ID              string
    Title           string
    Description     string
    Question        string
    Context         map[string]interface{}
    CreatedBy       string
    CreatedAt       time.Time
    Deadline        time.Time
    Status          DecisionStatus
    RequiredAgents  []string
    QuorumThreshold float64
    Votes           map[string]Vote
    Result          *DecisionResult
}
```

### Vote

```go
type Vote struct {
    AgentID    string
    Choice     VoteChoice // approve, reject, abstain
    Rationale  string
    Confidence float64 // 0.0-1.0
    Timestamp  time.Time
}
```

### DecisionResult

```go
type DecisionResult struct {
    FinalStatus       DecisionStatus
    ApproveCount      int
    RejectCount       int
    AbstainCount      int
    TotalVotes        int
    QuorumMet         bool
    ApprovalRate      float64 // approve / (approve + reject)
    ParticipationRate float64 // voted / required
    ResolvedAt        time.Time
}
```

## Implementation Details

- **Thread Safety**: All operations are thread-safe using RWMutex
- **Automatic Resolution**: Decisions resolve immediately when quorum+threshold met
- **Timeout Monitoring**: Background goroutine checks timeouts every minute
- **No Revoting**: Once a vote is cast, it cannot be changed
- **Voting After Resolution**: Cannot vote after decision is resolved (approved/rejected/timeout/cancelled)

## Performance

- **Memory**: ~500 bytes per decision + ~100 bytes per vote
- **Vote Latency**: < 1ms to record vote
- **Resolution Check**: O(n) where n = number of votes (typically < 20)
- **Timeout Check**: O(m) where m = number of pending decisions

## Related Documentation

- [Agent Communication Protocol](AGENT_COMMUNICATION.md)
- [Agent Message Bus](../internal/messaging/README.md)
- [Shared Bead Context](SHARED_CONTEXT.md)

## Future Enhancements

1. **Weighted Voting**: Different agents have different vote weights
2. **Delegate Voting**: Agents can delegate their vote to another agent
3. **Ranked Choice**: Support for multiple options with ranked voting
4. **Vote Revocation**: Allow agents to change their vote before deadline
5. **Persistent Storage**: Save decisions to database
6. **Notification Hooks**: Webhooks when decisions resolve
7. **Decision Templates**: Pre-defined decision types with standard thresholds
