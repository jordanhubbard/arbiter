package actions

import (
	"context"
	"time"

	"github.com/jordanhubbard/agenticorp/internal/linter"
)

// LinterRunnerAdapter adapts internal/linter.LinterRunner to the actions.LinterRunner interface
type LinterRunnerAdapter struct {
	runner     *linter.LinterRunner
	projectDir string
}

// NewLinterRunnerAdapter creates a new adapter for the linter runner
func NewLinterRunnerAdapter(projectDir string) *LinterRunnerAdapter {
	return &LinterRunnerAdapter{
		runner:     linter.NewLinterRunner(projectDir),
		projectDir: projectDir,
	}
}

// Run executes linter and returns structured results
func (a *LinterRunnerAdapter) Run(ctx context.Context, projectPath string, files []string, framework string, timeoutSeconds int) (map[string]interface{}, error) {
	// Use provided project path or fall back to adapter's project dir
	if projectPath == "" || projectPath == "." {
		projectPath = a.projectDir
	}

	// Build lint request
	req := linter.LintRequest{
		ProjectPath: projectPath,
		Files:       files,
		Framework:   framework,
		Timeout:     linter.DefaultLintTimeout,
	}

	// Apply custom timeout if specified
	if timeoutSeconds > 0 {
		req.Timeout = time.Duration(timeoutSeconds) * time.Second
	}

	// Execute linter
	result, err := a.runner.Run(ctx, req)
	if err != nil {
		return nil, err
	}

	// Convert LintResult to map for JSON serialization
	metadata := map[string]interface{}{
		"framework":  result.Framework,
		"success":    result.Success,
		"exit_code":  result.ExitCode,
		"timed_out":  result.TimedOut,
		"duration":   result.Duration.String(),
		"raw_output": result.RawOutput,
	}

	// Add error if present
	if result.Error != "" {
		metadata["error"] = result.Error
	}

	// Add violations if present
	if len(result.Violations) > 0 {
		violations := make([]map[string]interface{}, 0, len(result.Violations))
		for _, v := range result.Violations {
			violation := map[string]interface{}{
				"file":     v.File,
				"line":     v.Line,
				"column":   v.Column,
				"rule":     v.Rule,
				"severity": v.Severity,
				"message":  v.Message,
				"linter":   v.Linter,
			}
			violations = append(violations, violation)
		}
		metadata["violations"] = violations
		metadata["violation_count"] = len(violations)
	} else {
		metadata["violations"] = []interface{}{}
		metadata["violation_count"] = 0
	}

	return metadata, nil
}
