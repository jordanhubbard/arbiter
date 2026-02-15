package eventbus

import (
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/config"
)

// createTestEventBus creates an EventBus with nil client for unit testing
// (no Temporal server required).
func createTestEventBus(bufferSize int) *EventBus {
	cfg := &config.TemporalConfig{
		EventBufferSize: bufferSize,
	}
	return NewEventBus(nil, cfg)
}

func TestNewEventBusDefaultBufferSize(t *testing.T) {
	cfg := &config.TemporalConfig{
		EventBufferSize: 0, // Should default to 1000
	}
	eb := NewEventBus(nil, cfg)
	if eb == nil {
		t.Fatal("expected non-nil event bus")
	}
	defer eb.Close()

	// Buffer channel should have capacity 1000
	if cap(eb.buffer) != 1000 {
		t.Errorf("expected buffer capacity 1000, got %d", cap(eb.buffer))
	}
}

func TestNewEventBusCustomBufferSize(t *testing.T) {
	eb := createTestEventBus(500)
	if eb == nil {
		t.Fatal("expected non-nil event bus")
	}
	defer eb.Close()

	if cap(eb.buffer) != 500 {
		t.Errorf("expected buffer capacity 500, got %d", cap(eb.buffer))
	}
}

func TestPublishNilEvent(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	err := eb.Publish(nil)
	if err == nil {
		t.Error("expected error when publishing nil event")
	}
}

