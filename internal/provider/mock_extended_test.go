package provider

import (
	"context"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// MockProvider: CreateChatCompletion
// ---------------------------------------------------------------------------

func TestMockProvider_CreateChatCompletion_EchoesLastMessage(t *testing.T) {
	p := NewMockProvider()

	resp, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model: "mock-model",
		Messages: []ChatMessage{
			{Role: "system", Content: "you are helpful"},
			{Role: "user", Content: "what is 2+2"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp.Choices[0].Message.Content, "what is 2+2") {
		t.Errorf("expected echo of last user message, got %q", resp.Choices[0].Message.Content)
	}
	if !strings.HasPrefix(resp.Choices[0].Message.Content, "[mock] ") {
		t.Errorf("expected [mock] prefix, got %q", resp.Choices[0].Message.Content)
	}
	if resp.Choices[0].Message.Role != "assistant" {
		t.Errorf("role = %q, want assistant", resp.Choices[0].Message.Role)
	}
	if resp.Choices[0].Finish != "stop" {
		t.Errorf("finish = %q, want stop", resp.Choices[0].Finish)
	}
}

func TestMockProvider_CreateChatCompletion_EmptyMessages(t *testing.T) {
	p := NewMockProvider()

	resp, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "mock-model",
		Messages: []ChatMessage{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should use default "mock response"
	if !strings.Contains(resp.Choices[0].Message.Content, "[mock] mock response") {
		t.Errorf("expected default mock response, got %q", resp.Choices[0].Message.Content)
	}
}

func TestMockProvider_CreateChatCompletion_EmptyContent(t *testing.T) {
	p := NewMockProvider()

	resp, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model: "mock-model",
		Messages: []ChatMessage{
			{Role: "user", Content: ""},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty content falls back to "mock response"
	if !strings.Contains(resp.Choices[0].Message.Content, "[mock] mock response") {
		t.Errorf("expected default, got %q", resp.Choices[0].Message.Content)
	}
}

func TestMockProvider_CreateChatCompletion_PreservesModel(t *testing.T) {
	p := NewMockProvider()

	resp, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model: "custom-model-name",
		Messages: []ChatMessage{
			{Role: "user", Content: "test"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Model != "custom-model-name" {
		t.Errorf("model = %q, want %q", resp.Model, "custom-model-name")
	}
}

func TestMockProvider_CreateChatCompletion_UsageCounts(t *testing.T) {
	p := NewMockProvider()

	resp, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model: "m",
		Messages: []ChatMessage{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Usage.PromptTokens == 0 {
		t.Error("PromptTokens should be non-zero")
	}
	if resp.Usage.CompletionTokens == 0 {
		t.Error("CompletionTokens should be non-zero")
	}
	if resp.Usage.TotalTokens != resp.Usage.PromptTokens+resp.Usage.CompletionTokens {
		t.Error("TotalTokens should be sum of prompt + completion")
	}
}

func TestMockProvider_CreateChatCompletion_ResponseFields(t *testing.T) {
	p := NewMockProvider()

	resp, err := p.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model: "m",
		Messages: []ChatMessage{
			{Role: "user", Content: "test"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "mock-completion" {
		t.Errorf("ID = %q, want mock-completion", resp.ID)
	}
	if resp.Object != "chat.completion" {
		t.Errorf("Object = %q, want chat.completion", resp.Object)
	}
	if resp.Created == 0 {
		t.Error("Created should be non-zero")
	}
}

// ---------------------------------------------------------------------------
// MockProvider: GetModels
// ---------------------------------------------------------------------------

func TestMockProvider_GetModels(t *testing.T) {
	p := NewMockProvider()

	models, err := p.GetModels(context.Background())
	if err != nil {
		t.Fatalf("GetModels: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].ID != "mock-model" {
		t.Errorf("model ID = %q, want mock-model", models[0].ID)
	}
	if models[0].Object != "model" {
		t.Errorf("model Object = %q, want model", models[0].Object)
	}
	if models[0].OwnedBy != "mock" {
		t.Errorf("model OwnedBy = %q, want mock", models[0].OwnedBy)
	}
}

// ---------------------------------------------------------------------------
// MockProvider: CreateChatCompletionStream cancellation
// ---------------------------------------------------------------------------

func TestMockProvider_Stream_Cancellation(t *testing.T) {
	p := NewMockProvider()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := p.CreateChatCompletionStream(ctx, &ChatCompletionRequest{
		Model: "m",
		Messages: []ChatMessage{
			{Role: "user", Content: "long message that should be chunked"},
		},
	}, func(chunk *StreamChunk) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ---------------------------------------------------------------------------
// MockProvider: CreateChatCompletionStream default content
// ---------------------------------------------------------------------------

func TestMockProvider_Stream_EmptyMessages(t *testing.T) {
	p := NewMockProvider()

	var content strings.Builder
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{},
	}, func(chunk *StreamChunk) error {
		if len(chunk.Choices) > 0 {
			content.WriteString(chunk.Choices[0].Delta.Content)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content.String(), "mock response") {
		t.Errorf("expected mock response default, got %q", content.String())
	}
}

func TestMockProvider_Stream_EmptyContent(t *testing.T) {
	p := NewMockProvider()

	var content strings.Builder
	err := p.CreateChatCompletionStream(context.Background(), &ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: ""}},
	}, func(chunk *StreamChunk) error {
		if len(chunk.Choices) > 0 {
			content.WriteString(chunk.Choices[0].Delta.Content)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content.String(), "mock response") {
		t.Errorf("expected default, got %q", content.String())
	}
}
