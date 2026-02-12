package persona

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

// Manager handles persona loading, saving, and live editing
type Manager struct {
	personaDir string
	personas   map[string]*models.Persona
}

// NewManager creates a new persona manager
func NewManager(personaDir string) *Manager {
	return &Manager{
		personaDir: personaDir,
		personas:   make(map[string]*models.Persona),
	}
}

// LoadPersona loads a persona from a directory
func (m *Manager) LoadPersona(name string) (*models.Persona, error) {
	personaPath := filepath.Join(m.personaDir, name)

	// Check if cached
	if persona, ok := m.personas[name]; ok {
		return persona, nil
	}

	// Load PERSONA.md
	personaFile := filepath.Join(personaPath, "PERSONA.md")
	personaContent, err := os.ReadFile(personaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read PERSONA.md: %w", err)
	}

	// Load AI_START_HERE.md
	instructionsFile := filepath.Join(personaPath, "AI_START_HERE.md")
	instructionsContent, err := os.ReadFile(instructionsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read AI_START_HERE.md: %w", err)
	}

	// Parse the markdown files (basic parsing for now)
	persona := &models.Persona{
		Name:             name,
		PersonaFile:      personaFile,
		InstructionsFile: instructionsFile,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Parse PERSONA.md sections
	m.parsePersonaFile(persona, string(personaContent))

	// Parse AI_START_HERE.md sections
	m.parseInstructionsFile(persona, string(instructionsContent))

	// Default autonomy to "semi" if not defined in persona file
	if persona.AutonomyLevel == "" {
		persona.AutonomyLevel = string(models.AutonomySemi)
	}

	// Cache it
	m.personas[name] = persona

	return persona, nil
}

// parsePersonaFile parses PERSONA.md content
func (m *Manager) parsePersonaFile(persona *models.Persona, content string) {
	sections := m.parseSections(content)

	if val, ok := sections["Character"]; ok {
		persona.Character = val
	}
	if val, ok := sections["Tone"]; ok {
		persona.Tone = val
	}
	if val, ok := sections["Autonomy Level"]; ok {
		persona.AutonomyLevel = m.extractAutonomyLevel(val)
	}
	if val, ok := sections["Decision Making"]; ok {
		persona.DecisionMaking = val
	}
	if val, ok := sections["Persistence & Housekeeping"]; ok {
		persona.Housekeeping = val
	}
	if val, ok := sections["Collaboration"]; ok {
		persona.Collaboration = val
	}

	// Parse lists
	if val, ok := sections["Focus Areas"]; ok {
		persona.FocusAreas = m.parseList(val)
	}
	if val, ok := sections["Capabilities"]; ok {
		persona.Capabilities = m.parseList(val)
	}
	if val, ok := sections["Standards & Conventions"]; ok {
		persona.Standards = m.parseList(val)
	}
}

// parseInstructionsFile parses AI_START_HERE.md content
func (m *Manager) parseInstructionsFile(persona *models.Persona, content string) {
	sections := m.parseSections(content)

	if val, ok := sections["Your Mission"]; ok {
		persona.Mission = val
	}
	if val, ok := sections["Your Personality"]; ok {
		persona.Personality = val
	}
	if val, ok := sections["Your Autonomy"]; ok {
		persona.AutonomyInstructions = val
	}
	if val, ok := sections["Decision Points"]; ok {
		persona.DecisionInstructions = val
	}
	if val, ok := sections["Persistent Tasks"]; ok {
		persona.PersistentTasks = val
	}
}

// parseSections splits markdown into sections by headers
func (m *Manager) parseSections(content string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(content, "\n")

	var currentSection string
	var currentContent []string

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// Save previous section
			if currentSection != "" {
				sections[currentSection] = strings.TrimSpace(strings.Join(currentContent, "\n"))
			}
			// Start new section
			currentSection = strings.TrimPrefix(line, "## ")
			currentContent = []string{}
		} else if currentSection != "" {
			currentContent = append(currentContent, line)
		}
	}

	// Save last section
	if currentSection != "" {
		sections[currentSection] = strings.TrimSpace(strings.Join(currentContent, "\n"))
	}

	return sections
}

// parseList parses a bulleted or numbered list
func (m *Manager) parseList(content string) []string {
	var items []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			item := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			items = append(items, strings.TrimSpace(item))
		} else if len(line) > 2 && line[0] >= '0' && line[0] <= '9' && line[1] == '.' {
			item := line[2:]
			items = append(items, strings.TrimSpace(item))
		}
	}

	return items
}

// extractAutonomyLevel extracts the autonomy level from text
func (m *Manager) extractAutonomyLevel(content string) string {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "full") {
		return string(models.AutonomyFull)
	} else if strings.Contains(lower, "semi") {
		return string(models.AutonomySemi)
	} else if strings.Contains(lower, "supervised") {
		return string(models.AutonomySupervised)
	}
	return string(models.AutonomySemi) // default
}

// SavePersona saves a persona back to disk
func (m *Manager) SavePersona(persona *models.Persona) error {
	// Generate PERSONA.md content
	personaContent := m.generatePersonaContent(persona)
	if err := os.WriteFile(persona.PersonaFile, []byte(personaContent), 0644); err != nil {
		return fmt.Errorf("failed to write PERSONA.md: %w", err)
	}

	// Generate AI_START_HERE.md content
	instructionsContent := m.generateInstructionsContent(persona)
	if err := os.WriteFile(persona.InstructionsFile, []byte(instructionsContent), 0644); err != nil {
		return fmt.Errorf("failed to write AI_START_HERE.md: %w", err)
	}

	persona.UpdatedAt = time.Now()
	m.personas[persona.Name] = persona

	return nil
}

