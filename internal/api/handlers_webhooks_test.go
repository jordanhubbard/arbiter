package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jordanhubbard/loom/pkg/config"
)

func TestGitHubWebhook_PROpened(t *testing.T) {
	// Create test server
	cfg := &config.Config{
		Security: config.SecurityConfig{
			WebhookSecret: "test-secret",
		},
	}
	server := NewServer(nil, nil, nil, cfg)

	// Create PR opened payload
	payload := map[string]interface{}{
		"action": "opened",
		"number": 123,
		"pull_request": map[string]interface{}{
			"number": 123,
			"title":  "Test PR",
			"state":  "open",
			"draft":  false,
			"user": map[string]interface{}{
				"login": "testuser",
			},
			"head": map[string]interface{}{
				"ref": "feature-branch",
				"sha": "abc123",
			},
			"base": map[string]interface{}{
				"ref": "main",
				"sha": "def456",
			},
			"html_url": "https://github.com/owner/repo/pull/123",
		},
		"repository": map[string]interface{}{
			"full_name": "owner/repo",
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	// Create request with signature
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", generateSignature(payloadBytes, "test-secret"))

	// Record response
	w := httptest.NewRecorder()

	// Handle request
	server.handleGitHubWebhook(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "received" {
		t.Errorf("Expected status 'received', got %v", response["status"])
	}
}

func TestGitHubWebhook_PRSynchronized(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			WebhookSecret: "test-secret",
		},
	}
	server := NewServer(nil, nil, nil, cfg)

	payload := map[string]interface{}{
		"action": "synchronize",
		"number": 123,
		"pull_request": map[string]interface{}{
			"number": 123,
			"head": map[string]interface{}{
				"sha": "new-sha",
			},
			"html_url": "https://github.com/owner/repo/pull/123",
		},
		"repository": map[string]interface{}{
			"full_name": "owner/repo",
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", generateSignature(payloadBytes, "test-secret"))

	w := httptest.NewRecorder()
	server.handleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGitHubWebhook_InvalidSignature(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			WebhookSecret: "test-secret",
		},
	}
	server := NewServer(nil, nil, nil, cfg)

	payload := map[string]interface{}{
		"action": "opened",
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")

	w := httptest.NewRecorder()
	server.handleGitHubWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestGitHubWebhook_MissingEventHeader(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			WebhookSecret: "test-secret",
		},
	}
	server := NewServer(nil, nil, nil, cfg)

	payload := map[string]interface{}{"action": "opened"}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", generateSignature(payloadBytes, "test-secret"))
	// Missing X-GitHub-Event header

	w := httptest.NewRecorder()
	server.handleGitHubWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGitHubWebhook_IssueOpened(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			WebhookSecret: "test-secret",
		},
	}
	server := NewServer(nil, nil, nil, cfg)

	payload := map[string]interface{}{
		"action": "opened",
		"issue": map[string]interface{}{
			"number": 456,
			"title":  "Bug report",
			"state":  "open",
			"user": map[string]interface{}{
				"login": "reporter",
			},
			"html_url": "https://github.com/owner/repo/issues/456",
		},
		"repository": map[string]interface{}{
			"full_name": "owner/repo",
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-Hub-Signature-256", generateSignature(payloadBytes, "test-secret"))

	w := httptest.NewRecorder()
	server.handleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGitHubWebhook_ReleasePublished(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			WebhookSecret: "test-secret",
		},
	}
	server := NewServer(nil, nil, nil, cfg)

	payload := map[string]interface{}{
		"action": "published",
		"release": map[string]interface{}{
			"tag_name": "v1.0.0",
			"name":     "Release 1.0.0",
			"author": map[string]interface{}{
				"login": "releaser",
			},
			"html_url":   "https://github.com/owner/repo/releases/tag/v1.0.0",
			"draft":      false,
			"prerelease": false,
		},
		"repository": map[string]interface{}{
			"full_name": "owner/repo",
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "release")
	req.Header.Set("X-Hub-Signature-256", generateSignature(payloadBytes, "test-secret"))

	w := httptest.NewRecorder()
	server.handleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestWebhookStatus(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			WebhookSecret: "configured",
		},
	}
	server := NewServer(nil, nil, nil, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/status", nil)
	w := httptest.NewRecorder()

	server.handleWebhookStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["github_webhook_enabled"] != true {
		t.Errorf("Expected github_webhook_enabled to be true")
	}

	if response["webhook_secret_configured"] != true {
		t.Errorf("Expected webhook_secret_configured to be true")
	}
}

func TestProcessGitHubEvent_UnsupportedEvent(t *testing.T) {
	server := NewServer(nil, nil, nil, nil)

	payload := &GitHubWebhookPayload{
		Action: "some_action",
	}

	event := server.processGitHubEvent("unsupported_event", payload)

	if event != nil {
		t.Errorf("Expected nil for unsupported event, got %v", event)
	}
}

func TestProcessGitHubEvent_PRReadyForReview(t *testing.T) {
	server := NewServer(nil, nil, nil, nil)

	payload := &GitHubWebhookPayload{
		Action: "ready_for_review",
		PullRequest: &GitHubPullRequest{
			Number: 123,
			URL:    "https://github.com/owner/repo/pull/123",
		},
		Repository: &GitHubRepository{
			FullName: "owner/repo",
		},
	}

	event := server.processGitHubEvent("pull_request", payload)

	if event == nil {
		t.Fatal("Expected event, got nil")
	}

	if event.Type != "github_pr_ready" {
		t.Errorf("Expected type 'github_pr_ready', got %s", event.Type)
	}

	if event.Repository != "owner/repo" {
		t.Errorf("Expected repository 'owner/repo', got %s", event.Repository)
	}

	// Check trigger flag
	if trigger, ok := event.Data["trigger_code_review"].(bool); !ok || !trigger {
		t.Errorf("Expected trigger_code_review to be true")
	}
}

func TestVerifyGitHubSignature(t *testing.T) {
	payload := []byte(`{"test":"data"}`)
	secret := "my-secret"

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Test valid signature
	if !verifyGitHubSignature(payload, validSig, secret) {
		t.Error("Valid signature should pass verification")
	}

	// Test invalid signature
	if verifyGitHubSignature(payload, "sha256=invalid", secret) {
		t.Error("Invalid signature should fail verification")
	}

	// Test empty signature
	if verifyGitHubSignature(payload, "", secret) {
		t.Error("Empty signature should fail verification")
	}

	// Test empty secret
	if verifyGitHubSignature(payload, validSig, "") {
		t.Error("Empty secret should fail verification")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a very long string that needs truncation", 20, "this is a very lo..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

// Helper function to generate HMAC signature
func generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Integration test with full server setup
func TestWebhookIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create minimal Loom instance for testing
	// This would require a test database and full setup
	// For now, we test with nil loom to verify webhook parsing

	cfg := &config.Config{
		Security: config.SecurityConfig{
			WebhookSecret: "test-secret-123",
		},
	}

	// Create server without loom (tests webhook parsing only)
	server := NewServer(nil, nil, nil, cfg)

	// Test PR webhook end-to-end
	payload := map[string]interface{}{
		"action": "opened",
		"number": 999,
		"pull_request": map[string]interface{}{
			"number": 999,
			"title":  "Integration test PR",
			"state":  "open",
			"draft":  false,
			"user":   map[string]interface{}{"login": "testbot"},
			"head":   map[string]interface{}{"ref": "test-branch", "sha": "test123"},
			"base":   map[string]interface{}{"ref": "main", "sha": "main456"},
			"html_url": "https://github.com/test/repo/pull/999",
		},
		"repository": map[string]interface{}{
			"full_name": "test/repo",
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", generateSignature(payloadBytes, "test-secret-123"))

	w := httptest.NewRecorder()
	server.handleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Integration test failed with status %d: %s", w.Code, w.Body.String())
	}
}
