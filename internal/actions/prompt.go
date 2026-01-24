package actions

const ActionPrompt = `
You must respond with strict JSON only. Do not include any surrounding text.

The response must be a single JSON object with this shape:
{
  "actions": [
    {
      "type": "ask_followup|read_code|edit_code|run_command|create_bead|escalate_ceo",
      "question": "string",
      "path": "string",
      "patch": "string",
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

Only include fields required for the selected action type.
`
