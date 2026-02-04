# Conversation Context Architecture

## Overview

This document describes the conversation session management system that enables multi-turn conversations between agents and beads. This is the foundation for iterative problem-solving, debugging, and complex tasks that require multiple rounds of investigation.

## Current Problem

The current implementation in `internal/worker/worker.go:86-147` sends only a single system + user message with no conversation history:

```go
// Current implementation (single-shot)
req := &provider.ChatCompletionRequest{
    Model: w.provider.Config.Model,
    Messages: []provider.ChatMessage{
        {Role: "system", Content: systemPrompt},  // Persona instructions
        {Role: "user", Content: userPrompt},      // Task description
    },
    Temperature: 0.7,
}
```

**Problems:**
- Each dispatch is a fresh conversation with no memory
- Agents cannot ask clarifying questions and get answers
- Agents cannot try multiple approaches and learn from failures
- Agents cannot reference previous findings in their investigation
- No way to build on previous context across redispatches

## Solution: Conversation Context

### Core Concept

Each bead gets a **conversation session** that persists across multiple dispatches. The session maintains:
- Complete message history (system, user, assistant messages)
- Session metadata (bead_id, agent_id, project_id)
- Lifecycle management (creation, continuation, expiration)

### Data Model

#### ConversationContext

```go
// pkg/models/conversation.go
type ConversationContext struct {
    SessionID   string                 `json:"session_id" db:"session_id"`
    BeadID      string                 `json:"bead_id" db:"bead_id"`
    ProjectID   string                 `json:"project_id" db:"project_id"`
    Messages    []provider.ChatMessage `json:"messages" db:"messages"`  // JSONB in PostgreSQL
    CreatedAt   time.Time             `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time             `json:"updated_at" db:"updated_at"`
    ExpiresAt   time.Time             `json:"expires_at" db:"expires_at"`
    TokenCount  int                   `json:"token_count" db:"token_count"`  // Track total tokens
    Metadata    map[string]string     `json:"metadata" db:"metadata"`        // JSONB
}

