// Package plugin provides the provider plugin interface for Loom.
//
// This package defines the contract that all provider plugins must implement
// to integrate with Loom. Plugins allow users to add support for
// custom AI providers without modifying Loom's source code.
package plugin

import (
	"context"
	"time"
)

// PluginVersion is the current version of the plugin API
const PluginVersion = "1.0.0"

// Plugin is the main interface that all provider plugins must implement.
// A plugin represents an AI provider that can process chat completion requests.
type Plugin interface {
	// GetMetadata returns plugin metadata for registration and discovery
	GetMetadata() *Metadata

	// Initialize is called once when the plugin is loaded.
	// It receives configuration specific to this plugin instance.
	Initialize(ctx context.Context, config map[string]interface{}) error

	// HealthCheck verifies the plugin and provider are operational.
	// This is called periodically and during registration.
	HealthCheck(ctx context.Context) (*HealthStatus, error)

	// CreateChatCompletion sends a chat completion request to the provider.
	// This is the primary method for interacting with the AI provider.
	CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)

	// GetModels returns the list of models supported by this provider.
	// This is used for model discovery and routing.
	GetModels(ctx context.Context) ([]ModelInfo, error)

	// Cleanup is called when the plugin is being unloaded.
	// Use this to release resources, close connections, etc.
	Cleanup(ctx context.Context) error
}

// StreamingPlugin extends Plugin with streaming support.
// Plugins that support streaming responses should implement this interface.
type StreamingPlugin interface {
	Plugin

	// CreateChatCompletionStream sends a streaming chat completion request.
	// Chunks are sent to the provided callback as they arrive.
	CreateChatCompletionStream(ctx context.Context, req *ChatCompletionRequest, callback StreamCallback) error
}

// Metadata describes a plugin for registration and discovery.
type Metadata struct {
	// Name is the human-readable plugin name (e.g., "OpenAI Plugin")
	Name string `json:"name"`

	// Version is the plugin version (semantic versioning recommended)
	Version string `json:"version"`

	// PluginAPIVersion is the version of the plugin API this plugin implements
	PluginAPIVersion string `json:"plugin_api_version"`

	// ProviderType is the provider type identifier (e.g., "openai", "anthropic", "custom-llm")
	ProviderType string `json:"provider_type"`

	// Description provides a brief description of the plugin
	Description string `json:"description"`

	// Author is the plugin author or organization
	Author string `json:"author"`

	// Homepage is the URL to the plugin's homepage or documentation
	Homepage string `json:"homepage,omitempty"`

	// License is the plugin's license (e.g., "MIT", "Apache-2.0")
	License string `json:"license,omitempty"`

	// Capabilities describes what the plugin supports
	Capabilities Capabilities `json:"capabilities"`

	// ConfigSchema describes the configuration fields this plugin accepts
	ConfigSchema []ConfigField `json:"config_schema,omitempty"`
}

// Capabilities describes plugin capabilities.
type Capabilities struct {
	// Streaming indicates if the plugin supports streaming responses
	Streaming bool `json:"streaming"`

	// FunctionCalling indicates if the plugin supports function/tool calling
	FunctionCalling bool `json:"function_calling"`

	// Vision indicates if the plugin supports multimodal/vision inputs
	Vision bool `json:"vision"`

	// Embeddings indicates if the plugin supports generating embeddings
	Embeddings bool `json:"embeddings"`

	// FineTuning indicates if the plugin supports fine-tuning
	FineTuning bool `json:"fine_tuning"`

	// CustomCapabilities allows plugins to declare custom capabilities
	CustomCapabilities map[string]bool `json:"custom_capabilities,omitempty"`
}

