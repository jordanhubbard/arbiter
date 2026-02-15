package project

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// NewBootstrapService tests
// ---------------------------------------------------------------------------

func TestNewBootstrapService(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/templates", "/workspace", nil, "sqlite")

	if bs == nil {
		t.Fatal("Expected non-nil BootstrapService")
	}
	if bs.projectManager != pm {
		t.Error("Expected projectManager to be set")
	}
	if bs.templateDir != "/templates" {
		t.Errorf("Expected templateDir '/templates', got %q", bs.templateDir)
	}
	if bs.workspaceDir != "/workspace" {
		t.Errorf("Expected workspaceDir '/workspace', got %q", bs.workspaceDir)
	}
	if bs.gitopsManager != nil {
		t.Error("Expected gitopsManager to be nil")
	}
	if bs.beadsBackend != "sqlite" {
		t.Errorf("Expected beadsBackend 'sqlite', got %q", bs.beadsBackend)
	}
}

func TestNewBootstrapService_DoltBackend(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "dolt")

	if bs.beadsBackend != "dolt" {
		t.Errorf("Expected beadsBackend 'dolt', got %q", bs.beadsBackend)
	}
}

func TestNewBootstrapService_EmptyBackend(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "", "", nil, "")

	if bs.beadsBackend != "" {
		t.Errorf("Expected empty beadsBackend, got %q", bs.beadsBackend)
	}
	if bs.templateDir != "" {
		t.Errorf("Expected empty templateDir, got %q", bs.templateDir)
	}
}

// ---------------------------------------------------------------------------
// Bootstrap validation tests
// ---------------------------------------------------------------------------

func TestBootstrap_MissingGitHubURL(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "sqlite")

	req := BootstrapRequest{
		Name:    "Test Project",
		Branch:  "main",
		PRDText: "Some PRD content",
	}

	_, err := bs.Bootstrap(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for missing GitHubURL")
	}
	if !strings.Contains(err.Error(), "github_url, name, and branch are required") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestBootstrap_MissingName(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "sqlite")

	req := BootstrapRequest{
		GitHubURL: "https://github.com/test/repo",
		Branch:    "main",
		PRDText:   "Some PRD content",
	}

	_, err := bs.Bootstrap(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for missing Name")
	}
	if !strings.Contains(err.Error(), "github_url, name, and branch are required") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestBootstrap_MissingBranch(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "sqlite")

	req := BootstrapRequest{
		GitHubURL: "https://github.com/test/repo",
		Name:      "Test Project",
		PRDText:   "Some PRD content",
	}

	_, err := bs.Bootstrap(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for missing Branch")
	}
	if !strings.Contains(err.Error(), "github_url, name, and branch are required") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestBootstrap_MissingPRD(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "sqlite")

	req := BootstrapRequest{
		GitHubURL: "https://github.com/test/repo",
		Name:      "Test Project",
		Branch:    "main",
	}

	_, err := bs.Bootstrap(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for missing PRD")
	}
	if !strings.Contains(err.Error(), "either prd_text or prd_file must be provided") {
		t.Errorf("Expected PRD validation error, got: %v", err)
	}
}

func TestBootstrap_AllFieldsMissing(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "sqlite")

	req := BootstrapRequest{}

	_, err := bs.Bootstrap(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty request")
	}
	if !strings.Contains(err.Error(), "github_url, name, and branch are required") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestBootstrap_EmptyPRDTextAndEmptyPRDFile(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "sqlite")

	req := BootstrapRequest{
		GitHubURL: "https://github.com/test/repo",
		Name:      "Test Project",
		Branch:    "main",
		PRDText:   "",
		PRDFile:   []byte{},
	}

	_, err := bs.Bootstrap(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty PRD content")
	}
	if !strings.Contains(err.Error(), "either prd_text or prd_file must be provided") {
		t.Errorf("Expected PRD validation error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// BootstrapRequest struct tests
// ---------------------------------------------------------------------------

func TestBootstrapRequest_Fields(t *testing.T) {
	req := BootstrapRequest{
		GitHubURL: "https://github.com/org/repo",
		Name:      "My Project",
		Branch:    "develop",
		PRDText:   "Product requirements document",
		PRDFile:   []byte("file content"),
	}

	if req.GitHubURL != "https://github.com/org/repo" {
		t.Errorf("Expected GitHubURL, got %q", req.GitHubURL)
	}
	if req.Name != "My Project" {
		t.Errorf("Expected Name 'My Project', got %q", req.Name)
	}
	if req.Branch != "develop" {
		t.Errorf("Expected Branch 'develop', got %q", req.Branch)
	}
	if req.PRDText != "Product requirements document" {
		t.Errorf("Expected PRDText, got %q", req.PRDText)
	}
	if string(req.PRDFile) != "file content" {
		t.Errorf("Expected PRDFile 'file content', got %q", string(req.PRDFile))
	}
}

// ---------------------------------------------------------------------------
// BootstrapResult struct tests
// ---------------------------------------------------------------------------

func TestBootstrapResult_Fields(t *testing.T) {
	result := BootstrapResult{
		ProjectID:            "proj-123",
		Status:               "ready",
		InitialBead:          "bead-456",
		PublicKey:             "ssh-ed25519 AAAA...",
		GitSetupInstructions: "Add this deploy key",
		Error:                "",
	}

	if result.ProjectID != "proj-123" {
		t.Errorf("Expected ProjectID 'proj-123', got %q", result.ProjectID)
	}
	if result.Status != "ready" {
		t.Errorf("Expected Status 'ready', got %q", result.Status)
	}
	if result.InitialBead != "bead-456" {
		t.Errorf("Expected InitialBead 'bead-456', got %q", result.InitialBead)
	}
	if result.PublicKey != "ssh-ed25519 AAAA..." {
		t.Errorf("Expected PublicKey, got %q", result.PublicKey)
	}
	if result.GitSetupInstructions != "Add this deploy key" {
		t.Errorf("Expected GitSetupInstructions, got %q", result.GitSetupInstructions)
	}
	if result.Error != "" {
		t.Errorf("Expected empty Error, got %q", result.Error)
	}
}

func TestBootstrapResult_WithError(t *testing.T) {
	result := BootstrapResult{
		ProjectID: "proj-123",
		Status:    "error",
		Error:     "something went wrong",
	}

	if result.Error != "something went wrong" {
		t.Errorf("Expected error message, got %q", result.Error)
	}
}

// ---------------------------------------------------------------------------
// copyTemplateFiles tests
// ---------------------------------------------------------------------------

func TestCopyTemplateFiles(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "sqlite")

	// Create a temp directory for testing
	tmpDir := filepath.Join(os.TempDir(), "loom-test-copy-template")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err := bs.copyTemplateFiles(tmpDir)
	if err != nil {
		t.Fatalf("copyTemplateFiles failed: %v", err)
	}

	// Verify settings.json was created
	settingsPath := filepath.Join(tmpDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}
	if !strings.Contains(string(data), "mcpServers") {
		t.Error("Expected settings.json to contain 'mcpServers'")
	}
	if !strings.Contains(string(data), "responsible-vibe-mcp") {
		t.Error("Expected settings.json to contain 'responsible-vibe-mcp'")
	}

	// Verify .mcp.json was created
	mcpPath := filepath.Join(tmpDir, ".mcp.json")
	data, err = os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("Failed to read .mcp.json: %v", err)
	}
	if !strings.Contains(string(data), "mcpServers") {
		t.Error("Expected .mcp.json to contain 'mcpServers'")
	}
}

