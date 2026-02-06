# Integration Tests

This directory contains end-to-end integration tests for Loom's agentic workflows.

## Overview

Integration tests validate complete workflows from start to finish, ensuring that all components work together correctly. These tests focus on realistic agent scenarios rather than individual unit functionality.

## Test Structure

### agentic_workflows_test.go

Comprehensive tests for complete agentic workflows:

#### 1. Bug Fix Workflow (`TestBugFixWorkflow`)
Tests the complete bug fix cycle:
- Agent receives bug report bead
- Reads code to understand the issue
- Fixes the bug with code edits
- Runs tests to verify the fix
- Commits changes with proper attribution
- Pushes to remote branch
- Creates pull request for review

**Validates:**
- Action sequencing (read → fix → test → commit → PR)
- Proper error handling at each step
- Git workflow integration
- Test feedback loop

#### 2. Feature Development Workflow (`TestFeatureDevelopmentWorkflow`)
Tests the EPCC (Explore, Plan, Code, Commit) workflow:
- **Explore Phase**: Read existing code, search patterns, analyze structure
- **Plan Phase**: Get workflow guidance, design approach
- **Code Phase**: Implement feature, write tests, verify (build/lint/test)
- **Commit Phase**: Git operations (commit/push/PR)

**Validates:**
- Workflow phase transitions
- Action coordination within phases
- Build → Lint → Test feedback loop
- Workflow resumption after breaks

#### 3. Multi-Agent Collaboration (`TestMultiAgentCollaboration`)
Tests coordination between multiple agent personas:
- Product Manager creates PRD bead
- Engineering Manager creates technical breakdown
- Implementation agent writes code
- QA agent tests the implementation
- Code Reviewer reviews the PR

**Validates:**
- Bead-based coordination
- Inter-agent dependencies
- Work handoff between agents
- Parallel agent operations

#### 4. Escalation Workflow (`TestEscalationWorkflow`)
Tests the decision escalation system:
- Agent encounters decision point
- Creates decision bead with options
- Escalates to CEO with `escalate_ceo` action
- CEO approves or rejects with reasons
- Agent proceeds based on decision

**Validates:**
- Decision bead creation
- Escalation mechanism
- Approval/rejection handling
- Workflow continuation after decision

#### 5. Feedback Loop Integration (`TestFeedbackLoopIntegration`)
Tests the iterative build → lint → test cycle:
- Agent writes code (build fails)
- Fixes build errors
- Runs linter (lint fails)
- Fixes lint errors
- Runs tests (tests fail)
- Fixes test failures
- All checks pass → commits

**Validates:**
- Iterative error fixing
- Build/lint/test orchestration
- Error feedback parsing
- Quality gate enforcement

#### 6. Workflow Resumption (`TestWorkflowResumption`)
Tests resuming work after conversation breaks:
- Agent working in code phase
- Session interrupted (conversation compaction)
- Agent resumes workflow
- Continues from where left off

**Validates:**
- Workflow state persistence
- Context recovery after breaks
- `resume_workflow` action
- Seamless continuation

#### 7. Complete End-to-End Scenario (`TestCompleteEndToEndScenario`)
Realistic scenario: Bug report → Investigation → Fix → PR → Close:
1. User reports bug via bead
2. Agent investigates (read files, search)
3. Agent fixes bug
4. Agent runs tests/linter
5. Agent commits and creates PR
6. Agent closes bead

**Validates:**
- Complete workflow from start to finish
- All action types working together
- Realistic agent behavior
- Proper bead lifecycle

## Running Tests

### Run All Integration Tests

```bash
go test ./tests/integration/... -v
```

### Run Specific Test

```bash
go test ./tests/integration/... -v -run TestBugFixWorkflow
```

### Skip Integration Tests (Short Mode)

```bash
go test ./tests/integration/... -short
# Integration tests will be skipped
```

### Run with Timeout

```bash
go test ./tests/integration/... -v -timeout 60s
```

## Test Philosophy

### What These Tests Validate

