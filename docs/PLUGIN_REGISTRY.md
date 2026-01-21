# AgentiCorp Plugin Registry

The AgentiCorp Plugin Registry is a community-driven marketplace for discovering and sharing provider plugins.

## Overview

The registry enables:
- 🔍 **Discovery** - Search and browse community plugins
- 📦 **Installation** - One-command plugin installation
- ⭐ **Quality** - Ratings and reviews from users
- 🛡️ **Security** - Verified and trusted plugins
- 📊 **Analytics** - Download counts and popularity metrics

## Registry Structure

### Official Registry

The official registry is hosted at: `https://registry.agenticorp.io`

**Local Registry**:  
AgentiCorp also supports local registries stored in:
- `~/.agenticorp/registry/` (user-level)
- `/etc/agenticorp/registry/` (system-level)
- Project-specific: `.agenticorp/registry/`

### Registry Index

The registry maintains an index of available plugins in `registry.json`:

```json
{
  "version": "1.0",
  "plugins": [
    {
      "id": "openai-plugin",
      "name": "OpenAI Provider",
      "provider_type": "openai",
      "description": "Official OpenAI API integration",
      "author": "AgentiCorp Team",
      "version": "1.2.0",
      "license": "MIT",
      "homepage": "https://github.com/agenticorp/plugin-openai",
      "repository": "https://github.com/agenticorp/plugin-openai",
      "downloads": 15420,
      "rating": 4.8,
      "reviews": 156,
      "verified": true,
      "tags": ["official", "openai", "gpt"],
      "capabilities": {
        "streaming": true,
        "function_calling": true,
        "vision": true
      },
      "install": {
        "type": "http",
        "manifest_url": "https://raw.githubusercontent.com/agenticorp/plugin-openai/main/plugin.yaml",
        "docker_image": "agenticorp/plugin-openai:1.2.0"
      },
      "published_at": "2026-01-15T10:00:00Z",
      "updated_at": "2026-01-20T14:30:00Z"
    }
  ]
}
```

## Using the Registry

### Search for Plugins

```bash
# Search by name or tags
agenticorp plugin search openai

# List all plugins
agenticorp plugin list

# Show plugin details
agenticorp plugin show openai-plugin
```

### Install a Plugin

```bash
# Install from registry
agenticorp plugin install openai-plugin

# Install specific version
agenticorp plugin install openai-plugin@1.2.0

# Install from URL
agenticorp plugin install https://example.com/plugin.yaml
```

### Update Plugins

```bash
# Update all plugins
agenticorp plugin update

# Update specific plugin
agenticorp plugin update openai-plugin
```

### Remove Plugins

```bash
# Remove a plugin
agenticorp plugin remove openai-plugin
```

## Publishing Plugins

### Prerequisites

1. **Working Plugin** - Fully tested and documented
2. **GitHub Repository** - Public repo with source code
3. **Manifest File** - `plugin.yaml` in repo root
4. **README** - Clear documentation
5. **License** - Open source license (MIT, Apache, etc.)

### Submission Process

#### 1. Prepare Your Plugin

Ensure your plugin meets quality standards:

- ✅ All required endpoints implemented
- ✅ Comprehensive error handling
- ✅ Health checks working
- ✅ Documentation complete
- ✅ Tests included
- ✅ Example usage provided
- ✅ License file present

#### 2. Test Thoroughly

```bash
# Run automated tests
pytest tests/

# Test all endpoints
./test_plugin.sh

# Load test
ab -n 1000 -c 10 http://localhost:8090/health
```

#### 3. Create Submission

Create `registry-submission.json`:

