package loom

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/config"
	"github.com/jordanhubbard/loom/pkg/models"
)

// testLoom creates a Loom instance suitable for testing, using a temp directory
// for gitops to avoid filesystem issues. Caller must defer cleanup of the returned dir.
func testLoom(t *testing.T, opts ...func(*config.Config)) (*Loom, string) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "loom-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			DefaultPersonaPath: "../../personas",
			MaxConcurrent:      10,
		},
		Database: config.DatabaseConfig{
			Type: "sqlite",
			Path: ":memory:",
		},
		Git: config.GitConfig{
			ProjectKeyDir: tmpDir,
		},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	l, err := New(cfg)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create loom: %v", err)
	}
	return l, tmpDir
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []func(*config.Config)
		wantErr bool
	}{
		{
			name: "minimal config with database",
		},
		{
			name: "config with empty persona path",
			opts: []func(*config.Config){
				func(c *config.Config) {
					c.Agents.DefaultPersonaPath = ""
					c.Agents.MaxConcurrent = 5
				},
			},
		},
		{
			name: "config without database",
			opts: []func(*config.Config){
				func(c *config.Config) {
					c.Database = config.DatabaseConfig{}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loom, tmpDir := testLoom(t, tt.opts...)
			defer os.RemoveAll(tmpDir)

			if loom == nil {
				t.Fatal("New() returned nil loom")
			}

			// Verify core components are initialized
			if loom.config == nil {
				t.Error("config not initialized")
			}
			if loom.providerRegistry == nil {
				t.Error("providerRegistry not initialized")
			}
			if loom.personaManager == nil {
				t.Error("personaManager not initialized")
			}
			if loom.agentManager == nil {
				t.Error("agentManager not initialized")
			}
		})
	}
}

func TestNew_WithTemporal(t *testing.T) {
	t.Skip("Requires Temporal server - skipping in unit tests")
}

func TestLoom_GetAgentManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetAgentManager() == nil {
		t.Error("GetAgentManager() returned nil")
	}
}

func TestLoom_GetDatabase(t *testing.T) {
	t.Run("with sqlite database", func(t *testing.T) {
		loom, tmpDir := testLoom(t)
		defer os.RemoveAll(tmpDir)
		if loom.GetDatabase() == nil {
			t.Error("GetDatabase() returned nil for sqlite config")
		}
	})

	t.Run("without database", func(t *testing.T) {
		loom, tmpDir := testLoom(t, func(c *config.Config) {
			c.Database = config.DatabaseConfig{}
		})
		defer os.RemoveAll(tmpDir)
		if loom.GetDatabase() != nil {
			t.Error("GetDatabase() returned non-nil without database config")
		}
	})
}

func TestLoom_GetProviderRegistry(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetProviderRegistry() == nil {
		t.Error("GetProviderRegistry() returned nil")
	}
}

func TestLoom_GetPersonaManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetPersonaManager() == nil {
		t.Error("GetPersonaManager() returned nil")
	}
}

func TestLoom_GetProjectManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetProjectManager() == nil {
		t.Error("GetProjectManager() returned nil")
	}
}

func TestLoom_GetBeadsManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetBeadsManager() == nil {
		t.Error("GetBeadsManager() returned nil")
	}
}

func TestLoom_GetEventBus(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetEventBus() == nil {
		t.Error("GetEventBus() returned nil")
	}
}

func TestLoom_GetDispatcher(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetDispatcher() == nil {
		t.Error("GetDispatcher() returned nil")
	}
}

func TestLoom_GetGitOpsManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetGitOpsManager() == nil {
		t.Error("GetGitOpsManager() returned nil")
	}
}

func TestLoom_GetModelCatalog(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetModelCatalog() == nil {
		t.Error("GetModelCatalog() returned nil")
	}
}

func TestLoom_GetMetrics(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetMetrics() == nil {
		t.Error("GetMetrics() returned nil")
	}
}

func TestLoom_GetOrgChartManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetOrgChartManager() == nil {
		t.Error("GetOrgChartManager() returned nil")
	}
}

func TestLoom_GetDecisionManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetDecisionManager() == nil {
		t.Error("GetDecisionManager() returned nil")
	}
}

func TestLoom_GetMotivationRegistry(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetMotivationRegistry() == nil {
		t.Error("GetMotivationRegistry() returned nil")
	}
}

