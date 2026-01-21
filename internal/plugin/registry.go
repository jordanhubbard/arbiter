package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jordanhubbard/agenticorp/pkg/plugin"
)

// RegistryEntry represents a plugin in the registry.
type RegistryEntry struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	ProviderType    string                 `json:"provider_type"`
	Description     string                 `json:"description"`
	Author          string                 `json:"author"`
	Version         string                 `json:"version"`
	License         string                 `json:"license"`
	Homepage        string                 `json:"homepage"`
	Repository      string                 `json:"repository"`
	Downloads       int64                  `json:"downloads"`
	Rating          float64                `json:"rating"`
	Reviews         int64                  `json:"reviews"`
	Verified        bool                   `json:"verified"`
	Tags            []string               `json:"tags"`
	Capabilities    plugin.Capabilities    `json:"capabilities"`
	Install         InstallConfig          `json:"install"`
	Screenshots     []string               `json:"screenshots,omitempty"`
	DocumentationURL string                `json:"documentation_url,omitempty"`
	PublishedAt     time.Time              `json:"published_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// InstallConfig describes how to install the plugin.
type InstallConfig struct {
	Type        string `json:"type"`         // http, grpc, docker
	ManifestURL string `json:"manifest_url"` // URL to plugin.yaml
	DockerImage string `json:"docker_image,omitempty"`
}

// RegistryIndex represents the registry index file.
type RegistryIndex struct {
	Version string           `json:"version"`
	Plugins []*RegistryEntry `json:"plugins"`
}

// Registry manages plugin discovery and installation from registries.
type Registry struct {
	sources []RegistrySource
	cache   map[string]*RegistryEntry
}

// RegistrySource represents a plugin registry source.
type RegistrySource struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// NewRegistry creates a new plugin registry client.
func NewRegistry(sources []RegistrySource) *Registry {
	return &Registry{
		sources: sources,
		cache:   make(map[string]*RegistryEntry),
	}
}

// NewDefaultRegistry creates a registry with default sources.
func NewDefaultRegistry() *Registry {
	sources := []RegistrySource{
		{
			Name:    "official",
			URL:     "https://registry.agenticorp.io",
			Enabled: true,
		},
		{
			Name:    "local",
			URL:     "file://" + getLocalRegistryPath(),
			Enabled: true,
		},
	}

	return NewRegistry(sources)
}

// Search searches for plugins matching the query.
func (r *Registry) Search(ctx context.Context, query string) ([]*RegistryEntry, error) {
	// Load all plugins from sources
	allPlugins, err := r.loadAll(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by query
	var results []*RegistryEntry
	queryLower := strings.ToLower(query)

	for _, plugin := range allPlugins {
		// Match against name, description, tags, author
		if strings.Contains(strings.ToLower(plugin.Name), queryLower) ||
			strings.Contains(strings.ToLower(plugin.Description), queryLower) ||
			strings.Contains(strings.ToLower(plugin.Author), queryLower) ||
			containsTag(plugin.Tags, queryLower) {
			results = append(results, plugin)
		}
	}

	// Sort by relevance (downloads * rating)
	sort.Slice(results, func(i, j int) bool {
		scoreI := float64(results[i].Downloads) * results[i].Rating
		scoreJ := float64(results[j].Downloads) * results[j].Rating
		return scoreI > scoreJ
	})

	return results, nil
}

// Get retrieves a specific plugin by ID.
func (r *Registry) Get(ctx context.Context, pluginID string) (*RegistryEntry, error) {
	// Check cache first
	if cached, ok := r.cache[pluginID]; ok {
		return cached, nil
	}

	// Load from sources
	allPlugins, err := r.loadAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, plugin := range allPlugins {
		if plugin.ID == pluginID {
			r.cache[pluginID] = plugin
			return plugin, nil
		}
	}

	return nil, fmt.Errorf("plugin not found: %s", pluginID)
}

// List lists all available plugins.
func (r *Registry) List(ctx context.Context) ([]*RegistryEntry, error) {
	return r.loadAll(ctx)
}

// Install installs a plugin from the registry.
func (r *Registry) Install(ctx context.Context, pluginID, targetDir string) error {
	// Get plugin from registry
	entry, err := r.Get(ctx, pluginID)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}

	// Download manifest
	manifestData, err := r.downloadFile(ctx, entry.Install.ManifestURL)
	if err != nil {
		return fmt.Errorf("failed to download manifest: %w", err)
	}

	// Create plugin directory
	pluginDir := filepath.Join(targetDir, pluginID)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Save manifest
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	// Record installation
	entry.Downloads++

	return nil
}

// loadAll loads plugins from all enabled sources.
func (r *Registry) loadAll(ctx context.Context) ([]*RegistryEntry, error) {
	var allPlugins []*RegistryEntry
	seen := make(map[string]bool)

	for _, source := range r.sources {
		if !source.Enabled {
			continue
		}

		plugins, err := r.loadSource(ctx, source)
		if err != nil {
			// Log error but continue with other sources
			continue
		}

		// Deduplicate plugins (first source wins)
		for _, plugin := range plugins {
			if !seen[plugin.ID] {
				allPlugins = append(allPlugins, plugin)
				seen[plugin.ID] = true
			}
		}
	}

	return allPlugins, nil
}

// loadSource loads plugins from a specific source.
func (r *Registry) loadSource(ctx context.Context, source RegistrySource) ([]*RegistryEntry, error) {
	if strings.HasPrefix(source.URL, "file://") {
		// Local file source
		path := strings.TrimPrefix(source.URL, "file://")
		return r.loadLocalRegistry(path)
	}

	// HTTP source
	return r.loadHTTPRegistry(ctx, source.URL)
}

// loadLocalRegistry loads a local registry index.
func (r *Registry) loadLocalRegistry(path string) ([]*RegistryEntry, error) {
	indexPath := filepath.Join(path, "registry.json")

	// Check if file exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return nil, nil // Empty registry
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return index.Plugins, nil
}

// loadHTTPRegistry loads a remote registry index via HTTP.
func (r *Registry) loadHTTPRegistry(ctx context.Context, url string) ([]*RegistryEntry, error) {
	indexURL := url + "/registry.json"

	req, err := http.NewRequestWithContext(ctx, "GET", indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var index RegistryIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return index.Plugins, nil
}

// downloadFile downloads a file from a URL.
func (r *Registry) downloadFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// CreateLocalRegistry creates a new local registry.
func CreateLocalRegistry(path string) error {
	// Create directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create empty index
	index := &RegistryIndex{
		Version: "1.0",
		Plugins: []*RegistryEntry{},
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	indexPath := filepath.Join(path, "registry.json")
	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}

// AddToLocalRegistry adds a plugin to a local registry.
func AddToLocalRegistry(registryPath string, entry *RegistryEntry) error {
	indexPath := filepath.Join(registryPath, "registry.json")

	// Read existing index
	var index RegistryIndex
	if data, err := os.ReadFile(indexPath); err == nil {
		json.Unmarshal(data, &index)
	} else {
		index = RegistryIndex{Version: "1.0", Plugins: []*RegistryEntry{}}
	}

	// Check if plugin already exists
	found := false
	for i, plugin := range index.Plugins {
		if plugin.ID == entry.ID {
			// Update existing
			index.Plugins[i] = entry
			found = true
			break
		}
	}

	if !found {
		// Add new
		index.Plugins = append(index.Plugins, entry)
	}

	// Write index
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}

// Helper functions

func getLocalRegistryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/agenticorp/registry"
	}
	return filepath.Join(home, ".agenticorp", "registry")
}

func containsTag(tags []string, query string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}
