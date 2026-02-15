package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/plugin"
)

// --- Loader tests ---

func TestNewLoader(t *testing.T) {
	loader := NewLoader("/tmp/plugins")
	if loader == nil {
		t.Fatal("Expected non-nil loader")
	}
	if loader.pluginsDir != "/tmp/plugins" {
		t.Errorf("Expected pluginsDir '/tmp/plugins', got '%s'", loader.pluginsDir)
	}
	if loader.plugins == nil {
		t.Error("Expected non-nil plugins map")
	}
}

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

func TestLoadManifest_MissingMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	data := []byte(`{"type": "http", "endpoint": "http://localhost:8090"}`)
	path := filepath.Join(tmpDir, "plugin.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	_, err := loader.loadManifest(path)
	if err == nil {
		t.Fatal("Expected error for missing metadata")
	}
	if !strings.Contains(err.Error(), "missing metadata") {
		t.Errorf("Expected 'missing metadata' error, got: %v", err)
	}
}

func TestLoadManifest_MissingName(t *testing.T) {
	tmpDir := t.TempDir()
	data := []byte(`{"type": "http", "endpoint": "http://localhost:8090", "metadata": {"provider_type": "test", "version": "1.0.0"}}`)
	path := filepath.Join(tmpDir, "plugin.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	_, err := loader.loadManifest(path)
	if err == nil {
		t.Fatal("Expected error for missing name")
	}
	if !strings.Contains(err.Error(), "missing name") {
		t.Errorf("Expected 'missing name' error, got: %v", err)
	}
}

func TestLoadManifest_MissingProviderType(t *testing.T) {
	tmpDir := t.TempDir()
	data := []byte(`{"type": "http", "endpoint": "http://localhost:8090", "metadata": {"name": "test", "version": "1.0.0"}}`)
	path := filepath.Join(tmpDir, "plugin.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	_, err := loader.loadManifest(path)
	if err == nil {
		t.Fatal("Expected error for missing provider_type")
	}
	if !strings.Contains(err.Error(), "missing provider_type") {
		t.Errorf("Expected 'missing provider_type' error, got: %v", err)
	}
}

func TestLoadManifest_MissingType(t *testing.T) {
	tmpDir := t.TempDir()
	data := []byte(`{"metadata": {"name": "test", "provider_type": "tp", "version": "1.0.0"}}`)
	path := filepath.Join(tmpDir, "plugin.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	_, err := loader.loadManifest(path)
	if err == nil {
		t.Fatal("Expected error for missing type")
	}
	if !strings.Contains(err.Error(), "missing type") {
		t.Errorf("Expected 'missing type' error, got: %v", err)
	}
}

func TestLoadManifest_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "plugin.json")
	if err := os.WriteFile(path, []byte("not valid json{"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	_, err := loader.loadManifest(path)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestLoadManifest_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "plugin.yaml")
	if err := os.WriteFile(path, []byte(":\n  invalid:\nyaml: ["), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	_, err := loader.loadManifest(path)
	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}
}

func TestLoadManifest_FileNotFound(t *testing.T) {
	loader := NewLoader("/tmp")
	_, err := loader.loadManifest("/nonexistent/plugin.json")
	if err == nil {
		t.Fatal("Expected error for non-existent file")
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

func TestDiscoverPlugins_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)
	ctx := context.Background()
	manifests, err := loader.DiscoverPlugins(ctx)
	if err != nil {
		t.Fatalf("DiscoverPlugins failed: %v", err)
	}
	if len(manifests) != 0 {
		t.Errorf("Expected 0 plugins in empty directory, found %d", len(manifests))
	}
}

func TestDiscoverPlugins_NonExistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "nonexistent")
	loader := NewLoader(pluginsDir)
	ctx := context.Background()
	manifests, err := loader.DiscoverPlugins(ctx)
	if err != nil {
		t.Fatalf("DiscoverPlugins failed: %v", err)
	}
	if manifests != nil {
		t.Errorf("Expected nil manifests for new directory, got %d", len(manifests))
	}
	// Verify directory was created
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		t.Error("Expected plugins directory to be created")
	}
}