func TestLoom_GetMotivationEngine(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// MotivationEngine may be nil if no motivations are registered
	_ = loom.GetMotivationEngine()
}

func TestLoom_GetWorkflowEngine(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetWorkflowEngine() == nil {
		t.Error("GetWorkflowEngine() returned nil")
	}
}

func TestLoom_GetActivityManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetActivityManager() == nil {
		t.Error("GetActivityManager() returned nil")
	}
}

func TestLoom_GetNotificationManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetNotificationManager() == nil {
		t.Error("GetNotificationManager() returned nil")
	}
}

func TestLoom_GetCommentsManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetCommentsManager() == nil {
		t.Error("GetCommentsManager() returned nil")
	}
}

func TestLoom_GetLogManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetLogManager() == nil {
		t.Error("GetLogManager() returned nil")
	}
}

func TestLoom_GetPatternManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetPatternManager() == nil {
		t.Error("GetPatternManager() returned nil")
	}
}

func TestLoom_GetKeyManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// KeyManager is nil by default (set via SetKeyManager)
	if loom.GetKeyManager() != nil {
		t.Error("GetKeyManager() returned non-nil before SetKeyManager called")
	}
}

func TestLoom_SetKeyManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Initially nil
	if loom.GetKeyManager() != nil {
		t.Error("expected nil before set")
	}

	// SetKeyManager with nil should not panic
	loom.SetKeyManager(nil)
	if loom.GetKeyManager() != nil {
		t.Error("expected nil after setting nil")
	}
}

func TestLoom_GetTemporalManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Without Temporal config, should be nil
	if loom.GetTemporalManager() != nil {
		t.Error("GetTemporalManager() returned non-nil without temporal config")
	}
}

func TestLoom_GetDoltCoordinator(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// DoltCoordinator may or may not be initialized depending on config
	_ = loom.GetDoltCoordinator()
}

func TestLoom_GetIdleDetector(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetIdleDetector() == nil {
		t.Error("GetIdleDetector() returned nil")
	}
}

func TestLoom_GetWorkerManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetWorkerManager() == nil {
		t.Error("GetWorkerManager() returned nil")
	}
}

func TestLoom_GetFileLockManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetFileLockManager() == nil {
		t.Error("GetFileLockManager() returned nil")
	}
}

func TestLoom_GetActionRouter(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	if loom.GetActionRouter() == nil {
		t.Error("GetActionRouter() returned nil")
	}
}

func TestLoom_CheckProjectReadiness(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Empty project ID should return ready
	ready, issues := loom.CheckProjectReadiness(ctx, "")
	if !ready {
		t.Errorf("CheckProjectReadiness('') = false, want true")
	}
	if len(issues) != 0 {
		t.Errorf("CheckProjectReadiness('') issues = %v, want empty", issues)
	}

	// Non-existent project should return not ready
	ready, issues = loom.CheckProjectReadiness(ctx, "non-existent")
	if ready {
		t.Error("CheckProjectReadiness('non-existent') = true, want false")
	}
	if len(issues) == 0 {
		t.Error("CheckProjectReadiness('non-existent') should have issues")
	}
}

func TestLoom_ProjectReadinessCache(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	projectID := "test-project"

	// First call - should check and cache
	ready1, issues1 := loom.CheckProjectReadiness(ctx, projectID)

	// Second call - should use cache
	ready2, issues2 := loom.CheckProjectReadiness(ctx, projectID)

	if ready1 != ready2 {
		t.Errorf("Cached result differs: first=%v, second=%v", ready1, ready2)
	}
	if len(issues1) != len(issues2) {
		t.Errorf("Cached issues differ: first=%d, second=%d", len(issues1), len(issues2))
	}
}

func TestLoom_InitializeWithCustomModels(t *testing.T) {
	loom, tmpDir := testLoom(t, func(c *config.Config) {
		c.Models = config.ModelsConfig{
			PreferredModels: []config.PreferredModel{
				{Name: "custom-model", Rank: 1, MinVRAMGB: 8},
				{Name: "backup-model", Rank: 2, MinVRAMGB: 4},
			},
		}
	})
	defer os.RemoveAll(tmpDir)

	catalog := loom.GetModelCatalog()
	if catalog == nil {
		t.Fatal("Model catalog not initialized")
	}

	models := catalog.List()
	if len(models) == 0 {
		t.Error("No models in catalog after loading preferred models")
	}
}

