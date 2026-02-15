package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/memory"
	internalmodels "github.com/jordanhubbard/loom/internal/models"
	"github.com/jordanhubbard/loom/internal/workflow"
	"github.com/jordanhubbard/loom/pkg/models"
)

// newTestDB creates a fresh in-memory SQLite database for testing.
func newTestDB(t *testing.T) *Database {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ---------------------------------------------------------------------------
// 1. Core: New, Close, DB, Type, SupportsHA
// ---------------------------------------------------------------------------

func TestNew_Success(t *testing.T) {
	db := newTestDB(t)
	if db == nil {
		t.Fatal("Expected non-nil Database")
	}
}

func TestNew_InvalidPath(t *testing.T) {
	// A path that cannot be opened should produce an error during schema init
	// (SQLite creates files, but a deeply nested non-existent dir should fail)
	_, err := New("/nonexistent/deeply/nested/path/db.sqlite")
	if err == nil {
		t.Fatal("Expected error for invalid database path, got nil")
	}
}

func TestDB_ReturnsUnderlyingDB(t *testing.T) {
	db := newTestDB(t)
	sqlDB := db.DB()
	if sqlDB == nil {
		t.Fatal("DB() returned nil")
	}
	// Verify the connection is alive
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestType_ReturnsSqlite(t *testing.T) {
	db := newTestDB(t)
	if got := db.Type(); got != "sqlite" {
		t.Errorf("Type() = %q, want %q", got, "sqlite")
	}
}

func TestSupportsHA_ReturnsFalse(t *testing.T) {
	db := newTestDB(t)
	if db.SupportsHA() {
		t.Error("SupportsHA() should return false for SQLite")
	}
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "close_test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}
	// After close, operations should fail
	if err := db.DB().Ping(); err == nil {
		t.Error("Expected error after Close(), got nil")
	}
}

// ---------------------------------------------------------------------------
// 2. Config KV: SetConfigValue, GetConfigValue
// ---------------------------------------------------------------------------

func TestConfigValue_SetAndGet(t *testing.T) {
	db := newTestDB(t)

	err := db.SetConfigValue("theme", "dark")
	if err != nil {
		t.Fatalf("SetConfigValue failed: %v", err)
	}

	val, found, err := db.GetConfigValue("theme")
	if err != nil {
		t.Fatalf("GetConfigValue failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find config key 'theme'")
	}
	if val != "dark" {
		t.Errorf("GetConfigValue = %q, want %q", val, "dark")
	}
}

func TestConfigValue_GetMissingKey(t *testing.T) {
	db := newTestDB(t)

	val, found, err := db.GetConfigValue("nonexistent")
	if err != nil {
		t.Fatalf("GetConfigValue failed: %v", err)
	}
	if found {
		t.Error("Expected found=false for missing key")
	}
	if val != "" {
		t.Errorf("Expected empty string for missing key, got %q", val)
	}
}

func TestConfigValue_Update(t *testing.T) {
	db := newTestDB(t)

	if err := db.SetConfigValue("color", "red"); err != nil {
		t.Fatalf("SetConfigValue failed: %v", err)
	}
	if err := db.SetConfigValue("color", "blue"); err != nil {
		t.Fatalf("SetConfigValue (update) failed: %v", err)
	}

	val, found, err := db.GetConfigValue("color")
	if err != nil {
		t.Fatalf("GetConfigValue failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find config key 'color'")
	}
	if val != "blue" {
		t.Errorf("GetConfigValue after update = %q, want %q", val, "blue")
	}
}

func TestConfigValue_MultipleKeys(t *testing.T) {
	db := newTestDB(t)

	keys := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	for k, v := range keys {
		if err := db.SetConfigValue(k, v); err != nil {
			t.Fatalf("SetConfigValue(%q) failed: %v", k, err)
		}
	}
	for k, want := range keys {
		got, found, err := db.GetConfigValue(k)
		if err != nil {
			t.Fatalf("GetConfigValue(%q) failed: %v", k, err)
		}
		if !found {
			t.Errorf("Expected to find key %q", k)
		}
		if got != want {
			t.Errorf("GetConfigValue(%q) = %q, want %q", k, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// 3. Projects: UpsertProject, ListProjects, DeleteProject
// ---------------------------------------------------------------------------

func makeTestProject(id, name string) *models.Project {
	return &models.Project{
		ID:          id,
		Name:        name,
		GitRepo:     "https://github.com/test/" + name,
		Branch:      "main",
		BeadsPath:   ".beads",
		GitStrategy: models.GitStrategyDirect,
		Status:      models.ProjectStatusOpen,
		Context:     map[string]string{"env": "test"},
	}
}

func TestUpsertProject_Create(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProject("proj-1", "TestProject")

	if err := db.UpsertProject(p); err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}
	if p.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if p.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestUpsertProject_NilProject(t *testing.T) {
	db := newTestDB(t)
	err := db.UpsertProject(nil)
	if err == nil {
		t.Fatal("Expected error for nil project, got nil")
	}
}

func TestUpsertProject_Update(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProject("proj-1", "Original")
	if err := db.UpsertProject(p); err != nil {
		t.Fatalf("UpsertProject (create) failed: %v", err)
	}

	p.Name = "Updated"
	p.IsPerpetual = true
	p.IsSticky = true
	p.Status = models.ProjectStatusClosed
	if err := db.UpsertProject(p); err != nil {
		t.Fatalf("UpsertProject (update) failed: %v", err)
	}

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}
	if projects[0].Name != "Updated" {
		t.Errorf("Name = %q, want %q", projects[0].Name, "Updated")
	}
	if !projects[0].IsPerpetual {
		t.Error("Expected IsPerpetual = true")
	}
	if !projects[0].IsSticky {
		t.Error("Expected IsSticky = true")
	}
	if projects[0].Status != models.ProjectStatusClosed {
		t.Errorf("Status = %q, want %q", projects[0].Status, models.ProjectStatusClosed)
	}
}

func TestUpsertProject_DefaultGitStrategy(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProject("proj-default-git", "GitDefault")
	p.GitStrategy = "" // empty should default to "direct"
	if err := db.UpsertProject(p); err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}
	if projects[0].GitStrategy != models.GitStrategyDirect {
		t.Errorf("GitStrategy = %q, want %q", projects[0].GitStrategy, models.GitStrategyDirect)
	}
}

func TestUpsertProject_WithContext(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProject("proj-ctx", "ContextProject")
	p.Context = map[string]string{
		"language": "go",
		"team":     "backend",
	}
	if err := db.UpsertProject(p); err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}
	if projects[0].Context["language"] != "go" {
		t.Errorf("Context[language] = %q, want %q", projects[0].Context["language"], "go")
	}
	if projects[0].Context["team"] != "backend" {
		t.Errorf("Context[team] = %q, want %q", projects[0].Context["team"], "backend")
	}
}

func TestUpsertProject_NilContext(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProject("proj-nil-ctx", "NilContextProject")
	p.Context = nil
	if err := db.UpsertProject(p); err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}
	// nil context should be returned as empty map
	if projects[0].Context == nil {
		t.Error("Expected non-nil Context map")
	}
}

func TestListProjects_Empty(t *testing.T) {
	db := newTestDB(t)

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
}

func TestListProjects_Multiple(t *testing.T) {
	db := newTestDB(t)

	for i := 0; i < 5; i++ {
		p := makeTestProject(
			"proj-"+string(rune('a'+i)),
			"Project"+string(rune('A'+i)),
		)
		if err := db.UpsertProject(p); err != nil {
			t.Fatalf("UpsertProject failed for project %d: %v", i, err)
		}
	}

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 5 {
		t.Errorf("Expected 5 projects, got %d", len(projects))
	}
}

func TestListProjects_FieldIntegrity(t *testing.T) {
	db := newTestDB(t)

	p := makeTestProject("proj-fields", "FieldTest")
	p.GitStrategy = models.GitStrategyBranch
	p.IsPerpetual = true
	p.IsSticky = true
	p.Status = models.ProjectStatusReopened
	if err := db.UpsertProject(p); err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}

	got := projects[0]
	if got.ID != "proj-fields" {
		t.Errorf("ID = %q, want %q", got.ID, "proj-fields")
	}
	if got.Name != "FieldTest" {
		t.Errorf("Name = %q, want %q", got.Name, "FieldTest")
	}
	if got.GitRepo != "https://github.com/test/FieldTest" {
		t.Errorf("GitRepo = %q, want %q", got.GitRepo, "https://github.com/test/FieldTest")
	}
	if got.Branch != "main" {
		t.Errorf("Branch = %q, want %q", got.Branch, "main")
	}
	if got.BeadsPath != ".beads" {
		t.Errorf("BeadsPath = %q, want %q", got.BeadsPath, ".beads")
	}
	if got.GitStrategy != models.GitStrategyBranch {
		t.Errorf("GitStrategy = %q, want %q", got.GitStrategy, models.GitStrategyBranch)
	}
	if !got.IsPerpetual {
		t.Error("Expected IsPerpetual = true")
	}
	if !got.IsSticky {
		t.Error("Expected IsSticky = true")
	}
	if got.Status != models.ProjectStatusReopened {
		t.Errorf("Status = %q, want %q", got.Status, models.ProjectStatusReopened)
	}
	if got.Agents == nil {
		t.Error("Expected Agents to be non-nil (empty slice)")
	}
	if got.Comments == nil {
		t.Error("Expected Comments to be non-nil (empty slice)")
	}
}

func TestDeleteProject_Existing(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProject("proj-del", "DeleteMe")
	if err := db.UpsertProject(p); err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	if err := db.DeleteProject("proj-del"); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects after delete, got %d", len(projects))
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.DeleteProject("nonexistent-id")
	if err == nil {
		t.Fatal("Expected error when deleting non-existent project, got nil")
	}
}

func TestDeleteAllProjects(t *testing.T) {
	db := newTestDB(t)

	for i := 0; i < 3; i++ {
		p := makeTestProject("proj-all-"+string(rune('0'+i)), "Proj")
		if err := db.UpsertProject(p); err != nil {
			t.Fatalf("UpsertProject failed: %v", err)
		}
	}

	if err := db.DeleteAllProjects(); err != nil {
		t.Fatalf("DeleteAllProjects failed: %v", err)
	}

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects after DeleteAll, got %d", len(projects))
	}
}

func TestDeleteAllProjects_EmptyDB(t *testing.T) {
	db := newTestDB(t)
	// Should not error on empty table
	if err := db.DeleteAllProjects(); err != nil {
		t.Fatalf("DeleteAllProjects on empty DB failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 4. Agents: UpsertAgent, ListAgents, DeleteAgent
// ---------------------------------------------------------------------------

func makeTestAgent(id, name string) *models.Agent {
	return &models.Agent{
		ID:          id,
		Name:        name,
		Role:        "developer",
		PersonaName: "test-persona",
		Status:      "idle",
	}
}

func TestUpsertAgent_Create(t *testing.T) {
	db := newTestDB(t)
	a := makeTestAgent("agent-1", "Agent One")
	if err := db.UpsertAgent(a); err != nil {
		t.Fatalf("UpsertAgent failed: %v", err)
	}
	if a.StartedAt.IsZero() {
		t.Error("Expected StartedAt to be set")
	}
	if a.LastActive.IsZero() {
		t.Error("Expected LastActive to be set")
	}
}

func TestUpsertAgent_NilAgent(t *testing.T) {
	db := newTestDB(t)
	err := db.UpsertAgent(nil)
	if err == nil {
		t.Fatal("Expected error for nil agent, got nil")
	}
}

func TestUpsertAgent_Update(t *testing.T) {
	db := newTestDB(t)
	a := makeTestAgent("agent-1", "Original")
	if err := db.UpsertAgent(a); err != nil {
		t.Fatalf("UpsertAgent (create) failed: %v", err)
	}

	a.Name = "Updated Agent"
	a.Status = "working"
	a.Role = "reviewer"
	if err := db.UpsertAgent(a); err != nil {
		t.Fatalf("UpsertAgent (update) failed: %v", err)
	}

	agents, err := db.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(agents))
	}
	if agents[0].Name != "Updated Agent" {
		t.Errorf("Name = %q, want %q", agents[0].Name, "Updated Agent")
	}
	if agents[0].Status != "working" {
		t.Errorf("Status = %q, want %q", agents[0].Status, "working")
	}
	if agents[0].Role != "reviewer" {
		t.Errorf("Role = %q, want %q", agents[0].Role, "reviewer")
	}
}

func TestUpsertAgent_WithTimestamps(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().Add(-1 * time.Hour) // set explicit past time
	a := makeTestAgent("agent-ts", "Timestamp Agent")
	a.StartedAt = now
	a.LastActive = now

	if err := db.UpsertAgent(a); err != nil {
		t.Fatalf("UpsertAgent failed: %v", err)
	}

	agents, err := db.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(agents))
	}
	// The timestamps should be roughly what we set (within a second)
	diff := agents[0].StartedAt.Sub(now)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("StartedAt difference too large: %v", diff)
	}
}

