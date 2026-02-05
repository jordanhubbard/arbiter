package messaging

import (
	"context"
	"fmt"
)

// ActionMessageSender adapts AgentMessageBus to the actions.MessageSender interface
type ActionMessageSender struct {
	bus           *AgentMessageBus
	agentRegistry AgentRegistry
}

// AgentRegistry provides agent discovery functionality
type AgentRegistry interface {
	FindAgentByRole(ctx context.Context, role string) (string, error)
	ListAgents(ctx context.Context) ([]AgentInfo, error)
}

// AgentInfo contains basic agent information
type AgentInfo struct {
	AgentID      string
	PersonaType  string
	Capabilities []string
	Status       string
}

// NewActionMessageSender creates a new ActionMessageSender
func NewActionMessageSender(bus *AgentMessageBus, registry AgentRegistry) *ActionMessageSender {
	return &ActionMessageSender{
		bus:           bus,
		agentRegistry: registry,
	}
}

// SendMessage sends a message to another agent
func (s *ActionMessageSender) SendMessage(ctx context.Context, fromAgentID, toAgentID, messageType, subject, body string, payload map[string]interface{}) (string, error) {
	// Map action message types to agent message types
	var msgType MessageType
	switch messageType {
	case "question":
		msgType = MessageTypeDirect
	case "delegation":
		msgType = MessageTypeRequest
	case "notification":
		msgType = MessageTypeNotification
	default:
		return "", fmt.Errorf("unsupported message type: %s", messageType)
	}

	// Determine priority based on message type
	var priority Priority
	switch messageType {
	case "question":
		priority = PriorityNormal
	case "delegation":
		priority = PriorityHigh
	case "notification":
		priority = PriorityNormal
	default:
		priority = PriorityNormal
	}

	// Build context from payload
	context := make(map[string]interface{})
	if payload != nil {
		for k, v := range payload {
			context[k] = v
		}
	}

	// Create and send message
	msg := &AgentMessage{
		Type:             msgType,
		FromAgentID:      fromAgentID,
		ToAgentID:        toAgentID,
		Subject:          subject,
		Body:             body,
		Priority:         priority,
		RequiresResponse: messageType == "question" || messageType == "delegation",
		Context:          context,
		Payload:          payload,
	}

	if err := s.bus.Send(ctx, msg); err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	return msg.MessageID, nil
}

// FindAgentByRole finds an agent ID by role/persona type
func (s *ActionMessageSender) FindAgentByRole(ctx context.Context, role string) (string, error) {
	if s.agentRegistry == nil {
		return "", fmt.Errorf("agent registry not configured")
	}

	return s.agentRegistry.FindAgentByRole(ctx, role)
}
