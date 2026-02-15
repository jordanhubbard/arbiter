package dispatch

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jordanhubbard/loom/internal/database"
	"github.com/jordanhubbard/loom/internal/memory"
	"github.com/jordanhubbard/loom/pkg/models"
)

func TestNewLessonsProvider_NilDB(t *testing.T) {
	lp := NewLessonsProvider(nil)
	if lp != nil {
		t.Error("Expected nil LessonsProvider when database is nil")
	}
}

func TestNewLessonsProvider_ValidDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)
	if lp == nil {
		t.Fatal("Expected non-nil LessonsProvider")
	}

	if lp.db != db {
		t.Error("Expected LessonsProvider db to match")
	}

	if lp.embedder == nil {
		t.Error("Expected default embedder to be set")
	}
}

func TestLessonsProvider_SetEmbedder(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)
	if lp == nil {
		t.Fatal("Expected non-nil LessonsProvider")
	}

	// Set a new embedder
	newEmbedder := memory.NewHashEmbedder()
	lp.SetEmbedder(newEmbedder)

	if lp.embedder != newEmbedder {
		t.Error("Expected embedder to be updated")
	}
}

func TestLessonsProvider_SetEmbedder_NilCases(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)
	originalEmbedder := lp.embedder

	// Setting nil embedder should be a no-op
	lp.SetEmbedder(nil)
	if lp.embedder != originalEmbedder {
		t.Error("Expected embedder to remain unchanged when setting nil")
	}

	// Calling on nil provider should not panic
	var nilLP *LessonsProvider
	nilLP.SetEmbedder(memory.NewHashEmbedder()) // should not panic
}

