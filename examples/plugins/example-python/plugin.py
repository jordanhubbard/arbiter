#!/usr/bin/env python3
"""
Example Loom Plugin - Python Implementation

This is a complete working example of a Loom provider plugin
implemented in Python using Flask. It demonstrates:

- All required plugin endpoints
- Proper error handling
- Configuration management
- Health checking
- Integration with an OpenAI-compatible API

Usage:
  pip install flask requests
  python plugin.py

Then test:
  curl http://localhost:8090/metadata
  curl http://localhost:8090/health
"""

from flask import Flask, jsonify, request
from datetime import datetime
import requests
import logging
import sys

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = Flask(__name__)

# Plugin configuration (set via /initialize)
config = {
    'api_key': '',
    'endpoint': 'https://api.openai.com/v1',
    'timeout': 30,
}

# Plugin metadata
METADATA = {
    "name": "Example Python Plugin",
    "version": "1.0.0",
    "plugin_api_version": "1.0.0",
    "provider_type": "example-python",
    "description": "Example plugin demonstrating Python implementation",
    "author": "Loom Team",
    "homepage": "https://github.com/jordanhubbard/Loom",
    "license": "MIT",
    "capabilities": {
        "streaming": False,
        "function_calling": True,
        "vision": False,
        "embeddings": False,
        "fine_tuning": False
    },
    "config_schema": [
        {
            "name": "api_key",
            "type": "string",
            "required": True,
            "description": "OpenAI API key",
            "sensitive": True
        },
        {
            "name": "endpoint",
            "type": "string",
            "required": False,
            "description": "API endpoint URL",
            "default": "https://api.openai.com/v1"
        },
        {
            "name": "timeout",
            "type": "int",
            "required": False,
            "description": "Request timeout in seconds",
            "default": 30,
            "validation": {
                "min": 1,
                "max": 300
            }
        }
    ]
}

# Supported models
MODELS = [
    {
        "id": "gpt-4",
        "name": "GPT-4",
        "description": "Most capable model, best for complex tasks",
        "context_window": 8192,
        "max_output_tokens": 4096,
        "cost_per_mtoken": 0.03,
        "capabilities": {
            "streaming": True,
            "function_calling": True,
            "vision": False
        }
    },
    {
        "id": "gpt-3.5-turbo",
        "name": "GPT-3.5 Turbo",
        "description": "Fast and cost-effective model",
        "context_window": 4096,
        "max_output_tokens": 4096,
        "cost_per_mtoken": 0.001,
        "capabilities": {
            "streaming": True,
            "function_calling": True,
            "vision": False
        }
    }
]


@app.route('/metadata', methods=['GET'])
def get_metadata():
    """Return plugin metadata."""
    logger.info("Metadata requested")
    return jsonify(METADATA)


@app.route('/initialize', methods=['POST'])
def initialize():
    """Initialize plugin with configuration."""
    global config
    
    try:
        new_config = request.json or {}
        logger.info(f"Initializing with config: {list(new_config.keys())}")
        
        # Validate required fields
        if 'api_key' not in new_config:
            return jsonify({
                "code": "invalid_request",
                "message": "api_key is required",
                "transient": False
            }), 400
        
        # Update configuration
        config.update(new_config)
        
        # Test connection
        try:
            resp = requests.get(
                f"{config['endpoint']}/models",
                headers={'Authorization': f"Bearer {config['api_key']}"},
                timeout=5
            )
            if resp.status_code != 200:
                logger.warning(f"Provider returned status {resp.status_code}")
        except Exception as e:
            logger.warning(f"Could not verify connection: {e}")
        
        logger.info("Initialization successful")
        return jsonify({})
        
    except Exception as e:
        logger.error(f"Initialization failed: {e}")
        return jsonify({
            "code": "internal_error",
            "message": str(e),
            "transient": False
        }), 500


