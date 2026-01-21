# Plugin Development Guide

Welcome to the AgentiCorp Plugin Development Guide! This guide will help you create custom provider plugins to extend AgentiCorp with support for any AI provider.

## Table of Contents

1. [Overview](#overview)
2. [Plugin Architecture](#plugin-architecture)
3. [Quick Start](#quick-start)
4. [Plugin Interface](#plugin-interface)
5. [Creating an HTTP Plugin](#creating-an-http-plugin)
6. [Plugin Manifest](#plugin-manifest)
7. [Testing Your Plugin](#testing-your-plugin)
8. [Deployment](#deployment)
9. [Best Practices](#best-practices)
10. [Troubleshooting](#troubleshooting)
11. [Examples](#examples)

---

## Overview

AgentiCorp uses a plugin system to support custom AI providers. Plugins allow you to:

- **Add any AI provider** without modifying AgentiCorp's source code
- **Keep plugins isolated** - plugin crashes don't affect AgentiCorp
- **Hot-reload** plugins without restarting the system
- **Share plugins** with the community

### Plugin Types

AgentiCorp supports three plugin types:

1. **HTTP Plugins** - RESTful HTTP services (recommended)
2. **gRPC Plugins** - High-performance RPC (future)
3. **Built-in Plugins** - Compiled into AgentiCorp (advanced)

This guide focuses on **HTTP plugins** as they provide the best balance of:
- **Isolation**: Run as separate processes
- **Language flexibility**: Write in any language
- **Simplicity**: Standard HTTP/JSON APIs

---

## Plugin Architecture

```
┌─────────────┐         HTTP          ┌──────────────┐
│  AgentiCorp │ ◄──────────────────► │  Your Plugin │
└─────────────┘                       └──────────────┘
      │                                       │
      │                                       │
      ▼                                       ▼
  Plugin Loader                        AI Provider API
```

**Flow:**
1. AgentiCorp loads your plugin manifest
2. Plugin loader starts HTTP communication
3. AgentiCorp calls plugin endpoints (initialize, health, completions)
4. Plugin forwards requests to the AI provider
5. Plugin returns responses to AgentiCorp

---

## Quick Start

### Prerequisites

- Go 1.24+ (for testing with AgentiCorp)
- Any language/framework for your plugin (Python, Node.js, Go, etc.)
- An AI provider to integrate (OpenAI, Anthropic, local LLM, etc.)

### 1. Choose Your Language

**Python:**
```bash
mkdir my-plugin && cd my-plugin
python3 -m venv venv
source venv/bin/activate
pip install flask requests
```

**Node.js:**
```bash
mkdir my-plugin && cd my-plugin
npm init -y
npm install express axios
```

**Go:**
```bash
mkdir my-plugin && cd my-plugin
go mod init my-plugin
go get github.com/gin-gonic/gin
```

### 2. Implement the Plugin API

Your plugin must implement these HTTP endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/metadata` | Return plugin metadata |
| POST | `/initialize` | Initialize with config |
| GET | `/health` | Health check |
| POST | `/chat/completions` | Process completion request |
| GET | `/models` | List available models |
| POST | `/cleanup` | Cleanup before unload |

### 3. Create a Manifest

Create `plugin.yaml`:

```yaml
type: http
endpoint: http://localhost:8090
metadata:
  name: My Custom Plugin
  version: 1.0.0
  plugin_api_version: "1.0.0"
  provider_type: my-provider
  description: Integration with My AI Provider
  author: Your Name
  license: MIT
  capabilities:
    streaming: false
    function_calling: false
    vision: false
  config_schema:
    - name: api_key
      type: string
      required: true
      description: API key for authentication
      sensitive: true
auto_start: true
health_check_interval: 60
```

### 4. Run Your Plugin

```bash
# Start your plugin server
python plugin.py  # or npm start, or go run .

# Test it
curl http://localhost:8090/metadata
curl http://localhost:8090/health
```

### 5. Deploy to AgentiCorp

```bash
# Copy manifest to AgentiCorp plugins directory
cp plugin.yaml /path/to/agenticorp/plugins/my-provider/plugin.yaml

# Restart AgentiCorp or trigger hot-reload
# Plugin will be automatically discovered and loaded
```

---

## Plugin Interface

### Metadata Endpoint

**GET /metadata**

Returns plugin information for registration.

**Response:**
```json
{
  "name": "My Custom Plugin",
  "version": "1.0.0",
  "plugin_api_version": "1.0.0",
  "provider_type": "my-provider",
  "description": "Integration with My AI Provider",
  "author": "Your Name",
  "homepage": "https://github.com/you/my-plugin",
  "license": "MIT",
  "capabilities": {
    "streaming": false,
    "function_calling": false,
    "vision": false
  },
  "config_schema": [
    {
      "name": "api_key",
      "type": "string",
      "required": true,
      "description": "API key",
      "sensitive": true
    }
  ]
}
```

### Initialize Endpoint

**POST /initialize**

Called once when the plugin is loaded. Receives configuration.

**Request:**
```json
{
  "api_key": "sk-...",
  "endpoint": "https://api.example.com",
  "timeout": 30
}
```

**Response:**
```json
{}
```

### Health Check Endpoint

**GET /health**

Called periodically to verify plugin is operational.

**Response:**
```json
{
  "healthy": true,
  "message": "OK",
  "latency_ms": 5,
  "timestamp": "2026-01-21T10:00:00Z",
  "details": {
    "provider_status": "connected"
  }
}
```

### Chat Completions Endpoint

**POST /chat/completions**

Main endpoint for processing completion requests.

**Request:**
```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "temperature": 0.7,
  "max_tokens": 1000
}
```

**Response:**
```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1706000000,
  "model": "gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 8,
    "total_tokens": 18,
    "cost_usd": 0.000054
  }
}
```

### Models Endpoint

**GET /models**

Returns list of models supported by this provider.

**Response:**
```json
[
  {
    "id": "gpt-4",
    "name": "GPT-4",
    "description": "Most capable model",
    "context_window": 8192,
    "max_output_tokens": 4096,
    "cost_per_mtoken": 0.03,
    "capabilities": {
      "streaming": true,
      "function_calling": true,
      "vision": false
    }
  }
]
```

### Cleanup Endpoint

**POST /cleanup**

Called before plugin is unloaded. Use this to close connections, save state, etc.

**Response:**
```json
{}
```

---

## Creating an HTTP Plugin

### Python Example (Flask)

Complete working example in `examples/plugins/example-python/plugin.py`.

Key points:
```python
from flask import Flask, jsonify, request
import requests

app = Flask(__name__)
config = {}

@app.route('/metadata')
def metadata():
    return jsonify({
        "name": "Python Example Plugin",
        "version": "1.0.0",
        "provider_type": "example-python",
        # ... rest of metadata
    })

@app.route('/initialize', methods=['POST'])
def initialize():
    global config
    config = request.json
    return jsonify({})

@app.route('/health')
def health():
    return jsonify({
        "healthy": True,
        "message": "OK",
        "latency_ms": 5,
        "timestamp": datetime.now().isoformat()
    })

@app.route('/chat/completions', methods=['POST'])
def chat_completions():
    req = request.json
    
    # Call your AI provider API
    response = requests.post(
        config['endpoint'] + '/chat/completions',
        json=req,
        headers={'Authorization': f"Bearer {config['api_key']}"}
    )
    
    return jsonify(response.json())

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8090)
```

### Node.js Example (Express)

```javascript
const express = require('express');
const axios = require('axios');

const app = express();
app.use(express.json());

let config = {};

app.get('/metadata', (req, res) => {
  res.json({
    name: "Node.js Example Plugin",
    version: "1.0.0",
    provider_type: "example-nodejs",
    // ... rest of metadata
  });
});

app.post('/initialize', (req, res) => {
  config = req.body;
  res.json({});
});

app.get('/health', (req, res) => {
  res.json({
    healthy: true,
    message: "OK",
    latency_ms: 5,
    timestamp: new Date().toISOString()
  });
});

app.post('/chat/completions', async (req, res) => {
  try {
    const response = await axios.post(
      config.endpoint + '/chat/completions',
      req.body,
      { headers: { 'Authorization': `Bearer ${config.api_key}` } }
    );
    res.json(response.data);
  } catch (error) {
    res.status(500).json({
      code: "provider_unavailable",
      message: error.message,
      transient: true
    });
  }
});

app.listen(8090, () => console.log('Plugin running on port 8090'));
```

---

## Plugin Manifest

The `plugin.yaml` file tells AgentiCorp how to load and use your plugin.

### Complete Example

```yaml
# Plugin type: http, grpc, or builtin
type: http

# Endpoint where your plugin is running
endpoint: http://localhost:8090

# Metadata (must match /metadata endpoint)
metadata:
  name: My AI Provider Plugin
  version: 1.2.0
  plugin_api_version: "1.0.0"
  provider_type: my-ai-provider
  description: Integration with My AI Provider's API
  author: Your Name <you@example.com>
  homepage: https://github.com/you/my-plugin
  license: MIT
  
  # What does your plugin support?
  capabilities:
    streaming: true          # Streaming responses
    function_calling: false  # Function/tool calling
    vision: false           # Image inputs
    embeddings: false       # Generate embeddings
    fine_tuning: false      # Fine-tuning support
  
  # Configuration schema
  config_schema:
    - name: api_key
      type: string
      required: true
      description: API key for authentication
      sensitive: true
      
    - name: endpoint
      type: string
      required: false
      description: Custom API endpoint
      default: "https://api.example.com"
      
    - name: timeout
      type: int
      required: false
      description: Request timeout in seconds
      default: 30
      validation:
        min: 1
        max: 300

# Auto-start this plugin when AgentiCorp starts
auto_start: true

# Health check interval in seconds
health_check_interval: 60

# Optional: Command to start plugin process (if not already running)
command: python3
args:
  - plugin.py
  - --port
  - "8090"

# Optional: Environment variables for the plugin process
env:
  LOG_LEVEL: info
  PORT: "8090"
```

---

## Testing Your Plugin

### Manual Testing

```bash
# 1. Start your plugin
python plugin.py

# 2. Test metadata
curl http://localhost:8090/metadata | jq

# 3. Test initialization
curl -X POST http://localhost:8090/initialize \
  -H "Content-Type: application/json" \
  -d '{"api_key": "test-key"}'

# 4. Test health
curl http://localhost:8090/health | jq

# 5. Test completion
curl -X POST http://localhost:8090/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }' | jq

# 6. Test models
curl http://localhost:8090/models | jq
```

### Integration Testing

Create a test manifest and load it in AgentiCorp:

```bash
# Copy to plugins directory
mkdir -p /path/to/agenticorp/plugins/test
cp plugin.yaml /path/to/agenticorp/plugins/test/

# Restart AgentiCorp or trigger reload
# Check logs for "Plugin loaded: test"
```

### Automated Testing

Create a test suite for your plugin:

**Python (pytest):**
```python
import pytest
import requests

BASE_URL = "http://localhost:8090"

def test_metadata():
    resp = requests.get(f"{BASE_URL}/metadata")
    assert resp.status_code == 200
    data = resp.json()
    assert data["name"]
    assert data["provider_type"]

def test_health():
    resp = requests.get(f"{BASE_URL}/health")
    assert resp.status_code == 200
    data = resp.json()
    assert data["healthy"] == True

def test_completion():
    resp = requests.post(f"{BASE_URL}/chat/completions", json={
        "model": "test-model",
        "messages": [{"role": "user", "content": "Test"}]
    })
    assert resp.status_code == 200
    data = resp.json()
    assert "choices" in data
```

---

## Deployment

### Local Development

```bash
# 1. Start your plugin
python plugin.py &

# 2. Copy manifest
mkdir -p ~/.agenticorp/plugins/my-plugin
cp plugin.yaml ~/.agenticorp/plugins/my-plugin/

# 3. Restart AgentiCorp
docker compose restart agenticorp
```

### Production Deployment

**Option 1: Docker**

```dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY plugin.py .
EXPOSE 8090
CMD ["python", "plugin.py"]
```

```bash
docker build -t my-plugin .
docker run -d -p 8090:8090 my-plugin
```

**Option 2: Systemd Service**

```ini
[Unit]
Description=My AgentiCorp Plugin
After=network.target

[Service]
Type=simple
User=agenticorp
WorkingDirectory=/opt/my-plugin
ExecStart=/usr/bin/python3 plugin.py
Restart=always

[Install]
WantedBy=multi-user.target
```

---

## Best Practices

### 1. Error Handling

Return structured errors:

```python
{
  "code": "rate_limit_exceeded",
  "message": "Rate limit exceeded, retry after 60s",
  "transient": True,
  "details": {
    "retry_after": 60,
    "limit": 1000,
    "remaining": 0
  }
}
```

Use standard error codes:
- `authentication_failed` - Invalid API key
- `rate_limit_exceeded` - Rate limit hit
- `invalid_request` - Bad request parameters
- `model_not_found` - Model doesn't exist
- `provider_unavailable` - Provider is down
- `timeout` - Request timeout
- `internal_error` - Plugin internal error

### 2. Logging

Log important events:
```python
import logging

logger = logging.getLogger(__name__)

@app.route('/chat/completions', methods=['POST'])
def chat_completions():
    logger.info(f"Received completion request for model: {request.json.get('model')}")
    try:
        # ... process
        logger.info(f"Completion successful, tokens: {response['usage']['total_tokens']}")
    except Exception as e:
        logger.error(f"Completion failed: {e}")
```

### 3. Configuration Validation

Validate config on initialize:
```python
@app.route('/initialize', methods=['POST'])
def initialize():
    config = request.json
    
    if 'api_key' not in config:
        return jsonify({
            "code": "invalid_request",
            "message": "api_key is required"
        }), 400
    
    if not config['api_key'].startswith('sk-'):
        return jsonify({
            "code": "authentication_failed",
            "message": "Invalid API key format"
        }), 401
    
    # Store config
    app.config.update(config)
    return jsonify({})
```

### 4. Health Checks

Return detailed health information:
```python
@app.route('/health')
def health():
    try:
        # Test connection to provider
        resp = requests.get(provider_url, timeout=5)
        
        return jsonify({
            "healthy": True,
            "message": "OK",
            "latency_ms": int(resp.elapsed.total_seconds() * 1000),
            "timestamp": datetime.now().isoformat(),
            "details": {
                "provider_status": "connected",
                "models_available": len(get_models())
            }
        })
    except Exception as e:
        return jsonify({
            "healthy": False,
            "message": str(e),
            "latency_ms": 5000,
            "timestamp": datetime.now().isoformat()
        })
```

### 5. Cost Tracking

Include cost information in responses:
```python
# Calculate cost based on token usage
cost_per_token = 0.00003  # $0.03 per 1000 tokens
cost_usd = usage['total_tokens'] * cost_per_token

response['usage']['cost_usd'] = cost_usd
```

---

## Troubleshooting

### Plugin Not Loading

**Problem:** Plugin doesn't appear in AgentiCorp

**Solutions:**
1. Check manifest syntax: `yamllint plugin.yaml`
2. Verify endpoint is accessible: `curl http://localhost:8090/metadata`
3. Check AgentiCorp logs: `docker logs agenticorp | grep plugin`
4. Ensure `auto_start: true` in manifest
5. Verify plugins directory path

### Health Check Failing

**Problem:** Plugin shows as unhealthy

**Solutions:**
1. Test health endpoint: `curl http://localhost:8090/health`
2. Check provider connectivity
3. Verify API credentials
4. Review plugin logs
5. Increase timeout in plugin

### Completion Requests Failing

**Problem:** Completions return errors

**Solutions:**
1. Verify request format matches API
2. Check API key is valid
3. Test provider API directly
4. Review token limits
5. Check rate limits

### Plugin Crashes

**Problem:** Plugin process crashes

**Solutions:**
1. Add exception handling to all endpoints
2. Validate all inputs
3. Add request timeouts
4. Implement graceful error recovery
5. Use process supervisor (systemd, supervisor, etc.)

---

## Examples

See the `examples/plugins/` directory for complete working examples:

1. **`example-python/`** - Python/Flask plugin with OpenAI
2. **`example-nodejs/`** - Node.js/Express plugin
3. **`example-go/`** - Go/Gin plugin

Each example includes:
- Complete plugin implementation
- Plugin manifest
- README with setup instructions
- Test suite
- Docker configuration

---

## Next Steps

1. **Start with an example** - Copy one of the examples and modify it
2. **Test thoroughly** - Use the testing section to verify your plugin
3. **Share with community** - Submit to plugin registry (see bd-088)
4. **Get feedback** - Join AgentiCorp community discussions

## Resources

- [Plugin API Reference](PLUGIN_API.md)
- [Example Plugins](../examples/plugins/)
- [Plugin Registry](PLUGIN_REGISTRY.md)
- [GitHub Discussions](https://github.com/jordanhubbard/AgentiCorp/discussions)
- [Plugin API Version History](PLUGIN_VERSIONS.md)

---

**Happy plugin development!** 🚀

If you have questions or run into issues, please:
- Check the [Troubleshooting](#troubleshooting) section
- Review the [Examples](#examples)
- Ask in [GitHub Discussions](https://github.com/jordanhubbard/AgentiCorp/discussions)
- File an issue if you find a bug
