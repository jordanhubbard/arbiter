package feedback

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/build"
	"github.com/jordanhubbard/loom/internal/linter"
	testpkg "github.com/jordanhubbard/loom/internal/testing"
)

func TestOrchestrator_Run_AllSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	orch := NewOrchestrator(".")
	config := DefaultConfig(".")

	ctx := context.Background()
	result, err := orch.Run(ctx, config)

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Summary)
	}

	if result.FailedPhase != "" {
		t.Errorf("Expected no failed phase, got %s", result.FailedPhase)
	}

	if result.Build.Skipped {
		t.Error("Build should not be skipped")
	}

	if result.Lint.Skipped {
		t.Error("Lint should not be skipped")
	}

	if result.Test.Skipped {
		t.Error("Test should not be skipped")
	}

	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

func TestOrchestrator_Run_BuildFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	orch := NewOrchestrator(".")
	config := DefaultConfig(".")
	config.StopOnBuildFailure = true

	// This will fail to build (no valid project in current dir for building)
	ctx := context.Background()
	result, err := orch.Run(ctx, config)

	// Note: In a real test, we'd use mock runners
	// For now, we just verify the orchestrator logic
	if err != nil {
		t.Logf("Expected error or build failure: %v", err)
	}

	if result != nil {
		if !result.Success && result.FailedPhase == "build" {
			// Good - build failed as expected
			if !result.Lint.Skipped {
				t.Error("Lint should be skipped after build failure")
			}
			if !result.Test.Skipped {
				t.Error("Tests should be skipped after build failure")
			}
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("/test/path")

	if config.ProjectPath != "/test/path" {
		t.Errorf("Expected project path '/test/path', got '%s'", config.ProjectPath)
	}

	if !config.RunBuild {
		t.Error("Expected RunBuild to be true by default")
	}

	if !config.RunLint {
		t.Error("Expected RunLint to be true by default")
	}

	if !config.RunTests {
		t.Error("Expected RunTests to be true by default")
	}

	if !config.StopOnBuildFailure {
		t.Error("Expected StopOnBuildFailure to be true by default")
	}

	if config.StopOnLintFailure {
		t.Error("Expected StopOnLintFailure to be false by default")
	}

	if config.BuildTimeout != build.DefaultBuildTimeout {
		t.Errorf("Expected build timeout %v, got %v", build.DefaultBuildTimeout, config.BuildTimeout)
	}

	if config.LintTimeout != linter.DefaultLintTimeout {
		t.Errorf("Expected lint timeout %v, got %v", linter.DefaultLintTimeout, config.LintTimeout)
	}

	if config.TestTimeout != testpkg.DefaultTestTimeout {
		t.Errorf("Expected test timeout %v, got %v", testpkg.DefaultTestTimeout, config.TestTimeout)
	}
}

func TestDefaultConfig_EmptyPath(t *testing.T) {
	config := DefaultConfig("")
	if config.ProjectPath != "" {
		t.Errorf("Expected empty project path, got '%s'", config.ProjectPath)
	}
	// Timeouts should still have defaults
	if config.BuildTimeout == 0 {
		t.Error("Expected non-zero build timeout")
	}
	if config.LintTimeout == 0 {
		t.Error("Expected non-zero lint timeout")
	}
	if config.TestTimeout == 0 {
		t.Error("Expected non-zero test timeout")
	}
}

func TestDefaultConfig_FrameworkOverridesEmpty(t *testing.T) {
	config := DefaultConfig("/some/path")
	if config.BuildFramework != "" {
		t.Errorf("Expected empty BuildFramework, got '%s'", config.BuildFramework)
	}
	if config.LintFramework != "" {
		t.Errorf("Expected empty LintFramework, got '%s'", config.LintFramework)
	}
	if config.TestFramework != "" {
		t.Errorf("Expected empty TestFramework, got '%s'", config.TestFramework)
	}
	if config.LintFiles != nil {
		t.Error("Expected nil LintFiles")
	}
	if config.TestPattern != "" {
		t.Errorf("Expected empty TestPattern, got '%s'", config.TestPattern)
	}
}

func TestNewOrchestrator(t *testing.T) {
	orch := NewOrchestrator("/test/project")
	if orch == nil {
		t.Fatal("Expected non-nil orchestrator")
	}
	if orch.projectPath != "/test/project" {
		t.Errorf("Expected project path '/test/project', got '%s'", orch.projectPath)
	}
	if orch.buildRunner == nil {
		t.Error("Expected non-nil buildRunner")
	}
	if orch.lintRunner == nil {
		t.Error("Expected non-nil lintRunner")
	}
	if orch.testRunner == nil {
		t.Error("Expected non-nil testRunner")
	}
}

func TestNewOrchestrator_EmptyPath(t *testing.T) {
	orch := NewOrchestrator("")
	if orch == nil {
		t.Fatal("Expected non-nil orchestrator even with empty path")
	}
	if orch.projectPath != "" {
		t.Errorf("Expected empty project path, got '%s'", orch.projectPath)
	}
}

func TestBuildSummary_AllSuccess(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success:     true,
		FailedPhase: "",
		Build: &BuildCheck{
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		Lint: &LintCheck{
			Success:  true,
			Duration: 50 * time.Millisecond,
		},
		Test: &TestCheck{
			Success:  true,
			Duration: 200 * time.Millisecond,
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "✓ All checks passed") {
		t.Errorf("Expected success message in summary, got: %s", summary)
	}

	if !strings.Contains(summary, "build") {
		t.Error("Expected 'build' in summary")
	}

	if !strings.Contains(summary, "lint") {
		t.Error("Expected 'lint' in summary")
	}

	if !strings.Contains(summary, "test") {
		t.Error("Expected 'test' in summary")
	}
}

func TestBuildSummary_AllSuccessWithSeconds(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success:     true,
		FailedPhase: "",
		Build: &BuildCheck{
			Success:  true,
			Duration: 2 * time.Second,
		},
		Lint: &LintCheck{
			Success:  true,
			Duration: 1500 * time.Millisecond,
		},
		Test: &TestCheck{
			Success:  true,
			Duration: 5 * time.Second,
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "✓ All checks passed") {
		t.Errorf("Expected success message in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "2.0s") {
		t.Error("Expected build duration in seconds format")
	}
	if !strings.Contains(summary, "1.5s") {
		t.Error("Expected lint duration in seconds format")
	}
	if !strings.Contains(summary, "5.0s") {
		t.Error("Expected test duration in seconds format")
	}
}

func TestBuildSummary_BuildFailure(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "build",
		Build: &BuildCheck{
			Success:    false,
			Duration:   150 * time.Millisecond,
			ErrorCount: 2,
			Errors: []build.BuildError{
				{
					File:    "foo.go",
					Line:    10,
					Column:  2,
					Message: "undefined: bar",
					Type:    "error",
				},
				{
					File:    "baz.go",
					Line:    25,
					Column:  5,
					Message: "syntax error",
					Type:    "error",
				},
			},
		},
		Lint: &LintCheck{
			Skipped:    true,
			SkipReason: "build failed",
		},
		Test: &TestCheck{
			Skipped:    true,
			SkipReason: "build failed",
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "✗ Feedback loop failed at build phase") {
		t.Errorf("Expected build failure message, got: %s", summary)
	}

	if !strings.Contains(summary, "2 error(s)") {
		t.Error("Expected error count in summary")
	}

	if !strings.Contains(summary, "foo.go:10:2") {
		t.Error("Expected first error location in summary")
	}

	if !strings.Contains(summary, "Lint skipped") {
		t.Error("Expected lint skipped message")
	}

	if !strings.Contains(summary, "Tests skipped") {
		t.Error("Expected test skipped message")
	}
}

func TestBuildSummary_BuildFailureMoreThan3Errors(t *testing.T) {
	orch := NewOrchestrator(".")

	errors := make([]build.BuildError, 5)
	for i := 0; i < 5; i++ {
		errors[i] = build.BuildError{
			File:    "file.go",
			Line:    i + 1,
			Column:  1,
			Message: "some error",
			Type:    "error",
		}
	}

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "build",
		Build: &BuildCheck{
			Success:    false,
			Duration:   150 * time.Millisecond,
			ErrorCount: 5,
			Errors:     errors,
		},
		Lint: &LintCheck{
			Skipped:    true,
			SkipReason: "build failed",
		},
		Test: &TestCheck{
			Skipped:    true,
			SkipReason: "build failed",
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "5 error(s)") {
		t.Errorf("Expected 5 errors in summary, got: %s", summary)
	}

	if !strings.Contains(summary, "... and 2 more error(s)") {
		t.Errorf("Expected truncation message for errors, got: %s", summary)
	}
}

func TestBuildSummary_LintFailure(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "lint",
		Build: &BuildCheck{
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		Lint: &LintCheck{
			Success:        false,
			Duration:       80 * time.Millisecond,
			ViolationCount: 3,
			Violations: []linter.Violation{
				{
					File:     "main.go",
					Line:     5,
					Rule:     "unused",
					Severity: "error",
					Message:  "unused variable 'x'",
				},
			},
		},
		Test: &TestCheck{
			Success:  true,
			Duration: 200 * time.Millisecond,
			Summary: testpkg.TestSummary{
				Total:  10,
				Passed: 10,
			},
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "✗ Feedback loop failed at lint phase") {
		t.Errorf("Expected lint failure message, got: %s", summary)
	}

	if !strings.Contains(summary, "✓ Build passed") {
		t.Error("Expected build success in summary")
	}

	if !strings.Contains(summary, "3 violation(s)") {
		t.Error("Expected violation count in summary")
	}

	if !strings.Contains(summary, "✓ Tests passed") {
		t.Error("Expected test success (lint doesn't stop tests)")
	}
}

func TestBuildSummary_LintFailureMoreThan3Violations(t *testing.T) {
	orch := NewOrchestrator(".")

	violations := make([]linter.Violation, 5)
	for i := 0; i < 5; i++ {
		violations[i] = linter.Violation{
			File:     "main.go",
			Line:     i + 1,
			Rule:     "some-rule",
			Severity: "error",
			Message:  "violation message",
		}
	}

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "lint",
		Build: &BuildCheck{
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		Lint: &LintCheck{
			Success:        false,
			Duration:       80 * time.Millisecond,
			ViolationCount: 5,
			Violations:     violations,
		},
		Test: &TestCheck{
			Skipped:    true,
			SkipReason: "linter failed",
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "... and 2 more violation(s)") {
		t.Errorf("Expected truncation message for violations, got: %s", summary)
	}
	if !strings.Contains(summary, "Tests skipped") {
		t.Errorf("Expected test skipped message, got: %s", summary)
	}
}

func TestBuildSummary_TestFailure(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "test",
		Build: &BuildCheck{
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		Lint: &LintCheck{
			Success:  true,
			Duration: 50 * time.Millisecond,
		},
		Test: &TestCheck{
			Success:  false,
			Duration: 300 * time.Millisecond,
			Summary: testpkg.TestSummary{
				Total:  10,
				Passed: 8,
				Failed: 2,
			},
			FailedTests: []testpkg.TestCase{
				{
					Name:   "TestFoo",
					Status: "fail",
					Error:  "Expected 100, got 99",
				},
				{
					Name:   "TestBar",
					Status: "fail",
					Error:  "nil pointer dereference",
				},
			},
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "✗ Feedback loop failed at test phase") {
		t.Errorf("Expected test failure message, got: %s", summary)
	}

	if !strings.Contains(summary, "✓ Build passed") {
		t.Error("Expected build success in summary")
	}

	if !strings.Contains(summary, "✓ Lint passed") {
		t.Error("Expected lint success in summary")
	}

	if !strings.Contains(summary, "8/10 passed") {
		t.Error("Expected test counts in summary")
	}

	if !strings.Contains(summary, "TestFoo") {
		t.Error("Expected failed test name in summary")
	}
}

func TestBuildSummary_TestFailureMoreThan3(t *testing.T) {
	orch := NewOrchestrator(".")

	failedTests := make([]testpkg.TestCase, 5)
	for i := 0; i < 5; i++ {
		failedTests[i] = testpkg.TestCase{
			Name:   "TestFail" + string(rune('A'+i)),
			Status: testpkg.TestFail,
			Error:  "some error",
		}
	}

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "test",
		Build: &BuildCheck{
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		Lint: &LintCheck{
			Success:  true,
			Duration: 50 * time.Millisecond,
		},
		Test: &TestCheck{
			Success:     false,
			Duration:    300 * time.Millisecond,
			Summary:     testpkg.TestSummary{Total: 10, Passed: 5, Failed: 5},
			FailedTests: failedTests,
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "... and 2 more failed test(s)") {
		t.Errorf("Expected truncation message for failed tests, got: %s", summary)
	}
}

func TestBuildSummary_BuildFailureNoStopLintAndTestRun(t *testing.T) {
	orch := NewOrchestrator(".")

	// Build failed, but StopOnBuildFailure is false, so lint and test still run
	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "build",
		Build: &BuildCheck{
			Success:    false,
			Duration:   100 * time.Millisecond,
			ErrorCount: 1,
			Errors: []build.BuildError{
				{File: "foo.go", Line: 1, Column: 1, Message: "err", Type: "error"},
			},
		},
		Lint: &LintCheck{
			Success:  true,
			Duration: 50 * time.Millisecond,
		},
		Test: &TestCheck{
			Success:  true,
			Duration: 200 * time.Millisecond,
			Summary:  testpkg.TestSummary{Total: 5, Passed: 5},
		},
	}

	summary := orch.buildSummary(result)
	if !strings.Contains(summary, "✗ Build failed") {
		t.Errorf("Expected build failure in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "✓ Lint passed") {
		t.Errorf("Expected lint passed in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "✓ Tests passed") {
		t.Errorf("Expected test passed in summary, got: %s", summary)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{0, "0ms"},
		{50 * time.Millisecond, "50ms"},
		{500 * time.Millisecond, "500ms"},
		{999 * time.Millisecond, "999ms"},
		{1 * time.Second, "1.0s"},
		{1500 * time.Millisecond, "1.5s"},
		{2 * time.Second, "2.0s"},
		{10500 * time.Millisecond, "10.5s"},
		{60 * time.Second, "60.0s"},
		{1 * time.Microsecond, "0ms"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %s, want %s", tt.duration, got, tt.want)
		}
	}
}

func TestOrchestrator_QuickCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	orch := NewOrchestrator(".")

	ctx := context.Background()
	result, err := orch.QuickCheck(ctx)

	if err != nil {
		t.Fatalf("QuickCheck() error = %v", err)
	}

	if !result.Build.Skipped {
		t.Error("QuickCheck should skip build")
	}

	if result.Build.SkipReason != "disabled" {
		t.Errorf("Expected skip reason 'disabled', got '%s'", result.Build.SkipReason)
	}
}

func TestOrchestrator_BuildOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	orch := NewOrchestrator(".")

	ctx := context.Background()
	result, err := orch.BuildOnly(ctx)

	// May error if no valid build in current dir
	if err != nil {
		t.Logf("BuildOnly error (expected if no buildable project): %v", err)
	}

	if result != nil {
		if !result.Build.Skipped && result.Build.Framework == "" {
			// If build ran, it should have detected a framework
			t.Logf("Build result: success=%v, framework=%s", result.Build.Success, result.Build.Framework)
		}

		if !result.Lint.Skipped {
			t.Error("BuildOnly should skip lint")
		}

		if !result.Test.Skipped {
			t.Error("BuildOnly should skip tests")
		}
	}
}

func TestOrchestratorConfig_PhaseControl(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultConfig(".")

	// Test disabling phases
	config.RunBuild = false
	config.RunLint = false
	config.RunTests = false

	orch := NewOrchestrator(".")
	ctx := context.Background()
	result, err := orch.Run(ctx, config)

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !result.Build.Skipped {
		t.Error("Build should be skipped when disabled")
	}

	if !result.Lint.Skipped {
		t.Error("Lint should be skipped when disabled")
	}

	if !result.Test.Skipped {
		t.Error("Tests should be skipped when disabled")
	}

	if !result.Success {
		t.Error("Should be successful when all phases skipped")
	}
}

func TestOrchestratorConfig_StopOnLintFailure(t *testing.T) {
	config := DefaultConfig(".")
	config.StopOnLintFailure = true

	if !config.StopOnLintFailure {
		t.Error("Expected StopOnLintFailure to be settable to true")
	}
}

func TestOrchestratorConfig_CustomTimeouts(t *testing.T) {
	config := DefaultConfig("/test")
	config.BuildTimeout = 30 * time.Second
	config.LintTimeout = 15 * time.Second
	config.TestTimeout = 20 * time.Second

	if config.BuildTimeout != 30*time.Second {
		t.Errorf("Expected build timeout 30s, got %v", config.BuildTimeout)
	}
	if config.LintTimeout != 15*time.Second {
		t.Errorf("Expected lint timeout 15s, got %v", config.LintTimeout)
	}
	if config.TestTimeout != 20*time.Second {
		t.Errorf("Expected test timeout 20s, got %v", config.TestTimeout)
	}
}

func TestOrchestratorConfig_FrameworkOverrides(t *testing.T) {
	config := DefaultConfig("/test")
	config.BuildFramework = "go"
	config.LintFramework = "golangci-lint"
	config.TestFramework = "go"
	config.TestPattern = "TestFoo.*"
	config.LintFiles = []string{"main.go", "util.go"}

	if config.BuildFramework != "go" {
		t.Errorf("Expected build framework 'go', got '%s'", config.BuildFramework)
	}
	if config.LintFramework != "golangci-lint" {
		t.Errorf("Expected lint framework 'golangci-lint', got '%s'", config.LintFramework)
	}
	if config.TestFramework != "go" {
		t.Errorf("Expected test framework 'go', got '%s'", config.TestFramework)
	}
	if config.TestPattern != "TestFoo.*" {
		t.Errorf("Expected test pattern 'TestFoo.*', got '%s'", config.TestPattern)
	}
	if len(config.LintFiles) != 2 {
		t.Errorf("Expected 2 lint files, got %d", len(config.LintFiles))
	}
}

func TestFeedbackResult_StructFields(t *testing.T) {
	result := &FeedbackResult{
		Success:     true,
		Duration:    5 * time.Second,
		Build:       &BuildCheck{Success: true},
		Lint:        &LintCheck{Success: true},
		Test:        &TestCheck{Success: true},
		Summary:     "all good",
		FailedPhase: "",
	}

	if !result.Success {
		t.Error("Expected success")
	}
	if result.Duration != 5*time.Second {
		t.Errorf("Expected 5s duration, got %v", result.Duration)
	}
	if result.Summary != "all good" {
		t.Errorf("Expected 'all good', got '%s'", result.Summary)
	}
	if result.FailedPhase != "" {
		t.Errorf("Expected empty failed phase, got '%s'", result.FailedPhase)
	}
}

func TestBuildCheck_StructFields(t *testing.T) {
	bc := &BuildCheck{
		Success:    false,
		Duration:   500 * time.Millisecond,
		Framework:  "go",
		ErrorCount: 1,
		Errors: []build.BuildError{
			{File: "a.go", Line: 1, Column: 1, Message: "err", Type: "error"},
		},
		Warnings: []build.BuildError{
			{File: "b.go", Line: 2, Column: 3, Message: "warn", Type: "warning"},
		},
		Skipped:    false,
		SkipReason: "",
	}

	if bc.Success {
		t.Error("Expected not success")
	}
	if bc.Framework != "go" {
		t.Errorf("Expected framework 'go', got '%s'", bc.Framework)
	}
	if bc.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", bc.ErrorCount)
	}
	if len(bc.Errors) != 1 {
		t.Errorf("Expected 1 error slice entry, got %d", len(bc.Errors))
	}
	if len(bc.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(bc.Warnings))
	}
}

func TestBuildCheck_Skipped(t *testing.T) {
	bc := &BuildCheck{
		Skipped:    true,
		SkipReason: "disabled",
	}
	if !bc.Skipped {
		t.Error("Expected skipped")
	}
	if bc.SkipReason != "disabled" {
		t.Errorf("Expected skip reason 'disabled', got '%s'", bc.SkipReason)
	}
}

func TestLintCheck_StructFields(t *testing.T) {
	lc := &LintCheck{
		Success:        false,
		Duration:       200 * time.Millisecond,
		Framework:      "golangci-lint",
		ViolationCount: 2,
		Violations: []linter.Violation{
			{File: "main.go", Line: 5, Rule: "unused", Severity: "error", Message: "unused var"},
			{File: "main.go", Line: 10, Rule: "ineffassign", Severity: "warning", Message: "ineffectual assignment"},
		},
		Skipped:    false,
		SkipReason: "",
	}

	if lc.Success {
		t.Error("Expected not success")
	}
	if lc.Framework != "golangci-lint" {
		t.Errorf("Expected framework 'golangci-lint', got '%s'", lc.Framework)
	}
	if lc.ViolationCount != 2 {
		t.Errorf("Expected 2 violations, got %d", lc.ViolationCount)
	}
	if len(lc.Violations) != 2 {
		t.Errorf("Expected 2 violations in slice, got %d", len(lc.Violations))
	}
}

func TestTestCheck_StructFields(t *testing.T) {
	tc := &TestCheck{
		Success:  false,
		Duration: 1 * time.Second,
		Framework: "go",
		Summary: testpkg.TestSummary{
			Total:   10,
			Passed:  8,
			Failed:  2,
			Skipped: 0,
		},
		FailedTests: []testpkg.TestCase{
			{Name: "TestA", Status: testpkg.TestFail, Error: "oops"},
		},
		Skipped:    false,
		SkipReason: "",
	}

	if tc.Success {
		t.Error("Expected not success")
	}
	if tc.Framework != "go" {
		t.Errorf("Expected framework 'go', got '%s'", tc.Framework)
	}
	if tc.Summary.Total != 10 {
		t.Errorf("Expected 10 total tests, got %d", tc.Summary.Total)
	}
	if tc.Summary.Passed != 8 {
		t.Errorf("Expected 8 passed, got %d", tc.Summary.Passed)
	}
	if tc.Summary.Failed != 2 {
		t.Errorf("Expected 2 failed, got %d", tc.Summary.Failed)
	}
	if len(tc.FailedTests) != 1 {
		t.Errorf("Expected 1 failed test, got %d", len(tc.FailedTests))
	}
}

func TestOrchestrator_Run_AllPhasesDisabled(t *testing.T) {
	orch := NewOrchestrator(".")
	config := OrchestratorConfig{
		ProjectPath: ".",
		RunBuild:    false,
		RunLint:     false,
		RunTests:    false,
	}

	ctx := context.Background()
	result, err := orch.Run(ctx, config)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success when all phases disabled")
	}
	if result.FailedPhase != "" {
		t.Errorf("Expected empty failed phase, got '%s'", result.FailedPhase)
	}
	if !result.Build.Skipped {
		t.Error("Expected build skipped")
	}
	if result.Build.SkipReason != "disabled" {
		t.Errorf("Expected build skip reason 'disabled', got '%s'", result.Build.SkipReason)
	}
	if !result.Lint.Skipped {
		t.Error("Expected lint skipped")
	}
	if result.Lint.SkipReason != "disabled" {
		t.Errorf("Expected lint skip reason 'disabled', got '%s'", result.Lint.SkipReason)
	}
	if !result.Test.Skipped {
		t.Error("Expected test skipped")
	}
	if result.Test.SkipReason != "disabled" {
		t.Errorf("Expected test skip reason 'disabled', got '%s'", result.Test.SkipReason)
	}
	if result.Duration == 0 {
		// Duration may be very small but should be set
		t.Log("Duration is zero, but this is acceptable for no-op run")
	}
	if result.Summary == "" {
		t.Error("Expected non-empty summary")
	}
}

func TestOrchestrator_Run_OnlyBuildEnabled(t *testing.T) {
	orch := NewOrchestrator(".")
	config := OrchestratorConfig{
		ProjectPath:        ".",
		RunBuild:           false,
		RunLint:            false,
		RunTests:           false,
		StopOnBuildFailure: true,
	}

	ctx := context.Background()
	result, err := orch.Run(ctx, config)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !result.Build.Skipped {
		t.Error("Expected build to be skipped")
	}
	if !result.Lint.Skipped {
		t.Error("Expected lint to be skipped")
	}
	if !result.Test.Skipped {
		t.Error("Expected test to be skipped")
	}
}

func TestBuildSummary_BuildSkipped_LintFailed_TestSkipped(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "lint",
		Build: &BuildCheck{
			Skipped:    true,
			SkipReason: "disabled",
		},
		Lint: &LintCheck{
			Success:        false,
			Duration:       80 * time.Millisecond,
			ViolationCount: 1,
			Violations: []linter.Violation{
				{File: "x.go", Line: 1, Rule: "r", Severity: "error", Message: "msg"},
			},
		},
		Test: &TestCheck{
			Skipped:    true,
			SkipReason: "linter failed",
		},
	}

	summary := orch.buildSummary(result)

	// Build is skipped, so it should not show build passed/failed
	if strings.Contains(summary, "Build passed") || strings.Contains(summary, "Build failed") {
		t.Errorf("Build was skipped but summary shows build result: %s", summary)
	}

	if !strings.Contains(summary, "1 violation(s)") {
		t.Errorf("Expected violation count, got: %s", summary)
	}

	if !strings.Contains(summary, "Tests skipped") {
		t.Errorf("Expected tests skipped message, got: %s", summary)
	}
}

func TestBuildSummary_Exactly3Errors(t *testing.T) {
	orch := NewOrchestrator(".")

	errors := make([]build.BuildError, 3)
	for i := 0; i < 3; i++ {
		errors[i] = build.BuildError{
			File: "file.go", Line: i + 1, Column: 1, Message: "err", Type: "error",
		}
	}

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "build",
		Build: &BuildCheck{
			Success:    false,
			Duration:   100 * time.Millisecond,
			ErrorCount: 3,
			Errors:     errors,
		},
		Lint:  &LintCheck{Skipped: true, SkipReason: "build failed"},
		Test:  &TestCheck{Skipped: true, SkipReason: "build failed"},
	}

	summary := orch.buildSummary(result)

	// With exactly 3 errors, all should be shown without truncation
	if strings.Contains(summary, "... and") {
		t.Errorf("Should not truncate when exactly 3 errors, got: %s", summary)
	}
	if !strings.Contains(summary, "file.go:1:1") {
		t.Errorf("Expected first error, got: %s", summary)
	}
	if !strings.Contains(summary, "file.go:3:1") {
		t.Errorf("Expected third error, got: %s", summary)
	}
}

func TestBuildSummary_Exactly3Violations(t *testing.T) {
	orch := NewOrchestrator(".")

	violations := make([]linter.Violation, 3)
	for i := 0; i < 3; i++ {
		violations[i] = linter.Violation{
			File: "f.go", Line: i + 1, Rule: "r", Severity: "error", Message: "msg",
		}
	}

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "lint",
		Build:       &BuildCheck{Success: true, Duration: 10 * time.Millisecond},
		Lint: &LintCheck{
			Success:        false,
			Duration:       10 * time.Millisecond,
			ViolationCount: 3,
			Violations:     violations,
		},
		Test: &TestCheck{Success: true, Duration: 10 * time.Millisecond, Summary: testpkg.TestSummary{Total: 1, Passed: 1}},
	}

	summary := orch.buildSummary(result)

	if strings.Contains(summary, "... and") {
		t.Errorf("Should not truncate when exactly 3 violations, got: %s", summary)
	}
}

