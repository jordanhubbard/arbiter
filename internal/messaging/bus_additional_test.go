package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/temporal/eventbus"
	"github.com/jordanhubbard/loom/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestBus2(t *testing.T) *AgentMessageBus {
	t.Helper()
	cfg := &config.TemporalConfig{
		EventBufferSize: 100,
	}
	eb := eventbus.NewEventBus(nil, cfg)
	return NewAgentMessageBus(eb)
}

// ---------------------------------------------------------------------------
// Send - defaults and pre-set fields
// ---------------------------------------------------------------------------

func TestSend_PreSetMessageID(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		MessageID:   "custom-id-123",
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)

	// MessageID should NOT be overwritten
	assert.Equal(t, "custom-id-123", msg.MessageID)
}

func TestSend_PreSetTimestamp(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	fixedTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Timestamp:   fixedTime,
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)

	// Timestamp should NOT be overwritten
	assert.Equal(t, fixedTime, msg.Timestamp)
}

func TestSend_PreSetPriority(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Priority:    PriorityUrgent,
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)

	// Priority should NOT be overwritten
	assert.Equal(t, PriorityUrgent, msg.Priority)
}

func TestSend_DefaultPriority(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		// Priority not set
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)

	assert.Equal(t, PriorityNormal, msg.Priority)
}

func TestSend_DefaultTimestamp(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	before := time.Now()
	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		// Timestamp not set (zero value)
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)
	after := time.Now()

	assert.False(t, msg.Timestamp.IsZero())
	assert.False(t, msg.Timestamp.Before(before))
	assert.False(t, msg.Timestamp.After(after))
}

func TestSend_NotificationMessage(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeNotification,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Subject:     "Status update",
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, "sent", msg.Status)
	assert.NotEmpty(t, msg.MessageID)
}

func TestSend_ResponseMessage(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeResponse,
		FromAgentID: "agent-2",
		ToAgentID:   "agent-1",
		InReplyTo:   "original-msg-id",
		Body:        "Here is my response",
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, "sent", msg.Status)
}

func TestSend_ConsensusVote(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeConsensusVote,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-pm",
		InReplyTo:   "consensus-req-id",
		Payload: map[string]interface{}{
			"vote": "yes",
		},
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, "sent", msg.Status)
}

// ---------------------------------------------------------------------------
// SendAndWait - non-request type
// ---------------------------------------------------------------------------

func TestSendAndWait_NonRequestType(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
	}

	_, err := bus.SendAndWait(context.Background(), msg, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SendAndWait only works with request messages")
}

func TestSendAndWait_BroadcastType(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeBroadcast,
		FromAgentID: "agent-1",
	}

	_, err := bus.SendAndWait(context.Background(), msg, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SendAndWait only works with request messages")
}

func TestSendAndWait_NotificationType(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeNotification,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
	}

	_, err := bus.SendAndWait(context.Background(), msg, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SendAndWait only works with request messages")
}

func TestSendAndWait_ResponseType(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeResponse,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
	}

	_, err := bus.SendAndWait(context.Background(), msg, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SendAndWait only works with request messages")
}

// ---------------------------------------------------------------------------
// Subscribe - dedup
// ---------------------------------------------------------------------------

func TestSubscribe_DuplicateReturnsExisting(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect},
	}

	sub1 := bus.Subscribe("sub-dup", "agent-1", filter)
	sub2 := bus.Subscribe("sub-dup", "agent-1", filter)

	// Should return the same subscription
	assert.Equal(t, sub1, sub2)
	assert.Equal(t, sub1.ID, sub2.ID)
	assert.Equal(t, sub1.Channel, sub2.Channel)
}

func TestSubscribe_DifferentIDs(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect},
	}

	sub1 := bus.Subscribe("sub-a", "agent-1", filter)
	sub2 := bus.Subscribe("sub-b", "agent-2", filter)

	assert.NotEqual(t, sub1, sub2)
	assert.NotEqual(t, sub1.ID, sub2.ID)
}