func TestUpsertAgent_WithOptionalFields(t *testing.T) {
	db := newTestDB(t)

	// First create a provider so we can reference it
	provider := makeTestProvider("prov-for-agent", "TestProvider")
	if err := db.UpsertProvider(provider); err != nil {
		t.Fatalf("UpsertProvider failed: %v", err)
	}

	// Create a project so we can reference it
	p := makeTestProject("proj-for-agent", "AgentProject")
	if err := db.UpsertProject(p); err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	a := makeTestAgent("agent-opt", "OptionalFields")
	a.ProviderID = "prov-for-agent"
	a.ProjectID = "proj-for-agent"
	a.CurrentBead = "bead-123"

	if err := db.UpsertAgent(a); err != nil {
		t.Fatalf("UpsertAgent failed: %v", err)
	}

	agents, err := db.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(agents))
	}
	if agents[0].ProviderID != "prov-for-agent" {
		t.Errorf("ProviderID = %q, want %q", agents[0].ProviderID, "prov-for-agent")
	}
	if agents[0].ProjectID != "proj-for-agent" {
		t.Errorf("ProjectID = %q, want %q", agents[0].ProjectID, "proj-for-agent")
	}
	if agents[0].CurrentBead != "bead-123" {
		t.Errorf("CurrentBead = %q, want %q", agents[0].CurrentBead, "bead-123")
	}
}

func TestUpsertAgent_EmptyOptionalFields(t *testing.T) {
	db := newTestDB(t)
	a := makeTestAgent("agent-empty", "EmptyOptional")
	// ProviderID, ProjectID, CurrentBead all empty -- should be stored as NULL

	if err := db.UpsertAgent(a); err != nil {
		t.Fatalf("UpsertAgent failed: %v", err)
	}

	agents, err := db.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(agents))
	}
	if agents[0].ProviderID != "" {
		t.Errorf("ProviderID = %q, want empty", agents[0].ProviderID)
	}
	if agents[0].ProjectID != "" {
		t.Errorf("ProjectID = %q, want empty", agents[0].ProjectID)
	}
	if agents[0].CurrentBead != "" {
		t.Errorf("CurrentBead = %q, want empty", agents[0].CurrentBead)
	}
}

func TestListAgents_Empty(t *testing.T) {
	db := newTestDB(t)
	agents, err := db.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}
}

func TestListAgents_Multiple(t *testing.T) {
	db := newTestDB(t)
	for i := 0; i < 4; i++ {
		a := makeTestAgent(
			"agent-multi-"+string(rune('0'+i)),
			"Agent"+string(rune('A'+i)),
		)
		if err := db.UpsertAgent(a); err != nil {
			t.Fatalf("UpsertAgent failed: %v", err)
		}
	}

	agents, err := db.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 4 {
		t.Errorf("Expected 4 agents, got %d", len(agents))
	}
}

func TestDeleteAgent_Existing(t *testing.T) {
	db := newTestDB(t)
	a := makeTestAgent("agent-del", "DeleteMe")
	if err := db.UpsertAgent(a); err != nil {
		t.Fatalf("UpsertAgent failed: %v", err)
	}

	if err := db.DeleteAgent("agent-del"); err != nil {
		t.Fatalf("DeleteAgent failed: %v", err)
	}

	agents, err := db.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents after delete, got %d", len(agents))
	}
}

func TestDeleteAgent_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.DeleteAgent("nonexistent")
	if err == nil {
		t.Fatal("Expected error when deleting non-existent agent, got nil")
	}
}

func TestDeleteAllAgents(t *testing.T) {
	db := newTestDB(t)
	for i := 0; i < 3; i++ {
		a := makeTestAgent("agent-all-"+string(rune('0'+i)), "Agent")
		if err := db.UpsertAgent(a); err != nil {
			t.Fatalf("UpsertAgent failed: %v", err)
		}
	}

	if err := db.DeleteAllAgents(); err != nil {
		t.Fatalf("DeleteAllAgents failed: %v", err)
	}

	agents, err := db.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents after DeleteAll, got %d", len(agents))
	}
}

func TestDeleteAllAgents_EmptyDB(t *testing.T) {
	db := newTestDB(t)
	if err := db.DeleteAllAgents(); err != nil {
		t.Fatalf("DeleteAllAgents on empty DB failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 5. Providers: CreateProvider, UpsertProvider, GetProvider, ListProviders,
//    UpdateProvider, DeleteProvider, DeleteAllProviders
// ---------------------------------------------------------------------------

func makeTestProvider(id, name string) *internalmodels.Provider {
	return &internalmodels.Provider{
		ID:          id,
		Name:        name,
		Type:        "openai",
		Endpoint:    "http://localhost:8080",
		Model:       "gpt-4",
		Description: "Test provider",
		RequiresKey: false,
		Status:      "active",
	}
}

func TestCreateProvider(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProvider("prov-create", "CreateTest")

	if err := db.CreateProvider(p); err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	if p.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if p.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify it was persisted by querying the count directly.
	// Note: CreateProvider inserts fewer columns than GetProvider/ListProviders
	// scan, so reading back via those methods encounters NULL-to-string scan
	// errors. We verify persistence with a simple count query.
	var count int
	err := db.DB().QueryRow("SELECT COUNT(*) FROM providers WHERE id = ?", "prov-create").Scan(&count)
	if err != nil {
		t.Fatalf("Count query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}
}

func TestCreateProvider_Duplicate(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProvider("prov-dup", "Duplicate")
	if err := db.CreateProvider(p); err != nil {
		t.Fatalf("CreateProvider (first) failed: %v", err)
	}

	// Creating again with the same ID should fail (INSERT, not UPSERT)
	p2 := makeTestProvider("prov-dup", "Duplicate2")
	err := db.CreateProvider(p2)
	if err == nil {
		t.Fatal("Expected error on duplicate CreateProvider, got nil")
	}
}

func TestUpsertProvider_Create(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProvider("prov-upsert-new", "UpsertNew")

	if err := db.UpsertProvider(p); err != nil {
		t.Fatalf("UpsertProvider failed: %v", err)
	}

	got, err := db.GetProvider("prov-upsert-new")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}
	if got.Name != "UpsertNew" {
		t.Errorf("Name = %q, want %q", got.Name, "UpsertNew")
	}
}

func TestUpsertProvider_NilProvider(t *testing.T) {
	db := newTestDB(t)
	err := db.UpsertProvider(nil)
	if err == nil {
		t.Fatal("Expected error for nil provider, got nil")
	}
}

func TestUpsertProvider_Update(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProvider("prov-upsert-upd", "OriginalProvider")
	if err := db.UpsertProvider(p); err != nil {
		t.Fatalf("UpsertProvider (create) failed: %v", err)
	}

	p.Name = "UpdatedProvider"
	p.Model = "gpt-4-turbo"
	p.Status = "inactive"
	p.ConfiguredModel = "gpt-4-turbo"
	p.SelectedModel = "gpt-4-turbo"
	p.SelectionReason = "best performance"
	p.ModelScore = 0.95
	p.SelectedGPU = "A100"
	p.OwnerID = "user-1"
	p.IsShared = true
	p.ContextWindow = 128000
	if err := db.UpsertProvider(p); err != nil {
		t.Fatalf("UpsertProvider (update) failed: %v", err)
	}

	got, err := db.GetProvider("prov-upsert-upd")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}
	if got.Name != "UpdatedProvider" {
		t.Errorf("Name = %q, want %q", got.Name, "UpdatedProvider")
	}
	if got.Model != "gpt-4-turbo" {
		t.Errorf("Model = %q, want %q", got.Model, "gpt-4-turbo")
	}
	if got.Status != "inactive" {
		t.Errorf("Status = %q, want %q", got.Status, "inactive")
	}
	if got.ConfiguredModel != "gpt-4-turbo" {
		t.Errorf("ConfiguredModel = %q, want %q", got.ConfiguredModel, "gpt-4-turbo")
	}
	if got.SelectedModel != "gpt-4-turbo" {
		t.Errorf("SelectedModel = %q, want %q", got.SelectedModel, "gpt-4-turbo")
	}
	if got.SelectionReason != "best performance" {
		t.Errorf("SelectionReason = %q, want %q", got.SelectionReason, "best performance")
	}
	if got.ContextWindow != 128000 {
		t.Errorf("ContextWindow = %d, want %d", got.ContextWindow, 128000)
	}
}

func TestUpsertProvider_PreservesCreatedAt(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProvider("prov-preserve-ts", "PreserveTS")
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	p.CreatedAt = fixedTime

	if err := db.UpsertProvider(p); err != nil {
		t.Fatalf("UpsertProvider failed: %v", err)
	}
	// CreatedAt was set, so it should be preserved (not overwritten to now)
	if p.CreatedAt != fixedTime {
		t.Errorf("CreatedAt was modified; got %v, want %v", p.CreatedAt, fixedTime)
	}
}

func TestGetProvider_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.GetProvider("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent provider, got nil")
	}
}

func TestGetProvider_FieldIntegrity(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProvider("prov-fields", "FieldsProvider")
	p.RequiresKey = true
	p.KeyID = "key-abc"
	p.Description = "A description"
	p.LastHeartbeatLatencyMs = 42
	p.LastHeartbeatError = "timeout"

	if err := db.UpsertProvider(p); err != nil {
		t.Fatalf("UpsertProvider failed: %v", err)
	}

	got, err := db.GetProvider("prov-fields")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}
	if got.ID != "prov-fields" {
		t.Errorf("ID = %q, want %q", got.ID, "prov-fields")
	}
	if got.Type != "openai" {
		t.Errorf("Type = %q, want %q", got.Type, "openai")
	}
	if got.Endpoint != "http://localhost:8080" {
		t.Errorf("Endpoint = %q, want %q", got.Endpoint, "http://localhost:8080")
	}
	if got.RequiresKey != true {
		t.Error("Expected RequiresKey = true")
	}
	if got.KeyID != "key-abc" {
		t.Errorf("KeyID = %q, want %q", got.KeyID, "key-abc")
	}
	if got.Description != "A description" {
		t.Errorf("Description = %q, want %q", got.Description, "A description")
	}
}

func TestListProviders_Empty(t *testing.T) {
	db := newTestDB(t)
	providers, err := db.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
}

func TestListProviders_Multiple(t *testing.T) {
	db := newTestDB(t)
	for i := 0; i < 3; i++ {
		p := makeTestProvider(
			"prov-list-"+string(rune('0'+i)),
			"Provider"+string(rune('A'+i)),
		)
		if err := db.UpsertProvider(p); err != nil {
			t.Fatalf("UpsertProvider failed: %v", err)
		}
	}

	providers, err := db.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providers))
	}
}

func TestListProviders_OwnerAndSharedFields(t *testing.T) {
	db := newTestDB(t)

	p := makeTestProvider("prov-owner", "OwnedProvider")
	p.OwnerID = "user-42"
	p.IsShared = false
	if err := db.UpsertProvider(p); err != nil {
		t.Fatalf("UpsertProvider failed: %v", err)
	}

	providers, err := db.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(providers))
	}
	if providers[0].OwnerID != "user-42" {
		t.Errorf("OwnerID = %q, want %q", providers[0].OwnerID, "user-42")
	}
	if providers[0].IsShared != false {
		t.Error("Expected IsShared = false")
	}
}

func TestListProvidersForUser(t *testing.T) {
	db := newTestDB(t)

	// Shared provider (no owner)
	shared := makeTestProvider("prov-shared", "Shared")
	shared.IsShared = true
	if err := db.UpsertProvider(shared); err != nil {
		t.Fatalf("UpsertProvider (shared) failed: %v", err)
	}

	// Private provider owned by user-1
	private1 := makeTestProvider("prov-private1", "Private1")
	private1.OwnerID = "user-1"
	private1.IsShared = false
	if err := db.UpsertProvider(private1); err != nil {
		t.Fatalf("UpsertProvider (private1) failed: %v", err)
	}

	// Private provider owned by user-2
	private2 := makeTestProvider("prov-private2", "Private2")
	private2.OwnerID = "user-2"
	private2.IsShared = false
	if err := db.UpsertProvider(private2); err != nil {
		t.Fatalf("UpsertProvider (private2) failed: %v", err)
	}

	// user-1 should see shared + their own
	providers, err := db.ListProvidersForUser("user-1")
	if err != nil {
		t.Fatalf("ListProvidersForUser failed: %v", err)
	}
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers for user-1, got %d", len(providers))
	}

	// user-2 should see shared + their own
	providers2, err := db.ListProvidersForUser("user-2")
	if err != nil {
		t.Fatalf("ListProvidersForUser failed: %v", err)
	}
	if len(providers2) != 2 {
		t.Errorf("Expected 2 providers for user-2, got %d", len(providers2))
	}

	// unknown user should see only shared
	providers3, err := db.ListProvidersForUser("user-unknown")
	if err != nil {
		t.Fatalf("ListProvidersForUser failed: %v", err)
	}
	if len(providers3) != 1 {
		t.Errorf("Expected 1 provider for unknown user, got %d", len(providers3))
	}
}