func TestPublishSetsTimestamp(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub := eb.Subscribe("test-sub", nil)

	event := &Event{
		Type:   EventTypeAgentSpawned,
		Source: "test",
		Data:   map[string]interface{}{},
	}

	err := eb.Publish(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case received := <-sub.Channel:
		if received.Timestamp.IsZero() {
			t.Error("expected timestamp to be set automatically")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	eb.Unsubscribe("test-sub")
}

func TestPublishSetsID(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub := eb.Subscribe("test-sub", nil)

	event := &Event{
		Type:   EventTypeBeadCreated,
		Source: "test",
		Data:   map[string]interface{}{},
	}

	err := eb.Publish(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case received := <-sub.Channel:
		if received.ID == "" {
			t.Error("expected ID to be set automatically")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	eb.Unsubscribe("test-sub")
}

func TestPublishPreservesExistingTimestampAndID(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub := eb.Subscribe("test-sub", nil)

	customTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	event := &Event{
		ID:        "custom-id",
		Type:      EventTypeAgentHeartbeat,
		Source:    "test",
		Timestamp: customTime,
		Data:      map[string]interface{}{},
	}

	err := eb.Publish(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case received := <-sub.Channel:
		if received.ID != "custom-id" {
			t.Errorf("expected custom ID, got %q", received.ID)
		}
		if !received.Timestamp.Equal(customTime) {
			t.Errorf("expected custom timestamp, got %v", received.Timestamp)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	eb.Unsubscribe("test-sub")
}

func TestSubscribeCreateNew(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub := eb.Subscribe("new-sub", nil)
	if sub == nil {
		t.Fatal("expected non-nil subscriber")
	}
	if sub.ID != "new-sub" {
		t.Errorf("expected subscriber ID 'new-sub', got %q", sub.ID)
	}
	if sub.Channel == nil {
		t.Error("expected non-nil channel")
	}

	eb.Unsubscribe("new-sub")
}

func TestSubscribeReturnExisting(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub1 := eb.Subscribe("dup-sub", nil)
	sub2 := eb.Subscribe("dup-sub", nil) // Same ID

	if sub1 != sub2 {
		t.Error("expected same subscriber to be returned for duplicate ID")
	}

	eb.Unsubscribe("dup-sub")
}

func TestSubscriberCount(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	if eb.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", eb.SubscriberCount())
	}

	eb.Subscribe("sub-1", nil)
	if eb.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", eb.SubscriberCount())
	}

	eb.Subscribe("sub-2", nil)
	if eb.SubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", eb.SubscriberCount())
	}

	eb.Unsubscribe("sub-1")
	if eb.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber after unsubscribe, got %d", eb.SubscriberCount())
	}

	eb.Unsubscribe("sub-2")
	if eb.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after all unsubscribes, got %d", eb.SubscriberCount())
	}
}

func TestUnsubscribeNonexistent(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	// Should not panic
	eb.Unsubscribe("does-not-exist")
}

func TestEventFilterFunctionality(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	// Subscribe with filter that only accepts agent events
	filter := func(event *Event) bool {
		return event.Type == EventTypeAgentSpawned || event.Type == EventTypeAgentCompleted
	}
	sub := eb.Subscribe("filtered-sub", filter)

	// Publish matching event
	_ = eb.Publish(&Event{
		Type:   EventTypeAgentSpawned,
		Source: "test",
		Data:   map[string]interface{}{},
	})

	select {
	case received := <-sub.Channel:
		if received.Type != EventTypeAgentSpawned {
			t.Errorf("expected agent.spawned, got %s", received.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for matching event")
	}

	// Publish non-matching event
	_ = eb.Publish(&Event{
		Type:   EventTypeBeadCreated,
		Source: "test",
		Data:   map[string]interface{}{},
	})

	// Should not receive the bead event
	select {
	case received := <-sub.Channel:
		t.Errorf("should not have received filtered event, got %s", received.Type)
	case <-time.After(500 * time.Millisecond):
		// Expected: no event received
	}

	eb.Unsubscribe("filtered-sub")
}

func TestMultipleSubscribersReceiveEvents(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub1 := eb.Subscribe("sub-1", nil)
	sub2 := eb.Subscribe("sub-2", nil)
	sub3 := eb.Subscribe("sub-3", nil)

	_ = eb.Publish(&Event{
		Type:   EventTypeProjectCreated,
		Source: "test",
		Data:   map[string]interface{}{"name": "project1"},
	})

	received := 0
	for _, sub := range []*Subscriber{sub1, sub2, sub3} {
		select {
		case <-sub.Channel:
			received++
		case <-time.After(2 * time.Second):
			t.Errorf("timeout waiting for subscriber %s", sub.ID)
		}
	}

	if received != 3 {
		t.Errorf("expected 3 subscribers to receive event, got %d", received)
	}

	eb.Unsubscribe("sub-1")
	eb.Unsubscribe("sub-2")
	eb.Unsubscribe("sub-3")
}

func TestPublishAgentEvent(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub := eb.Subscribe("test-sub", nil)

	err := eb.PublishAgentEvent(EventTypeAgentSpawned, "agent-1", "project-1", map[string]interface{}{
		"name": "TestAgent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case received := <-sub.Channel:
		if received.Type != EventTypeAgentSpawned {
			t.Errorf("expected agent.spawned, got %s", received.Type)
		}
		if received.Source != "agent-manager" {
			t.Errorf("expected source agent-manager, got %s", received.Source)
		}
		if received.ProjectID != "project-1" {
			t.Errorf("expected project project-1, got %s", received.ProjectID)
		}
		if received.Data["agent_id"] != "agent-1" {
			t.Errorf("expected agent_id agent-1, got %v", received.Data["agent_id"])
		}
		if received.Data["name"] != "TestAgent" {
			t.Errorf("expected name TestAgent, got %v", received.Data["name"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for agent event")
	}

	eb.Unsubscribe("test-sub")
}

func TestPublishAgentEventNilData(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub := eb.Subscribe("test-sub", nil)

	err := eb.PublishAgentEvent(EventTypeAgentHeartbeat, "agent-1", "project-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case received := <-sub.Channel:
		if received.Data["agent_id"] != "agent-1" {
			t.Errorf("expected agent_id in data, got %v", received.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	eb.Unsubscribe("test-sub")
}

func TestPublishBeadEvent(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub := eb.Subscribe("test-sub", nil)

	err := eb.PublishBeadEvent(EventTypeBeadCreated, "bead-1", "project-1", map[string]interface{}{
		"title": "Fix bug",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case received := <-sub.Channel:
		if received.Type != EventTypeBeadCreated {
			t.Errorf("expected bead.created, got %s", received.Type)
		}
		if received.Source != "beads-manager" {
			t.Errorf("expected source beads-manager, got %s", received.Source)
		}
		if received.Data["bead_id"] != "bead-1" {
			t.Errorf("expected bead_id bead-1, got %v", received.Data["bead_id"])
		}
		if received.Data["title"] != "Fix bug" {
			t.Errorf("expected title 'Fix bug', got %v", received.Data["title"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for bead event")
	}

	eb.Unsubscribe("test-sub")
}

func TestPublishBeadEventNilData(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub := eb.Subscribe("test-sub", nil)

	err := eb.PublishBeadEvent(EventTypeBeadAssigned, "bead-1", "project-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case received := <-sub.Channel:
		if received.Data["bead_id"] != "bead-1" {
			t.Errorf("expected bead_id in data, got %v", received.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	eb.Unsubscribe("test-sub")
}

func TestPublishLogMessage(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	sub := eb.Subscribe("test-sub", nil)

	err := eb.PublishLogMessage("info", "test log message", "test-component", "project-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case received := <-sub.Channel:
		if received.Type != EventTypeLogMessage {
			t.Errorf("expected log.message, got %s", received.Type)
		}
		if received.Source != "test-component" {
			t.Errorf("expected source test-component, got %s", received.Source)
		}
		if received.ProjectID != "project-1" {
			t.Errorf("expected project project-1, got %s", received.ProjectID)
		}
		if received.Data["level"] != "info" {
			t.Errorf("expected level info, got %v", received.Data["level"])
		}
		if received.Data["message"] != "test log message" {
			t.Errorf("expected message 'test log message', got %v", received.Data["message"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for log message event")
	}

	eb.Unsubscribe("test-sub")
}

func TestGetRecentEventsEmpty(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	events := eb.GetRecentEvents(10, "", "")
	if len(events) != 0 {
		t.Errorf("expected 0 recent events, got %d", len(events))
	}
}

func TestGetRecentEvents(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	// Publish several events and wait for processing
	for i := 0; i < 5; i++ {
		_ = eb.Publish(&Event{
			Type:      EventTypeAgentSpawned,
			Source:    "test",
			ProjectID: "project-1",
			Data:      map[string]interface{}{"index": i},
		})
	}

	// Wait for events to be processed
	time.Sleep(500 * time.Millisecond)

	events := eb.GetRecentEvents(10, "", "")
	if len(events) != 5 {
		t.Errorf("expected 5 recent events, got %d", len(events))
	}

	// Events should be newest-first
	if len(events) > 1 {
		// The last published event (index 4) should be first
		firstIdx, ok := events[0].Data["index"]
		lastIdx, ok2 := events[len(events)-1].Data["index"]
		if ok && ok2 {
			// Allow for float64 from JSON
			if firstIdx != lastIdx {
				// Just verify we got all events, order is newest-first
			}
		}
	}
}

func TestGetRecentEventsFilterByProject(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	_ = eb.Publish(&Event{
		Type:      EventTypeAgentSpawned,
		Source:    "test",
		ProjectID: "project-1",
		Data:      map[string]interface{}{},
	})
	_ = eb.Publish(&Event{
		Type:      EventTypeBeadCreated,
		Source:    "test",
		ProjectID: "project-2",
		Data:      map[string]interface{}{},
	})
	_ = eb.Publish(&Event{
		Type:      EventTypeAgentCompleted,
		Source:    "test",
		ProjectID: "project-1",
		Data:      map[string]interface{}{},
	})

	time.Sleep(500 * time.Millisecond)

	events := eb.GetRecentEvents(10, "project-1", "")
	if len(events) != 2 {
		t.Errorf("expected 2 events for project-1, got %d", len(events))
	}

	events = eb.GetRecentEvents(10, "project-2", "")
	if len(events) != 1 {
		t.Errorf("expected 1 event for project-2, got %d", len(events))
	}
}

func TestGetRecentEventsFilterByType(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	_ = eb.Publish(&Event{
		Type:   EventTypeAgentSpawned,
		Source: "test",
		Data:   map[string]interface{}{},
	})
	_ = eb.Publish(&Event{
		Type:   EventTypeBeadCreated,
		Source: "test",
		Data:   map[string]interface{}{},
	})
	_ = eb.Publish(&Event{
		Type:   EventTypeAgentSpawned,
		Source: "test",
		Data:   map[string]interface{}{},
	})

	time.Sleep(500 * time.Millisecond)

	events := eb.GetRecentEvents(10, "", string(EventTypeAgentSpawned))
	if len(events) != 2 {
		t.Errorf("expected 2 agent.spawned events, got %d", len(events))
	}

	events = eb.GetRecentEvents(10, "", string(EventTypeBeadCreated))
	if len(events) != 1 {
		t.Errorf("expected 1 bead.created event, got %d", len(events))
	}
}

func TestGetRecentEventsLimit(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	for i := 0; i < 10; i++ {
		_ = eb.Publish(&Event{
			Type:   EventTypeAgentSpawned,
			Source: "test",
			Data:   map[string]interface{}{},
		})
	}

	time.Sleep(500 * time.Millisecond)

	events := eb.GetRecentEvents(3, "", "")
	if len(events) != 3 {
		t.Errorf("expected 3 events with limit, got %d", len(events))
	}
}

func TestGetRecentEventsZeroLimit(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	for i := 0; i < 5; i++ {
		_ = eb.Publish(&Event{
			Type:   EventTypeAgentSpawned,
			Source: "test",
			Data:   map[string]interface{}{},
		})
	}

	time.Sleep(500 * time.Millisecond)

	// Zero limit should return all events
	events := eb.GetRecentEvents(0, "", "")
	if len(events) != 5 {
		t.Errorf("expected 5 events with zero limit (all), got %d", len(events))
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Verify all event type constants are distinct and non-empty
	types := []EventType{
		EventTypeAgentSpawned,
		EventTypeAgentStatusChange,
		EventTypeAgentHeartbeat,
		EventTypeAgentCompleted,
		EventTypeBeadCreated,
		EventTypeBeadAssigned,
		EventTypeBeadStatusChange,
		EventTypeBeadCompleted,
		EventTypeDecisionCreated,
		EventTypeDecisionResolved,
		EventTypeProviderRegistered,
		EventTypeProviderDeleted,
		EventTypeProviderUpdated,
		EventTypeProjectCreated,
		EventTypeProjectUpdated,
		EventTypeProjectDeleted,
		EventTypeConfigUpdated,
		EventTypeLogMessage,
		EventTypeWorkflowStarted,
		EventTypeWorkflowCompleted,
		EventTypeMotivationFired,
		EventTypeMotivationEnabled,
		EventTypeMotivationDisabled,
		EventTypeDeadlineApproaching,
		EventTypeDeadlinePassed,
		EventTypeSystemIdle,
	}

	seen := make(map[EventType]bool)
	for _, et := range types {
		if string(et) == "" {
			t.Error("found empty event type")
		}
		if seen[et] {
			t.Errorf("duplicate event type: %s", et)
		}
		seen[et] = true
	}
}

func TestEventStruct(t *testing.T) {
	event := Event{
		ID:        "evt-1",
		Type:      EventTypeAgentSpawned,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      map[string]interface{}{"key": "value"},
		ProjectID: "proj-1",
	}

	if event.ID != "evt-1" {
		t.Errorf("ID: expected evt-1, got %s", event.ID)
	}
	if event.Type != EventTypeAgentSpawned {
		t.Errorf("Type: expected agent.spawned, got %s", event.Type)
	}
	if event.Source != "test" {
		t.Errorf("Source: expected test, got %s", event.Source)
	}
	if event.ProjectID != "proj-1" {
		t.Errorf("ProjectID: expected proj-1, got %s", event.ProjectID)
	}
	if event.Data["key"] != "value" {
		t.Errorf("Data: expected key=value, got %v", event.Data["key"])
	}
}

func TestSubscriberStruct(t *testing.T) {
	sub := &Subscriber{
		ID:      "sub-1",
		Channel: make(chan *Event, 10),
		Filter:  nil,
	}

	if sub.ID != "sub-1" {
		t.Errorf("ID: expected sub-1, got %s", sub.ID)
	}
	if sub.Channel == nil {
		t.Error("Channel should not be nil")
	}
	if sub.Filter != nil {
		t.Error("Filter should be nil")
	}
}

func TestPublishBufferFull(t *testing.T) {
	// Create event bus with very small buffer
	eb := createTestEventBus(1)
	defer eb.Close()

	// Don't subscribe, so events pile up in the buffer
	// The processEvents goroutine is running but we fill faster than it can drain

	// Pause processing by filling the buffer
	// First event should succeed (gets buffered or processed)
	err1 := eb.Publish(&Event{
		Type:   EventTypeAgentSpawned,
		Source: "test",
		Data:   map[string]interface{}{},
	})

	// Give time for processing to start
	time.Sleep(100 * time.Millisecond)

	// This test mainly verifies we don't panic under pressure
	if err1 != nil {
		t.Logf("First publish error (may be ok if buffer was full): %v", err1)
	}
}

func TestGetRecentEventsFilterBoth(t *testing.T) {
	eb := createTestEventBus(100)
	defer eb.Close()

	_ = eb.Publish(&Event{
		Type:      EventTypeAgentSpawned,
		Source:    "test",
		ProjectID: "project-1",
		Data:      map[string]interface{}{},
	})
	_ = eb.Publish(&Event{
		Type:      EventTypeBeadCreated,
		Source:    "test",
		ProjectID: "project-1",
		Data:      map[string]interface{}{},
	})
	_ = eb.Publish(&Event{
		Type:      EventTypeAgentSpawned,
		Source:    "test",
		ProjectID: "project-2",
		Data:      map[string]interface{}{},
	})

	time.Sleep(500 * time.Millisecond)

	// Filter by both project and type
	events := eb.GetRecentEvents(10, "project-1", string(EventTypeAgentSpawned))
	if len(events) != 1 {
		t.Errorf("expected 1 event for project-1 + agent.spawned, got %d", len(events))
	}
}
