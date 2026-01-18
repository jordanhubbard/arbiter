# Arbiter Architecture

## Overview

The Arbiter is a secure orchestration system that manages interactions between AI agents and providers. It maintains its own database and is the sole reader/writer to ensure data integrity.

## Core Concepts

### Agents
An **Agent** is an LLM (Large Language Model) wrapped in glue code that performs tasks. Agents are configured to use a specific provider and can have custom configurations.

### Providers
A **Provider** is an AI engine running on-premise or in the cloud (e.g., OpenAI, Anthropic, local models). Providers may require API credentials (keys) to communicate.

### Key Manager
The **Key Manager** securely stores provider credentials with strong encryption. Keys are encrypted at rest and only accessible when the key store is unlocked with a password.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        Arbiter                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Main Orchestrator                       │   │
│  │  - Manages Agent/Provider lifecycle                  │   │
│  │  - Coordinates operations                            │   │
│  │  - Ensures security and data integrity               │   │
│  └────────────┬──────────────────────┬──────────────────┘   │
│               │                      │                       │
│  ┌────────────▼──────────┐  ┌───────▼────────────────┐    │
│  │    Database Layer      │  │   Key Manager          │    │
│  │                        │  │                        │    │
│  │  - Agents table        │  │  - Encrypted storage   │    │
│  │  - Providers table     │  │  - AES-256-GCM         │    │
│  │  - SQLite backend      │  │  - PBKDF2 (100k iter)  │    │
│  │  - Foreign keys        │  │  - Per-key salt/nonce  │    │
│  └────────────────────────┘  └────────────────────────┘    │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │             Configuration                             │  │
│  │  - Password from env or secure prompt                │  │
│  │  - Data directory management                         │  │
│  │  - No password storage                               │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### 1. Initialization
```
User starts Arbiter
  ↓
Check ARBITER_PASSWORD env variable
  ↓
If not found, prompt user (hidden input)
  ↓
Initialize database connection
  ↓
Unlock key manager with password
  ↓
Ready to orchestrate
```

### 2. Creating a Provider with Credentials
```
User creates provider with API key
  ↓
Arbiter encrypts API key with key manager
  ↓
Store encrypted key with unique ID
  ↓
Store provider record in database with key_id reference
  ↓
Provider ready for use
```

### 3. Creating an Agent
```
User creates agent with provider reference
  ↓
Arbiter verifies provider exists
  ↓
Store agent record in database
  ↓
Foreign key ensures referential integrity
  ↓
Agent ready to use provider
```

### 4. Using an Agent
```
Retrieve agent by ID
  ↓
Get associated provider
  ↓
If provider requires key, decrypt from key manager
  ↓
Return agent, provider, and decrypted API key
  ↓
Use credentials to communicate with provider
```

## Security Model

### Encryption at Rest
- **Algorithm**: AES-256-GCM (Galois/Counter Mode)
- **Key Derivation**: PBKDF2 with SHA-256
- **Iterations**: 100,000 (protects against brute force)
- **Salt**: 32 bytes per key (unique)
- **Nonce**: 12 bytes per key (unique)

### Password Handling
- **Never stored**: Password exists only in memory
- **Environment variable**: `ARBITER_PASSWORD` (for automation)
- **Interactive prompt**: Hidden input using `golang.org/x/term`
- **Memory clearing**: Password cleared when key manager locks

### File Permissions
- **Key store**: `0600` (owner read/write only)
- **Database**: Default SQLite permissions
- **Data directory**: `0700` (owner access only)

### Key Store Structure
```json
{
  "keys": {
    "key_openai-gpt4": {
      "id": "key_openai-gpt4",
      "name": "OpenAI GPT-4",
      "description": "API Key for OpenAI GPT-4",
      "encrypted_data": "base64-encoded-encrypted-key",
      "created_at": "2026-01-18T17:00:00Z",
      "updated_at": "2026-01-18T17:00:00Z"
    }
  }
}
```

