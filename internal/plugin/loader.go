package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jordanhubbard/agenticorp/pkg/plugin"
	"gopkg.in/yaml.v3"
)

// Loader manages loading and registration of plugins.
type Loader struct {
	pluginsDir string
	plugins    map[string]*LoadedPlugin
	mu         sync.RWMutex
}

// LoadedPlugin represents a loaded plugin with its manifest.
type LoadedPlugin struct {
	Manifest *PluginManifest
	Client   plugin.Plugin
}

// PluginManifest describes a plugin's configuration and how to load it.
type PluginManifest struct {
	// Metadata from plugin interface
	Metadata *plugin.Metadata `json:"metadata" yaml:"metadata"`

	// Type indicates how to load the plugin: "http", "grpc", "builtin"
	Type string `json:"type" yaml:"type"`

	// Endpoint is the plugin endpoint (for http/grpc plugins)
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`

	// Command is the command to start the plugin process (optional)
	Command string `json:"command,omitempty" yaml:"command,omitempty"`

	// Args are arguments for the command
	Args []string `json:"args,omitempty" yaml:"args,omitempty"`

	// Env contains environment variables for the plugin process
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`

	// AutoStart indicates if the plugin should be started automatically
	AutoStart bool `json:"auto_start" yaml:"auto_start"`

	// HealthCheckInterval is how often to check plugin health (seconds)
	HealthCheckInterval int `json:"health_check_interval,omitempty" yaml:"health_check_interval,omitempty"`
}

// NewLoader creates a new plugin loader.
func NewLoader(pluginsDir string) *Loader {
	return &Loader{
		pluginsDir: pluginsDir,
		plugins:    make(map[string]*LoadedPlugin),
	}
}

// DiscoverPlugins scans the plugins directory for plugin manifests.
// Returns a list of discovered plugin manifests.
func (l *Loader) DiscoverPlugins(ctx context.Context) ([]*PluginManifest, error) {
	// Check if plugins directory exists
	if _, err := os.Stat(l.pluginsDir); os.IsNotExist(err) {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(l.pluginsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create plugins directory: %w", err)
		}
		return nil, nil
	}

	var manifests []*PluginManifest

	// Walk the plugins directory
	err := filepath.Walk(l.pluginsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Look for manifest files
		ext := filepath.Ext(path)
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			return nil
		}

		// Check filename pattern (must be plugin.json or plugin.yaml)
		base := filepath.Base(path)
		if base != "plugin.json" && base != "plugin.yaml" && base != "plugin.yml" {
			return nil
		}

		// Load manifest
		manifest, err := l.loadManifest(path)
		if err != nil {
			// Log error but continue discovery
			fmt.Fprintf(os.Stderr, "[WARN] Failed to load plugin manifest %s: %v\n", path, err)
			return nil
		}

		manifests = append(manifests, manifest)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk plugins directory: %w", err)
	}

	return manifests, nil
}

// LoadPlugin loads a plugin from its manifest.
func (l *Loader) LoadPlugin(ctx context.Context, manifest *PluginManifest) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already loaded
	if _, exists := l.plugins[manifest.Metadata.ProviderType]; exists {
		return fmt.Errorf("plugin %s already loaded", manifest.Metadata.ProviderType)
	}

	// Create plugin client based on type
	var client plugin.Plugin
	var err error

	switch manifest.Type {
	case "http":
		client, err = NewHTTPPluginClient(manifest.Endpoint)
	case "grpc":
		return fmt.Errorf("grpc plugins not yet implemented")
	case "builtin":
		return fmt.Errorf("builtin plugins not yet implemented")
	default:
		return fmt.Errorf("unsupported plugin type: %s", manifest.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create plugin client: %w", err)
	}

	// Initialize plugin
	config := make(map[string]interface{})
	if err := client.Initialize(ctx, config); err != nil {
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	// Verify metadata matches
	pluginMetadata := client.GetMetadata()
	if pluginMetadata.ProviderType != manifest.Metadata.ProviderType {
		return fmt.Errorf("provider type mismatch: manifest=%s, plugin=%s",
			manifest.Metadata.ProviderType, pluginMetadata.ProviderType)
	}

	// Health check
	health, err := client.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("plugin health check failed: %w", err)
	}
	if !health.Healthy {
		return fmt.Errorf("plugin is unhealthy: %s", health.Message)
	}

	// Store loaded plugin
	l.plugins[manifest.Metadata.ProviderType] = &LoadedPlugin{
		Manifest: manifest,
		Client:   client,
	}

	return nil
}

// UnloadPlugin unloads a plugin and performs cleanup.
func (l *Loader) UnloadPlugin(ctx context.Context, providerType string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	loaded, exists := l.plugins[providerType]
	if !exists {
		return fmt.Errorf("plugin %s not loaded", providerType)
	}

	// Cleanup plugin
	if err := loaded.Client.Cleanup(ctx); err != nil {
		return fmt.Errorf("plugin cleanup failed: %w", err)
	}

	// Remove from loaded plugins
	delete(l.plugins, providerType)

	return nil
}

