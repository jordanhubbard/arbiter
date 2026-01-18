package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jordanhubbard/arbiter/pkg/secrets"
)

const (
	configFileName = ".arbiter.json"
)

// Provider represents an AI service provider configuration
type Provider struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
}

// Config holds the application configuration
type Config struct {
	Providers   []Provider         `json:"providers"`
	ServerPort  int                `json:"server_port"`
	SecretStore *secrets.Store     `json:"-"`
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		Providers:   []Provider{},
		ServerPort:  8080,
		SecretStore: secrets.NewStore(),
	}
}

// LoadConfig loads configuration from the config file
func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Initialize secret store
	cfg.SecretStore = secrets.NewStore()
	if err := cfg.SecretStore.Load(); err != nil {
		return nil, fmt.Errorf("failed to load secrets: %w", err)
	}

	return &cfg, nil
}

// SaveConfig saves configuration to the config file
func SaveConfig(cfg *Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return err
	}

	// Save secrets separately
	if cfg.SecretStore != nil {
		if err := cfg.SecretStore.Save(); err != nil {
			return fmt.Errorf("failed to save secrets: %w", err)
		}
	}

	return nil
}

// getConfigPath returns the path to the configuration file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configFileName), nil
}

// LookupProviderEndpoint attempts to find the API endpoint for a provider
func LookupProviderEndpoint(providerName string) (string, error) {
	// Map of known providers to their default endpoints
	knownProviders := map[string]string{
		"claude":       "https://api.anthropic.com/v1",
		"openai":       "https://api.openai.com/v1",
		"cursor":       "https://api.cursor.sh/v1",
		"factory":      "https://api.factory.ai/v1",
		"cohere":       "https://api.cohere.ai/v1",
		"huggingface":  "https://api-inference.huggingface.co",
		"replicate":    "https://api.replicate.com/v1",
		"together":     "https://api.together.xyz/v1",
		"mistral":      "https://api.mistral.ai/v1",
		"perplexity":   "https://api.perplexity.ai",
	}

	// Check if we have a known endpoint
	if endpoint, ok := knownProviders[providerName]; ok {
		return endpoint, nil
	}

	// For unknown providers, we would use Google's Custom Search API here
	// For now, return an error to prompt for manual entry
	return "", fmt.Errorf("unknown provider: %s", providerName)
}
