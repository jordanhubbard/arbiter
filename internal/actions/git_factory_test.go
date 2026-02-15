package actions

import (
	"context"
	"testing"
)

func TestWithProjectID(t *testing.T) {
	ctx := context.Background()
	ctx = WithProjectID(ctx, "proj-123")
	got := ProjectIDFromContext(ctx)
	if got != "proj-123" {
		t.Errorf("expected proj-123, got %s", got)
	}
}

func TestProjectIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := ProjectIDFromContext(ctx)
	if got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}

func TestWithProjectID_Override(t *testing.T) {
	ctx := context.Background()
	ctx = WithProjectID(ctx, "first")
	ctx = WithProjectID(ctx, "second")
	got := ProjectIDFromContext(ctx)
	if got != "second" {
		t.Errorf("expected second, got %s", got)
	}
}