## Database Schema

### Providers Table
```sql
CREATE TABLE providers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,              -- openai, anthropic, local, etc.
    endpoint TEXT NOT NULL,          -- URL or path
    description TEXT,
    requires_key BOOLEAN NOT NULL,   -- Does it need credentials?
    key_id TEXT,                     -- Reference to key manager
    status TEXT NOT NULL,            -- active, inactive, etc.
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

### Agents Table
```sql
CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    provider_id TEXT NOT NULL,       -- Foreign key to providers
    status TEXT NOT NULL,            -- active, inactive, etc.
    config TEXT,                     -- JSON configuration
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
);
```

## API Usage Examples

### Creating a Provider
```go
provider := &models.Provider{
    ID:          "openai-gpt4",
    Name:        "OpenAI GPT-4",
    Type:        "openai",
    Endpoint:    "https://api.openai.com/v1",
    Description: "OpenAI GPT-4 API",
    RequiresKey: true,
    Status:      "active",
}

apiKey := "sk-..."
err := arbiter.CreateProvider(provider, apiKey)
```

### Creating an Agent
```go
agent := &models.Agent{
    ID:          "coding-agent",
    Name:        "Coding Assistant",
    Description: "AI coding assistant",
    ProviderID:  "openai-gpt4",
    Status:      "active",
    Config:      `{"model": "gpt-4", "temperature": 0.7}`,
}

err := arbiter.CreateAgent(agent)
```

### Using an Agent
```go
agent, provider, apiKey, err := arbiter.GetAgentWithProvider("coding-agent")
if err != nil {
    log.Fatal(err)
}

// Use agent, provider, and apiKey to make AI requests
// The apiKey is decrypted and ready to use
```

## Directory Structure

```
arbiter/
├── cmd/arbiter/              # Main application
│   └── main.go              # Entry point and orchestrator
├── internal/
│   ├── config/              # Configuration management
│   │   └── config.go        # Password handling, data paths
│   ├── database/            # Database layer
│   │   ├── database.go      # SQLite operations
│   │   └── database_test.go # Database tests
│   ├── keymanager/          # Key management
│   │   ├── keymanager.go    # Encryption/decryption
│   │   └── keymanager_test.go # Key manager tests
│   └── models/              # Data models
│       ├── agent.go         # Agent model
│       └── provider.go      # Provider model
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
├── README.md               # User documentation
├── ARCHITECTURE.md         # This file
└── LICENSE                 # License information
```

## Design Decisions

### Why SQLite?
- **Embedded**: No separate database server needed
- **ACID**: Full transaction support
- **Portable**: Single file database
- **Lightweight**: Perfect for local orchestration
- **Mature**: Well-tested and reliable

### Why AES-256-GCM?
- **Authenticated encryption**: Detects tampering
- **NIST approved**: Widely trusted standard
- **Fast**: Hardware acceleration on modern CPUs
- **Secure**: No known practical attacks

### Why PBKDF2?
- **Standard**: NIST and IETF approved
- **Tunable**: Iteration count can increase over time
- **Well-understood**: Extensively analyzed
- **Compatible**: Wide library support

### Why No Password Storage?
- **Security**: Reduces attack surface
- **Best practice**: Password should only exist in user's mind
- **Ephemeral**: Process memory is temporary
- **User control**: User must be present to unlock

## Future Considerations

### Possible Enhancements
- Add agent execution engine
- Implement provider communication layer
- Add task queuing and scheduling
- Support for agent chaining
- Monitoring and logging system
- Web UI for management
- Multi-user support with RBAC
- Backup and restore functionality

### Security Enhancements
- Hardware security module (HSM) support
- Biometric authentication
- Two-factor authentication
- Audit logging
- Key rotation
- Certificate-based authentication for providers

### Scalability
- Distributed orchestration
- Provider pooling
- Load balancing
- High availability
- Horizontal scaling
