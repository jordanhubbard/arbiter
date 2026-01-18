package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jordanhubbard/arbiter/pkg/config"
)

// Server represents the Arbiter HTTP server
type Server struct {
	config *config.Config
	mux    *http.ServeMux
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config) *Server {
	s := &Server{
		config: cfg,
		mux:    http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Web frontend
	s.mux.HandleFunc("/", s.handleHome)
	s.mux.HandleFunc("/api/providers", s.handleProviders)

	// OpenAI-compatible API
	s.mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	s.mux.HandleFunc("/v1/completions", s.handleCompletions)
	s.mux.HandleFunc("/v1/models", s.handleModels)

	// Health check
	s.mux.HandleFunc("/health", s.handleHealth)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.ServerPort)

	fmt.Printf("\n✓ Arbiter server started successfully!\n")
	fmt.Printf("\n  Web Interface:  http://localhost%s\n", addr)
	fmt.Printf("  OpenAI API:     http://localhost%s/v1/...\n", addr)
	fmt.Printf("  Health Check:   http://localhost%s/health\n\n", addr)
	fmt.Println("Press Ctrl+C to stop the server")

	server := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server.ListenAndServe()
}

// handleHome serves the web frontend
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := getHomeHTML(s.config)
	w.Write([]byte(html))
}

// handleProviders returns configured providers
func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": s.config.Providers,
		"count":     len(s.config.Providers),
	})
}

// handleChatCompletions handles OpenAI-compatible chat completions
func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Route to appropriate provider based on model or default
	provider := s.selectProvider(req.Model)
	if provider == nil {
		http.Error(w, "No providers configured", http.StatusServiceUnavailable)
		return
	}

	// Forward request to provider
	resp, err := s.forwardToProvider(provider, "/chat/completions", req)
	if err != nil {
		log.Printf("Error forwarding to provider: %v", err)
		http.Error(w, "Provider request failed", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleCompletions handles OpenAI-compatible completions
func (s *Server) handleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	provider := s.selectProvider(req.Model)
	if provider == nil {
		http.Error(w, "No providers configured", http.StatusServiceUnavailable)
		return
	}

	resp, err := s.forwardToProvider(provider, "/completions", req)
	if err != nil {
		log.Printf("Error forwarding to provider: %v", err)
		http.Error(w, "Provider request failed", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleModels lists available models from all providers
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	models := []Model{}

	for _, provider := range s.config.Providers {
		// Add generic model entry for each provider
		models = append(models, Model{
			ID:      fmt.Sprintf("%s-default", provider.Name),
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: provider.Name,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   models,
	})
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"providers": len(s.config.Providers),
		"timestamp": time.Now().Unix(),
	})
}

// selectProvider selects a provider based on model name or returns the first one
func (s *Server) selectProvider(modelName string) *config.Provider {
	if len(s.config.Providers) == 0 {
		return nil
	}

	// Try to match provider by name in model
	for i := range s.config.Providers {
		providerName := s.config.Providers[i].Name
		if modelName != "" && providerName != "" {
			// Simple substring match - check length first to avoid panic
			if len(modelName) >= len(providerName) &&
				modelName[:len(providerName)] == providerName {
				return &s.config.Providers[i]
			}
		}
	}

	// Default to first provider
	return &s.config.Providers[0]
}

// forwardToProvider forwards a request to the specified provider
func (s *Server) forwardToProvider(provider *config.Provider, endpoint string, request interface{}) (interface{}, error) {
	// NOTE: This is a placeholder implementation that returns mock data
	// In a production system, this would:
	// 1. Retrieve the API key from the secret store
	// 2. Make an HTTP request to provider.Endpoint + endpoint
	// 3. Forward the request with proper authentication headers
	// 4. Return the actual provider response

	log.Printf("NOTE: Forwarding to %s is not yet implemented. Returning mock response.", provider.Name)

	return map[string]interface{}{
		"id":      fmt.Sprintf("arbiter-%d", time.Now().Unix()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   provider.Name,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": fmt.Sprintf("Mock response from %s provider. Forwarding logic needs to be implemented for production use.", provider.Name),
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
		},
	}, nil
}

// getHomeHTML returns the HTML for the web frontend
func getHomeHTML(cfg *config.Config) string {
	providersHTML := ""
	for _, p := range cfg.Providers {
		providersHTML += fmt.Sprintf(`
			<tr>
				<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
				<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
				<td style="padding: 8px; border: 1px solid #ddd;">✓ Configured</td>
			</tr>`, p.Name, p.Endpoint)
	}

	if providersHTML == "" {
		providersHTML = `<tr><td colspan="3" style="padding: 8px; text-align: center;">No providers configured</td></tr>`
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Arbiter - AI Coding Agent Orchestrator</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
			max-width: 1200px;
			margin: 0 auto;
			padding: 20px;
			background-color: #f5f5f5;
		}
		.header {
			background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
			color: white;
			padding: 30px;
			border-radius: 10px;
			margin-bottom: 30px;
		}
		.header h1 {
			margin: 0 0 10px 0;
		}
		.card {
			background: white;
			border-radius: 10px;
			padding: 25px;
			margin-bottom: 20px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
		}
		.card h2 {
			margin-top: 0;
			color: #333;
		}
		table {
			width: 100%%;
			border-collapse: collapse;
		}
		th {
			background-color: #667eea;
			color: white;
			padding: 12px;
			text-align: left;
		}
		.endpoint {
			background: #f8f9fa;
			padding: 15px;
			border-radius: 5px;
			font-family: monospace;
			margin: 10px 0;
		}
		.status {
			display: inline-block;
			padding: 5px 10px;
			background-color: #10b981;
			color: white;
			border-radius: 5px;
			font-size: 14px;
		}
	</style>
</head>
<body>
	<div class="header">
		<h1>🤖 Arbiter</h1>
		<p>AI Coding Agent Orchestrator & Dispatcher</p>
		<div class="status">Server Running</div>
	</div>
	
	<div class="card">
		<h2>📊 Configured Providers</h2>
		<table>
			<thead>
				<tr>
					<th>Provider</th>
					<th>Endpoint</th>
					<th>Status</th>
				</tr>
			</thead>
			<tbody>
				%s
			</tbody>
		</table>
	</div>
	
	<div class="card">
		<h2>🔌 API Endpoints</h2>
		<p>Arbiter provides an OpenAI-compatible API. Use these endpoints with your favorite tools:</p>
		<div class="endpoint">POST /v1/chat/completions</div>
		<div class="endpoint">POST /v1/completions</div>
		<div class="endpoint">GET /v1/models</div>
		<div class="endpoint">GET /health</div>
		<div class="endpoint">GET /api/providers</div>
	</div>
	
	<div class="card">
		<h2>📖 Usage</h2>
		<p>Configure your API client to use Arbiter as the base URL:</p>
		<div class="endpoint">http://localhost:%d/v1</div>
		<p>Arbiter will automatically route requests to the appropriate AI provider based on your configuration.</p>
	</div>
</body>
</html>`, providersHTML, cfg.ServerPort)
}
