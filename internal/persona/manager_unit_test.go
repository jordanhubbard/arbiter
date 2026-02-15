package persona

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

func createTestSkillMd(t *testing.T, dir, name, content string) {
	t.Helper()
	personaDir := filepath.Join(dir, name)
	if err := os.MkdirAll(personaDir, 0755); err != nil {
		t.Fatalf("failed to create persona dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(personaDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}
}

const validSkillMd = `---
name: test-persona
description: A test persona for unit testing
license: MIT
compatibility: go 1.21+
metadata:
  autonomy_level: full
  specialties:
    - testing
    - validation
---

## Instructions

This is the body of the skill file with instructions for the agent.

## Focus Areas

- Unit testing
- Integration testing
`

func TestNewManager(t *testing.T) {
	m := NewManager("/tmp/personas")
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.personaDir != "/tmp/personas" {
		t.Errorf("personaDir = %q, want /tmp/personas", m.personaDir)
	}
	if m.personas == nil {
		t.Error("personas map should be initialized")
	}
}

func TestLoadPersona_ValidSkillMd(t *testing.T) {
	tmpDir := t.TempDir()
	createTestSkillMd(t, tmpDir, "test-agent", validSkillMd)

	m := NewManager(tmpDir)
	persona, err := m.LoadPersona("test-agent")
	if err != nil {
		t.Fatalf("LoadPersona() error = %v", err)
	}

	if persona.Name != "test-agent" {
		t.Errorf("Name = %q, want test-agent", persona.Name)
	}
	if persona.Description != "A test persona for unit testing" {
		t.Errorf("Description = %q", persona.Description)
	}
	if persona.License != "MIT" {
		t.Errorf("License = %q, want MIT", persona.License)
	}
	if persona.AutonomyLevel != "full" {
		t.Errorf("AutonomyLevel = %q, want full", persona.AutonomyLevel)
	}
	if len(persona.FocusAreas) != 2 {
		t.Errorf("FocusAreas length = %d, want 2", len(persona.FocusAreas))
	}
	if persona.Instructions == "" {
		t.Error("Instructions should not be empty")
	}
}

func TestLoadPersona_CachesResult(t *testing.T) {
	tmpDir := t.TempDir()
	createTestSkillMd(t, tmpDir, "cached-agent", validSkillMd)

	m := NewManager(tmpDir)
	p1, err := m.LoadPersona("cached-agent")
	if err != nil {
		t.Fatalf("First LoadPersona() error = %v", err)
	}

	// Second call should return cached version
	p2, err := m.LoadPersona("cached-agent")
	if err != nil {
		t.Fatalf("Second LoadPersona() error = %v", err)
	}

	if p1 != p2 {
		t.Error("expected same pointer from cache")
	}
}

func TestLoadPersona_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	_, err := m.LoadPersona("nonexistent")
	if err == nil {
		t.Error("expected error for missing persona")
	}
}

func TestLoadPersona_MalformedFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "no-frontmatter",
			content: "Just some text without frontmatter",
		},
		{
			name:    "no-closing-delimiter",
			content: "---\nname: test\ndescription: test\nno closing delimiter here",
		},
		{
			name:    "missing-name",
			content: "---\ndescription: test\n---\nbody",
		},
		{
			name:    "missing-description",
			content: "---\nname: test\n---\nbody",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createTestSkillMd(t, tmpDir, tt.name, tt.content)
			m := NewManager(tmpDir)
			_, err := m.LoadPersona(tt.name)
			if err == nil {
				t.Error("expected error for malformed SKILL.md")
			}
		})
	}
}