func TestUpdateProvider(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProvider("prov-upd", "BeforeUpdate")
	// Use UpsertProvider for creation so all columns are populated
	if err := db.UpsertProvider(p); err != nil {
		t.Fatalf("UpsertProvider failed: %v", err)
	}

	p.Name = "AfterUpdate"
	p.Type = "anthropic"
	p.Endpoint = "http://new-endpoint:9090"
	p.Model = "claude-3"
	p.Description = "Updated description"
	p.RequiresKey = true
	p.KeyID = "new-key"
	p.Status = "inactive"

	if err := db.UpdateProvider(p); err != nil {
		t.Fatalf("UpdateProvider failed: %v", err)
	}

	got, err := db.GetProvider("prov-upd")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}
	if got.Name != "AfterUpdate" {
		t.Errorf("Name = %q, want %q", got.Name, "AfterUpdate")
	}
	if got.Type != "anthropic" {
		t.Errorf("Type = %q, want %q", got.Type, "anthropic")
	}
	if got.Endpoint != "http://new-endpoint:9090" {
		t.Errorf("Endpoint = %q, want %q", got.Endpoint, "http://new-endpoint:9090")
	}
	if got.Model != "claude-3" {
		t.Errorf("Model = %q, want %q", got.Model, "claude-3")
	}
	if got.Description != "Updated description" {
		t.Errorf("Description = %q, want %q", got.Description, "Updated description")
	}
	if !got.RequiresKey {
		t.Error("Expected RequiresKey = true")
	}
	if got.Status != "inactive" {
		t.Errorf("Status = %q, want %q", got.Status, "inactive")
	}
}

func TestUpdateProvider_NotFound(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProvider("prov-ghost", "Ghost")
	err := db.UpdateProvider(p)
	if err == nil {
		t.Fatal("Expected error when updating non-existent provider, got nil")
	}
}

func TestDeleteProvider_Existing(t *testing.T) {
	db := newTestDB(t)
	p := makeTestProvider("prov-del", "DeleteMe")
	if err := db.UpsertProvider(p); err != nil {
		t.Fatalf("UpsertProvider failed: %v", err)
	}

	if err := db.DeleteProvider("prov-del"); err != nil {
		t.Fatalf("DeleteProvider failed: %v", err)
	}

	_, err := db.GetProvider("prov-del")
	if err == nil {
		t.Fatal("Expected error after deleting provider, got nil")
	}
}

func TestDeleteProvider_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.DeleteProvider("nonexistent")
	if err == nil {
		t.Fatal("Expected error when deleting non-existent provider, got nil")
	}
}

func TestDeleteAllProviders(t *testing.T) {
	db := newTestDB(t)
	for i := 0; i < 3; i++ {
		p := makeTestProvider("prov-da-"+string(rune('0'+i)), "Provider")
		if err := db.UpsertProvider(p); err != nil {
			t.Fatalf("UpsertProvider failed: %v", err)
		}
	}

	if err := db.DeleteAllProviders(); err != nil {
		t.Fatalf("DeleteAllProviders failed: %v", err)
	}

	providers, err := db.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers after DeleteAll, got %d", len(providers))
	}
}

func TestDeleteAllProviders_EmptyDB(t *testing.T) {
	db := newTestDB(t)
	if err := db.DeleteAllProviders(); err != nil {
		t.Fatalf("DeleteAllProviders on empty DB failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 6. Credentials: UpsertCredential, GetCredentialByProjectID,
//    GetCredential, DeleteCredential
// ---------------------------------------------------------------------------

func makeTestCredential(id, projectID string) *models.Credential {
	return &models.Credential{
		ID:                  id,
		ProjectID:           projectID,
		Type:                "ssh_ed25519",
		PrivateKeyEncrypted: "encrypted-key-data",
		PublicKey:           "ssh-ed25519 AAAA...",
		KeyID:               "km-key-1",
		Description:         "Test credential",
	}
}

func ensureProjectExists(t *testing.T, db *Database, projectID string) {
	t.Helper()
	p := makeTestProject(projectID, "Project-"+projectID)
	if err := db.UpsertProject(p); err != nil {
		t.Fatalf("Failed to create prerequisite project %q: %v", projectID, err)
	}
}

func ensureUserExists(t *testing.T, db *Database, userID, username string) {
	t.Helper()
	if err := db.CreateUser(userID, username, username+"@test.com", "member"); err != nil {
		// Ignore duplicate errors
		_ = err
	}
}

func TestUpsertCredential_Create(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-1")
	cred := makeTestCredential("cred-1", "proj-1")
	if err := db.UpsertCredential(cred); err != nil {
		t.Fatalf("UpsertCredential failed: %v", err)
	}
	if cred.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if cred.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestUpsertCredential_Update(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-upd")
	cred := makeTestCredential("cred-upd", "proj-upd")
	if err := db.UpsertCredential(cred); err != nil {
		t.Fatalf("UpsertCredential (create) failed: %v", err)
	}

	cred.PrivateKeyEncrypted = "new-encrypted-data"
	cred.PublicKey = "ssh-ed25519 BBBB..."
	cred.Description = "Updated credential"
	now := time.Now()
	cred.RotatedAt = &now
	if err := db.UpsertCredential(cred); err != nil {
		t.Fatalf("UpsertCredential (update) failed: %v", err)
	}

	got, err := db.GetCredential("cred-upd")
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected non-nil credential")
	}
	if got.PrivateKeyEncrypted != "new-encrypted-data" {
		t.Errorf("PrivateKeyEncrypted = %q, want %q", got.PrivateKeyEncrypted, "new-encrypted-data")
	}
	if got.PublicKey != "ssh-ed25519 BBBB..." {
		t.Errorf("PublicKey = %q, want %q", got.PublicKey, "ssh-ed25519 BBBB...")
	}
	if got.Description != "Updated credential" {
		t.Errorf("Description = %q, want %q", got.Description, "Updated credential")
	}
	if got.RotatedAt == nil {
		t.Error("Expected RotatedAt to be set")
	}
}

func TestGetCredential_ByID(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-get")
	cred := makeTestCredential("cred-get", "proj-get")
	if err := db.UpsertCredential(cred); err != nil {
		t.Fatalf("UpsertCredential failed: %v", err)
	}

	got, err := db.GetCredential("cred-get")
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected non-nil credential")
	}
	if got.ID != "cred-get" {
		t.Errorf("ID = %q, want %q", got.ID, "cred-get")
	}
	if got.ProjectID != "proj-get" {
		t.Errorf("ProjectID = %q, want %q", got.ProjectID, "proj-get")
	}
	if got.Type != "ssh_ed25519" {
		t.Errorf("Type = %q, want %q", got.Type, "ssh_ed25519")
	}
	if got.PrivateKeyEncrypted != "encrypted-key-data" {
		t.Errorf("PrivateKeyEncrypted = %q, want %q", got.PrivateKeyEncrypted, "encrypted-key-data")
	}
	if got.PublicKey != "ssh-ed25519 AAAA..." {
		t.Errorf("PublicKey = %q, want %q", got.PublicKey, "ssh-ed25519 AAAA...")
	}
	if got.KeyID != "km-key-1" {
		t.Errorf("KeyID = %q, want %q", got.KeyID, "km-key-1")
	}
	if got.Description != "Test credential" {
		t.Errorf("Description = %q, want %q", got.Description, "Test credential")
	}
}

func TestGetCredential_NotFound(t *testing.T) {
	db := newTestDB(t)
	got, err := db.GetCredential("nonexistent")
	if err != nil {
		t.Fatalf("GetCredential should not error for missing credential: %v", err)
	}
	if got != nil {
		t.Error("Expected nil credential for non-existent ID")
	}
}

func TestGetCredentialByProjectID(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-abc")
	cred := makeTestCredential("cred-proj", "proj-abc")
	if err := db.UpsertCredential(cred); err != nil {
		t.Fatalf("UpsertCredential failed: %v", err)
	}

	got, err := db.GetCredentialByProjectID("proj-abc")
	if err != nil {
		t.Fatalf("GetCredentialByProjectID failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected non-nil credential")
	}
	if got.ProjectID != "proj-abc" {
		t.Errorf("ProjectID = %q, want %q", got.ProjectID, "proj-abc")
	}
}

func TestGetCredentialByProjectID_NotFound(t *testing.T) {
	db := newTestDB(t)
	got, err := db.GetCredentialByProjectID("nonexistent-proj")
	if err != nil {
		t.Fatalf("GetCredentialByProjectID should not error for missing: %v", err)
	}
	if got != nil {
		t.Error("Expected nil credential for non-existent project ID")
	}
}

func TestGetCredentialByProjectID_MultipleCredentials(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-multi")
	// Insert two credentials for the same project
	cred1 := makeTestCredential("cred-multi-1", "proj-multi")
	cred2 := makeTestCredential("cred-multi-2", "proj-multi")
	if err := db.UpsertCredential(cred1); err != nil {
		t.Fatalf("UpsertCredential failed: %v", err)
	}
	if err := db.UpsertCredential(cred2); err != nil {
		t.Fatalf("UpsertCredential failed: %v", err)
	}

	// Should return one (LIMIT 1)
	got, err := db.GetCredentialByProjectID("proj-multi")
	if err != nil {
		t.Fatalf("GetCredentialByProjectID failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected non-nil credential")
	}
	if got.ProjectID != "proj-multi" {
		t.Errorf("ProjectID = %q, want %q", got.ProjectID, "proj-multi")
	}
}

func TestDeleteCredential(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-del")
	cred := makeTestCredential("cred-del", "proj-del")
	if err := db.UpsertCredential(cred); err != nil {
		t.Fatalf("UpsertCredential failed: %v", err)
	}

	if err := db.DeleteCredential("cred-del"); err != nil {
		t.Fatalf("DeleteCredential failed: %v", err)
	}

	got, err := db.GetCredential("cred-del")
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}
	if got != nil {
		t.Error("Expected nil credential after delete")
	}
}

func TestDeleteCredential_NotFound(t *testing.T) {
	db := newTestDB(t)
	// DeleteCredential does not error on missing rows (no RowsAffected check)
	err := db.DeleteCredential("nonexistent")
	if err != nil {
		t.Fatalf("DeleteCredential should not error for missing credential: %v", err)
	}
}

func TestCredential_WithoutOptionalFields(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-minimal")
	cred := &models.Credential{
		ID:                  "cred-minimal",
		ProjectID:           "proj-minimal",
		Type:                "ssh_ed25519",
		PrivateKeyEncrypted: "encrypted",
		PublicKey:           "public",
		// KeyID, Description, RotatedAt all omitted
	}
	if err := db.UpsertCredential(cred); err != nil {
		t.Fatalf("UpsertCredential failed: %v", err)
	}

	got, err := db.GetCredential("cred-minimal")
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected non-nil credential")
	}
	if got.KeyID != "" {
		t.Errorf("KeyID = %q, want empty", got.KeyID)
	}
	if got.Description != "" {
		t.Errorf("Description = %q, want empty", got.Description)
	}
	if got.RotatedAt != nil {
		t.Error("Expected RotatedAt to be nil")
	}
}

// ---------------------------------------------------------------------------
// 7. Integration / cross-entity tests
// ---------------------------------------------------------------------------

