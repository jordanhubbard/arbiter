package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaProvider implements Protocol for Ollama-compatible APIs.
// See: https://github.com/ollama/ollama/blob/main/docs/api.md
type OllamaProvider struct {
	endpoint string
	client   *http.Client
}

func NewOllamaProvider(endpoint string) *OllamaProvider {
	return &OllamaProvider{
		endpoint: strings.TrimSuffix(endpoint, "/"),
		client: &http.Client{
			Timeout: 15 * time.Minute, // Increased for action loops with 25 iterations
		},
	}
}

func (p *OllamaProvider) GetModels(ctx context.Context) ([]Model, error) {
	url := fmt.Sprintf("%s/api/tags", p.endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	models := make([]Model, 0, len(tagsResp.Models))
	for _, m := range tagsResp.Models {
		if strings.TrimSpace(m.Name) == "" {
			continue
		}
		models = append(models, Model{ID: m.Name, Object: "model"})
	}

	return models, nil
}

func (p *OllamaProvider) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	url := fmt.Sprintf("%s/api/chat", p.endpoint)
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return nil, fmt.Errorf("model is required")
	}

	ollamaReq := struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream  bool   `json:"stream"`
		Format  string `json:"format,omitempty"`
		Options struct {
			Temperature float64 `json:"temperature,omitempty"`
		} `json:"options,omitempty"`
	}{
		Model:  model,
		Stream: false,
	}
	ollamaReq.Options.Temperature = req.Temperature
	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_object" {
		ollamaReq.Format = "json"
	}
	for _, msg := range req.Messages {
		ollamaReq.Messages = append(ollamaReq.Messages, struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{Role: msg.Role, Content: msg.Content})
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		bodyStr := string(respBody)
		if resp.StatusCode == http.StatusBadRequest && isContextLengthError(bodyStr) {
			return nil, &ContextLengthError{StatusCode: resp.StatusCode, Body: bodyStr}
		}
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, bodyStr)
	}

	var ollamaResp struct {
		Model   string `json:"model"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	completion := &ChatCompletionResponse{Model: ollamaResp.Model}
	completion.Choices = append(completion.Choices, struct {
		Index   int         `json:"index"`
		Message ChatMessage `json:"message"`
		Finish  string      `json:"finish_reason"`
	}{
		Index:   0,
		Message: ChatMessage{Role: ollamaResp.Message.Role, Content: ollamaResp.Message.Content},
		Finish:  "stop",
	})

	return completion, nil
}
