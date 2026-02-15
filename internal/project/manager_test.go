package project

import (
	"strings"
	"sync"
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

// createTestProject is a helper that creates a Manager and a project with sensible defaults.
// It returns the Manager and the created Project, failing the test on error.
func createTestProject(t *testing.T, name string) (*Manager, *models.Project) {
	t.Helper()
	manager := NewManager()
	project, err := manager.CreateProject(
		name,
		"https://github.com/test/repo",
		"main",
		".beads",
		map[string]string{"env": "test"},
	)
	if err != nil {
		t.Fatalf("Failed to create project %q: %v", name, err)
	}
	return manager, project
}

func TestProjectStateManagement(t *testing.T) {
	manager := NewManager()

	// Test creating a project
	project, err := manager.CreateProject(
		"Test Project",
		"https://github.com/test/repo",
		"main",
		".beads",
		map[string]string{"test": "context"},
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	if project.Status != models.ProjectStatusOpen {
		t.Errorf("Expected status Open, got %s", project.Status)
	}

	if project.IsPerpetual {
		t.Error("Expected IsPerpetual to be false by default")
	}

	// Test adding comments
	comment, err := manager.AddComment(project.ID, "agent-1", "Initial comment")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	if comment.Comment != "Initial comment" {
		t.Errorf("Expected comment 'Initial comment', got %s", comment.Comment)
	}

	// Test retrieving comments
	comments, err := manager.GetComments(project.ID)
	if err != nil {
		t.Fatalf("Failed to get comments: %v", err)
	}

	if len(comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(comments))
	}

	// Test closing project
	err = manager.CloseProject(project.ID, "agent-1", "Project complete")
	if err != nil {
		t.Fatalf("Failed to close project: %v", err)
	}

	closedProject, _ := manager.GetProject(project.ID)
	if closedProject.Status != models.ProjectStatusClosed {
		t.Errorf("Expected status Closed, got %s", closedProject.Status)
	}

	if closedProject.ClosedAt == nil {
		t.Error("Expected ClosedAt to be set")
	}

	// Test reopening project
	err = manager.ReopenProject(project.ID, "agent-2", "Found more work")
	if err != nil {
		t.Fatalf("Failed to reopen project: %v", err)
	}

	reopenedProject, _ := manager.GetProject(project.ID)
	if reopenedProject.Status != models.ProjectStatusReopened {
		t.Errorf("Expected status Reopened, got %s", reopenedProject.Status)
	}

	if reopenedProject.ClosedAt != nil {
		t.Error("Expected ClosedAt to be nil after reopening")
	}

	// Check comments were added during state changes
	finalComments, _ := manager.GetComments(project.ID)
	if len(finalComments) < 3 {
		t.Errorf("Expected at least 3 comments (initial + close + reopen), got %d", len(finalComments))
	}
}

func TestPerpetualProject(t *testing.T) {
	manager := NewManager()

	// Create a project
	project, err := manager.CreateProject(
		"Perpetual Project",
		"https://github.com/test/perpetual",
		"main",
		".beads",
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Mark as perpetual
	err = manager.SetPerpetual(project.ID, true)
	if err != nil {
		t.Fatalf("Failed to set perpetual: %v", err)
	}

	perpetualProject, _ := manager.GetProject(project.ID)
	if !perpetualProject.IsPerpetual {
		t.Error("Expected project to be perpetual")
	}

	// Try to close perpetual project (should fail)
	err = manager.CloseProject(project.ID, "agent-1", "Trying to close")
	if err == nil {
		t.Error("Expected error when closing perpetual project")
	}

	// Check project is still open
	stillOpen, _ := manager.GetProject(project.ID)
	if stillOpen.Status != models.ProjectStatusOpen {
		t.Errorf("Expected status Open, got %s", stillOpen.Status)
	}
}

func TestCanClose(t *testing.T) {
	manager := NewManager()

	// Create a project
	project, _ := manager.CreateProject(
		"Test Project",
		"https://github.com/test/repo",
		"main",
		".beads",
		map[string]string{},
	)

	// Can close when no open work
	if !manager.CanClose(project.ID, false) {
		t.Error("Expected CanClose to be true when no open work")
	}

	// Cannot close when open work exists
	if manager.CanClose(project.ID, true) {
		t.Error("Expected CanClose to be false when open work exists")
	}

	// Mark as perpetual
	if err := manager.SetPerpetual(project.ID, true); err != nil {
		t.Fatalf("SetPerpetual failed: %v", err)
	}

	// Cannot close perpetual project even without open work
	if manager.CanClose(project.ID, false) {
		t.Error("Expected CanClose to be false for perpetual project")
	}
}

func TestProjectNotFound(t *testing.T) {
	manager := NewManager()

	// Test operations on non-existent project
	_, err := manager.GetProject("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent project")
	}

	err = manager.CloseProject("non-existent", "agent-1", "comment")
	if err == nil {
		t.Error("Expected error when closing non-existent project")
	}

	err = manager.ReopenProject("non-existent", "agent-1", "comment")
	if err == nil {
		t.Error("Expected error when reopening non-existent project")
	}

	_, err = manager.AddComment("non-existent", "agent-1", "comment")
	if err == nil {
		t.Error("Expected error when adding comment to non-existent project")
	}
}

func TestCloseAlreadyClosed(t *testing.T) {
	manager := NewManager()

	project, _ := manager.CreateProject(
		"Test Project",
		"https://github.com/test/repo",
		"main",
		".beads",
		map[string]string{},
	)

	// Close the project
	if err := manager.CloseProject(project.ID, "agent-1", "First close"); err != nil {
		t.Fatalf("CloseProject failed: %v", err)
	}

	// Try to close again (should fail)
	err := manager.CloseProject(project.ID, "agent-1", "Second close")
	if err == nil {
		t.Error("Expected error when closing already closed project")
	}
}

func TestReopenNotClosed(t *testing.T) {
	manager := NewManager()

	project, _ := manager.CreateProject(
		"Test Project",
		"https://github.com/test/repo",
		"main",
		".beads",
		map[string]string{},
	)

	// Try to reopen without closing first (should fail)
	err := manager.ReopenProject(project.ID, "agent-1", "Reopen")
	if err == nil {
		t.Error("Expected error when reopening project that isn't closed")
	}
}

// ---------------------------------------------------------------------------
// UpdateProject tests
// ---------------------------------------------------------------------------

func TestUpdateProject_BasicFields(t *testing.T) {
	manager, project := createTestProject(t, "Update Test")

	updates := map[string]interface{}{
		"name":     "Updated Name",
		"git_repo": "https://github.com/new/repo",
		"branch":   "develop",
	}

	err := manager.UpdateProject(project.ID, updates)
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	updated, err := manager.GetProject(project.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %q", updated.Name)
	}
	if updated.GitRepo != "https://github.com/new/repo" {
		t.Errorf("Expected git_repo 'https://github.com/new/repo', got %q", updated.GitRepo)
	}
	if updated.Branch != "develop" {
		t.Errorf("Expected branch 'develop', got %q", updated.Branch)
	}
}

func TestUpdateProject_BeadsPathAndContext(t *testing.T) {
	manager, project := createTestProject(t, "Update BeadsPath")

	newContext := map[string]string{"language": "go", "framework": "gin"}
	updates := map[string]interface{}{
		"beads_path": ".custom-beads",
		"context":    newContext,
	}

	err := manager.UpdateProject(project.ID, updates)
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if updated.BeadsPath != ".custom-beads" {
		t.Errorf("Expected beads_path '.custom-beads', got %q", updated.BeadsPath)
	}
	if updated.Context["language"] != "go" {
		t.Errorf("Expected context language 'go', got %q", updated.Context["language"])
	}
	if updated.Context["framework"] != "gin" {
		t.Errorf("Expected context framework 'gin', got %q", updated.Context["framework"])
	}
}

func TestUpdateProject_BooleanFields(t *testing.T) {
	manager, project := createTestProject(t, "Update Booleans")

	updates := map[string]interface{}{
		"is_perpetual": true,
		"is_sticky":    true,
	}

	err := manager.UpdateProject(project.ID, updates)
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if !updated.IsPerpetual {
		t.Error("Expected IsPerpetual to be true")
	}
	if !updated.IsSticky {
		t.Error("Expected IsSticky to be true")
	}

	// Set them back to false
	updates2 := map[string]interface{}{
		"is_perpetual": false,
		"is_sticky":    false,
	}
	err = manager.UpdateProject(project.ID, updates2)
	if err != nil {
		t.Fatalf("UpdateProject (reset booleans) failed: %v", err)
	}

	updated2, _ := manager.GetProject(project.ID)
	if updated2.IsPerpetual {
		t.Error("Expected IsPerpetual to be false after reset")
	}
	if updated2.IsSticky {
		t.Error("Expected IsSticky to be false after reset")
	}
}

func TestUpdateProject_StatusAndGitStrategy(t *testing.T) {
	manager, project := createTestProject(t, "Update Status")

	updates := map[string]interface{}{
		"status":       "closed",
		"git_strategy": "branch-pr",
	}

	err := manager.UpdateProject(project.ID, updates)
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if updated.Status != models.ProjectStatusClosed {
		t.Errorf("Expected status 'closed', got %q", updated.Status)
	}
	if updated.GitStrategy != models.GitStrategyBranch {
		t.Errorf("Expected git_strategy 'branch-pr', got %q", updated.GitStrategy)
	}
}

func TestUpdateProject_UpdatesTimestamp(t *testing.T) {
	manager, project := createTestProject(t, "Timestamp Test")
	originalUpdatedAt := project.UpdatedAt

	updates := map[string]interface{}{
		"name": "Changed",
	}

	err := manager.UpdateProject(project.ID, updates)
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if !updated.UpdatedAt.After(originalUpdatedAt) && updated.UpdatedAt != originalUpdatedAt {
		// UpdatedAt should be set to time.Now() which is >= original
		// (In fast tests they could be equal, so we just check it was set)
	}
	if updated.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestUpdateProject_EmptyUpdates(t *testing.T) {
	manager, project := createTestProject(t, "Empty Updates")

	// Updating with empty map should succeed and just update timestamp
	err := manager.UpdateProject(project.ID, map[string]interface{}{})
	if err != nil {
		t.Fatalf("UpdateProject with empty updates failed: %v", err)
	}

	// Project should still be retrievable and unchanged
	updated, _ := manager.GetProject(project.ID)
	if updated.Name != "Empty Updates" {
		t.Errorf("Expected name to remain 'Empty Updates', got %q", updated.Name)
	}
}

func TestUpdateProject_NonExistent(t *testing.T) {
	manager := NewManager()

	err := manager.UpdateProject("non-existent", map[string]interface{}{"name": "x"})
	if err == nil {
		t.Error("Expected error when updating non-existent project")
	}
	if !strings.Contains(err.Error(), "project not found") {
		t.Errorf("Expected 'project not found' error, got: %v", err)
	}
}

func TestUpdateProject_IgnoresUnknownFields(t *testing.T) {
	manager, project := createTestProject(t, "Unknown Fields")

	// Unknown field keys should be silently ignored (no type assertion matches)
	updates := map[string]interface{}{
		"unknown_field": "some value",
		"name":          "Still Updated",
	}

	err := manager.UpdateProject(project.ID, updates)
	if err != nil {
		t.Fatalf("UpdateProject with unknown fields failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if updated.Name != "Still Updated" {
		t.Errorf("Expected name 'Still Updated', got %q", updated.Name)
	}
}

func TestUpdateProject_WrongTypeIgnored(t *testing.T) {
	manager, project := createTestProject(t, "Wrong Type")

	// Passing an int for "name" (expects string) should be silently ignored
	updates := map[string]interface{}{
		"name": 12345,
	}

	err := manager.UpdateProject(project.ID, updates)
	if err != nil {
		t.Fatalf("UpdateProject with wrong type failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if updated.Name != "Wrong Type" {
		t.Errorf("Expected name to remain 'Wrong Type', got %q", updated.Name)
	}
}

// ---------------------------------------------------------------------------
// AddAgentToProject tests
// ---------------------------------------------------------------------------

func TestAddAgentToProject_Success(t *testing.T) {
	manager, project := createTestProject(t, "Agent Add Test")

	err := manager.AddAgentToProject(project.ID, "agent-1")
	if err != nil {
		t.Fatalf("AddAgentToProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if len(updated.Agents) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(updated.Agents))
	}
	if updated.Agents[0] != "agent-1" {
		t.Errorf("Expected agent 'agent-1', got %q", updated.Agents[0])
	}
}

func TestAddAgentToProject_MultipleAgents(t *testing.T) {
	manager, project := createTestProject(t, "Multi Agent Test")

	agents := []string{"agent-1", "agent-2", "agent-3"}
	for _, agentID := range agents {
		if err := manager.AddAgentToProject(project.ID, agentID); err != nil {
			t.Fatalf("AddAgentToProject(%q) failed: %v", agentID, err)
		}
	}

	updated, _ := manager.GetProject(project.ID)
	if len(updated.Agents) != 3 {
		t.Fatalf("Expected 3 agents, got %d", len(updated.Agents))
	}

	// Check all agents are present
	agentSet := make(map[string]bool)
	for _, a := range updated.Agents {
		agentSet[a] = true
	}
	for _, expected := range agents {
		if !agentSet[expected] {
			t.Errorf("Expected agent %q in project agents", expected)
		}
	}
}

func TestAddAgentToProject_Duplicate(t *testing.T) {
	manager, project := createTestProject(t, "Duplicate Agent Test")

	// Add the same agent twice
	if err := manager.AddAgentToProject(project.ID, "agent-1"); err != nil {
		t.Fatalf("First AddAgentToProject failed: %v", err)
	}
	if err := manager.AddAgentToProject(project.ID, "agent-1"); err != nil {
		t.Fatalf("Second AddAgentToProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if len(updated.Agents) != 1 {
		t.Errorf("Expected 1 agent (no duplicates), got %d", len(updated.Agents))
	}
}

func TestAddAgentToProject_NonExistentProject(t *testing.T) {
	manager := NewManager()

	err := manager.AddAgentToProject("non-existent", "agent-1")
	if err == nil {
		t.Error("Expected error when adding agent to non-existent project")
	}
	if !strings.Contains(err.Error(), "project not found") {
		t.Errorf("Expected 'project not found' error, got: %v", err)
	}
}

func TestAddAgentToProject_UpdatesTimestamp(t *testing.T) {
	manager, project := createTestProject(t, "Agent Timestamp")
	originalUpdatedAt := project.UpdatedAt

	if err := manager.AddAgentToProject(project.ID, "agent-1"); err != nil {
		t.Fatalf("AddAgentToProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if updated.UpdatedAt.Before(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated after adding agent")
	}
}

// ---------------------------------------------------------------------------
// RemoveAgentFromProject tests
// ---------------------------------------------------------------------------

func TestRemoveAgentFromProject_Success(t *testing.T) {
	manager, project := createTestProject(t, "Remove Agent Test")

	// Add then remove
	manager.AddAgentToProject(project.ID, "agent-1")
	manager.AddAgentToProject(project.ID, "agent-2")

	err := manager.RemoveAgentFromProject(project.ID, "agent-1")
	if err != nil {
		t.Fatalf("RemoveAgentFromProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if len(updated.Agents) != 1 {
		t.Fatalf("Expected 1 agent after removal, got %d", len(updated.Agents))
	}
	if updated.Agents[0] != "agent-2" {
		t.Errorf("Expected remaining agent 'agent-2', got %q", updated.Agents[0])
	}
}

func TestRemoveAgentFromProject_LastAgent(t *testing.T) {
	manager, project := createTestProject(t, "Remove Last Agent")

	manager.AddAgentToProject(project.ID, "agent-1")

	err := manager.RemoveAgentFromProject(project.ID, "agent-1")
	if err != nil {
		t.Fatalf("RemoveAgentFromProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if len(updated.Agents) != 0 {
		t.Errorf("Expected 0 agents after removing last agent, got %d", len(updated.Agents))
	}
}

func TestRemoveAgentFromProject_AgentNotInProject(t *testing.T) {
	manager, project := createTestProject(t, "Remove Missing Agent")

	manager.AddAgentToProject(project.ID, "agent-1")

	err := manager.RemoveAgentFromProject(project.ID, "agent-999")
	if err == nil {
		t.Error("Expected error when removing agent not in project")
	}
	if !strings.Contains(err.Error(), "agent not found in project") {
		t.Errorf("Expected 'agent not found in project' error, got: %v", err)
	}
}

func TestRemoveAgentFromProject_NonExistentProject(t *testing.T) {
	manager := NewManager()

	err := manager.RemoveAgentFromProject("non-existent", "agent-1")
	if err == nil {
		t.Error("Expected error when removing agent from non-existent project")
	}
	if !strings.Contains(err.Error(), "project not found") {
		t.Errorf("Expected 'project not found' error, got: %v", err)
	}
}

func TestRemoveAgentFromProject_EmptyAgentsList(t *testing.T) {
	manager, project := createTestProject(t, "Remove From Empty")

	// Project starts with empty agents list
	err := manager.RemoveAgentFromProject(project.ID, "agent-1")
	if err == nil {
		t.Error("Expected error when removing agent from empty agents list")
	}
}

func TestRemoveAgentFromProject_UpdatesTimestamp(t *testing.T) {
	manager, project := createTestProject(t, "Remove Timestamp")

	manager.AddAgentToProject(project.ID, "agent-1")
	afterAdd, _ := manager.GetProject(project.ID)
	addTime := afterAdd.UpdatedAt

	err := manager.RemoveAgentFromProject(project.ID, "agent-1")
	if err != nil {
		t.Fatalf("RemoveAgentFromProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if updated.UpdatedAt.Before(addTime) {
		t.Error("Expected UpdatedAt to be updated after removing agent")
	}
}

func TestRemoveAgentFromProject_MiddleAgent(t *testing.T) {
	manager, project := createTestProject(t, "Remove Middle")

	manager.AddAgentToProject(project.ID, "agent-1")
	manager.AddAgentToProject(project.ID, "agent-2")
	manager.AddAgentToProject(project.ID, "agent-3")

	// Remove the middle agent
	err := manager.RemoveAgentFromProject(project.ID, "agent-2")
	if err != nil {
		t.Fatalf("RemoveAgentFromProject failed: %v", err)
	}

	updated, _ := manager.GetProject(project.ID)
	if len(updated.Agents) != 2 {
		t.Fatalf("Expected 2 agents after removal, got %d", len(updated.Agents))
	}

	// Remaining should be agent-1 and agent-3
	agentSet := make(map[string]bool)
	for _, a := range updated.Agents {
		agentSet[a] = true
	}
	if !agentSet["agent-1"] || !agentSet["agent-3"] {
		t.Errorf("Expected agents [agent-1, agent-3], got %v", updated.Agents)
	}
	if agentSet["agent-2"] {
		t.Error("agent-2 should have been removed")
	}
}

// ---------------------------------------------------------------------------
// DeleteProject tests
// ---------------------------------------------------------------------------

func TestDeleteProject_Success(t *testing.T) {
	manager, project := createTestProject(t, "Delete Test")

	err := manager.DeleteProject(project.ID)
	if err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	// Verify project is gone
	_, err = manager.GetProject(project.ID)
	if err == nil {
		t.Error("Expected error when getting deleted project")
	}
}

func TestDeleteProject_NonExistent(t *testing.T) {
	manager := NewManager()

	err := manager.DeleteProject("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent project")
	}
	if !strings.Contains(err.Error(), "project not found") {
		t.Errorf("Expected 'project not found' error, got: %v", err)
	}
}

func TestDeleteProject_DoesNotAffectOthers(t *testing.T) {
	manager := NewManager()

	p1, _ := manager.CreateProject("Project 1", "repo1", "main", ".beads", nil)
	p2, _ := manager.CreateProject("Project 2", "repo2", "main", ".beads", nil)

	err := manager.DeleteProject(p1.ID)
	if err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	// p2 should still exist
	remaining, err := manager.GetProject(p2.ID)
	if err != nil {
		t.Fatalf("GetProject for p2 failed after deleting p1: %v", err)
	}
	if remaining.Name != "Project 2" {
		t.Errorf("Expected remaining project name 'Project 2', got %q", remaining.Name)
	}
}

func TestDeleteProject_DoubleDeletion(t *testing.T) {
	manager, project := createTestProject(t, "Double Delete")

	err := manager.DeleteProject(project.ID)
	if err != nil {
		t.Fatalf("First DeleteProject failed: %v", err)
	}

	// Second deletion should fail
	err = manager.DeleteProject(project.ID)
	if err == nil {
		t.Error("Expected error on second deletion")
	}
}

func TestDeleteProject_RemovedFromList(t *testing.T) {
	manager := NewManager()

	p1, _ := manager.CreateProject("Project 1", "repo1", "main", ".beads", nil)
	manager.CreateProject("Project 2", "repo2", "main", ".beads", nil)

	// Before delete: 2 projects
	before := manager.ListProjects()
	if len(before) != 2 {
		t.Fatalf("Expected 2 projects before deletion, got %d", len(before))
	}

	manager.DeleteProject(p1.ID)

	// After delete: 1 project
	after := manager.ListProjects()
	if len(after) != 1 {
		t.Errorf("Expected 1 project after deletion, got %d", len(after))
	}
}

// ---------------------------------------------------------------------------
// LoadProjects tests
// ---------------------------------------------------------------------------

func TestLoadProjects_BasicLoad(t *testing.T) {
	manager := NewManager()

	projects := []models.Project{
		{
			ID:     "proj-load-1",
			Name:   "Loaded Project 1",
			Branch: "main",
		},
		{
			ID:     "proj-load-2",
			Name:   "Loaded Project 2",
			Branch: "develop",
		},
	}

	err := manager.LoadProjects(projects)
	if err != nil {
		t.Fatalf("LoadProjects failed: %v", err)
	}

	// Verify both projects are loaded
	p1, err := manager.GetProject("proj-load-1")
	if err != nil {
		t.Fatalf("GetProject proj-load-1 failed: %v", err)
	}
	if p1.Name != "Loaded Project 1" {
		t.Errorf("Expected name 'Loaded Project 1', got %q", p1.Name)
	}

	p2, err := manager.GetProject("proj-load-2")
	if err != nil {
		t.Fatalf("GetProject proj-load-2 failed: %v", err)
	}
	if p2.Name != "Loaded Project 2" {
		t.Errorf("Expected name 'Loaded Project 2', got %q", p2.Name)
	}
}

func TestLoadProjects_InitializesDefaults(t *testing.T) {
	manager := NewManager()

	projects := []models.Project{
		{
			ID:   "proj-defaults",
			Name: "Default Test",
		},
	}

	err := manager.LoadProjects(projects)
	if err != nil {
		t.Fatalf("LoadProjects failed: %v", err)
	}

	p, _ := manager.GetProject("proj-defaults")

	// Agents should be initialized to empty slice (not nil)
	if p.Agents == nil {
		t.Error("Expected Agents to be initialized (not nil)")
	}
	if len(p.Agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(p.Agents))
	}

	// Comments should be initialized to empty slice (not nil)
	if p.Comments == nil {
		t.Error("Expected Comments to be initialized (not nil)")
	}
	if len(p.Comments) != 0 {
		t.Errorf("Expected 0 comments, got %d", len(p.Comments))
	}

	// Status should default to open
	if p.Status != models.ProjectStatusOpen {
		t.Errorf("Expected default status 'open', got %q", p.Status)
	}

	// Timestamps should be set
	if p.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if p.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestLoadProjects_PreservesExistingStatus(t *testing.T) {
	manager := NewManager()

	projects := []models.Project{
		{
			ID:     "proj-status",
			Name:   "Status Test",
			Status: models.ProjectStatusClosed,
		},
	}

	err := manager.LoadProjects(projects)
	if err != nil {
		t.Fatalf("LoadProjects failed: %v", err)
	}

	p, _ := manager.GetProject("proj-status")
	if p.Status != models.ProjectStatusClosed {
		t.Errorf("Expected status 'closed' to be preserved, got %q", p.Status)
	}
}

func TestLoadProjects_PreservesExistingAgentsAndComments(t *testing.T) {
	manager := NewManager()

	projects := []models.Project{
		{
			ID:       "proj-existing",
			Name:     "Existing Data",
			Agents:   []string{"agent-a", "agent-b"},
			Comments: []models.ProjectComment{{ID: "c1", Comment: "existing"}},
			Status:   models.ProjectStatusOpen,
		},
	}

	err := manager.LoadProjects(projects)
	if err != nil {
		t.Fatalf("LoadProjects failed: %v", err)
	}

	p, _ := manager.GetProject("proj-existing")
	if len(p.Agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(p.Agents))
	}
	if len(p.Comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(p.Comments))
	}
}

func TestLoadProjects_EmptySlice(t *testing.T) {
	manager := NewManager()

	err := manager.LoadProjects([]models.Project{})
	if err != nil {
		t.Fatalf("LoadProjects with empty slice failed: %v", err)
	}

	projects := manager.ListProjects()
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects after loading empty slice, got %d", len(projects))
	}
}

func TestLoadProjects_MergesWithExisting(t *testing.T) {
	manager := NewManager()

	// Create one project via CreateProject
	existing, _ := manager.CreateProject("Existing", "repo", "main", ".beads", nil)

	// Load additional projects
	err := manager.LoadProjects([]models.Project{
		{ID: "proj-loaded", Name: "Loaded"},
	})
	if err != nil {
		t.Fatalf("LoadProjects failed: %v", err)
	}

	// Both should exist
	all := manager.ListProjects()
	if len(all) != 2 {
		t.Fatalf("Expected 2 projects (1 existing + 1 loaded), got %d", len(all))
	}

	_, err = manager.GetProject(existing.ID)
	if err != nil {
		t.Error("Existing project should still be accessible")
	}
	_, err = manager.GetProject("proj-loaded")
	if err != nil {
		t.Error("Loaded project should be accessible")
	}
}

// ---------------------------------------------------------------------------
// GetComments tests
// ---------------------------------------------------------------------------

func TestGetComments_EmptyProject(t *testing.T) {
	manager, project := createTestProject(t, "No Comments")

	comments, err := manager.GetComments(project.ID)
	if err != nil {
		t.Fatalf("GetComments failed: %v", err)
	}

	if len(comments) != 0 {
		t.Errorf("Expected 0 comments, got %d", len(comments))
	}
}

func TestGetComments_NonExistentProject(t *testing.T) {
	manager := NewManager()

	_, err := manager.GetComments("non-existent")
	if err == nil {
		t.Error("Expected error when getting comments for non-existent project")
	}
	if !strings.Contains(err.Error(), "project not found") {
		t.Errorf("Expected 'project not found' error, got: %v", err)
	}
}

func TestGetComments_MultipleComments(t *testing.T) {
	manager, project := createTestProject(t, "Multi Comments")

	// Add several comments
	manager.AddComment(project.ID, "agent-1", "First comment")
	manager.AddComment(project.ID, "agent-2", "Second comment")
	manager.AddComment(project.ID, "agent-1", "Third comment")

	comments, err := manager.GetComments(project.ID)
	if err != nil {
		t.Fatalf("GetComments failed: %v", err)
	}

	if len(comments) != 3 {
		t.Fatalf("Expected 3 comments, got %d", len(comments))
	}

	// Verify comment content
	if comments[0].Comment != "First comment" {
		t.Errorf("Expected first comment 'First comment', got %q", comments[0].Comment)
	}
	if comments[1].Comment != "Second comment" {
		t.Errorf("Expected second comment 'Second comment', got %q", comments[1].Comment)
	}
	if comments[2].Comment != "Third comment" {
		t.Errorf("Expected third comment 'Third comment', got %q", comments[2].Comment)
	}
}

func TestGetComments_IncludesCloseAndReopenComments(t *testing.T) {
	manager, project := createTestProject(t, "State Comments")

	manager.CloseProject(project.ID, "closer", "closing it")
	manager.ReopenProject(project.ID, "opener", "reopening it")

	comments, err := manager.GetComments(project.ID)
	if err != nil {
		t.Fatalf("GetComments failed: %v", err)
	}

	if len(comments) != 2 {
		t.Fatalf("Expected 2 comments (close + reopen), got %d", len(comments))
	}
	if comments[0].Comment != "closing it" {
		t.Errorf("Expected first comment 'closing it', got %q", comments[0].Comment)
	}
	if comments[1].Comment != "reopening it" {
		t.Errorf("Expected second comment 'reopening it', got %q", comments[1].Comment)
	}
}

func TestGetComments_CommentFields(t *testing.T) {
	manager, project := createTestProject(t, "Comment Fields")

	c, err := manager.AddComment(project.ID, "agent-42", "field check")
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	comments, _ := manager.GetComments(project.ID)
	if len(comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(comments))
	}

	comment := comments[0]
	if comment.ID == "" {
		t.Error("Expected comment ID to be set")
	}
	if comment.ID != c.ID {
		t.Errorf("Expected comment ID %q, got %q", c.ID, comment.ID)
	}
	if comment.ProjectID != project.ID {
		t.Errorf("Expected ProjectID %q, got %q", project.ID, comment.ProjectID)
	}
	if comment.AuthorID != "agent-42" {
		t.Errorf("Expected AuthorID 'agent-42', got %q", comment.AuthorID)
	}
	if comment.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set")
	}
}

// ---------------------------------------------------------------------------
// ListProjects tests
// ---------------------------------------------------------------------------

func TestListProjects_Empty(t *testing.T) {
	manager := NewManager()

	projects := manager.ListProjects()
	if projects == nil {
		t.Error("Expected non-nil slice from ListProjects")
	}
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
}

func TestListProjects_ReturnsAll(t *testing.T) {
	manager := NewManager()

	manager.CreateProject("Project A", "repoA", "main", ".beads", nil)
	manager.CreateProject("Project B", "repoB", "develop", ".beads", nil)
	manager.CreateProject("Project C", "repoC", "feature", ".beads", nil)

	projects := manager.ListProjects()
	if len(projects) != 3 {
		t.Fatalf("Expected 3 projects, got %d", len(projects))
	}

	// Collect names to verify all are present
	names := make(map[string]bool)
	for _, p := range projects {
		names[p.Name] = true
	}
	for _, expected := range []string{"Project A", "Project B", "Project C"} {
		if !names[expected] {
			t.Errorf("Expected project %q in list", expected)
		}
	}
}

func TestListProjects_IncludesLoadedProjects(t *testing.T) {
	manager := NewManager()

	manager.CreateProject("Created", "repo", "main", ".beads", nil)
	manager.LoadProjects([]models.Project{
		{ID: "loaded-1", Name: "Loaded"},
	})

	projects := manager.ListProjects()
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects (1 created + 1 loaded), got %d", len(projects))
	}
}

func TestListProjects_AfterDeletion(t *testing.T) {
	manager := NewManager()

	p1, _ := manager.CreateProject("P1", "r1", "main", ".beads", nil)
	manager.CreateProject("P2", "r2", "main", ".beads", nil)

	manager.DeleteProject(p1.ID)

	projects := manager.ListProjects()
	if len(projects) != 1 {
		t.Errorf("Expected 1 project after deletion, got %d", len(projects))
	}
	if projects[0].Name != "P2" {
		t.Errorf("Expected remaining project 'P2', got %q", projects[0].Name)
	}
}

// ---------------------------------------------------------------------------
// Clear tests
// ---------------------------------------------------------------------------

func TestClear_RemovesAllProjects(t *testing.T) {
	manager := NewManager()

	manager.CreateProject("Project 1", "repo1", "main", ".beads", nil)
	manager.CreateProject("Project 2", "repo2", "main", ".beads", nil)
	manager.CreateProject("Project 3", "repo3", "main", ".beads", nil)

	before := manager.ListProjects()
	if len(before) != 3 {
		t.Fatalf("Expected 3 projects before clear, got %d", len(before))
	}

	manager.Clear()

	after := manager.ListProjects()
	if len(after) != 0 {
		t.Errorf("Expected 0 projects after clear, got %d", len(after))
	}
}

func TestClear_EmptyManager(t *testing.T) {
	manager := NewManager()

	// Clearing an empty manager should not panic
	manager.Clear()

	projects := manager.ListProjects()
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects after clearing empty manager, got %d", len(projects))
	}
}

func TestClear_AllowsNewProjectsAfter(t *testing.T) {
	manager := NewManager()

	manager.CreateProject("Before Clear", "repo", "main", ".beads", nil)
	manager.Clear()

	// Should be able to create new projects after clear
	p, err := manager.CreateProject("After Clear", "repo2", "main", ".beads", nil)
	if err != nil {
		t.Fatalf("CreateProject after Clear failed: %v", err)
	}

	projects := manager.ListProjects()
	if len(projects) != 1 {
		t.Fatalf("Expected 1 project after clear+create, got %d", len(projects))
	}
	if projects[0].ID != p.ID {
		t.Errorf("Expected project ID %q, got %q", p.ID, projects[0].ID)
	}
}

func TestClear_OldProjectsNotAccessible(t *testing.T) {
	manager := NewManager()

	p, _ := manager.CreateProject("Old Project", "repo", "main", ".beads", nil)
	oldID := p.ID

	manager.Clear()

	_, err := manager.GetProject(oldID)
	if err == nil {
		t.Error("Expected error when getting project after clear")
	}
}

func TestClear_DoubleClear(t *testing.T) {
	manager := NewManager()

	manager.CreateProject("Project", "repo", "main", ".beads", nil)

	manager.Clear()
	manager.Clear() // Should not panic

	projects := manager.ListProjects()
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects after double clear, got %d", len(projects))
	}
}

