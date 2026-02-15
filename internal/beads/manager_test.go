package beads

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/config"
	"github.com/jordanhubbard/loom/pkg/models"
)

// TestNewManager tests manager creation
func TestNewManager(t *testing.T) {
	manager := NewManager("")

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.beadsPath != ".beads" {
		t.Errorf("beadsPath = %q, want %q", manager.beadsPath, ".beads")
	}

	if manager.beads == nil {
		t.Error("Expected beads map to be initialized")
	}

	if manager.workGraph == nil {
		t.Error("Expected workGraph to be initialized")
	}

	if manager.nextID != 1 {
		t.Errorf("nextID = %d, want %d", manager.nextID, 1)
	}
}

// TestManager_SetBeadsPath tests setting the beads path
func TestManager_SetBeadsPath(t *testing.T) {
	manager := NewManager("")

	newPath := "/custom/beads/path"
	manager.SetBeadsPath(newPath)

	if manager.beadsPath != newPath {
		t.Errorf("beadsPath = %q, want %q", manager.beadsPath, newPath)
	}
}

// TestManager_Reset tests resetting the manager state
func TestManager_Reset(t *testing.T) {
	manager := NewManager("")

	// Add some data
	bead := &models.Bead{
		ID:    "bd-001",
		Title: "Test Bead",
	}
	manager.beads[bead.ID] = bead
	manager.workGraph.Beads[bead.ID] = bead
	manager.nextID = 10

	// Reset
	manager.Reset()

	if len(manager.beads) != 0 {
		t.Error("Expected beads map to be empty after reset")
	}

	if len(manager.workGraph.Beads) != 0 {
		t.Error("Expected workGraph beads to be empty after reset")
	}

	if manager.nextID != 1 {
		t.Errorf("nextID = %d, want %d after reset", manager.nextID, 1)
	}
}

// TestManager_SetProjectPrefix tests setting project prefix
func TestManager_SetProjectPrefix(t *testing.T) {
	manager := NewManager("")

	projectID := "test-project"
	prefix := "tp"

	manager.SetProjectPrefix(projectID, prefix)

	gotPrefix := manager.GetProjectPrefix(projectID)
	if gotPrefix != prefix {
		t.Errorf("GetProjectPrefix() = %q, want %q", gotPrefix, prefix)
	}

	// Check that project next ID was initialized
	if _, exists := manager.projectNextIDs[projectID]; !exists {
		t.Error("Expected projectNextIDs to be initialized")
	}
}

// TestManager_GetProjectPrefix tests getting project prefix with defaults
func TestManager_GetProjectPrefix(t *testing.T) {
	manager := NewManager("")

	// Default prefix for unknown project
	prefix := manager.GetProjectPrefix("unknown-project")
	if prefix != "bd" {
		t.Errorf("GetProjectPrefix(unknown) = %q, want %q", prefix, "bd")
	}

	// Set custom prefix
	manager.SetProjectPrefix("project1", "p1")
	prefix = manager.GetProjectPrefix("project1")
	if prefix != "p1" {
		t.Errorf("GetProjectPrefix(project1) = %q, want %q", prefix, "p1")
	}
}

// TestManager_CreateBead tests bead creation without bd CLI
func TestManager_CreateBead(t *testing.T) {
	manager := NewManager("") // No bd CLI path

	title := "Test Bead"
	description := "Test Description"
	priority := models.BeadPriorityP2
	beadType := "task"
	projectID := "test-project"

	bead, err := manager.CreateBead(title, description, priority, beadType, projectID)
	if err != nil {
		t.Fatalf("CreateBead() error = %v", err)
	}

	if bead == nil {
		t.Fatal("Expected non-nil bead")
	}

	if bead.Title != title {
		t.Errorf("bead.Title = %q, want %q", bead.Title, title)
	}

	if bead.Description != description {
		t.Errorf("bead.Description = %q, want %q", bead.Description, description)
	}

	if bead.Priority != priority {
		t.Errorf("bead.Priority = %v, want %v", bead.Priority, priority)
	}

	if bead.Type != beadType {
		t.Errorf("bead.Type = %q, want %q", bead.Type, beadType)
	}

	if bead.ProjectID != projectID {
		t.Errorf("bead.ProjectID = %q, want %q", bead.ProjectID, projectID)
	}

	if bead.Status != models.BeadStatusOpen {
		t.Errorf("bead.Status = %q, want %q", bead.Status, models.BeadStatusOpen)
	}

	// Check that bead is in cache
	cachedBead, err := manager.GetBead(bead.ID)
	if err != nil {
		t.Fatalf("GetBead() error = %v", err)
	}

	if cachedBead.ID != bead.ID {
		t.Errorf("Cached bead ID = %q, want %q", cachedBead.ID, bead.ID)
	}
}

