package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jordanhubbard/arbiter/internal/agent"
	"github.com/jordanhubbard/arbiter/internal/decision"
	"github.com/jordanhubbard/arbiter/internal/dispatcher"
	"github.com/jordanhubbard/arbiter/pkg/types"
)

func main() {
	fmt.Println("Arbiter - AI Coding Agent Orchestrator")
	fmt.Println("======================================")

	// Initialize decision maker
	decisionMaker := decision.NewSimpleMaker()

	// Initialize dispatcher
	disp := dispatcher.NewTaskDispatcher(decisionMaker)

	// Register some example agents
	registerExampleAgents(disp)

	// Create and dispatch example tasks
	ctx := context.Background()
	runExampleTasks(ctx, disp)
}

func registerExampleAgents(disp *dispatcher.TaskDispatcher) {
	// Register general purpose agent
	generalAgent := &types.Agent{
		ID:           "agent-1",
		Name:         "General Agent 1",
		Type:         types.AgentTypeGeneral,
		Capabilities: []string{"coding", "documentation", "testing"},
		Status:       types.AgentStatusIdle,
	}
	disp.RegisterAgent(generalAgent)

	// Register specialist agent
	specialistAgent := &types.Agent{
		ID:           "agent-2",
		Name:         "Python Specialist",
		Type:         types.AgentTypeSpecialist,
		Capabilities: []string{"python", "coding", "debugging"},
		Status:       types.AgentStatusIdle,
	}
	disp.RegisterAgent(specialistAgent)

	// Register reviewer agent
	reviewerAgent := &types.Agent{
		ID:           "agent-3",
		Name:         "Code Reviewer",
		Type:         types.AgentTypeReviewer,
		Capabilities: []string{"review", "testing", "quality"},
		Status:       types.AgentStatusIdle,
	}
	disp.RegisterAgent(reviewerAgent)

	fmt.Printf("Registered %d agents\n\n", len(disp.GetAgents()))
}

func runExampleTasks(ctx context.Context, disp *dispatcher.TaskDispatcher) {
	tasks := []*types.Task{
		{
			ID:          "task-1",
			Description: "Fix urgent bug in Python code",
			Priority:    8,
			Status:      types.TaskStatusPending,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "task-2",
			Description: "Write documentation for API",
			Priority:    3,
			Status:      types.TaskStatusPending,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "task-3",
			Description: "Review code for quality issues",
			Priority:    5,
			Status:      types.TaskStatusPending,
			CreatedAt:   time.Now(),
		},
	}

	fmt.Println("Dispatching tasks:")
	fmt.Println("------------------")

	for _, task := range tasks {
		// Evaluate priority
		decisionMaker := decision.NewSimpleMaker()
		evaluatedPriority := decisionMaker.EvaluatePriority(task)
		task.Priority = evaluatedPriority

		fmt.Printf("\nTask: %s\n", task.ID)
		fmt.Printf("  Description: %s\n", task.Description)
		fmt.Printf("  Priority: %d\n", task.Priority)

		// Assign task to agent
		assignedAgent, err := disp.AssignTask(ctx, task)
		if err != nil {
			log.Printf("  Error: Failed to assign task: %v\n", err)
			continue
		}

		fmt.Printf("  Assigned to: %s (%s)\n", assignedAgent.Name, assignedAgent.Type)
		fmt.Printf("  Agent capabilities: %v\n", assignedAgent.Capabilities)

		// Simulate task execution
		baseAgent := agent.NewBaseAgent(
			assignedAgent.ID,
			assignedAgent.Name,
			assignedAgent.Type,
			assignedAgent.Capabilities,
		)

		result, err := baseAgent.Execute(ctx, task)
		if err != nil {
			log.Printf("  Error: Task execution failed: %v\n", err)
			task.Status = types.TaskStatusFailed
		} else {
			task.Status = types.TaskStatusCompleted
			task.Result = result
			fmt.Printf("  Status: %s\n", task.Status)
			fmt.Printf("  Result: %s\n", result.Message)
		}

		// Reset agent status
		assignedAgent.Status = types.AgentStatusIdle
		assignedAgent.CurrentTask = nil
	}

	fmt.Println("\n======================================")
	fmt.Println("All tasks processed successfully!")
}
