# Arbiter

An AI coding agent orchestrator, dispatcher, and automatic decision maker written in Go.

## Overview

Arbiter is a sophisticated system designed to orchestrate multiple AI coding agents, intelligently dispatch tasks, and automatically make decisions about which agent should handle which task. It provides a framework for managing coding tasks across different types of specialized AI agents.

## Features

- **Agent Orchestration**: Manage multiple AI coding agents with different capabilities
- **Intelligent Task Dispatch**: Automatically route tasks to the most appropriate agent
- **Decision Making**: Smart algorithms to evaluate task priority and agent selection
- **Multiple Agent Types**:
  - General purpose agents for broad coding tasks
  - Specialist agents with domain-specific expertise
  - Reviewer agents for code quality and testing

## Architecture

Arbiter consists of several key components:

- **Types** (`pkg/types`): Core data structures and interfaces
- **Agent** (`internal/agent`): Agent implementation and execution logic
- **Dispatcher** (`internal/dispatcher`): Task distribution and agent management
- **Decision Maker** (`internal/decision`): Intelligent agent selection and priority evaluation

## Getting Started

### Prerequisites

- Go 1.16 or higher

### Installation

```bash
git clone https://github.com/jordanhubbard/arbiter.git
cd arbiter
go build -o arbiter ./cmd/arbiter
```

### Running Arbiter

```bash
./arbiter
```

This will run the example demonstration which:
1. Registers three different types of agents
2. Creates sample tasks with different priorities
3. Automatically assigns tasks to appropriate agents
4. Executes the tasks and reports results

## Usage Example

```go
package main

import (
    "context"
    "github.com/jordanhubbard/arbiter/internal/agent"
    "github.com/jordanhubbard/arbiter/internal/decision"
    "github.com/jordanhubbard/arbiter/internal/dispatcher"
    "github.com/jordanhubbard/arbiter/pkg/types"
)

func main() {
    // Initialize decision maker
    decisionMaker := decision.NewSimpleMaker()
    
    // Initialize dispatcher
    disp := dispatcher.NewTaskDispatcher(decisionMaker)
    
    // Register an agent
    agent := &types.Agent{
        ID:           "agent-1",
        Name:         "My Agent",
        Type:         types.AgentTypeGeneral,
        Capabilities: []string{"coding", "testing"},
        Status:       types.AgentStatusIdle,
    }
    disp.RegisterAgent(agent)
    
    // Create and assign a task
    task := &types.Task{
        ID:          "task-1",
        Description: "Write unit tests",
        Priority:    5,
        Status:      types.TaskStatusPending,
    }
    
    ctx := context.Background()
    assignedAgent, err := disp.AssignTask(ctx, task)
    if err != nil {
        // Handle error
    }
    
    // Execute the task (in a real implementation)
    // ...
}
```

## Testing

Run the test suite:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

## Project Structure

```
arbiter/
├── cmd/
│   └── arbiter/          # Main application entry point
│       └── main.go
├── internal/
│   ├── agent/            # Agent implementation
│   │   ├── agent.go
│   │   └── agent_test.go
│   ├── dispatcher/       # Task dispatcher
│   │   ├── dispatcher.go
│   │   └── dispatcher_test.go
│   └── decision/         # Decision making logic
│       ├── decision.go
│       └── decision_test.go
├── pkg/
│   └── types/            # Core types and interfaces
│       ├── types.go
│       └── types_test.go
├── go.mod
├── LICENSE
└── README.md
```

## Agent Types

### General Agent
General purpose agents that can handle a wide variety of coding tasks including:
- Code writing
- Documentation
- Testing

### Specialist Agent
Domain-specific agents with expertise in particular technologies or languages:
- Python specialist
- JavaScript specialist
- Database specialist
- etc.

### Reviewer Agent
Agents focused on code quality and review:
- Code review
- Testing
- Quality assurance

## Decision Making

The decision maker uses a scoring algorithm to select the best agent for each task:

1. **Capability Matching**: Agents with capabilities matching task keywords receive higher scores
2. **Agent Type Preference**: Specialist agents receive bonus points for focused tasks
3. **Priority Evaluation**: Tasks are automatically prioritized based on keywords like "urgent", "critical", "bug"

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

See the [LICENSE](LICENSE) file for details.
