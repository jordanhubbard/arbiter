package actions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMessageSender struct {
	sentMessages []mockSentMessage
	findError    error
	sendError    error
}

type mockSentMessage struct {
	fromAgentID string
	toAgentID   string
	messageType string
	subject     string
	body        string
	payload     map[string]interface{}
}

func (m *mockMessageSender) SendMessage(ctx context.Context, fromAgentID, toAgentID, messageType, subject, body string, payload map[string]interface{}) (string, error) {
	if m.sendError != nil {
		return "", m.sendError
	}

	m.sentMessages = append(m.sentMessages, mockSentMessage{
		fromAgentID: fromAgentID,
		toAgentID:   toAgentID,
		messageType: messageType,
		subject:     subject,
		body:        body,
		payload:     payload,
	})

	return "msg-test-123", nil
}

func (m *mockMessageSender) FindAgentByRole(ctx context.Context, role string) (string, error) {
	if m.findError != nil {
		return "", m.findError
	}

	// Map some test roles
	roleToID := map[string]string{
		"qa-engineer":   "agent-qa-1",
		"code-reviewer": "agent-reviewer-1",
		"engineer":      "agent-eng-1",
	}

	if agentID, ok := roleToID[role]; ok {
		return agentID, nil
	}

	return "", assert.AnError
}

func TestHandleSendAgentMessage_DirectByID(t *testing.T) {
	mockBus := &mockMessageSender{}
	router := &Router{MessageBus: mockBus}

	action := Action{
		Type:           ActionSendAgentMessage,
		ToAgentID:      "agent-qa-1",
		MessageType:    "question",
		MessageSubject: "Test coverage",
		MessageBody:    "Can you check the test coverage for auth module?",
		MessagePayload: map[string]interface{}{
			"module": "auth",
		},
	}

	actx := ActionContext{
		AgentID:   "agent-eng-1",
		BeadID:    "bead-abc-123",
		ProjectID: "project-1",
	}

	result := router.handleSendAgentMessage(context.Background(), action, actx)

	assert.Equal(t, ActionSendAgentMessage, result.ActionType)
	assert.Equal(t, "executed", result.Status)
	assert.Contains(t, result.Message, "agent-qa-1")
	assert.Equal(t, "msg-test-123", result.Metadata["message_id"])

	require.Len(t, mockBus.sentMessages, 1)
	msg := mockBus.sentMessages[0]
	assert.Equal(t, "agent-eng-1", msg.fromAgentID)
	assert.Equal(t, "agent-qa-1", msg.toAgentID)
	assert.Equal(t, "question", msg.messageType)
	assert.Equal(t, "Test coverage", msg.subject)
}

func TestHandleSendAgentMessage_ByRole(t *testing.T) {
	mockBus := &mockMessageSender{}
	router := &Router{MessageBus: mockBus}

	action := Action{
		Type:           ActionSendAgentMessage,
		ToAgentRole:    "code-reviewer",
		MessageType:    "delegation",
		MessageSubject: "Review PR",
		MessageBody:    "Please review PR #456",
	}

	actx := ActionContext{
		AgentID:   "agent-eng-1",
		BeadID:    "bead-abc-123",
		ProjectID: "project-1",
	}

	result := router.handleSendAgentMessage(context.Background(), action, actx)

	assert.Equal(t, "executed", result.Status)
	assert.Equal(t, "agent-reviewer-1", result.Metadata["to_agent_id"])

	require.Len(t, mockBus.sentMessages, 1)
	msg := mockBus.sentMessages[0]
	assert.Equal(t, "agent-reviewer-1", msg.toAgentID)
	assert.Equal(t, "delegation", msg.messageType)
}

func TestHandleSendAgentMessage_Notification(t *testing.T) {
	mockBus := &mockMessageSender{}
	router := &Router{MessageBus: mockBus}

	action := Action{
		Type:           ActionSendAgentMessage,
		ToAgentID:      "agent-pm-1",
		MessageType:    "notification",
		MessageSubject: "Build completed",
		MessageBody:    "Build finished successfully",
		MessagePayload: map[string]interface{}{
			"build_id": "build-789",
			"status":   "success",
		},
	}

	actx := ActionContext{
		AgentID:   "agent-builder-1",
		BeadID:    "bead-build-123",
		ProjectID: "project-1",
	}

	result := router.handleSendAgentMessage(context.Background(), action, actx)

	assert.Equal(t, "executed", result.Status)

	require.Len(t, mockBus.sentMessages, 1)
	msg := mockBus.sentMessages[0]
	assert.Equal(t, "notification", msg.messageType)
	assert.Equal(t, "build-789", msg.payload["build_id"])
}

func TestHandleSendAgentMessage_ValidationErrors(t *testing.T) {
	mockBus := &mockMessageSender{}
	router := &Router{MessageBus: mockBus}

	actx := ActionContext{AgentID: "agent-1"}

	tests := []struct {
		name    string
		action  Action
		wantErr string
	}{
		{
			name: "missing both to_agent_id and to_agent_role",
			action: Action{
				Type:        ActionSendAgentMessage,
				MessageType: "question",
			},
			wantErr: "either to_agent_id or to_agent_role is required",
		},
		{
			name: "missing message_type",
			action: Action{
				Type:      ActionSendAgentMessage,
				ToAgentID: "agent-qa-1",
			},
			wantErr: "message_type is required",
		},
		{
			name: "invalid message_type",
			action: Action{
				Type:        ActionSendAgentMessage,
				ToAgentID:   "agent-qa-1",
				MessageType: "invalid",
			},
			wantErr: "message_type must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.handleSendAgentMessage(context.Background(), tt.action, actx)

			assert.Equal(t, "error", result.Status)
			assert.Contains(t, result.Message, tt.wantErr)
		})
	}
}

func TestHandleSendAgentMessage_MessageBusNotConfigured(t *testing.T) {
	router := &Router{MessageBus: nil}

	action := Action{
		Type:        ActionSendAgentMessage,
		ToAgentID:   "agent-qa-1",
		MessageType: "question",
	}

	actx := ActionContext{AgentID: "agent-1"}

	result := router.handleSendAgentMessage(context.Background(), action, actx)

	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Message, "message bus not configured")
}

func TestHandleSendAgentMessage_RoleNotFound(t *testing.T) {
	mockBus := &mockMessageSender{
		findError: assert.AnError,
	}
	router := &Router{MessageBus: mockBus}

	action := Action{
		Type:        ActionSendAgentMessage,
		ToAgentRole: "nonexistent-role",
		MessageType: "question",
	}

	actx := ActionContext{AgentID: "agent-1"}

	result := router.handleSendAgentMessage(context.Background(), action, actx)

	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Message, "failed to find agent")
}

func TestHandleSendAgentMessage_SendError(t *testing.T) {
	mockBus := &mockMessageSender{
		sendError: assert.AnError,
	}
	router := &Router{MessageBus: mockBus}

	action := Action{
		Type:        ActionSendAgentMessage,
		ToAgentID:   "agent-qa-1",
		MessageType: "question",
	}

	actx := ActionContext{AgentID: "agent-1"}

	result := router.handleSendAgentMessage(context.Background(), action, actx)

	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Message, "failed to send message")
}
