package persona

import (
	"testing"
)

func TestLoadQAEngineerPersona(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping persona file test in short mode")
	}

	manager := NewManager("../../personas")

	persona, err := manager.LoadPersona("default/qa-engineer")
	if err != nil {
		t.Skipf("Failed to load qa-engineer persona (files may not exist): %v", err)
	}

	if persona.Name != "default/qa-engineer" {
		t.Errorf("Expected name 'default/qa-engineer', got '%s'", persona.Name)
	}

	if len(persona.FocusAreas) == 0 {
		t.Error("Expected focus areas to be populated")
	}

	if persona.Mission == "" {
		t.Error("Expected mission to be populated")
	}
}

func TestLoadProjectManagerPersona(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping persona file test in short mode")
	}

	manager := NewManager("../../personas")

	persona, err := manager.LoadPersona("default/project-manager")
	if err != nil {
		t.Skipf("Failed to load project-manager persona (files may not exist): %v", err)
	}

	if persona.Name != "default/project-manager" {
		t.Errorf("Expected name 'default/project-manager', got '%s'", persona.Name)
	}

	if len(persona.FocusAreas) == 0 {
		t.Error("Expected focus areas to be populated")
	}

	if persona.Mission == "" {
		t.Error("Expected mission to be populated")
	}
}

func TestListPersonas(t *testing.T) {
	manager := NewManager("../../personas")

	personas, err := manager.ListPersonas()
	if err != nil {
		t.Fatalf("Failed to list personas: %v", err)
	}

	// Should have at least the 5 personas we know about
	expectedPersonas := []string{"default/code-reviewer", "default/decision-maker", "default/housekeeping-bot", "default/qa-engineer", "default/project-manager"}

	for _, expected := range expectedPersonas {
		found := false
		for _, persona := range personas {
			if persona == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find persona '%s' in list", expected)
		}
	}
}
