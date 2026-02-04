# Linter Integration

This document describes the linter integration system for AgentiCorp agents. Agents can run linters to check code quality, automatically fix violations, and iterate until code meets standards.

## Overview

The linter integration enables agents to:

1. **Execute linters** across multiple frameworks (golangci-lint, eslint, pylint)
2. **Parse violations** with file, line, rule, and severity information
3. **Fix issues** automatically by reading context and applying patches
4. **Iterate** in lint-fix-relint loops until code is clean
5. **Learn patterns** from recurring violations

## Supported Linters

### golangci-lint (Go)

**Auto-Detection:**
- Presence of `go.mod` file
- `*.go` files in project

**Command:**
```bash
golangci-lint run ./...
```

**Output Format:**
```
internal/foo/bar.go:10:2: unused variable 'x' (unused)
internal/baz/qux.go:25:1: func name will be used as baz.BazFoo (golint)
```

**Configuration:**
- Uses `.golangci.yml` or `.golangci.yaml` if present
- Respects project-level linter settings

### eslint (JavaScript/TypeScript)

**Auto-Detection:**
- `.eslintrc.js`, `.eslintrc.json`, or `.eslintrc.yml` files
- `eslint` in `package.json` dependencies

**Command:**
```bash
eslint --format compact .
```

**Output Format:**
```
src/app.js: line 10, col 5, Error - 'foo' is defined but never used (no-unused-vars)
src/utils.js: line 25, col 1, Warning - Unexpected console statement (no-console)
```

**Configuration:**
- Uses `.eslintrc.*` configuration files
- Supports `eslint:recommended` and custom rule sets

### pylint (Python)

**Auto-Detection:**
- `.pylintrc` or `pylintrc` files
- `*.py` files in project

**Command:**
```bash
pylint --output-format=text .
```

**Output Format:**
```
src/app.py:10:0: C0301: Line too long (line-too-long)
src/utils.py:25:4: E0602: Undefined variable 'foo' (undefined-variable)
```

**Configuration:**
- Uses `.pylintrc` if present
- Respects `pyproject.toml` and `setup.cfg` configurations

## Architecture

### LinterRunner Service

**Location:** `internal/linter/runner.go`

Core service for executing linters and parsing results.

```go
type LinterRunner struct {
    workDir string
}

type LintRequest struct {
    ProjectPath  string
    LintCommand  string
    Framework    string
    Files        []string
    Environment  map[string]string
    Timeout      time.Duration
}

type LintResult struct {
    Framework  string
    Success    bool
    ExitCode   int
    Violations []Violation
    RawOutput  string
    Duration   time.Duration
    TimedOut   bool
    Error      string
}

type Violation struct {
    File     string // File path
    Line     int    // Line number
    Column   int    // Column number
    Rule     string // Rule ID
    Severity string // "error", "warning", "info"
    Message  string // Human-readable message
    Linter   string // Specific linter name
}
```

**Key Methods:**
- `Run(ctx, req)` - Execute linter and return parsed results
- `DetectFramework(projectPath)` - Auto-detect linter from project
- `BuildCommand(framework, projectPath, files, customCommand)` - Construct command
- `parseOutput(framework, output, exitCode)` - Parse framework-specific output

### Agent Action

**Location:** `internal/actions/schema.go`, `internal/actions/router.go`

Agents use `ActionRunLinter` to execute linters.

```json
{
  "type": "run_linter",
  "files": ["internal/*.go"],
  "framework": "golangci-lint",
  "timeout_seconds": 300
}
```

**Response:**
```json
{
  "action_type": "run_linter",
  "status": "executed",
  "message": "linter executed",
  "metadata": {
    "framework": "golangci-lint",
    "success": false,
    "exit_code": 1,
    "violation_count": 2,
    "violations": [
      {
        "file": "internal/foo.go",
        "line": 10,
        "column": 2,
        "rule": "unused",
        "severity": "error",
        "message": "unused variable 'x'",
        "linter": "unused"
      }
    ]
  }
}
```

## Usage Patterns

### Pattern 1: Basic Lint Check

```json
{
  "actions": [
    {"type": "run_linter"}
  ],
  "notes": "Checking code quality"
}
```

### Pattern 2: Lint Specific Files

