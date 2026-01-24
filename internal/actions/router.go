package actions

import (
	"context"
	"fmt"

	"github.com/jordanhubbard/agenticorp/internal/executor"
	"github.com/jordanhubbard/agenticorp/pkg/models"
)

type BeadCreator interface {
	CreateBead(title, description string, priority models.BeadPriority, beadType, projectID string) (*models.Bead, error)
}

type BeadEscalator interface {
	EscalateBeadToCEO(beadID, reason, returnedTo string) (*models.DecisionBead, error)
}

type CommandExecutor interface {
	ExecuteCommand(ctx context.Context, req executor.ExecuteCommandRequest) (*executor.ExecuteCommandResult, error)
}

type ActionLogger interface {
	LogAction(ctx context.Context, actx ActionContext, action Action, result Result)
}

type ActionContext struct {
	AgentID   string
	BeadID    string
	ProjectID string
}

type Result struct {
	ActionType string                 `json:"action_type"`
	Status     string                 `json:"status"`
	Message    string                 `json:"message"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type Router struct {
	Beads      BeadCreator
	Escalator  BeadEscalator
	Commands   CommandExecutor
	Logger     ActionLogger
	BeadType   string
	BeadTags   []string
	DefaultP0  bool
}

func (r *Router) Execute(ctx context.Context, env *ActionEnvelope, actx ActionContext) ([]Result, error) {
	if env == nil {
		return nil, fmt.Errorf("action envelope is nil")
	}

	results := make([]Result, 0, len(env.Actions))
	for _, action := range env.Actions {
		result := r.executeAction(ctx, action, actx)
		if r.Logger != nil {
			r.Logger.LogAction(ctx, actx, action, result)
		}
		results = append(results, result)
	}

	return results, nil
}

func (r *Router) AutoFileParseFailure(ctx context.Context, actx ActionContext, err error, raw string) Result {
	if r.Beads == nil {
		return Result{ActionType: ActionCreateBead, Status: "error", Message: "bead creator not configured"}
	}
	priority := models.BeadPriority(0)
	if !r.DefaultP0 {
		priority = models.BeadPriority(2)
	}
	description := fmt.Sprintf("Failed to parse strict JSON actions.\n\nError:\n%s\n\nRaw response:\n%s", err.Error(), raw)
	bead, beadErr := r.Beads.CreateBead("Action parse failed", description, priority, "bug", actx.ProjectID)
	if beadErr != nil {
		return Result{ActionType: ActionCreateBead, Status: "error", Message: beadErr.Error()}
	}
	result := Result{
		ActionType: ActionCreateBead,
		Status:     "executed",
		Message:    "auto-filed action parse failure",
		Metadata:   map[string]interface{}{"bead_id": bead.ID},
	}
	if r.Logger != nil {
		r.Logger.LogAction(ctx, actx, Action{Type: ActionCreateBead}, result)
	}
	return result
}

func (r *Router) executeAction(ctx context.Context, action Action, actx ActionContext) Result {
	switch action.Type {
	case ActionAskFollowup:
		return r.createBeadFromAction("Follow-up question", action.Question, actx)
	case ActionReadCode:
		return r.createBeadFromAction("Read code", action.Path, actx)
	case ActionEditCode:
		return r.createBeadFromAction("Edit code", fmt.Sprintf("%s\n\nPatch:\n%s", action.Path, action.Patch), actx)
	case ActionRunCommand:
		if r.Commands == nil {
			return r.createBeadFromAction("Run command", action.Command, actx)
		}
		req := executor.ExecuteCommandRequest{
			AgentID:    actx.AgentID,
			BeadID:     actx.BeadID,
			ProjectID:  actx.ProjectID,
			Command:    action.Command,
			WorkingDir: action.WorkingDir,
			Context: map[string]interface{}{
				"action_type": action.Type,
				"reason":      action.Reason,
			},
		}
		res, err := r.Commands.ExecuteCommand(ctx, req)
		if err != nil {
			return Result{ActionType: action.Type, Status: "error", Message: err.Error()}
		}
		return Result{
			ActionType: action.Type,
			Status:     "executed",
			Message:    "command executed",
			Metadata: map[string]interface{}{
				"command_id": res.ID,
				"exit_code":  res.ExitCode,
			},
		}
	case ActionCreateBead:
		if action.Bead == nil {
			return Result{ActionType: action.Type, Status: "error", Message: "missing bead payload"}
		}
		if r.Beads == nil {
			return Result{ActionType: action.Type, Status: "error", Message: "bead creator not configured"}
		}
		beadType := action.Bead.Type
		if beadType == "" {
			beadType = r.BeadType
		}
		if beadType == "" {
			beadType = "task"
		}
		priority := models.BeadPriority(action.Bead.Priority)
		bead, err := r.Beads.CreateBead(action.Bead.Title, action.Bead.Description, priority, beadType, action.Bead.ProjectID)
		if err != nil {
			return Result{ActionType: action.Type, Status: "error", Message: err.Error()}
		}
		return Result{
			ActionType: action.Type,
			Status:     "executed",
			Message:    "bead created",
			Metadata:   map[string]interface{}{"bead_id": bead.ID},
		}
	case ActionEscalateCEO:
		if r.Escalator == nil {
			return Result{ActionType: action.Type, Status: "error", Message: "escalator not configured"}
		}
		decision, err := r.Escalator.EscalateBeadToCEO(action.BeadID, action.Reason, action.ReturnedTo)
		if err != nil {
			return Result{ActionType: action.Type, Status: "error", Message: err.Error()}
		}
		return Result{
			ActionType: action.Type,
			Status:     "executed",
			Message:    "escalated to CEO",
			Metadata:   map[string]interface{}{"decision_id": decision.ID},
		}
	default:
		return Result{ActionType: action.Type, Status: "error", Message: "unsupported action"}
	}
}

func (r *Router) createBeadFromAction(title, detail string, actx ActionContext) Result {
	if r.Beads == nil {
		return Result{ActionType: ActionCreateBead, Status: "error", Message: "bead creator not configured"}
	}
	beadType := r.BeadType
	if beadType == "" {
		beadType = "task"
	}
	priority := models.BeadPriority(2)
	if r.DefaultP0 {
		priority = models.BeadPriority(0)
	}
	bead, err := r.Beads.CreateBead(title, detail, priority, beadType, actx.ProjectID)
	if err != nil {
		return Result{ActionType: ActionCreateBead, Status: "error", Message: err.Error()}
	}
	return Result{
		ActionType: ActionCreateBead,
		Status:     "executed",
		Message:    "bead created",
		Metadata:   map[string]interface{}{"bead_id": bead.ID},
	}
}
