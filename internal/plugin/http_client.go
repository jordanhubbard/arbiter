package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jordanhubbard/agenticorp/pkg/plugin"
)

// HTTPPluginClient implements the plugin.Plugin interface over HTTP.
// This allows plugins to run as separate processes, providing isolation.
type HTTPPluginClient struct {
	endpoint string
	client   *http.Client
	metadata *plugin.Metadata
}

// NewHTTPPluginClient creates a new HTTP plugin client.
func NewHTTPPluginClient(endpoint string) (*HTTPPluginClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	return &HTTPPluginClient{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GetMetadata returns plugin metadata.
func (c *HTTPPluginClient) GetMetadata() *plugin.Metadata {
	if c.metadata != nil {
		return c.metadata
	}

	// Fetch metadata from plugin
	ctx := context.Background()
	resp, err := c.doRequest(ctx, "GET", "/metadata", nil)
	if err != nil {
		return nil
	}

	var metadata plugin.Metadata
	if err := json.Unmarshal(resp, &metadata); err != nil {
		return nil
	}

	c.metadata = &metadata
	return c.metadata
}

// Initialize initializes the plugin with configuration.
func (c *HTTPPluginClient) Initialize(ctx context.Context, config map[string]interface{}) error {
	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	_, err = c.doRequest(ctx, "POST", "/initialize", body)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	// Cache metadata after initialization
	c.GetMetadata()

	return nil
}

// HealthCheck performs a health check on the plugin.
func (c *HTTPPluginClient) HealthCheck(ctx context.Context) (*plugin.HealthStatus, error) {
	start := time.Now()

	resp, err := c.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		latency := time.Since(start).Milliseconds()
		return &plugin.HealthStatus{
			Healthy:   false,
			Message:   err.Error(),
			Latency:   latency,
			Timestamp: time.Now(),
		}, nil
	}

	var health plugin.HealthStatus
	if err := json.Unmarshal(resp, &health); err != nil {
		return nil, fmt.Errorf("failed to parse health response: %w", err)
	}

	return &health, nil
}

// CreateChatCompletion sends a chat completion request.
func (c *HTTPPluginClient) CreateChatCompletion(ctx context.Context, req *plugin.ChatCompletionRequest) (*plugin.ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", "/chat/completions", body)
	if err != nil {
		return nil, fmt.Errorf("completion request failed: %w", err)
	}

	var completion plugin.ChatCompletionResponse
	if err := json.Unmarshal(resp, &completion); err != nil {
		return nil, fmt.Errorf("failed to parse completion response: %w", err)
	}

	return &completion, nil
}

// GetModels retrieves the list of available models.
func (c *HTTPPluginClient) GetModels(ctx context.Context) ([]plugin.ModelInfo, error) {
	resp, err := c.doRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return nil, fmt.Errorf("models request failed: %w", err)
	}

	var models []plugin.ModelInfo
	if err := json.Unmarshal(resp, &models); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	return models, nil
}

// Cleanup performs plugin cleanup.
func (c *HTTPPluginClient) Cleanup(ctx context.Context) error {
	_, err := c.doRequest(ctx, "POST", "/cleanup", nil)
	if err != nil {
		return fmt.Errorf("cleanup request failed: %w", err)
	}

	return nil
}

// doRequest performs an HTTP request to the plugin.
func (c *HTTPPluginClient) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	url := c.endpoint + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse as PluginError
		var pluginErr plugin.PluginError
		if err := json.Unmarshal(respBody, &pluginErr); err == nil {
			return nil, &pluginErr
		}

		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
