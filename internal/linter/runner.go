package linter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Violation represents a single linter violation
type Violation struct {
	File     string `json:"file"`      // File path relative to project
	Line     int    `json:"line"`      // Line number
	Column   int    `json:"column"`    // Column number (if available)
	Rule     string `json:"rule"`      // Rule identifier (e.g., "unused-var")
	Severity string `json:"severity"`  // "error", "warning", "info"
	Message  string `json:"message"`   // Human-readable message
	Linter   string `json:"linter"`    // Specific linter that reported (e.g., "staticcheck")
}

// LintResult contains the complete linting result
type LintResult struct {
	Framework  string      `json:"framework"`   // "golangci-lint", "eslint", "pylint"
	Success    bool        `json:"success"`     // True if no violations
	ExitCode   int         `json:"exit_code"`   // Process exit code
	Violations []Violation `json:"violations"`  // List of violations
	RawOutput  string      `json:"raw_output"`  // Full linter output
	Duration   time.Duration `json:"duration"` // Execution time
	TimedOut   bool        `json:"timed_out"`   // Whether execution timed out
	Error      string      `json:"error"`       // Error message if execution failed
}

// LintRequest defines parameters for linter execution
type LintRequest struct {
	ProjectPath  string            // Absolute path to project
	LintCommand  string            // Optional: override lint command
	Framework    string            // Optional: specify linter (auto-detect if empty)
	Files        []string          // Optional: specific files to lint
	Environment  map[string]string // Environment variables
	Timeout      time.Duration     // Max execution time
}

const (
	// DefaultLintTimeout is the default maximum linter execution time
	DefaultLintTimeout = 5 * time.Minute
	// MaxLintTimeout is the absolute maximum allowed timeout
	MaxLintTimeout = 15 * time.Minute
)

// LinterRunner executes linters and parses results
type LinterRunner struct {
	workDir string
}

// NewLinterRunner creates a new LinterRunner instance
func NewLinterRunner(workDir string) *LinterRunner {
	return &LinterRunner{
		workDir: workDir,
	}
}

