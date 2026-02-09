package actions

import "strings"

// SimpleJSONPrompt is a minimal JSON action prompt — only ~10 action types
// instead of 60+, designed for local 30B models with response_format: json_object.
const SimpleJSONPrompt = `You must respond with strict JSON only. No text outside JSON.

Response format:
{"action": "<type>", ...required fields..., "notes": "your reasoning"}

## Available Actions

### Navigation & Reading
{"action": "scope", "path": "dir/"}                    — List directory contents
{"action": "read", "path": "file.go"}                   — Read a file
{"action": "search", "query": "pattern"}                 — Search for text in project
{"action": "search", "query": "pattern", "path": "dir/"} — Search in specific directory

### Editing
{"action": "edit", "path": "file.go", "old": "exact text to find", "new": "replacement text"}
{"action": "write", "path": "file.go", "content": "full file content"}

### Build & Test
{"action": "build"}                                      — Build the project
{"action": "test"}                                       — Run all tests
{"action": "test", "pattern": "TestFoo"}                 — Run specific tests
{"action": "bash", "command": "go vet ./..."}            — Run shell command

### Git & Completion
{"action": "git_commit", "message": "fix: description"}  — Commit changes
{"action": "git_push"}                                    — Push to remote
{"action": "done", "reason": "summary of work done"}     — Signal completion
{"action": "close_bead", "reason": "work complete"}       — Close the bead

## Workflow

1. scope "." to see the project structure
2. read files to understand the code
3. search for relevant patterns
4. edit files (use exact old text from read output)
5. build to verify
6. test to verify
7. done when finished

## Rules

- Paths relative to project root
- For edit: "old" must match file content EXACTLY
- Only one action per response
- Always build after editing
- Respond with JSON only — no text outside the JSON object

LESSONS_PLACEHOLDER

## Example

Task: Fix a bug in the status check.

Response:
{"action": "search", "query": "isProviderHealthy", "notes": "Finding the status check function"}
`

// BuildSimpleJSONPrompt replaces the lessons placeholder.
func BuildSimpleJSONPrompt(lessons string, progressContext string) string {
	prompt := SimpleJSONPrompt

	if lessons != "" {
		prompt = strings.Replace(prompt, "LESSONS_PLACEHOLDER", "## Lessons Learned\n\n"+lessons, 1)
	} else {
		prompt = strings.Replace(prompt, "LESSONS_PLACEHOLDER", "", 1)
	}

	if progressContext != "" {
		prompt += "\n## Progress Context\n\n" + progressContext + "\n"
	}

	return prompt
}