func TestSubscribe_FieldsPopulated(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect, MessageTypeBroadcast},
		FromAgentIDs: []string{"agent-1"},
		ToAgentID:    "agent-2",
		MinPriority:  PriorityHigh,
	}

	sub := bus.Subscribe("sub-fields", "agent-2", filter)
	assert.Equal(t, "sub-fields", sub.ID)
	assert.Equal(t, "agent-2", sub.AgentID)
	assert.Equal(t, filter, sub.Filter)
	assert.NotNil(t, sub.Channel)
}

// ---------------------------------------------------------------------------
// Unsubscribe - edge cases
// ---------------------------------------------------------------------------

func TestUnsubscribe_NonExistent(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	// Should not panic
	bus.Unsubscribe("non-existent-sub")
}

func TestUnsubscribe_Twice(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect},
	}
	bus.Subscribe("sub-twice", "agent-1", filter)

	// First unsubscribe
	bus.Unsubscribe("sub-twice")
	// Second unsubscribe should not panic
	bus.Unsubscribe("sub-twice")
}

// ---------------------------------------------------------------------------
// GetHistory - edge cases
// ---------------------------------------------------------------------------

func TestGetHistory_NonExistentAgent(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	history := bus.GetHistory("nonexistent-agent", 10)
	assert.NotNil(t, history)
	assert.Len(t, history, 0)
}

func TestGetHistory_ZeroLimit(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	// Send some messages first
	for i := 0; i < 3; i++ {
		msg := &AgentMessage{
			Type:        MessageTypeDirect,
			FromAgentID: "agent-1",
			ToAgentID:   "agent-2",
			Subject:     "Test",
		}
		bus.Send(context.Background(), msg)
	}

	// Zero limit should return all messages
	history := bus.GetHistory("agent-1", 0)
	assert.Len(t, history, 3)
}

func TestGetHistory_NegativeLimit(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	for i := 0; i < 3; i++ {
		msg := &AgentMessage{
			Type:        MessageTypeDirect,
			FromAgentID: "agent-1",
			ToAgentID:   "agent-2",
			Subject:     "Test",
		}
		bus.Send(context.Background(), msg)
	}

	// Negative limit should return all messages
	history := bus.GetHistory("agent-1", -5)
	assert.Len(t, history, 3)
}

func TestGetHistory_LimitGreaterThanMessages(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Subject:     "Test",
	}
	bus.Send(context.Background(), msg)

	// Limit > messages should return all
	history := bus.GetHistory("agent-1", 100)
	assert.Len(t, history, 1)
}

func TestGetHistory_ExactLimit(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	for i := 0; i < 5; i++ {
		msg := &AgentMessage{
			Type:        MessageTypeDirect,
			FromAgentID: "agent-1",
			ToAgentID:   "agent-2",
			Subject:     "Test",
		}
		bus.Send(context.Background(), msg)
	}

	// Exact limit
	history := bus.GetHistory("agent-1", 5)
	assert.Len(t, history, 5)
}

// ---------------------------------------------------------------------------
// addToHistory - various scenarios
// ---------------------------------------------------------------------------

func TestAddToHistory_BroadcastNoReceiver(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeBroadcast,
		FromAgentID: "agent-1",
		Subject:     "Broadcast",
		// No ToAgentID
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)

	// Only sender should have history
	senderHistory := bus.GetHistory("agent-1", 10)
	assert.Len(t, senderHistory, 1)

	// No receiver in history since broadcast has no ToAgentID
	receiverHistory := bus.GetHistory("agent-2", 10)
	assert.Len(t, receiverHistory, 0)
}

func TestAddToHistory_ConsensusToMultipleRecipients(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeConsensusRequest,
		FromAgentID: "agent-pm",
		ToAgentIDs:  []string{"agent-1", "agent-2", "agent-3"},
		Subject:     "Vote needed",
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)

	// Sender should have history
	pmHistory := bus.GetHistory("agent-pm", 10)
	assert.Len(t, pmHistory, 1)

	// Each recipient should have history
	for _, agentID := range []string{"agent-1", "agent-2", "agent-3"} {
		history := bus.GetHistory(agentID, 10)
		assert.Len(t, history, 1, "Expected 1 message in history for %s", agentID)
	}
}

func TestAddToHistory_DirectBothSenderAndReceiver(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "sender",
		ToAgentID:   "receiver",
		Body:        "hello",
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)

	senderHistory := bus.GetHistory("sender", 10)
	assert.Len(t, senderHistory, 1)

	receiverHistory := bus.GetHistory("receiver", 10)
	assert.Len(t, receiverHistory, 1)

	// Both should reference same message ID
	assert.Equal(t, senderHistory[0].MessageID, receiverHistory[0].MessageID)
}