func TestAgentWithProjectAndProvider(t *testing.T) {
	db := newTestDB(t)

	// Create a provider
	prov := makeTestProvider("prov-int", "IntProvider")
	if err := db.UpsertProvider(prov); err != nil {
		t.Fatalf("UpsertProvider failed: %v", err)
	}

	// Create a project
	proj := makeTestProject("proj-int", "IntProject")
	if err := db.UpsertProject(proj); err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	// Create an agent linked to both
	agent := makeTestAgent("agent-int", "IntAgent")
	agent.ProviderID = "prov-int"
	agent.ProjectID = "proj-int"
	if err := db.UpsertAgent(agent); err != nil {
		t.Fatalf("UpsertAgent failed: %v", err)
	}

	// Verify
	agents, err := db.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(agents))
	}
	if agents[0].ProviderID != "prov-int" {
		t.Errorf("ProviderID = %q, want %q", agents[0].ProviderID, "prov-int")
	}
	if agents[0].ProjectID != "proj-int" {
		t.Errorf("ProjectID = %q, want %q", agents[0].ProjectID, "proj-int")
	}
}

func TestDeleteAllEntities_Independence(t *testing.T) {
	db := newTestDB(t)

	// Create providers, projects, and agents
	prov := makeTestProvider("prov-indep", "IndepProvider")
	if err := db.UpsertProvider(prov); err != nil {
		t.Fatalf("UpsertProvider failed: %v", err)
	}
	proj := makeTestProject("proj-indep", "IndepProject")
	if err := db.UpsertProject(proj); err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}
	agent := makeTestAgent("agent-indep", "IndepAgent")
	if err := db.UpsertAgent(agent); err != nil {
		t.Fatalf("UpsertAgent failed: %v", err)
	}

	// Delete all agents; providers and projects should remain
	if err := db.DeleteAllAgents(); err != nil {
		t.Fatalf("DeleteAllAgents failed: %v", err)
	}
	agents, _ := db.ListAgents()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents after DeleteAllAgents, got %d", len(agents))
	}

	providers, _ := db.ListProviders()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider after DeleteAllAgents, got %d", len(providers))
	}

	projects, _ := db.ListProjects()
	if len(projects) != 1 {
		t.Errorf("Expected 1 project after DeleteAllAgents, got %d", len(projects))
	}
}

func TestMultipleNewDatabases_Isolated(t *testing.T) {
	db1 := newTestDB(t)
	db2 := newTestDB(t)

	// Insert in db1 only
	if err := db1.SetConfigValue("isolation", "db1"); err != nil {
		t.Fatalf("SetConfigValue on db1 failed: %v", err)
	}

	// db2 should not see it
	_, found, err := db2.GetConfigValue("isolation")
	if err != nil {
		t.Fatalf("GetConfigValue on db2 failed: %v", err)
	}
	if found {
		t.Error("db2 should not find key from db1")
	}
}

// ---------------------------------------------------------------------------
// 8. Comments: CreateComment, GetCommentsByBeadID, GetComment, UpdateComment,
//    DeleteComment, CreateMention, GetMentionsByComment, MarkMentionNotified
// ---------------------------------------------------------------------------

func makeTestComment(id, beadID string) *BeadComment {
	now := time.Now()
	return &BeadComment{
		ID:             id,
		BeadID:         beadID,
		AuthorID:       "user-1",
		AuthorUsername: "testuser",
		Content:        "This is a test comment",
		CreatedAt:      now,
		UpdatedAt:      now,
		Edited:         false,
		Deleted:        false,
	}
}

func TestCreateComment_AndGetByID(t *testing.T) {
	db := newTestDB(t)
	c := makeTestComment("comment-1", "bead-1")

	if err := db.CreateComment(c); err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}

	got, err := db.GetComment("comment-1")
	if err != nil {
		t.Fatalf("GetComment failed: %v", err)
	}
	if got.ID != "comment-1" {
		t.Errorf("ID = %q, want %q", got.ID, "comment-1")
	}
	if got.BeadID != "bead-1" {
		t.Errorf("BeadID = %q, want %q", got.BeadID, "bead-1")
	}
	if got.AuthorID != "user-1" {
		t.Errorf("AuthorID = %q, want %q", got.AuthorID, "user-1")
	}
	if got.AuthorUsername != "testuser" {
		t.Errorf("AuthorUsername = %q, want %q", got.AuthorUsername, "testuser")
	}
	if got.Content != "This is a test comment" {
		t.Errorf("Content = %q, want %q", got.Content, "This is a test comment")
	}
	if got.Edited {
		t.Error("Expected Edited = false")
	}
	if got.Deleted {
		t.Error("Expected Deleted = false")
	}
}

func TestCreateComment_WithParentID(t *testing.T) {
	db := newTestDB(t)

	parent := makeTestComment("comment-parent", "bead-1")
	if err := db.CreateComment(parent); err != nil {
		t.Fatalf("CreateComment (parent) failed: %v", err)
	}

	reply := makeTestComment("comment-reply", "bead-1")
	reply.ParentID = "comment-parent"
	if err := db.CreateComment(reply); err != nil {
		t.Fatalf("CreateComment (reply) failed: %v", err)
	}

	got, err := db.GetComment("comment-reply")
	if err != nil {
		t.Fatalf("GetComment failed: %v", err)
	}
	if got.ParentID != "comment-parent" {
		t.Errorf("ParentID = %q, want %q", got.ParentID, "comment-parent")
	}
}

func TestGetComment_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.GetComment("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent comment, got nil")
	}
}

func TestGetCommentsByBeadID(t *testing.T) {
	db := newTestDB(t)

	c1 := makeTestComment("comment-b1", "bead-list")
	c1.Content = "First comment"
	c2 := makeTestComment("comment-b2", "bead-list")
	c2.Content = "Second comment"
	c3 := makeTestComment("comment-other", "bead-other")
	c3.Content = "Other bead"

	for _, c := range []*BeadComment{c1, c2, c3} {
		if err := db.CreateComment(c); err != nil {
			t.Fatalf("CreateComment failed: %v", err)
		}
	}

	comments, err := db.GetCommentsByBeadID("bead-list")
	if err != nil {
		t.Fatalf("GetCommentsByBeadID failed: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("Expected 2 comments for bead-list, got %d", len(comments))
	}
}

func TestGetCommentsByBeadID_Empty(t *testing.T) {
	db := newTestDB(t)
	comments, err := db.GetCommentsByBeadID("no-such-bead")
	if err != nil {
		t.Fatalf("GetCommentsByBeadID failed: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("Expected 0 comments, got %d", len(comments))
	}
}

func TestGetCommentsByBeadID_ExcludesDeleted(t *testing.T) {
	db := newTestDB(t)

	c := makeTestComment("comment-del-filter", "bead-del-filter")
	if err := db.CreateComment(c); err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}
	if err := db.DeleteComment("comment-del-filter"); err != nil {
		t.Fatalf("DeleteComment failed: %v", err)
	}

	comments, err := db.GetCommentsByBeadID("bead-del-filter")
	if err != nil {
		t.Fatalf("GetCommentsByBeadID failed: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("Expected 0 comments (deleted), got %d", len(comments))
	}
}

func TestUpdateComment(t *testing.T) {
	db := newTestDB(t)
	c := makeTestComment("comment-upd", "bead-upd")
	if err := db.CreateComment(c); err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}

	if err := db.UpdateComment("comment-upd", "Updated content"); err != nil {
		t.Fatalf("UpdateComment failed: %v", err)
	}

	got, err := db.GetComment("comment-upd")
	if err != nil {
		t.Fatalf("GetComment failed: %v", err)
	}
	if got.Content != "Updated content" {
		t.Errorf("Content = %q, want %q", got.Content, "Updated content")
	}
	if !got.Edited {
		t.Error("Expected Edited = true after update")
	}
}

func TestUpdateComment_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.UpdateComment("nonexistent", "content")
	if err == nil {
		t.Fatal("Expected error when updating non-existent comment, got nil")
	}
}

func TestDeleteComment_SoftDelete(t *testing.T) {
	db := newTestDB(t)
	c := makeTestComment("comment-sdel", "bead-sdel")
	if err := db.CreateComment(c); err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}

	if err := db.DeleteComment("comment-sdel"); err != nil {
		t.Fatalf("DeleteComment failed: %v", err)
	}

	// GetComment still finds it (soft delete), but Deleted=true
	got, err := db.GetComment("comment-sdel")
	if err != nil {
		t.Fatalf("GetComment failed: %v", err)
	}
	if !got.Deleted {
		t.Error("Expected Deleted = true after soft delete")
	}
}

func TestDeleteComment_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.DeleteComment("nonexistent")
	if err == nil {
		t.Fatal("Expected error when deleting non-existent comment, got nil")
	}
}

func TestCreateMention_AndGet(t *testing.T) {
	db := newTestDB(t)
	ensureUserExists(t, db, "user-2", "otheruser")

	c := makeTestComment("comment-mention", "bead-mention")
	if err := db.CreateComment(c); err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}

	mention := &CommentMention{
		ID:                "mention-1",
		CommentID:         "comment-mention",
		MentionedUserID:   "user-2",
		MentionedUsername: "otheruser",
		CreatedAt:         time.Now(),
	}
	if err := db.CreateMention(mention); err != nil {
		t.Fatalf("CreateMention failed: %v", err)
	}

	mentions, err := db.GetMentionsByComment("comment-mention")
	if err != nil {
		t.Fatalf("GetMentionsByComment failed: %v", err)
	}
	if len(mentions) != 1 {
		t.Fatalf("Expected 1 mention, got %d", len(mentions))
	}
	if mentions[0].MentionedUserID != "user-2" {
		t.Errorf("MentionedUserID = %q, want %q", mentions[0].MentionedUserID, "user-2")
	}
	if mentions[0].MentionedUsername != "otheruser" {
		t.Errorf("MentionedUsername = %q, want %q", mentions[0].MentionedUsername, "otheruser")
	}
	if mentions[0].NotifiedAt != nil {
		t.Error("Expected NotifiedAt to be nil initially")
	}
}

func TestGetMentionsByComment_Empty(t *testing.T) {
	db := newTestDB(t)
	mentions, err := db.GetMentionsByComment("no-such-comment")
	if err != nil {
		t.Fatalf("GetMentionsByComment failed: %v", err)
	}
	if len(mentions) != 0 {
		t.Errorf("Expected 0 mentions, got %d", len(mentions))
	}
}

func TestMarkMentionNotified(t *testing.T) {
	db := newTestDB(t)
	ensureUserExists(t, db, "user-3", "user3")

	c := makeTestComment("comment-mn", "bead-mn")
	if err := db.CreateComment(c); err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}

	mention := &CommentMention{
		ID:                "mention-notif",
		CommentID:         "comment-mn",
		MentionedUserID:   "user-3",
		MentionedUsername: "user3",
		CreatedAt:         time.Now(),
	}
	if err := db.CreateMention(mention); err != nil {
		t.Fatalf("CreateMention failed: %v", err)
	}

	if err := db.MarkMentionNotified("mention-notif"); err != nil {
		t.Fatalf("MarkMentionNotified failed: %v", err)
	}

	mentions, err := db.GetMentionsByComment("comment-mn")
	if err != nil {
		t.Fatalf("GetMentionsByComment failed: %v", err)
	}
	if len(mentions) != 1 {
		t.Fatalf("Expected 1 mention, got %d", len(mentions))
	}
	if mentions[0].NotifiedAt == nil {
		t.Error("Expected NotifiedAt to be set after MarkMentionNotified")
	}
}

// ---------------------------------------------------------------------------
// 9. Activity: CreateActivity, ListActivities, GetRecentAggregatableActivity,
//    UpdateAggregatedActivity
// ---------------------------------------------------------------------------

func makeTestActivity(id string) *Activity {
	return &Activity{
		ID:               id,
		EventType:        "bead.created",
		Timestamp:        time.Now(),
		Source:           "test",
		Action:           "create",
		ResourceType:     "bead",
		ResourceID:       "bead-act-1",
		AggregationCount: 1,
		Visibility:       "project",
	}
}

func TestCreateActivity_AndList(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-act")
	a := makeTestActivity("act-1")
	a.ProjectID = "proj-act"
	a.ActorID = "agent-act"
	a.ActorType = "agent"
	a.ResourceTitle = "Test Bead"

	if err := db.CreateActivity(a); err != nil {
		t.Fatalf("CreateActivity failed: %v", err)
	}

	activities, err := db.ListActivities(ActivityFilters{Limit: 10})
	if err != nil {
		t.Fatalf("ListActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity, got %d", len(activities))
	}
	if activities[0].ID != "act-1" {
		t.Errorf("ID = %q, want %q", activities[0].ID, "act-1")
	}
	if activities[0].EventType != "bead.created" {
		t.Errorf("EventType = %q, want %q", activities[0].EventType, "bead.created")
	}
	if activities[0].ProjectID != "proj-act" {
		t.Errorf("ProjectID = %q, want %q", activities[0].ProjectID, "proj-act")
	}
	if activities[0].ActorID != "agent-act" {
		t.Errorf("ActorID = %q, want %q", activities[0].ActorID, "agent-act")
	}
	if activities[0].ResourceTitle != "Test Bead" {
		t.Errorf("ResourceTitle = %q, want %q", activities[0].ResourceTitle, "Test Bead")
	}
}

