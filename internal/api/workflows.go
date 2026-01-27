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

// handleWorkflowAnalytics handles GET /api/v1/workflows/analytics - get workflow metrics
func (s *Server) handleWorkflowAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get workflow engine
	engine := s.agenticorp.GetWorkflowEngine()
	if engine == nil {
		http.Error(w, "Workflow engine not available", http.StatusServiceUnavailable)
		return
	}

	// Query database for analytics
	// Get workflow execution counts by status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM workflow_executions
		GROUP BY status
	`
	statusRows, err := s.agenticorp.GetDatabase().DB().Query(statusQuery)
	if err != nil {
		http.Error(w, "Failed to query execution stats: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer statusRows.Close()

	statusCounts := make(map[string]int)
	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err == nil {
			statusCounts[status] = count
		}
	}

	// Get workflow execution counts by workflow type
	typeQuery := `
		SELECT w.workflow_type, COUNT(*) as count
		FROM workflow_executions we
		JOIN workflows w ON we.workflow_id = w.id
		GROUP BY w.workflow_type
	`
	typeRows, err := s.agenticorp.GetDatabase().DB().Query(typeQuery)
	if err != nil {
		http.Error(w, "Failed to query type stats: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer typeRows.Close()

	typeCounts := make(map[string]int)
	for typeRows.Next() {
		var workflowType string
		var count int
		if err := typeRows.Scan(&workflowType, &count); err == nil {
			typeCounts[workflowType] = count
		}
	}

	// Get average cycle counts
	cycleQuery := `
		SELECT AVG(cycle_count) as avg_cycles, MAX(cycle_count) as max_cycles
		FROM workflow_executions
		WHERE status = 'active' OR status = 'completed'
	`
	var avgCycles, maxCycles float64
	err = s.agenticorp.GetDatabase().DB().QueryRow(cycleQuery).Scan(&avgCycles, &maxCycles)
	if err != nil {
		avgCycles = 0
		maxCycles = 0
	}

	// Get escalation rate
	escalationQuery := `
		SELECT
			COUNT(CASE WHEN status = 'escalated' THEN 1 END) as escalated_count,
			COUNT(*) as total_count
		FROM workflow_executions
	`
	var escalatedCount, totalCount int
	err = s.agenticorp.GetDatabase().DB().QueryRow(escalationQuery).Scan(&escalatedCount, &totalCount)
	if err != nil {
		escalatedCount = 0
		totalCount = 0
	}

	escalationRate := 0.0
	if totalCount > 0 {
		escalationRate = float64(escalatedCount) / float64(totalCount) * 100
	}

	// Get recent executions
	recentQuery := `
		SELECT we.id, we.bead_id, we.workflow_id, we.current_node_key, we.status, we.cycle_count, we.started_at, w.name
		FROM workflow_executions we
		JOIN workflows w ON we.workflow_id = w.id
		ORDER BY we.started_at DESC
		LIMIT 10
	`
	recentRows, err := s.agenticorp.GetDatabase().DB().Query(recentQuery)
	if err != nil {
		http.Error(w, "Failed to query recent executions: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer recentRows.Close()

	type RecentExecution struct {
		ID              string `json:"id"`
		BeadID          string `json:"bead_id"`
		WorkflowID      string `json:"workflow_id"`
		WorkflowName    string `json:"workflow_name"`
		CurrentNodeKey  string `json:"current_node_key"`
		Status          string `json:"status"`
		CycleCount      int    `json:"cycle_count"`
		StartedAt       string `json:"started_at"`
	}

	recentExecutions := []RecentExecution{}
	for recentRows.Next() {
		var exec RecentExecution
		if err := recentRows.Scan(&exec.ID, &exec.BeadID, &exec.WorkflowID, &exec.CurrentNodeKey, &exec.Status, &exec.CycleCount, &exec.StartedAt, &exec.WorkflowName); err == nil {
			recentExecutions = append(recentExecutions, exec)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status_counts":      statusCounts,
		"type_counts":        typeCounts,
		"average_cycles":     avgCycles,
		"max_cycles":         maxCycles,
		"escalation_rate":    escalationRate,
		"total_executions":   totalCount,
		"escalated_count":    escalatedCount,
		"recent_executions":  recentExecutions,
	})
}
