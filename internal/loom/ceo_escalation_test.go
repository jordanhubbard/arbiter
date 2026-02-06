package loom

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/config"
	"github.com/jordanhubbard/loom/pkg/models"
)

func newTestLoom(t *testing.T) (*Loom, string) {
	t.Helper()

	tmp, err := os.MkdirTemp("", "loom-ceo-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create subdirectories for test isolation
	keysDir := tmp + "/keys"
	srcDir := tmp + "/src"
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		t.Fatalf("failed to create keys dir: %v", err)
	}
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{MaxConcurrent: 1, DefaultPersonaPath: "./personas", HeartbeatInterval: 10 * time.Second, FileLockTimeout: 10 * time.Minute},
		Beads:  config.BeadsConfig{BDPath: ""},
		Projects: []config.ProjectConfig{
			{BeadsPath: tmp}, // Use temp directory for gitops
		},
		Temporal: config.TemporalConfig{
			Host: "",
		},
		Database: config.DatabaseConfig{Type: "", Path: ""},
		Git: config.GitConfig{
			ProjectKeyDir: keysDir,
		},
	}

	a, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create loom: %v", err)
	}

	a.GetBeadsManager().SetBeadsPath(tmp)
	return a, tmp
}

func TestCEODecisionApproveClosesParentBead(t *testing.T) {
	a, tmp := newTestLoom(t)
	defer os.RemoveAll(tmp)

	bead, err := a.GetBeadsManager().CreateBead("Test bead", "", models.BeadPriorityP2, "task", "loom")
	if err != nil {
		t.Fatalf("failed to create bead: %v", err)
	}

	decision, err := a.EscalateBeadToCEO(bead.ID, "loop", "agent-1")
	if err != nil {
		t.Fatalf("failed to escalate: %v", err)
	}

	if err := a.MakeDecision(decision.ID, "user-test", "approve", "ok"); err != nil {
		t.Fatalf("failed to make decision: %v", err)
	}

	updated, _ := a.GetBeadsManager().GetBead(bead.ID)
	if updated.Status != models.BeadStatusClosed {
		t.Fatalf("expected bead closed, got %s", updated.Status)
	}
}

func TestCEODecisionDenyReopensAndUnassignsParentBead(t *testing.T) {
	a, tmp := newTestLoom(t)
	defer os.RemoveAll(tmp)

	bead, err := a.GetBeadsManager().CreateBead("Test bead", "", models.BeadPriorityP2, "task", "loom")
	if err != nil {
		t.Fatalf("failed to create bead: %v", err)
	}
	_ = a.GetBeadsManager().UpdateBead(bead.ID, map[string]interface{}{
		"assigned_to": "agent-1",
		"status":      models.BeadStatusInProgress,
	})

	decision, err := a.EscalateBeadToCEO(bead.ID, "loop", "agent-1")
	if err != nil {
		t.Fatalf("failed to escalate: %v", err)
	}

	if err := a.MakeDecision(decision.ID, "user-test", "deny", "no"); err != nil {
		t.Fatalf("failed to make decision: %v", err)
	}

	updated, _ := a.GetBeadsManager().GetBead(bead.ID)
	if updated.Status != models.BeadStatusOpen {
		t.Fatalf("expected bead open, got %s", updated.Status)
	}
	if updated.AssignedTo != "" {
		t.Fatalf("expected bead unassigned, got %q", updated.AssignedTo)
	}
}

func TestCEODecisionNeedsMoreInfoReturnsToPriorOwner(t *testing.T) {
	a, tmp := newTestLoom(t)
	defer os.RemoveAll(tmp)

	bead, err := a.GetBeadsManager().CreateBead("Test bead", "", models.BeadPriorityP2, "task", "loom")
	if err != nil {
		t.Fatalf("failed to create bead: %v", err)
	}

	decision, err := a.EscalateBeadToCEO(bead.ID, "missing info", "agent-42")
	if err != nil {
		t.Fatalf("failed to escalate: %v", err)
	}

	if err := a.MakeDecision(decision.ID, "user-test", "needs_more_info", "gather logs"); err != nil {
		t.Fatalf("failed to make decision: %v", err)
	}

	updated, _ := a.GetBeadsManager().GetBead(bead.ID)
	if updated.Status != models.BeadStatusOpen {
		t.Fatalf("expected bead open, got %s", updated.Status)
	}
	if updated.AssignedTo != "agent-42" {
		t.Fatalf("expected bead assigned to agent-42, got %q", updated.AssignedTo)
	}
	if updated.Context == nil || updated.Context["redispatch_requested"] != "true" {
		t.Fatalf("expected redispatch_requested true")
	}
}

func TestGlobalDispatcherDoesNotPanicWithNoProjects(t *testing.T) {
	a, tmp := newTestLoom(t)
	defer os.RemoveAll(tmp)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, _ = a.GetDispatcher().DispatchOnce(ctx, "")
}