// ---------------------------------------------------------------------------
// matchesFilter - additional scenarios
// ---------------------------------------------------------------------------

func TestMatchesFilter_NoFilterMatchesAll(t *testing.T) {
	bus := setupTestBus2(t)

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Priority:    PriorityLow,
	}

	// Empty filter should match everything
	filter := MessageFilter{}
	assert.True(t, bus.matchesFilter(msg, filter, "agent-2"))
}

func TestMatchesFilter_FromAgentIDFilter(t *testing.T) {
	bus := setupTestBus2(t)

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Priority:    PriorityNormal,
	}

	// Matching from agent
	filter := MessageFilter{
		FromAgentIDs: []string{"agent-1"},
	}
	assert.True(t, bus.matchesFilter(msg, filter, "agent-2"))

	// Non-matching from agent
	filter2 := MessageFilter{
		FromAgentIDs: []string{"agent-3", "agent-4"},
	}
	assert.False(t, bus.matchesFilter(msg, filter2, "agent-2"))
}

func TestMatchesFilter_MultipleFromAgents(t *testing.T) {
	bus := setupTestBus2(t)

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-2",
		ToAgentID:   "agent-3",
		Priority:    PriorityNormal,
	}

	filter := MessageFilter{
		FromAgentIDs: []string{"agent-1", "agent-2", "agent-3"},
	}
	assert.True(t, bus.matchesFilter(msg, filter, "agent-3"))
}

func TestMatchesFilter_ConsensusRecipientMatch(t *testing.T) {
	bus := setupTestBus2(t)

	msg := &AgentMessage{
		Type:        MessageTypeConsensusRequest,
		FromAgentID: "agent-pm",
		ToAgentIDs:  []string{"agent-1", "agent-2", "agent-3"},
		Priority:    PriorityNormal,
	}

	// agent-2 is in ToAgentIDs, should match with ToAgentID filter set
	filter := MessageFilter{
		ToAgentID: "agent-2",
	}
	assert.True(t, bus.matchesFilter(msg, filter, "agent-2"))
}

func TestMatchesFilter_ConsensusRecipientNotFound(t *testing.T) {
	bus := setupTestBus2(t)

	msg := &AgentMessage{
		Type:        MessageTypeConsensusRequest,
		FromAgentID: "agent-pm",
		ToAgentIDs:  []string{"agent-1", "agent-2"},
		Priority:    PriorityNormal,
	}

	// agent-5 is NOT in ToAgentIDs
	filter := MessageFilter{
		ToAgentID: "agent-5",
	}
	assert.False(t, bus.matchesFilter(msg, filter, "agent-5"))
}

func TestMatchesFilter_MessageTypeFilter(t *testing.T) {
	bus := setupTestBus2(t)

	msg := &AgentMessage{
		Type:        MessageTypeBroadcast,
		FromAgentID: "agent-1",
		Priority:    PriorityNormal,
	}

	// Matching type
	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeBroadcast},
	}
	assert.True(t, bus.matchesFilter(msg, filter, "agent-2"))

	// Non-matching type
	filter2 := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect, MessageTypeRequest},
	}
	assert.False(t, bus.matchesFilter(msg, filter2, "agent-2"))
}

