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
// ContextLengthError
// ---------------------------------------------------------------------------

func TestContextLengthError_Error(t *testing.T) {
	e := &ContextLengthError{StatusCode: 400, Body: "context length exceeded"}
	msg := e.Error()
	if !strings.Contains(msg, "400") {
		t.Errorf("error should contain status code, got %q", msg)
	}
	if !strings.Contains(msg, "context length exceeded") {
		t.Errorf("error should contain body, got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// isContextLengthError
// ---------------------------------------------------------------------------

func TestIsContextLengthError(t *testing.T) {
	tests := []struct {
		body   string
		expect bool
	}{
		{"context length exceeded", true},
		{"context_length_exceeded", true},
		{"prompt is too long", true},
		{"input is too long", true},
		{"maximum context window", true},
		{"token limit reached", true},
		{"too many tokens in request", true},
		{"model cannot exceed 4096", true}, // contains "exceed"
		{"something else entirely", false},
		{"", false},
		{"CONTEXT LENGTH EXCEEDED", true}, // case insensitive
	}

	for _, tt := range tests {
		got := isContextLengthError(tt.body)
		if got != tt.expect {
			t.Errorf("isContextLengthError(%q) = %v, want %v", tt.body, got, tt.expect)
		}
	}
}

// ---------------------------------------------------------------------------
// findClosingBrace edge cases
// ---------------------------------------------------------------------------

func TestFindClosingBrace_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty object", `{}`, 1},
		{"nested", `{"a":{"b":{}}}`, 13},
		{"with strings containing braces", `{"a":"{}", "b":"}}"}`, 19},
		{"escaped quotes in strings", `{"a":"\""}`, 9},
		{"no closing", `{"a":`, -1},
		{"deeply nested", `{"a":{"b":{"c":{}}}}`, 19},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findClosingBrace([]byte(tt.input))
			if got != tt.want {
				t.Errorf("findClosingBrace(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// findClosingBracket edge cases
// ---------------------------------------------------------------------------

func TestFindClosingBracket_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty array", `[]`, 1},
		{"nested", `[[1,2],[3]]`, 10},
		{"with strings containing brackets", `["[a]", "]"]`, 11},
		{"no closing", `[1, 2, `, -1},
		{"mixed nesting", `[{"a":[1]}]`, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findClosingBracket([]byte(tt.input))
			if got != tt.want {
				t.Errorf("findClosingBracket(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// extractJSON additional cases
// ---------------------------------------------------------------------------

func TestExtractJSON_ArrayWithPrefix(t *testing.T) {
	// extractJSON prefers objects over arrays, so for mixed input it finds the first object
	// Test pure array (no embedded objects) to exercise array path
	input := `here is the data: [1, 2, 3]`
	got := extractJSON([]byte(input))
	if got == nil {
		t.Fatal("expected non-nil JSON")
	}
	if string(got) != `[1, 2, 3]` {
		t.Errorf("extractJSON = %q", string(got))
	}
}

func TestExtractJSON_EmptyInput(t *testing.T) {
	got := extractJSON([]byte(""))
	if got != nil {
		t.Errorf("expected nil for empty input, got %q", string(got))
	}
}

func TestExtractJSON_WhitespaceOnly(t *testing.T) {
	got := extractJSON([]byte("   \n\t  "))
	if got != nil {
		t.Errorf("expected nil for whitespace, got %q", string(got))
	}
}

func TestExtractJSON_NoJSONPresent(t *testing.T) {
	got := extractJSON([]byte("plain text with no json"))
	if got != nil {
		t.Errorf("expected nil, got %q", string(got))
	}
}

// ---------------------------------------------------------------------------
// unmarshalJSON additional cases
// ---------------------------------------------------------------------------

func TestUnmarshalJSON_JSONArray(t *testing.T) {
	// extractJSON prefers objects over arrays; verify it still succeeds on plain array input
	input := `[{"id":"a"}, {"id":"b"}]`
	var result []map[string]interface{}
	err := unmarshalJSON([]byte(input), &result)
	if err != nil {
		t.Fatalf("unmarshalJSON: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
}

func TestUnmarshalJSON_EmptyString(t *testing.T) {
	var result map[string]interface{}
	err := unmarshalJSON([]byte(""), &result)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

// ---------------------------------------------------------------------------
// OpenAIProvider: CreateChatCompletion via httptest
// ---------------------------------------------------------------------------

func TestOpenAIProvider_CreateChatCompletion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected json content type, got %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer auth, got %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		resp := ChatCompletionResponse{
			ID:      "test-resp",
			Object:  "chat.completion",
			Model:   "gpt-4",
			Created: 1234,
		}
		resp.Choices = append(resp.Choices, struct {
			Index   int         `json:"index"`
			Message ChatMessage `json:"message"`
			Finish  string      `json:"finish_reason"`
		}{
			Index:   0,
			Message: ChatMessage{Role: "assistant", Content: "hello back"},
			Finish:  "stop",
		})
		resp.Usage.TotalTokens = 20
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "test-key")
	resp, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "gpt-4",
		Messages: []ChatMessage{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("CreateChatCompletion: %v", err)
	}
	if resp.ID != "test-resp" {
		t.Errorf("ID = %q, want %q", resp.ID, "test-resp")
	}
	if resp.Choices[0].Message.Content != "hello back" {
		t.Errorf("content = %q", resp.Choices[0].Message.Content)
	}
}

func TestOpenAIProvider_CreateChatCompletion_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authorization header should not be present
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1","choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "")
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIProvider_CreateChatCompletion_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain 500, got %v", err)
	}
}

func TestOpenAIProvider_CreateChatCompletion_ContextLengthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "context length exceeded: 8192 tokens"}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "m",
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

func TestOpenAIProvider_CreateChatCompletion_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json at all`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	_, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestOpenAIProvider_CreateChatCompletion_WithResponseFormat(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1","choices":[{"message":{"content":"{}"}}]}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	_, _ = p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:          "m",
		Messages:       []ChatMessage{{Role: "user", Content: "hi"}},
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	})

	// Verify response_format was sent
	if rf, ok := receivedBody["response_format"].(map[string]interface{}); ok {
		if rf["type"] != "json_object" {
			t.Errorf("response_format.type = %v, want json_object", rf["type"])
		}
	} else {
		t.Error("response_format not found in request body")
	}
}

// ---------------------------------------------------------------------------
// OpenAIProvider: GetModels via httptest
// ---------------------------------------------------------------------------

func TestOpenAIProvider_GetModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("expected /models path, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": "model-1", "object": "model", "created": 1000, "owned_by": "test"},
				{"id": "model-2", "object": "model", "created": 2000, "owned_by": "test"}
			]
		}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	models, err := p.GetModels(context.Background())
	if err != nil {
		t.Fatalf("GetModels: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "model-1" {
		t.Errorf("model[0].ID = %q", models[0].ID)
	}
}

func TestOpenAIProvider_GetModels_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	_, err := p.GetModels(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenAIProvider_GetModels_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	_, err := p.GetModels(context.Background())
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestOpenAIProvider_GetModels_NoAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no auth header, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "")
	_, err := p.GetModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// OpenAIProvider: Endpoint trailing slash
// ---------------------------------------------------------------------------

func TestOpenAIProvider_TrailingSlash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not have double slash
		if strings.Contains(r.URL.Path, "//") {
			t.Errorf("double slash in path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL+"/", "key")
	_, _ = p.GetModels(context.Background())
}

// ---------------------------------------------------------------------------
// OpenAIProvider: Context cancellation
// ---------------------------------------------------------------------------

func TestOpenAIProvider_CreateChatCompletion_ContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until context is cancelled
		<-r.Context().Done()
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := p.CreateChatCompletion(ctx, &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ---------------------------------------------------------------------------
// OpenAIProvider: Streaming with context length error
// ---------------------------------------------------------------------------

func TestOpenAIProvider_Streaming_ContextLengthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "too many tokens in the prompt"}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error { return nil })

	if err == nil {
		t.Fatal("expected error")
	}
	var cle *ContextLengthError
	if !errors.As(err, &cle) {
		t.Errorf("expected ContextLengthError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// OpenAIProvider: Streaming server error (non-context-length)
// ---------------------------------------------------------------------------

func TestOpenAIProvider_Streaming_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error { return nil })

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 in error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// OpenAIProvider: Streaming with empty data (no chunks)
// ---------------------------------------------------------------------------

func TestOpenAIProvider_Streaming_EmptyStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Close immediately without data
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error { return nil })

	if err == nil {
		t.Fatal("expected error for empty stream")
	}
	if !strings.Contains(err.Error(), "without receiving any data") {
		t.Errorf("expected 'without receiving any data' error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// OpenAIProvider: Streaming handler error
// ---------------------------------------------------------------------------

func TestOpenAIProvider_Streaming_HandlerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"id":"1","choices":[{"delta":{"content":"hi"}}]}` + "\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	handlerErr := errors.New("handler failed")
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "m",
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
// OpenAIProvider: Streaming with SSE comments and empty lines
// ---------------------------------------------------------------------------

func TestOpenAIProvider_Streaming_SSEComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// SSE comments (lines starting with :) and empty lines should be skipped
		lines := []string{
			": this is a comment",
			"",
			`data: {"id":"1","choices":[{"delta":{"content":"ok"}}]}`,
			"",
			"data: [DONE]",
		}
		for _, l := range lines {
			_, _ = w.Write([]byte(l + "\n"))
		}
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	var chunks []*StreamChunk
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
}

// ---------------------------------------------------------------------------
// OpenAIProvider: Streaming with invalid JSON chunk (should skip)
// ---------------------------------------------------------------------------

func TestOpenAIProvider_Streaming_InvalidJSONChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		lines := []string{
			`data: {invalid json}`,
			`data: {"id":"1","choices":[{"delta":{"content":"ok"}}]}`,
			"data: [DONE]",
		}
		for _, l := range lines {
			_, _ = w.Write([]byte(l + "\n"))
		}
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	var chunks []*StreamChunk
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Invalid JSON chunk should be skipped, only valid one received
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk (skipping invalid), got %d", len(chunks))
	}
}

// ---------------------------------------------------------------------------
// OpenAIProvider: Streaming with non-SSE lines (no "data: " prefix)
// ---------------------------------------------------------------------------

func TestOpenAIProvider_Streaming_NonSSELines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		lines := []string{
			"event: message",
			`data: {"id":"1","choices":[{"delta":{"content":"hi"}}]}`,
			"id: 123",
			"data: [DONE]",
		}
		for _, l := range lines {
			_, _ = w.Write([]byte(l + "\n"))
		}
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "key")
	var chunks []*StreamChunk
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk *StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
}
