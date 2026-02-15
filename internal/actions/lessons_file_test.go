package actions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLessonsFile(t *testing.T) {
	lf := NewLessonsFile("/tmp/claude/test-project")
	if lf == nil {
		t.Fatal("expected non-nil LessonsFile")
	}
	if lf.projectDir != "/tmp/claude/test-project" {
		t.Errorf("expected project dir, got %s", lf.projectDir)
	}
}

func TestLessonsFile_GetLessonsForPrompt_NoFile(t *testing.T) {
	lf := NewLessonsFile("/tmp/claude/nonexistent-dir-12345")
	content := lf.GetLessonsForPrompt()
	if content != "" {
		t.Errorf("expected empty string for missing file, got %q", content)
	}
}

func TestLessonsFile_GetLessonsForPrompt_WithContent(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "claude", "lessons-test-get")
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	lessonsPath := filepath.Join(dir, "LESSONS.md")
	err := os.WriteFile(lessonsPath, []byte("## Lesson 1\n- Always test\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write test lessons: %v", err)
	}

	lf := NewLessonsFile(dir)
	content := lf.GetLessonsForPrompt()
	if !strings.Contains(content, "Lesson 1") {
		t.Errorf("expected lesson content, got %q", content)
	}
}

func TestLessonsFile_GetLessonsForPrompt_Truncation(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "claude", "lessons-test-trunc")
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	// Write content larger than maxLessonsSize
	largeContent := strings.Repeat("x", maxLessonsSize+1000)
	lessonsPath := filepath.Join(dir, "LESSONS.md")
	err := os.WriteFile(lessonsPath, []byte(largeContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test lessons: %v", err)
	}

	lf := NewLessonsFile(dir)
	content := lf.GetLessonsForPrompt()
	if !strings.Contains(content, "truncated") {
		t.Error("expected truncation marker")
	}
	if len(content) > maxLessonsSize+100 {
		t.Errorf("content too long: %d", len(content))
	}
}

func TestLessonsFile_RecordLesson(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "claude", "lessons-test-record")
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	lf := NewLessonsFile(dir)
	err := lf.RecordLesson("test_category", "Test Title", "Test detail", "bead-1", "agent-1")
	if err != nil {
		t.Fatalf("RecordLesson failed: %v", err)
	}

	// Read back and verify
	content := lf.GetLessonsForPrompt()
	if !strings.Contains(content, "TEST_CATEGORY") {
		t.Error("expected category in recorded lesson")
	}
	if !strings.Contains(content, "Test Title") {
		t.Error("expected title in recorded lesson")
	}
	if !strings.Contains(content, "Test detail") {
		t.Error("expected detail in recorded lesson")
	}
	if !strings.Contains(content, "bead-1") {
		t.Error("expected bead ID in recorded lesson")
	}
}

func TestLessonsFile_RecordLesson_Append(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "claude", "lessons-test-append")
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	lf := NewLessonsFile(dir)
	err := lf.RecordLesson("cat1", "Title1", "Detail1", "b1", "a1")
	if err != nil {
		t.Fatalf("first RecordLesson failed: %v", err)
	}
	err = lf.RecordLesson("cat2", "Title2", "Detail2", "b2", "a2")
	if err != nil {
		t.Fatalf("second RecordLesson failed: %v", err)
	}

	content := lf.GetLessonsForPrompt()
	if !strings.Contains(content, "Title1") || !strings.Contains(content, "Title2") {
		t.Error("expected both lessons appended")
	}
}

func TestLessonsFile_RecordBuildFailure(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "claude", "lessons-test-build")
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	lf := NewLessonsFile(dir)
	errorOutput := "foo.go:10:2: error: undefined variable x\nbar.go:20: cannot find package\nsome normal output"
	err := lf.RecordBuildFailure(errorOutput, "bead-1", "agent-1")
	if err != nil {
		t.Fatalf("RecordBuildFailure failed: %v", err)
	}

	content := lf.GetLessonsForPrompt()
	if !strings.Contains(content, "BUILD_FAILURE") {
		t.Error("expected BUILD_FAILURE category")
	}
	if !strings.Contains(content, "undefined variable") {
		t.Error("expected error line in lesson")
	}
}

func TestLessonsFile_RecordBuildFailure_NoErrorLines(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "claude", "lessons-test-build-no-err")
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	lf := NewLessonsFile(dir)
	err := lf.RecordBuildFailure("all normal output\nno issues", "bead-1", "agent-1")
	if err != nil {
		t.Fatalf("RecordBuildFailure failed: %v", err)
	}

	content := lf.GetLessonsForPrompt()
	if !strings.Contains(content, "Build failed") {
		t.Error("expected default build failure message")
	}
}

func TestLessonsFile_RecordEditFailure(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "claude", "lessons-test-edit")
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	lf := NewLessonsFile(dir)
	err := lf.RecordEditFailure("router.go", "OLD text not found", "bead-1", "agent-1")
	if err != nil {
		t.Fatalf("RecordEditFailure failed: %v", err)
	}

	content := lf.GetLessonsForPrompt()
	if !strings.Contains(content, "EDIT_FAILURE") {
		t.Error("expected EDIT_FAILURE category")
	}
	if !strings.Contains(content, "router.go") {
		t.Error("expected file path in lesson")
	}
	if !strings.Contains(content, "READ the file") {
		t.Error("expected advice in lesson")
	}
}

func TestLessonsFile_RecordLesson_BadDir(t *testing.T) {
	lf := NewLessonsFile("/nonexistent/path/that/cannot/exist")
	err := lf.RecordLesson("cat", "title", "detail", "b", "a")
	if err == nil {
		t.Error("expected error for bad directory")
	}
}
