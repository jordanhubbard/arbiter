package actions

const ActionPrompt = `
You must respond with strict JSON only. Do not include any surrounding text or model reasoning markers (e.g. <think>).

The response must be a single JSON object with this shape:
{
  "actions": [
    {
      "type": "ask_followup|read_code|read_file|read_tree|search_text|write_file|edit_code|apply_patch|git_status|git_diff|run_command|create_bead|close_bead|escalate_ceo",
      "question": "string",
      "path": "string",
      "content": "string",
      "patch": "string",
      "query": "string",
      "max_depth": 2,
      "limit": 100,
      "command": "string",
      "working_dir": "string",
      "bead": {
        "title": "string",
        "description": "string",
        "priority": 0,
        "type": "task",
        "project_id": "string",
        "tags": ["string"]
      },
      "bead_id": "string",
      "reason": "string",
      "returned_to": "string"
    }
  ],
  "notes": "string"
}

## Action Types

- read_code/read_file: Read file contents. Requires: path
- read_tree: List directory. Requires: path
- search_text: Search for text. Requires: query, optional: path
- write_file: Write entire file contents. Requires: path, content (PREFERRED for code changes)
- edit_code/apply_patch: Apply unified diff patch. Requires: path, patch (in unified diff format)
- git_status/git_diff: Check git state
- run_command: Execute shell command. Requires: command
- create_bead: Create work item. Requires: bead object
- close_bead: Close/complete a bead. Requires: bead_id, optional: reason
- escalate_ceo: Escalate decision. Requires: bead_id, reason
- ask_followup: Ask clarifying question. Requires: question

## Writing Code Changes

For code changes, PREFER write_file over edit_code/apply_patch:
- write_file takes the complete new file content
- edit_code/apply_patch requires a valid unified diff (git format)

Example write_file:
{
  "actions": [{"type": "write_file", "path": "src/config.go", "content": "package src\n\nconst Version = \"1.0.0\"\n"}],
  "notes": "Updated version"
}

Only include fields required for the selected action type.
Paths are always relative to the project root.
`
