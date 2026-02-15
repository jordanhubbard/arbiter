package messaging

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jordanhubbard/loom/internal/temporal/eventbus"
	"github.com/jordanhubbard/loom/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgentRegistry implements AgentRegistry for testing
type mockAgentRegistry struct {
	agents map[string]string // role -> agentID
	err    error
}

func (m *mockAgentRegistry) FindAgentByRole(ctx context.Context, role string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	agentID, ok := m.agents[role]
	if !ok {
		return "", fmt.Errorf("no agent found for role: %s", role)
	}
	return agentID, nil
}

func (m *mockAgentRegistry) ListAgents(ctx context.Context) ([]AgentInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	infos := make([]AgentInfo, 0, len(m.agents))
	for role, id := range m.agents {
		infos = append(infos, AgentInfo{
			AgentID:     id,
			PersonaType: role,
			Status:      "active",
		})
	}
	return infos, nil
}

func setupAdapterTestBus(t *testing.T) *AgentMessageBus {
	t.Helper()
	cfg := &config.TemporalConfig{
		EventBufferSize: 100,
	}
	eb := eventbus.NewEventBus(nil, cfg)
	return NewAgentMessageBus(eb)
}

// ---------------------------------------------------------------------------
// NewActionMessageSender tests
// ---------------------------------------------------------------------------

func TestNewActionMessageSender(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	registry := &mockAgentRegistry{
		agents: map[string]string{"pm": "agent-pm"},
	}

	sender := NewActionMessageSender(bus, registry)
	assert.NotNil(t, sender)
	assert.Equal(t, bus, sender.bus)
	assert.Equal(t, registry, sender.agentRegistry)
}

func TestNewActionMessageSender_NilRegistry(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	sender := NewActionMessageSender(bus, nil)
	assert.NotNil(t, sender)
	assert.Nil(t, sender.agentRegistry)
}

// ---------------------------------------------------------------------------
// SendMessage tests
// ---------------------------------------------------------------------------

func TestSendMessage_Question(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	registry := &mockAgentRegistry{agents: map[string]string{}}
	sender := NewActionMessageSender(bus, registry)

	payload := map[string]interface{}{"key": "value"}
	msgID, err := sender.SendMessage(
		context.Background(),
		"agent-1", "agent-2",
		"question",
		"Need help",
		"Can you review this?",
		payload,
	)

	require.NoError(t, err)
	assert.NotEmpty(t, msgID)

	// Verify message was stored in history
	history := bus.GetHistory("agent-1", 10)
	assert.Len(t, history, 1)
	assert.Equal(t, MessageTypeDirect, history[0].Type)
	assert.Equal(t, PriorityNormal, history[0].Priority)
	assert.True(t, history[0].RequiresResponse)
	assert.Equal(t, "Need help", history[0].Subject)
	assert.Equal(t, "Can you review this?", history[0].Body)
}

func TestSendMessage_Delegation(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	registry := &mockAgentRegistry{agents: map[string]string{}}
	sender := NewActionMessageSender(bus, registry)

	msgID, err := sender.SendMessage(
		context.Background(),
		"agent-pm", "agent-dev",
		"delegation",
		"Build feature",
		"Please implement the login page",
		nil,
	)

	require.NoError(t, err)
	assert.NotEmpty(t, msgID)

	// Verify message type and priority
	history := bus.GetHistory("agent-pm", 10)
	assert.Len(t, history, 1)
	assert.Equal(t, MessageTypeRequest, history[0].Type)
	assert.Equal(t, PriorityHigh, history[0].Priority)
	assert.True(t, history[0].RequiresResponse)
}

func TestSendMessage_Notification(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	registry := &mockAgentRegistry{agents: map[string]string{}}
	sender := NewActionMessageSender(bus, registry)

	msgID, err := sender.SendMessage(
		context.Background(),
		"agent-1", "agent-2",
		"notification",
		"Status Update",
		"Task completed",
		nil,
	)

	require.NoError(t, err)
	assert.NotEmpty(t, msgID)

	history := bus.GetHistory("agent-1", 10)
	assert.Len(t, history, 1)
	assert.Equal(t, MessageTypeNotification, history[0].Type)
	assert.Equal(t, PriorityNormal, history[0].Priority)
	assert.False(t, history[0].RequiresResponse)
}

