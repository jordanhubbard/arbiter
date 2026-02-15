package models

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNewEntityMetadata tests entity metadata creation
func TestNewEntityMetadata(t *testing.T) {
	version := AgentSchemaVersion
	meta := NewEntityMetadata(version)

	if meta.SchemaVersion != version {
		t.Errorf("SchemaVersion = %v, want %v", meta.SchemaVersion, version)
	}

	if meta.Attributes == nil {
		t.Error("Attributes should be initialized")
	}

	if len(meta.Attributes) != 0 {
		t.Errorf("Attributes length = %d, want 0", len(meta.Attributes))
	}
}

// TestEntityMetadata_GetAttribute tests getting attributes
func TestEntityMetadata_GetAttribute(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)

	// Get from empty attributes
	val, ok := meta.GetAttribute("key1")
	if ok {
		t.Error("Expected ok=false for non-existent key")
	}
	if val != nil {
		t.Errorf("Expected nil value, got %v", val)
	}

	// Set and get attribute
	meta.Attributes["key1"] = "value1"
	meta.Attributes["key2"] = 42

	val, ok = meta.GetAttribute("key1")
	if !ok {
		t.Error("Expected ok=true for existing key")
	}
	if val != "value1" {
		t.Errorf("Value = %v, want %q", val, "value1")
	}

	val, ok = meta.GetAttribute("key2")
	if !ok {
		t.Error("Expected ok=true for existing key")
	}
	if val != 42 {
		t.Errorf("Value = %v, want %d", val, 42)
	}

	// Get from nil attributes
	meta.Attributes = nil
	val, ok = meta.GetAttribute("key1")
	if ok {
		t.Error("Expected ok=false when attributes is nil")
	}
}

// TestEntityMetadata_GetStringAttribute tests getting string attributes
func TestEntityMetadata_GetStringAttribute(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)

	// Get with default (empty attributes)
	val := meta.GetStringAttribute("key1", "default")
	if val != "default" {
		t.Errorf("Value = %q, want %q", val, "default")
	}

	// Set and get
	meta.Attributes["key1"] = "actual"
	val = meta.GetStringAttribute("key1", "default")
	if val != "actual" {
		t.Errorf("Value = %q, want %q", val, "actual")
	}

	// Get non-string value (should return default)
	meta.Attributes["key2"] = 42
	val = meta.GetStringAttribute("key2", "default")
	if val != "default" {
		t.Errorf("Value = %q, want %q (default for non-string)", val, "default")
	}

	// Get from nil attributes
	meta.Attributes = nil
	val = meta.GetStringAttribute("key1", "default")
	if val != "default" {
		t.Errorf("Value = %q, want %q (default for nil attributes)", val, "default")
	}
}

// TestEntityMetadata_GetIntAttribute tests getting int attributes
func TestEntityMetadata_GetIntAttribute(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)

	// Get with default (empty attributes)
	val := meta.GetIntAttribute("key1", 99)
	if val != 99 {
		t.Errorf("Value = %d, want %d", val, 99)
	}

	// Set and get int
	meta.Attributes["key1"] = 42
	val = meta.GetIntAttribute("key1", 99)
	if val != 42 {
		t.Errorf("Value = %d, want %d", val, 42)
	}

	// Set and get int64
	meta.Attributes["key2"] = int64(123)
	val = meta.GetIntAttribute("key2", 99)
	if val != 123 {
		t.Errorf("Value = %d, want %d", val, 123)
	}

	// Set and get float64
	meta.Attributes["key3"] = float64(456.7)
	val = meta.GetIntAttribute("key3", 99)
	if val != 456 {
		t.Errorf("Value = %d, want %d", val, 456)
	}

	// Set and get json.Number
	meta.Attributes["key4"] = json.Number("789")
	val = meta.GetIntAttribute("key4", 99)
	if val != 789 {
		t.Errorf("Value = %d, want %d", val, 789)
	}

	// Get non-int value (should return default)
	meta.Attributes["key5"] = "not an int"
	val = meta.GetIntAttribute("key5", 99)
	if val != 99 {
		t.Errorf("Value = %d, want %d (default for non-int)", val, 99)
	}

	// Get from nil attributes
	meta.Attributes = nil
	val = meta.GetIntAttribute("key1", 99)
	if val != 99 {
		t.Errorf("Value = %d, want %d (default for nil attributes)", val, 99)
	}
}