@app.route('/health', methods=['GET'])
def health_check():
    """Perform health check."""
    start_time = datetime.now()
    
    try:
        # Check if initialized
        if not config.get('api_key'):
            return jsonify({
                "healthy": False,
                "message": "Not initialized",
                "latency_ms": 0,
                "timestamp": datetime.now().isoformat()
            })
        
        # Ping provider
        resp = requests.get(
            f"{config['endpoint']}/models",
            headers={'Authorization': f"Bearer {config['api_key']}"},
            timeout=5
        )
        
        latency_ms = int((datetime.now() - start_time).total_seconds() * 1000)
        
        if resp.status_code == 200:
            return jsonify({
                "healthy": True,
                "message": "OK",
                "latency_ms": latency_ms,
                "timestamp": datetime.now().isoformat(),
                "details": {
                    "provider_status": "connected",
                    "models_available": len(MODELS)
                }
            })
        else:
            return jsonify({
                "healthy": False,
                "message": f"Provider returned status {resp.status_code}",
                "latency_ms": latency_ms,
                "timestamp": datetime.now().isoformat()
            })
            
    except requests.Timeout:
        latency_ms = int((datetime.now() - start_time).total_seconds() * 1000)
        return jsonify({
            "healthy": False,
            "message": "Health check timeout",
            "latency_ms": latency_ms,
            "timestamp": datetime.now().isoformat()
        })
    except Exception as e:
        latency_ms = int((datetime.now() - start_time).total_seconds() * 1000)
        logger.error(f"Health check failed: {e}")
        return jsonify({
            "healthy": False,
            "message": str(e),
            "latency_ms": latency_ms,
            "timestamp": datetime.now().isoformat()
        })


@app.route('/chat/completions', methods=['POST'])
def chat_completions():
    """Handle chat completion request."""
    try:
        req_data = request.json
        logger.info(f"Completion request for model: {req_data.get('model')}")
        
        # Validate request
        if not req_data.get('model'):
            return jsonify({
                "code": "invalid_request",
                "message": "model is required",
                "transient": False
            }), 400
        
        if not req_data.get('messages'):
            return jsonify({
                "code": "invalid_request",
                "message": "messages is required",
                "transient": False
            }), 400
        
        # Forward request to provider
        headers = {
            'Authorization': f"Bearer {config['api_key']}",
            'Content-Type': 'application/json'
        }
        
        start_time = datetime.now()
        
        resp = requests.post(
            f"{config['endpoint']}/chat/completions",
            json=req_data,
            headers=headers,
            timeout=config.get('timeout', 30)
        )
        
        latency_ms = int((datetime.now() - start_time).total_seconds() * 1000)
        
        if resp.status_code == 200:
            result = resp.json()
            
            # Add cost calculation
            if 'usage' in result and 'total_tokens' in result['usage']:
                tokens = result['usage']['total_tokens']
                cost_per_token = 0.00003  # Example: $0.03 per 1K tokens
                result['usage']['cost_usd'] = tokens * cost_per_token
            
            logger.info(f"Completion successful, latency: {latency_ms}ms")
            return jsonify(result)
        
        # Handle error responses
        error_body = resp.json() if resp.headers.get('content-type') == 'application/json' else {}
        error_msg = error_body.get('error', {}).get('message', resp.text)
        
        # Map HTTP status to plugin error code
        if resp.status_code == 401:
            code = "authentication_failed"
        elif resp.status_code == 429:
            code = "rate_limit_exceeded"
        elif resp.status_code == 404:
            code = "model_not_found"
        elif resp.status_code >= 500:
            code = "provider_unavailable"
        else:
            code = "internal_error"
        
        logger.error(f"Completion failed: {code} - {error_msg}")
        
        return jsonify({
            "code": code,
            "message": error_msg,
            "transient": code in ["rate_limit_exceeded", "provider_unavailable", "timeout"],
            "details": {
                "status_code": resp.status_code,
                "latency_ms": latency_ms
            }
        }), resp.status_code
        
    except requests.Timeout:
        logger.error("Completion request timeout")
        return jsonify({
            "code": "timeout",
            "message": "Request timeout",
            "transient": True
        }), 504
    except Exception as e:
        logger.error(f"Completion request failed: {e}")
        return jsonify({
            "code": "internal_error",
            "message": str(e),
            "transient": False
        }), 500


@app.route('/models', methods=['GET'])
def get_models():
    """Return list of available models."""
    logger.info("Models requested")
    return jsonify(MODELS)


@app.route('/cleanup', methods=['POST'])
def cleanup():
    """Cleanup resources before plugin unload."""
    logger.info("Cleanup requested")
    # Close connections, save state, etc.
    return jsonify({})


def main():
    """Run the plugin server."""
    port = 8090
    logger.info(f"Starting Example Python Plugin on port {port}")
    logger.info(f"Metadata: {METADATA['name']} v{METADATA['version']}")
    logger.info("Endpoints:")
    logger.info(f"  GET  http://localhost:{port}/metadata")
    logger.info(f"  POST http://localhost:{port}/initialize")
    logger.info(f"  GET  http://localhost:{port}/health")
    logger.info(f"  POST http://localhost:{port}/chat/completions")
    logger.info(f"  GET  http://localhost:{port}/models")
    logger.info(f"  POST http://localhost:{port}/cleanup")
    
    app.run(host='0.0.0.0', port=port, debug=False)


if __name__ == '__main__':
    main()