func TestDiscoverPlugins_SkipsNonManifestFiles(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a non-manifest file
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("not a plugin"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a JSON file that's not named plugin.json
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	ctx := context.Background()
	manifests, err := loader.DiscoverPlugins(ctx)
	if err != nil {
		t.Fatalf("DiscoverPlugins failed: %v", err)
	}
	if len(manifests) != 0 {
		t.Errorf("Expected 0 plugins (no manifest files), found %d", len(manifests))
	}
}

func TestDiscoverPlugins_SkipsInvalidManifests(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "broken")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write an invalid manifest
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	ctx := context.Background()
	manifests, err := loader.DiscoverPlugins(ctx)
	if err != nil {
		t.Fatalf("DiscoverPlugins failed: %v", err)
	}
	// Should skip invalid manifest without error
	if len(manifests) != 0 {
		t.Errorf("Expected 0 valid plugins, found %d", len(manifests))
	}
}

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest *PluginManifest
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid http manifest",
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
			name: "valid grpc manifest",
			manifest: &PluginManifest{
				Type:     "grpc",
				Endpoint: "localhost:50051",
				Metadata: &plugin.Metadata{
					Name:         "GRPC Plugin",
					Version:      "1.0.0",
					ProviderType: "grpc-provider",
				},
			},
			wantErr: false,
		},
		{
			name: "valid builtin manifest",
			manifest: &PluginManifest{
				Type: "builtin",
				Metadata: &plugin.Metadata{
					Name:         "Builtin Plugin",
					Version:      "1.0.0",
					ProviderType: "builtin-provider",
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
			errMsg:  "metadata is required",
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
			errMsg:  "name is required",
		},
		{
			name: "missing provider_type",
			manifest: &PluginManifest{
				Type:     "http",
				Endpoint: "http://localhost:8090",
				Metadata: &plugin.Metadata{
					Name:    "Test",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "provider_type is required",
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
			errMsg:  "version is required",
		},
		{
			name: "missing type",
			manifest: &PluginManifest{
				Endpoint: "http://localhost:8090",
				Metadata: &plugin.Metadata{
					Name:         "Test",
					Version:      "1.0.0",
					ProviderType: "test",
				},
			},
			wantErr: true,
			errMsg:  "type is required",
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
			errMsg:  "endpoint is required",
		},
		{
			name: "missing endpoint for grpc",
			manifest: &PluginManifest{
				Type: "grpc",
				Metadata: &plugin.Metadata{
					Name:         "Test",
					Version:      "1.0.0",
					ProviderType: "test",
				},
			},
			wantErr: true,
			errMsg:  "endpoint is required",
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
			errMsg:  "unsupported plugin type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateManifest(tt.manifest)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
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
	if !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("Expected 'not loaded' error, got: %v", err)
	}
}

func TestGetPlugin_NotLoaded(t *testing.T) {
	loader := NewLoader(t.TempDir())
	_, err := loader.GetPlugin("something")
	if err == nil {
		t.Fatal("Expected error for plugin not loaded")
	}
}

func TestListPlugins_Empty(t *testing.T) {
	loader := NewLoader(t.TempDir())
	plugins := loader.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("Expected 0, got %d", len(plugins))
	}
}

func TestUnloadPlugin_NotLoaded(t *testing.T) {
	loader := NewLoader(t.TempDir())
	ctx := context.Background()
	err := loader.UnloadPlugin(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error for unloading non-existent plugin")
	}
	if !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("Expected 'not loaded' error, got: %v", err)
	}
}

func TestReloadPlugin_NotLoaded(t *testing.T) {
	loader := NewLoader(t.TempDir())
	ctx := context.Background()
	err := loader.ReloadPlugin(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error for reloading non-existent plugin")
	}
}

func TestLoadPlugin_UnsupportedType(t *testing.T) {
	loader := NewLoader(t.TempDir())
	ctx := context.Background()
	manifest := &PluginManifest{
		Type: "unsupported",
		Metadata: &plugin.Metadata{
			Name:         "Bad Plugin",
			ProviderType: "bad-provider",
		},
	}
	err := loader.LoadPlugin(ctx, manifest)
	if err == nil {
		t.Fatal("Expected error for unsupported type")
	}
	if !strings.Contains(err.Error(), "unsupported plugin type") {
		t.Errorf("Expected 'unsupported plugin type' error, got: %v", err)
	}
}

func TestLoadPlugin_GrpcNotImplemented(t *testing.T) {
	loader := NewLoader(t.TempDir())
	ctx := context.Background()
	manifest := &PluginManifest{
		Type:     "grpc",
		Endpoint: "localhost:50051",
		Metadata: &plugin.Metadata{
			Name:         "GRPC Plugin",
			ProviderType: "grpc-provider",
		},
	}
	err := loader.LoadPlugin(ctx, manifest)
	if err == nil {
		t.Fatal("Expected error for grpc (not yet implemented)")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("Expected 'not yet implemented' error, got: %v", err)
	}
}

func TestLoadPlugin_BuiltinNotImplemented(t *testing.T) {
	loader := NewLoader(t.TempDir())
	ctx := context.Background()
	manifest := &PluginManifest{
		Type: "builtin",
		Metadata: &plugin.Metadata{
			Name:         "Builtin Plugin",
			ProviderType: "builtin-provider",
		},
	}
	err := loader.LoadPlugin(ctx, manifest)
	if err == nil {
		t.Fatal("Expected error for builtin (not yet implemented)")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("Expected 'not yet implemented' error, got: %v", err)
	}
}

func TestSaveManifest_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := &PluginManifest{
		Type:     "http",
		Endpoint: "http://localhost:8090",
		Metadata: &plugin.Metadata{
			Name:         "Test",
			Version:      "1.0.0",
			ProviderType: "test",
		},
	}

	path := filepath.Join(tmpDir, "plugin.json")
	err := SaveManifest(manifest, path)
	if err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded PluginManifest
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if loaded.Type != "http" {
		t.Errorf("Expected type 'http', got '%s'", loaded.Type)
	}
}

func TestSaveManifest_YAML(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := &PluginManifest{
		Type:     "http",
		Endpoint: "http://localhost:9090",
		Metadata: &plugin.Metadata{
			Name:         "YAML Plugin",
			Version:      "2.0.0",
			ProviderType: "yaml-test",
		},
	}

	path := filepath.Join(tmpDir, "plugin.yaml")
	err := SaveManifest(manifest, path)
	if err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Expected file to exist")
	}
}

