package actions

import (
	"strings"
	"testing"
)

func TestFormatResultsAsUserMessage_Empty(t *testing.T) {
	result := FormatResultsAsUserMessage(nil)
	if result != "No actions were executed." {
		t.Errorf("expected empty message, got %q", result)
	}
	result2 := FormatResultsAsUserMessage([]Result{})
	if result2 != "No actions were executed." {
		t.Errorf("expected empty message, got %q", result2)
	}
}

func TestFormatResultsAsUserMessage_SingleResult(t *testing.T) {
	results := []Result{
		{ActionType: ActionDone, Status: "executed", Message: "agent signaled done"},
	}
	output := FormatResultsAsUserMessage(results)
	if !strings.Contains(output, "## Action Results") {
		t.Error("expected header")
	}
	if !strings.Contains(output, "done") {
		t.Error("expected action type")
	}
	if !strings.Contains(output, "what would you like to do next?") {
		t.Error("expected trailing prompt")
	}
}

func TestFormatResultsAsUserMessage_MultipleResults(t *testing.T) {
	results := []Result{
		{ActionType: ActionGitStatus, Status: "executed", Message: "git status", Metadata: map[string]interface{}{"output": "clean"}},
		{ActionType: ActionGitDiff, Status: "executed", Message: "git diff", Metadata: map[string]interface{}{"output": "no changes"}},
	}
	output := FormatResultsAsUserMessage(results)
	if !strings.Contains(output, "---") {
		t.Error("expected separator between results")
	}
}

func TestFormatSingleResult_Error(t *testing.T) {
	r := Result{ActionType: ActionBuildProject, Status: "error", Message: "build failed"}
	output := formatSingleResult(r)
	if !strings.Contains(output, "**Error:**") {
		t.Error("expected error label")
	}
	if !strings.Contains(output, "build failed") {
		t.Error("expected error message")
	}
}

