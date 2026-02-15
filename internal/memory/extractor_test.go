package memory

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

// --- Mock LessonStore ---

type mockLessonStore struct {
	lessons          []*models.Lesson
	embeddings       [][]float32
	createErr        error
	storeEmbedErr    error
}

func (m *mockLessonStore) StoreLessonWithEmbedding(lesson *models.Lesson, embedding []float32) error {
	if m.storeEmbedErr != nil {
		return m.storeEmbedErr
	}
	m.lessons = append(m.lessons, lesson)
	m.embeddings = append(m.embeddings, embedding)
	return nil
}

func (m *mockLessonStore) CreateLesson(lesson *models.Lesson) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.lessons = append(m.lessons, lesson)
	return nil
}

// --- Mock failing embedder ---

type failingEmbedder struct{}

func (e *failingEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	return nil, fmt.Errorf("embedding failed")
}

// --- Tests for NewExtractor ---

func TestNewExtractor(t *testing.T) {
	store := &mockLessonStore{}
	embedder := NewHashEmbedder()

	ext := NewExtractor(store, embedder)
	if ext == nil {
		t.Fatal("NewExtractor returned nil")
	}
	if ext.store != store {
		t.Error("store not set correctly")
	}
	if ext.embedder != embedder {
		t.Error("embedder not set correctly")
	}
}

func TestNewExtractor_NilArgs(t *testing.T) {
	ext := NewExtractor(nil, nil)
	if ext == nil {
		t.Fatal("NewExtractor returned nil")
	}
}

// --- Tests for ExtractFromLoop ---

func TestExtractFromLoop_NilExtractor(t *testing.T) {
	// Should not panic
	var ext *Extractor
	ext.ExtractFromLoop("proj", "bead", []ActionEntry{{ActionType: "build_project", Status: "error", Message: "fail"}}, "")
}

func TestExtractFromLoop_NilStore(t *testing.T) {
	ext := &Extractor{store: nil, embedder: NewHashEmbedder()}
	// Should not panic
	ext.ExtractFromLoop("proj", "bead", []ActionEntry{{ActionType: "build_project", Status: "error", Message: "fail"}}, "")
}

func TestExtractFromLoop_EmptyEntries(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, NewHashEmbedder())
	ext.ExtractFromLoop("proj", "bead", nil, "")
	if len(store.lessons) != 0 {
		t.Errorf("expected 0 lessons for nil entries, got %d", len(store.lessons))
	}

	ext.ExtractFromLoop("proj", "bead", []ActionEntry{}, "")
	if len(store.lessons) != 0 {
		t.Errorf("expected 0 lessons for empty entries, got %d", len(store.lessons))
	}
}

func TestExtractFromLoop_BuildPatterns(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, NewHashEmbedder())

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "build_project", Status: "error", Message: "undefined: foo"},
		{Iteration: 2, ActionType: "build_project", Status: "error", Message: "cannot find package"},
		{Iteration: 3, ActionType: "build_project", Status: "success", Message: "ok"},
	}

	ext.ExtractFromLoop("proj-1", "bead-1", entries, "")

	if len(store.lessons) == 0 {
		t.Fatal("expected at least one lesson for repeated build failures")
	}

	found := false
	for _, l := range store.lessons {
		if strings.Contains(l.Title, "build failures") {
			found = true
			if l.ProjectID != "proj-1" {
				t.Errorf("expected ProjectID 'proj-1', got %q", l.ProjectID)
			}
			if l.Category != "conversation_insight" {
				t.Errorf("expected Category 'conversation_insight', got %q", l.Category)
			}
			if l.SourceBeadID != "bead-1" {
				t.Errorf("expected SourceBeadID 'bead-1', got %q", l.SourceBeadID)
			}
		}
	}
	if !found {
		t.Error("expected a lesson about build failures")
	}
}

func TestExtractFromLoop_TestPatterns(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, NewHashEmbedder())

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "run_tests", Status: "error", Message: "FAIL: TestFoo"},
		{Iteration: 2, ActionType: "run_tests", Status: "error", Message: "FAIL: TestBar"},
	}

	ext.ExtractFromLoop("proj-1", "bead-1", entries, "")

	found := false
	for _, l := range store.lessons {
		if strings.Contains(l.Title, "test failures") {
			found = true
		}
	}
	if !found {
		t.Error("expected a lesson about test failures")
	}
}