// TestManager_CreateBead_WithPrefix tests bead creation with custom prefix
func TestManager_CreateBead_WithPrefix(t *testing.T) {
	manager := NewManager("")
	projectID := "loom-self"

	manager.SetProjectPrefix(projectID, "ac")

	bead, err := manager.CreateBead("Test", "Desc", models.BeadPriorityP3, "task", projectID)
	if err != nil {
		t.Fatalf("CreateBead() error = %v", err)
	}

	// ID should start with "ac-"
	if len(bead.ID) < 3 || bead.ID[:3] != "ac-" {
		t.Errorf("bead.ID = %q, expected to start with 'ac-'", bead.ID)
	}
}

// TestManager_GetBead tests getting a bead by ID
func TestManager_GetBead(t *testing.T) {
	manager := NewManager("")

	// Create a bead
	bead, err := manager.CreateBead("Test", "Desc", models.BeadPriorityP2, "task", "project1")
	if err != nil {
		t.Fatalf("CreateBead() error = %v", err)
	}

	// Get the bead
	retrieved, err := manager.GetBead(bead.ID)
	if err != nil {
		t.Fatalf("GetBead() error = %v", err)
	}

	if retrieved.ID != bead.ID {
		t.Errorf("GetBead().ID = %q, want %q", retrieved.ID, bead.ID)
	}
}

// TestManager_ListBeads tests listing beads
func TestManager_ListBeads(t *testing.T) {
	manager := NewManager("")

	// Create several beads
	bead1, _ := manager.CreateBead("Bead 1", "Desc 1", models.BeadPriorityP1, "task", "project1")
	bead2, _ := manager.CreateBead("Bead 2", "Desc 2", models.BeadPriorityP3, "bug", "project1")
	bead3, _ := manager.CreateBead("Bead 3", "Desc 3", models.BeadPriorityP2, "task", "project2")

	// List all beads
	allBeads, err := manager.ListBeads(nil)
	if err != nil {
		t.Fatalf("ListBeads() error = %v", err)
	}

	if len(allBeads) != 3 {
		t.Errorf("ListBeads() returned %d beads, want %d", len(allBeads), 3)
	}

	// List beads for project1
	filters := map[string]interface{}{
		"project_id": "project1",
	}
	project1Beads, err := manager.ListBeads(filters)
	if err != nil {
		t.Fatalf("ListBeads(project1) error = %v", err)
	}

	if len(project1Beads) != 2 {
		t.Errorf("ListBeads(project1) returned %d beads, want %d", len(project1Beads), 2)
	}

	// List beads by type
	filters = map[string]interface{}{
		"type": "task",
	}
	taskBeads, err := manager.ListBeads(filters)
	if err != nil {
		t.Fatalf("ListBeads(type=task) error = %v", err)
	}

	if len(taskBeads) != 2 {
		t.Errorf("ListBeads(type=task) returned %d beads, want %d", len(taskBeads), 2)
	}

	// Verify bead IDs
	foundBead1 := false
	foundBead2 := false
	for _, b := range project1Beads {
		if b.ID == bead1.ID {
			foundBead1 = true
		}
		if b.ID == bead2.ID {
			foundBead2 = true
		}
	}

	if !foundBead1 || !foundBead2 {
		t.Error("Expected to find bead1 and bead2 in project1 beads")
	}

	_ = bead3 // Silence unused warning
}