func TestSendMessage_UnsupportedType(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	registry := &mockAgentRegistry{agents: map[string]string{}}
	sender := NewActionMessageSender(bus, registry)

	_, err := sender.SendMessage(
		context.Background(),
		"agent-1", "agent-2",
		"unknown_type",
		"Test",
		"Body",
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported message type: unknown_type")
}

func TestSendMessage_PayloadCopiedToContext(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	registry := &mockAgentRegistry{agents: map[string]string{}}
	sender := NewActionMessageSender(bus, registry)

	payload := map[string]interface{}{
		"project_id": "proj-123",
		"bead_id":    "bead-456",
	}

	_, err := sender.SendMessage(
		context.Background(),
		"agent-1", "agent-2",
		"question",
		"Subject",
		"Body",
		payload,
	)

	require.NoError(t, err)

	history := bus.GetHistory("agent-1", 10)
	require.Len(t, history, 1)
	// Verify payload is set on message
	assert.Equal(t, "proj-123", history[0].Payload["project_id"])
	assert.Equal(t, "bead-456", history[0].Payload["bead_id"])
	// Verify context is also populated from payload
	assert.Equal(t, "proj-123", history[0].Context["project_id"])
	assert.Equal(t, "bead-456", history[0].Context["bead_id"])
}

func TestSendMessage_NilPayload(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	registry := &mockAgentRegistry{agents: map[string]string{}}
	sender := NewActionMessageSender(bus, registry)

	msgID, err := sender.SendMessage(
		context.Background(),
		"agent-1", "agent-2",
		"notification",
		"Subject",
		"Body",
		nil,
	)

	require.NoError(t, err)
	assert.NotEmpty(t, msgID)
}

// ---------------------------------------------------------------------------
// FindAgentByRole tests
// ---------------------------------------------------------------------------

func TestFindAgentByRole_Success(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	registry := &mockAgentRegistry{
		agents: map[string]string{
			"pm":        "agent-pm-1",
			"developer": "agent-dev-1",
		},
	}
	sender := NewActionMessageSender(bus, registry)

	agentID, err := sender.FindAgentByRole(context.Background(), "pm")
	require.NoError(t, err)
	assert.Equal(t, "agent-pm-1", agentID)
}

func TestFindAgentByRole_NotFound(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	registry := &mockAgentRegistry{
		agents: map[string]string{},
	}
	sender := NewActionMessageSender(bus, registry)

	_, err := sender.FindAgentByRole(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no agent found for role")
}

func TestFindAgentByRole_NilRegistry(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	sender := NewActionMessageSender(bus, nil)

	_, err := sender.FindAgentByRole(context.Background(), "pm")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent registry not configured")
}

func TestFindAgentByRole_RegistryError(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	registry := &mockAgentRegistry{
		agents: map[string]string{"pm": "agent-pm"},
		err:    fmt.Errorf("registry unavailable"),
	}
	sender := NewActionMessageSender(bus, registry)

	_, err := sender.FindAgentByRole(context.Background(), "pm")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry unavailable")
}

// ---------------------------------------------------------------------------
// AgentInfo struct tests
// ---------------------------------------------------------------------------

func TestAgentInfo_Fields(t *testing.T) {
	info := AgentInfo{
		AgentID:      "agent-123",
		PersonaType:  "developer",
		Capabilities: []string{"code", "test"},
		Status:       "active",
	}

	assert.Equal(t, "agent-123", info.AgentID)
	assert.Equal(t, "developer", info.PersonaType)
	assert.Equal(t, []string{"code", "test"}, info.Capabilities)
	assert.Equal(t, "active", info.Status)
}

// ---------------------------------------------------------------------------
// mockAgentRegistry ListAgents tests
// ---------------------------------------------------------------------------

func TestMockAgentRegistry_ListAgents(t *testing.T) {
	registry := &mockAgentRegistry{
		agents: map[string]string{
			"pm":  "agent-pm",
			"dev": "agent-dev",
		},
	}

	agents, err := registry.ListAgents(context.Background())
	require.NoError(t, err)
	assert.Len(t, agents, 2)
}

func TestMockAgentRegistry_ListAgents_Error(t *testing.T) {
	registry := &mockAgentRegistry{
		err: fmt.Errorf("service error"),
	}

	_, err := registry.ListAgents(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service error")
}

// ---------------------------------------------------------------------------
// SendMessage edge cases
// ---------------------------------------------------------------------------

func TestSendMessage_EmptySubjectAndBody(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	sender := NewActionMessageSender(bus, nil)

	msgID, err := sender.SendMessage(
		context.Background(),
		"agent-1", "agent-2",
		"notification",
		"",
		"",
		nil,
	)

	require.NoError(t, err)
	assert.NotEmpty(t, msgID)
}

func TestSendMessage_RequiresResponseForQuestion(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	sender := NewActionMessageSender(bus, nil)

	_, err := sender.SendMessage(
		context.Background(),
		"agent-1", "agent-2",
		"question",
		"Q", "Body", nil,
	)
	require.NoError(t, err)

	history := bus.GetHistory("agent-1", 10)
	require.Len(t, history, 1)
	assert.True(t, history[0].RequiresResponse, "question messages should require response")
}

func TestSendMessage_RequiresResponseForDelegation(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	sender := NewActionMessageSender(bus, nil)

	_, err := sender.SendMessage(
		context.Background(),
		"agent-1", "agent-2",
		"delegation",
		"D", "Body", nil,
	)
	require.NoError(t, err)

	history := bus.GetHistory("agent-1", 10)
	require.Len(t, history, 1)
	assert.True(t, history[0].RequiresResponse, "delegation messages should require response")
}

func TestSendMessage_DoesNotRequireResponseForNotification(t *testing.T) {
	bus := setupAdapterTestBus(t)
	defer bus.Close()

	sender := NewActionMessageSender(bus, nil)

	_, err := sender.SendMessage(
		context.Background(),
		"agent-1", "agent-2",
		"notification",
		"N", "Body", nil,
	)
	require.NoError(t, err)

	history := bus.GetHistory("agent-1", 10)
	require.Len(t, history, 1)
	assert.False(t, history[0].RequiresResponse, "notification messages should not require response")
}

// ---------------------------------------------------------------------------
// Multiple message types validation
// ---------------------------------------------------------------------------

func TestSendMessage_AllValidTypes(t *testing.T) {
	validTypes := []struct {
		msgType      string
		expectedType MessageType
		expectedPri  Priority
	}{
		{"question", MessageTypeDirect, PriorityNormal},
		{"delegation", MessageTypeRequest, PriorityHigh},
		{"notification", MessageTypeNotification, PriorityNormal},
	}

	for _, tt := range validTypes {
		t.Run(tt.msgType, func(t *testing.T) {
			bus := setupAdapterTestBus(t)
			defer bus.Close()

			sender := NewActionMessageSender(bus, nil)

			_, err := sender.SendMessage(
				context.Background(),
				"from", "to",
				tt.msgType,
				"subject", "body", nil,
			)
			require.NoError(t, err)

			history := bus.GetHistory("from", 10)
			require.Len(t, history, 1)
			assert.Equal(t, tt.expectedType, history[0].Type)
			assert.Equal(t, tt.expectedPri, history[0].Priority)
		})
	}
}

func TestSendMessage_InvalidTypes(t *testing.T) {
	invalidTypes := []string{
		"",
		"broadcast",
		"consensus",
		"response",
		"direct",
		"request",
		"QUESTION",
	}

	for _, msgType := range invalidTypes {
		t.Run("type_"+msgType, func(t *testing.T) {
			bus := setupAdapterTestBus(t)
			defer bus.Close()

			sender := NewActionMessageSender(bus, nil)

			_, err := sender.SendMessage(
				context.Background(),
				"from", "to",
				msgType,
				"subject", "body", nil,
			)
			require.Error(t, err)
			assert.True(t, strings.Contains(err.Error(), "unsupported message type"))
		})
	}
}