func TestMatchesFilter_PriorityLevels(t *testing.T) {
	bus := setupTestBus2(t)

	tests := []struct {
		name        string
		msgPriority Priority
		minPriority Priority
		want        bool
	}{
		{"low >= low", PriorityLow, PriorityLow, true},
		{"normal >= low", PriorityNormal, PriorityLow, true},
		{"high >= low", PriorityHigh, PriorityLow, true},
		{"urgent >= low", PriorityUrgent, PriorityLow, true},
		{"low >= normal", PriorityLow, PriorityNormal, false},
		{"normal >= normal", PriorityNormal, PriorityNormal, true},
		{"high >= normal", PriorityHigh, PriorityNormal, true},
		{"urgent >= normal", PriorityUrgent, PriorityNormal, true},
		{"low >= high", PriorityLow, PriorityHigh, false},
		{"normal >= high", PriorityNormal, PriorityHigh, false},
		{"high >= high", PriorityHigh, PriorityHigh, true},
		{"urgent >= high", PriorityUrgent, PriorityHigh, true},
		{"low >= urgent", PriorityLow, PriorityUrgent, false},
		{"normal >= urgent", PriorityNormal, PriorityUrgent, false},
		{"high >= urgent", PriorityHigh, PriorityUrgent, false},
		{"urgent >= urgent", PriorityUrgent, PriorityUrgent, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &AgentMessage{
				Type:        MessageTypeDirect,
				FromAgentID: "agent-1",
				Priority:    tt.msgPriority,
			}
			filter := MessageFilter{
				MinPriority: tt.minPriority,
			}
			got := bus.matchesFilter(msg, filter, "any")
			assert.Equal(t, tt.want, got, "Expected %v for %s priority vs min %s", tt.want, tt.msgPriority, tt.minPriority)
		})
	}
}

func TestMatchesFilter_NoPriorityFilter(t *testing.T) {
	bus := setupTestBus2(t)

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		Priority:    PriorityLow,
	}

	// No MinPriority set (empty string) - should match any priority
	filter := MessageFilter{}
	assert.True(t, bus.matchesFilter(msg, filter, "any"))
}

func TestMatchesFilter_CombinedFilters(t *testing.T) {
	bus := setupTestBus2(t)

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Priority:    PriorityHigh,
	}

	// All filters match
	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect},
		FromAgentIDs: []string{"agent-1"},
		ToAgentID:    "agent-2",
		MinPriority:  PriorityNormal,
	}
	assert.True(t, bus.matchesFilter(msg, filter, "agent-2"))

	// Type matches but from doesn't
	filter2 := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect},
		FromAgentIDs: []string{"agent-3"},
	}
	assert.False(t, bus.matchesFilter(msg, filter2, "agent-2"))
}

// ---------------------------------------------------------------------------
// Close tests
// ---------------------------------------------------------------------------

func TestClose_EmptyBus(t *testing.T) {
	bus := setupTestBus2(t)
	// Should not panic with no subscriptions
	bus.Close()
}

func TestClose_WithSubscriptions(t *testing.T) {
	bus := setupTestBus2(t)

	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect},
	}
	sub1 := bus.Subscribe("sub-1", "agent-1", filter)
	sub2 := bus.Subscribe("sub-2", "agent-2", filter)

	bus.Close()

	// All channels should be closed
	_, ok1 := <-sub1.Channel
	assert.False(t, ok1, "sub1 channel should be closed")

	_, ok2 := <-sub2.Channel
	assert.False(t, ok2, "sub2 channel should be closed")
}

// ---------------------------------------------------------------------------
// MessageType and Priority constants tests
// ---------------------------------------------------------------------------

func TestMessageTypeConstants(t *testing.T) {
	assert.Equal(t, MessageType("agent_message"), MessageTypeDirect)
	assert.Equal(t, MessageType("broadcast"), MessageTypeBroadcast)
	assert.Equal(t, MessageType("request"), MessageTypeRequest)
	assert.Equal(t, MessageType("response"), MessageTypeResponse)
	assert.Equal(t, MessageType("notification"), MessageTypeNotification)
	assert.Equal(t, MessageType("consensus_request"), MessageTypeConsensusRequest)
	assert.Equal(t, MessageType("consensus_vote"), MessageTypeConsensusVote)
}

func TestPriorityConstants(t *testing.T) {
	assert.Equal(t, Priority("low"), PriorityLow)
	assert.Equal(t, Priority("normal"), PriorityNormal)
	assert.Equal(t, Priority("high"), PriorityHigh)
	assert.Equal(t, Priority("urgent"), PriorityUrgent)
}

// ---------------------------------------------------------------------------
// AgentMessage struct tests
// ---------------------------------------------------------------------------