// ConfigField describes a configuration field for the plugin.
type ConfigField struct {
	// Name is the field name (e.g., "api_key", "endpoint")
	Name string `json:"name"`

	// Type is the field type ("string", "int", "bool", "float")
	Type string `json:"type"`

	// Required indicates if this field is required
	Required bool `json:"required"`

	// Description explains what this field is for
	Description string `json:"description"`

	// Default is the default value if not provided (optional)
	Default interface{} `json:"default,omitempty"`

	// Sensitive indicates if this field contains sensitive data (e.g., API keys)
	Sensitive bool `json:"sensitive"`

	// Validation contains validation rules (optional)
	Validation *ValidationRule `json:"validation,omitempty"`
}

// ValidationRule defines validation constraints for a config field.
type ValidationRule struct {
	// MinLength for string fields
	MinLength int `json:"min_length,omitempty"`

	// MaxLength for string fields
	MaxLength int `json:"max_length,omitempty"`

	// Pattern is a regex pattern for string validation
	Pattern string `json:"pattern,omitempty"`

	// Min for numeric fields
	Min *float64 `json:"min,omitempty"`

	// Max for numeric fields
	Max *float64 `json:"max,omitempty"`

	// Enum lists allowed values
	Enum []interface{} `json:"enum,omitempty"`
}

// HealthStatus represents the health status of a plugin/provider.
type HealthStatus struct {
	// Healthy indicates if the plugin is operational
	Healthy bool `json:"healthy"`

	// Message provides additional context about the health status
	Message string `json:"message,omitempty"`

	// Latency is the health check latency in milliseconds
	Latency int64 `json:"latency_ms"`

	// Timestamp is when the health check was performed
	Timestamp time.Time `json:"timestamp"`

	// Details provides additional health information
	Details map[string]interface{} `json:"details,omitempty"`
}

// ChatCompletionRequest represents a chat completion request.
type ChatCompletionRequest struct {
	// Model is the model to use for completion
	Model string `json:"model"`

	// Messages is the conversation history
	Messages []ChatMessage `json:"messages"`

	// Temperature controls randomness (0.0-2.0, typically 0.0-1.0)
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxTokens limits the response length
	MaxTokens *int `json:"max_tokens,omitempty"`

	// TopP controls nucleus sampling (0.0-1.0)
	TopP *float64 `json:"top_p,omitempty"`

	// FrequencyPenalty reduces repetition (-2.0 to 2.0)
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// PresencePenalty encourages new topics (-2.0 to 2.0)
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// Stop sequences that stop generation
	Stop []string `json:"stop,omitempty"`

	// Stream indicates if streaming is requested
	Stream bool `json:"stream,omitempty"`

	// User identifier for tracking (optional)
	User string `json:"user,omitempty"`

	// PluginSpecific allows plugins to pass custom parameters
	PluginSpecific map[string]interface{} `json:"plugin_specific,omitempty"`
}

// ChatMessage represents a message in the conversation.
type ChatMessage struct {
	// Role is the message role: "system", "user", "assistant", "function"
	Role string `json:"role"`

	// Content is the message content
	Content string `json:"content"`

	// Name is the function/tool name (for role="function")
	Name string `json:"name,omitempty"`

	// FunctionCall contains function call data (if applicable)
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
}

// FunctionCall represents a function/tool call request.
type FunctionCall struct {
	// Name is the function name
	Name string `json:"name"`

	// Arguments is the function arguments (JSON string)
	Arguments string `json:"arguments"`
}

// ChatCompletionResponse represents a chat completion response.
type ChatCompletionResponse struct {
	// ID is the unique response identifier
	ID string `json:"id"`

	// Object type (e.g., "chat.completion")
	Object string `json:"object"`

	// Created is the Unix timestamp of creation
	Created int64 `json:"created"`

	// Model is the model used
	Model string `json:"model"`

	// Choices contains the completion choices
	Choices []Choice `json:"choices"`

	// Usage contains token usage information
	Usage *UsageInfo `json:"usage,omitempty"`

	// PluginSpecific allows plugins to return custom metadata
	PluginSpecific map[string]interface{} `json:"plugin_specific,omitempty"`
}

