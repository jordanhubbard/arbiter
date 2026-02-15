package actions

import (
	"strings"
	"testing"
)

func TestBuildEnhancedPrompt_NoLessonsNoContext(t *testing.T) {
	prompt := BuildEnhancedPrompt("", "")
	if strings.Contains(prompt, "LESSONS_PLACEHOLDER") {
		t.Error("placeholder should be removed")
	}
	if strings.Contains(prompt, "Lessons Learned") {
		t.Error("should not contain lessons section without lessons")
	}
	if strings.Contains(prompt, "Progress Context") {
		t.Error("should not contain progress context without context")
	}
}

func TestBuildEnhancedPrompt_WithLessons(t *testing.T) {
	prompt := BuildEnhancedPrompt("- Always run tests", "")
	if !strings.Contains(prompt, "Lessons Learned") {
		t.Error("should contain lessons header")
	}
	if !strings.Contains(prompt, "Always run tests") {
		t.Error("should contain lesson content")
	}
}

func TestBuildEnhancedPrompt_WithContext(t *testing.T) {
	prompt := BuildEnhancedPrompt("", "Iteration 3 of 25, last action: BUILD passed")
	if !strings.Contains(prompt, "Progress Context") {
		t.Error("should contain progress context header")
	}
	if !strings.Contains(prompt, "Iteration 3") {
		t.Error("should contain context content")
	}
}

func TestBuildEnhancedPrompt_WithBoth(t *testing.T) {
	prompt := BuildEnhancedPrompt("lesson1", "context1")
	if !strings.Contains(prompt, "Lessons Learned") {
		t.Error("should contain lessons")
	}
	if !strings.Contains(prompt, "Progress Context") {
		t.Error("should contain context")
	}
}

func TestBuildEnhancedPrompt_ContainsActionTypes(t *testing.T) {
	prompt := BuildEnhancedPrompt("", "")
	requiredTypes := []string{
		"read_file", "write_file", "edit_code", "build_project",
		"run_tests", "git_commit", "done", "search_text",
	}
	for _, typ := range requiredTypes {
		if !strings.Contains(prompt, typ) {
			t.Errorf("prompt should contain action type %s", typ)
		}
	}
}

func TestBuildTextPrompt_NoLessonsNoContext(t *testing.T) {
	prompt := BuildTextPrompt("", "")
	if strings.Contains(prompt, "LESSONS_PLACEHOLDER") {
		t.Error("placeholder should be removed")
	}
	if strings.Contains(prompt, "Lessons Learned") {
		t.Error("should not contain lessons section")
	}
}

func TestBuildTextPrompt_WithLessons(t *testing.T) {
	prompt := BuildTextPrompt("- Build first", "")
	if !strings.Contains(prompt, "Lessons Learned") {
		t.Error("should contain lessons header")
	}
	if !strings.Contains(prompt, "Build first") {
		t.Error("should contain lesson content")
	}
}

func TestBuildTextPrompt_WithContext(t *testing.T) {
	prompt := BuildTextPrompt("", "Step 5 of 25")
	if !strings.Contains(prompt, "Progress Context") {
		t.Error("should contain progress context")
	}
	if !strings.Contains(prompt, "Step 5") {
		t.Error("should contain context content")
	}
}

func TestBuildTextPrompt_ContainsCommands(t *testing.T) {
	prompt := BuildTextPrompt("", "")
	commands := []string{"SCOPE", "TREE", "READ", "SEARCH", "EDIT", "WRITE", "BUILD", "TEST", "BASH", "DONE"}
	for _, cmd := range commands {
		if !strings.Contains(prompt, cmd) {
			t.Errorf("text prompt should contain command %s", cmd)
		}
	}
}

func TestBuildSimpleJSONPrompt_NoLessonsNoContext(t *testing.T) {
	prompt := BuildSimpleJSONPrompt("", "")
	if strings.Contains(prompt, "LESSONS_PLACEHOLDER") {
		t.Error("placeholder should be removed")
	}
}

func TestBuildSimpleJSONPrompt_WithLessons(t *testing.T) {
	prompt := BuildSimpleJSONPrompt("- Commit before done", "")
	if !strings.Contains(prompt, "Lessons Learned") {
		t.Error("should contain lessons header")
	}
	if !strings.Contains(prompt, "Commit before done") {
		t.Error("should contain lesson content")
	}
}

func TestBuildSimpleJSONPrompt_WithContext(t *testing.T) {
	prompt := BuildSimpleJSONPrompt("", "iteration 10")
	if !strings.Contains(prompt, "Progress Context") {
		t.Error("should contain progress context")
	}
}

func TestBuildSimpleJSONPrompt_ContainsActions(t *testing.T) {
	prompt := BuildSimpleJSONPrompt("", "")
	actions := []string{"scope", "read", "search", "edit", "write", "build", "test", "bash", "done", "git_commit"}
	for _, a := range actions {
		if !strings.Contains(prompt, `"`+a+`"`) {
			t.Errorf("simple JSON prompt should contain action %q", a)
		}
	}
}
