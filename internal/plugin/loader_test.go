package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jordanhubbard/agenticorp/pkg/plugin"
)

func TestLoadManifest(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create test manifest
	manifest := &PluginManifest{
		Type:     "http",
		Endpoint: "http://localhost:8090",
		Metadata: &plugin.Metadata{
			Name:             "Test Plugin",
			Version:          "1.0.0",
			PluginAPIVersion: plugin.PluginVersion,
			ProviderType:     "test-provider",
			Description:      "A test plugin",
			Author:           "Test Author",
			Capabilities: plugin.Capabilities{
				Streaming: true,
			},
		},
		AutoStart:           true,
		HealthCheckInterval: 60,
	}

	// Save as JSON
	jsonPath := filepath.Join(tmpDir, "plugin.json")
	err := SaveManifest(manifest, jsonPath)
	if err != nil {
		t.Fatalf("Failed to save JSON manifest: %v", err)
	}

	// Load JSON
	loader := NewLoader(tmpDir)
	loadedJSON, err := loader.loadManifest(jsonPath)
	if err != nil {
		t.Fatalf("Failed to load JSON manifest: %v", err)
	}

	if loadedJSON.Metadata.Name != "Test Plugin" {
		t.Errorf("Expected name 'Test Plugin', got '%s'", loadedJSON.Metadata.Name)
	}

	if loadedJSON.Type != "http" {
		t.Errorf("Expected type 'http', got '%s'", loadedJSON.Type)
	}

	// Save as YAML
	yamlPath := filepath.Join(tmpDir, "plugin.yaml")
	err = SaveManifest(manifest, yamlPath)
	if err != nil {
		t.Fatalf("Failed to save YAML manifest: %v", err)
	}

	// Load YAML
	loadedYAML, err := loader.loadManifest(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load YAML manifest: %v", err)
	}

	if loadedYAML.Metadata.Name != "Test Plugin" {
		t.Errorf("Expected name 'Test Plugin', got '%s'", loadedYAML.Metadata.Name)
	}
}

func TestDiscoverPlugins(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create multiple plugin manifests
	for i := 1; i <= 3; i++ {
		manifest := &PluginManifest{
			Type:     "http",
			Endpoint: "http://localhost:8090",
			Metadata: &plugin.Metadata{
				Name:             "Plugin " + string(rune('0'+i)),
				Version:          "1.0.0",
				PluginAPIVersion: plugin.PluginVersion,
				ProviderType:     "provider-" + string(rune('0'+i)),
				Description:      "Test plugin",
			},
			AutoStart: true,
		}

		dir := filepath.Join(tmpDir, "plugin"+string(rune('0'+i)))
		path := filepath.Join(dir, "plugin.yaml")
		if err := SaveManifest(manifest, path); err != nil {
			t.Fatalf("Failed to save manifest %d: %v", i, err)
		}
	}

	// Discover plugins
	loader := NewLoader(tmpDir)
	ctx := context.Background()
	manifests, err := loader.DiscoverPlugins(ctx)
	if err != nil {
		t.Fatalf("Failed to discover plugins: %v", err)
	}

	if len(manifests) != 3 {
		t.Errorf("Expected 3 plugins, found %d", len(manifests))
	}
}

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name    string
		manifest *PluginManifest
		wantErr bool
	}{
		{
			name: "valid manifest",
			manifest: &PluginManifest{
				Type:     "http",
				Endpoint: "http://localhost:8090",
				Metadata: &plugin.Metadata{
					Name:             "Valid Plugin",
					Version:          "1.0.0",
					PluginAPIVersion: plugin.PluginVersion,
					ProviderType:     "valid-provider",
				},
			},
			wantErr: false,
		},
		{
			name: "missing metadata",
			manifest: &PluginManifest{
				Type:     "http",
				Endpoint: "http://localhost:8090",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			manifest: &PluginManifest{
				Type:     "http",
				Endpoint: "http://localhost:8090",
				Metadata: &plugin.Metadata{
					Version:      "1.0.0",
					ProviderType: "test",
				},
			},
			wantErr: true,
		},
		{
			name: "missing version",
			manifest: &PluginManifest{
				Type:     "http",
				Endpoint: "http://localhost:8090",
				Metadata: &plugin.Metadata{
					Name:         "Test",
					ProviderType: "test",
				},
			},
			wantErr: true,
		},
		{
			name: "missing endpoint for http",
			manifest: &PluginManifest{
				Type: "http",
				Metadata: &plugin.Metadata{
					Name:         "Test",
					Version:      "1.0.0",
					ProviderType: "test",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			manifest: &PluginManifest{
				Type:     "invalid",
				Endpoint: "http://localhost:8090",
				Metadata: &plugin.Metadata{
					Name:         "Test",
					Version:      "1.0.0",
					ProviderType: "test",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateManifest(tt.manifest)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateExampleManifest(t *testing.T) {
	tmpDir := t.TempDir()

	err := CreateExampleManifest(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create example manifest: %v", err)
	}

	// Verify file was created
	path := filepath.Join(tmpDir, "example", "plugin.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Example manifest file was not created")
	}

	// Load and validate
	loader := NewLoader(tmpDir)
	manifest, err := loader.loadManifest(path)
	if err != nil {
		t.Fatalf("Failed to load example manifest: %v", err)
	}

	if manifest.Metadata.Name != "Example Plugin" {
		t.Errorf("Expected name 'Example Plugin', got '%s'", manifest.Metadata.Name)
	}

	if len(manifest.Metadata.ConfigSchema) == 0 {
		t.Error("Example manifest should have config schema")
	}
}

func TestLoaderLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	// Test empty directory
	ctx := context.Background()
	manifests, err := loader.DiscoverPlugins(ctx)
	if err != nil {
		t.Fatalf("DiscoverPlugins failed: %v", err)
	}

	if len(manifests) != 0 {
		t.Errorf("Expected 0 plugins in empty directory, found %d", len(manifests))
	}

	// Test ListPlugins on empty loader
	plugins := loader.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("Expected 0 loaded plugins, found %d", len(plugins))
	}

	// Test GetPlugin on empty loader
	_, err = loader.GetPlugin("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent plugin")
	}
}