func TestListActivities_Empty(t *testing.T) {
	db := newTestDB(t)
	activities, err := db.ListActivities(ActivityFilters{})
	if err != nil {
		t.Fatalf("ListActivities failed: %v", err)
	}
	if len(activities) != 0 {
		t.Errorf("Expected 0 activities, got %d", len(activities))
	}
}

func TestListActivities_Filters(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-filter")
	ensureProjectExists(t, db, "proj-other")

	a1 := makeTestActivity("act-f1")
	a1.ProjectID = "proj-filter"
	a1.EventType = "bead.created"
	a1.ActorID = "user-f1"
	a1.ResourceType = "bead"

	a2 := makeTestActivity("act-f2")
	a2.ProjectID = "proj-filter"
	a2.EventType = "agent.started"
	a2.ActorID = "user-f2"
	a2.ResourceType = "agent"

	a3 := makeTestActivity("act-f3")
	a3.ProjectID = "proj-other"
	a3.EventType = "bead.created"

	for _, a := range []*Activity{a1, a2, a3} {
		if err := db.CreateActivity(a); err != nil {
			t.Fatalf("CreateActivity failed: %v", err)
		}
	}

	// Filter by event type
	activities, err := db.ListActivities(ActivityFilters{EventType: "bead.created"})
	if err != nil {
		t.Fatalf("ListActivities (event type filter) failed: %v", err)
	}
	if len(activities) != 2 {
		t.Errorf("Expected 2 activities with event type bead.created, got %d", len(activities))
	}

	// Filter by actor
	activities, err = db.ListActivities(ActivityFilters{ActorID: "user-f1"})
	if err != nil {
		t.Fatalf("ListActivities (actor filter) failed: %v", err)
	}
	if len(activities) != 1 {
		t.Errorf("Expected 1 activity for user-f1, got %d", len(activities))
	}

	// Filter by resource type
	activities, err = db.ListActivities(ActivityFilters{ResourceType: "agent"})
	if err != nil {
		t.Fatalf("ListActivities (resource type filter) failed: %v", err)
	}
	if len(activities) != 1 {
		t.Errorf("Expected 1 activity with resource type agent, got %d", len(activities))
	}

	// Filter by project IDs
	activities, err = db.ListActivities(ActivityFilters{ProjectIDs: []string{"proj-filter"}})
	if err != nil {
		t.Fatalf("ListActivities (project filter) failed: %v", err)
	}
	if len(activities) != 2 {
		t.Errorf("Expected 2 activities for proj-filter, got %d", len(activities))
	}

	// Filter by limit and offset
	activities, err = db.ListActivities(ActivityFilters{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("ListActivities (limit/offset) failed: %v", err)
	}
	if len(activities) != 1 {
		t.Errorf("Expected 1 activity with limit=1 offset=1, got %d", len(activities))
	}

	// Filter by time range
	activities, err = db.ListActivities(ActivityFilters{Since: time.Now().Add(-1 * time.Hour), Until: time.Now().Add(1 * time.Hour)})
	if err != nil {
		t.Fatalf("ListActivities (time range) failed: %v", err)
	}
	if len(activities) != 3 {
		t.Errorf("Expected 3 activities in time range, got %d", len(activities))
	}

	// Filter by aggregated
	agg := true
	activities, err = db.ListActivities(ActivityFilters{Aggregated: &agg})
	if err != nil {
		t.Fatalf("ListActivities (aggregated filter) failed: %v", err)
	}
	// All test activities have IsAggregated=false
	if len(activities) != 0 {
		t.Errorf("Expected 0 aggregated activities, got %d", len(activities))
	}
}

func TestGetRecentAggregatableActivity(t *testing.T) {
	db := newTestDB(t)

	a := makeTestActivity("act-agg")
	a.AggregationKey = "bead.proj-1"
	a.IsAggregated = true
	a.AggregationCount = 5
	if err := db.CreateActivity(a); err != nil {
		t.Fatalf("CreateActivity failed: %v", err)
	}

	got, err := db.GetRecentAggregatableActivity("bead.proj-1", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("GetRecentAggregatableActivity failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected non-nil activity")
	}
	if got.AggregationCount != 5 {
		t.Errorf("AggregationCount = %d, want 5", got.AggregationCount)
	}
}

func TestGetRecentAggregatableActivity_NotFound(t *testing.T) {
	db := newTestDB(t)
	got, err := db.GetRecentAggregatableActivity("nonexistent", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("GetRecentAggregatableActivity failed: %v", err)
	}
	if got != nil {
		t.Error("Expected nil for non-existent aggregation key")
	}
}

func TestUpdateAggregatedActivity(t *testing.T) {
	db := newTestDB(t)

	a := makeTestActivity("act-upd-agg")
	a.AggregationKey = "test.key"
	a.IsAggregated = true
	a.AggregationCount = 1
	if err := db.CreateActivity(a); err != nil {
		t.Fatalf("CreateActivity failed: %v", err)
	}

	if err := db.UpdateAggregatedActivity("act-upd-agg", 10); err != nil {
		t.Fatalf("UpdateAggregatedActivity failed: %v", err)
	}

	got, err := db.GetRecentAggregatableActivity("test.key", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("GetRecentAggregatableActivity failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected non-nil activity")
	}
	if got.AggregationCount != 10 {
		t.Errorf("AggregationCount = %d, want 10", got.AggregationCount)
	}
}

// ---------------------------------------------------------------------------
// 10. Notifications: CreateNotification, ListNotifications,
//     MarkNotificationRead, MarkAllNotificationsRead
// ---------------------------------------------------------------------------

func TestCreateNotification_AndList(t *testing.T) {
	db := newTestDB(t)
	ensureUserExists(t, db, "user-1", "user1")

	notif := &Notification{
		ID:        "notif-1",
		UserID:    "user-1",
		EventType: "bead.created",
		Title:     "New Bead",
		Message:   "A new bead was created",
		Status:    "unread",
		Priority:  "normal",
		CreatedAt: time.Now(),
	}
	if err := db.CreateNotification(notif); err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	notifications, err := db.ListNotifications("user-1", "", 10, 0)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}
	if notifications[0].Title != "New Bead" {
		t.Errorf("Title = %q, want %q", notifications[0].Title, "New Bead")
	}
	if notifications[0].Status != "unread" {
		t.Errorf("Status = %q, want %q", notifications[0].Status, "unread")
	}
}

func TestListNotifications_FilterByStatus(t *testing.T) {
	db := newTestDB(t)
	ensureUserExists(t, db, "user-s", "user_s")

	n1 := &Notification{
		ID: "notif-s1", UserID: "user-s", EventType: "e", Title: "T1",
		Message: "M1", Status: "unread", Priority: "normal", CreatedAt: time.Now(),
	}
	n2 := &Notification{
		ID: "notif-s2", UserID: "user-s", EventType: "e", Title: "T2",
		Message: "M2", Status: "read", Priority: "normal", CreatedAt: time.Now(),
	}
	for _, n := range []*Notification{n1, n2} {
		if err := db.CreateNotification(n); err != nil {
			t.Fatalf("CreateNotification failed: %v", err)
		}
	}

	unread, err := db.ListNotifications("user-s", "unread", 10, 0)
	if err != nil {
		t.Fatalf("ListNotifications (unread) failed: %v", err)
	}
	if len(unread) != 1 {
		t.Errorf("Expected 1 unread notification, got %d", len(unread))
	}
}

func TestListNotifications_LimitAndOffset(t *testing.T) {
	db := newTestDB(t)
	ensureUserExists(t, db, "user-lo", "user_lo")
	for i := 0; i < 5; i++ {
		n := &Notification{
			ID: "notif-lo-" + string(rune('0'+i)), UserID: "user-lo", EventType: "e",
			Title: "T", Message: "M", Status: "unread", Priority: "normal", CreatedAt: time.Now(),
		}
		if err := db.CreateNotification(n); err != nil {
			t.Fatalf("CreateNotification failed: %v", err)
		}
	}

	notifs, err := db.ListNotifications("user-lo", "", 2, 1)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(notifs) != 2 {
		t.Errorf("Expected 2 notifications with limit=2 offset=1, got %d", len(notifs))
	}
}

func TestMarkNotificationRead(t *testing.T) {
	db := newTestDB(t)
	ensureUserExists(t, db, "user-mr", "user_mr")
	n := &Notification{
		ID: "notif-mr", UserID: "user-mr", EventType: "e", Title: "T",
		Message: "M", Status: "unread", Priority: "normal", CreatedAt: time.Now(),
	}
	if err := db.CreateNotification(n); err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	if err := db.MarkNotificationRead("notif-mr"); err != nil {
		t.Fatalf("MarkNotificationRead failed: %v", err)
	}

	notifs, err := db.ListNotifications("user-mr", "read", 10, 0)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(notifs) != 1 {
		t.Fatalf("Expected 1 read notification, got %d", len(notifs))
	}
	if notifs[0].ReadAt == nil {
		t.Error("Expected ReadAt to be set")
	}
}

func TestMarkAllNotificationsRead(t *testing.T) {
	db := newTestDB(t)
	ensureUserExists(t, db, "user-mar", "user_mar")
	for i := 0; i < 3; i++ {
		n := &Notification{
			ID: "notif-mar-" + string(rune('0'+i)), UserID: "user-mar", EventType: "e",
			Title: "T", Message: "M", Status: "unread", Priority: "normal", CreatedAt: time.Now(),
		}
		if err := db.CreateNotification(n); err != nil {
			t.Fatalf("CreateNotification failed: %v", err)
		}
	}

	if err := db.MarkAllNotificationsRead("user-mar"); err != nil {
		t.Fatalf("MarkAllNotificationsRead failed: %v", err)
	}

	unread, err := db.ListNotifications("user-mar", "unread", 10, 0)
	if err != nil {
		t.Fatalf("ListNotifications (unread) failed: %v", err)
	}
	if len(unread) != 0 {
		t.Errorf("Expected 0 unread notifications, got %d", len(unread))
	}

	read, err := db.ListNotifications("user-mar", "read", 10, 0)
	if err != nil {
		t.Fatalf("ListNotifications (read) failed: %v", err)
	}
	if len(read) != 3 {
		t.Errorf("Expected 3 read notifications, got %d", len(read))
	}
}

// ---------------------------------------------------------------------------
// 11. Notification Preferences
// ---------------------------------------------------------------------------

func TestNotificationPreferences_UpsertAndGet(t *testing.T) {
	db := newTestDB(t)
	ensureUserExists(t, db, "user-pref", "user_pref")

	prefs := &NotificationPreferences{
		ID:          "pref-1",
		UserID:      "user-pref",
		EnableInApp: true,
		DigestMode:  "daily",
		MinPriority: "normal",
		UpdatedAt:   time.Now(),
	}

	if err := db.UpsertNotificationPreferences(prefs); err != nil {
		t.Fatalf("UpsertNotificationPreferences failed: %v", err)
	}

	got, err := db.GetNotificationPreferences("user-pref")
	if err != nil {
		t.Fatalf("GetNotificationPreferences failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected non-nil preferences")
	}
	if !got.EnableInApp {
		t.Error("Expected EnableInApp = true")
	}
	if got.DigestMode != "daily" {
		t.Errorf("DigestMode = %q, want %q", got.DigestMode, "daily")
	}
}

func TestNotificationPreferences_NotFound(t *testing.T) {
	db := newTestDB(t)
	got, err := db.GetNotificationPreferences("nonexistent-user")
	if err != nil {
		t.Fatalf("GetNotificationPreferences failed: %v", err)
	}
	if got != nil {
		t.Error("Expected nil preferences for non-existent user")
	}
}

func TestNotificationPreferences_Update(t *testing.T) {
	db := newTestDB(t)
	ensureUserExists(t, db, "user-pref-upd", "user_pref_upd")

	prefs := &NotificationPreferences{
		ID:          "pref-upd",
		UserID:      "user-pref-upd",
		EnableInApp: true,
		DigestMode:  "daily",
		MinPriority: "normal",
		UpdatedAt:   time.Now(),
	}
	if err := db.UpsertNotificationPreferences(prefs); err != nil {
		t.Fatalf("UpsertNotificationPreferences (create) failed: %v", err)
	}

	prefs.EnableEmail = true
	prefs.DigestMode = "weekly"
	if err := db.UpsertNotificationPreferences(prefs); err != nil {
		t.Fatalf("UpsertNotificationPreferences (update) failed: %v", err)
	}

	got, err := db.GetNotificationPreferences("user-pref-upd")
	if err != nil {
		t.Fatalf("GetNotificationPreferences failed: %v", err)
	}
	if !got.EnableEmail {
		t.Error("Expected EnableEmail = true")
	}
	if got.DigestMode != "weekly" {
		t.Errorf("DigestMode = %q, want %q", got.DigestMode, "weekly")
	}
}