func TestLoom_ConcurrentGetters(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	done := make(chan bool)

	getters := []func(){
		func() { _ = loom.GetAgentManager(); done <- true },
		func() { _ = loom.GetDatabase(); done <- true },
		func() { _ = loom.GetProviderRegistry(); done <- true },
		func() { _ = loom.GetPersonaManager(); done <- true },
		func() { _ = loom.GetProjectManager(); done <- true },
		func() { _ = loom.GetBeadsManager(); done <- true },
		func() { _ = loom.GetEventBus(); done <- true },
		func() { _ = loom.GetDispatcher(); done <- true },
		func() { _ = loom.GetModelCatalog(); done <- true },
		func() { _ = loom.GetMetrics(); done <- true },
	}

	for _, getter := range getters {
		go getter()
	}

	for range getters {
		<-done
	}
}

func TestReadinessCacheTTL(t *testing.T) {
	if readinessCacheTTL < 1*time.Minute {
		t.Error("readinessCacheTTL is too short, may cause excessive checks")
	}
	if readinessCacheTTL > 5*time.Minute {
		t.Error("readinessCacheTTL is too long, may miss project changes")
	}
}

func TestLoom_ListModelCatalog(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Default catalog should have models
	models := loom.ListModelCatalog()
	if len(models) == 0 {
		t.Error("ListModelCatalog() returned empty list")
	}
}

func TestLoom_Shutdown(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Shutdown should not panic
	loom.Shutdown()
}

func TestLoom_GetGitopsManager(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Note: GetGitopsManager (lowercase 'o') is a separate method from GetGitOpsManager
	if loom.GetGitopsManager() == nil {
		t.Error("GetGitopsManager() returned nil")
	}
}

func TestLoom_AdvanceWorkflowWithCondition(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Should fail gracefully for non-existent bead
	err := loom.AdvanceWorkflowWithCondition("nonexistent", "agent-1", "done", nil)
	if err == nil {
		t.Error("AdvanceWorkflowWithCondition should fail for non-existent bead")
	}
}

func TestLoom_ListProviders(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// ListProviders may error if providers table doesn't exist yet
	_, err := loom.ListProviders()
	// Just ensure it doesn't panic; error is acceptable if table isn't initialized
	_ = err
}

func TestLoom_GetReadyBeads(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	beads, err := loom.GetReadyBeads("test-project")
	if err != nil {
		t.Fatalf("GetReadyBeads() error = %v", err)
	}
	if beads == nil {
		t.Error("GetReadyBeads() returned nil")
	}
}

func TestLoom_GetWorkGraph(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	graph, err := loom.GetWorkGraph("test-project")
	if err != nil {
		t.Fatalf("GetWorkGraph() error = %v", err)
	}
	if graph == nil {
		t.Error("GetWorkGraph() returned nil")
	}
}

func TestLoom_CreateBead(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Create a project first (CreateBead requires a project)
	project, err := loom.CreateProject("bead-project", ".", "", "", nil)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	bead, err := loom.CreateBead("Test Bead", "Test description", models.BeadPriorityP2, "task", project.ID)
	if err != nil {
		t.Fatalf("CreateBead() error = %v", err)
	}
	if bead == nil {
		t.Fatal("CreateBead() returned nil bead")
	}
	if bead.Title != "Test Bead" {
		t.Errorf("CreateBead() title = %q, want %q", bead.Title, "Test Bead")
	}
	if bead.Description != "Test description" {
		t.Errorf("CreateBead() description = %q, want %q", bead.Description, "Test description")
	}

	// CreateBead with non-existent project should error
	_, err = loom.CreateBead("Bad", "desc", models.BeadPriorityP2, "task", "nonexistent")
	if err == nil {
		t.Error("CreateBead with nonexistent project should fail")
	}
}