// ---------------------------------------------------------------------------
// initializeProjectStructure tests
// ---------------------------------------------------------------------------

func TestInitializeProjectStructure(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "sqlite")

	// Create a temp directory for testing
	tmpDir := filepath.Join(os.TempDir(), "loom-test-init-structure")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	prdContent := "# Product Requirements\n\nThis is a test PRD."

	err := bs.initializeProjectStructure(ctx, tmpDir, prdContent)
	if err != nil {
		t.Fatalf("initializeProjectStructure failed: %v", err)
	}

	// Verify plans directory was created
	plansDir := filepath.Join(tmpDir, "plans")
	info, err := os.Stat(plansDir)
	if err != nil {
		t.Fatalf("plans directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected plans to be a directory")
	}

	// Verify BOOTSTRAP.md was created with PRD content
	bootstrapPath := filepath.Join(plansDir, "BOOTSTRAP.md")
	data, err := os.ReadFile(bootstrapPath)
	if err != nil {
		t.Fatalf("Failed to read BOOTSTRAP.md: %v", err)
	}
	if string(data) != prdContent {
		t.Errorf("Expected BOOTSTRAP.md content %q, got %q", prdContent, string(data))
	}

	// Verify template files were also created
	settingsPath := filepath.Join(tmpDir, "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Error("Expected settings.json to be created by initializeProjectStructure")
	}
}

func TestInitializeProjectStructure_InvalidPath(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "sqlite")

	ctx := context.Background()
	// Use a path that cannot be created (null byte in path)
	err := bs.initializeProjectStructure(ctx, "/dev/null/impossible/path", "PRD content")
	if err == nil {
		t.Fatal("Expected error for invalid path")
	}
}

// ---------------------------------------------------------------------------
// Bootstrap integration-style tests with cancelled context
// ---------------------------------------------------------------------------

func TestBootstrap_ContextCancelled(t *testing.T) {
	pm := NewManager()
	// Use a temp directory that exists so MkdirAll succeeds but git clone will fail
	tmpDir := filepath.Join(os.TempDir(), "loom-test-bootstrap-cancel")
	defer os.RemoveAll(tmpDir)

	bs := NewBootstrapService(pm, "/tpl", tmpDir, nil, "sqlite")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := BootstrapRequest{
		GitHubURL: "https://github.com/test/repo",
		Name:      "Test Project",
		Branch:    "main",
		PRDText:   "Some PRD content",
	}

	_, err := bs.Bootstrap(ctx, req)
	// Should fail because git clone with cancelled context will fail
	if err == nil {
		t.Fatal("Expected error with cancelled context")
	}
}

// ---------------------------------------------------------------------------
// initializeBeads tests (indirect — backend selection)
// ---------------------------------------------------------------------------

func TestInitializeBeads_SqliteBackend(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "sqlite")

	// We can't run bd init in a test, but verify the backend is stored correctly
	if bs.beadsBackend != "sqlite" {
		t.Errorf("Expected beadsBackend 'sqlite', got %q", bs.beadsBackend)
	}
}

func TestInitializeBeads_DoltBackend(t *testing.T) {
	pm := NewManager()
	bs := NewBootstrapService(pm, "/tpl", "/ws", nil, "dolt")

	if bs.beadsBackend != "dolt" {
		t.Errorf("Expected beadsBackend 'dolt', got %q", bs.beadsBackend)
	}
}