type ChatMessage struct {
    Role      string    `json:"role"`       // "system", "user", "assistant"
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
    TokenCount int      `json:"token_count,omitempty"`
}
```

**Key Fields:**
- `session_id`: UUID identifying this conversation session
- `bead_id`: Links session to specific bead being worked on
- `messages`: Full conversation history as JSONB array
- `expires_at`: Sessions expire after 24 hours (configurable)
- `token_count`: Track cumulative tokens to manage costs
- `metadata`: Agent assignments, provider info, etc.

### Database Schema

```sql
-- internal/database/migrations_conversation.go
CREATE TABLE conversation_contexts (
    session_id   TEXT PRIMARY KEY,
    bead_id      TEXT NOT NULL,
    project_id   TEXT NOT NULL,
    messages     JSONB NOT NULL DEFAULT '[]',
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMP NOT NULL,
    token_count  INTEGER NOT NULL DEFAULT 0,
    metadata     JSONB NOT NULL DEFAULT '{}',

    CONSTRAINT fk_bead FOREIGN KEY (bead_id) REFERENCES beads(id) ON DELETE CASCADE,
    CONSTRAINT fk_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX idx_conversation_bead ON conversation_contexts(bead_id);
CREATE INDEX idx_conversation_expires ON conversation_contexts(expires_at);
CREATE INDEX idx_conversation_updated ON conversation_contexts(updated_at);
```

**Indexes:**
- `bead_id`: Fast lookup of conversation by bead
- `expires_at`: Efficient expiration cleanup
- `updated_at`: Find recent conversations for caching

### Session Lifecycle

#### 1. Session Creation (First Dispatch)

```
Dispatcher receives bead → Check bead context for session_id
  ├─ session_id exists? → Load session (go to #2)
  └─ session_id missing? → Create new session

Create Session:
1. Generate UUID: session_id
2. Initialize ConversationContext:
   - session_id = UUID
   - bead_id = bead.ID
   - project_id = bead.ProjectID
   - messages = [system_message] (persona instructions)
   - expires_at = now + 24h
   - metadata = {agent_id, provider_id, ...}
3. Save to database
4. Store session_id in bead.Context["conversation_session_id"]
5. Pass to Worker
```

#### 2. Session Continuation (Redispatch)

```
Dispatcher receives bead with session_id
  ├─ Load session from database
  ├─ Check expiration
  │   ├─ Expired? → Create new session (go to #1)
  │   └─ Valid? → Continue
  ├─ Append new user message to history
  ├─ Pass full history to Worker
  └─ Worker sends full history to LLM

Worker receives response:
1. Append assistant response to messages
2. Update token_count
3. Update updated_at
4. Save to database
5. Update bead context with results
```

#### 3. Session Expiration

```
Automatic Cleanup (Cron Job):
- Run every 1 hour
- DELETE FROM conversation_contexts WHERE expires_at < NOW()
- Or: Archive old sessions to S3/cold storage

On-Demand Expiration:
- When loading session, check expires_at
- If expired, treat as non-existent (create new)
```

### Token Limit Handling

LLMs have token limits (e.g., Claude: 200K, GPT-4: 128K). Long conversations must be truncated.

#### Strategy 1: Sliding Window (Simple)

```
If total_tokens > model_limit * 0.8:
    Keep: [system_message, last_N_messages]
    Truncate: middle messages

Example (limit = 100K tokens):
    System: 5K tokens (always keep)
    Message 1-10: 60K tokens (truncate)
    Message 11-20: 30K tokens (keep recent)
    → Total: 35K tokens (within limit)
```

#### Strategy 2: Summarization (Advanced)

```
If total_tokens > model_limit * 0.8:
    1. Keep system message
    2. Summarize messages 1 to N-10
    3. Keep last 10 messages verbatim
    4. Insert summary as special message:
       {role: "system", content: "Previous conversation summary: ..."}
```

**Implementation Choice:**
- **Phase 1**: Use Strategy 1 (sliding window) - simple, works well
- **Phase 2**: Add Strategy 2 (summarization) - better context retention

### Integration Points

#### 1. Dispatcher Integration

**File:** `internal/dispatch/dispatcher.go`

**Current Code (line 536):**
```go
result, execErr := d.agents.ExecuteTask(ctx, ag.ID, task)
```

**New Code:**
```go
// Check for existing conversation session
sessionID := candidate.Context["conversation_session_id"]
var session *models.ConversationContext

if sessionID != "" {
    // Load existing session
    session, err = d.conversationManager.GetSession(ctx, sessionID)
    if err != nil || session.ExpiresAt.Before(time.Now()) {
        // Session expired or not found, create new
        session = nil
        sessionID = ""
    }
}

if session == nil {
    // Create new session
    session, err = d.conversationManager.CreateSession(ctx, &models.ConversationContext{
        SessionID:  uuid.New().String(),
        BeadID:     candidate.ID,
        ProjectID:  candidate.ProjectID,
        ExpiresAt:  time.Now().Add(24 * time.Hour),
        Messages:   []provider.ChatMessage{},
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create conversation session: %w", err)
    }

    // Store session ID in bead context
    candidate.Context["conversation_session_id"] = session.SessionID
    d.beads.UpdateBead(candidate.ID, map[string]interface{}{
        "context": candidate.Context,
    })
}

// Pass session to task
task.ConversationSession = session

result, execErr := d.agents.ExecuteTask(ctx, ag.ID, task)
```

#### 2. Worker Integration

**File:** `internal/worker/worker.go`

**Current Code (line 86-147):**
```go
func (w *Worker) ExecuteTask(ctx context.Context, task *Task) (*TaskResult, error) {
    systemPrompt := w.buildSystemPrompt()
    userPrompt := task.Description

    req := &provider.ChatCompletionRequest{
        Model: w.provider.Config.Model,
        Messages: []provider.ChatMessage{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: userPrompt},
        },
    }
    // ...
}
```

**New Code:**
```go
func (w *Worker) ExecuteTask(ctx context.Context, task *Task) (*TaskResult, error) {
    var messages []provider.ChatMessage

    if task.ConversationSession != nil {
        // Load conversation history
        messages = task.ConversationSession.Messages

        // If first message (no history yet), add system prompt
        if len(messages) == 0 {
            systemPrompt := w.buildSystemPrompt()
            messages = append(messages, provider.ChatMessage{
                Role:      "system",
                Content:   systemPrompt,
                Timestamp: time.Now(),
            })
        }

        // Append new user message
        userPrompt := task.Description
        if task.Context != "" {
            userPrompt = fmt.Sprintf("%s\n\nContext:\n%s", userPrompt, task.Context)
        }

        messages = append(messages, provider.ChatMessage{
            Role:      "user",
            Content:   userPrompt,
            Timestamp: time.Now(),
        })

        // Handle token limits
        messages = w.handleTokenLimits(messages)

    } else {
        // Fallback to single-shot (backward compatibility)
        systemPrompt := w.buildSystemPrompt()
        userPrompt := task.Description
        if task.Context != "" {
            userPrompt = fmt.Sprintf("%s\n\nContext:\n%s", userPrompt, task.Context)
        }
        messages = []provider.ChatMessage{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: userPrompt},
        }
    }

    req := &provider.ChatCompletionRequest{
        Model:    w.provider.Config.Model,
        Messages: messages,  // Full conversation history
        Temperature: 0.7,
    }

    resp, err := w.provider.Protocol.CreateChatCompletion(ctx, req)
    if err != nil {
        return nil, err
    }

    // Append assistant response to history
    assistantMessage := provider.ChatMessage{
        Role:       "assistant",
        Content:    resp.Choices[0].Message.Content,
        Timestamp:  time.Now(),
        TokenCount: resp.Usage.CompletionTokens,
    }

    if task.ConversationSession != nil {
        messages = append(messages, assistantMessage)
        task.ConversationSession.Messages = messages
        task.ConversationSession.TokenCount += resp.Usage.TotalTokens
        task.ConversationSession.UpdatedAt = time.Now()

        // Save updated session
        err = w.conversationManager.UpdateSession(ctx, task.ConversationSession)
        if err != nil {
            log.Printf("Failed to update conversation session: %v", err)
        }
    }

    result := &TaskResult{
        TaskID:      task.ID,
        WorkerID:    w.id,
        AgentID:     w.agent.ID,
        Response:    resp.Choices[0].Message.Content,
        TokensUsed:  resp.Usage.TotalTokens,
        CompletedAt: time.Now(),
        Success:     true,
    }

    return result, nil
}

func (w *Worker) handleTokenLimits(messages []provider.ChatMessage) []provider.ChatMessage {
    // Get model token limit
    modelLimit := w.getModelTokenLimit()
    maxTokens := int(float64(modelLimit) * 0.8)  // Use 80% of limit

    // Calculate current tokens
    totalTokens := 0
    for _, msg := range messages {
        // Rough estimate: 1 token ~= 4 characters
        totalTokens += len(msg.Content) / 4
    }

    if totalTokens <= maxTokens {
        return messages  // No truncation needed
    }

    // Strategy 1: Sliding window
    // Keep system message + last N messages
    systemMsg := messages[0]  // Always system message

    // Find how many recent messages fit
    recentTokens := len(systemMsg.Content) / 4
    keepMessages := []provider.ChatMessage{systemMsg}

    for i := len(messages) - 1; i > 0; i-- {
        msgTokens := len(messages[i].Content) / 4
        if recentTokens+msgTokens > maxTokens {
            break
        }
        recentTokens += msgTokens
        keepMessages = append([]provider.ChatMessage{messages[i]}, keepMessages...)
    }

    // Add truncation notice
    if len(keepMessages) < len(messages) {
        truncatedCount := len(messages) - len(keepMessages)
        noticeMsg := provider.ChatMessage{
            Role:      "system",
            Content:   fmt.Sprintf("[Note: %d older messages truncated to stay within token limit]", truncatedCount),
            Timestamp: time.Now(),
        }
        keepMessages = append([]provider.ChatMessage{systemMsg, noticeMsg}, keepMessages[1:]...)
    }

    return keepMessages
}
```

### Configuration

**File:** `config.yaml`

```yaml
conversation:
  # Session expiration (default: 24 hours)
  session_expiration: 24h

  # Cleanup interval (default: 1 hour)
  cleanup_interval: 1h

  # Token limit strategy: "sliding_window" or "summarization"
  token_strategy: sliding_window

  # Max tokens per conversation (80% of model limit)
  max_tokens: 100000

  # Number of recent messages to always keep
  keep_recent_messages: 10
```

### API Endpoints

```go
// GET /api/v1/conversations/:id
// Returns full conversation history
type GetConversationResponse struct {
    SessionID   string          `json:"session_id"`
    BeadID      string          `json:"bead_id"`
    ProjectID   string          `json:"project_id"`
    Messages    []ChatMessage   `json:"messages"`
    CreatedAt   time.Time       `json:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at"`
    ExpiresAt   time.Time       `json:"expires_at"`
    TokenCount  int             `json:"token_count"`
}

// GET /api/v1/beads/:id/conversation
// Get conversation for specific bead
// Returns same structure as above

// DELETE /api/v1/conversations/:id
// Delete/cleanup session
type DeleteConversationResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
}

// POST /api/v1/conversations/:id/reset
// Clear history but keep session
// Useful for starting fresh while keeping session metadata
type ResetConversationRequest struct {
    KeepSystemMessage bool `json:"keep_system_message"`  // Keep first system message
}
```

### Benefits

1. **Iterative Problem-Solving**: Agents can investigate step-by-step
2. **Clarification**: Agents can ask questions and get answers
3. **Learning**: Agents remember what they've tried and what failed
4. **Context Retention**: No need to re-explain everything on each dispatch
5. **Better Debugging**: Agents can follow a trail of investigation
6. **Cost Tracking**: Track tokens per conversation for analytics

### Example: Bug Investigation with Conversation History

**Dispatch 1:**
```
User: "Investigate memory leak in order processing"
Agent: "I'll start by reading the order processing code.
        [reads internal/orders/processor.go]
        I see several goroutines spawned but no cleanup.
        Let me check if there's a defer statement..."
```

**Dispatch 2 (Redispatch):**
```
System: [Previous conversation history included]
User: "Continue investigation"
Agent: "Continuing from where I left off. I found that goroutines
        are spawned in ProcessOrder() but there's no context cancellation.
        Let me search for other files that spawn goroutines..."
```

**Dispatch 3:**
```
System: [Full conversation history]
User: "Continue investigation"
Agent: "Based on my previous investigation, I've identified the issue:
        1. ProcessOrder() spawns goroutines (line 45)
        2. No context.WithCancel() used
        3. No goroutine cleanup on error

        Proposed fix: Add context cancellation and defer cleanup.
        Creating fix bead..."
```

**Without Conversation History:**
Each dispatch would start from scratch, repeating the same investigation.

**With Conversation History:**
Agent builds on previous work, remembers findings, and makes progress.

### Migration Plan

1. **Database Migration**: Add `conversation_contexts` table
2. **Backward Compatibility**: Worker falls back to single-shot if no session
3. **Gradual Rollout**: Enable per-project with feature flag
4. **Monitoring**: Track session count, token usage, expiration rate
5. **Cleanup**: Schedule cron job for expired sessions

### Performance Considerations

- **Session Lookup**: Indexed by bead_id for O(1) lookup
- **Token Counting**: Approximate (4 chars = 1 token) for speed
- **JSONB Storage**: PostgreSQL efficiently stores/queries JSON
- **Caching**: Cache recent 100 sessions in memory for faster access

### Security Considerations

- **Expiration**: Sessions auto-expire to prevent unbounded growth
- **Isolation**: Each session tied to specific bead/project
- **No Cross-Contamination**: Sessions cannot leak between beads
- **Audit Trail**: Full message history for debugging/review

---

## Next Steps

1. **Implementation**: See Story ac-c0f.2 - Implement ConversationContext Model
2. **Testing**: See Story ac-c0f.3 - Add Conversation History to Worker
3. **Integration**: See Story ac-c0f.4 - Update Dispatcher for Session Management
4. **API**: See Story ac-c0f.5 - Add Session Management API Endpoints

---

**Document Version:** 1.0
**Last Updated:** 2026-02-04
**Author:** AgentiCorp Enhancement Initiative
**Status:** Design Complete, Ready for Implementation