func TestLoom_CreateBead_Priorities(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	project, err := loom.CreateProject("priority-project", ".", "", "", nil)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	priorities := []models.BeadPriority{models.BeadPriorityP0, models.BeadPriorityP1, models.BeadPriorityP2, models.BeadPriorityP3}
	for i, p := range priorities {
		t.Run(fmt.Sprintf("priority_%d", i), func(t *testing.T) {
			bead, err := loom.CreateBead("Test", "desc", p, "task", project.ID)
			if err != nil {
				t.Fatalf("CreateBead(%v) error = %v", p, err)
			}
			if bead == nil {
				t.Fatalf("CreateBead(%v) returned nil", p)
			}
		})
	}
}

func TestLoom_CloseBead(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	project, err := loom.CreateProject("close-project", ".", "", "", nil)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	bead, err := loom.CreateBead("To Close", "will be closed", models.BeadPriorityP3, "task", project.ID)
	if err != nil {
		t.Fatalf("CreateBead() error = %v", err)
	}

	err = loom.CloseBead(bead.ID, "completed")
	if err != nil {
		t.Errorf("CloseBead() error = %v", err)
	}

	// Closing non-existent bead should error
	err = loom.CloseBead("nonexistent", "test")
	if err == nil {
		t.Error("CloseBead('nonexistent') should fail")
	}
}

func TestLoom_ClaimBead(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Claiming non-existent bead should error
	err := loom.ClaimBead("nonexistent", "agent-1")
	if err == nil {
		t.Error("ClaimBead('nonexistent') should fail")
	}

	// Claiming with non-existent agent should also error
	project, err := loom.CreateProject("claim-project", ".", "", "", nil)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	bead, err := loom.CreateBead("To Claim", "will be claimed", models.BeadPriorityP2, "task", project.ID)
	if err != nil {
		t.Fatalf("CreateBead() error = %v", err)
	}
	err = loom.ClaimBead(bead.ID, "nonexistent-agent")
	if err == nil {
		t.Error("ClaimBead with nonexistent agent should fail")
	}
}

func TestLoom_UpdateBead(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	project, err := loom.CreateProject("update-project", ".", "", "", nil)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	bead, err := loom.CreateBead("To Update", "original desc", models.BeadPriorityP2, "task", project.ID)
	if err != nil {
		t.Fatalf("CreateBead() error = %v", err)
	}

	updated, err := loom.UpdateBead(bead.ID, map[string]interface{}{
		"title":       "Updated Title",
		"description": "updated desc",
	})
	if err != nil {
		t.Fatalf("UpdateBead() error = %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("UpdateBead() title = %q, want %q", updated.Title, "Updated Title")
	}

	// Update non-existent bead
	_, err = loom.UpdateBead("nonexistent", map[string]interface{}{"title": "x"})
	if err == nil {
		t.Error("UpdateBead('nonexistent') should fail")
	}
}

func TestLoom_CreateProject(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	project, err := loom.CreateProject("test-project", ".", "", "", nil)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if project == nil {
		t.Fatal("CreateProject() returned nil")
	}
	if project.Name != "test-project" {
		t.Errorf("CreateProject() name = %q, want %q", project.Name, "test-project")
	}
}

func TestLoom_DeleteProject(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Create project first
	project, err := loom.CreateProject("to-delete", ".", "", "", nil)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	err = loom.DeleteProject(project.ID)
	if err != nil {
		t.Errorf("DeleteProject() error = %v", err)
	}

	// Delete non-existent project should error
	err = loom.DeleteProject("nonexistent")
	if err == nil {
		t.Error("DeleteProject('nonexistent') should fail")
	}
}

func TestLoom_RequestFileAccess(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// RequestFileAccess requires a valid agent - test error path
	_, err := loom.RequestFileAccess("proj1", "/some/file.go", "nonexistent-agent", "bead-1")
	if err == nil {
		t.Error("RequestFileAccess with nonexistent agent should fail")
	}
}

func TestLoom_ReleaseFileAccess(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Release without prior request should still work (no-op or error)
	err := loom.ReleaseFileAccess("proj1", "/some/file.go", "agent-1")
	// Just verify it doesn't panic
	_ = err
}

func TestLoom_GetCommandLog(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	_, err := loom.GetCommandLog("nonexistent")
	// May or may not error depending on implementation
	_ = err
}

func TestLoom_setupProviderMetrics(t *testing.T) {
	loom, tmpDir := testLoom(t)
	defer os.RemoveAll(tmpDir)

	// Should not panic
	loom.setupProviderMetrics()
}