// TestManager_UpdateBead tests updating a bead
func TestManager_UpdateBead(t *testing.T) {
	manager := NewManager("")

	bead, err := manager.CreateBead("Original", "Desc", models.BeadPriorityP3, "task", "project1")
	if err != nil {
		t.Fatalf("CreateBead() error = %v", err)
	}

	// Update bead
	updates := map[string]interface{}{
		"title":       "Updated Title",
		"status":      models.BeadStatusInProgress,
		"priority":    models.BeadPriorityP1,
		"assigned_to": "agent-1",
	}

	err = manager.UpdateBead(bead.ID, updates)
	if err != nil {
		t.Fatalf("UpdateBead() error = %v", err)
	}

	// Verify updates
	updated, _ := manager.GetBead(bead.ID)
	if updated.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", updated.Title, "Updated Title")
	}

	if updated.Status != models.BeadStatusInProgress {
		t.Errorf("Status = %q, want %q", updated.Status, models.BeadStatusInProgress)
	}

	if updated.Priority != models.BeadPriorityP1 {
		t.Errorf("Priority = %v, want %v", updated.Priority, models.BeadPriorityP1)
	}

	if updated.AssignedTo != "agent-1" {
		t.Errorf("AssignedTo = %q, want %q", updated.AssignedTo, "agent-1")
	}
}

// TestManager_UpdateBead_StatusClosed tests closing a bead
func TestManager_UpdateBead_StatusClosed(t *testing.T) {
	manager := NewManager("")

	bead, _ := manager.CreateBead("Test", "Desc", models.BeadPriorityP2, "task", "project1")

	// Close the bead
	updates := map[string]interface{}{
		"status": models.BeadStatusClosed,
	}

	err := manager.UpdateBead(bead.ID, updates)
	if err != nil {
		t.Fatalf("UpdateBead() error = %v", err)
	}

	updated, _ := manager.GetBead(bead.ID)
	if updated.Status != models.BeadStatusClosed {
		t.Errorf("Status = %q, want %q", updated.Status, models.BeadStatusClosed)
	}

	if updated.ClosedAt == nil {
		t.Error("Expected ClosedAt to be set")
	}

	// Reopen the bead
	updates = map[string]interface{}{
		"status": models.BeadStatusOpen,
	}

	err = manager.UpdateBead(bead.ID, updates)
	if err != nil {
		t.Fatalf("UpdateBead() error = %v", err)
	}

	reopened, _ := manager.GetBead(bead.ID)
	if reopened.ClosedAt != nil {
		t.Error("Expected ClosedAt to be nil after reopening")
	}
}

// TestManager_ClaimBead tests claiming a bead
func TestManager_ClaimBead(t *testing.T) {
	manager := NewManager("")

	bead, _ := manager.CreateBead("Test", "Desc", models.BeadPriorityP2, "task", "project1")

	agentID := "agent-1"
	err := manager.ClaimBead(bead.ID, agentID)
	if err != nil {
		t.Fatalf("ClaimBead() error = %v", err)
	}

	claimed, _ := manager.GetBead(bead.ID)
	if claimed.AssignedTo != agentID {
		t.Errorf("AssignedTo = %q, want %q", claimed.AssignedTo, agentID)
	}

	if claimed.Status != models.BeadStatusInProgress {
		t.Errorf("Status = %q, want %q", claimed.Status, models.BeadStatusInProgress)
	}
}

// TestManager_ClaimBead_AlreadyClaimed tests claiming an already claimed bead
func TestManager_ClaimBead_AlreadyClaimed(t *testing.T) {
	manager := NewManager("")

	bead, _ := manager.CreateBead("Test", "Desc", models.BeadPriorityP2, "task", "project1")

	// Claim by agent-1
	manager.ClaimBead(bead.ID, "agent-1")

	// Try to claim by agent-2
	err := manager.ClaimBead(bead.ID, "agent-2")
	if err == nil {
		t.Error("Expected error when claiming already claimed bead")
	}

	// Same agent can reclaim
	err = manager.ClaimBead(bead.ID, "agent-1")
	if err != nil {
		t.Errorf("Same agent should be able to reclaim: %v", err)
	}
}

// TestManager_ClaimBead_NotFound tests claiming a non-existent bead
func TestManager_ClaimBead_NotFound(t *testing.T) {
	manager := NewManager("")

	err := manager.ClaimBead("nonexistent", "agent-1")
	if err == nil {
		t.Error("Expected error when claiming non-existent bead")
	}
}