// Run executes linter and returns structured results
func (r *LinterRunner) Run(ctx context.Context, req LintRequest) (*LintResult, error) {
	// Validate request
	if req.ProjectPath == "" {
		req.ProjectPath = r.workDir
	}

	// Validate timeout
	if req.Timeout == 0 {
		req.Timeout = DefaultLintTimeout
	} else if req.Timeout > MaxLintTimeout {
		req.Timeout = MaxLintTimeout
	}

	// Auto-detect framework if not specified
	framework := req.Framework
	if framework == "" {
		detected, err := r.DetectFramework(req.ProjectPath)
		if err != nil {
			return nil, fmt.Errorf("failed to detect linter framework: %w", err)
		}
		framework = detected
	}

	// Build lint command
	cmdArgs, err := r.BuildCommand(framework, req.ProjectPath, req.Files, req.LintCommand)
	if err != nil {
		return nil, fmt.Errorf("failed to build lint command: %w", err)
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	// Execute linter
	startTime := time.Now()
	output, exitCode, timedOut, err := r.executeCommand(timeoutCtx, cmdArgs, req.ProjectPath, req.Environment)
	duration := time.Since(startTime)

	// If execution failed completely, return error result
	if err != nil && !timedOut {
		return &LintResult{
			Framework: framework,
			Success:   false,
			Duration:  duration,
			RawOutput: output,
			ExitCode:  exitCode,
			TimedOut:  false,
			Error:     err.Error(),
		}, nil
	}

	// Parse output based on framework
	result, err := r.parseOutput(framework, output, exitCode)
	if err != nil {
		// If parsing fails, return a basic result with raw output
		return &LintResult{
			Framework: framework,
			Success:   exitCode == 0,
			Duration:  duration,
			RawOutput: output,
			ExitCode:  exitCode,
			TimedOut:  timedOut,
			Error:     fmt.Sprintf("failed to parse output: %v", err),
		}, nil
	}

	// Update result with execution details
	result.Duration = duration
	result.TimedOut = timedOut

	return result, nil
}

// DetectFramework auto-detects the linter framework based on project structure
func (r *LinterRunner) DetectFramework(projectPath string) (string, error) {
	// Check for Go
	if r.fileExists(filepath.Join(projectPath, "go.mod")) {
		return "golangci-lint", nil
	}
	if matches, _ := filepath.Glob(filepath.Join(projectPath, "*.go")); len(matches) > 0 {
		return "golangci-lint", nil
	}

	// Check for Node.js/ESLint
	if r.fileExists(filepath.Join(projectPath, ".eslintrc.js")) ||
		r.fileExists(filepath.Join(projectPath, ".eslintrc.json")) ||
		r.fileExists(filepath.Join(projectPath, ".eslintrc.yml")) {
		return "eslint", nil
	}

	packageJSON := filepath.Join(projectPath, "package.json")
	if r.fileExists(packageJSON) {
		data, err := os.ReadFile(packageJSON)
		if err == nil && strings.Contains(string(data), "eslint") {
			return "eslint", nil
		}
	}

	// Check for Python/pylint
	if r.fileExists(filepath.Join(projectPath, ".pylintrc")) ||
		r.fileExists(filepath.Join(projectPath, "pylintrc")) {
		return "pylint", nil
	}

	// Check for Python files
	if matches, _ := filepath.Glob(filepath.Join(projectPath, "*.py")); len(matches) > 0 {
		return "pylint", nil
	}

	return "", fmt.Errorf("could not detect linter framework in %s", projectPath)
}

// BuildCommand constructs the linter command based on framework
func (r *LinterRunner) BuildCommand(framework, projectPath string, files []string, customCommand string) ([]string, error) {
	// Use custom command if provided
	if customCommand != "" {
		return strings.Fields(customCommand), nil
	}

	switch framework {
	case "golangci-lint":
		cmd := []string{"golangci-lint", "run"}
		if len(files) > 0 {
			cmd = append(cmd, files...)
		} else {
			cmd = append(cmd, "./...")
		}
		return cmd, nil

	case "eslint":
		cmd := []string{"eslint", "--format", "compact"}
		if len(files) > 0 {
			cmd = append(cmd, files...)
		} else {
			cmd = append(cmd, ".")
		}
		return cmd, nil

	case "pylint":
		cmd := []string{"pylint", "--output-format=text"}
		if len(files) > 0 {
			cmd = append(cmd, files...)
		} else {
			cmd = append(cmd, ".")
		}
		return cmd, nil

	default:
		return nil, fmt.Errorf("unsupported linter framework: %s", framework)
	}
}

// executeCommand runs the linter command and captures output
func (r *LinterRunner) executeCommand(ctx context.Context, cmdArgs []string, workDir string, env map[string]string) (output string, exitCode int, timedOut bool, err error) {
	if len(cmdArgs) == 0 {
		return "", 1, false, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = workDir

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture combined output
	outputBytes, err := cmd.CombinedOutput()
	output = string(outputBytes)

	// Check for timeout first
	if ctx.Err() == context.DeadlineExceeded {
		return output, 124, true, nil
	}

	// Get exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return output, 1, false, err
		}
	}

	return output, exitCode, false, nil
}

// parseOutput parses linter output based on framework
func (r *LinterRunner) parseOutput(framework, output string, exitCode int) (*LintResult, error) {
	switch framework {
	case "golangci-lint":
		return r.parseGolangciLintOutput(output, exitCode)
	case "eslint":
		return r.parseESLintOutput(output, exitCode)
	case "pylint":
		return r.parsePylintOutput(output, exitCode)
	default:
		return r.parseGenericOutput(output, exitCode, framework)
	}
}

