package linter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLinterRunner(t *testing.T) {
	runner := NewLinterRunner("/tmp/test")
	if runner == nil {
		t.Fatal("Expected LinterRunner instance, got nil")
	}
	if runner.workDir != "/tmp/test" {
		t.Errorf("Expected workDir /tmp/test, got %s", runner.workDir)
	}
}

func TestLinterRunner_DetectFramework_Golangci(t *testing.T) {
	tmpDir := t.TempDir()
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	runner := NewLinterRunner(tmpDir)
	framework, err := runner.DetectFramework(tmpDir)
	if err != nil {
		t.Fatalf("DetectFramework failed: %v", err)
	}

	if framework != "golangci-lint" {
		t.Errorf("Expected framework 'golangci-lint', got '%s'", framework)
	}
}

func TestLinterRunner_DetectFramework_ESLint(t *testing.T) {
	tmpDir := t.TempDir()
	eslintrcPath := filepath.Join(tmpDir, ".eslintrc.json")
	if err := os.WriteFile(eslintrcPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create .eslintrc.json: %v", err)
	}

	runner := NewLinterRunner(tmpDir)
	framework, err := runner.DetectFramework(tmpDir)
	if err != nil {
		t.Fatalf("DetectFramework failed: %v", err)
	}

	if framework != "eslint" {
		t.Errorf("Expected framework 'eslint', got '%s'", framework)
	}
}

func TestLinterRunner_DetectFramework_Pylint(t *testing.T) {
	tmpDir := t.TempDir()
	pylintrcPath := filepath.Join(tmpDir, ".pylintrc")
	if err := os.WriteFile(pylintrcPath, []byte("[MASTER]"), 0644); err != nil {
		t.Fatalf("Failed to create .pylintrc: %v", err)
	}

	runner := NewLinterRunner(tmpDir)
	framework, err := runner.DetectFramework(tmpDir)
	if err != nil {
		t.Fatalf("DetectFramework failed: %v", err)
	}

	if framework != "pylint" {
		t.Errorf("Expected framework 'pylint', got '%s'", framework)
	}
}

func TestLinterRunner_DetectFramework_Unknown(t *testing.T) {
	tmpDir := t.TempDir()

	runner := NewLinterRunner(tmpDir)
	_, err := runner.DetectFramework(tmpDir)
	if err == nil {
		t.Error("Expected error for unknown framework, got nil")
	}

	if !strings.Contains(err.Error(), "could not detect linter framework") {
		t.Errorf("Expected 'could not detect' error, got: %v", err)
	}
}

func TestLinterRunner_BuildCommand_Golangci(t *testing.T) {
	runner := NewLinterRunner("/tmp/test")

	tests := []struct {
		name     string
		files    []string
		expected []string
	}{
		{
			name:     "No files",
			files:    nil,
			expected: []string{"golangci-lint", "run", "./..."},
		},
		{
			name:     "Specific files",
			files:    []string{"foo.go", "bar.go"},
			expected: []string{"golangci-lint", "run", "foo.go", "bar.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := runner.BuildCommand("golangci-lint", "/tmp/test", tt.files, "")
			if err != nil {
				t.Fatalf("BuildCommand failed: %v", err)
			}

			if len(cmd) != len(tt.expected) {
				t.Errorf("Expected command length %d, got %d", len(tt.expected), len(cmd))
			}

			for i, arg := range tt.expected {
				if i >= len(cmd) || cmd[i] != arg {
					t.Errorf("Expected arg[%d] = %s, got %s", i, arg, cmd[i])
				}
			}
		})
	}
}

func TestLinterRunner_BuildCommand_ESLint(t *testing.T) {
	runner := NewLinterRunner("/tmp/test")

	cmd, err := runner.BuildCommand("eslint", "/tmp/test", nil, "")
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	if cmd[0] != "eslint" {
		t.Errorf("Expected first arg 'eslint', got '%s'", cmd[0])
	}

	if !contains(cmd, "--format") {
		t.Error("Expected command to contain --format")
	}

	if !contains(cmd, "compact") {
		t.Error("Expected command to contain compact")
	}
}

func TestLinterRunner_BuildCommand_CustomCommand(t *testing.T) {
	runner := NewLinterRunner("/tmp/test")

	custom := "make lint"
	cmd, err := runner.BuildCommand("golangci-lint", "/tmp/test", nil, custom)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	expected := []string{"make", "lint"}
	if len(cmd) != len(expected) {
		t.Errorf("Expected command length %d, got %d", len(expected), len(cmd))
	}

	for i, arg := range expected {
		if cmd[i] != arg {
			t.Errorf("Expected arg[%d] = %s, got %s", i, arg, cmd[i])
		}
	}
}