// TestEntityMetadata_GetBoolAttribute tests getting bool attributes
func TestEntityMetadata_GetBoolAttribute(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)

	// Get with default (empty attributes)
	val := meta.GetBoolAttribute("key1", true)
	if val != true {
		t.Errorf("Value = %v, want %v", val, true)
	}

	// Set and get bool
	meta.Attributes["key1"] = false
	val = meta.GetBoolAttribute("key1", true)
	if val != false {
		t.Errorf("Value = %v, want %v", val, false)
	}

	meta.Attributes["key2"] = true
	val = meta.GetBoolAttribute("key2", false)
	if val != true {
		t.Errorf("Value = %v, want %v", val, true)
	}

	// Get non-bool value (should return default)
	meta.Attributes["key3"] = "not a bool"
	val = meta.GetBoolAttribute("key3", true)
	if val != true {
		t.Errorf("Value = %v, want %v (default for non-bool)", val, true)
	}

	// Get from nil attributes
	meta.Attributes = nil
	val = meta.GetBoolAttribute("key1", true)
	if val != true {
		t.Errorf("Value = %v, want %v (default for nil attributes)", val, true)
	}
}

// TestEntityMetadata_SetAttribute tests setting attributes
func TestEntityMetadata_SetAttribute(t *testing.T) {
	meta := EntityMetadata{}

	// Set on nil attributes (should initialize)
	meta.SetAttribute("key1", "value1")
	if meta.Attributes == nil {
		t.Error("Attributes should be initialized")
	}

	if meta.Attributes["key1"] != "value1" {
		t.Errorf("Attribute value = %v, want %q", meta.Attributes["key1"], "value1")
	}

	// Set another attribute
	meta.SetAttribute("key2", 42)
	if meta.Attributes["key2"] != 42 {
		t.Errorf("Attribute value = %v, want %d", meta.Attributes["key2"], 42)
	}

	// Overwrite existing attribute
	meta.SetAttribute("key1", "new value")
	if meta.Attributes["key1"] != "new value" {
		t.Errorf("Attribute value = %v, want %q", meta.Attributes["key1"], "new value")
	}
}

// TestEntityMetadata_DeleteAttribute tests deleting attributes
func TestEntityMetadata_DeleteAttribute(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)
	meta.Attributes["key1"] = "value1"
	meta.Attributes["key2"] = "value2"

	// Delete existing attribute
	meta.DeleteAttribute("key1")
	if _, ok := meta.Attributes["key1"]; ok {
		t.Error("key1 should be deleted")
	}

	if _, ok := meta.Attributes["key2"]; !ok {
		t.Error("key2 should still exist")
	}

	// Delete non-existent attribute (should not panic)
	meta.DeleteAttribute("nonexistent")

	// Delete from nil attributes (should not panic)
	meta.Attributes = nil
	meta.DeleteAttribute("key1")
}

// TestEntityMetadata_HasAttribute tests checking attribute existence
func TestEntityMetadata_HasAttribute(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)

	// Check non-existent attribute
	if meta.HasAttribute("key1") {
		t.Error("Should return false for non-existent attribute")
	}

	// Set and check
	meta.Attributes["key1"] = "value1"
	if !meta.HasAttribute("key1") {
		t.Error("Should return true for existing attribute")
	}

	// Check with nil attributes
	meta.Attributes = nil
	if meta.HasAttribute("key1") {
		t.Error("Should return false when attributes is nil")
	}
}

// TestEntityMetadata_MergeAttributes tests merging attributes
func TestEntityMetadata_MergeAttributes(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)
	meta.Attributes["key1"] = "value1"
	meta.Attributes["key2"] = "value2"

	// Merge new attributes
	newAttrs := map[string]any{
		"key2": "updated",
		"key3": "new",
	}

	meta.MergeAttributes(newAttrs)

	if meta.Attributes["key1"] != "value1" {
		t.Error("key1 should be unchanged")
	}

	if meta.Attributes["key2"] != "updated" {
		t.Errorf("key2 = %v, want %q (should be updated)", meta.Attributes["key2"], "updated")
	}

	if meta.Attributes["key3"] != "new" {
		t.Errorf("key3 = %v, want %q (should be added)", meta.Attributes["key3"], "new")
	}

	// Merge into nil attributes
	meta.Attributes = nil
	meta.MergeAttributes(newAttrs)
	if meta.Attributes == nil {
		t.Error("Attributes should be initialized")
	}

	if len(meta.Attributes) != 2 {
		t.Errorf("Attributes length = %d, want 2", len(meta.Attributes))
	}
}