// TestManager_AddDependency tests adding dependencies between beads
func TestManager_AddDependency(t *testing.T) {
	manager := NewManager("")

	bead1, _ := manager.CreateBead("Bead 1", "Desc", models.BeadPriorityP2, "task", "project1")
	bead2, _ := manager.CreateBead("Bead 2", "Desc", models.BeadPriorityP2, "task", "project1")

	// Add blocking relationship
	err := manager.AddDependency(bead2.ID, bead1.ID, "blocks")
	if err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	// Verify relationship
	child, _ := manager.GetBead(bead2.ID)
	if len(child.BlockedBy) != 1 || child.BlockedBy[0] != bead1.ID {
		t.Errorf("child.BlockedBy = %v, want [%s]", child.BlockedBy, bead1.ID)
	}

	parent, _ := manager.GetBead(bead1.ID)
	if len(parent.Blocks) != 1 || parent.Blocks[0] != bead2.ID {
		t.Errorf("parent.Blocks = %v, want [%s]", parent.Blocks, bead2.ID)
	}

	// Verify edge in work graph
	if len(manager.workGraph.Edges) != 1 {
		t.Errorf("workGraph.Edges length = %d, want 1", len(manager.workGraph.Edges))
	}
}

// TestManager_AddDependency_Parent tests parent-child relationship
func TestManager_AddDependency_Parent(t *testing.T) {
	manager := NewManager("")

	parent, _ := manager.CreateBead("Parent", "Desc", models.BeadPriorityP2, "task", "project1")
	child, _ := manager.CreateBead("Child", "Desc", models.BeadPriorityP2, "task", "project1")

	err := manager.AddDependency(child.ID, parent.ID, "parent")
	if err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	childBead, _ := manager.GetBead(child.ID)
	if childBead.Parent != parent.ID {
		t.Errorf("child.Parent = %q, want %q", childBead.Parent, parent.ID)
	}

	parentBead, _ := manager.GetBead(parent.ID)
	if len(parentBead.Children) != 1 || parentBead.Children[0] != child.ID {
		t.Errorf("parent.Children = %v, want [%s]", parentBead.Children, child.ID)
	}
}

// TestManager_AddDependency_Related tests related relationship
func TestManager_AddDependency_Related(t *testing.T) {
	manager := NewManager("")

	bead1, _ := manager.CreateBead("Bead 1", "Desc", models.BeadPriorityP2, "task", "project1")
	bead2, _ := manager.CreateBead("Bead 2", "Desc", models.BeadPriorityP2, "task", "project1")

	err := manager.AddDependency(bead1.ID, bead2.ID, "related")
	if err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	b1, _ := manager.GetBead(bead1.ID)
	if len(b1.RelatedTo) != 1 || b1.RelatedTo[0] != bead2.ID {
		t.Errorf("bead1.RelatedTo = %v, want [%s]", b1.RelatedTo, bead2.ID)
	}

	b2, _ := manager.GetBead(bead2.ID)
	if len(b2.RelatedTo) != 1 || b2.RelatedTo[0] != bead1.ID {
		t.Errorf("bead2.RelatedTo = %v, want [%s]", b2.RelatedTo, bead1.ID)
	}
}

// TestManager_AddDependency_InvalidRelationship tests invalid relationship type
func TestManager_AddDependency_InvalidRelationship(t *testing.T) {
	manager := NewManager("")

	bead1, _ := manager.CreateBead("Bead 1", "Desc", models.BeadPriorityP2, "task", "project1")
	bead2, _ := manager.CreateBead("Bead 2", "Desc", models.BeadPriorityP2, "task", "project1")

	err := manager.AddDependency(bead1.ID, bead2.ID, "invalid")
	if err == nil {
		t.Error("Expected error for invalid relationship type")
	}
}