func TestLinterRunner_BuildCommand_UnsupportedFramework(t *testing.T) {
	runner := NewLinterRunner("/tmp/test")

	_, err := runner.BuildCommand("unknown", "/tmp/test", nil, "")
	if err == nil {
		t.Error("Expected error for unsupported framework, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported linter framework") {
		t.Errorf("Expected 'unsupported framework' error, got: %v", err)
	}
}

func TestLinterRunner_ParseGolangciLintOutput(t *testing.T) {
	runner := NewLinterRunner("/tmp/test")

	output := `internal/foo/bar.go:10:2: unused variable 'x' (unused)
internal/baz/qux.go:25:1: func name will be used as baz.BazFoo by other packages, and that stutters; consider calling this Foo (golint)
`

	result, err := runner.parseGolangciLintOutput(output, 1)
	if err != nil {
		t.Fatalf("parseGolangciLintOutput failed: %v", err)
	}

	if result.Framework != "golangci-lint" {
		t.Errorf("Expected framework 'golangci-lint', got '%s'", result.Framework)
	}

	if result.Success {
		t.Error("Expected success=false for exit code 1")
	}

	if len(result.Violations) != 2 {
		t.Fatalf("Expected 2 violations, got %d", len(result.Violations))
	}

	// Check first violation
	v1 := result.Violations[0]
	if v1.File != "internal/foo/bar.go" {
		t.Errorf("Violation 0: expected file 'internal/foo/bar.go', got '%s'", v1.File)
	}
	if v1.Line != 10 {
		t.Errorf("Violation 0: expected line 10, got %d", v1.Line)
	}
	if v1.Column != 2 {
		t.Errorf("Violation 0: expected column 2, got %d", v1.Column)
	}
	if v1.Linter != "unused" {
		t.Errorf("Violation 0: expected linter 'unused', got '%s'", v1.Linter)
	}
}

func TestLinterRunner_ParseESLintOutput(t *testing.T) {
	runner := NewLinterRunner("/tmp/test")

	output := `src/app.js: line 10, col 5, Error - 'foo' is defined but never used (no-unused-vars)
src/utils.js: line 25, col 1, Warning - Unexpected console statement (no-console)
`

	result, err := runner.parseESLintOutput(output, 1)
	if err != nil {
		t.Fatalf("parseESLintOutput failed: %v", err)
	}

	if result.Framework != "eslint" {
		t.Errorf("Expected framework 'eslint', got '%s'", result.Framework)
	}

	if len(result.Violations) != 2 {
		t.Fatalf("Expected 2 violations, got %d", len(result.Violations))
	}

	// Check first violation
	v1 := result.Violations[0]
	if v1.File != "src/app.js" {
		t.Errorf("Violation 0: expected file 'src/app.js', got '%s'", v1.File)
	}
	if v1.Line != 10 {
		t.Errorf("Violation 0: expected line 10, got %d", v1.Line)
	}
	if v1.Severity != "error" {
		t.Errorf("Violation 0: expected severity 'error', got '%s'", v1.Severity)
	}
	if v1.Rule != "no-unused-vars" {
		t.Errorf("Violation 0: expected rule 'no-unused-vars', got '%s'", v1.Rule)
	}

	// Check second violation
	v2 := result.Violations[1]
	if v2.Severity != "warning" {
		t.Errorf("Violation 1: expected severity 'warning', got '%s'", v2.Severity)
	}
}

func TestLinterRunner_ParsePylintOutput(t *testing.T) {
	runner := NewLinterRunner("/tmp/test")

	output := `src/app.py:10:0: C0301: Line too long (line-too-long)
src/utils.py:25:4: E0602: Undefined variable 'foo' (undefined-variable)
`

	result, err := runner.parsePylintOutput(output, 1)
	if err != nil {
		t.Fatalf("parsePylintOutput failed: %v", err)
	}

	if result.Framework != "pylint" {
		t.Errorf("Expected framework 'pylint', got '%s'", result.Framework)
	}

	if len(result.Violations) != 2 {
		t.Fatalf("Expected 2 violations, got %d", len(result.Violations))
	}

	// Check first violation (convention)
	v1 := result.Violations[0]
	if v1.Severity != "info" {
		t.Errorf("Violation 0: expected severity 'info' for C code, got '%s'", v1.Severity)
	}

	// Check second violation (error)
	v2 := result.Violations[1]
	if v2.Severity != "error" {
		t.Errorf("Violation 1: expected severity 'error' for E code, got '%s'", v2.Severity)
	}
	if v2.Rule != "undefined-variable" {
		t.Errorf("Violation 1: expected rule 'undefined-variable', got '%s'", v2.Rule)
	}
}

func TestLinterRunner_Run_BasicExecution(t *testing.T) {
	if _, err := os.Stat("/bin/echo"); err != nil {
		t.Skip("Skipping test: /bin/echo not available")
	}

	tmpDir := t.TempDir()
	runner := NewLinterRunner(tmpDir)

	req := LintRequest{
		ProjectPath: tmpDir,
		LintCommand: "echo test output",
		Framework:   "generic",
		Timeout:     5 * time.Second,
	}

	ctx := context.Background()
	result, err := runner.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.RawOutput, "test output") {
		t.Errorf("Expected output to contain 'test output', got: %s", result.RawOutput)
	}
}

func TestLinterRunner_Run_Timeout(t *testing.T) {
	if _, err := os.Stat("/bin/sleep"); err != nil {
		t.Skip("Skipping test: /bin/sleep not available")
	}

	tmpDir := t.TempDir()
	runner := NewLinterRunner(tmpDir)

	req := LintRequest{
		ProjectPath: tmpDir,
		LintCommand: "sleep 5",
		Framework:   "generic",
		Timeout:     500 * time.Millisecond,
	}

	ctx := context.Background()
	result, err := runner.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !result.TimedOut {
		t.Errorf("Expected lint to timeout, but it didn't. Exit code: %d, Duration: %v", result.ExitCode, result.Duration)
	}

	if result.Duration >= 5*time.Second {
		t.Error("Lint should have been killed before completing 5 seconds")
	}
}

func TestLinterRunner_Run_DefaultTimeout(t *testing.T) {
	if _, err := os.Stat("/bin/echo"); err != nil {
		t.Skip("Skipping test: /bin/echo not available")
	}

	tmpDir := t.TempDir()
	runner := NewLinterRunner(tmpDir)

	req := LintRequest{
		ProjectPath: tmpDir,
		LintCommand: "echo test",
		Framework:   "generic",
		// Timeout not specified - should use default
	}

	ctx := context.Background()
	result, err := runner.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.TimedOut {
		t.Error("Command should not have timed out with default timeout")
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