func TestExtractFromLoop_EditPatterns(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, NewHashEmbedder())

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "apply_patch", Status: "error", Message: "patch failed", Path: "main.go"},
		{Iteration: 2, ActionType: "edit_code", Status: "error", Message: "edit failed", Path: "main.go"},
		{Iteration: 3, ActionType: "apply_patch", Status: "error", Message: "patch failed again", Path: "utils.go"},
	}

	ext.ExtractFromLoop("proj-1", "bead-1", entries, "")

	found := false
	for _, l := range store.lessons {
		if strings.Contains(l.Title, "edit failures on main.go") {
			found = true
		}
	}
	if !found {
		t.Error("expected a lesson about edit failures on main.go")
	}

	// utils.go had only 1 failure, should not generate a lesson
	for _, l := range store.lessons {
		if strings.Contains(l.Title, "utils.go") {
			t.Error("utils.go should not generate a lesson with only 1 failure")
		}
	}
}

func TestExtractFromLoop_TerminalInsight_MaxIterations(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, NewHashEmbedder())

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "build_project", Status: "success", Message: "ok"},
	}

	ext.ExtractFromLoop("proj-1", "bead-1", entries, "max_iterations")

	found := false
	for _, l := range store.lessons {
		if strings.Contains(l.Title, "max iterations") {
			found = true
			if !strings.Contains(l.Detail, "1 total actions") {
				t.Errorf("detail should mention total actions: %q", l.Detail)
			}
		}
	}
	if !found {
		t.Error("expected a lesson about max iterations")
	}
}

func TestExtractFromLoop_TerminalInsight_InnerLoop(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, NewHashEmbedder())

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "build_project", Status: "success", Message: "ok"},
	}

	ext.ExtractFromLoop("proj-1", "bead-1", entries, "inner_loop")

	found := false
	for _, l := range store.lessons {
		if strings.Contains(l.Title, "stuck in action loop") {
			found = true
		}
	}
	if !found {
		t.Error("expected a lesson about inner loop")
	}
}

func TestExtractFromLoop_TerminalInsight_ParseFailures(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, NewHashEmbedder())

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "build_project", Status: "success", Message: "ok"},
	}

	ext.ExtractFromLoop("proj-1", "bead-1", entries, "parse_failures")

	found := false
	for _, l := range store.lessons {
		if strings.Contains(l.Title, "unparseable responses") {
			found = true
		}
	}
	if !found {
		t.Error("expected a lesson about parse failures")
	}
}

func TestExtractFromLoop_TerminalInsight_UnknownReason(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, NewHashEmbedder())

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "build_project", Status: "success", Message: "ok"},
	}

	// Unknown reason should not generate a terminal insight
	ext.ExtractFromLoop("proj-1", "bead-1", entries, "unknown_reason")
	if len(store.lessons) != 0 {
		t.Errorf("expected 0 lessons for unknown terminal reason + no patterns, got %d", len(store.lessons))
	}
}

func TestExtractFromLoop_NoEmbedder_FallsBackToCreateLesson(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, nil) // nil embedder

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "build_project", Status: "error", Message: "fail1"},
		{Iteration: 2, ActionType: "build_project", Status: "error", Message: "fail2"},
	}

	ext.ExtractFromLoop("proj-1", "bead-1", entries, "")

	if len(store.lessons) == 0 {
		t.Fatal("expected lessons even without embedder")
	}
	// With nil embedder, should use CreateLesson (no embedding stored)
	if len(store.embeddings) != 0 {
		t.Errorf("expected 0 embeddings with nil embedder, got %d", len(store.embeddings))
	}
}

func TestExtractFromLoop_FailingEmbedder_FallsBackToCreateLesson(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, &failingEmbedder{})

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "build_project", Status: "error", Message: "fail1"},
		{Iteration: 2, ActionType: "build_project", Status: "error", Message: "fail2"},
	}

	ext.ExtractFromLoop("proj-1", "bead-1", entries, "")

	if len(store.lessons) == 0 {
		t.Fatal("expected lessons even with failing embedder")
	}
}