func TestFormatFileRead(t *testing.T) {
	r := Result{
		ActionType: ActionReadFile,
		Status:     "executed",
		Message:    "file read",
		Metadata: map[string]interface{}{
			"path":    "foo.go",
			"content": "package foo\n",
			"size":    float64(12),
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "`foo.go`") {
		t.Error("expected file path")
	}
	if !strings.Contains(output, "package foo") {
		t.Error("expected file content")
	}
}

func TestFormatFileRead_LargeContent(t *testing.T) {
	content := strings.Repeat("x", maxFileContentLen+100)
	r := Result{
		ActionType: ActionReadCode,
		Status:     "executed",
		Metadata: map[string]interface{}{
			"path":    "big.go",
			"content": content,
			"size":    float64(len(content)),
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "... (truncated)") {
		t.Error("expected truncation marker")
	}
}

func TestFormatFileWrite(t *testing.T) {
	r := Result{
		ActionType: ActionWriteFile,
		Status:     "executed",
		Metadata: map[string]interface{}{
			"path":          "out.go",
			"bytes_written": float64(42),
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "42") {
		t.Error("expected bytes written")
	}
	if !strings.Contains(output, "out.go") {
		t.Error("expected path")
	}
}

func TestFormatPatchApply(t *testing.T) {
	r := Result{
		ActionType: ActionEditCode,
		Status:     "executed",
		Metadata:   map[string]interface{}{"output": "applied hunk 1"},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "Patch applied") {
		t.Error("expected patch message")
	}
	if !strings.Contains(output, "applied hunk 1") {
		t.Error("expected output")
	}
}

func TestFormatPatchApply_EmptyOutput(t *testing.T) {
	r := Result{
		ActionType: ActionApplyPatch,
		Status:     "executed",
		Metadata:   map[string]interface{}{"output": ""},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "Patch applied") {
		t.Error("expected patch message")
	}
}

func TestFormatBuildResult_Success(t *testing.T) {
	r := Result{
		ActionType: ActionBuildProject,
		Status:     "executed",
		Metadata: map[string]interface{}{
			"success":   true,
			"exit_code": float64(0),
			"output":    "",
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "PASSED") {
		t.Error("expected PASSED")
	}
}

func TestFormatBuildResult_Failure(t *testing.T) {
	r := Result{
		ActionType: ActionBuildProject,
		Status:     "executed",
		Metadata: map[string]interface{}{
			"success":   false,
			"exit_code": float64(1),
			"output":    "error: undefined variable\n",
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "FAILED") {
		t.Error("expected FAILED")
	}
	if !strings.Contains(output, "fix the build") {
		t.Error("expected fix suggestion")
	}
}

func TestFormatBuildResult_NilMetadata(t *testing.T) {
	r := Result{
		ActionType: ActionBuildProject,
		Status:     "executed",
		Message:    "build executed",
		Metadata:   nil,
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "build executed") {
		t.Error("expected message fallback")
	}
}

func TestFormatTestResult_Success(t *testing.T) {
	r := Result{
		ActionType: ActionRunTests,
		Status:     "executed",
		Metadata: map[string]interface{}{
			"success": true,
			"passed":  float64(10),
			"failed":  float64(0),
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "PASSED") {
		t.Error("expected PASSED")
	}
}

func TestFormatTestResult_Failure(t *testing.T) {
	r := Result{
		ActionType: ActionRunTests,
		Status:     "executed",
		Metadata: map[string]interface{}{
			"success": false,
			"passed":  float64(5),
			"failed":  float64(2),
			"output":  "FAIL TestFoo",
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "FAILED") {
		t.Error("expected FAILED")
	}
	if !strings.Contains(output, "fix the failing") {
		t.Error("expected fix suggestion")
	}
}

func TestFormatTestResult_NilMetadata(t *testing.T) {
	r := Result{
		ActionType: ActionRunTests,
		Status:     "executed",
		Message:    "tests executed",
		Metadata:   nil,
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "tests executed") {
		t.Error("expected message fallback")
	}
}

func TestFormatLintResult_NoOutput(t *testing.T) {
	r := Result{
		ActionType: ActionRunLinter,
		Status:     "executed",
		Metadata:   map[string]interface{}{"output": ""},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "no issues") {
		t.Error("expected no issues message")
	}
}

func TestFormatLintResult_WithOutput(t *testing.T) {
	r := Result{
		ActionType: ActionRunLinter,
		Status:     "executed",
		Metadata:   map[string]interface{}{"output": "foo.go:10: unused variable"},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "unused variable") {
		t.Error("expected lint output")
	}
}

func TestFormatLintResult_NilMetadata(t *testing.T) {
	r := Result{
		ActionType: ActionRunLinter,
		Status:     "executed",
		Message:    "linter executed",
		Metadata:   nil,
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "linter executed") {
		t.Error("expected message fallback")
	}
}

func TestFormatSearchResult_NoMatches(t *testing.T) {
	r := Result{
		ActionType: ActionSearchText,
		Status:     "executed",
		Metadata:   map[string]interface{}{"matches": nil},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "No matches") {
		t.Error("expected no matches message")
	}
}

func TestFormatSearchResult_WithMatches(t *testing.T) {
	r := Result{
		ActionType: ActionSearchText,
		Status:     "executed",
		Metadata: map[string]interface{}{
			"matches": []interface{}{
				map[string]interface{}{"path": "foo.go", "line": 10},
			},
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "foo.go") {
		t.Error("expected match data")
	}
}

func TestFormatTreeResult_Empty(t *testing.T) {
	r := Result{
		ActionType: ActionReadTree,
		Status:     "executed",
		Metadata:   map[string]interface{}{"entries": nil},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "Empty directory") {
		t.Error("expected empty directory message")
	}
}

func TestFormatTreeResult_WithEntries(t *testing.T) {
	r := Result{
		ActionType: ActionReadTree,
		Status:     "executed",
		Metadata: map[string]interface{}{
			"entries": []interface{}{
				map[string]interface{}{"path": "src/", "type": "dir"},
			},
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "src/") {
		t.Error("expected entry data")
	}
}

func TestFormatGitOutput_Empty(t *testing.T) {
	r := Result{
		ActionType: ActionGitStatus,
		Status:     "executed",
		Metadata:   map[string]interface{}{"output": ""},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "(empty)") {
		t.Error("expected empty marker")
	}
}

func TestFormatGitOutput_WithContent(t *testing.T) {
	r := Result{
		ActionType: ActionGitDiff,
		Status:     "executed",
		Metadata:   map[string]interface{}{"output": "+added line\n-removed line\n"},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "+added line") {
		t.Error("expected diff content")
	}
}

func TestFormatGitCommit_WithSHA(t *testing.T) {
	r := Result{
		ActionType: ActionGitCommit,
		Status:     "executed",
		Metadata:   map[string]interface{}{"sha": "abc123", "message": "fix bug"},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "abc123") {
		t.Error("expected commit SHA")
	}
	if !strings.Contains(output, "fix bug") {
		t.Error("expected commit message")
	}
}

func TestFormatGitCommit_NoSHA(t *testing.T) {
	r := Result{
		ActionType: ActionGitCommit,
		Status:     "executed",
		Message:    "commit created",
		Metadata:   map[string]interface{}{},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "commit created") {
		t.Error("expected fallback message")
	}
}

func TestFormatGitLog(t *testing.T) {
	r := Result{
		ActionType: ActionGitLog,
		Status:     "executed",
		Metadata:   map[string]interface{}{"output": "abc123 fix bug\ndef456 add feature\n"},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "abc123") {
		t.Error("expected log content")
	}
}

func TestFormatCommandResult(t *testing.T) {
	r := Result{
		ActionType: ActionRunCommand,
		Status:     "executed",
		Metadata: map[string]interface{}{
			"exit_code": float64(0),
			"stdout":    "hello world\n",
			"stderr":    "",
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "Exit code") {
		t.Error("expected exit code")
	}
	if !strings.Contains(output, "hello world") {
		t.Error("expected stdout")
	}
}

func TestFormatCommandResult_WithStderr(t *testing.T) {
	r := Result{
		ActionType: ActionRunCommand,
		Status:     "executed",
		Metadata: map[string]interface{}{
			"exit_code": float64(1),
			"stdout":    "",
			"stderr":    "error occurred\n",
		},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "stderr") {
		t.Error("expected stderr label")
	}
	if !strings.Contains(output, "error occurred") {
		t.Error("expected stderr content")
	}
}

func TestFormatBeadCreated(t *testing.T) {
	r := Result{
		ActionType: ActionCreateBead,
		Status:     "executed",
		Metadata:   map[string]interface{}{"bead_id": "bead-123"},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "bead-123") {
		t.Error("expected bead id")
	}
}

func TestFormatCloseBead(t *testing.T) {
	r := Result{
		ActionType: ActionCloseBead,
		Status:     "executed",
		Message:    "bead closed",
		Metadata:   map[string]interface{}{"bead_id": "bead-123"},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "closed") {
		t.Error("expected close message")
	}
}

func TestFormatDone(t *testing.T) {
	r := Result{
		ActionType: ActionDone,
		Status:     "executed",
		Message:    "agent signaled done",
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "complete") {
		t.Error("expected done message")
	}
}

func TestFormatDefault(t *testing.T) {
	r := Result{
		ActionType: "unknown_type",
		Status:     "executed",
		Message:    "something happened",
		Metadata:   map[string]interface{}{"key": "value"},
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "something happened") {
		t.Error("expected message")
	}
	if !strings.Contains(output, "value") {
		t.Error("expected metadata")
	}
}

func TestFormatDefault_NilMetadata(t *testing.T) {
	r := Result{
		ActionType: "custom",
		Status:     "executed",
		Message:    "custom action",
		Metadata:   nil,
	}
	output := formatSingleResult(r)
	if !strings.Contains(output, "custom action") {
		t.Error("expected message")
	}
}

func TestTruncateBuildOutput_Short(t *testing.T) {
	output := "short output"
	result := truncateBuildOutput(output)
	if result != output {
		t.Errorf("expected unchanged output, got %q", result)
	}
}

func TestTruncateBuildOutput_LongWithErrors(t *testing.T) {
	lines := make([]string, 0)
	for i := 0; i < 100; i++ {
		lines = append(lines, "normal line "+strings.Repeat("x", 50))
	}
	lines = append(lines, "error: undefined variable 'x'")
	lines = append(lines, "cannot find module")
	output := strings.Join(lines, "\n")
	result := truncateBuildOutput(output)
	if !strings.Contains(result, "error: undefined") {
		t.Error("expected error line to be preserved")
	}
	if !strings.Contains(result, "cannot find") {
		t.Error("expected cannot line to be preserved")
	}
}

func TestTruncateBuildOutput_LongNoErrors(t *testing.T) {
	output := strings.Repeat("normal output line\n", 500)
	result := truncateBuildOutput(output)
	if !strings.Contains(result, "showing last portion") {
		t.Error("expected truncation from end")
	}
}

func TestTruncateOutput_Short(t *testing.T) {
	result := truncateOutput("short", 100)
	if result != "short" {
		t.Errorf("expected unchanged, got %q", result)
	}
}

func TestTruncateOutput_Long(t *testing.T) {
	long := strings.Repeat("x", 200)
	result := truncateOutput(long, 100)
	if !strings.Contains(result, "truncated") {
		t.Error("expected truncation marker")
	}
	if len(result) > 120 {
		t.Error("expected truncated length")
	}
}

func TestWriteErrorSuggestion_EditNotFound(t *testing.T) {
	var sb strings.Builder
	r := Result{ActionType: ActionEditCode, Status: "error", Message: "OLD text not found in file.go"}
	writeErrorSuggestion(&sb, r)
	output := sb.String()
	if !strings.Contains(output, "READ the file") {
		t.Error("expected READ suggestion")
	}
}

func TestWriteErrorSuggestion_FileNotFound(t *testing.T) {
	var sb strings.Builder
	r := Result{ActionType: ActionReadFile, Status: "error", Message: "no such file or directory"}
	writeErrorSuggestion(&sb, r)
	output := sb.String()
	if !strings.Contains(output, "SCOPE") || !strings.Contains(output, "TREE") {
		t.Error("expected file navigation suggestion")
	}
}

func TestWriteErrorSuggestion_PathEscape(t *testing.T) {
	var sb strings.Builder
	r := Result{ActionType: ActionReadFile, Status: "error", Message: "path escapes project directory"}
	writeErrorSuggestion(&sb, r)
	output := sb.String()
	if !strings.Contains(output, "relative paths") {
		t.Error("expected relative path suggestion")
	}
}

func TestWriteErrorSuggestion_BuildFail(t *testing.T) {
	var sb strings.Builder
	r := Result{ActionType: ActionBuildProject, Status: "error", Message: "build failed with exit code 1"}
	writeErrorSuggestion(&sb, r)
	output := sb.String()
	if !strings.Contains(output, "fix") {
		t.Error("expected fix suggestion")
	}
}

func TestWriteErrorSuggestion_NotCloned(t *testing.T) {
	var sb strings.Builder
	r := Result{ActionType: ActionGitStatus, Status: "error", Message: "repository not cloned"}
	writeErrorSuggestion(&sb, r)
	output := sb.String()
	if !strings.Contains(output, "not cloned") {
		t.Error("expected clone suggestion")
	}
}

func TestWriteErrorSuggestion_Default(t *testing.T) {
	var sb strings.Builder
	r := Result{ActionType: ActionRunCommand, Status: "error", Message: "some random error"}
	writeErrorSuggestion(&sb, r)
	output := sb.String()
	if !strings.Contains(output, "adjusting your approach") {
		t.Error("expected default suggestion")
	}
}

func TestFormatResultsWithContext(t *testing.T) {
	results := []Result{
		{ActionType: ActionDone, Status: "executed", Message: "done"},
	}
	output := FormatResultsWithContext(results, "/home/project")
	if !strings.Contains(output, "/home/project") {
		t.Error("expected project root")
	}
}

func TestFormatResultsWithContext_NoRoot(t *testing.T) {
	results := []Result{
		{ActionType: ActionDone, Status: "executed", Message: "done"},
	}
	output := FormatResultsWithContext(results, "")
	if strings.Contains(output, "Working directory") {
		t.Error("should not contain working directory when empty")
	}
}