func TestSaveManifest_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := &PluginManifest{
		Type:     "http",
		Endpoint: "http://localhost:8090",
		Metadata: &plugin.Metadata{
			Name:         "Test",
			Version:      "1.0.0",
			ProviderType: "test",
		},
	}

	path := filepath.Join(tmpDir, "nested", "dir", "plugin.yaml")
	err := SaveManifest(manifest, path)
	if err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Expected nested file to exist")
	}
}

func TestLoadAll_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)
	ctx := context.Background()
	loaded, err := loader.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if loaded != 0 {
		t.Errorf("Expected 0 loaded plugins, got %d", loaded)
	}
}

func TestLoadAll_SkipsNonAutoStart(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a manifest with AutoStart=false
	manifest := &PluginManifest{
		Type:     "http",
		Endpoint: "http://localhost:8090",
		Metadata: &plugin.Metadata{
			Name:         "Non-Auto Plugin",
			Version:      "1.0.0",
			ProviderType: "non-auto",
		},
		AutoStart: false,
	}

	dir := filepath.Join(tmpDir, "noauto")
	path := filepath.Join(dir, "plugin.yaml")
	if err := SaveManifest(manifest, path); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	ctx := context.Background()
	loaded, err := loader.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if loaded != 0 {
		t.Errorf("Expected 0 loaded plugins (autostart=false), got %d", loaded)
	}
}

func TestPluginManifest_Fields(t *testing.T) {
	m := &PluginManifest{
		Type:     "http",
		Endpoint: "http://localhost:8080",
		Command:  "/usr/bin/plugin",
		Args:     []string{"--port", "8080"},
		Env:      map[string]string{"KEY": "val"},
		AutoStart: true,
		HealthCheckInterval: 30,
		Metadata: &plugin.Metadata{
			Name:         "Full Plugin",
			Version:      "3.0.0",
			ProviderType: "full",
		},
	}

	if m.Type != "http" {
		t.Errorf("Expected type 'http', got '%s'", m.Type)
	}
	if m.Endpoint != "http://localhost:8080" {
		t.Errorf("Expected endpoint, got '%s'", m.Endpoint)
	}
	if m.Command != "/usr/bin/plugin" {
		t.Errorf("Expected command, got '%s'", m.Command)
	}
	if len(m.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(m.Args))
	}
	if m.Env["KEY"] != "val" {
		t.Errorf("Expected env KEY=val, got '%s'", m.Env["KEY"])
	}
	if !m.AutoStart {
		t.Error("Expected AutoStart true")
	}
	if m.HealthCheckInterval != 30 {
		t.Errorf("Expected health check interval 30, got %d", m.HealthCheckInterval)
	}
}

func TestLoadedPlugin_Fields(t *testing.T) {
	lp := &LoadedPlugin{
		Manifest: &PluginManifest{Type: "http"},
		Client:   nil,
	}
	if lp.Manifest.Type != "http" {
		t.Errorf("Expected type 'http', got '%s'", lp.Manifest.Type)
	}
}

// --- HTTPPluginClient tests ---

func TestNewHTTPPluginClient(t *testing.T) {
	client, err := NewHTTPPluginClient("http://localhost:8080")
	if err != nil {
		t.Fatalf("NewHTTPPluginClient: %v", err)
	}
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.endpoint != "http://localhost:8080" {
		t.Errorf("Expected endpoint 'http://localhost:8080', got '%s'", client.endpoint)
	}
}

func TestNewHTTPPluginClient_EmptyEndpoint(t *testing.T) {
	_, err := NewHTTPPluginClient("")
	if err == nil {
		t.Fatal("Expected error for empty endpoint")
	}
	if !strings.Contains(err.Error(), "endpoint is required") {
		t.Errorf("Expected 'endpoint is required' error, got: %v", err)
	}
}

func TestHTTPPluginClient_GetMetadata_Cached(t *testing.T) {
	client := &HTTPPluginClient{
		endpoint: "http://localhost:8080",
		client:   &http.Client{Timeout: 5 * time.Second},
		metadata: &plugin.Metadata{
			Name:         "Cached Plugin",
			ProviderType: "cached",
		},
	}

	meta := client.GetMetadata()
	if meta == nil {
		t.Fatal("Expected non-nil metadata")
	}
	if meta.Name != "Cached Plugin" {
		t.Errorf("Expected 'Cached Plugin', got '%s'", meta.Name)
	}
}

func TestHTTPPluginClient_GetMetadata_FetchFails(t *testing.T) {
	// Client with invalid endpoint so fetch fails
	client := &HTTPPluginClient{
		endpoint: "http://localhost:1", // nothing listening
		client:   &http.Client{Timeout: 100 * time.Millisecond},
	}

	meta := client.GetMetadata()
	if meta != nil {
		t.Error("Expected nil metadata when fetch fails")
	}
}