func TestExtractFromLoop_StoreEmbeddingError(t *testing.T) {
	store := &mockLessonStore{storeEmbedErr: fmt.Errorf("db error")}
	ext := NewExtractor(store, NewHashEmbedder())

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "build_project", Status: "error", Message: "fail1"},
		{Iteration: 2, ActionType: "build_project", Status: "error", Message: "fail2"},
	}

	// Should not panic even when StoreLessonWithEmbedding fails
	ext.ExtractFromLoop("proj-1", "bead-1", entries, "")
}

func TestExtractFromLoop_CreateLessonError(t *testing.T) {
	store := &mockLessonStore{createErr: fmt.Errorf("db error")}
	ext := NewExtractor(store, nil) // nil embedder forces CreateLesson path

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "build_project", Status: "error", Message: "fail1"},
		{Iteration: 2, ActionType: "build_project", Status: "error", Message: "fail2"},
	}

	// Should not panic even when CreateLesson fails
	ext.ExtractFromLoop("proj-1", "bead-1", entries, "")
}

func TestExtractFromLoop_CombinedPatterns(t *testing.T) {
	store := &mockLessonStore{}
	ext := NewExtractor(store, NewHashEmbedder())

	entries := []ActionEntry{
		{Iteration: 1, ActionType: "build_project", Status: "error", Message: "fail1"},
		{Iteration: 2, ActionType: "build_project", Status: "error", Message: "fail2"},
		{Iteration: 3, ActionType: "run_tests", Status: "error", Message: "FAIL: TestFoo"},
		{Iteration: 4, ActionType: "run_tests", Status: "error", Message: "FAIL: TestBar"},
		{Iteration: 5, ActionType: "apply_patch", Status: "error", Message: "patch fail", Path: "main.go"},
		{Iteration: 6, ActionType: "edit_code", Status: "error", Message: "edit fail", Path: "main.go"},
	}

	ext.ExtractFromLoop("proj-1", "bead-1", entries, "max_iterations")

	// Should have lessons for: build failures, test failures, edit failures, max iterations
	if len(store.lessons) < 4 {
		t.Errorf("expected at least 4 lessons for combined patterns, got %d", len(store.lessons))
	}
}

// --- Tests for extractBuildPatterns ---

func TestExtractBuildPatterns_SingleFailure(t *testing.T) {
	entries := []ActionEntry{
		{ActionType: "build_project", Status: "error", Message: "fail"},
	}
	result := extractBuildPatterns(entries)
	if len(result) != 0 {
		t.Errorf("expected 0 lessons for 1 build failure, got %d", len(result))
	}
}

func TestExtractBuildPatterns_NoFailures(t *testing.T) {
	entries := []ActionEntry{
		{ActionType: "build_project", Status: "success", Message: "ok"},
		{ActionType: "run_tests", Status: "error", Message: "fail"},
	}
	result := extractBuildPatterns(entries)
	if len(result) != 0 {
		t.Errorf("expected 0 lessons with no build failures, got %d", len(result))
	}
}

func TestExtractBuildPatterns_MoreThanThreeFailures(t *testing.T) {
	entries := []ActionEntry{
		{ActionType: "build_project", Status: "error", Message: "fail1"},
		{ActionType: "build_project", Status: "error", Message: "fail2"},
		{ActionType: "build_project", Status: "error", Message: "fail3"},
		{ActionType: "build_project", Status: "error", Message: "fail4"},
	}
	result := extractBuildPatterns(entries)
	if len(result) != 1 {
		t.Fatalf("expected 1 lesson, got %d", len(result))
	}
	if !strings.Contains(result[0].title, "4 times") {
		t.Errorf("expected '4 times' in title, got %q", result[0].title)
	}
	// Detail should only contain first 3 failures
	parts := strings.Split(result[0].detail, "; ")
	// The detail starts with "Build failed multiple times: " so first part has that prefix
	if len(parts) > 3 {
		t.Errorf("detail should contain at most 3 failures, got %d parts", len(parts))
	}
}

// --- Tests for extractTestPatterns ---

func TestExtractTestPatterns_SingleFailure(t *testing.T) {
	entries := []ActionEntry{
		{ActionType: "run_tests", Status: "error", Message: "FAIL"},
	}
	result := extractTestPatterns(entries)
	if len(result) != 0 {
		t.Errorf("expected 0 lessons for 1 test failure, got %d", len(result))
	}
}