// TestManager_GetReadyBeads tests getting beads with no blockers
func TestManager_GetReadyBeads(t *testing.T) {
	manager := NewManager("")

	bead1, _ := manager.CreateBead("Bead 1", "Desc", models.BeadPriorityP2, "task", "project1")
	bead2, _ := manager.CreateBead("Bead 2", "Desc", models.BeadPriorityP2, "task", "project1")
	bead3, _ := manager.CreateBead("Bead 3", "Desc", models.BeadPriorityP2, "task", "project1")

	// Bead 2 blocks bead 3
	manager.AddDependency(bead3.ID, bead2.ID, "blocks")

	ready, err := manager.GetReadyBeads("project1")
	if err != nil {
		t.Fatalf("GetReadyBeads() error = %v", err)
	}

	// Should have bead1 and bead2, but not bead3 (blocked)
	readyIDs := make(map[string]bool)
	for _, b := range ready {
		readyIDs[b.ID] = true
	}

	if !readyIDs[bead1.ID] {
		t.Error("Expected bead1 to be ready")
	}

	if !readyIDs[bead2.ID] {
		t.Error("Expected bead2 to be ready")
	}

	if readyIDs[bead3.ID] {
		t.Error("Did not expect bead3 to be ready (blocked)")
	}
}

// TestManager_UnblockBead tests unblocking a bead
func TestManager_UnblockBead(t *testing.T) {
	manager := NewManager("")

	blocker, _ := manager.CreateBead("Blocker", "Desc", models.BeadPriorityP2, "task", "project1")
	blocked, _ := manager.CreateBead("Blocked", "Desc", models.BeadPriorityP2, "task", "project1")

	// Set bead to in-progress first, then block it
	updates := map[string]interface{}{
		"status": models.BeadStatusInProgress,
	}
	manager.UpdateBead(blocked.ID, updates)

	// Set up blocking - this should change status to blocked
	manager.AddDependency(blocked.ID, blocker.ID, "blocks")

	// Verify blocked
	b, _ := manager.GetBead(blocked.ID)
	if b.Status != models.BeadStatusBlocked {
		t.Errorf("Status = %q, want %q", b.Status, models.BeadStatusBlocked)
	}

	// Unblock
	err := manager.UnblockBead(blocked.ID, blocker.ID)
	if err != nil {
		t.Fatalf("UnblockBead() error = %v", err)
	}

	// Verify unblocked
	unblocked, _ := manager.GetBead(blocked.ID)
	if len(unblocked.BlockedBy) != 0 {
		t.Error("Expected BlockedBy to be empty after unblocking")
	}

	if unblocked.Status != models.BeadStatusOpen {
		t.Errorf("Status = %q, want %q after unblocking", unblocked.Status, models.BeadStatusOpen)
	}
}

// TestManager_GetWorkGraph tests getting the work graph
func TestManager_GetWorkGraph(t *testing.T) {
	manager := NewManager("")

	bead1, _ := manager.CreateBead("Bead 1", "Desc", models.BeadPriorityP2, "task", "project1")
	bead2, _ := manager.CreateBead("Bead 2", "Desc", models.BeadPriorityP2, "task", "project2")

	manager.AddDependency(bead2.ID, bead1.ID, "related")

	// Get full graph
	graph, err := manager.GetWorkGraph("")
	if err != nil {
		t.Fatalf("GetWorkGraph() error = %v", err)
	}

	if len(graph.Beads) != 2 {
		t.Errorf("Graph beads count = %d, want 2", len(graph.Beads))
	}

	if len(graph.Edges) != 1 {
		t.Errorf("Graph edges count = %d, want 1", len(graph.Edges))
	}

	// Get project-filtered graph
	graph1, err := manager.GetWorkGraph("project1")
	if err != nil {
		t.Fatalf("GetWorkGraph(project1) error = %v", err)
	}

	if len(graph1.Beads) != 1 {
		t.Errorf("Project1 graph beads count = %d, want 1", len(graph1.Beads))
	}

	if _, ok := graph1.Beads[bead1.ID]; !ok {
		t.Error("Expected bead1 in project1 graph")
	}
}

// Helper function tests

