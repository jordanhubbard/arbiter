# Test Coverage Gap Analysis

**Date:** 2026-02-12
**Current Overall Coverage:** ~25% (estimated)
**Target Coverage:** 75%
**Gap:** ~50 percentage points

## Package Coverage Status

| Package | Current | Target | Gap | Priority | Est. Tests Needed |
|---------|---------|--------|-----|----------|-------------------|
| internal/messaging | 77.5% | 75% | ✅ PASS | - | 0 |
| internal/worker | 29.7% | 75% | -45.3% | High | 15-20 |
| internal/dispatch | 26.2% | 75% | -48.8% | High | 20-25 |
| internal/persona | 19.0% | 75% | -56.0% | High | 10-15 |
| internal/loom | 8.4% | 75% | -66.6% | **CRITICAL** | 30-40 |
| internal/agent | 3.4% | 75% | -71.6% | **CRITICAL** | 20-25 |
| internal/api | 3.3% | 75% | -71.7% | **CRITICAL** | 25-30 |

## Critical Gaps (< 10% coverage)

### 1. internal/api (3.3%)
**Impact:** High - User-facing REST API
**Risk:** API bugs directly affect users
**Missing Tests:**
- Handler functions (GET/POST/PUT/DELETE endpoints)
- Request validation
- Error responses
- Authentication/authorization
- JSON serialization/deserialization

**Priority Tests:**
1. CRUD operations for beads, agents, projects
2. Error handling (400, 404, 500 responses)
3. Query parameter validation
4. Response format validation

### 2. internal/agent (3.4%)
**Impact:** Critical - Core agent lifecycle
**Risk:** Agent spawning/management failures
**Missing Tests:**
- Agent creation and initialization
- Agent state transitions (paused → idle → working)
- Agent assignment to projects
- Agent worker spawning
- Provider allocation

**Priority Tests:**
1. CreateAgent() function
2. SpawnAgentWorker() function
3. Agent status transitions
4. GetAvailableAgents() filtering

### 3. internal/loom (8.4%)
**Impact:** Critical - Main orchestrator
**Risk:** System-wide failures
**Missing Tests:**
- Loom initialization
- Component wiring
- Configuration loading
- System startup/shutdown
- Health checks

**Priority Tests:**
1. NewLoom() with various configs
2. Start() and Stop() lifecycle
3. Component initialization failures
4. Configuration validation

## Medium Gaps (10-30% coverage)

### 4. internal/persona (19.0%)
**Current Tests:** 3 tests (LoadQAEngineer, LoadProjectManager, ListPersonas)
**Missing Tests:**
- SKILL.md parsing with invalid frontmatter
- Missing required fields (name, description)
- Invalid YAML syntax
- Metadata extraction edge cases
- ClonePersona() function
- SavePersona() error handling

**Priority Tests:**
1. Parse errors (malformed YAML, missing fields)
2. Metadata validation
3. Clone and save operations

### 5. internal/dispatch (26.2%)
**Impact:** High - Task assignment logic
**Missing Tests:**
- Dispatcher initialization
- Agent registration
- Task assignment algorithms
- Priority handling
- Redispatch logic
- Loop detection integration

**Priority Tests:**
1. AssignTask with no agents
2. AssignTask with multiple candidates
3. Priority-based selection
4. Redispatch after failure

### 6. internal/worker (29.7%)
**Impact:** High - Task execution engine
**Missing Tests:**
- Worker lifecycle (start/stop)
- Task execution
- Error handling
- Timeout handling
- Action loop iterations
- Max iterations enforcement

**Priority Tests:**
1. Worker startup and shutdown
2. Task execution success/failure
3. Timeout handling
4. Max iteration limits

## Test Implementation Plan

### Phase 1: Critical Packages (Weeks 1-2)
**Goal:** Bring critical packages to 50%+ coverage

1. **internal/api tests** (25-30 tests)
   - CRUD handlers for all entities
   - Error responses
   - Validation logic

2. **internal/agent tests** (20-25 tests)
   - Agent lifecycle
   - State transitions
   - Worker management

3. **internal/loom tests** (30-40 tests)
   - Initialization
   - Configuration
   - Component wiring

### Phase 2: High-Priority Packages (Weeks 3-4)
**Goal:** Bring all packages to 60%+ coverage

4. **internal/dispatch tests** (20-25 tests)
5. **internal/worker tests** (15-20 tests)
6. **internal/persona tests** (10-15 tests)

### Phase 3: Threshold Achievement (Week 5)
**Goal:** Achieve 75%+ overall coverage

- Fill remaining gaps
- Add integration tests
- Add error path coverage
- Add edge case coverage

## Testing Patterns to Use

### 1. Table-Driven Tests
```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        want    Output
        wantErr bool
    }{
        {"success case", validInput, expectedOutput, false},
        {"error case", invalidInput, nil, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 2. Mock External Dependencies
- Event bus
- Providers (LLM APIs)
- Database
- Git operations
- HTTP clients

### 3. Test Fixtures
- Sample agents, beads, projects
- Mock provider responses
- Sample YAML configurations

## Success Metrics

- [ ] All packages ≥ 75% coverage
- [ ] Critical paths 100% covered
- [ ] Error paths tested
- [ ] Edge cases covered
- [ ] Integration tests added
- [ ] CI/CD enforces coverage threshold

## Continuous Maintenance

**Every PR must:**
1. Include tests for new code
2. Update tests for modified code
3. Run `make test-coverage` before commit
4. Pass 75% threshold check

**Weekly review:**
- Check coverage trends
- Identify new gaps
- Prioritize gap closure