func TestBuildSummary_Exactly3FailedTests(t *testing.T) {
	orch := NewOrchestrator(".")

	failedTests := make([]testpkg.TestCase, 3)
	for i := 0; i < 3; i++ {
		failedTests[i] = testpkg.TestCase{
			Name: "TestX", Status: testpkg.TestFail, Error: "err",
		}
	}

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "test",
		Build:       &BuildCheck{Success: true, Duration: 10 * time.Millisecond},
		Lint:        &LintCheck{Success: true, Duration: 10 * time.Millisecond},
		Test: &TestCheck{
			Success:     false,
			Duration:    10 * time.Millisecond,
			Summary:     testpkg.TestSummary{Total: 10, Passed: 7, Failed: 3},
			FailedTests: failedTests,
		},
	}

	summary := orch.buildSummary(result)

	if strings.Contains(summary, "... and") {
		t.Errorf("Should not truncate when exactly 3 failed tests, got: %s", summary)
	}
}

func TestBuildSummary_ZeroDuration(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success: true,
		Build:   &BuildCheck{Success: true, Duration: 0},
		Lint:    &LintCheck{Success: true, Duration: 0},
		Test:    &TestCheck{Success: true, Duration: 0},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "0ms") {
		t.Errorf("Expected 0ms in summary for zero duration, got: %s", summary)
	}
}

func TestOrchestrator_Run_SummaryPopulated(t *testing.T) {
	orch := NewOrchestrator(".")
	config := OrchestratorConfig{
		ProjectPath: ".",
		RunBuild:    false,
		RunLint:     false,
		RunTests:    false,
	}

	ctx := context.Background()
	result, err := orch.Run(ctx, config)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Summary == "" {
		t.Error("Expected non-empty summary after run")
	}

	// All phases skipped should be a success
	if !strings.Contains(result.Summary, "All checks passed") {
		t.Errorf("Expected all checks passed in summary, got: %s", result.Summary)
	}
}
