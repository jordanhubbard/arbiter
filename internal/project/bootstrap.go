package project

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BootstrapRequest contains the parameters for bootstrapping a new project
type BootstrapRequest struct {
	GitHubURL string `json:"github_url"`
	Name      string `json:"name"`
	Branch    string `json:"branch"`
	PRDText   string `json:"prd_text,omitempty"`   // PRD as text
	PRDFile   []byte `json:"prd_file,omitempty"`   // Or uploaded file content
}

// BootstrapResult contains the result of a bootstrap operation
type BootstrapResult struct {
	ProjectID   string `json:"project_id"`
	Status      string `json:"status"`
	InitialBead string `json:"initial_bead_id,omitempty"` // PM's PRD expansion bead
	Error       string `json:"error,omitempty"`
}

// BootstrapService handles project bootstrap operations
type BootstrapService struct {
	projectManager *Manager
	templateDir    string
	workspaceDir   string
}

// NewBootstrapService creates a new bootstrap service
func NewBootstrapService(pm *Manager, templateDir, workspaceDir string) *BootstrapService {
	return &BootstrapService{
		projectManager: pm,
		templateDir:    templateDir,
		workspaceDir:   workspaceDir,
	}
}

// Bootstrap creates a new project from a PRD
func (bs *BootstrapService) Bootstrap(ctx context.Context, req BootstrapRequest) (*BootstrapResult, error) {
	// Validate request
	if req.GitHubURL == "" || req.Name == "" || req.Branch == "" {
		return nil, fmt.Errorf("github_url, name, and branch are required")
	}
	if req.PRDText == "" && len(req.PRDFile) == 0 {
		return nil, fmt.Errorf("either prd_text or prd_file must be provided")
	}

	// Extract PRD content
	prdContent := req.PRDText
	if prdContent == "" {
		prdContent = string(req.PRDFile)
	}

	// Generate project ID
	projectID := fmt.Sprintf("proj-%d", time.Now().Unix())

	// Create project directory in workspace
	projectPath := filepath.Join(bs.workspaceDir, projectID)
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	// Clone or initialize repository
	if err := bs.cloneRepository(ctx, req.GitHubURL, req.Branch, projectPath); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Initialize project structure
	if err := bs.initializeProjectStructure(ctx, projectPath, prdContent); err != nil {
		return nil, fmt.Errorf("failed to initialize project structure: %w", err)
	}

	// Initialize beads
	if err := bs.initializeBeads(ctx, projectPath); err != nil {
		return nil, fmt.Errorf("failed to initialize beads: %w", err)
	}

	// Commit initial structure
	if err := bs.commitInitialStructure(ctx, projectPath); err != nil {
		return nil, fmt.Errorf("failed to commit initial structure: %w", err)
	}

	// Register project with Loom
	project, err := bs.projectManager.CreateProject(req.Name, projectPath, req.Branch, ".beads", map[string]string{
		"bootstrap":    "true",
		"github_url":   req.GitHubURL,
		"description": fmt.Sprintf("Bootstrapped project from PRD"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register project: %w", err)
	}

	// Create PM bead for PRD expansion
	initialBeadID, err := bs.createPMExpandPRDBead(ctx, projectPath, project.ID)
	if err != nil {
		// Log warning but don't fail bootstrap
		fmt.Printf("Warning: Failed to create PM bead: %v\n", err)
	}

	return &BootstrapResult{
		ProjectID:   project.ID,
		Status:      "ready",
		InitialBead: initialBeadID,
	}, nil
}

// cloneRepository clones or initializes a git repository
func (bs *BootstrapService) cloneRepository(ctx context.Context, gitURL, branch, destPath string) error {
	// Try to clone the repository
	cmd := exec.CommandContext(ctx, "git", "clone", "--branch", branch, gitURL, destPath)
	if err := cmd.Run(); err != nil {
		// If clone fails, try to initialize and set remote
		cmd = exec.CommandContext(ctx, "git", "init", destPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}

		// Set remote
		cmd = exec.CommandContext(ctx, "git", "-C", destPath, "remote", "add", "origin", gitURL)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}

		// Create initial branch
		cmd = exec.CommandContext(ctx, "git", "-C", destPath, "checkout", "-b", branch)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}
	}

	return nil
}

// initializeProjectStructure creates the project directory structure and template files
func (bs *BootstrapService) initializeProjectStructure(ctx context.Context, projectPath, prdContent string) error {
	// Create plans directory
	plansDir := filepath.Join(projectPath, "plans")
	if err := os.MkdirAll(plansDir, 0755); err != nil {
		return fmt.Errorf("failed to create plans directory: %w", err)
	}

	// Write BOOTSTRAP.md with initial PRD
	bootstrapPath := filepath.Join(plansDir, "BOOTSTRAP.md")
	if err := os.WriteFile(bootstrapPath, []byte(prdContent), 0644); err != nil {
		return fmt.Errorf("failed to write BOOTSTRAP.md: %w", err)
	}

	// Copy template files
	if err := bs.copyTemplateFiles(projectPath); err != nil {
		return fmt.Errorf("failed to copy template files: %w", err)
	}

	return nil
}