// parseGolangciLintOutput parses golangci-lint output
func (r *LinterRunner) parseGolangciLintOutput(output string, exitCode int) (*LintResult, error) {
	result := &LintResult{
		Framework:  "golangci-lint",
		Success:    exitCode == 0,
		RawOutput:  output,
		ExitCode:   exitCode,
		Violations: []Violation{},
	}

	// golangci-lint format: path/to/file.go:123:45: message (linter)
	// Example: internal/foo/bar.go:10:2: unused variable 'x' (unused)
	re := regexp.MustCompile(`^(.+?):(\d+):(\d+):\s+(.+?)\s+\((\w+)\)`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 6 {
			file := matches[1]
			line := parseInt(matches[2])
			col := parseInt(matches[3])
			message := matches[4]
			linter := matches[5]

			violation := Violation{
				File:     file,
				Line:     line,
				Column:   col,
				Rule:     linter,
				Severity: "error", // golangci-lint reports everything as errors
				Message:  message,
				Linter:   linter,
			}
			result.Violations = append(result.Violations, violation)
		}
	}

	return result, nil
}

// parseESLintOutput parses ESLint compact format output
func (r *LinterRunner) parseESLintOutput(output string, exitCode int) (*LintResult, error) {
	result := &LintResult{
		Framework:  "eslint",
		Success:    exitCode == 0,
		RawOutput:  output,
		ExitCode:   exitCode,
		Violations: []Violation{},
	}

	// ESLint compact format: path/to/file.js: line 10, col 5, Error - message (rule-name)
	re := regexp.MustCompile(`^(.+?):\s+line\s+(\d+),\s+col\s+(\d+),\s+(\w+)\s+-\s+(.+?)\s+\(([^)]+)\)`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 7 {
			file := matches[1]
			lineNum := parseInt(matches[2])
			col := parseInt(matches[3])
			severity := strings.ToLower(matches[4])
			message := matches[5]
			rule := matches[6]

			violation := Violation{
				File:     file,
				Line:     lineNum,
				Column:   col,
				Rule:     rule,
				Severity: severity,
				Message:  message,
				Linter:   "eslint",
			}
			result.Violations = append(result.Violations, violation)
		}
	}

	return result, nil
}

// parsePylintOutput parses pylint text format output
func (r *LinterRunner) parsePylintOutput(output string, exitCode int) (*LintResult, error) {
	result := &LintResult{
		Framework:  "pylint",
		Success:    exitCode == 0,
		RawOutput:  output,
		ExitCode:   exitCode,
		Violations: []Violation{},
	}

	// Pylint format: path/to/file.py:123:45: C0301: Line too long (rule-name)
	re := regexp.MustCompile(`^(.+?):(\d+):(\d+):\s+([CRWEF]\d+):\s+(.+?)\s+\(([^)]+)\)`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 7 {
			file := matches[1]
			lineNum := parseInt(matches[2])
			col := parseInt(matches[3])
			code := matches[4]
			message := matches[5]
			rule := matches[6]

			// Map pylint severity codes
			severity := "info"
			switch code[0] {
			case 'E', 'F':
				severity = "error"
			case 'W':
				severity = "warning"
			case 'C', 'R':
				severity = "info"
			}

			violation := Violation{
				File:     file,
				Line:     lineNum,
				Column:   col,
				Rule:     rule,
				Severity: severity,
				Message:  message,
				Linter:   "pylint",
			}
			result.Violations = append(result.Violations, violation)
		}
	}

	return result, nil
}

// parseGenericOutput provides fallback parsing for unknown linters
func (r *LinterRunner) parseGenericOutput(output string, exitCode int, framework string) (*LintResult, error) {
	result := &LintResult{
		Framework:  framework,
		Success:    exitCode == 0,
		RawOutput:  output,
		ExitCode:   exitCode,
		Violations: []Violation{},
	}

	// Try to parse common patterns
	// Pattern: file:line:col: message
	re := regexp.MustCompile(`^(.+?):(\d+):(\d+):\s+(.+)`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 5 {
			violation := Violation{
				File:     matches[1],
				Line:     parseInt(matches[2]),
				Column:   parseInt(matches[3]),
				Message:  matches[4],
				Severity: "error",
				Linter:   framework,
			}
			result.Violations = append(result.Violations, violation)
		}
	}

	return result, nil
}

// fileExists checks if a file exists
func (r *LinterRunner) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// parseInt safely parses an integer from string
func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