// TestSanitizeFilename tests filename sanitization
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Lowercase and hyphens",
			input: "Test Title",
			want:  "test-title",
		},
		{
			name:  "Remove special chars",
			input: "Test@Title#2023!",
			want:  "testtitle2023",
		},
		{
			name:  "Multiple spaces",
			input: "Test   Title",
			want:  "test---title",
		},
		{
			name:  "Very long string",
			input: "This is a very long string that should be truncated to fifty characters maximum",
			want:  "this-is-a-very-long-string-that-should-be-truncate",
		},
		{
			name:  "Already sanitized",
			input: "test-title",
			want:  "test-title",
		},
		{
			name:  "Numbers and hyphens",
			input: "bug-123-fix",
			want:  "bug-123-fix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}

			// Verify length constraint
			if len(got) > 50 {
				t.Errorf("sanitizeFilename(%q) length = %d, want <= 50", tt.input, len(got))
			}
		})
	}
}

// TestBeadsRootDir tests finding the beads root directory
func TestBeadsRootDir(t *testing.T) {
	tests := []struct {
		name      string
		beadsPath string
		want      string
	}{
		{
			name:      "Empty path",
			beadsPath: "",
			want:      "",
		},
		{
			name:      ".beads directory",
			beadsPath: "/path/to/.beads",
			want:      "/path/to",
		},
		{
			name:      "beads directory",
			beadsPath: "/path/to/beads",
			want:      "/path",
		},
		{
			name:      "Other directory",
			beadsPath: "/path/to/data",
			want:      "/path/to",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := beadsRootDir(tt.beadsPath)
			if got != tt.want {
				t.Errorf("beadsRootDir(%q) = %q, want %q", tt.beadsPath, got, tt.want)
			}
		})
	}
}

