# Persona Testing and Validation

Ensure personas behave correctly before deployment.

## Overview

Comprehensive testing framework for personas including:
- Capability validation
- Response quality checks
- Performance testing
- Behavior verification
- Integration testing

## Testing Modes

### 1. Static Validation

Check persona definition without execution:

```bash
# Validate syntax
agenticorp validate-persona personas/backend-dev.md

# Check completeness
agenticorp check-persona --completeness personas/backend-dev.md

# Lint best practices
agenticorp lint-persona personas/backend-dev.md
```

**Checks:**
- Valid YAML/Markdown format
- Required fields present
- Capability format correct
- Instructions not empty
- No forbidden patterns

### 2. Interactive Testing

Test persona responses interactively:

```bash
# Start test session
agenticorp test-persona backend-dev

# Send test prompts
> Explain how to optimize a database query
> Generate tests for a REST API
> Review this code for security issues
```

**Evaluates:**
- Response relevance
- Technical accuracy
- Tone/style consistency
- Following instructions
- Capability demonstration

### 3. Automated Test Suites

Run predefined test scenarios:

```yaml
# personas/backend-dev/tests.yaml
test_suite:
  name: "Backend Developer Tests"
  persona: "backend-dev"
  
  tests:
    - name: "API Design"
      prompt: "Design a REST API for user management"
      expectations:
        - contains: ["GET", "POST", "PUT", "DELETE"]
        - mentions: ["authentication", "validation"]
        - format: "structured"
        - length: [100, 1000]
    
    - name: "Code Review"
      prompt: "Review this code:\n${code_sample}"
      expectations:
        - identifies_issues: true
        - suggests_improvements: true
        - explains_reasoning: true
    
    - name: "Performance Optimization"
      prompt: "This query is slow:\n${slow_query}"
      expectations:
        - analyzes_problem: true
        - suggests_solution: true
        - explains_impact: true
```

Run tests:

```bash
agenticorp run-tests personas/backend-dev/tests.yaml
```

### 4. Capability Verification

Test each claimed capability:

```bash
# Generate capability tests
agenticorp generate-capability-tests backend-dev

# Run capability verification
agenticorp verify-capabilities backend-dev
```

**For each capability**, generates and runs tests:
- "Design and implement RESTful APIs" → API design task
- "Optimize database queries" → Query optimization task
- "Write comprehensive tests" → Test generation task

### 5. Comparison Testing

Compare personas against each other:

```bash
# Compare two personas
agenticorp compare-personas backend-dev senior-backend-dev

# A/B test
agenticorp ab-test --persona1 backend-dev --persona2 alternative-backend --prompts test-set.txt
```

**Metrics:**
- Response quality
- Consistency
- Speed
- Accuracy
- User preference

## Test Results

### Validation Report

```
Persona: backend-developer
Status: ✓ PASSED

Static Checks:
  ✓ Valid format
  ✓ All required fields
  ✓ 10 capabilities defined
  ✓ Instructions comprehensive
  ✓ No forbidden patterns

Capability Tests (10/10 passed):
  ✓ Design APIs
  ✓ Optimize queries
  ✓ Write tests
  ✓ Review code
  ✓ Debug issues
  ...

Quality Metrics:
  Relevance:     9.2/10
  Accuracy:      8.8/10
  Completeness:  9.0/10
  Consistency:   9.5/10
  
Overall Score: 9.1/10 (Excellent)
```

### Failure Report

```
Persona: junior-dev
Status: ✗ FAILED

Issues Found:
  ✗ Capability "advanced architecture" not demonstrated
  ✗ Response too generic (3/5 tests)
  ! Slow response time (avg 12s, target <5s)
  
Recommendations:
  - Add more specific instructions
  - Include architecture examples
  - Reduce context window size
```

## Test Configuration

```yaml
# test-config.yaml
testing:
  timeout: 30s
  retries: 2
  parallel: 3
  
  quality_thresholds:
    relevance: 8.0
    accuracy: 7.5
    consistency: 8.5
  
  performance:
    max_response_time: 5s
    max_tokens: 2000
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Test Personas

on:
  pull_request:
    paths:
      - 'personas/**'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Validate Personas
        run: agenticorp validate-personas personas/
      - name: Run Tests
        run: agenticorp run-all-tests
      - name: Upload Results
        uses: actions/upload-artifact@v2
        with:
          name: test-results
          path: test-results/
```

## Best Practices

1. **Test Early**: Validate while developing
2. **Comprehensive Coverage**: Test all capabilities
3. **Real Scenarios**: Use actual use cases
4. **Automate**: CI/CD integration
5. **Version Tests**: Track with persona versions
6. **Document**: Record test rationale
7. **Iterate**: Refine based on results

## Validation Checklist

Before deploying a persona:

- [ ] Static validation passes
- [ ] All capabilities tested
- [ ] Quality score ≥ 8.0/10
- [ ] Response time < 5s
- [ ] No security issues
- [ ] Instructions clear
- [ ] Examples provided
- [ ] Edge cases handled
- [ ] Documented limitations
- [ ] Peer reviewed

## API

```bash
# Validate
POST /api/v1/personas/:id/validate
Response: { "valid": true, "issues": [] }

# Test
POST /api/v1/personas/:id/test
{
  "prompts": ["...", "..."],
  "expectations": {...}
}

# Results
GET /api/v1/personas/:id/test-results
```

## Metrics

### Quality Dimensions

- **Relevance**: Answers the question asked
- **Accuracy**: Technically correct
- **Completeness**: Covers all aspects
- **Clarity**: Easy to understand
- **Actionability**: Provides concrete guidance

### Performance Metrics

- Response time (target: <5s)
- Token usage (budget: <2000)
- Cache hit rate
- Error rate (target: <1%)

---

**Test personas thoroughly for reliable agents!** ✅