```json
{
  "actions": [
    {"type": "run_linter", "files": ["internal/api/*.go"]}
  ],
  "notes": "Linting modified API files"
}
```

### Pattern 3: Lint-Fix-Relint Loop

```json
{
  "actions": [
    {"type": "run_linter"},
    {"type": "read_file", "path": "internal/foo.go"},
    {"type": "edit_code", "path": "internal/foo.go", "patch": "..."},
    {"type": "run_linter", "files": ["internal/foo.go"]}
  ],
  "notes": "Fixed unused variable violation"
}
```

### Pattern 4: Complete Quality Check

```json
{
  "actions": [
    {"type": "run_tests"},
    {"type": "run_linter"},
    {"type": "git_status"}
  ],
  "notes": "Verifying tests pass and code quality checks"
}
```

### Pattern 5: Iterative Fixing

```json
{
  "actions": [
    {"type": "run_linter", "framework": "golangci-lint"},
    // If violations found:
    {"type": "read_file", "path": "internal/parser.go"},
    {"type": "edit_code", "path": "internal/parser.go", "patch": "..."},
    {"type": "run_linter", "files": ["internal/parser.go"]},
    // If still violations:
    {"type": "edit_code", "path": "internal/parser.go", "patch": "..."},
    {"type": "run_linter", "files": ["internal/parser.go"]}
  ],
  "notes": "Iteratively fixed all linter violations"
}
```

## Violation Severity Levels

Linters report violations at different severity levels:

- **error**: Critical issues that must be fixed (e.g., undefined variables, type errors)
- **warning**: Potential problems that should be addressed (e.g., unused imports, deprecated APIs)
- **info**: Style and convention suggestions (e.g., naming conventions, formatting)

**Priority Strategy:**

1. Fix all **errors** first (blocks merge/deploy)
2. Address **warnings** that impact functionality
3. Apply **info** suggestions for consistency

## Common Violation Categories

### 1. Unused Code

**Examples:**
- Unused variables: `unused variable 'x'`
- Unused imports: `imported and not used: "fmt"`
- Unused functions: `func Add is unused`

**Fix Strategy:**
- Remove unused declarations
- Or add `_ = x` to explicitly ignore

### 2. Code Style

**Examples:**
- Line too long: `Line exceeds 120 characters`
- Missing docstring: `exported function Foo should have comment`
- Naming conventions: `var fooBar should be fooBar`

**Fix Strategy:**
- Reformat code to meet style guide
- Add missing documentation
- Rename identifiers

### 3. Logic Errors

**Examples:**
- Nil pointer: `possible nil pointer dereference`
- Type mismatch: `cannot use string as int`
- Unreachable code: `unreachable code`

**Fix Strategy:**
- Add nil checks
- Fix type conversions
- Remove dead code

### 4. Security Issues

**Examples:**
- SQL injection: `potential SQL injection`
- Path traversal: `potential directory traversal`
- Weak crypto: `weak cryptographic primitive`

**Fix Strategy:**
- Use parameterized queries
- Validate and sanitize paths
- Use strong crypto libraries

## Timeout Configuration

```go
const (
    DefaultLintTimeout = 5 * time.Minute
    MaxLintTimeout     = 15 * time.Minute
)
```

**Recommendations:**
- **Small projects** (<1000 files): 1-2 minutes
- **Medium projects** (1000-5000 files): 2-5 minutes
- **Large projects** (>5000 files): 5-15 minutes

**Custom Timeout:**
```json
{
  "type": "run_linter",
  "timeout_seconds": 600
}
```

## Error Handling

### Linter Not Found

```json
{
  "status": "error",
  "message": "failed to execute: exec: \"golangci-lint\": executable file not found"
}
```

**Solution:** Install linter or use custom command

### Timeout

```json
{
  "status": "executed",
  "metadata": {
    "timed_out": true,
    "duration": "5m0s"
  }
}
```

**Solution:** Increase timeout or lint specific files

### Parse Error

```json
{
  "status": "executed",
  "metadata": {
    "error": "failed to parse output: unexpected format"
  }
}
```

**Solution:** Check linter version or use custom output format

## Integration with Conversation Context

Linter results are added to the agent's conversation history:

```
Linter execution completed:
Framework: golangci-lint
Status: FAILED (2 violations)
Duration: 2.3s

Violations:
1. internal/foo.go:10:2 (unused)
   unused variable 'x'

2. internal/bar.go:25:1 (golint)
   exported func Bar should have comment
```

This enables agents to:
- Understand the specific violations
- Read relevant source files
- Apply targeted fixes
- Re-run linter to verify

## Best Practices

### 1. Run Linter Before Committing

```json
{
  "actions": [
    {"type": "run_tests"},
    {"type": "run_linter"},
    {"type": "git_status"}
  ]
}
```

### 2. Lint Only Changed Files

```json
{
  "actions": [
    {"type": "git_diff"},
    {"type": "run_linter", "files": ["path/to/changed.go"]}
  ]
}
```

### 3. Fix High Severity First

```json
{
  "actions": [
    {"type": "run_linter"},
    // Parse violations, prioritize by severity
    // Fix errors first, then warnings, then info
  ]
}
```

### 4. Use Specific Linters for Specific Tasks

```json
{
  "type": "run_linter",
  "framework": "golangci-lint",
  "files": ["internal/security/*.go"]
}
```

### 5. Iterate Until Clean

```
while violations exist:
    run_linter()
    identify_violation()
    read_context()
    apply_fix()
    run_linter(specific_files)
```

## Performance Optimization

### Caching

- Linter runners cache framework detection results
- Parsed configurations are reused across runs

### Incremental Linting

```json
{
  "type": "run_linter",
  "files": ["internal/modified_file.go"]
}
```

Only lints changed files instead of entire project.

### Parallel Execution

Future enhancement: Run multiple linters in parallel

```json
{
  "actions": [
    {"type": "run_linter", "framework": "golangci-lint"},
    {"type": "run_linter", "framework": "staticcheck"}
  ]
}
```

## Configuration Files

### golangci-lint (.golangci.yml)

```yaml
run:
  timeout: 5m
  tests: true

linters:
  enable:
    - unused
    - staticcheck
    - gosimple
    - ineffassign

linters-settings:
  govet:
    check-shadowing: true
```

### eslint (.eslintrc.json)

```json
{
  "extends": ["eslint:recommended"],
  "rules": {
    "no-unused-vars": "error",
    "no-console": "warn",
    "semi": ["error", "always"]
  }
}
```

### pylint (.pylintrc)

```ini
[MESSAGES CONTROL]
disable=C0111,R0903

[FORMAT]
max-line-length=120
```

## Testing

**Unit Tests:** 15 tests covering:
- Framework detection (Go, JavaScript, Python)
- Command building
- Output parsing (golangci-lint, eslint, pylint)
- Timeout handling
- Error scenarios

**Integration Tests:** Real linter execution on sample projects

## Related Documentation

- [Agent Actions Reference](AGENT_ACTIONS.md) - Complete action schema
- [Test Execution Design](TEST_EXECUTION_DESIGN.md) - Test runner architecture
- [Conversation Architecture](CONVERSATION_ARCHITECTURE.md) - Multi-turn workflows

## Implementation

The linter integration is implemented in:

- `internal/linter/runner.go` - Core linter execution service
- `internal/linter/runner_test.go` - Unit and integration tests
- `internal/actions/schema.go` - ActionRunLinter definition
- `internal/actions/router.go` - Action routing
- `internal/actions/linterrunner_adapter.go` - Actions interface adapter

## Future Enhancements

### Phase 2

- **Auto-fix**: Automatically apply fixes for common violations
- **Violation history**: Track recurring issues over time
- **Custom rules**: Project-specific linting rules
- **Parallel execution**: Run multiple linters concurrently

### Phase 3

- **ML-powered fixes**: Learn from historical fixes to suggest better solutions
- **Violation clustering**: Group related violations for batch fixing
- **Performance profiling**: Track linter performance and optimize
- **IDE integration**: Real-time linting feedback in development

## Contributing

When adding support for a new linter:

1. Add detection logic to `DetectFramework()`
2. Add command building to `BuildCommand()`
3. Implement parser in `parseOutput()`
4. Add unit tests for the new framework
5. Add integration tests with real linter
6. Update this documentation

## License

See [LICENSE](../LICENSE) for details.