// generatePersonaContent generates PERSONA.md content from a persona
func (m *Manager) generatePersonaContent(p *models.Persona) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s - Agent Persona\n\n", p.Name))

	if p.Character != "" {
		sb.WriteString("## Character\n\n")
		sb.WriteString(p.Character)
		sb.WriteString("\n\n")
	}

	if p.Tone != "" {
		sb.WriteString("## Tone\n\n")
		sb.WriteString(p.Tone)
		sb.WriteString("\n\n")
	}

	if len(p.FocusAreas) > 0 {
		sb.WriteString("## Focus Areas\n\n")
		for i, area := range p.FocusAreas {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, area))
		}
		sb.WriteString("\n")
	}

	if p.AutonomyLevel != "" {
		sb.WriteString("## Autonomy Level\n\n")
		sb.WriteString(fmt.Sprintf("**Level:** %s\n\n", p.AutonomyLevel))
	}

	if len(p.Capabilities) > 0 {
		sb.WriteString("## Capabilities\n\n")
		for _, cap := range p.Capabilities {
			sb.WriteString(fmt.Sprintf("- %s\n", cap))
		}
		sb.WriteString("\n")
	}

	if p.DecisionMaking != "" {
		sb.WriteString("## Decision Making\n\n")
		sb.WriteString(p.DecisionMaking)
		sb.WriteString("\n\n")
	}

	if p.Housekeeping != "" {
		sb.WriteString("## Persistence & Housekeeping\n\n")
		sb.WriteString(p.Housekeeping)
		sb.WriteString("\n\n")
	}

	if p.Collaboration != "" {
		sb.WriteString("## Collaboration\n\n")
		sb.WriteString(p.Collaboration)
		sb.WriteString("\n\n")
	}

	if len(p.Standards) > 0 {
		sb.WriteString("## Standards & Conventions\n\n")
		for _, std := range p.Standards {
			sb.WriteString(fmt.Sprintf("- %s\n", std))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateInstructionsContent generates AI_START_HERE.md content
func (m *Manager) generateInstructionsContent(p *models.Persona) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s - Agent Instructions\n\n", p.Name))

	sb.WriteString("## Your Identity\n\n")
	sb.WriteString(fmt.Sprintf("You are **%s**, an autonomous agent working within the Loom orchestration system.\n\n", p.Name))

	if p.Mission != "" {
		sb.WriteString("## Your Mission\n\n")
		sb.WriteString(p.Mission)
		sb.WriteString("\n\n")
	}

	if p.Personality != "" {
		sb.WriteString("## Your Personality\n\n")
		sb.WriteString(p.Personality)
		sb.WriteString("\n\n")
	}

	if p.AutonomyInstructions != "" {
		sb.WriteString("## Your Autonomy\n\n")
		sb.WriteString(p.AutonomyInstructions)
		sb.WriteString("\n\n")
	}

	if p.DecisionInstructions != "" {
		sb.WriteString("## Decision Points\n\n")
		sb.WriteString(p.DecisionInstructions)
		sb.WriteString("\n\n")
	}

	if p.PersistentTasks != "" {
		sb.WriteString("## Persistent Tasks\n\n")
		sb.WriteString(p.PersistentTasks)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// ListPersonas returns all available persona names
func (m *Manager) ListPersonas() ([]string, error) {
	// Check if persona directory exists
	if _, err := os.Stat(m.personaDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	var personas []string
	err := filepath.WalkDir(m.personaDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // Skip errors, continue walking
		}
		if !d.IsDir() {
			return nil
		}
		if path == m.personaDir {
			return nil
		}

		personaFile := filepath.Join(path, "PERSONA.md")
		instructionsFile := filepath.Join(path, "AI_START_HERE.md")
		if _, err := os.Stat(personaFile); err != nil {
			return nil
		}
		if _, err := os.Stat(instructionsFile); err != nil {
			return nil
		}

		rel, err := filepath.Rel(m.personaDir, path)
		if err != nil {
			return err
		}
		personas = append(personas, filepath.ToSlash(rel))
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(personas)
	return personas, nil
}

// ClonePersona copies a persona directory to a new location under the persona root.
func (m *Manager) ClonePersona(sourceName, destName string) (*models.Persona, error) {
	if sourceName == "" || destName == "" {
		return nil, errors.New("source and destination persona names are required")
	}
	if filepath.IsAbs(destName) || strings.Contains(destName, "..") {
		return nil, errors.New("invalid destination persona name")
	}
	cleanDest := filepath.Clean(destName)
	if cleanDest == "." {
		return nil, errors.New("invalid destination persona name")
	}

	sourcePath := filepath.Join(m.personaDir, filepath.FromSlash(sourceName))
	destPath := filepath.Join(m.personaDir, filepath.FromSlash(cleanDest))

	if _, err := os.Stat(destPath); err == nil {
		return nil, fmt.Errorf("destination persona already exists: %s", destName)
	}

	if err := copyDir(sourcePath, destPath); err != nil {
		return nil, err
	}

	name := filepath.ToSlash(cleanDest)
	persona, err := m.LoadPersona(name)
	if err != nil {
		return nil, err
	}
	return persona, nil
}

func copyDir(source, dest string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source is not a directory: %s", source)
	}

	if err := os.MkdirAll(dest, info.Mode()); err != nil {
		return err
	}

	return filepath.WalkDir(source, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})
}

func copyFile(source, dest string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	info, err := input.Stat()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	output, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, input)
	return err
}

// InvalidateCache removes a persona from cache, forcing reload
func (m *Manager) InvalidateCache(name string) {
	delete(m.personas, name)
}