✅ **Action Schema Correctness**
- All action types validate properly
- Required fields are enforced
- JSON encoding/decoding works

✅ **Workflow Integrity**
- Actions execute in logical order
- Dependencies are respected
- Feedback loops work correctly

✅ **Integration Points**
- Git operations integrate with actions
- Workflow system integrates with actions
- Bead system integrates with actions
- Test/lint/build feedback integrates

✅ **Realistic Scenarios**
- Workflows mirror actual agent behavior
- Error handling is realistic
- Multi-agent coordination works

### What These Tests DON'T Validate

❌ **Actual Execution**
- Tests validate action structure, not execution results
- No real git operations performed
- No real builds/tests run
- No real LLM calls made

❌ **External Dependencies**
- GitHub API integration (use gh CLI directly for that)
- LLM provider connections
- Database operations

These are integration tests focused on **workflow structure and coordination**, not full system tests with real external services.

## Mock vs. Real Integration

These tests use **structural validation** rather than full mocks:

```go
// Tests validate structure
envelope := &actions.ActionEnvelope{
    Actions: []actions.Action{
        {Type: actions.ActionGitCommit, CommitMessage: "..."},
    },
}
err := actions.Validate(envelope)  // ✓ Structure valid

// They don't execute actions
// router.Execute(envelope)  // ✗ Not done in these tests
```

For full execution tests with mocks, see `cmd/loom/integration_test.go` (when implemented).

## Adding New Tests

When adding new workflow tests:

1. **Name test after workflow**: `TestXXXWorkflow`
2. **Document the scenario**: Add comments explaining the flow
3. **Test action sequence**: Verify actions are in logical order
4. **Validate structure**: Use `actions.Validate()` for each envelope
5. **Test JSON round-trip**: Ensure encoding/decoding works
6. **Use realistic data**: Bead IDs, paths, commit messages should be realistic

### Template

```go
func TestMyNewWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    ctx := context.Background()

    // Step 1: Description
    envelope := &actions.ActionEnvelope{
        Actions: []actions.Action{
            // Your actions
        },
        Notes: "What this step does",
    }

    // Validate
    err := actions.Validate(envelope)
    if err != nil {
        t.Fatalf("Validation failed: %v", err)
    }

    // Test JSON round-trip
    jsonData, _ := json.Marshal(envelope)
    _, err = actions.DecodeStrict(jsonData)
    if err != nil {
        t.Fatalf("Decode failed: %v", err)
    }

    t.Log("✓ My workflow validated")
}
```

## Related Documentation

- [Agent Actions Reference](../../docs/WORKFLOW_ACTIONS.md) - Action schema documentation
- [Git Workflow](../../docs/GIT_SECURITY_MODEL.md) - Git operations and security
- [Feedback Loops](../../docs/FEEDBACK_LOOPS.md) - Build/lint/test orchestration
- [Loom Story](../../docs/LOOM_STORY.md) - High-level system overview

## CI/CD Integration

These tests should run in CI pipelines:

```yaml
# .github/workflows/integration-tests.yml
- name: Run Integration Tests
  run: go test ./tests/integration/... -v -timeout 60s
```

Integration tests are fast (structural validation only) and should always pass if the action schema is correct.

## Troubleshooting

### Test Fails: "unknown action type"

Check that the action constant is defined in `internal/actions/schema.go`:
```go
const (
    ActionMyNewType = "my_new_type"
)
```

### Test Fails: "validation failed"

Check that validation rules are correct in `internal/actions/schema.go`:
```go
case ActionMyNewType:
    if action.RequiredField == "" {
        return errors.New("my_new_type requires required_field")
    }
```

### Test Fails: JSON Decode Error

Ensure the action struct has proper JSON tags:
```go
type Action struct {
    MyField string `json:"my_field,omitempty"`
}
```

## Performance

These tests are **fast** because they only validate structure:
- No network calls
- No file I/O
- No external processes
- No LLM API calls

Expected runtime: **< 1 second** for all tests.

If tests are slow, something is wrong (likely attempting real execution).