// Choice represents a completion choice.
type Choice struct {
	// Index is the choice index
	Index int `json:"index"`

	// Message is the completion message
	Message ChatMessage `json:"message"`

	// FinishReason indicates why generation stopped
	// Values: "stop", "length", "function_call", "content_filter", "null"
	FinishReason string `json:"finish_reason"`
}

// UsageInfo contains token usage statistics.
type UsageInfo struct {
	// PromptTokens is the number of tokens in the prompt
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the total number of tokens
	TotalTokens int `json:"total_tokens"`

	// CostUSD is the estimated cost in USD (optional)
	CostUSD *float64 `json:"cost_usd,omitempty"`
}

// ModelInfo describes a model provided by the plugin.
type ModelInfo struct {
	// ID is the unique model identifier
	ID string `json:"id"`

	// Name is the human-readable model name
	Name string `json:"name"`

	// Description describes the model's capabilities
	Description string `json:"description,omitempty"`

	// ContextWindow is the maximum context size in tokens
	ContextWindow int `json:"context_window,omitempty"`

	// MaxOutputTokens is the maximum output length
	MaxOutputTokens int `json:"max_output_tokens,omitempty"`

	// CostPerMToken is the cost per million tokens in USD
	CostPerMToken *float64 `json:"cost_per_mtoken,omitempty"`

	// Capabilities describes model-specific capabilities
	Capabilities Capabilities `json:"capabilities"`

	// Deprecated indicates if this model is deprecated
	Deprecated bool `json:"deprecated,omitempty"`

	// Metadata contains additional model information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// StreamCallback is called for each chunk in a streaming response.
type StreamCallback func(chunk *StreamChunk) error

// StreamChunk represents a chunk in a streaming response.
type StreamChunk struct {
	// ID is the unique response identifier
	ID string `json:"id"`

	// Object type (e.g., "chat.completion.chunk")
	Object string `json:"object"`

	// Created is the Unix timestamp
	Created int64 `json:"created"`

	// Model is the model used
	Model string `json:"model"`

	// Choices contains delta choices
	Choices []StreamChoice `json:"choices"`

	// Done indicates if this is the final chunk
	Done bool `json:"done"`

	// Usage contains final token usage (present in last chunk)
	Usage *UsageInfo `json:"usage,omitempty"`
}

// StreamChoice represents a streaming choice with delta content.
type StreamChoice struct {
	// Index is the choice index
	Index int `json:"index"`

	// Delta contains the incremental message content
	Delta ChatMessage `json:"delta"`

	// FinishReason indicates why generation stopped (present in last chunk)
	FinishReason string `json:"finish_reason,omitempty"`
}

// PluginError represents an error from a plugin.
type PluginError struct {
	// Code is a machine-readable error code
	Code string `json:"code"`

	// Message is a human-readable error message
	Message string `json:"message"`

	// Details contains additional error information
	Details map[string]interface{} `json:"details,omitempty"`

	// Transient indicates if the error is temporary (retry-able)
	Transient bool `json:"transient"`
}

// Error implements the error interface.
func (e *PluginError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

// NewPluginError creates a new plugin error.
func NewPluginError(code, message string, transient bool) *PluginError {
	return &PluginError{
		Code:      code,
		Message:   message,
		Transient: transient,
	}
}

// Common error codes that plugins should use
const (
	ErrorCodeAuthenticationFailed = "authentication_failed"
	ErrorCodeRateLimitExceeded    = "rate_limit_exceeded"
	ErrorCodeInvalidRequest       = "invalid_request"
	ErrorCodeModelNotFound        = "model_not_found"
	ErrorCodeProviderUnavailable  = "provider_unavailable"
	ErrorCodeTimeout              = "timeout"
	ErrorCodeInternalError        = "internal_error"
	ErrorCodeQuotaExceeded        = "quota_exceeded"
	ErrorCodeContentFilter        = "content_filter"
)
