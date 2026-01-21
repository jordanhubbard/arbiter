# Example Python Plugin

This is a complete working example of an AgentiCorp plugin implemented in Python using Flask.

## Features

- ✅ All required plugin endpoints
- ✅ Configuration validation
- ✅ Health checking with provider connectivity test
- ✅ Error handling with proper error codes
- ✅ Cost calculation
- ✅ Comprehensive logging
- ✅ OpenAI-compatible API integration

## Prerequisites

- Python 3.8+
- pip

## Setup

```bash
# Create virtual environment
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install flask requests

# Or install from requirements.txt
pip install -r requirements.txt
```

## Running the Plugin

```bash
# Start the plugin server
python plugin.py

# You should see:
# INFO - Starting Example Python Plugin on port 8090
# INFO - Metadata: Example Python Plugin v1.0.0
# ...
```

## Testing

### Manual Testing

```bash
# Test metadata endpoint
curl http://localhost:8090/metadata | jq

# Test health endpoint
curl http://localhost:8090/health | jq

# Initialize plugin
curl -X POST http://localhost:8090/initialize \
  -H "Content-Type: application/json" \
  -d '{"api_key": "your-api-key-here"}' | jq

# Test completion
curl -X POST http://localhost:8090/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }' | jq

# Get models
curl http://localhost:8090/models | jq
```

### Automated Tests

```bash
# Install test dependencies
pip install pytest requests

# Run tests
pytest test_plugin.py -v
```

## Deployment

### Option 1: Direct Deployment

```bash
# Start the plugin
python plugin.py &

# Copy manifest to AgentiCorp
mkdir -p /path/to/agenticorp/plugins/example-python
cp plugin.yaml /path/to/agenticorp/plugins/example-python/

# Restart AgentiCorp to load the plugin
```

### Option 2: Docker

```bash
# Build image
docker build -t example-python-plugin .

# Run container
docker run -d \
  -p 8090:8090 \
  --name example-python-plugin \
  example-python-plugin

# Check logs
docker logs example-python-plugin
```

### Option 3: Systemd Service

Create `/etc/systemd/system/example-python-plugin.service`:

```ini
[Unit]
Description=AgentiCorp Example Python Plugin
After=network.target

[Service]
Type=simple
User=agenticorp
WorkingDirectory=/opt/example-python-plugin
ExecStart=/usr/bin/python3 /opt/example-python-plugin/plugin.py
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start service
sudo systemctl enable example-python-plugin
sudo systemctl start example-python-plugin

# Check status
sudo systemctl status example-python-plugin
```

## Configuration

The plugin accepts configuration via the `/initialize` endpoint:

```json
{
  "api_key": "sk-...",
  "endpoint": "https://api.openai.com/v1",
  "timeout": 30
}
```

Configuration is validated according to the schema in `plugin.yaml`.

## Error Handling

The plugin returns structured errors compatible with AgentiCorp:

```json
{
  "code": "rate_limit_exceeded",
  "message": "Rate limit exceeded",
  "transient": true,
  "details": {
    "status_code": 429,
    "latency_ms": 123
  }
}
```

Error codes:
- `authentication_failed` - Invalid API key (401)
- `rate_limit_exceeded` - Rate limit hit (429)
- `invalid_request` - Bad request (400)
- `model_not_found` - Model doesn't exist (404)
- `provider_unavailable` - Provider is down (5xx)
- `timeout` - Request timeout (504)
- `internal_error` - Plugin internal error (500)

## Logging

The plugin uses Python's logging module:

```
2026-01-21 10:00:00 - INFO - Starting Example Python Plugin on port 8090
2026-01-21 10:00:05 - INFO - Metadata requested
2026-01-21 10:00:10 - INFO - Initializing with config: ['api_key', 'endpoint']
2026-01-21 10:00:15 - INFO - Completion request for model: gpt-3.5-turbo
2026-01-21 10:00:17 - INFO - Completion successful, latency: 2000ms
```

## Customization

To adapt this plugin for your AI provider:

1. Update `METADATA` with your provider details
2. Update `MODELS` with your supported models
3. Modify the API calls in `chat_completions()` to match your provider's API
4. Adjust error handling for provider-specific errors
5. Update configuration schema as needed

## Troubleshooting

**Plugin won't start:**
- Check Python version: `python3 --version` (need 3.8+)
- Verify dependencies: `pip list | grep -E "flask|requests"`
- Check port availability: `lsof -i :8090`

**Health checks failing:**
- Verify API key is valid
- Check endpoint URL is correct
- Test provider connection: `curl https://api.openai.com/v1/models -H "Authorization: Bearer $API_KEY"`

**Completions failing:**
- Check request format matches provider API
- Verify model name is correct
- Review plugin logs for details
- Test directly against provider API

## Next Steps

- Read the [Plugin Development Guide](../../../docs/PLUGIN_DEVELOPMENT.md)
- Explore other examples
- Customize for your AI provider
- Share with the community!

## License

MIT License - see LICENSE file for details