func TestLoadPersona_DefaultAutonomy(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
name: simple-agent
description: Simple agent without metadata
---

Body content here.
`
	createTestSkillMd(t, tmpDir, "simple", content)

	m := NewManager(tmpDir)
	persona, err := m.LoadPersona("simple")
	if err != nil {
		t.Fatalf("LoadPersona() error = %v", err)
	}

	if persona.AutonomyLevel != "semi" {
		t.Errorf("AutonomyLevel = %q, want semi (default)", persona.AutonomyLevel)
	}
}

func TestInvalidateCache(t *testing.T) {
	tmpDir := t.TempDir()
	createTestSkillMd(t, tmpDir, "cache-test", validSkillMd)

	m := NewManager(tmpDir)
	_, err := m.LoadPersona("cache-test")
	if err != nil {
		t.Fatalf("LoadPersona() error = %v", err)
	}

	// Should be cached
	if _, ok := m.personas["cache-test"]; !ok {
		t.Error("expected persona to be in cache")
	}

	// Invalidate
	m.InvalidateCache("cache-test")

	if _, ok := m.personas["cache-test"]; ok {
		t.Error("expected persona to be removed from cache")
	}
}

func TestListPersonas_TempDir(t *testing.T) {
	tmpDir := t.TempDir()
	createTestSkillMd(t, tmpDir, "agent-a", validSkillMd)
	createTestSkillMd(t, tmpDir, "agent-b", validSkillMd)

	m := NewManager(tmpDir)
	personas, err := m.ListPersonas()
	if err != nil {
		t.Fatalf("ListPersonas() error = %v", err)
	}

	if len(personas) != 2 {
		t.Errorf("len(personas) = %d, want 2", len(personas))
	}
}

func TestListPersonas_NonexistentDir(t *testing.T) {
	m := NewManager("/nonexistent/path/to/personas")
	personas, err := m.ListPersonas()
	if err != nil {
		t.Fatalf("ListPersonas() error = %v", err)
	}
	if len(personas) != 0 {
		t.Errorf("expected empty list, got %d personas", len(personas))
	}
}

func TestListPersonas_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	personas, err := m.ListPersonas()
	if err != nil {
		t.Fatalf("ListPersonas() error = %v", err)
	}
	if len(personas) != 0 {
		t.Errorf("expected empty list, got %d personas", len(personas))
	}
}

func TestListPersonas_SkipsDirsWithoutSkillMd(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a dir without SKILL.md
	os.MkdirAll(filepath.Join(tmpDir, "no-skill"), 0755)
	// Create one with SKILL.md
	createTestSkillMd(t, tmpDir, "has-skill", validSkillMd)

	m := NewManager(tmpDir)
	personas, err := m.ListPersonas()
	if err != nil {
		t.Fatalf("ListPersonas() error = %v", err)
	}
	if len(personas) != 1 {
		t.Errorf("len(personas) = %d, want 1", len(personas))
	}
	if len(personas) > 0 && personas[0] != "has-skill" {
		t.Errorf("personas[0] = %q, want has-skill", personas[0])
	}
}

func TestClonePersona(t *testing.T) {
	tmpDir := t.TempDir()
	createTestSkillMd(t, tmpDir, "source", validSkillMd)

	m := NewManager(tmpDir)
	persona, err := m.ClonePersona("source", "destination")
	if err != nil {
		t.Fatalf("ClonePersona() error = %v", err)
	}
	if persona == nil {
		t.Fatal("expected non-nil persona")
	}

	// Verify destination SKILL.md exists
	destSkill := filepath.Join(tmpDir, "destination", "SKILL.md")
	if _, err := os.Stat(destSkill); os.IsNotExist(err) {
		t.Error("destination SKILL.md not created")
	}
}

func TestClonePersona_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	createTestSkillMd(t, tmpDir, "source", validSkillMd)

	m := NewManager(tmpDir)

	tests := []struct {
		name     string
		source   string
		dest     string
	}{
		{"empty-source", "", "dest"},
		{"empty-dest", "source", ""},
		{"absolute-dest", "source", "/absolute/path"},
		{"traversal-dest", "source", "../escape"},
		{"dot-dest", "source", "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.ClonePersona(tt.source, tt.dest)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestClonePersona_DestExists(t *testing.T) {
	tmpDir := t.TempDir()
	createTestSkillMd(t, tmpDir, "source", validSkillMd)
	createTestSkillMd(t, tmpDir, "existing", validSkillMd)

	m := NewManager(tmpDir)
	_, err := m.ClonePersona("source", "existing")
	if err == nil {
		t.Error("expected error when destination exists")
	}
}

func TestSavePersona_NotImplemented(t *testing.T) {
	m := NewManager(t.TempDir())
	err := m.SavePersona(nil)
	if err == nil {
		t.Error("expected error from SavePersona (not implemented)")
	}
}

func TestParseSkillMd(t *testing.T) {
	m := NewManager(t.TempDir())

	fm, body, err := m.parseSkillMd(validSkillMd)
	if err != nil {
		t.Fatalf("parseSkillMd() error = %v", err)
	}
	if fm.Name != "test-persona" {
		t.Errorf("Name = %q", fm.Name)
	}
	if fm.Description != "A test persona for unit testing" {
		t.Errorf("Description = %q", fm.Description)
	}
	if body == "" {
		t.Error("body should not be empty")
	}
}

func TestParseSections(t *testing.T) {
	m := NewManager(t.TempDir())

	content := `## Section One

Content for section one.

## Section Two

Content for section two.
More content.
`
	sections := m.parseSections(content)
	if len(sections) != 2 {
		t.Errorf("len(sections) = %d, want 2", len(sections))
	}
	if _, ok := sections["Section One"]; !ok {
		t.Error("missing Section One")
	}
	if _, ok := sections["Section Two"]; !ok {
		t.Error("missing Section Two")
	}
}

func TestParseSections_Empty(t *testing.T) {
	m := NewManager(t.TempDir())
	sections := m.parseSections("")
	if len(sections) != 0 {
		t.Errorf("expected empty sections, got %d", len(sections))
	}
}

func TestParseList(t *testing.T) {
	m := NewManager(t.TempDir())

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{"bullet-dash", "- item1\n- item2\n- item3", 3},
		{"bullet-star", "* item1\n* item2", 2},
		{"numbered", "1. first\n2. second", 2},
		{"mixed", "- bullet\n1. numbered", 2},
		{"empty", "", 0},
		{"no-list", "just some text\nmore text", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := m.parseList(tt.content)
			if len(items) != tt.want {
				t.Errorf("len(items) = %d, want %d", len(items), tt.want)
			}
		})
	}
}

func TestExtractAutonomyLevel(t *testing.T) {
	m := NewManager(t.TempDir())

	tests := []struct {
		input string
		want  string
	}{
		{"Full autonomy granted", "full"},
		{"Semi-autonomous mode", "semi"},
		{"Supervised by human", "supervised"},
		{"FULL AUTONOMY", "full"},
		{"unknown text", "semi"},
		{"", "semi"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := m.extractAutonomyLevel(tt.input)
			if got != tt.want {
				t.Errorf("extractAutonomyLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParsePersonaFile(t *testing.T) {
	m := NewManager(t.TempDir())
	content := `## Character

A brave and helpful agent.

## Tone

Professional and friendly.

## Autonomy Level

Full autonomy.

## Decision Making

Decides quickly.

## Persistence & Housekeeping

Keeps things tidy.

## Collaboration

Works well with others.

## Focus Areas

- Testing
- Validation

## Capabilities

- Write code
- Run tests

## Standards & Conventions

- Follow Go conventions
`

	persona := &models.Persona{}
	m.parsePersonaFile(persona, content)

	if persona.Character != "A brave and helpful agent." {
		t.Errorf("Character = %q", persona.Character)
	}
	if persona.Tone != "Professional and friendly." {
		t.Errorf("Tone = %q", persona.Tone)
	}
	if persona.AutonomyLevel != "full" {
		t.Errorf("AutonomyLevel = %q", persona.AutonomyLevel)
	}
	if persona.DecisionMaking != "Decides quickly." {
		t.Errorf("DecisionMaking = %q", persona.DecisionMaking)
	}
	if persona.Housekeeping != "Keeps things tidy." {
		t.Errorf("Housekeeping = %q", persona.Housekeeping)
	}
	if persona.Collaboration != "Works well with others." {
		t.Errorf("Collaboration = %q", persona.Collaboration)
	}
	if len(persona.FocusAreas) != 2 {
		t.Errorf("FocusAreas = %v", persona.FocusAreas)
	}
	if len(persona.Capabilities) != 2 {
		t.Errorf("Capabilities = %v", persona.Capabilities)
	}
	if len(persona.Standards) != 1 {
		t.Errorf("Standards = %v", persona.Standards)
	}
}

func TestParseInstructionsFile(t *testing.T) {
	m := NewManager(t.TempDir())
	content := `## Your Mission

Build great software.

## Your Personality

Friendly and focused.

## Your Autonomy

You have full autonomy.

## Decision Points

Escalate P0 decisions.

## Persistent Tasks

Run daily checks.
`

	persona := &models.Persona{}
	m.parseInstructionsFile(persona, content)

	if persona.Mission != "Build great software." {
		t.Errorf("Mission = %q", persona.Mission)
	}
	if persona.Personality != "Friendly and focused." {
		t.Errorf("Personality = %q", persona.Personality)
	}
	if persona.AutonomyInstructions != "You have full autonomy." {
		t.Errorf("AutonomyInstructions = %q", persona.AutonomyInstructions)
	}
	if persona.DecisionInstructions != "Escalate P0 decisions." {
		t.Errorf("DecisionInstructions = %q", persona.DecisionInstructions)
	}
	if persona.PersistentTasks != "Run daily checks." {
		t.Errorf("PersistentTasks = %q", persona.PersistentTasks)
	}
}

func TestGeneratePersonaContent(t *testing.T) {
	m := NewManager(t.TempDir())
	persona := &models.Persona{
		Name:           "test-agent",
		Character:      "Helpful",
		Tone:           "Professional",
		FocusAreas:     []string{"Testing", "Code review"},
		AutonomyLevel:  "full",
		Capabilities:   []string{"Write code"},
		DecisionMaking: "Quick decisions",
		Housekeeping:   "Keep logs",
		Collaboration:  "Pair programming",
		Standards:      []string{"Follow best practices"},
	}

	content := m.generatePersonaContent(persona)
	if content == "" {
		t.Error("expected non-empty content")
	}
	if !contains(content, "test-agent") {
		t.Error("content should contain persona name")
	}
	if !contains(content, "Helpful") {
		t.Error("content should contain character")
	}
	if !contains(content, "Testing") {
		t.Error("content should contain focus areas")
	}
}

func TestGenerateInstructionsContent(t *testing.T) {
	m := NewManager(t.TempDir())
	persona := &models.Persona{
		Name:                 "test-agent",
		Mission:              "Build stuff",
		Personality:          "Friendly",
		AutonomyInstructions: "Full access",
		DecisionInstructions: "Escalate P0",
		PersistentTasks:      "Daily review",
	}

	content := m.generateInstructionsContent(persona)
	if content == "" {
		t.Error("expected non-empty content")
	}
	if !contains(content, "Build stuff") {
		t.Error("content should contain mission")
	}
	if !contains(content, "Friendly") {
		t.Error("content should contain personality")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