func TestAgentMessage_Fields(t *testing.T) {
	now := time.Now()
	delivered := now.Add(time.Second)
	read := now.Add(2 * time.Second)

	msg := AgentMessage{
		MessageID:        "msg-123",
		Type:             MessageTypeDirect,
		FromAgentID:      "agent-1",
		ToAgentID:        "agent-2",
		ToAgentIDs:       []string{"agent-3", "agent-4"},
		Subject:          "Test Subject",
		Body:             "Test Body",
		Payload:          map[string]interface{}{"key": "val"},
		Priority:         PriorityHigh,
		RequiresResponse: true,
		InReplyTo:        "msg-000",
		Context:          map[string]interface{}{"ctx": "data"},
		Timestamp:        now,
		Status:           "sent",
		DeliveredAt:      &delivered,
		ReadAt:           &read,
	}

	assert.Equal(t, "msg-123", msg.MessageID)
	assert.Equal(t, MessageTypeDirect, msg.Type)
	assert.Equal(t, "agent-1", msg.FromAgentID)
	assert.Equal(t, "agent-2", msg.ToAgentID)
	assert.Len(t, msg.ToAgentIDs, 2)
	assert.Equal(t, "Test Subject", msg.Subject)
	assert.Equal(t, "Test Body", msg.Body)
	assert.Equal(t, "val", msg.Payload["key"])
	assert.Equal(t, PriorityHigh, msg.Priority)
	assert.True(t, msg.RequiresResponse)
	assert.Equal(t, "msg-000", msg.InReplyTo)
	assert.Equal(t, "data", msg.Context["ctx"])
	assert.Equal(t, now, msg.Timestamp)
	assert.Equal(t, "sent", msg.Status)
	assert.NotNil(t, msg.DeliveredAt)
	assert.NotNil(t, msg.ReadAt)
}

// ---------------------------------------------------------------------------
// MessageFilter struct tests
// ---------------------------------------------------------------------------

func TestMessageFilter_Fields(t *testing.T) {
	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect, MessageTypeBroadcast},
		FromAgentIDs: []string{"agent-1", "agent-2"},
		ToAgentID:    "agent-3",
		Topics:       []string{"topic-a", "topic-b"},
		MinPriority:  PriorityHigh,
	}

	assert.Len(t, filter.MessageTypes, 2)
	assert.Len(t, filter.FromAgentIDs, 2)
	assert.Equal(t, "agent-3", filter.ToAgentID)
	assert.Len(t, filter.Topics, 2)
	assert.Equal(t, PriorityHigh, filter.MinPriority)
}

// ---------------------------------------------------------------------------
// Subscription struct tests
// ---------------------------------------------------------------------------

func TestSubscription_Fields(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect},
		ToAgentID:    "agent-1",
	}

	sub := bus.Subscribe("test-sub", "agent-1", filter)

	assert.Equal(t, "test-sub", sub.ID)
	assert.Equal(t, "agent-1", sub.AgentID)
	assert.Equal(t, filter, sub.Filter)
	assert.NotNil(t, sub.Channel)
}

// ---------------------------------------------------------------------------
// Multiple sequential sends
// ---------------------------------------------------------------------------

func TestSend_MultipleMessages(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()

	for i := 0; i < 10; i++ {
		msg := &AgentMessage{
			Type:        MessageTypeDirect,
			FromAgentID: "agent-1",
			ToAgentID:   "agent-2",
		}
		err := bus.Send(context.Background(), msg)
		require.NoError(t, err)
	}

	history := bus.GetHistory("agent-1", 20)
	assert.Len(t, history, 10)
}

// ---------------------------------------------------------------------------
// History trimming
// ---------------------------------------------------------------------------

func TestHistoryTrimming_ExactBoundary(t *testing.T) {
	bus := setupTestBus2(t)
	defer bus.Close()
	bus.maxHistory = 5

	for i := 0; i < 5; i++ {
		msg := &AgentMessage{
			Type:        MessageTypeBroadcast,
			FromAgentID: "agent-1",
			Subject:     "msg",
		}
		bus.Send(context.Background(), msg)
	}

	// At exactly max, should not trim
	history := bus.GetHistory("agent-1", 100)
	assert.Len(t, history, 5)

	// Add one more to trigger trim
	msg := &AgentMessage{
		Type:        MessageTypeBroadcast,
		FromAgentID: "agent-1",
		Subject:     "overflow",
	}
	bus.Send(context.Background(), msg)

	history = bus.GetHistory("agent-1", 100)
	assert.Len(t, history, 5, "History should be trimmed to maxHistory")
}