// ---------------------------------------------------------------------------
// 12. Users: CreateUser, ListUsers
// ---------------------------------------------------------------------------

func TestCreateUser_AndList(t *testing.T) {
	db := newTestDB(t)

	if err := db.CreateUser("user-cu-1", "alice", "alice@test.com", "admin"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if err := db.CreateUser("user-cu-2", "bob", "bob@test.com", "member"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	users, err := db.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	// There is already a default admin user created during migration
	if len(users) < 2 {
		t.Errorf("Expected at least 2 users, got %d", len(users))
	}

	foundAlice := false
	foundBob := false
	for _, u := range users {
		if u.Username == "alice" {
			foundAlice = true
			if u.Role != "admin" {
				t.Errorf("alice role = %q, want %q", u.Role, "admin")
			}
		}
		if u.Username == "bob" {
			foundBob = true
		}
	}
	if !foundAlice {
		t.Error("Expected to find user alice")
	}
	if !foundBob {
		t.Error("Expected to find user bob")
	}
}

// ---------------------------------------------------------------------------
// 13. Workflows: UpsertWorkflow, GetWorkflow, ListWorkflows, nodes, edges,
//     executions, history
// ---------------------------------------------------------------------------

func TestWorkflow_CRUD(t *testing.T) {
	db := newTestDB(t)

	wf := &workflow.Workflow{
		ID:           "wf-1",
		Name:         "Bug Fix",
		Description:  "Workflow for bug fixes",
		WorkflowType: "bug",
		IsDefault:    true,
	}
	if err := db.UpsertWorkflow(wf); err != nil {
		t.Fatalf("UpsertWorkflow failed: %v", err)
	}
	if wf.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	got, err := db.GetWorkflow("wf-1")
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}
	if got.Name != "Bug Fix" {
		t.Errorf("Name = %q, want %q", got.Name, "Bug Fix")
	}
	if got.WorkflowType != "bug" {
		t.Errorf("WorkflowType = %q, want %q", got.WorkflowType, "bug")
	}
	if !got.IsDefault {
		t.Error("Expected IsDefault = true")
	}
}

func TestUpsertWorkflow_Nil(t *testing.T) {
	db := newTestDB(t)
	err := db.UpsertWorkflow(nil)
	if err == nil {
		t.Fatal("Expected error for nil workflow, got nil")
	}
}

func TestWorkflow_Update(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-wf")

	wf := &workflow.Workflow{
		ID:           "wf-upd",
		Name:         "Original",
		WorkflowType: "feature",
	}
	if err := db.UpsertWorkflow(wf); err != nil {
		t.Fatalf("UpsertWorkflow (create) failed: %v", err)
	}

	wf.Name = "Updated"
	wf.Description = "Updated desc"
	wf.ProjectID = "proj-wf"
	if err := db.UpsertWorkflow(wf); err != nil {
		t.Fatalf("UpsertWorkflow (update) failed: %v", err)
	}

	got, err := db.GetWorkflow("wf-upd")
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}
	if got.Name != "Updated" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated")
	}
	if got.ProjectID != "proj-wf" {
		t.Errorf("ProjectID = %q, want %q", got.ProjectID, "proj-wf")
	}
}

func TestGetWorkflow_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.GetWorkflow("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent workflow, got nil")
	}
}

func TestListWorkflows(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-wfl")

	wf1 := &workflow.Workflow{ID: "wf-l1", Name: "WF1", WorkflowType: "bug", IsDefault: true}
	wf2 := &workflow.Workflow{ID: "wf-l2", Name: "WF2", WorkflowType: "feature"}
	wf3 := &workflow.Workflow{ID: "wf-l3", Name: "WF3", WorkflowType: "bug", ProjectID: "proj-wfl"}

	for _, wf := range []*workflow.Workflow{wf1, wf2, wf3} {
		if err := db.UpsertWorkflow(wf); err != nil {
			t.Fatalf("UpsertWorkflow failed: %v", err)
		}
	}

	// List all
	all, err := db.ListWorkflows("", "")
	if err != nil {
		t.Fatalf("ListWorkflows (all) failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 workflows, got %d", len(all))
	}

	// Filter by type
	bugs, err := db.ListWorkflows("bug", "")
	if err != nil {
		t.Fatalf("ListWorkflows (bug) failed: %v", err)
	}
	if len(bugs) != 2 {
		t.Errorf("Expected 2 bug workflows, got %d", len(bugs))
	}

	// Filter by project (includes global ones where project_id IS NULL)
	projWfs, err := db.ListWorkflows("", "proj-wfl")
	if err != nil {
		t.Fatalf("ListWorkflows (project) failed: %v", err)
	}
	if len(projWfs) != 3 {
		t.Errorf("Expected 3 workflows for project (2 global + 1 project), got %d", len(projWfs))
	}
}

func TestWorkflowNode_CRUD(t *testing.T) {
	db := newTestDB(t)

	wf := &workflow.Workflow{ID: "wf-node", Name: "NodeTest", WorkflowType: "custom"}
	if err := db.UpsertWorkflow(wf); err != nil {
		t.Fatalf("UpsertWorkflow failed: %v", err)
	}

	node := &workflow.WorkflowNode{
		ID:             "node-1",
		WorkflowID:     "wf-node",
		NodeKey:        "investigate",
		NodeType:       workflow.NodeTypeTask,
		RoleRequired:   "developer",
		PersonaHint:    "senior-dev",
		MaxAttempts:    3,
		TimeoutMinutes: 30,
		Instructions:   "Investigate the bug",
		Metadata:       map[string]string{"priority": "high"},
	}
	if err := db.UpsertWorkflowNode(node); err != nil {
		t.Fatalf("UpsertWorkflowNode failed: %v", err)
	}

	nodes, err := db.ListWorkflowNodes("wf-node")
	if err != nil {
		t.Fatalf("ListWorkflowNodes failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	if nodes[0].NodeKey != "investigate" {
		t.Errorf("NodeKey = %q, want %q", nodes[0].NodeKey, "investigate")
	}
	if nodes[0].NodeType != workflow.NodeTypeTask {
		t.Errorf("NodeType = %q, want %q", nodes[0].NodeType, workflow.NodeTypeTask)
	}
	if nodes[0].MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", nodes[0].MaxAttempts)
	}
	if nodes[0].Metadata["priority"] != "high" {
		t.Errorf("Metadata[priority] = %q, want %q", nodes[0].Metadata["priority"], "high")
	}
}

func TestUpsertWorkflowNode_Nil(t *testing.T) {
	db := newTestDB(t)
	err := db.UpsertWorkflowNode(nil)
	if err == nil {
		t.Fatal("Expected error for nil node, got nil")
	}
}