```json
{
  "plugin": {
    "id": "my-ai-plugin",
    "name": "My AI Provider Plugin",
    "provider_type": "my-provider",
    "description": "Integration with My AI Provider",
    "author": "Your Name <you@example.com>",
    "version": "1.0.0",
    "license": "MIT",
    "homepage": "https://github.com/you/my-plugin",
    "repository": "https://github.com/you/my-plugin",
    "tags": ["ai", "llm", "custom"],
    "capabilities": {
      "streaming": true,
      "function_calling": false,
      "vision": false
    },
    "install": {
      "type": "http",
      "manifest_url": "https://raw.githubusercontent.com/you/my-plugin/main/plugin.yaml",
      "docker_image": "you/my-plugin:1.0.0"
    },
    "screenshots": [
      "https://github.com/you/my-plugin/raw/main/docs/screenshot.png"
    ],
    "documentation_url": "https://github.com/you/my-plugin/blob/main/README.md"
  },
  "submission": {
    "contact_email": "you@example.com",
    "notes": "First release of my plugin"
  }
}
```

#### 4. Submit to Registry

```bash
# Submit via CLI
agenticorp plugin submit registry-submission.json

# Or via web interface
# Visit: https://registry.agenticorp.io/submit
# Upload: registry-submission.json
```

#### 5. Review Process

1. **Automated Checks** - Manifest validation, security scan
2. **Manual Review** - Code quality, documentation review
3. **Testing** - Automated and manual testing
4. **Approval** - Plugin added to registry (typically 3-5 days)

### Quality Standards

To be accepted into the registry, plugins must meet:

**Required:**
- ✅ Implements all required endpoints
- ✅ Returns proper error codes
- ✅ Includes health check
- ✅ Has README with setup instructions
- ✅ Open source license
- ✅ No security vulnerabilities
- ✅ Passes automated tests

**Recommended:**
- ⭐ Comprehensive documentation
- ⭐ Example usage
- ⭐ Docker image available
- ⭐ Test suite included
- ⭐ CI/CD configured
- ⭐ Versioned releases

## Registry API

### Get Plugin List

**GET /api/v1/plugins**

Query parameters:
- `search` - Search term
- `tags` - Filter by tags (comma-separated)
- `verified` - Filter verified plugins (true/false)
- `sort` - Sort by: downloads, rating, updated
- `limit` - Results per page (default: 20)
- `offset` - Pagination offset

Response:
```json
{
  "total": 42,
  "plugins": [...],
  "page": 1,
  "pages": 3
}
```

### Get Plugin Details

**GET /api/v1/plugins/:id**

Response:
```json
{
  "id": "openai-plugin",
  "name": "OpenAI Provider",
  "description": "...",
  "versions": [
    {
      "version": "1.2.0",
      "published_at": "2026-01-20T14:30:00Z",
      "changes": "Added vision support"
    }
  ],
  "stats": {
    "downloads": 15420,
    "downloads_last_30d": 1234,
    "stars": 450
  },
  "reviews": [
    {
      "rating": 5,
      "comment": "Excellent plugin!",
      "author": "user123",
      "date": "2026-01-18"
    }
  ]
}
```

### Submit Plugin

**POST /api/v1/plugins**

Headers:
- `Authorization: Bearer <your-api-key>`

Body: `registry-submission.json`

Response:
```json
{
  "submission_id": "sub-123456",
  "status": "pending_review",
  "message": "Plugin submitted for review",
  "estimated_review_time": "3-5 days"
}
```

### Get Submission Status

**GET /api/v1/submissions/:id**

Response:
```json
{
  "submission_id": "sub-123456",
  "status": "approved",
  "plugin_id": "my-ai-plugin",
  "reviewed_by": "reviewer@agenticorp.io",
  "reviewed_at": "2026-01-22T10:00:00Z",
  "notes": "Approved! Great plugin."
}
```

## Local Registry

AgentiCorp supports local plugin registries for:
- Private/enterprise plugins
- Development and testing
- Air-gapped environments
- Custom plugin collections

### Setting Up Local Registry

```bash
# Create registry directory
mkdir -p ~/.agenticorp/registry

# Create index
cat > ~/.agenticorp/registry/registry.json << 'EOF'
{
  "version": "1.0",
  "plugins": []
}
EOF
```

### Adding Plugins to Local Registry

