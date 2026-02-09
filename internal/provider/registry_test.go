package provider_test

import (
	"testing"

	"github.com/jordanhubbard/loom/internal/provider"
)

func TestProviderRegistry(t *testing.T) {
	registry := provider.NewRegistry()

	// Test registering a provider
	config := &provider.ProviderConfig{
		ID:       "test-provider",
		Name:     "Test Provider",
		Type:     "openai",
		Endpoint: "http://localhost:8000/v1",
		APIKey:   "test-key",
		Model:    "test-model",
	}

	err := registry.Register(config)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	// Test getting the provider
	registered, err := registry.Get("test-provider")
	if err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if registered.Config.Name != "Test Provider" {
		t.Errorf("Expected provider name 'Test Provider', got '%s'", registered.Config.Name)
	}

	// Test listing providers
	providers := registry.List()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	// Test duplicate registration
	err = registry.Register(config)
	if err == nil {
		t.Error("Expected error when registering duplicate provider")
	}

	// Test unregister
	err = registry.Unregister("test-provider")
	if err != nil {
		t.Fatalf("Failed to unregister provider: %v", err)
	}

	providers = registry.List()
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers after unregister, got %d", len(providers))
	}
}

func TestProviderProtocol(t *testing.T) {
	// Create a provider with OpenAI protocol
	prov := provider.NewOpenAIProvider("http://localhost:8000/v1", "test-key")

	if prov == nil {
		t.Fatal("Failed to create OpenAI provider")
	}

	// Note: We can't test actual API calls without a real endpoint
	// This test just verifies the provider can be created
	t.Log("OpenAI provider created successfully")
}

func TestChatCompletionRequest(t *testing.T) {
	req := &provider.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "Hello!"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	if req.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", req.Model)
	}

	if len(req.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(req.Messages))
	}
}

func TestListActiveIncludesActiveAndHealthy(t *testing.T) {
	registry := provider.NewRegistry()

	// Register provider with "active" status (set by health check on startup)
	activeConfig := &provider.ProviderConfig{
		ID:       "prov-active",
		Name:     "Active Provider",
		Type:     "openai",
		Endpoint: "http://localhost:8000/v1",
		APIKey:   "key",
		Model:    "model",
		Status:   "active",
	}
	if err := registry.Register(activeConfig); err != nil {
		t.Fatalf("Register active: %v", err)
	}

	// Register provider with "healthy" status (set by heartbeat workflow)
	healthyConfig := &provider.ProviderConfig{
		ID:       "prov-healthy",
		Name:     "Healthy Provider",
		Type:     "openai",
		Endpoint: "http://localhost:8000/v1",
		APIKey:   "key",
		Model:    "model",
		Status:   "healthy",
	}
	if err := registry.Register(healthyConfig); err != nil {
		t.Fatalf("Register healthy: %v", err)
	}

	// Register provider with "pending" status (should NOT be active)
	pendingConfig := &provider.ProviderConfig{
		ID:       "prov-pending",
		Name:     "Pending Provider",
		Type:     "openai",
		Endpoint: "http://localhost:8000/v1",
		APIKey:   "key",
		Model:    "model",
		Status:   "pending",
	}
	if err := registry.Register(pendingConfig); err != nil {
		t.Fatalf("Register pending: %v", err)
	}

	active := registry.ListActive()
	if len(active) != 2 {
		t.Errorf("ListActive() returned %d providers, want 2 (active + healthy)", len(active))
	}

	if !registry.IsActive("prov-active") {
		t.Error("IsActive(prov-active) = false, want true")
	}
	if !registry.IsActive("prov-healthy") {
		t.Error("IsActive(prov-healthy) = false, want true")
	}
	if registry.IsActive("prov-pending") {
		t.Error("IsActive(prov-pending) = true, want false")
	}

	// Cleanup
	_ = registry.Unregister("prov-active")
	_ = registry.Unregister("prov-healthy")
	_ = registry.Unregister("prov-pending")
}

func TestProviderTypes(t *testing.T) {
	registry := provider.NewRegistry()

	testCases := []struct {
		name     string
		provType string
		wantErr  bool
	}{
		{"OpenAI type", "openai", false},
		{"Anthropic type", "anthropic", false},
		{"Local type", "local", false},
		{"Custom type", "custom", false},
		{"Unknown type", "unknown", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &provider.ProviderConfig{
				ID:       "test-" + tc.provType,
				Name:     "Test " + tc.provType,
				Type:     tc.provType,
				Endpoint: "http://localhost:8000/v1",
				Model:    "test-model",
			}

			err := registry.Register(config)
			if (err != nil) != tc.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tc.wantErr)
			}

			// Clean up successful registrations
			if err == nil {
				_ = registry.Unregister(config.ID)
			}
		})
	}
}