func TestWorkflowNode_NilMetadata(t *testing.T) {
	db := newTestDB(t)

	wf := &workflow.Workflow{ID: "wf-nmeta", Name: "NilMeta", WorkflowType: "custom"}
	if err := db.UpsertWorkflow(wf); err != nil {
		t.Fatalf("UpsertWorkflow failed: %v", err)
	}

	node := &workflow.WorkflowNode{
		ID:         "node-nmeta",
		WorkflowID: "wf-nmeta",
		NodeKey:    "step1",
		NodeType:   workflow.NodeTypeTask,
	}
	if err := db.UpsertWorkflowNode(node); err != nil {
		t.Fatalf("UpsertWorkflowNode failed: %v", err)
	}

	nodes, err := db.ListWorkflowNodes("wf-nmeta")
	if err != nil {
		t.Fatalf("ListWorkflowNodes failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Metadata == nil {
		t.Error("Expected non-nil Metadata map")
	}
}

func TestWorkflowEdge_CRUD(t *testing.T) {
	db := newTestDB(t)

	wf := &workflow.Workflow{ID: "wf-edge", Name: "EdgeTest", WorkflowType: "custom"}
	if err := db.UpsertWorkflow(wf); err != nil {
		t.Fatalf("UpsertWorkflow failed: %v", err)
	}

	edge := &workflow.WorkflowEdge{
		ID:          "edge-1",
		WorkflowID:  "wf-edge",
		FromNodeKey: "investigate",
		ToNodeKey:   "fix",
		Condition:   workflow.EdgeConditionSuccess,
		Priority:    10,
	}
	if err := db.UpsertWorkflowEdge(edge); err != nil {
		t.Fatalf("UpsertWorkflowEdge failed: %v", err)
	}

	edges, err := db.ListWorkflowEdges("wf-edge")
	if err != nil {
		t.Fatalf("ListWorkflowEdges failed: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(edges))
	}
	if edges[0].FromNodeKey != "investigate" {
		t.Errorf("FromNodeKey = %q, want %q", edges[0].FromNodeKey, "investigate")
	}
	if edges[0].ToNodeKey != "fix" {
		t.Errorf("ToNodeKey = %q, want %q", edges[0].ToNodeKey, "fix")
	}
	if edges[0].Condition != workflow.EdgeConditionSuccess {
		t.Errorf("Condition = %q, want %q", edges[0].Condition, workflow.EdgeConditionSuccess)
	}
	if edges[0].Priority != 10 {
		t.Errorf("Priority = %d, want 10", edges[0].Priority)
	}
}

func TestUpsertWorkflowEdge_Nil(t *testing.T) {
	db := newTestDB(t)
	err := db.UpsertWorkflowEdge(nil)
	if err == nil {
		t.Fatal("Expected error for nil edge, got nil")
	}
}

func TestWorkflowEdge_EmptyFromTo(t *testing.T) {
	db := newTestDB(t)

	wf := &workflow.Workflow{ID: "wf-edge-empty", Name: "EdgeEmpty", WorkflowType: "custom"}
	if err := db.UpsertWorkflow(wf); err != nil {
		t.Fatalf("UpsertWorkflow failed: %v", err)
	}

	edge := &workflow.WorkflowEdge{
		ID:         "edge-start",
		WorkflowID: "wf-edge-empty",
		Condition:  workflow.EdgeConditionSuccess,
	}
	if err := db.UpsertWorkflowEdge(edge); err != nil {
		t.Fatalf("UpsertWorkflowEdge failed: %v", err)
	}

	edges, err := db.ListWorkflowEdges("wf-edge-empty")
	if err != nil {
		t.Fatalf("ListWorkflowEdges failed: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(edges))
	}
	if edges[0].FromNodeKey != "" {
		t.Errorf("FromNodeKey = %q, want empty", edges[0].FromNodeKey)
	}
	if edges[0].ToNodeKey != "" {
		t.Errorf("ToNodeKey = %q, want empty", edges[0].ToNodeKey)
	}
}

func TestWorkflowExecution_CRUD(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-exec")

	wf := &workflow.Workflow{ID: "wf-exec", Name: "ExecTest", WorkflowType: "custom"}
	if err := db.UpsertWorkflow(wf); err != nil {
		t.Fatalf("UpsertWorkflow failed: %v", err)
	}

	exec := &workflow.WorkflowExecution{
		ID:               "exec-1",
		WorkflowID:       "wf-exec",
		BeadID:           "bead-exec-1",
		ProjectID:        "proj-exec",
		CurrentNodeKey:   "investigate",
		Status:           workflow.ExecutionStatusActive,
		CycleCount:       0,
		NodeAttemptCount: 1,
	}
	if err := db.UpsertWorkflowExecution(exec); err != nil {
		t.Fatalf("UpsertWorkflowExecution failed: %v", err)
	}

	got, err := db.GetWorkflowExecution("exec-1")
	if err != nil {
		t.Fatalf("GetWorkflowExecution failed: %v", err)
	}
	if got.WorkflowID != "wf-exec" {
		t.Errorf("WorkflowID = %q, want %q", got.WorkflowID, "wf-exec")
	}
	if got.CurrentNodeKey != "investigate" {
		t.Errorf("CurrentNodeKey = %q, want %q", got.CurrentNodeKey, "investigate")
	}
	if got.Status != workflow.ExecutionStatusActive {
		t.Errorf("Status = %q, want %q", got.Status, workflow.ExecutionStatusActive)
	}

	byBead, err := db.GetWorkflowExecutionByBeadID("bead-exec-1")
	if err != nil {
		t.Fatalf("GetWorkflowExecutionByBeadID failed: %v", err)
	}
	if byBead == nil {
		t.Fatal("Expected non-nil execution by bead ID")
	}
	if byBead.ID != "exec-1" {
		t.Errorf("ID = %q, want %q", byBead.ID, "exec-1")
	}
}

func TestUpsertWorkflowExecution_Nil(t *testing.T) {
	db := newTestDB(t)
	err := db.UpsertWorkflowExecution(nil)
	if err == nil {
		t.Fatal("Expected error for nil execution, got nil")
	}
}

func TestGetWorkflowExecution_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.GetWorkflowExecution("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent execution, got nil")
	}
}

func TestGetWorkflowExecutionByBeadID_NotFound(t *testing.T) {
	db := newTestDB(t)
	got, err := db.GetWorkflowExecutionByBeadID("nonexistent")
	if err != nil {
		t.Fatalf("GetWorkflowExecutionByBeadID should not error: %v", err)
	}
	if got != nil {
		t.Error("Expected nil for non-existent bead ID")
	}
}

func TestWorkflowExecution_UpdateViaUpsert(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-exec-upd")

	wf := &workflow.Workflow{ID: "wf-exec-upd", Name: "ExecUpd", WorkflowType: "custom"}
	if err := db.UpsertWorkflow(wf); err != nil {
		t.Fatalf("UpsertWorkflow failed: %v", err)
	}

	exec := &workflow.WorkflowExecution{
		ID:             "exec-upd",
		WorkflowID:     "wf-exec-upd",
		BeadID:         "bead-exec-upd",
		ProjectID:      "proj-exec-upd",
		CurrentNodeKey: "step1",
		Status:         workflow.ExecutionStatusActive,
	}
	if err := db.UpsertWorkflowExecution(exec); err != nil {
		t.Fatalf("UpsertWorkflowExecution (create) failed: %v", err)
	}

	exec.CurrentNodeKey = "step2"
	exec.Status = workflow.ExecutionStatusCompleted
	now := time.Now()
	exec.CompletedAt = &now
	if err := db.UpsertWorkflowExecution(exec); err != nil {
		t.Fatalf("UpsertWorkflowExecution (update) failed: %v", err)
	}

	got, err := db.GetWorkflowExecution("exec-upd")
	if err != nil {
		t.Fatalf("GetWorkflowExecution failed: %v", err)
	}
	if got.CurrentNodeKey != "step2" {
		t.Errorf("CurrentNodeKey = %q, want %q", got.CurrentNodeKey, "step2")
	}
	if got.Status != workflow.ExecutionStatusCompleted {
		t.Errorf("Status = %q, want %q", got.Status, workflow.ExecutionStatusCompleted)
	}
	if got.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestWorkflowHistory_CRUD(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-hist")

	wf := &workflow.Workflow{ID: "wf-hist", Name: "HistTest", WorkflowType: "custom"}
	if err := db.UpsertWorkflow(wf); err != nil {
		t.Fatalf("UpsertWorkflow failed: %v", err)
	}

	exec := &workflow.WorkflowExecution{
		ID:         "exec-hist",
		WorkflowID: "wf-hist",
		BeadID:     "bead-hist",
		ProjectID:  "proj-hist",
		Status:     workflow.ExecutionStatusActive,
	}
	if err := db.UpsertWorkflowExecution(exec); err != nil {
		t.Fatalf("UpsertWorkflowExecution failed: %v", err)
	}

	h1 := &workflow.WorkflowExecutionHistory{
		ID:            "hist-1",
		ExecutionID:   "exec-hist",
		NodeKey:       "investigate",
		AgentID:       "agent-hist-1",
		Condition:     workflow.EdgeConditionSuccess,
		ResultData:    `{"status":"ok"}`,
		AttemptNumber: 1,
	}
	h2 := &workflow.WorkflowExecutionHistory{
		ID:            "hist-2",
		ExecutionID:   "exec-hist",
		NodeKey:       "fix",
		AgentID:       "agent-hist-2",
		Condition:     workflow.EdgeConditionSuccess,
		AttemptNumber: 1,
	}

	if err := db.InsertWorkflowHistory(h1); err != nil {
		t.Fatalf("InsertWorkflowHistory failed: %v", err)
	}
	if err := db.InsertWorkflowHistory(h2); err != nil {
		t.Fatalf("InsertWorkflowHistory failed: %v", err)
	}

	history, err := db.ListWorkflowHistory("exec-hist")
	if err != nil {
		t.Fatalf("ListWorkflowHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("Expected 2 history entries, got %d", len(history))
	}
	if history[0].NodeKey != "investigate" {
		t.Errorf("First history NodeKey = %q, want %q", history[0].NodeKey, "investigate")
	}
	if history[0].ResultData != `{"status":"ok"}` {
		t.Errorf("ResultData = %q, want %q", history[0].ResultData, `{"status":"ok"}`)
	}
	if history[1].NodeKey != "fix" {
		t.Errorf("Second history NodeKey = %q, want %q", history[1].NodeKey, "fix")
	}
}

func TestInsertWorkflowHistory_Nil(t *testing.T) {
	db := newTestDB(t)
	err := db.InsertWorkflowHistory(nil)
	if err == nil {
		t.Fatal("Expected error for nil history, got nil")
	}
}

func TestListWorkflowHistory_Empty(t *testing.T) {
	db := newTestDB(t)
	history, err := db.ListWorkflowHistory("nonexistent-exec")
	if err != nil {
		t.Fatalf("ListWorkflowHistory failed: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("Expected 0 history entries, got %d", len(history))
	}
}

// ---------------------------------------------------------------------------
// 14. Distributed (no-HA early returns for SQLite)
// ---------------------------------------------------------------------------

func TestDistributed_NoHA(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// AcquireLock should fail because SQLite does not support HA
	_, err := db.AcquireLock(ctx, "test-lock", 10*time.Second)
	if err == nil {
		t.Fatal("Expected error for AcquireLock on non-HA database, got nil")
	}

	// RegisterInstance should return empty string, nil
	instanceID, err := db.RegisterInstance(ctx, "host1", nil)
	if err != nil {
		t.Fatalf("RegisterInstance failed: %v", err)
	}
	if instanceID != "" {
		t.Errorf("Expected empty instanceID, got %q", instanceID)
	}

	// HeartbeatInstance should be no-op
	if err := db.HeartbeatInstance(ctx, "any-id"); err != nil {
		t.Fatalf("HeartbeatInstance failed: %v", err)
	}

	// UnregisterInstance should be no-op
	if err := db.UnregisterInstance(ctx, "any-id"); err != nil {
		t.Fatalf("UnregisterInstance failed: %v", err)
	}

	// ListActiveInstances should return nil, nil
	instances, err := db.ListActiveInstances(ctx)
	if err != nil {
		t.Fatalf("ListActiveInstances failed: %v", err)
	}
	if instances != nil {
		t.Errorf("Expected nil instances, got %v", instances)
	}

	// CleanupExpiredLocks should return 0, nil
	count, err := db.CleanupExpiredLocks(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredLocks failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}

	// CleanupStaleInstances should return 0, nil
	count, err = db.CleanupStaleInstances(ctx, time.Minute)
	if err != nil {
		t.Fatalf("CleanupStaleInstances failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}
}

func TestWithTransaction_Success(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Use a transaction to insert a config value
	err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO config_kv (key, value, updated_at) VALUES (?, ?, ?)",
			"tx-key", "tx-value", time.Now())
		return err
	})
	if err != nil {
		t.Fatalf("WithTransaction failed: %v", err)
	}

	// Verify the value was committed
	val, found, err := db.GetConfigValue("tx-key")
	if err != nil {
		t.Fatalf("GetConfigValue failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find tx-key after committed transaction")
	}
	if val != "tx-value" {
		t.Errorf("Value = %q, want %q", val, "tx-value")
	}
}

func TestWithTransaction_Rollback(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Use a transaction that returns an error -- should rollback
	err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO config_kv (key, value, updated_at) VALUES (?, ?, ?)",
			"rollback-key", "rollback-value", time.Now())
		if err != nil {
			return err
		}
		return fmt.Errorf("intentional error to trigger rollback")
	})
	if err == nil {
		t.Fatal("Expected error from WithTransaction, got nil")
	}

	// Verify the value was NOT committed
	_, found, err := db.GetConfigValue("rollback-key")
	if err != nil {
		t.Fatalf("GetConfigValue failed: %v", err)
	}
	if found {
		t.Error("Expected rollback-key to NOT be found after rolled-back transaction")
	}
}

// ---------------------------------------------------------------------------
// 15. Lessons: CreateLesson, GetLessonsForProject
// ---------------------------------------------------------------------------