func TestExtractTestPatterns_NoTestEntries(t *testing.T) {
	entries := []ActionEntry{
		{ActionType: "build_project", Status: "error", Message: "fail"},
	}
	result := extractTestPatterns(entries)
	if len(result) != 0 {
		t.Errorf("expected 0 lessons with no test entries, got %d", len(result))
	}
}

// --- Tests for extractEditPatterns ---

func TestExtractEditPatterns_NoEdits(t *testing.T) {
	entries := []ActionEntry{
		{ActionType: "build_project", Status: "error", Message: "fail"},
	}
	result := extractEditPatterns(entries)
	if len(result) != 0 {
		t.Errorf("expected 0 lessons with no edit entries, got %d", len(result))
	}
}

func TestExtractEditPatterns_SingleEditFailure(t *testing.T) {
	entries := []ActionEntry{
		{ActionType: "apply_patch", Status: "error", Message: "fail", Path: "main.go"},
	}
	result := extractEditPatterns(entries)
	if len(result) != 0 {
		t.Errorf("expected 0 lessons for 1 edit failure, got %d", len(result))
	}
}

func TestExtractEditPatterns_NoPath(t *testing.T) {
	entries := []ActionEntry{
		{ActionType: "apply_patch", Status: "error", Message: "fail", Path: ""},
		{ActionType: "edit_code", Status: "error", Message: "fail", Path: ""},
	}
	result := extractEditPatterns(entries)
	if len(result) != 0 {
		t.Errorf("expected 0 lessons for edit failures without paths, got %d", len(result))
	}
}

func TestExtractEditPatterns_MultipleFiles(t *testing.T) {
	entries := []ActionEntry{
		{ActionType: "apply_patch", Status: "error", Message: "fail", Path: "a.go"},
		{ActionType: "apply_patch", Status: "error", Message: "fail", Path: "a.go"},
		{ActionType: "edit_code", Status: "error", Message: "fail", Path: "b.go"},
		{ActionType: "edit_code", Status: "error", Message: "fail", Path: "b.go"},
	}
	result := extractEditPatterns(entries)
	if len(result) != 2 {
		t.Errorf("expected 2 lessons (one per file), got %d", len(result))
	}
}

// --- Tests for extractTerminalInsight ---

func TestExtractTerminalInsight_AllReasons(t *testing.T) {
	tests := []struct {
		reason   string
		hasTitle string
		isNil    bool
	}{
		{"max_iterations", "max iterations", false},
		{"inner_loop", "stuck in action loop", false},
		{"parse_failures", "unparseable responses", false},
		{"success", "", true},
		{"", "", true},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			result := extractTerminalInsight(tt.reason, 10)
			if tt.isNil {
				if result != nil {
					t.Errorf("expected nil for reason %q, got %+v", tt.reason, result)
				}
			} else {
				if result == nil {
					t.Fatalf("expected non-nil for reason %q", tt.reason)
				}
				if !strings.Contains(result.title, tt.hasTitle) {
					t.Errorf("expected title to contain %q, got %q", tt.hasTitle, result.title)
				}
			}
		})
	}
}

// --- Tests for truncateStr ---

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 3, "hel"},
		{"", 5, ""},
		{"hello world", 5, "hello"},
		{"a", 0, ""},
		{"abc", 1, "a"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q/%d", tt.input, tt.maxLen), func(t *testing.T) {
			got := truncateStr(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// --- Tests for tokenize ---

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  int // expected number of tokens (at least)
	}{
		{"build failure in Go compiler", 3},       // "build", "failure", "compiler" (stop words filtered)
		{"", 0},
		{"a b c", 0},                              // all single chars filtered out
		{"hello_world test123", 2},                 // underscores and digits allowed
		{"THE IS AT ON IN", 0},                     // all stop words
		{"running testing building", 3},            // 3 words, none are stop words
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := tokenize(tt.input)
			if len(result) < tt.want {
				t.Errorf("tokenize(%q) returned %d tokens, want at least %d: %v", tt.input, len(result), tt.want, result)
			}
		})
	}
}

// --- Tests for isStopWord ---

