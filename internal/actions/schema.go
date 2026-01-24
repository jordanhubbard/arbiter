package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	ActionAskFollowup = "ask_followup"
	ActionReadCode    = "read_code"
	ActionEditCode    = "edit_code"
	ActionRunCommand  = "run_command"
	ActionCreateBead  = "create_bead"
	ActionEscalateCEO = "escalate_ceo"
)

type ActionEnvelope struct {
	Actions []Action `json:"actions"`
	Notes   string   `json:"notes,omitempty"`
}

type Action struct {
	Type string `json:"type"`

	Question string `json:"question,omitempty"`

	Path  string `json:"path,omitempty"`
	Patch string `json:"patch,omitempty"`

	Command    string `json:"command,omitempty"`
	WorkingDir string `json:"working_dir,omitempty"`

	Bead *BeadPayload `json:"bead,omitempty"`

	BeadID     string `json:"bead_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
	ReturnedTo string `json:"returned_to,omitempty"`
}

type BeadPayload struct {
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Priority    int               `json:"priority,omitempty"`
	Type        string            `json:"type,omitempty"`
	ProjectID   string            `json:"project_id"`
	Tags        []string          `json:"tags,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
}

func DecodeStrict(payload []byte) (*ActionEnvelope, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var env ActionEnvelope
	if err := decoder.Decode(&env); err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, errors.New("unexpected trailing JSON tokens")
	}
	if err := Validate(&env); err != nil {
		return nil, err
	}
	return &env, nil
}

func Validate(env *ActionEnvelope) error {
	if env == nil {
		return errors.New("action envelope is nil")
	}
	if len(env.Actions) == 0 {
		return errors.New("action envelope must include at least one action")
	}

	for idx, action := range env.Actions {
		if action.Type == "" {
			return fmt.Errorf("action[%d] missing type", idx)
		}
		if err := validateAction(action); err != nil {
			return fmt.Errorf("action[%d] %s", idx, err.Error())
		}
	}
	return nil
}

func validateAction(action Action) error {
	switch action.Type {
	case ActionAskFollowup:
		if action.Question == "" {
			return errors.New("ask_followup requires question")
		}
	case ActionReadCode:
		if action.Path == "" {
			return errors.New("read_code requires path")
		}
	case ActionEditCode:
		if action.Path == "" || action.Patch == "" {
			return errors.New("edit_code requires path and patch")
		}
	case ActionRunCommand:
		if action.Command == "" {
			return errors.New("run_command requires command")
		}
	case ActionCreateBead:
		if action.Bead == nil {
			return errors.New("create_bead requires bead payload")
		}
		if action.Bead.Title == "" || action.Bead.ProjectID == "" {
			return errors.New("create_bead requires bead.title and bead.project_id")
		}
	case ActionEscalateCEO:
		if action.BeadID == "" {
			return errors.New("escalate_ceo requires bead_id")
		}
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}

	return nil
}