func TestCreateLesson_AndGetForProject(t *testing.T) {
	db := newTestDB(t)

	lesson := &models.Lesson{
		ID:            "lesson-1",
		ProjectID:     "proj-lesson",
		Category:      "compiler_error",
		Title:         "Missing import",
		Detail:        "Always check imports after refactoring",
		SourceBeadID:  "bead-l1",
		SourceAgentID: "agent-l1",
	}
	if err := db.CreateLesson(lesson); err != nil {
		t.Fatalf("CreateLesson failed: %v", err)
	}
	if lesson.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if lesson.RelevanceScore != 1.0 {
		t.Errorf("RelevanceScore = %f, want 1.0", lesson.RelevanceScore)
	}

	lessons, err := db.GetLessonsForProject("proj-lesson", 10, 0)
	if err != nil {
		t.Fatalf("GetLessonsForProject failed: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("Expected 1 lesson, got %d", len(lessons))
	}
	if lessons[0].Title != "Missing import" {
		t.Errorf("Title = %q, want %q", lessons[0].Title, "Missing import")
	}
	if lessons[0].Category != "compiler_error" {
		t.Errorf("Category = %q, want %q", lessons[0].Category, "compiler_error")
	}
}

func TestCreateLesson_Nil(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateLesson(nil)
	if err == nil {
		t.Fatal("Expected error for nil lesson, got nil")
	}
}

func TestGetLessonsForProject_DefaultLimit(t *testing.T) {
	db := newTestDB(t)

	// Passing limit=0 should default to 20
	lessons, err := db.GetLessonsForProject("empty-proj", 0, 0)
	if err != nil {
		t.Fatalf("GetLessonsForProject failed: %v", err)
	}
	if len(lessons) != 0 {
		t.Errorf("Expected 0 lessons, got %d", len(lessons))
	}
}

func TestGetLessonsForProject_MaxChars(t *testing.T) {
	db := newTestDB(t)

	// Use short details so first lesson fits within maxChars but total of all 5 does not
	for i := 0; i < 5; i++ {
		lesson := &models.Lesson{
			ID:        "lesson-mc-" + string(rune('0'+i)),
			ProjectID: "proj-maxchars",
			Category:  "test",
			Title:     "Title",
			Detail:    "Short detail text.", // 18 chars each
		}
		if err := db.CreateLesson(lesson); err != nil {
			t.Fatalf("CreateLesson failed: %v", err)
		}
	}

	// maxChars=50: first 2 lessons = 36 chars (under 50), third = 54 chars (over 50, breaks)
	// So we should get 2 lessons.
	lessons, err := db.GetLessonsForProject("proj-maxchars", 10, 50)
	if err != nil {
		t.Fatalf("GetLessonsForProject failed: %v", err)
	}
	if len(lessons) >= 5 {
		t.Errorf("Expected fewer than 5 lessons with maxChars=50, got %d", len(lessons))
	}
	if len(lessons) == 0 {
		t.Error("Expected at least 1 lesson")
	}
}

// ============================================================
// 16. StoreLessonWithEmbedding + SearchLessonsBySimilarity
// ============================================================

func TestStoreLessonWithEmbedding(t *testing.T) {
	db := newTestDB(t)

	embedding := []float32{0.1, 0.2, 0.3, 0.4}
	lesson := &models.Lesson{
		ID:        "les-emb-1",
		ProjectID: "proj-emb",
		Category:  "test_failure",
		Title:     "Embedding Lesson",
		Detail:    "Details about embedding.",
	}

	err := db.StoreLessonWithEmbedding(lesson, embedding)
	if err != nil {
		t.Fatalf("StoreLessonWithEmbedding failed: %v", err)
	}

	// Verify it was stored by reading it back
	lessons, err := db.GetLessonsForProject("proj-emb", 10, 0)
	if err != nil {
		t.Fatalf("GetLessonsForProject failed: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("Expected 1 lesson, got %d", len(lessons))
	}
	if lessons[0].ID != "les-emb-1" {
		t.Errorf("ID = %q, want %q", lessons[0].ID, "les-emb-1")
	}
}

func TestStoreLessonWithEmbedding_Nil(t *testing.T) {
	db := newTestDB(t)
	err := db.StoreLessonWithEmbedding(nil, []float32{0.1})
	if err == nil {
		t.Fatal("Expected error for nil lesson, got nil")
	}
}

func TestStoreLessonWithEmbedding_DefaultValues(t *testing.T) {
	db := newTestDB(t)

	lesson := &models.Lesson{
		ID:        "les-emb-def",
		ProjectID: "proj-emb-def",
		Category:  "compiler_error",
		Title:     "Default Values",
		Detail:    "Testing defaults.",
	}

	err := db.StoreLessonWithEmbedding(lesson, nil)
	if err != nil {
		t.Fatalf("StoreLessonWithEmbedding failed: %v", err)
	}

	// RelevanceScore should have been set to 1.0
	if lesson.RelevanceScore != 1.0 {
		t.Errorf("RelevanceScore = %f, want 1.0", lesson.RelevanceScore)
	}
	// CreatedAt should have been set
	if lesson.CreatedAt.IsZero() {
		t.Error("CreatedAt should have been auto-set")
	}
}

func TestSearchLessonsBySimilarity(t *testing.T) {
	db := newTestDB(t)

	// Store 3 lessons with different embeddings
	lessons := []struct {
		id        string
		title     string
		embedding []float32
	}{
		{"sim-1", "Close match", []float32{0.9, 0.1, 0.0, 0.0}},
		{"sim-2", "Moderate match", []float32{0.5, 0.5, 0.0, 0.0}},
		{"sim-3", "Distant match", []float32{0.0, 0.0, 0.9, 0.1}},
	}

	for _, l := range lessons {
		lesson := &models.Lesson{
			ID:        l.id,
			ProjectID: "proj-sim",
			Category:  "test",
			Title:     l.title,
			Detail:    "Detail for " + l.title,
		}
		err := db.StoreLessonWithEmbedding(lesson, l.embedding)
		if err != nil {
			t.Fatalf("StoreLessonWithEmbedding(%s) failed: %v", l.id, err)
		}
	}

	// Search with embedding close to "Close match"
	queryEmbedding := []float32{0.8, 0.2, 0.0, 0.0}
	results, err := db.SearchLessonsBySimilarity("proj-sim", queryEmbedding, 2)
	if err != nil {
		t.Fatalf("SearchLessonsBySimilarity failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// The first result should be the closest match
	if results[0].ID != "sim-1" {
		t.Errorf("First result ID = %q, want %q (closest match)", results[0].ID, "sim-1")
	}
}

func TestSearchLessonsBySimilarity_DefaultTopK(t *testing.T) {
	db := newTestDB(t)

	// topK=0 should default to 5
	results, err := db.SearchLessonsBySimilarity("proj-empty", nil, 0)
	if err != nil {
		t.Fatalf("SearchLessonsBySimilarity failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestSearchLessonsBySimilarity_NoEmbedding(t *testing.T) {
	db := newTestDB(t)

	// Store a lesson without embedding (using CreateLesson)
	lesson := &models.Lesson{
		ID:        "sim-no-emb",
		ProjectID: "proj-sim-ne",
		Category:  "test",
		Title:     "No Embedding",
		Detail:    "Has no embedding data.",
	}
	if err := db.CreateLesson(lesson); err != nil {
		t.Fatalf("CreateLesson failed: %v", err)
	}

	// Search - should return the lesson with low default similarity
	queryEmbedding := []float32{0.5, 0.5}
	results, err := db.SearchLessonsBySimilarity("proj-sim-ne", queryEmbedding, 5)
	if err != nil {
		t.Fatalf("SearchLessonsBySimilarity failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].ID != "sim-no-emb" {
		t.Errorf("ID = %q, want %q", results[0].ID, "sim-no-emb")
	}
}

// ============================================================
// 17. Additional ListActivities filter tests
// ============================================================

func TestListActivities_ActorFilter(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-af")

	act1 := makeTestActivity("act-af-1")
	act1.ProjectID = "proj-af"
	act1.ActorID = "actor-x"
	act2 := makeTestActivity("act-af-2")
	act2.ProjectID = "proj-af"
	act2.ActorID = "actor-y"

	if err := db.CreateActivity(act1); err != nil {
		t.Fatalf("CreateActivity failed: %v", err)
	}
	if err := db.CreateActivity(act2); err != nil {
		t.Fatalf("CreateActivity failed: %v", err)
	}

	filters := ActivityFilters{ActorID: "actor-x", Limit: 10}
	results, err := db.ListActivities(filters)
	if err != nil {
		t.Fatalf("ListActivities failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for actor-x, got %d", len(results))
	}
}

func TestListActivities_EventTypeFilter(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-tf")

	act1 := makeTestActivity("act-tf-1")
	act1.ProjectID = "proj-tf"
	act1.EventType = "bead.completed"
	act2 := makeTestActivity("act-tf-2")
	act2.ProjectID = "proj-tf"
	act2.EventType = "agent.spawned"

	if err := db.CreateActivity(act1); err != nil {
		t.Fatalf("CreateActivity failed: %v", err)
	}
	if err := db.CreateActivity(act2); err != nil {
		t.Fatalf("CreateActivity failed: %v", err)
	}

	filters := ActivityFilters{EventType: "agent.spawned", Limit: 10}
	results, err := db.ListActivities(filters)
	if err != nil {
		t.Fatalf("ListActivities failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for agent.spawned, got %d", len(results))
	}
}



// ============================================================
// 19. Additional edge case and filter tests
// ============================================================

func TestListActivities_TimeFilters(t *testing.T) {
	db := newTestDB(t)
	ensureProjectExists(t, db, "proj-time")

	act := makeTestActivity("act-time-1")
	act.ProjectID = "proj-time"
	if err := db.CreateActivity(act); err != nil {
		t.Fatalf("CreateActivity failed: %v", err)
	}

	// Filter with Since (before the activity was created)
	past := time.Now().Add(-1 * time.Hour)
	filters := ActivityFilters{
		ProjectIDs: []string{"proj-time"},
		Since:      past,
		Limit:      10,
	}
	results, err := db.ListActivities(filters)
	if err != nil {
		t.Fatalf("ListActivities (since) failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result with Since in past, got %d", len(results))
	}

	// Filter with Until (before the activity was created)
	filters2 := ActivityFilters{
		ProjectIDs: []string{"proj-time"},
		Until:      past,
		Limit:      10,
	}
	results2, err := db.ListActivities(filters2)
	if err != nil {
		t.Fatalf("ListActivities (until) failed: %v", err)
	}
	if len(results2) != 0 {
		t.Errorf("Expected 0 results with Until in past, got %d", len(results2))
	}
}

func TestListNotifications_StatusFilter(t *testing.T) {
	db := newTestDB(t)
	ensureUserExists(t, db, "user-unread", "userunread")

	// Create two notifications
	n1 := Notification{
		ID:        "notif-ur1",
		UserID:    "user-unread",
		Title:     "Unread notification",
		Message:   "msg1",
		Status:    "unread",
		Priority:  "normal",
		CreatedAt: time.Now(),
	}
	n2 := Notification{
		ID:        "notif-ur2",
		UserID:    "user-unread",
		Title:     "Read notification",
		Message:   "msg2",
		Status:    "unread",
		Priority:  "normal",
		CreatedAt: time.Now(),
	}

	if err := db.CreateNotification(&n1); err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}
	if err := db.CreateNotification(&n2); err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	// Mark second as read
	if err := db.MarkNotificationRead("notif-ur2"); err != nil {
		t.Fatalf("MarkNotificationRead failed: %v", err)
	}

	// List unread only
	results, err := db.ListNotifications("user-unread", "unread", 10, 0)
	if err != nil {
		t.Fatalf("ListNotifications (unread) failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 unread notification, got %d", len(results))
	}
	if results[0].ID != "notif-ur1" {
		t.Errorf("ID = %q, want %q", results[0].ID, "notif-ur1")
	}
}

// ============================================================
// 20. isAlterColumnExistsError tests (lessons.go)
// ============================================================

func TestIsAlterColumnExistsError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"duplicate error", fmt.Errorf("duplicate column name: embedding"), true},
		{"other error", fmt.Errorf("syntax error"), false},
		{"short error", fmt.Errorf("short"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAlterColumnExistsError(tt.err)
			if got != tt.want {
				t.Errorf("isAlterColumnExistsError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// ============================================================
// 21. Memory package usage verification
// ============================================================

func TestSearchLessonsBySimilarity_WithEmbeddings(t *testing.T) {
	db := newTestDB(t)

	// Store multiple lessons with varied embeddings
	type testLesson struct {
		id    string
		emb   []float32
		title string
	}
	testData := []testLesson{
		{"sem-1", []float32{1.0, 0.0, 0.0}, "Pure X"},
		{"sem-2", []float32{0.0, 1.0, 0.0}, "Pure Y"},
		{"sem-3", []float32{0.0, 0.0, 1.0}, "Pure Z"},
		{"sem-4", []float32{0.7, 0.7, 0.0}, "X-Y mix"},
	}

	for _, td := range testData {
		lesson := &models.Lesson{
			ID:        td.id,
			ProjectID: "proj-sem",
			Category:  "test",
			Title:     td.title,
			Detail:    "Detail for " + td.title,
		}
		if err := db.StoreLessonWithEmbedding(lesson, td.emb); err != nil {
			t.Fatalf("StoreLessonWithEmbedding(%s) failed: %v", td.id, err)
		}
	}

	// Query for X direction
	results, err := db.SearchLessonsBySimilarity("proj-sem", []float32{1.0, 0.0, 0.0}, 4)
	if err != nil {
		t.Fatalf("SearchLessonsBySimilarity failed: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("Expected 4 results, got %d", len(results))
	}

	// Verify the encoding/decoding roundtrip works
	enc := memory.EncodeEmbedding([]float32{1.0, 2.0, 3.0})
	dec := memory.DecodeEmbedding(enc)
	if len(dec) != 3 {
		t.Fatalf("Roundtrip embedding length = %d, want 3", len(dec))
	}
	if dec[0] != 1.0 || dec[1] != 2.0 || dec[2] != 3.0 {
		t.Errorf("Roundtrip embedding = %v, want [1.0 2.0 3.0]", dec)
	}
}

// ============================================================
// 22. Additional provider coverage tests
// ============================================================

func TestListProviders_WithAllFields(t *testing.T) {
	db := newTestDB(t)

	// UpsertProvider with many fields to exercise full scan path
	p := &internalmodels.Provider{
		ID:              "prov-scan",
		Name:            "Full Provider",
		Endpoint:        "https://api.example.com",
		Type:            "openai",
		Model:           "gpt-4",
		ConfiguredModel: "gpt-4",
		SelectedModel:   "gpt-4-turbo",
		SelectionReason: "best quality",
		ModelScore:      0.95,
		SelectedGPU:     "A100",
		ContextWindow:   128000,
		Description:     "A full test provider",
		RequiresKey:     true,
		KeyID:           "key-scan",
		OwnerID:         "owner-scan",
		IsShared:        true,
		Status:          "active",
	}
	if err := db.UpsertProvider(p); err != nil {
		t.Fatalf("UpsertProvider failed: %v", err)
	}

	providers, err := db.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}

	var found *internalmodels.Provider
	for _, pr := range providers {
		if pr.ID == "prov-scan" {
			found = pr
			break
		}
	}
	if found == nil {
		t.Fatal("Provider prov-scan not found in list")
	}
	if found.ConfiguredModel != "gpt-4" {
		t.Errorf("ConfiguredModel = %q, want %q", found.ConfiguredModel, "gpt-4")
	}
	if found.SelectedModel != "gpt-4-turbo" {
		t.Errorf("SelectedModel = %q, want %q", found.SelectedModel, "gpt-4-turbo")
	}
	if found.SelectionReason != "best quality" {
		t.Errorf("SelectionReason = %q, want %q", found.SelectionReason, "best quality")
	}
}

// ============================================================
// 23. ListUsers test
// ============================================================

func TestListUsers_MultipleUsers(t *testing.T) {
	db := newTestDB(t)

	// The default admin user is created during migration
	users, err := db.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	initialCount := len(users)

	// Create additional users
	if err := db.CreateUser("extra-user-1", "Extra User 1", "extra1@test.com", "user"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if err := db.CreateUser("extra-user-2", "Extra User 2", "extra2@test.com", "admin"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	users, err = db.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != initialCount+2 {
		t.Errorf("Expected %d users, got %d", initialCount+2, len(users))
	}
}