func TestHTTPPluginClient_GetMetadata_WithServer(t *testing.T) {
	metadata := plugin.Metadata{
		Name:         "Server Plugin",
		Version:      "1.0.0",
		ProviderType: "server-test",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metadata" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(metadata)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	meta := client.GetMetadata()
	if meta == nil {
		t.Fatal("Expected non-nil metadata from server")
	}
	if meta.Name != "Server Plugin" {
		t.Errorf("Expected 'Server Plugin', got '%s'", meta.Name)
	}
	// Verify caching
	meta2 := client.GetMetadata()
	if meta2.Name != "Server Plugin" {
		t.Errorf("Expected cached 'Server Plugin', got '%s'", meta2.Name)
	}
}

func TestHTTPPluginClient_HealthCheck(t *testing.T) {
	health := plugin.HealthStatus{
		Healthy:   true,
		Message:   "OK",
		Latency:   10,
		Timestamp: time.Now(),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(health)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	status, err := client.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	if !status.Healthy {
		t.Error("Expected healthy")
	}
	if status.Message != "OK" {
		t.Errorf("Expected 'OK', got '%s'", status.Message)
	}
}

func TestHTTPPluginClient_HealthCheck_ConnectionFailed(t *testing.T) {
	client := &HTTPPluginClient{
		endpoint: "http://localhost:1",
		client:   &http.Client{Timeout: 100 * time.Millisecond},
	}

	ctx := context.Background()
	status, err := client.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck should not return error: %v", err)
	}
	if status.Healthy {
		t.Error("Expected unhealthy when connection fails")
	}
	if status.Message == "" {
		t.Error("Expected non-empty message")
	}
}

func TestHTTPPluginClient_Initialize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/initialize" && r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}
		if r.URL.Path == "/metadata" {
			json.NewEncoder(w).Encode(plugin.Metadata{Name: "Init Plugin", ProviderType: "init"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	err := client.Initialize(ctx, map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
}

func TestHTTPPluginClient_Initialize_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":"error","message":"init failed"}`))
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	err := client.Initialize(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for server error")
	}
}

func TestHTTPPluginClient_CreateChatCompletion(t *testing.T) {
	response := plugin.ChatCompletionResponse{
		ID:    "resp-1",
		Model: "test-model",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	resp, err := client.CreateChatCompletion(ctx, &plugin.ChatCompletionRequest{
		Model: "test-model",
	})
	if err != nil {
		t.Fatalf("CreateChatCompletion: %v", err)
	}
	if resp.ID != "resp-1" {
		t.Errorf("Expected ID 'resp-1', got '%s'", resp.ID)
	}
}

func TestHTTPPluginClient_CreateChatCompletion_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"code":"invalid_request","message":"bad model"}`))
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	_, err := client.CreateChatCompletion(ctx, &plugin.ChatCompletionRequest{})
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestHTTPPluginClient_GetModels(t *testing.T) {
	models := []plugin.ModelInfo{
		{ID: "model-1", Name: "Model One"},
		{ID: "model-2", Name: "Model Two"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(models)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	result, err := client.GetModels(ctx)
	if err != nil {
		t.Fatalf("GetModels: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 models, got %d", len(result))
	}
}

func TestHTTPPluginClient_GetModels_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	_, err := client.GetModels(ctx)
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestHTTPPluginClient_Cleanup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cleanup" && r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	err := client.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
}

func TestHTTPPluginClient_Cleanup_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("cleanup failed"))
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	err := client.Cleanup(ctx)
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestHTTPPluginClient_DoRequest_PluginError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(plugin.PluginError{
			Code:    "test_error",
			Message: "this is a test error",
		})
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	_, err := client.doRequest(ctx, "GET", "/test", nil)
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "test_error") {
		t.Errorf("Expected 'test_error' in error, got: %v", err)
	}
}

func TestHTTPPluginClient_DoRequest_NonJSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("bad gateway"))
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	_, err := client.doRequest(ctx, "GET", "/test", nil)
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("Expected status code 502 in error, got: %v", err)
	}
}

func TestHTTPPluginClient_DoRequest_WithBody(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	_, err := client.doRequest(ctx, "POST", "/test", []byte(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	if receivedBody != `{"key":"value"}` {
		t.Errorf("Expected body '{\"key\":\"value\"}', got '%s'", receivedBody)
	}
}

func TestHTTPPluginClient_DoRequest_NilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := &HTTPPluginClient{
		endpoint: server.URL,
		client:   server.Client(),
	}

	ctx := context.Background()
	resp, err := client.doRequest(ctx, "GET", "/test", nil)
	if err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	if !strings.Contains(string(resp), "ok") {
		t.Errorf("Expected 'ok' in response, got '%s'", string(resp))
	}
}

// --- Registry tests ---

func TestNewRegistry(t *testing.T) {
	sources := []RegistrySource{
		{Name: "test", URL: "http://localhost:8080", Enabled: true},
	}
	reg := NewRegistry(sources)
	if reg == nil {
		t.Fatal("Expected non-nil registry")
	}
	if len(reg.sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(reg.sources))
	}
	if reg.cache == nil {
		t.Error("Expected non-nil cache")
	}
}

func TestNewDefaultRegistry(t *testing.T) {
	reg := NewDefaultRegistry()
	if reg == nil {
		t.Fatal("Expected non-nil registry")
	}
	if len(reg.sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(reg.sources))
	}
}

func TestRegistryEntry_Fields(t *testing.T) {
	entry := RegistryEntry{
		ID:           "plugin-1",
		Name:         "Test Plugin",
		ProviderType: "test",
		Description:  "A test plugin",
		Author:       "Test Author",
		Version:      "1.0.0",
		License:      "MIT",
		Homepage:     "https://example.com",
		Repository:   "https://github.com/test/plugin",
		Downloads:    100,
		Rating:       4.5,
		Reviews:      10,
		Verified:     true,
		Tags:         []string{"test", "demo"},
	}

	if entry.ID != "plugin-1" {
		t.Errorf("Expected ID 'plugin-1', got '%s'", entry.ID)
	}
	if entry.Name != "Test Plugin" {
		t.Errorf("Expected name 'Test Plugin', got '%s'", entry.Name)
	}
	if entry.Downloads != 100 {
		t.Errorf("Expected 100 downloads, got %d", entry.Downloads)
	}
	if entry.Rating != 4.5 {
		t.Errorf("Expected rating 4.5, got %f", entry.Rating)
	}
	if !entry.Verified {
		t.Error("Expected verified")
	}
	if len(entry.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(entry.Tags))
	}
}

func TestInstallConfig_Fields(t *testing.T) {
	ic := InstallConfig{
		Type:        "http",
		ManifestURL: "https://example.com/plugin.yaml",
		DockerImage: "plugin:latest",
	}
	if ic.Type != "http" {
		t.Errorf("Expected type 'http', got '%s'", ic.Type)
	}
	if ic.ManifestURL != "https://example.com/plugin.yaml" {
		t.Errorf("Expected manifest URL, got '%s'", ic.ManifestURL)
	}
}

func TestRegistrySource_Fields(t *testing.T) {
	rs := RegistrySource{
		Name:    "official",
		URL:     "https://registry.example.com",
		Enabled: true,
	}
	if rs.Name != "official" {
		t.Errorf("Expected name 'official', got '%s'", rs.Name)
	}
	if !rs.Enabled {
		t.Error("Expected enabled")
	}
}

func TestCreateLocalRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	registryDir := filepath.Join(tmpDir, "registry")

	err := CreateLocalRegistry(registryDir)
	if err != nil {
		t.Fatalf("CreateLocalRegistry: %v", err)
	}

	indexPath := filepath.Join(registryDir, "registry.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatal("Expected registry.json to exist")
	}

	// Verify content
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if index.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", index.Version)
	}
	if len(index.Plugins) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(index.Plugins))
	}
}

func TestAddToLocalRegistry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create registry
	err := CreateLocalRegistry(tmpDir)
	if err != nil {
		t.Fatalf("CreateLocalRegistry: %v", err)
	}

	// Add entry
	entry := &RegistryEntry{
		ID:           "test-plugin",
		Name:         "Test Plugin",
		ProviderType: "test",
		Version:      "1.0.0",
	}
	err = AddToLocalRegistry(tmpDir, entry)
	if err != nil {
		t.Fatalf("AddToLocalRegistry: %v", err)
	}

	// Verify
	data, err := os.ReadFile(filepath.Join(tmpDir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatal(err)
	}
	if len(index.Plugins) != 1 {
		t.Fatalf("Expected 1 plugin, got %d", len(index.Plugins))
	}
	if index.Plugins[0].ID != "test-plugin" {
		t.Errorf("Expected ID 'test-plugin', got '%s'", index.Plugins[0].ID)
	}
}

func TestAddToLocalRegistry_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()

	err := CreateLocalRegistry(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Add first version
	entry1 := &RegistryEntry{ID: "p1", Name: "Plugin V1", Version: "1.0.0"}
	if err := AddToLocalRegistry(tmpDir, entry1); err != nil {
		t.Fatal(err)
	}

	// Update same ID
	entry2 := &RegistryEntry{ID: "p1", Name: "Plugin V2", Version: "2.0.0"}
	if err := AddToLocalRegistry(tmpDir, entry2); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatal(err)
	}
	if len(index.Plugins) != 1 {
		t.Fatalf("Expected 1 plugin (updated), got %d", len(index.Plugins))
	}
	if index.Plugins[0].Name != "Plugin V2" {
		t.Errorf("Expected 'Plugin V2', got '%s'", index.Plugins[0].Name)
	}
}

func TestAddToLocalRegistry_NoExistingIndex(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create registry first - should handle missing file
	entry := &RegistryEntry{ID: "p1", Name: "Plugin", Version: "1.0.0"}
	err := AddToLocalRegistry(tmpDir, entry)
	if err != nil {
		t.Fatalf("AddToLocalRegistry should handle missing index: %v", err)
	}
}

func TestRegistry_Search_LocalSource(t *testing.T) {
	tmpDir := t.TempDir()

	// Create local registry with plugins
	if err := CreateLocalRegistry(tmpDir); err != nil {
		t.Fatal(err)
	}

	plugins := []*RegistryEntry{
		{ID: "p1", Name: "OpenAI Plugin", Description: "GPT integration", Author: "TestCo", Downloads: 100, Rating: 4.5, Tags: []string{"ai", "gpt"}},
		{ID: "p2", Name: "Claude Plugin", Description: "Anthropic integration", Author: "TestCo", Downloads: 50, Rating: 4.0, Tags: []string{"ai", "claude"}},
		{ID: "p3", Name: "Image Gen", Description: "Generate images", Author: "ArtCo", Downloads: 200, Rating: 3.5, Tags: []string{"image"}},
	}

	for _, p := range plugins {
		if err := AddToLocalRegistry(tmpDir, p); err != nil {
			t.Fatal(err)
		}
	}

	sources := []RegistrySource{
		{Name: "local", URL: "file://" + tmpDir, Enabled: true},
	}
	reg := NewRegistry(sources)

	ctx := context.Background()

	// Search by name
	results, err := reg.Search(ctx, "OpenAI")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'OpenAI', got %d", len(results))
	}

	// Search by tag
	results, err = reg.Search(ctx, "ai")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'ai' tag, got %d", len(results))
	}

	// Search by author
	results, err = reg.Search(ctx, "TestCo")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'TestCo' author, got %d", len(results))
	}
}

func TestRegistry_Get_Found(t *testing.T) {
	tmpDir := t.TempDir()
	if err := CreateLocalRegistry(tmpDir); err != nil {
		t.Fatal(err)
	}
	if err := AddToLocalRegistry(tmpDir, &RegistryEntry{ID: "p1", Name: "Test"}); err != nil {
		t.Fatal(err)
	}

	sources := []RegistrySource{
		{Name: "local", URL: "file://" + tmpDir, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	entry, err := reg.Get(ctx, "p1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if entry.Name != "Test" {
		t.Errorf("Expected 'Test', got '%s'", entry.Name)
	}

	// Verify caching works
	entry2, err := reg.Get(ctx, "p1")
	if err != nil {
		t.Fatalf("Get (cached): %v", err)
	}
	if entry2.Name != "Test" {
		t.Errorf("Expected cached 'Test', got '%s'", entry2.Name)
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	if err := CreateLocalRegistry(tmpDir); err != nil {
		t.Fatal(err)
	}

	sources := []RegistrySource{
		{Name: "local", URL: "file://" + tmpDir, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	_, err := reg.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent plugin")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	tmpDir := t.TempDir()
	if err := CreateLocalRegistry(tmpDir); err != nil {
		t.Fatal(err)
	}
	if err := AddToLocalRegistry(tmpDir, &RegistryEntry{ID: "p1", Name: "One"}); err != nil {
		t.Fatal(err)
	}
	if err := AddToLocalRegistry(tmpDir, &RegistryEntry{ID: "p2", Name: "Two"}); err != nil {
		t.Fatal(err)
	}

	sources := []RegistrySource{
		{Name: "local", URL: "file://" + tmpDir, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	plugins, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(plugins))
	}
}

func TestRegistry_DisabledSource(t *testing.T) {
	tmpDir := t.TempDir()
	if err := CreateLocalRegistry(tmpDir); err != nil {
		t.Fatal(err)
	}
	if err := AddToLocalRegistry(tmpDir, &RegistryEntry{ID: "p1", Name: "Hidden"}); err != nil {
		t.Fatal(err)
	}

	sources := []RegistrySource{
		{Name: "local", URL: "file://" + tmpDir, Enabled: false},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	plugins, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("Expected 0 plugins (source disabled), got %d", len(plugins))
	}
}

func TestRegistry_HTTPSource(t *testing.T) {
	index := RegistryIndex{
		Version: "1.0",
		Plugins: []*RegistryEntry{
			{ID: "http-p1", Name: "HTTP Plugin", Downloads: 100, Rating: 4.5},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/registry.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(index)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	sources := []RegistrySource{
		{Name: "remote", URL: server.URL, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	plugins, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].ID != "http-p1" {
		t.Errorf("Expected ID 'http-p1', got '%s'", plugins[0].ID)
	}
}

func TestRegistry_HTTPSource_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sources := []RegistrySource{
		{Name: "broken", URL: server.URL, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	// Should not error, just return empty
	plugins, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("Expected 0 plugins (server error), got %d", len(plugins))
	}
}

func TestRegistry_Install(t *testing.T) {
	// Set up manifest server
	manifestData := []byte(`type: http
endpoint: http://localhost:8080
metadata:
  name: Installable Plugin
  version: 1.0.0
  provider_type: installable`)

	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(manifestData)
	}))
	defer manifestServer.Close()

	// Set up registry server
	index := RegistryIndex{
		Version: "1.0",
		Plugins: []*RegistryEntry{
			{
				ID:   "installable",
				Name: "Installable Plugin",
				Install: InstallConfig{
					Type:        "http",
					ManifestURL: manifestServer.URL + "/plugin.yaml",
				},
			},
		},
	}

	registryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/registry.json" {
			json.NewEncoder(w).Encode(index)
			return
		}
		http.NotFound(w, r)
	}))
	defer registryServer.Close()

	sources := []RegistrySource{
		{Name: "remote", URL: registryServer.URL, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	tmpDir := t.TempDir()
	err := reg.Install(ctx, "installable", tmpDir)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Verify manifest was saved
	savedPath := filepath.Join(tmpDir, "installable", "plugin.yaml")
	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Fatal("Expected manifest to be saved")
	}
}

func TestRegistry_Install_PluginNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	if err := CreateLocalRegistry(tmpDir); err != nil {
		t.Fatal(err)
	}

	sources := []RegistrySource{
		{Name: "local", URL: "file://" + tmpDir, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	err := reg.Install(ctx, "nonexistent", t.TempDir())
	if err == nil {
		t.Fatal("Expected error for non-existent plugin")
	}
}

func TestRegistry_LoadLocalRegistry_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(nil)
	plugins, err := reg.loadLocalRegistry(tmpDir)
	if err != nil {
		t.Fatalf("loadLocalRegistry: %v", err)
	}
	if plugins != nil {
		t.Errorf("Expected nil for non-existent registry, got %d", len(plugins))
	}
}

func TestContainsTag(t *testing.T) {
	tags := []string{"ai", "GPT", "Production"}

	if !containsTag(tags, "ai") {
		t.Error("Expected to find 'ai' tag")
	}
	if !containsTag(tags, "gpt") {
		t.Error("Expected case-insensitive match for 'gpt'")
	}
	if !containsTag(tags, "production") {
		t.Error("Expected case-insensitive match for 'production'")
	}
	if containsTag(tags, "missing") {
		t.Error("Expected not to find 'missing' tag")
	}
	if containsTag(nil, "ai") {
		t.Error("Expected false for nil tags")
	}
	if containsTag([]string{}, "ai") {
		t.Error("Expected false for empty tags")
	}
}

func TestGetLocalRegistryPath(t *testing.T) {
	path := getLocalRegistryPath()
	if path == "" {
		t.Error("Expected non-empty path")
	}
	if !strings.Contains(path, "registry") {
		t.Errorf("Expected 'registry' in path, got '%s'", path)
	}
}

func TestRegistryIndex_Fields(t *testing.T) {
	idx := RegistryIndex{
		Version: "1.0",
		Plugins: []*RegistryEntry{
			{ID: "p1"},
			{ID: "p2"},
		},
	}
	if idx.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", idx.Version)
	}
	if len(idx.Plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(idx.Plugins))
	}
}

func TestRegistry_Search_SortsByRelevance(t *testing.T) {
	tmpDir := t.TempDir()
	if err := CreateLocalRegistry(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Add plugins with different download/rating scores
	plugins := []*RegistryEntry{
		{ID: "low", Name: "AI Low Score", Downloads: 10, Rating: 1.0},
		{ID: "high", Name: "AI High Score", Downloads: 1000, Rating: 5.0},
		{ID: "mid", Name: "AI Mid Score", Downloads: 100, Rating: 3.0},
	}
	for _, p := range plugins {
		if err := AddToLocalRegistry(tmpDir, p); err != nil {
			t.Fatal(err)
		}
	}

	sources := []RegistrySource{
		{Name: "local", URL: "file://" + tmpDir, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	results, err := reg.Search(ctx, "AI")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// First result should have highest score
	if results[0].ID != "high" {
		t.Errorf("Expected highest scored plugin first, got '%s'", results[0].ID)
	}

	// Verify descending order
	for i := 1; i < len(results); i++ {
		prevScore := float64(results[i-1].Downloads) * results[i-1].Rating
		currScore := float64(results[i].Downloads) * results[i].Rating
		if prevScore < currScore {
			t.Errorf("Results not sorted: score[%d]=%f < score[%d]=%f",
				i-1, prevScore, i, currScore)
		}
	}
}

func TestRegistry_Deduplication(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	if err := CreateLocalRegistry(tmpDir1); err != nil {
		t.Fatal(err)
	}
	if err := CreateLocalRegistry(tmpDir2); err != nil {
		t.Fatal(err)
	}

	// Same ID in both registries
	entry1 := &RegistryEntry{ID: "shared", Name: "From Source 1"}
	entry2 := &RegistryEntry{ID: "shared", Name: "From Source 2"}

	if err := AddToLocalRegistry(tmpDir1, entry1); err != nil {
		t.Fatal(err)
	}
	if err := AddToLocalRegistry(tmpDir2, entry2); err != nil {
		t.Fatal(err)
	}

	sources := []RegistrySource{
		{Name: "first", URL: "file://" + tmpDir1, Enabled: true},
		{Name: "second", URL: "file://" + tmpDir2, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	plugins, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Should have only 1 (first source wins)
	count := 0
	for _, p := range plugins {
		if p.ID == "shared" {
			count++
			if p.Name != "From Source 1" {
				t.Errorf("Expected first source to win, got '%s'", p.Name)
			}
		}
	}
	if count != 1 {
		t.Errorf("Expected exactly 1 'shared' plugin, got %d", count)
	}
}

func TestRegistry_Search_ByDescription(t *testing.T) {
	tmpDir := t.TempDir()
	if err := CreateLocalRegistry(tmpDir); err != nil {
		t.Fatal(err)
	}
	if err := AddToLocalRegistry(tmpDir, &RegistryEntry{
		ID: "p1", Name: "Plugin", Description: "Provides machine learning capabilities",
	}); err != nil {
		t.Fatal(err)
	}

	sources := []RegistrySource{
		{Name: "local", URL: "file://" + tmpDir, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	results, err := reg.Search(ctx, "machine learning")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for description search, got %d", len(results))
	}
}

func TestRegistry_Search_NoResults(t *testing.T) {
	tmpDir := t.TempDir()
	if err := CreateLocalRegistry(tmpDir); err != nil {
		t.Fatal(err)
	}
	if err := AddToLocalRegistry(tmpDir, &RegistryEntry{ID: "p1", Name: "Something"}); err != nil {
		t.Fatal(err)
	}

	sources := []RegistrySource{
		{Name: "local", URL: "file://" + tmpDir, Enabled: true},
	}
	reg := NewRegistry(sources)
	ctx := context.Background()

	results, err := reg.Search(ctx, "zzz_no_match")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

// This test verifies that the LoadPlugin function rejects duplicate plugins
func TestLoadPlugin_AlreadyLoaded(t *testing.T) {
	loader := NewLoader(t.TempDir())

	// Manually inject a plugin
	loader.mu.Lock()
	loader.plugins["test-provider"] = &LoadedPlugin{
		Manifest: &PluginManifest{Type: "http"},
	}
	loader.mu.Unlock()

	manifest := &PluginManifest{
		Type:     "http",
		Endpoint: "http://localhost:8080",
		Metadata: &plugin.Metadata{
			Name:         "Dup Plugin",
			ProviderType: "test-provider",
		},
	}

	ctx := context.Background()
	err := loader.LoadPlugin(ctx, manifest)
	if err == nil {
		t.Fatal("Expected error for already loaded plugin")
	}
	if !strings.Contains(err.Error(), "already loaded") {
		t.Errorf("Expected 'already loaded' error, got: %v", err)
	}
}

// Full integration test with mock server for LoadPlugin
func TestLoadPlugin_HTTP_FullIntegration(t *testing.T) {
	metadata := plugin.Metadata{
		Name:         "Integration Plugin",
		Version:      "1.0.0",
		ProviderType: "integration-test",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/initialize":
			w.Write([]byte(`{}`))
		case "/metadata":
			json.NewEncoder(w).Encode(metadata)
		case "/health":
			json.NewEncoder(w).Encode(plugin.HealthStatus{Healthy: true, Message: "OK", Timestamp: time.Now()})
		case "/cleanup":
			w.Write([]byte(`{}`))
		default:
			fmt.Fprintf(w, `{}`)
		}
	}))
	defer server.Close()

	loader := NewLoader(t.TempDir())
	ctx := context.Background()

	manifest := &PluginManifest{
		Type:     "http",
		Endpoint: server.URL,
		Metadata: &plugin.Metadata{
			Name:         "Integration Plugin",
			Version:      "1.0.0",
			ProviderType: "integration-test",
		},
	}

	err := loader.LoadPlugin(ctx, manifest)
	if err != nil {
		t.Fatalf("LoadPlugin: %v", err)
	}

	// Verify plugin is loaded
	loaded, err := loader.GetPlugin("integration-test")
	if err != nil {
		t.Fatalf("GetPlugin: %v", err)
	}
	if loaded.Manifest.Type != "http" {
		t.Errorf("Expected type 'http', got '%s'", loaded.Manifest.Type)
	}

	// List plugins
	plugins := loader.ListPlugins()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(plugins))
	}

	// Unload
	err = loader.UnloadPlugin(ctx, "integration-test")
	if err != nil {
		t.Fatalf("UnloadPlugin: %v", err)
	}

	// Should be gone
	_, err = loader.GetPlugin("integration-test")
	if err == nil {
		t.Error("Expected error after unloading plugin")
	}
}

func TestLoadPlugin_HTTP_HealthCheckFails(t *testing.T) {
	metadata := plugin.Metadata{
		Name:         "Unhealthy Plugin",
		Version:      "1.0.0",
		ProviderType: "unhealthy-test",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/initialize":
			w.Write([]byte(`{}`))
		case "/metadata":
			json.NewEncoder(w).Encode(metadata)
		case "/health":
			json.NewEncoder(w).Encode(plugin.HealthStatus{Healthy: false, Message: "DB down", Timestamp: time.Now()})
		default:
			w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	loader := NewLoader(t.TempDir())
	ctx := context.Background()

	manifest := &PluginManifest{
		Type:     "http",
		Endpoint: server.URL,
		Metadata: &plugin.Metadata{
			Name:         "Unhealthy Plugin",
			Version:      "1.0.0",
			ProviderType: "unhealthy-test",
		},
	}

	err := loader.LoadPlugin(ctx, manifest)
	if err == nil {
		t.Fatal("Expected error for unhealthy plugin")
	}
	if !strings.Contains(err.Error(), "unhealthy") {
		t.Errorf("Expected 'unhealthy' error, got: %v", err)
	}
}

func TestLoadPlugin_HTTP_MetadataMismatch(t *testing.T) {
	// Server returns different provider type than manifest
	metadata := plugin.Metadata{
		Name:         "Mismatch Plugin",
		Version:      "1.0.0",
		ProviderType: "different-type",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/initialize":
			w.Write([]byte(`{}`))
		case "/metadata":
			json.NewEncoder(w).Encode(metadata)
		case "/health":
			json.NewEncoder(w).Encode(plugin.HealthStatus{Healthy: true, Message: "OK", Timestamp: time.Now()})
		default:
			w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	loader := NewLoader(t.TempDir())
	ctx := context.Background()

	manifest := &PluginManifest{
		Type:     "http",
		Endpoint: server.URL,
		Metadata: &plugin.Metadata{
			Name:         "Mismatch Plugin",
			Version:      "1.0.0",
			ProviderType: "expected-type",
		},
	}

	err := loader.LoadPlugin(ctx, manifest)
	if err == nil {
		t.Fatal("Expected error for metadata mismatch")
	}
	if !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("Expected 'mismatch' error, got: %v", err)
	}
}