// TestEntityMetadata_AttributesJSON tests JSON marshaling of attributes
func TestEntityMetadata_AttributesJSON(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)
	meta.Attributes["key1"] = "value1"
	meta.Attributes["key2"] = 42

	jsonBytes, err := meta.AttributesJSON()
	if err != nil {
		t.Fatalf("AttributesJSON() error = %v", err)
	}

	// Unmarshal to verify
	var attrs map[string]any
	if err := json.Unmarshal(jsonBytes, &attrs); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if attrs["key1"] != "value1" {
		t.Errorf("attrs[key1] = %v, want %q", attrs["key1"], "value1")
	}

	// JSON numbers are float64
	if attrs["key2"] != float64(42) {
		t.Errorf("attrs[key2] = %v, want %f", attrs["key2"], float64(42))
	}
}

// TestEntityMetadata_SetAttributesFromJSON tests setting attributes from JSON
func TestEntityMetadata_SetAttributesFromJSON(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)

	jsonBytes := []byte(`{"key1": "value1", "key2": 42, "key3": true}`)
	err := meta.SetAttributesFromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("SetAttributesFromJSON() error = %v", err)
	}

	if meta.Attributes["key1"] != "value1" {
		t.Errorf("Attributes[key1] = %v, want %q", meta.Attributes["key1"], "value1")
	}

	if meta.Attributes["key2"] != float64(42) {
		t.Errorf("Attributes[key2] = %v, want %f", meta.Attributes["key2"], float64(42))
	}

	if meta.Attributes["key3"] != true {
		t.Errorf("Attributes[key3] = %v, want %v", meta.Attributes["key3"], true)
	}

	// Test invalid JSON
	err = meta.SetAttributesFromJSON([]byte(`{invalid json}`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	// Test empty JSON
	err = meta.SetAttributesFromJSON([]byte(`{}`))
	if err != nil {
		t.Errorf("SetAttributesFromJSON('{}') error = %v", err)
	}

	// Test null JSON
	err = meta.SetAttributesFromJSON([]byte(`null`))
	if err != nil {
		t.Errorf("SetAttributesFromJSON('null') error = %v", err)
	}
}

// TestEntityMetadata_MigratedAtTimestamp tests migration timestamp
func TestEntityMetadata_MigratedAtTimestamp(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)

	// No migration yet
	if meta.MigratedAt != nil {
		t.Error("MigratedAt should be nil initially")
	}

	// Set migration timestamp
	now := time.Now()
	meta.MigratedAt = &now

	if meta.MigratedAt == nil {
		t.Error("MigratedAt should not be nil after setting")
	}

	if !meta.MigratedAt.Equal(now) {
		t.Errorf("MigratedAt = %v, want %v", meta.MigratedAt, now)
	}

	// Set migrated from
	meta.MigratedFrom = SchemaVersion("0.9")
	if meta.MigratedFrom != "0.9" {
		t.Errorf("MigratedFrom = %v, want %q", meta.MigratedFrom, "0.9")
	}
}

// TestEntityMetadata_EmptyAttributesJSON tests AttributesJSON with empty attributes
func TestEntityMetadata_EmptyAttributesJSON(t *testing.T) {
	meta := NewEntityMetadata(AgentSchemaVersion)

	jsonBytes, err := meta.AttributesJSON()
	if err != nil {
		t.Fatalf("AttributesJSON() error = %v", err)
	}

	if string(jsonBytes) != "{}" {
		t.Errorf("Empty attributes JSON = %q, want %q", string(jsonBytes), "{}")
	}
}

// TestSchemaVersionConstants tests schema version constants
func TestSchemaVersionConstants(t *testing.T) {
	versions := map[string]SchemaVersion{
		"Agent":    AgentSchemaVersion,
		"Project":  ProjectSchemaVersion,
		"Provider": ProviderSchemaVersion,
		"OrgChart": OrgChartSchemaVersion,
		"Position": PositionSchemaVersion,
		"Persona":  PersonaSchemaVersion,
		"Bead":     BeadSchemaVersion,
	}

	for name, version := range versions {
		if version == "" {
			t.Errorf("%s schema version should not be empty", name)
		}
	}
}

// TestEntityTypeConstants tests entity type constants
func TestEntityTypeConstants(t *testing.T) {
	types := map[string]EntityType{
		"Agent":    EntityTypeAgent,
		"Project":  EntityTypeProject,
		"Provider": EntityTypeProvider,
		"OrgChart": EntityTypeOrgChart,
		"Position": EntityTypePosition,
		"Persona":  EntityTypePersona,
		"Bead":     EntityTypeBead,
	}

	for name, entityType := range types {
		if entityType == "" {
			t.Errorf("%s entity type should not be empty", name)
		}
	}
}