// TestExtractBeadID tests extracting bead IDs from output
func TestExtractBeadID(t *testing.T) {
	manager := NewManager("")

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "Simple output",
			output: "Created bd-12345",
			want:   "bd-12345",
		},
		{
			name:   "Output with punctuation",
			output: "Issue bd-12345, created successfully",
			want:   "bd-12345",
		},
		{
			name:   "Output with brackets",
			output: "[bd-12345] was created",
			want:   "bd-12345",
		},
		{
			name:   "Custom prefix",
			output: "Created ac-456",
			want:   "ac-456",
		},
		{
			name:   "No bead ID",
			output: "Error: failed to create",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.extractBeadID(tt.output)
			if got != tt.want {
				t.Errorf("extractBeadID(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

// TestExtractBeadIDWithPrefix tests extracting bead IDs with specific prefix
func TestExtractBeadIDWithPrefix(t *testing.T) {
	manager := NewManager("")

	tests := []struct {
		name   string
		output string
		prefix string
		want   string
	}{
		{
			name:   "Match with prefix",
			output: "Created ac-123",
			prefix: "ac",
			want:   "ac-123",
		},
		{
			name:   "No match different prefix",
			output: "Created bd-123",
			prefix: "ac",
			want:   "",
		},
		{
			name:   "Multiple IDs",
			output: "Created ac-123 and bd-456",
			prefix: "ac",
			want:   "ac-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.extractBeadIDWithPrefix(tt.output, tt.prefix)
			if got != tt.want {
				t.Errorf("extractBeadIDWithPrefix(%q, %q) = %q, want %q", tt.output, tt.prefix, got, tt.want)
			}
		})
	}
}

// TestMatchesFilters tests filter matching logic
func TestMatchesFilters(t *testing.T) {
	manager := NewManager("")

	bead := &models.Bead{
		ID:         "bd-001",
		ProjectID:  "project1",
		Status:     models.BeadStatusOpen,
		Type:       "task",
		AssignedTo: "agent-1",
	}

	tests := []struct {
		name    string
		filters map[string]interface{}
		want    bool
	}{
		{
			name:    "No filters",
			filters: map[string]interface{}{},
			want:    true,
		},
		{
			name: "Matching project_id",
			filters: map[string]interface{}{
				"project_id": "project1",
			},
			want: true,
		},
		{
			name: "Non-matching project_id",
			filters: map[string]interface{}{
				"project_id": "project2",
			},
			want: false,
		},
		{
			name: "Matching status",
			filters: map[string]interface{}{
				"status": models.BeadStatusOpen,
			},
			want: true,
		},
		{
			name: "Matching type",
			filters: map[string]interface{}{
				"type": "task",
			},
			want: true,
		},
		{
			name: "Matching assigned_to string",
			filters: map[string]interface{}{
				"assigned_to": "agent-1",
			},
			want: true,
		},
		{
			name: "Matching assigned_to slice",
			filters: map[string]interface{}{
				"assigned_to": []string{"agent-1", "agent-2"},
			},
			want: true,
		},
		{
			name: "Non-matching assigned_to slice",
			filters: map[string]interface{}{
				"assigned_to": []string{"agent-2", "agent-3"},
			},
			want: false,
		},
		{
			name: "Multiple matching filters",
			filters: map[string]interface{}{
				"project_id": "project1",
				"type":       "task",
				"status":     models.BeadStatusOpen,
			},
			want: true,
		},
		{
			name: "One non-matching filter",
			filters: map[string]interface{}{
				"project_id": "project1",
				"type":       "bug",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.matchesFilters(bead, tt.filters)
			if got != tt.want {
				t.Errorf("matchesFilters() = %v, want %v (filters: %v)", got, tt.want, tt.filters)
			}
		})
	}
}

// TestManager_SaveAndLoadBead tests filesystem save/load
func TestManager_SaveAndLoadBead(t *testing.T) {
	tmpDir := t.TempDir()
	beadsPath := filepath.Join(tmpDir, ".beads")

	manager := NewManager("")
	manager.SetBeadsPath(beadsPath)

	// Create a bead
	bead := &models.Bead{
		ID:          "bd-001",
		Type:        "task",
		Title:       "Test Bead",
		Description: "Test Description",
		Status:      models.BeadStatusOpen,
		Priority:    models.BeadPriorityP2,
		ProjectID:   "test-project",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to filesystem
	err := manager.SaveBeadToFilesystem(bead, beadsPath)
	if err != nil {
		t.Fatalf("SaveBeadToFilesystem() error = %v", err)
	}

	// Verify file was created
	beadsDir := filepath.Join(beadsPath, "beads")
	entries, err := os.ReadDir(beadsDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 file, got %d", len(entries))
	}

	// Load from filesystem
	manager2 := NewManager("")
	err = manager2.LoadBeadsFromFilesystem("test-project", beadsPath)
	if err != nil {
		t.Fatalf("LoadBeadsFromFilesystem() error = %v", err)
	}

	// Verify bead was loaded
	loaded, err := manager2.GetBead(bead.ID)
	if err != nil {
		t.Fatalf("GetBead() error = %v", err)
	}

	if loaded.Title != bead.Title {
		t.Errorf("Loaded bead title = %q, want %q", loaded.Title, bead.Title)
	}

	if loaded.Description != bead.Description {
		t.Errorf("Loaded bead description = %q, want %q", loaded.Description, bead.Description)
	}
}

// TestManager_LoadProjectPrefixFromConfig tests loading prefix from config
func TestManager_LoadProjectPrefixFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	beadsPath := filepath.Join(tmpDir, ".beads")

	// Create beads directory and config
	if err := os.MkdirAll(beadsPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	configPath := filepath.Join(beadsPath, "config.yaml")
	configContent := "issue-prefix: ac\n"
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manager := NewManager("")
	err := manager.LoadProjectPrefixFromConfig("test-project", beadsPath)
	if err != nil {
		t.Fatalf("LoadProjectPrefixFromConfig() error = %v", err)
	}

	prefix := manager.GetProjectPrefix("test-project")
	if prefix != "ac" {
		t.Errorf("GetProjectPrefix() = %q, want %q", prefix, "ac")
	}
}

// TestManager_SyncFederation tests federation sync
func TestManager_SyncFederation(t *testing.T) {
	manager := NewManager("")

	cfg := &config.BeadsFederationConfig{
		Enabled: false,
	}

	// Should return nil when disabled
	err := manager.SyncFederation(context.Background(), cfg)
	if err != nil {
		t.Errorf("SyncFederation() with disabled config error = %v, want nil", err)
	}

	// Should return nil when cfg is nil
	err = manager.SyncFederation(context.Background(), nil)
	if err != nil {
		t.Errorf("SyncFederation() with nil config error = %v, want nil", err)
	}
}
