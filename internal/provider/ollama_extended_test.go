package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// OllamaProvider: GetModels
// ---------------------------------------------------------------------------

func TestOllamaProvider_GetModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("expected /api/tags, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"models": [
				{"name": "llama2:7b"},
				{"name": "codellama:13b"},
				{"name": "mistral:7b"}
			]
		}`))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	models, err := p.GetModels(context.Background())
	if err != nil {
		t.Fatalf("GetModels: %v", err)
	}
	if len(models) != 3 {
		t.Errorf("expected 3 models, got %d", len(models))
	}
	if models[0].ID != "llama2:7b" {
		t.Errorf("model[0].ID = %q, want %q", models[0].ID, "llama2:7b")
	}
	if models[0].Object != "model" {
		t.Errorf("model[0].Object = %q, want %q", models[0].Object, "model")
	}
}

func TestOllamaProvider_GetModels_EmptyModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models": []}`))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	models, err := p.GetModels(context.Background())
	if err != nil {
		t.Fatalf("GetModels: %v", err)
	}
	if len(models) != 0 {
		t.Errorf("expected 0 models, got %d", len(models))
	}
}

func TestOllamaProvider_GetModels_SkipsEmptyNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"models": [
				{"name": "llama2"},
				{"name": ""},
				{"name": "  "}
			]
		}`))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	models, err := p.GetModels(context.Background())
	if err != nil {
		t.Fatalf("GetModels: %v", err)
	}
	if len(models) != 1 {
		t.Errorf("expected 1 model (skipping empty names), got %d", len(models))
	}
}

func TestOllamaProvider_GetModels_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	_, err := p.GetModels(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOllamaProvider_GetModels_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	_, err := p.GetModels(context.Background())
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

// ---------------------------------------------------------------------------
// OllamaProvider: CreateChatCompletion
// ---------------------------------------------------------------------------

func TestOllamaProvider_CreateChatCompletion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected /api/chat, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Verify request body
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "llama2" {
			t.Errorf("expected model llama2, got %v", body["model"])
		}
		if body["stream"] != false {
			t.Errorf("expected stream=false, got %v", body["stream"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "llama2",
			"message": {"role": "assistant", "content": "Hello! How can I help?"},
			"done": true
		}`))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	resp, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model: "llama2",
		Messages: []ChatMessage{
			{Role: "user", Content: "hi"},
		},
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("CreateChatCompletion: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello! How can I help?" {
		t.Errorf("content = %q", resp.Choices[0].Message.Content)
	}
	if resp.Choices[0].Finish != "stop" {
		t.Errorf("finish = %q, want stop", resp.Choices[0].Finish)
	}
	if resp.Model != "llama2" {
		t.Errorf("model = %q, want llama2", resp.Model)
	}
}

func TestOllamaProvider_CreateChatCompletion_EmptyModel(t *testing.T) {
	p := NewOllamaProvider("http://localhost:11434")
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for empty model")
	}
	if !strings.Contains(err.Error(), "model is required") {
		t.Errorf("expected 'model is required', got %v", err)
	}
}

func TestOllamaProvider_CreateChatCompletion_WhitespaceModel(t *testing.T) {
	p := NewOllamaProvider("http://localhost:11434")
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "  ",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only model")
	}
}

func TestOllamaProvider_CreateChatCompletion_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "llama2",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOllamaProvider_CreateChatCompletion_ContextLengthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "context length exceeded"}`))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "llama2",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var cle *ContextLengthError
	if !errors.As(err, &cle) {
		t.Errorf("expected ContextLengthError, got %T: %v", err, err)
	}
}

func TestOllamaProvider_CreateChatCompletion_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "llama2",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestOllamaProvider_CreateChatCompletion_WithJSONFormat(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "llama2",
			"message": {"role": "assistant", "content": "{}"},
			"done": true
		}`))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:          "llama2",
		Messages:       []ChatMessage{{Role: "user", Content: "give json"}},
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["format"] != "json" {
		t.Errorf("expected format=json, got %v", receivedBody["format"])
	}
}

func TestOllamaProvider_CreateChatCompletion_MultipleMessages(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "llama2",
			"message": {"role": "assistant", "content": "ok"},
			"done": true
		}`))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model: "llama2",
		Messages: []ChatMessage{
			{Role: "system", Content: "you are helpful"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "how are you"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs, ok := receivedBody["messages"].([]interface{})
	if !ok {
		t.Fatal("expected messages array in body")
	}
	if len(msgs) != 4 {
		t.Errorf("expected 4 messages, got %d", len(msgs))
	}
}

// ---------------------------------------------------------------------------
// OllamaProvider: Streaming - empty model
// ---------------------------------------------------------------------------

func TestOllamaProvider_Streaming_EmptyModel(t *testing.T) {
	p := NewOllamaProvider("http://localhost:11434")
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error { return nil })
	if err == nil {
		t.Fatal("expected error for empty model")
	}
}

// ---------------------------------------------------------------------------
// OllamaProvider: Streaming - server error
// ---------------------------------------------------------------------------

func TestOllamaProvider_Streaming_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "llama2",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error { return nil })
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// OllamaProvider: Streaming - handler error
// ---------------------------------------------------------------------------

func TestOllamaProvider_Streaming_HandlerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"llama2","message":{"role":"assistant","content":"hi"},"done":false}` + "\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	handlerErr := errors.New("handler failed")
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "llama2",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error {
		return handlerErr
	})
	if err == nil {
		t.Fatal("expected error from handler")
	}
	if !strings.Contains(err.Error(), "handler error") {
		t.Errorf("expected handler error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// OllamaProvider: Streaming - context cancellation
// ---------------------------------------------------------------------------

func TestOllamaProvider_Streaming_ContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Send first chunk then wait for context cancellation
		_, _ = w.Write([]byte(`{"model":"llama2","message":{"role":"assistant","content":"hi"},"done":false}` + "\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Block
		<-r.Context().Done()
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	ctx, cancel := context.WithCancel(context.Background())

	chunkCount := 0
	err := p.CreateChatCompletionStream(ctx, &ChatCompletionRequest{
		Model:    "llama2",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error {
		chunkCount++
		cancel()
		return nil
	})
	// Error may or may not appear depending on timing, but should not panic
	_ = err
}

// ---------------------------------------------------------------------------
// OllamaProvider: Streaming - invalid JSON chunks (skipped)
// ---------------------------------------------------------------------------

func TestOllamaProvider_Streaming_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Invalid JSON followed by valid
		_, _ = w.Write([]byte("not json\n"))
		_, _ = w.Write([]byte(`{"model":"llama2","message":{"role":"assistant","content":"ok"},"done":true}` + "\n"))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	var chunks []*StreamChunk
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "llama2",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 valid chunk, got %d", len(chunks))
	}
}

// ---------------------------------------------------------------------------
// OllamaProvider: Trailing slash
// ---------------------------------------------------------------------------

func TestOllamaProvider_TrailingSlash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "//") {
			t.Errorf("double slash in path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models": []}`))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL + "/")
	_, _ = p.GetModels(context.Background())
}

// ---------------------------------------------------------------------------
// OllamaProvider: Streaming - empty lines skipped
// ---------------------------------------------------------------------------

func TestOllamaProvider_Streaming_EmptyLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("\n"))
		_, _ = w.Write([]byte(`{"model":"llama2","message":{"role":"assistant","content":"hi"},"done":true}` + "\n"))
		_, _ = w.Write([]byte("\n"))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	var chunks []*StreamChunk
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "llama2",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk (empty lines skipped), got %d", len(chunks))
	}
}
