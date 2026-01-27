package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jordanhubbard/agenticorp/internal/workflow"
)

// handleWorkflows handles GET /api/v1/workflows - list all workflows
func (s *Server) handleWorkflows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	workflowType := r.URL.Query().Get("type")
	projectID := r.URL.Query().Get("project_id")

	// Get workflow engine
	engine := s.agenticorp.GetWorkflowEngine()
	if engine == nil {
		http.Error(w, "Workflow engine not available", http.StatusServiceUnavailable)
		return
	}

	// List workflows
	workflows, err := engine.GetDatabase().ListWorkflows(workflowType, projectID)
	if err != nil {
		http.Error(w, "Failed to list workflows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": workflows,
		"count":     len(workflows),
	})
}

// handleWorkflow handles GET /api/v1/workflows/{id} - get workflow details
func (s *Server) handleWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract workflow ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/workflows/")
	workflowID := strings.Split(path, "/")[0]

	if workflowID == "" {
		http.Error(w, "Workflow ID required", http.StatusBadRequest)
		return
	}

	// Get workflow engine
	engine := s.agenticorp.GetWorkflowEngine()
	if engine == nil {
		http.Error(w, "Workflow engine not available", http.StatusServiceUnavailable)
		return
	}

	// Get workflow
	wf, err := engine.GetDatabase().GetWorkflow(workflowID)
	if err != nil {
		http.Error(w, "Failed to get workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if wf == nil {
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

// handleWorkflowExecutions handles GET /api/v1/workflows/executions - list workflow executions
func (s *Server) handleWorkflowExecutions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	status := r.URL.Query().Get("status")
	workflowID := r.URL.Query().Get("workflow_id")
	beadID := r.URL.Query().Get("bead_id")

	// Get workflow engine
	engine := s.agenticorp.GetWorkflowEngine()
	if engine == nil {
		http.Error(w, "Workflow engine not available", http.StatusServiceUnavailable)
		return
	}

	// If bead_id specified, get that specific execution
	if beadID != "" {
		execution, err := engine.GetDatabase().GetWorkflowExecutionByBeadID(beadID)
		if err != nil {
			http.Error(w, "Failed to get execution: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if execution == nil {
			http.Error(w, "Execution not found", http.StatusNotFound)
			return
		}

		// Get workflow history
		history, err := engine.GetDatabase().ListWorkflowHistory(execution.ID)
		if err != nil {
			history = nil // Continue without history
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"execution": execution,
			"history":   history,
		})
		return
	}

	// Query database for executions matching filters
	// For now, we'll return all active executions as we don't have a generic ListExecutions method
	// This is a simplified implementation - in production, you'd add proper filtering

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "List all executions not yet implemented",
		"status":      status,
		"workflow_id": workflowID,
	})
}

// handleBeadWorkflow handles GET /api/v1/beads/workflow?bead_id={id} - get workflow for a bead
func (s *Server) handleBeadWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	beadID := r.URL.Query().Get("bead_id")
	if beadID == "" {
		http.Error(w, "bead_id parameter required", http.StatusBadRequest)
		return
	}

	// Get workflow engine
	engine := s.agenticorp.GetWorkflowEngine()
	if engine == nil {
		http.Error(w, "Workflow engine not available", http.StatusServiceUnavailable)
		return
	}

	// Get workflow execution for this bead
	execution, err := engine.GetDatabase().GetWorkflowExecutionByBeadID(beadID)
	if err != nil {
		http.Error(w, "Failed to get execution: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if execution == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "No workflow execution found for this bead",
			"bead_id": beadID,
		})
		return
	}

	// Get workflow details
	wf, err := engine.GetDatabase().GetWorkflow(execution.WorkflowID)
	if err != nil {
		http.Error(w, "Failed to get workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get execution history
	history, err := engine.GetDatabase().ListWorkflowHistory(execution.ID)
	if err != nil {
		history = nil // Continue without history
	}

	// Get current node if any
	var currentNode *workflow.WorkflowNode
	if execution.CurrentNodeKey != "" {
		node, err := engine.GetCurrentNode(execution.ID)
		if err == nil {
			currentNode = node
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bead_id":      beadID,
		"workflow":     wf,
		"execution":    execution,
		"current_node": currentNode,
		"history":      history,
	})
}
