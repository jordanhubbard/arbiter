# Contributing to Arbiter

Thank you for your interest in contributing to Arbiter! This document provides guidelines and instructions for contributing.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- [bd (beads)](https://github.com/steveyegge/beads) - for task tracking integration

### Development Setup

1. **Clone the repository**

```bash
git clone https://github.com/jordanhubbard/arbiter.git
cd arbiter
```

2. **Set up development environment**

```bash
make dev-setup
```

This will:
- Download dependencies
- Create a `config.yaml` from the example

3. **Build the project**

```bash
make build
```

4. **Run the application**

```bash
make run
```

The web UI will be available at http://localhost:8080

## Project Structure

```
arbiter/
├── api/                    # OpenAPI specifications
├── cmd/arbiter/           # Main application entry point
├── internal/              # Internal packages
│   ├── agent/            # Agent management
│   ├── arbiter/          # Core orchestration logic
│   ├── api/              # HTTP API handlers
│   ├── beads/            # Beads integration
│   ├── decision/         # Decision bead handling
│   ├── persona/          # Persona loading and editing
│   ├── project/          # Project management
│   └── web/              # Web UI handlers
├── pkg/                   # Public packages
│   ├── config/           # Configuration
│   └── models/           # Data models
├── personas/              # Persona definitions
│   ├── templates/        # Template personas
│   └── examples/         # Example personas
└── web/static/           # Web UI assets
```

## Development Workflow

### Making Changes

1. **Create a branch**

```bash
git checkout -b feature/your-feature-name
```

2. **File a bead for your work**

See [BEADS_WORKFLOW.md](BEADS_WORKFLOW.md) for detailed instructions.

```bash
# Quick example - create a bead file in .beads/beads/
# bd-XXX-your-feature.yaml
```

All work should have a corresponding bead to enable proper tracking and coordination.

3. **Make your changes**

Follow the coding standards (see below)

4. **Format your code**

```bash
make fmt
make vet
```

5. **Test your changes**

```bash
make test
```

6. **Update your bead**

Mark the bead as complete and move it to `.beads/closed/` when done.

7. **Commit your changes**

```bash
git add .
git commit -m "Brief description of changes"
```

Use clear, descriptive commit messages. Reference the bead ID if applicable.

8. **Push and create a pull request**

```bash
git push origin feature/your-feature-name
```

## Coding Standards

### Go Code

- Follow standard Go conventions
- Use `go fmt` for formatting (run `make fmt`)
- Run `go vet` before committing (run `make vet`)
- Write tests for new functionality
- Keep functions focused and single-purpose
- Add comments for exported functions and types
- Use meaningful variable and function names

### Persona Files

- Use markdown format
- Keep sections organized and consistent
- Provide clear examples
- Document autonomy levels clearly
- Include decision-making guidelines

### API Changes

- Update OpenAPI specification (`api/openapi.yaml`)
- Maintain backward compatibility when possible
- Document breaking changes clearly
- Add examples for new endpoints

### Web UI

- Keep JavaScript simple and readable
- Use vanilla JavaScript (no frameworks required)
- Maintain responsive design
- Test in multiple browsers

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make coverage
```

### Writing Tests

- Place tests in `*_test.go` files
- Use table-driven tests where appropriate
- Test both success and error cases
- Mock external dependencies

Example:

```go
func TestPersonaManager_LoadPersona(t *testing.T) {
    tests := []struct {
        name    string
        persona string
        wantErr bool
    }{
        {"valid persona", "code-reviewer", false},
        {"invalid persona", "nonexistent", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Adding New Features

### Adding a New Persona

1. Create a new directory in `personas/examples/`
2. Add `PERSONA.md` and `AI_START_HERE.md`
3. Follow the template structure
4. Test loading the persona via API

### Adding API Endpoints

1. Update OpenAPI spec: `api/openapi.yaml`
2. Add handler in `internal/api/`
3. Update router in `server.go`
4. Add tests
5. Update web UI if needed

### Adding Agent Capabilities

1. Update models in `pkg/models/`
2. Add logic to appropriate manager
3. Update API handlers
4. Add tests
5. Update documentation

## Documentation

- Update README.md for major features
- Keep OpenAPI spec in sync with code
- Comment complex logic
- Add examples for new personas
- Update this CONTRIBUTING.md if workflow changes

## Pull Request Process

1. **Ensure your PR**:
   - Has a clear description
   - References any related issues
   - Includes tests for new functionality
   - Passes all tests (`make test`)
   - Is formatted correctly (`make fmt`)
   - Has updated documentation

2. **PR Review**:
   - Wait for maintainer review
   - Address feedback promptly
   - Keep discussions professional and constructive

3. **After Approval**:
   - Squash commits if requested
   - Maintainers will merge your PR

## Issue Reporting

### Bug Reports

Include:
- Clear description of the bug
- Steps to reproduce
- Expected vs actual behavior
- Environment (OS, Go version, etc.)
- Relevant logs or error messages

### Feature Requests

Include:
- Clear description of the feature
- Use case / motivation
- Proposed implementation (optional)
- Any breaking changes

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Assume good intentions
- Help others learn and grow

## Questions?

- Open an issue for questions
- Check existing issues and PRs
- Review documentation first

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (see LICENSE file).

## Thank You!

Your contributions help make Arbiter better for everyone. We appreciate your time and effort!