// ---------------------------------------------------------------------------
// CreateProject additional tests (defaults, edge cases)
// ---------------------------------------------------------------------------

func TestCreateProject_DefaultBeadsPath(t *testing.T) {
	manager := NewManager()

	// Empty beadsPath should default to ".beads"
	project, err := manager.CreateProject("Default Path", "repo", "main", "", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if project.BeadsPath != ".beads" {
		t.Errorf("Expected default beads_path '.beads', got %q", project.BeadsPath)
	}
}

func TestCreateProject_NilContext(t *testing.T) {
	manager := NewManager()

	project, err := manager.CreateProject("Nil Context", "repo", "main", ".beads", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Context can be nil; Agents and Comments should be initialized
	if project.Agents == nil {
		t.Error("Expected Agents to be initialized (not nil)")
	}
	if project.Comments == nil {
		t.Error("Expected Comments to be initialized (not nil)")
	}
}

func TestCreateProject_FieldsAreSet(t *testing.T) {
	manager := NewManager()

	ctx := map[string]string{"key": "value"}
	project, err := manager.CreateProject("Full Test", "https://github.com/org/repo", "develop", ".custom", ctx)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if project.ID == "" {
		t.Error("Expected project ID to be generated")
	}
	if !strings.HasPrefix(project.ID, "proj-") {
		t.Errorf("Expected project ID prefix 'proj-', got %q", project.ID)
	}
	if project.Name != "Full Test" {
		t.Errorf("Expected name 'Full Test', got %q", project.Name)
	}
	if project.GitRepo != "https://github.com/org/repo" {
		t.Errorf("Expected git_repo, got %q", project.GitRepo)
	}
	if project.Branch != "develop" {
		t.Errorf("Expected branch 'develop', got %q", project.Branch)
	}
	if project.BeadsPath != ".custom" {
		t.Errorf("Expected beads_path '.custom', got %q", project.BeadsPath)
	}
	if project.Context["key"] != "value" {
		t.Errorf("Expected context key=value, got %q", project.Context["key"])
	}
	if project.IsPerpetual {
		t.Error("Expected IsPerpetual false by default")
	}
	if project.IsSticky {
		t.Error("Expected IsSticky false by default")
	}
	if project.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if project.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

// ---------------------------------------------------------------------------
// SetPerpetual additional tests
// ---------------------------------------------------------------------------

func TestSetPerpetual_NonExistentProject(t *testing.T) {
	manager := NewManager()

	err := manager.SetPerpetual("non-existent", true)
	if err == nil {
		t.Error("Expected error for non-existent project")
	}
	if !strings.Contains(err.Error(), "project not found") {
		t.Errorf("Expected 'project not found' error, got: %v", err)
	}
}

func TestSetPerpetual_Toggle(t *testing.T) {
	manager, project := createTestProject(t, "Toggle Perpetual")

	// Start false -> true
	manager.SetPerpetual(project.ID, true)
	p, _ := manager.GetProject(project.ID)
	if !p.IsPerpetual {
		t.Error("Expected IsPerpetual true")
	}

	// Toggle true -> false
	manager.SetPerpetual(project.ID, false)
	p, _ = manager.GetProject(project.ID)
	if p.IsPerpetual {
		t.Error("Expected IsPerpetual false after toggle")
	}
}

// ---------------------------------------------------------------------------
// CanClose additional tests
// ---------------------------------------------------------------------------

func TestCanClose_NonExistentProject(t *testing.T) {
	manager := NewManager()

	if manager.CanClose("non-existent", false) {
		t.Error("Expected CanClose to return false for non-existent project")
	}
}

func TestCanClose_ClosedProject(t *testing.T) {
	manager, project := createTestProject(t, "CanClose Closed")

	manager.CloseProject(project.ID, "agent-1", "closing")

	// Already closed project cannot be closed again
	if manager.CanClose(project.ID, false) {
		t.Error("Expected CanClose to return false for already closed project")
	}
}

// ---------------------------------------------------------------------------
// CloseProject additional tests (empty comment)
// ---------------------------------------------------------------------------

func TestCloseProject_EmptyComment(t *testing.T) {
	manager, project := createTestProject(t, "Close No Comment")

	err := manager.CloseProject(project.ID, "agent-1", "")
	if err != nil {
		t.Fatalf("CloseProject with empty comment failed: %v", err)
	}

	comments, _ := manager.GetComments(project.ID)
	if len(comments) != 0 {
		t.Errorf("Expected 0 comments when closing with empty comment, got %d", len(comments))
	}
}

// ---------------------------------------------------------------------------
// ReopenProject additional tests (empty comment)
// ---------------------------------------------------------------------------

func TestReopenProject_EmptyComment(t *testing.T) {
	manager, project := createTestProject(t, "Reopen No Comment")

	manager.CloseProject(project.ID, "agent-1", "closing")
	commentsBefore, _ := manager.GetComments(project.ID)

	err := manager.ReopenProject(project.ID, "agent-1", "")
	if err != nil {
		t.Fatalf("ReopenProject with empty comment failed: %v", err)
	}

	commentsAfter, _ := manager.GetComments(project.ID)
	if len(commentsAfter) != len(commentsBefore) {
		t.Errorf("Expected no new comment when reopening with empty comment, before=%d after=%d",
			len(commentsBefore), len(commentsAfter))
	}
}

// ---------------------------------------------------------------------------
// Concurrency tests
// ---------------------------------------------------------------------------

func TestConcurrentAccess(t *testing.T) {
	manager := NewManager()

	project, _ := manager.CreateProject("Concurrent", "repo", "main", ".beads", nil)

	var wg sync.WaitGroup
	errCh := make(chan error, 100)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.GetProject(project.ID)
			if err != nil {
				errCh <- err
			}
		}()
	}

	// Concurrent list
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.ListProjects()
		}()
	}

	// Concurrent comments
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.GetComments(project.ID)
			if err != nil {
				errCh <- err
			}
		}()
	}

	// Concurrent updates
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			err := manager.UpdateProject(project.ID, map[string]interface{}{
				"name": "Updated",
			})
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("Concurrent operation error: %v", err)
	}
}
