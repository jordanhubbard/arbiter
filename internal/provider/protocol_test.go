package provider

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:  "Pure JSON",
			input: `{"id": "test", "value": 123}`,
			want:  map[string]interface{}{"id": "test", "value": float64(123)},
		},
		{
			name:  "JSON with leading text",
			input: `Here is the response: {"id": "test", "value": 123}`,
			want:  map[string]interface{}{"id": "test", "value": float64(123)},
		},
		{
			name:  "JSON with trailing text",
			input: `{"id": "test", "value": 123} - end of response`,
			want:  map[string]interface{}{"id": "test", "value": float64(123)},
		},
		{
			name:  "JSON with leading and trailing text",
			input: `Response begins: {"id": "test", "value": 123} - done`,
			want:  map[string]interface{}{"id": "test", "value": float64(123)},
		},
		{
			name:  "JSON with nested objects",
			input: `{"id": "test", "nested": {"key": "value"}, "array": [1, 2, 3]}`,
			want:  map[string]interface{}{"id": "test", "nested": map[string]interface{}{"key": "value"}, "array": []interface{}{float64(1), float64(2), float64(3)}},
		},
		{
			name:  "JSON with string containing braces",
			input: `{"message": "Use {brackets} in strings", "id": 1}`,
			want:  map[string]interface{}{"message": "Use {brackets} in strings", "id": float64(1)},
		},
		{
			name:  "JSON with escaped quotes",
			input: `{"message": "He said \"hello\"", "id": 1}`,
			want:  map[string]interface{}{"message": `He said "hello"`, "id": float64(1)},
		},
		{
			name:    "Invalid JSON",
			input:   `This is not JSON at all`,
			wantErr: true,
		},
		{
			name:    "Incomplete JSON",
			input:   `{"id": "test", "value":`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]interface{}
			err := unmarshalJSON([]byte(tt.input), &got)

			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				wantJSON, _ := json.Marshal(tt.want)
				gotJSON, _ := json.Marshal(got)
				if string(wantJSON) != string(gotJSON) {
					t.Errorf("unmarshalJSON() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Pure JSON object",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "JSON with leading text",
			input: `Here is the data: {"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "JSON with trailing text",
			input: `{"key": "value"} and some more text`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "Nested JSON",
			input: `{"outer": {"inner": {"deep": "value"}}}`,
			want:  `{"outer": {"inner": {"deep": "value"}}}`,
		},
		{
			name:  "JSON array",
			input: `[1, 2, 3, 4]`,
			want:  `[1, 2, 3, 4]`,
		},
		{
			name:  "Complex with whitespace",
			input: "  \n  {\"key\": \"value\"}  \n  ",
			want:  `{"key": "value"}`,
		},
		{
			name:  "No JSON",
			input: `This is just plain text`,
			want:  ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON([]byte(tt.input))
			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("extractJSON() = %v, want %v", gotStr, tt.want)
			}
		})
	}
}

func TestChatCompletionResponseParsing(t *testing.T) {
	// Test with a realistic response that might have extra text
	input := `Additional text before JSON
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "gpt-3.5-turbo-0613",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello there! How can I help you?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
Additional text after JSON`

	var resp ChatCompletionResponse
	err := unmarshalJSON([]byte(input), &resp)
	if err != nil {
		t.Fatalf("Failed to unmarshal chat completion response: %v", err)
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("Expected ID 'chatcmpl-123', got '%s'", resp.ID)
	}

	if len(resp.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Hello there! How can I help you?" {
		t.Errorf("Unexpected message content: %s", resp.Choices[0].Message.Content)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("Expected 30 total tokens, got %d", resp.Usage.TotalTokens)
	}
}