```bash
# Add plugin entry to registry.json
cat >> ~/.agenticorp/registry/registry.json << 'EOF'
{
  "id": "my-local-plugin",
  "name": "My Local Plugin",
  "manifest_url": "file:///path/to/plugin.yaml"
}
EOF

# Or use CLI
agenticorp plugin add-local \
  --id my-local-plugin \
  --manifest /path/to/plugin.yaml
```

### Configure Registry Sources

Edit `~/.agenticorp/config.yaml`:

```yaml
plugin_registry:
  sources:
    - name: official
      url: https://registry.agenticorp.io
      enabled: true
      
    - name: local
      url: file://~/.agenticorp/registry
      enabled: true
      
    - name: enterprise
      url: https://plugins.company.com
      enabled: true
      auth:
        type: token
        token: ${ENTERPRISE_REGISTRY_TOKEN}
```

## Plugin Reviews and Ratings

### Submitting a Review

```bash
# Rate a plugin
agenticorp plugin rate openai-plugin --rating 5

# Add review
agenticorp plugin review openai-plugin \
  --rating 5 \
  --comment "Excellent plugin, works perfectly!"
```

### Review Guidelines

When reviewing plugins:
- ✅ Be constructive and specific
- ✅ Mention what works well
- ✅ Report bugs or issues encountered
- ✅ Suggest improvements
- ❌ Don't post spam or offensive content
- ❌ Don't review your own plugins

## Security

### Verified Plugins

Plugins marked as "verified" have been:
- Reviewed by AgentiCorp team
- Security scanned for vulnerabilities
- Tested with AgentiCorp
- Maintained by trusted authors

### Security Best Practices

**For Users:**
- Only install plugins from trusted sources
- Review plugin code before installation
- Check plugin permissions and capabilities
- Keep plugins updated
- Report security issues

**For Developers:**
- Never include API keys in code
- Use environment variables for secrets
- Keep dependencies updated
- Follow secure coding practices
- Respond quickly to security reports

### Reporting Security Issues

If you discover a security vulnerability:

1. **Do not** disclose publicly
2. Email: security@agenticorp.io
3. Include: plugin ID, description, steps to reproduce
4. Wait for response before disclosure

## Plugin Categories

Plugins are organized into categories:

- **Official** - Maintained by AgentiCorp team
- **Verified** - Reviewed and verified by AgentiCorp
- **Community** - Community-contributed plugins
- **Enterprise** - Commercial/enterprise plugins
- **Experimental** - Beta/experimental plugins

## Statistics and Analytics

Plugin authors can access analytics:

- Download counts (total, daily, monthly)
- Active installations
- Version distribution
- Geographic distribution
- Error rates
- User ratings and reviews

Access at: `https://registry.agenticorp.io/dashboard`

## Future Features

Planned enhancements:

- 🔄 **Automatic Updates** - Background plugin updates
- 🔐 **Plugin Signing** - Cryptographic signatures
- 📦 **Dependency Management** - Plugin dependencies
- 🌐 **Multi-Registry** - Multiple registry sources
- 💰 **Paid Plugins** - Commercial plugin marketplace
- 🏆 **Plugin Awards** - Recognize excellent plugins
- 📈 **Advanced Analytics** - Detailed usage metrics

## Contributing

Help improve the registry:

1. **Submit Plugins** - Share your plugins
2. **Review Plugins** - Test and review plugins
3. **Report Issues** - Bug reports and feedback
4. **Improve Docs** - Contribute to documentation
5. **Code Contributions** - Help build registry features

## Resources

- [Plugin Development Guide](PLUGIN_DEVELOPMENT.md)
- [Registry API Documentation](REGISTRY_API.md)
- [Security Guidelines](SECURITY.md)
- [GitHub Repository](https://github.com/jordanhubbard/AgentiCorp)
- [Community Forum](https://forum.agenticorp.io)

## Support

- **Documentation**: https://docs.agenticorp.io
- **Issues**: https://github.com/jordanhubbard/AgentiCorp/issues
- **Discussions**: https://github.com/jordanhubbard/AgentiCorp/discussions
- **Email**: support@agenticorp.io

---

**Join the community and share your plugins!** 🚀
