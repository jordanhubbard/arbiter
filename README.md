# Arbiter

An AI Coding Agent Orchestrator for both on-prem and off-prem development.

Arbiter is a lightweight AI coding agent orchestrator, dispatcher, and automatic decision maker. Instead of being just another frontend to systems like Claude or Cursor, Arbiter intelligently routes requests to multiple AI providers and presents a unified OpenAI-compatible API.

## Features

- 🤖 **Multi-Provider Support**: Configure and use multiple AI providers (Claude, OpenAI, Cursor, Factory, and more)
- 🔒 **Secure Secret Storage**: API keys are encrypted and stored securely, never committed to git
- 🌐 **Dual Interface**: Both OpenAI-compatible REST API and web frontend
- 🔍 **Automatic Provider Discovery**: Looks up API endpoints for known providers or accepts custom URLs
- ⚡ **Lightweight**: Minimal overhead, runs as a background service
- 🎯 **Smart Routing**: Automatically routes requests to appropriate providers

## Installation

### Prerequisites

- Go 1.21 or higher (tested with Go 1.24)

### Build from Source

```bash
git clone https://github.com/jordanhubbard/arbiter.git
cd arbiter
go build
```

This will create an `arbiter` binary in the current directory.

## Quick Start

1. **Run Arbiter**:
   ```bash
   ./arbiter
   ```

2. **First-time Setup**: On first run, Arbiter will interactively guide you through configuring your AI providers:
   - Enter the names of providers you have access to (e.g., `claude, openai, cursor`)
   - For each provider, either:
     - Provide a specific API endpoint URL, or
     - Let Arbiter look up the standard endpoint for known providers
   - Enter your API key for each provider

3. **Access the Interfaces**:
   - **Web UI**: http://localhost:8080
   - **OpenAI-compatible API**: http://localhost:8080/v1/...
   - **Health Check**: http://localhost:8080/health

## Configuration

Arbiter stores configuration in two files in your home directory:

- `~/.arbiter.json`: Provider configurations (endpoints, names)
- `~/.arbiter_secrets`: Encrypted API keys (machine-specific encryption)

**Security Note**: These files are never committed to git. The secrets file uses AES-GCM encryption with a machine-specific key derived from hostname and user directory.

## API Endpoints

Arbiter provides an OpenAI-compatible API:

### Chat Completions
```bash
POST /v1/chat/completions
Content-Type: application/json

{
  "model": "claude-default",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ]
}
```

### Text Completions
```bash
POST /v1/completions
Content-Type: application/json

{
  "model": "openai-default",
  "prompt": "Once upon a time"
}
```

### List Models
```bash
GET /v1/models
```

### Health Check
```bash
GET /health
```

### List Providers
```bash
GET /api/providers
```

## Usage Examples

### Using with curl

```bash
# Chat completion
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-default",
    "messages": [{"role": "user", "content": "Write a haiku about coding"}]
  }'

# Check health
curl http://localhost:8080/health
```

### Using with Python OpenAI Client

```python
from openai import OpenAI

# Point the client to Arbiter
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed"  # Arbiter manages keys
)

response = client.chat.completions.create(
    model="claude-default",
    messages=[{"role": "user", "content": "Hello!"}]
)

print(response.choices[0].message.content)
```

## Supported Providers

Arbiter has built-in support for the following providers with automatic endpoint lookup:

- **Claude** (Anthropic): `https://api.anthropic.com/v1`
- **OpenAI**: `https://api.openai.com/v1`
- **Cursor**: `https://api.cursor.sh/v1`
- **Factory**: `https://api.factory.ai/v1`
- **Cohere**: `https://api.cohere.ai/v1`
- **HuggingFace**: `https://api-inference.huggingface.co`
- **Replicate**: `https://api.replicate.com/v1`
- **Together**: `https://api.together.xyz/v1`
- **Mistral**: `https://api.mistral.ai/v1`
- **Perplexity**: `https://api.perplexity.ai`

For any other provider, you can manually specify the API endpoint during setup.

## Architecture

```
┌─────────────────────────────────────────┐
│           User Application              │
│  (CLI, IDE Plugin, Web Client, etc.)    │
└───────────────┬─────────────────────────┘
                │
                │ OpenAI-compatible API
                │
┌───────────────▼─────────────────────────┐
│           Arbiter Server                │
│  ┌─────────────────────────────────┐   │
│  │  Request Router & Dispatcher    │   │
│  └─────────────────────────────────┘   │
│  ┌─────────────────────────────────┐   │
│  │    Encrypted Secret Store       │   │
│  └─────────────────────────────────┘   │
└───────────────┬─────────────────────────┘
                │
        ┌───────┴───────┬─────────┐
        │               │         │
┌───────▼──────┐ ┌──────▼─────┐ ┌▼────────┐
│   Claude     │ │   OpenAI   │ │ Cursor  │
│   Provider   │ │  Provider  │ │ Provider│
└──────────────┘ └────────────┘ └─────────┘
```

## Development

### Building

```bash
go build
```

### Running

```bash
./arbiter
```

### Project Structure

```
arbiter/
├── main.go                    # Application entry point
├── pkg/
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── secrets/
│   │   └── store.go          # Encrypted secret storage
│   └── server/
│       ├── server.go         # HTTP server implementation
│       └── types.go          # API types
├── go.mod                     # Go module definition
├── README.md                  # This file
└── .gitignore                # Git ignore rules
```

## Security Considerations

- API keys are encrypted using AES-GCM with a 256-bit key
- Encryption key is derived from machine-specific data (hostname + home directory)
- Secrets file has restricted permissions (0600)
- Configuration and secrets are stored in home directory, never in repository
- No secrets are logged or exposed in API responses

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

See LICENSE file for details.

## Roadmap

- [ ] Implement actual HTTP forwarding to providers
- [ ] Add streaming support for real-time responses
- [ ] Implement request/response logging and analytics
- [ ] Add support for provider-specific features
- [ ] Implement load balancing and failover
- [ ] Add authentication for Arbiter API
- [ ] Support for custom provider plugins
- [ ] Add metrics and monitoring endpoints
- [ ] Implement rate limiting per provider
- [ ] Add caching layer for responses

## Support

For issues, questions, or contributions, please use the GitHub issue tracker.