func TestIsStopWord(t *testing.T) {
	// Known stop words
	stopWordList := []string{"the", "is", "at", "on", "in", "to", "for", "of", "and", "or"}
	for _, w := range stopWordList {
		if !isStopWord(w) {
			t.Errorf("expected %q to be a stop word", w)
		}
	}

	// Non-stop words
	nonStopWords := []string{"hello", "compiler", "failure", "build", "test", "xyz"}
	for _, w := range nonStopWords {
		if isStopWord(w) {
			t.Errorf("expected %q to NOT be a stop word", w)
		}
	}
}

// --- Tests for normalize ---

func TestNormalize_ZeroVector(t *testing.T) {
	vec := []float32{0, 0, 0}
	normalize(vec)
	// Should remain all zeros (no division by zero)
	for _, v := range vec {
		if v != 0 {
			t.Errorf("expected 0 after normalizing zero vector, got %f", v)
		}
	}
}

// --- Tests for ProviderEmbedder ---

func TestNewProviderEmbedder(t *testing.T) {
	pe := NewProviderEmbedder("http://localhost:11434/", "test-key", "model-1")
	if pe == nil {
		t.Fatal("NewProviderEmbedder returned nil")
	}
	if pe.endpoint != "http://localhost:11434" {
		t.Errorf("endpoint should have trailing slash trimmed, got %q", pe.endpoint)
	}
	if pe.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want test-key", pe.apiKey)
	}
	if pe.model != "model-1" {
		t.Errorf("model = %q, want model-1", pe.model)
	}
}

// ProviderEmbedder tests that require httptest.NewServer are skipped
// in short mode because the sandbox doesn't allow binding to ports.

func TestProviderEmbedder_Embed_ConnectionRefused(t *testing.T) {
	// Test with an endpoint that can't connect
	pe := NewProviderEmbedder("http://127.0.0.1:1", "test-key", "model-1")
	ctx := context.Background()
	_, err := pe.Embed(ctx, []string{"hello"})
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
	if !strings.Contains(err.Error(), "embedding request failed") {
		t.Errorf("error should mention request failure: %v", err)
	}
}

// --- Tests for FallbackEmbedder with failing primary ---

func TestFallbackEmbedder_PrimaryFails(t *testing.T) {
	fb := NewFallbackEmbedder(&failingEmbedder{})
	ctx := context.Background()
	result, err := fb.Embed(ctx, []string{"test"})
	if err != nil {
		t.Fatalf("FallbackEmbedder should not error when primary fails: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 embedding from fallback, got %d", len(result))
	}
	if len(result[0]) != hashDimensions {
		t.Fatalf("expected %d dimensions from hash fallback, got %d", hashDimensions, len(result[0]))
	}
}

// --- Tests for CosineSimilarity edge cases ---

func TestCosineSimilarity_EmptyVectors(t *testing.T) {
	sim := CosineSimilarity([]float32{}, []float32{})
	if sim != 0 {
		t.Errorf("expected 0 for empty vectors, got %f", sim)
	}
}

func TestCosineSimilarity_BothZero(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{0, 0, 0}
	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("expected 0 for both zero vectors, got %f", sim)
	}
}

// --- Tests for hashEmbed edge cases ---

func TestHashEmbed_EmptyString(t *testing.T) {
	vec := hashEmbed("")
	if len(vec) != hashDimensions {
		t.Fatalf("expected %d dimensions, got %d", hashDimensions, len(vec))
	}
	for _, v := range vec {
		if v != 0 {
			t.Fatal("expected zero vector for empty string")
		}
	}
}

func TestHashEmbed_SingleWord(t *testing.T) {
	vec := hashEmbed("compiler")
	if len(vec) != hashDimensions {
		t.Fatalf("expected %d dimensions, got %d", hashDimensions, len(vec))
	}
	// Should have at least one non-zero element
	hasNonZero := false
	for _, v := range vec {
		if v != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("expected non-zero vector for 'compiler'")
	}
}

func TestHashEmbed_StopWordsOnly(t *testing.T) {
	vec := hashEmbed("the is at on in")
	// All words are stop words, should result in zero vector
	for _, v := range vec {
		if v != 0 {
			t.Fatal("expected zero vector for stop-words-only input")
		}
	}
}