// copyTemplateFiles copies template configuration files to the project
func (bs *BootstrapService) copyTemplateFiles(projectPath string) error {
	// Create settings.json from template
	settingsContent := `{
  "mcpServers": {
    "responsible-vibe-mcp": {
      "command": "npx",
      "args": ["-y", "responsible-vibe-mcp"]
    }
  },
  "workflowMode": "guided",
  "enableReviews": true
}
`
	settingsPath := filepath.Join(projectPath, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	// Create .mcp.json from template
	mcpContent := `{
  "mcpServers": {
    "responsible-vibe-mcp": {
      "command": "npx",
      "args": ["-y", "responsible-vibe-mcp"]
    }
  }
}
`
	mcpPath := filepath.Join(projectPath, ".mcp.json")
	if err := os.WriteFile(mcpPath, []byte(mcpContent), 0644); err != nil {
		return fmt.Errorf("failed to write .mcp.json: %w", err)
	}

	return nil
}

// initializeBeads initializes the beads system for the project
func (bs *BootstrapService) initializeBeads(ctx context.Context, projectPath string) error {
	cmd := exec.CommandContext(ctx, "bd", "init")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run bd init: %w", err)
	}

	return nil
}

// commitInitialStructure commits the initial project structure
func (bs *BootstrapService) commitInitialStructure(ctx context.Context, projectPath string) error {
	// Stage all files
	cmd := exec.CommandContext(ctx, "git", "-C", projectPath, "add", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	// Commit
	commitMsg := "chore: initialize project with initial PRD\n\nBootstrapped by Loom\nCo-Authored-By: Loom <noreply@loom.dev>"
	cmd = exec.CommandContext(ctx, "git", "-C", projectPath, "commit", "-m", commitMsg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// createPMExpandPRDBead creates the initial PM bead for PRD expansion
func (bs *BootstrapService) createPMExpandPRDBead(ctx context.Context, projectPath, projectID string) (string, error) {
	// Create bead using bd CLI
	cmd := exec.CommandContext(ctx, "bd", "create",
		"--title", "[Bootstrap] Expand PRD with Best Practices",
		"--type", "task",
		"--priority", "0",
		"--description", "Transform the initial PRD into a comprehensive, actionable PRD",
	)
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create PM bead: %w (output: %s)", err, string(output))
	}

	// Parse bead ID from output (format: "✓ Created issue: ac-xxxx")
	outputStr := string(output)
	beadID := ""
	if idx := len(outputStr) - 8; idx > 0 && len(outputStr) >= 8 {
		// Extract last 7-8 characters which should be the bead ID
		beadID = outputStr[len(outputStr)-8:]
		if beadID[0] == '\n' {
			beadID = beadID[1:]
		}
	}

	// Tag the bead
	if beadID != "" {
		cmd = exec.CommandContext(ctx, "bd", "update", beadID, "--tags", "bootstrap,prd-expansion")
		cmd.Dir = projectPath
		_ = cmd.Run() // Ignore errors on tagging
	}

	// Commit the bead
	cmd = exec.CommandContext(ctx, "git", "-C", projectPath, "add", ".beads/")
	_ = cmd.Run()
	
	cmd = exec.CommandContext(ctx, "git", "-C", projectPath, "commit", "-m", 
		"chore: create PM bead for PRD expansion\n\nCo-Authored-By: Loom <noreply@loom.dev>")
	_ = cmd.Run()

	return beadID, nil
}

// CreateEpicBreakdownBead creates the epic/story breakdown bead (called by PM after PRD expansion)
func (bs *BootstrapService) CreateEpicBreakdownBead(ctx context.Context, projectPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "bd", "create",
		"--title", "[Bootstrap] Create Epics and Stories from PRD",
		"--type", "task",
		"--priority", "0",
		"--description", `Break down the comprehensive PRD into actionable epics and stories as beads.

## Input
- Comprehensive PRD: plans/ORIGINAL_PRD.md
- responsible-vibe-mcp guidance

## Tasks
1. Identify major features (epics)
2. Break down each epic into user stories (tasks)
3. Create bead hierarchy:
   - Epic beads (type: epic, P1-P2)
   - Story beads (type: task, P2-P3, parent: epic-id)
4. Assign beads to appropriate roles:
   - UI/UX work → web-designer or web-designer-engineer
   - Core engineering → engineering-manager
   - Infrastructure → devops-engineer
   - Testing strategy → qa-engineer
5. Set dependencies between beads
6. Ensure MVP features are P1, enhancements are P2-P3

## Acceptance Criteria
- All major features have epic beads
- Epics broken into concrete, actionable story beads
- Beads assigned to appropriate agent roles
- Dependencies set correctly (blockers, blocked-by)
- Clear acceptance criteria on each story bead

## Guidelines
- Follow responsible-vibe-mcp decomposition best practices
- Keep stories small and focused (1-3 days max)
- Set realistic priorities based on MVP definition
- Include documentation and testing stories`,
	)
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create epic breakdown bead: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}

// CreateCEODemoBead creates the CEO demo/review bead when MVP is ready
func (bs *BootstrapService) CreateCEODemoBead(ctx context.Context, projectPath, projectID string, completedFeatures []string) (string, error) {
	featuresText := "- " + fmt.Sprintf("%v", completedFeatures)
	
	description := fmt.Sprintf(`The initial implementation is ready for review and testing.

## What's Been Built
%s

## How to Launch
See docs/DEPLOYMENT.md or README.md for instructions.

Example:
` + "```bash\n" + `npm install
npm run dev
# Open http://localhost:3000
` + "```\n" + `

## Testing Checklist
- [ ] Application launches successfully
- [ ] Core features work as expected
- [ ] UI is presentable and functional
- [ ] No critical bugs observed
- [ ] Meets MVP acceptance criteria

## Next Steps
Choose one:
1. **Approve and Complete**: Mark project MVP as done
2. **Request Changes**: Create follow-up beads for improvements
3. **Pivot**: Major changes needed, update PRD and restart

## Feedback
[Provide detailed feedback here]`, featuresText)

	cmd := exec.CommandContext(ctx, "bd", "create",
		"--title", "[Demo] Review and Test Application",
		"--type", "decision",
		"--priority", "0",
		"--assignee", "ceo",
		"--description", description,
	)
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create CEO demo bead: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}