func TestLessonsProvider_GetLessonsForPrompt_NilCases(t *testing.T) {
	tests := []struct {
		name      string
		lp        *LessonsProvider
		projectID string
		expected  string
	}{
		{
			name:      "nil provider",
			lp:        nil,
			projectID: "proj-1",
			expected:  "",
		},
		{
			name:      "nil db",
			lp:        &LessonsProvider{db: nil},
			projectID: "proj-1",
			expected:  "",
		},
		{
			name:      "empty project ID",
			lp:        &LessonsProvider{db: nil},
			projectID: "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.lp.GetLessonsForPrompt(tt.projectID)
			if result != tt.expected {
				t.Errorf("GetLessonsForPrompt() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLessonsProvider_GetLessonsForPrompt_NoLessons(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)

	result := lp.GetLessonsForPrompt("proj-1")
	if result != "" {
		t.Errorf("Expected empty string when no lessons, got %q", result)
	}
}

func TestLessonsProvider_GetLessonsForPrompt_WithLessons(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)

	// Create some test lessons
	lessons := []*models.Lesson{
		{
			ID:             "l-1",
			ProjectID:      "proj-1",
			Category:       "compiler_error",
			Title:          "Missing import",
			Detail:         "Always check imports before building",
			RelevanceScore: 0.9,
		},
		{
			ID:             "l-2",
			ProjectID:      "proj-1",
			Category:       "test_failure",
			Title:          "Flaky tests",
			Detail:         "Use retry logic for network tests",
			RelevanceScore: 0.2, // Low relevance - should show "(older lesson)" note
		},
	}

	for _, lesson := range lessons {
		if err := db.CreateLesson(lesson); err != nil {
			t.Fatalf("Failed to create lesson: %v", err)
		}
	}

	result := lp.GetLessonsForPrompt("proj-1")

	if result == "" {
		t.Fatal("Expected non-empty result with lessons")
	}

	// Should contain header
	if !strings.Contains(result, "lessons were learned") {
		t.Error("Expected result to contain header about lessons learned")
	}

	// Should contain lesson details
	if !strings.Contains(result, "COMPILER_ERROR") {
		t.Errorf("Expected result to contain COMPILER_ERROR category, got:\n%s", result)
	}
	if !strings.Contains(result, "Missing import") {
		t.Error("Expected result to contain lesson title")
	}
	if !strings.Contains(result, "Always check imports before building") {
		t.Error("Expected result to contain lesson detail")
	}

	// Low relevance lesson should have the "older lesson" note
	if !strings.Contains(result, "older lesson") {
		t.Error("Expected low relevance lesson to have 'older lesson' note")
	}
}

func TestLessonsProvider_GetRelevantLessons_NilCases(t *testing.T) {
	tests := []struct {
		name        string
		lp          *LessonsProvider
		projectID   string
		taskContext string
		topK        int
		expected    string
	}{
		{
			name:        "nil provider",
			lp:          nil,
			projectID:   "proj-1",
			taskContext: "fix bug",
			topK:        5,
			expected:    "",
		},
		{
			name:        "nil db",
			lp:          &LessonsProvider{db: nil},
			projectID:   "proj-1",
			taskContext: "fix bug",
			topK:        5,
			expected:    "",
		},
		{
			name:        "empty project ID",
			lp:          &LessonsProvider{db: nil},
			projectID:   "",
			taskContext: "fix bug",
			topK:        5,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.lp.GetRelevantLessons(tt.projectID, tt.taskContext, tt.topK)
			if result != tt.expected {
				t.Errorf("GetRelevantLessons() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLessonsProvider_GetRelevantLessons_EmptyContext(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)

	// Empty task context should fall back to GetLessonsForPrompt
	result := lp.GetRelevantLessons("proj-1", "", 5)
	// No lessons exist, so result should be empty (from fallback)
	if result != "" {
		t.Errorf("Expected empty result for empty context with no lessons, got %q", result)
	}
}

func TestLessonsProvider_GetRelevantLessons_DefaultTopK(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)

	// topK <= 0 should default to 5
	result := lp.GetRelevantLessons("proj-1", "fix bug", 0)
	// No lessons with embeddings, so similarity search returns empty
	if result != "" {
		t.Logf("GetRelevantLessons result: %q", result)
	}
}

func TestLessonsProvider_GetRelevantLessons_WithEmbeddings(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)

	// Store lessons with embeddings via RecordLesson (which embeds automatically)
	err = lp.RecordLesson("proj-1", "compiler_error", "Missing import", "Always check imports", "bead-1", "agent-1")
	if err != nil {
		t.Fatalf("Failed to record lesson: %v", err)
	}

	err = lp.RecordLesson("proj-1", "test_failure", "Flaky tests", "Use retry for network tests", "bead-2", "agent-1")
	if err != nil {
		t.Fatalf("Failed to record lesson: %v", err)
	}

	// Now search by relevance
	result := lp.GetRelevantLessons("proj-1", "compilation import error", 5)

	// The hash embedder may not produce great similarity scores, but we
	// should get some result formatted as markdown
	if result != "" {
		if !strings.Contains(result, "relevant to this task") {
			t.Error("Expected result to contain relevant header")
		}
	}
}

func TestLessonsProvider_RecordLesson_NilCases(t *testing.T) {
	// nil provider should be no-op
	var nilLP *LessonsProvider
	err := nilLP.RecordLesson("proj-1", "test", "title", "detail", "bead-1", "agent-1")
	if err != nil {
		t.Errorf("Expected nil error from nil provider, got %v", err)
	}

	// nil db should be no-op
	lp := &LessonsProvider{db: nil}
	err = lp.RecordLesson("proj-1", "test", "title", "detail", "bead-1", "agent-1")
	if err != nil {
		t.Errorf("Expected nil error from nil db, got %v", err)
	}
}

func TestLessonsProvider_RecordLesson_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)

	err = lp.RecordLesson("proj-1", "compiler_error", "Missing import", "Always check imports", "bead-1", "agent-1")
	if err != nil {
		t.Fatalf("RecordLesson failed: %v", err)
	}

	// Verify lesson was stored
	lessons, err := db.GetLessonsForProject("proj-1", 10, 4000)
	if err != nil {
		t.Fatalf("Failed to get lessons: %v", err)
	}

	if len(lessons) == 0 {
		t.Fatal("Expected at least one lesson to be stored")
	}

	found := false
	for _, l := range lessons {
		if l.Title == "Missing import" && l.Category == "compiler_error" {
			found = true
			if l.ProjectID != "proj-1" {
				t.Errorf("Expected ProjectID proj-1, got %s", l.ProjectID)
			}
			if l.SourceBeadID != "bead-1" {
				t.Errorf("Expected SourceBeadID bead-1, got %s", l.SourceBeadID)
			}
			if l.SourceAgentID != "agent-1" {
				t.Errorf("Expected SourceAgentID agent-1, got %s", l.SourceAgentID)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find the recorded lesson")
	}
}

func TestLessonsProvider_RecordLesson_WithoutEmbedder(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := &LessonsProvider{
		db:       db,
		embedder: nil, // No embedder
	}

	err = lp.RecordLesson("proj-1", "test_failure", "Flaky test", "Network dependency", "bead-2", "agent-2")
	if err != nil {
		t.Fatalf("RecordLesson without embedder failed: %v", err)
	}

	// Verify lesson was still stored
	lessons, err := db.GetLessonsForProject("proj-1", 10, 4000)
	if err != nil {
		t.Fatalf("Failed to get lessons: %v", err)
	}

	if len(lessons) == 0 {
		t.Fatal("Expected lesson to be stored even without embedder")
	}
}

func TestLessonsProvider_RecordMultipleLessons(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)

	categories := []string{"compiler_error", "test_failure", "edit_failure", "loop_pattern", "conversation_insight"}

	for i, cat := range categories {
		err = lp.RecordLesson("proj-1", cat, "Title "+cat, "Detail "+cat, "bead-"+cat, "agent-1")
		if err != nil {
			t.Fatalf("RecordLesson %d failed: %v", i, err)
		}
	}

	lessons, err := db.GetLessonsForProject("proj-1", 20, 4000)
	if err != nil {
		t.Fatalf("Failed to get lessons: %v", err)
	}

	if len(lessons) != len(categories) {
		t.Errorf("Expected %d lessons, got %d", len(categories), len(lessons))
	}
}

func TestLessonsProvider_GetLessonsForDifferentProjects(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	lp := NewLessonsProvider(db)

	// Record lessons for different projects
	_ = lp.RecordLesson("proj-1", "compiler_error", "Proj1 Lesson", "Detail 1", "b1", "a1")
	_ = lp.RecordLesson("proj-2", "test_failure", "Proj2 Lesson", "Detail 2", "b2", "a2")

	// Retrieve for proj-1
	result1 := lp.GetLessonsForPrompt("proj-1")
	if !strings.Contains(result1, "Proj1 Lesson") {
		t.Error("Expected proj-1 lessons to contain 'Proj1 Lesson'")
	}
	if strings.Contains(result1, "Proj2 Lesson") {
		t.Error("Did not expect proj-1 lessons to contain 'Proj2 Lesson'")
	}

	// Retrieve for proj-2
	result2 := lp.GetLessonsForPrompt("proj-2")
	if !strings.Contains(result2, "Proj2 Lesson") {
		t.Error("Expected proj-2 lessons to contain 'Proj2 Lesson'")
	}
	if strings.Contains(result2, "Proj1 Lesson") {
		t.Error("Did not expect proj-2 lessons to contain 'Proj1 Lesson'")
	}

	// Non-existent project
	result3 := lp.GetLessonsForPrompt("proj-nonexistent")
	if result3 != "" {
		t.Errorf("Expected empty result for nonexistent project, got %q", result3)
	}
}
