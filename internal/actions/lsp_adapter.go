package actions

import (
	"context"

	"github.com/jordanhubbard/agenticorp/internal/lsp"
)

// LSPOperator interface for dependency injection
type LSPOperator interface {
	FindReferences(ctx context.Context, file string, line, column int, symbol string) (map[string]interface{}, error)
	GoToDefinition(ctx context.Context, file string, line, column int, symbol string) (map[string]interface{}, error)
	FindImplementations(ctx context.Context, file string, line, column int, symbol string) (map[string]interface{}, error)
}

// LSPServiceAdapter adapts LSPService to actions interface
type LSPServiceAdapter struct {
	service *lsp.LSPService
}

// NewLSPServiceAdapter creates a new adapter
func NewLSPServiceAdapter(projectPath string) (*LSPServiceAdapter, error) {
	service, err := lsp.NewLSPService(projectPath)
	if err != nil {
		return nil, err
	}

	return &LSPServiceAdapter{
		service: service,
	}, nil
}

// FindReferences finds all references to a symbol
func (a *LSPServiceAdapter) FindReferences(ctx context.Context, file string, line, column int, symbol string) (map[string]interface{}, error) {
	req := lsp.FindReferencesRequest{
		File:   file,
		Line:   line,
		Column: column,
		Symbol: symbol,
	}

	locations, err := a.service.FindReferences(ctx, req)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	locationMaps := make([]map[string]interface{}, len(locations))
	for i, loc := range locations {
		locationMaps[i] = map[string]interface{}{
			"file":   loc.File,
			"line":   loc.Line,
			"column": loc.Column,
			"text":   loc.Text,
		}
	}

	return map[string]interface{}{
		"references": locationMaps,
		"count":      len(locations),
	}, nil
}

// GoToDefinition finds the definition of a symbol
func (a *LSPServiceAdapter) GoToDefinition(ctx context.Context, file string, line, column int, symbol string) (map[string]interface{}, error) {
	req := lsp.GoToDefinitionRequest{
		File:   file,
		Line:   line,
		Column: column,
		Symbol: symbol,
	}

	location, err := a.service.GoToDefinition(ctx, req)
	if err != nil {
		return nil, err
	}

	if location == nil {
		return map[string]interface{}{
			"found": false,
		}, nil
	}

	return map[string]interface{}{
		"found":  true,
		"file":   location.File,
		"line":   location.Line,
		"column": location.Column,
		"text":   location.Text,
	}, nil
}

// FindImplementations finds all implementations of an interface/abstract method
func (a *LSPServiceAdapter) FindImplementations(ctx context.Context, file string, line, column int, symbol string) (map[string]interface{}, error) {
	req := lsp.FindImplementationsRequest{
		File:   file,
		Line:   line,
		Column: column,
		Symbol: symbol,
	}

	locations, err := a.service.FindImplementations(ctx, req)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	locationMaps := make([]map[string]interface{}, len(locations))
	for i, loc := range locations {
		locationMaps[i] = map[string]interface{}{
			"file":   loc.File,
			"line":   loc.Line,
			"column": loc.Column,
			"text":   loc.Text,
		}
	}

	return map[string]interface{}{
		"implementations": locationMaps,
		"count":           len(locations),
	}, nil
}

// Close closes the LSP service
func (a *LSPServiceAdapter) Close() error {
	return a.service.Close()
}