// GetPlugin retrieves a loaded plugin.
func (l *Loader) GetPlugin(providerType string) (*LoadedPlugin, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	loaded, exists := l.plugins[providerType]
	if !exists {
		return nil, fmt.Errorf("plugin %s not loaded", providerType)
	}

	return loaded, nil
}

// ListPlugins returns all loaded plugins.
func (l *Loader) ListPlugins() []*LoadedPlugin {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugins := make([]*LoadedPlugin, 0, len(l.plugins))
	for _, p := range l.plugins {
		plugins = append(plugins, p)
	}

	return plugins
}

// ReloadPlugin reloads a plugin (unload then load).
func (l *Loader) ReloadPlugin(ctx context.Context, providerType string) error {
	// Get current manifest
	loaded, err := l.GetPlugin(providerType)
	if err != nil {
		return err
	}

	manifest := loaded.Manifest

	// Unload
	if err := l.UnloadPlugin(ctx, providerType); err != nil {
		return fmt.Errorf("failed to unload plugin: %w", err)
	}

	// Load
	if err := l.LoadPlugin(ctx, manifest); err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	return nil
}

// LoadAll discovers and loads all plugins.
func (l *Loader) LoadAll(ctx context.Context) (int, error) {
	manifests, err := l.DiscoverPlugins(ctx)
	if err != nil {
		return 0, err
	}

	loaded := 0
	for _, manifest := range manifests {
		if !manifest.AutoStart {
			continue
		}

		if err := l.LoadPlugin(ctx, manifest); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to load plugin %s: %v\n",
				manifest.Metadata.Name, err)
			continue
		}

		loaded++
	}

	return loaded, nil
}

// loadManifest loads a plugin manifest from a file.
func (l *Loader) loadManifest(path string) (*PluginManifest, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest PluginManifest

	// Parse based on extension
	ext := filepath.Ext(path)
	if ext == ".json" {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse JSON manifest: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse YAML manifest: %w", err)
		}
	}

	// Validate manifest
	if manifest.Metadata == nil {
		return nil, fmt.Errorf("manifest missing metadata")
	}
	if manifest.Metadata.Name == "" {
		return nil, fmt.Errorf("manifest missing name")
	}
	if manifest.Metadata.ProviderType == "" {
		return nil, fmt.Errorf("manifest missing provider_type")
	}
	if manifest.Type == "" {
		return nil, fmt.Errorf("manifest missing type")
	}

	return &manifest, nil
}

// SaveManifest saves a plugin manifest to a file.
func SaveManifest(manifest *PluginManifest, path string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Determine format from extension
	ext := filepath.Ext(path)

	var data []byte
	var err error

	if ext == ".json" {
		data, err = json.MarshalIndent(manifest, "", "  ")
	} else {
		data, err = yaml.Marshal(manifest)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// ValidateManifest validates a plugin manifest.
func ValidateManifest(manifest *PluginManifest) error {
	if manifest.Metadata == nil {
		return fmt.Errorf("metadata is required")
	}

	if manifest.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	if manifest.Metadata.ProviderType == "" {
		return fmt.Errorf("metadata.provider_type is required")
	}

	if manifest.Metadata.Version == "" {
		return fmt.Errorf("metadata.version is required")
	}

	if manifest.Type == "" {
		return fmt.Errorf("type is required")
	}

	switch manifest.Type {
	case "http", "grpc":
		if manifest.Endpoint == "" {
			return fmt.Errorf("endpoint is required for %s plugins", manifest.Type)
		}
	case "builtin":
		// No endpoint required
	default:
		return fmt.Errorf("unsupported plugin type: %s", manifest.Type)
	}

	return nil
}

// CreateExampleManifest creates an example plugin manifest.
func CreateExampleManifest(pluginsDir string) error {
	manifest := &PluginManifest{
		Type:     "http",
		Endpoint: "http://localhost:8090",
		Metadata: &plugin.Metadata{
			Name:             "Example Plugin",
			Version:          "1.0.0",
			PluginAPIVersion: plugin.PluginVersion,
			ProviderType:     "example-provider",
			Description:      "An example plugin for demonstration",
			Author:           "AgentiCorp Team",
			License:          "MIT",
			Capabilities: plugin.Capabilities{
				Streaming:       true,
				FunctionCalling: false,
				Vision:          false,
			},
			ConfigSchema: []plugin.ConfigField{
				{
					Name:        "api_key",
					Type:        "string",
					Required:    true,
					Description: "API key for authentication",
					Sensitive:   true,
				},
				{
					Name:        "endpoint",
					Type:        "string",
					Required:    false,
					Description: "Custom endpoint URL",
					Default:     "https://api.example.com",
				},
			},
		},
		AutoStart:           false,
		HealthCheckInterval: 60,
	}

	path := filepath.Join(pluginsDir, "example", "plugin.yaml")
	return SaveManifest(manifest, path)
}
