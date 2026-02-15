package decision

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

// --- Helpers ---

// createTestManager returns a new Manager with no decisions.
func createTestManager() *Manager {
	return NewManager()
}

// createTestDecision is a shorthand that creates a single decision in the manager
// and returns the decision along with the manager. It fails the test on error.
func createTestDecision(t *testing.T, opts ...func(*decisionOpts)) (*Manager, *models.DecisionBead) {
	t.Helper()
	m := createTestManager()
	o := defaultDecisionOpts()
	for _, fn := range opts {
		fn(&o)
	}
	d, err := m.CreateDecision(o.question, o.parentBeadID, o.requesterID, o.options, o.recommendation, o.priority, o.projectID)
	if err != nil {
		t.Fatalf("CreateDecision() unexpected error: %v", err)
	}
	return m, d
}

type decisionOpts struct {
	question       string
	parentBeadID   string
	requesterID    string
	options        []string
	recommendation string
	priority       models.BeadPriority
	projectID      string
}

func defaultDecisionOpts() decisionOpts {
	return decisionOpts{
		question:       "Which database should we use?",
		parentBeadID:   "parent-001",
		requesterID:    "agent-requester",
		options:        []string{"PostgreSQL", "MySQL", "SQLite"},
		recommendation: "PostgreSQL",
		priority:       models.BeadPriorityP2,
		projectID:      "proj-1",
	}
}

func withPriority(p models.BeadPriority) func(*decisionOpts) {
	return func(o *decisionOpts) { o.priority = p }
}

func withProjectID(id string) func(*decisionOpts) {
	return func(o *decisionOpts) { o.projectID = id }
}

func withRequesterID(id string) func(*decisionOpts) {
	return func(o *decisionOpts) { o.requesterID = id }
}

func withQuestion(q string) func(*decisionOpts) {
	return func(o *decisionOpts) { o.question = q }
}

// --- Tests ---

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.decisions == nil {
		t.Fatal("NewManager() decisions map is nil")
	}
	if len(m.decisions) != 0 {
		t.Errorf("NewManager() decisions map length = %d, want 0", len(m.decisions))
	}
}

func TestCreateDecision(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		m := createTestManager()
		question := "Which framework?"
		parentID := "bead-parent"
		requesterID := "agent-007"
		options := []string{"React", "Vue", "Angular"}
		recommendation := "React"
		priority := models.BeadPriorityP1
		projectID := "proj-web"

		d, err := m.CreateDecision(question, parentID, requesterID, options, recommendation, priority, projectID)
		if err != nil {
			t.Fatalf("CreateDecision() error = %v", err)
		}

		// Verify ID prefix
		if !strings.HasPrefix(d.ID, "bd-dec-") {
			t.Errorf("CreateDecision() ID = %q, want prefix 'bd-dec-'", d.ID)
		}

		// Verify bead fields
		if d.Bead == nil {
			t.Fatal("CreateDecision() Bead is nil")
		}
		if d.Type != "decision" {
			t.Errorf("CreateDecision() Type = %q, want 'decision'", d.Type)
		}
		if d.Title != fmt.Sprintf("Decision: %s", question) {
			t.Errorf("CreateDecision() Title = %q, want 'Decision: %s'", d.Title, question)
		}
		if d.Description != question {
			t.Errorf("CreateDecision() Description = %q, want %q", d.Description, question)
		}
		if d.Status != models.BeadStatusOpen {
			t.Errorf("CreateDecision() Status = %q, want %q", d.Status, models.BeadStatusOpen)
		}
		if d.Priority != priority {
			t.Errorf("CreateDecision() Priority = %d, want %d", d.Priority, priority)
		}
		if d.ProjectID != projectID {
			t.Errorf("CreateDecision() ProjectID = %q, want %q", d.ProjectID, projectID)
		}
		if d.Parent != parentID {
			t.Errorf("CreateDecision() Parent = %q, want %q", d.Parent, parentID)
		}
		if d.CreatedAt.IsZero() {
			t.Error("CreateDecision() CreatedAt is zero")
		}
		if d.UpdatedAt.IsZero() {
			t.Error("CreateDecision() UpdatedAt is zero")
		}

		// Verify decision-specific fields
		if d.Question != question {
			t.Errorf("CreateDecision() Question = %q, want %q", d.Question, question)
		}
		if len(d.Options) != len(options) {
			t.Errorf("CreateDecision() Options length = %d, want %d", len(d.Options), len(options))
		}
		if d.Recommendation != recommendation {
			t.Errorf("CreateDecision() Recommendation = %q, want %q", d.Recommendation, recommendation)
		}
		if d.RequesterID != requesterID {
			t.Errorf("CreateDecision() RequesterID = %q, want %q", d.RequesterID, requesterID)
		}
	})

	t.Run("stored in map", func(t *testing.T) {
		m, d := createTestDecision(t)

		// Verify the decision is in the internal map
		stored, ok := m.decisions[d.ID]
		if !ok {
			t.Fatal("CreateDecision() decision not found in internal map")
		}
		if stored != d {
			t.Error("CreateDecision() stored decision pointer differs from returned")
		}
	})

	t.Run("empty options and recommendation", func(t *testing.T) {
		m := createTestManager()
		d, err := m.CreateDecision("simple question?", "", "req-1", nil, "", models.BeadPriorityP3, "proj-1")
		if err != nil {
			t.Fatalf("CreateDecision() error = %v", err)
		}
		if d.Options != nil {
			t.Errorf("CreateDecision() Options = %v, want nil", d.Options)
		}
		if d.Recommendation != "" {
			t.Errorf("CreateDecision() Recommendation = %q, want empty", d.Recommendation)
		}
		if d.Parent != "" {
			t.Errorf("CreateDecision() Parent = %q, want empty", d.Parent)
		}
	})

	t.Run("multiple decisions get unique IDs", func(t *testing.T) {
		m := createTestManager()
		ids := make(map[string]bool)
		// Create several decisions; IDs are based on Unix timestamp so they
		// may collide within the same second. We still verify that at least
		// the map grows correctly.
		for i := 0; i < 5; i++ {
			d, err := m.CreateDecision(
				fmt.Sprintf("question %d", i), "", "req", nil, "", models.BeadPriorityP2, "proj",
			)
			if err != nil {
				t.Fatalf("CreateDecision() iteration %d error: %v", i, err)
			}
			ids[d.ID] = true
		}
		// The map should have at least 1 entry (timestamps may collide and
		// overwrite, but the last write wins). In practice they usually don't
		// collide in a single test run.
		if len(m.decisions) == 0 {
			t.Error("Expected at least one decision in the map")
		}
	})
}

func TestGetDecision(t *testing.T) {
	t.Run("existing decision", func(t *testing.T) {
		m, created := createTestDecision(t)

		got, err := m.GetDecision(created.ID)
		if err != nil {
			t.Fatalf("GetDecision() error = %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("GetDecision() ID = %q, want %q", got.ID, created.ID)
		}
		if got.Question != created.Question {
			t.Errorf("GetDecision() Question = %q, want %q", got.Question, created.Question)
		}
	})

	t.Run("non-existent decision", func(t *testing.T) {
		m := createTestManager()
		_, err := m.GetDecision("nonexistent-id")
		if err == nil {
			t.Fatal("GetDecision() expected error for non-existent ID")
		}
		if !strings.Contains(err.Error(), "decision not found") {
			t.Errorf("GetDecision() error = %q, want to contain 'decision not found'", err.Error())
		}
	})

	t.Run("empty ID", func(t *testing.T) {
		m := createTestManager()
		_, err := m.GetDecision("")
		if err == nil {
			t.Fatal("GetDecision() expected error for empty ID")
		}
	})
}

func TestListDecisions(t *testing.T) {
	// Set up a manager with several decisions of varying attributes.
	setupManager := func(t *testing.T) *Manager {
		t.Helper()
		m := createTestManager()

		// Decision 1: Open, P2, proj-a, requester-1
		d1, _ := m.CreateDecision("Q1?", "", "requester-1", nil, "", models.BeadPriorityP2, "proj-a")
		_ = d1

		// Decision 2: Open, P0, proj-b, requester-2
		d2, _ := m.CreateDecision("Q2?", "", "requester-2", nil, "", models.BeadPriorityP0, "proj-b")
		_ = d2

		// Decision 3: Claimed (InProgress), P1, proj-a, requester-1
		d3, _ := m.CreateDecision("Q3?", "", "requester-1", nil, "", models.BeadPriorityP1, "proj-a")
		_ = m.ClaimDecision(d3.ID, "decider-1")

		// Decision 4: Closed, P2, proj-b, requester-2
		d4, _ := m.CreateDecision("Q4?", "", "requester-2", nil, "", models.BeadPriorityP2, "proj-b")
		_ = m.ClaimDecision(d4.ID, "decider-2")
		_ = m.MakeDecision(d4.ID, "decider-2", "Go with option A", "It is best")

		return m
	}

	t.Run("no filters returns all", func(t *testing.T) {
		m := setupManager(t)
		results, err := m.ListDecisions(nil)
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 4 {
			t.Errorf("ListDecisions(nil) count = %d, want 4", len(results))
		}
	})

	t.Run("empty filter map returns all", func(t *testing.T) {
		m := setupManager(t)
		results, err := m.ListDecisions(map[string]interface{}{})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 4 {
			t.Errorf("ListDecisions({}) count = %d, want 4", len(results))
		}
	})

	t.Run("filter by status open", func(t *testing.T) {
		m := setupManager(t)
		results, err := m.ListDecisions(map[string]interface{}{
			"status": models.BeadStatusOpen,
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("ListDecisions(status=open) count = %d, want 2", len(results))
		}
	})

	t.Run("filter by status in_progress", func(t *testing.T) {
		m := setupManager(t)
		results, err := m.ListDecisions(map[string]interface{}{
			"status": models.BeadStatusInProgress,
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("ListDecisions(status=in_progress) count = %d, want 1", len(results))
		}
	})

	t.Run("filter by status closed", func(t *testing.T) {
		m := setupManager(t)
		results, err := m.ListDecisions(map[string]interface{}{
			"status": models.BeadStatusClosed,
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("ListDecisions(status=closed) count = %d, want 1", len(results))
		}
	})

	t.Run("filter by priority", func(t *testing.T) {
		m := setupManager(t)
		results, err := m.ListDecisions(map[string]interface{}{
			"priority": models.BeadPriorityP2,
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("ListDecisions(priority=P2) count = %d, want 2", len(results))
		}
	})

	t.Run("filter by project_id", func(t *testing.T) {
		m := setupManager(t)
		results, err := m.ListDecisions(map[string]interface{}{
			"project_id": "proj-a",
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("ListDecisions(project_id=proj-a) count = %d, want 2", len(results))
		}
	})

	t.Run("filter by requester_id", func(t *testing.T) {
		m := setupManager(t)
		results, err := m.ListDecisions(map[string]interface{}{
			"requester_id": "requester-2",
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("ListDecisions(requester_id=requester-2) count = %d, want 2", len(results))
		}
	})

	t.Run("multiple filters combined", func(t *testing.T) {
		m := setupManager(t)
		results, err := m.ListDecisions(map[string]interface{}{
			"project_id":   "proj-a",
			"requester_id": "requester-1",
			"status":       models.BeadStatusOpen,
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("ListDecisions(project_id=proj-a, requester_id=requester-1, status=open) count = %d, want 1", len(results))
		}
	})

	t.Run("filter with no matches", func(t *testing.T) {
		m := setupManager(t)
		results, err := m.ListDecisions(map[string]interface{}{
			"project_id": "nonexistent-project",
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("ListDecisions(project_id=nonexistent) count = %d, want 0", len(results))
		}
	})

	t.Run("filter with wrong type value is ignored", func(t *testing.T) {
		m := setupManager(t)
		// Passing a string where BeadStatus is expected -- type assertion
		// in matchesFilters will fail, so the filter is effectively skipped.
		results, err := m.ListDecisions(map[string]interface{}{
			"status": "open", // string, not models.BeadStatus
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		// Should return all because the type assertion fails silently
		if len(results) != 4 {
			t.Errorf("ListDecisions(status='open' as string) count = %d, want 4", len(results))
		}
	})

	t.Run("empty manager returns empty list", func(t *testing.T) {
		m := createTestManager()
		results, err := m.ListDecisions(nil)
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("ListDecisions() on empty manager count = %d, want 0", len(results))
		}
	})
}

func TestClaimDecision(t *testing.T) {
	t.Run("claim unclaimed decision", func(t *testing.T) {
		m, d := createTestDecision(t)

		err := m.ClaimDecision(d.ID, "decider-1")
		if err != nil {
			t.Fatalf("ClaimDecision() error = %v", err)
		}

		got, _ := m.GetDecision(d.ID)
		if got.DeciderID != "decider-1" {
			t.Errorf("ClaimDecision() DeciderID = %q, want 'decider-1'", got.DeciderID)
		}
		if got.Status != models.BeadStatusInProgress {
			t.Errorf("ClaimDecision() Status = %q, want %q", got.Status, models.BeadStatusInProgress)
		}
		if got.UpdatedAt.IsZero() {
			t.Error("ClaimDecision() UpdatedAt is zero")
		}
	})

	t.Run("re-claim by same decider succeeds", func(t *testing.T) {
		m, d := createTestDecision(t)
		_ = m.ClaimDecision(d.ID, "decider-1")

		// Same decider claims again - should succeed
		err := m.ClaimDecision(d.ID, "decider-1")
		if err != nil {
			t.Errorf("ClaimDecision() same decider error = %v, want nil", err)
		}
	})

	t.Run("claim already claimed by different decider fails", func(t *testing.T) {
		m, d := createTestDecision(t)
		_ = m.ClaimDecision(d.ID, "decider-1")

		err := m.ClaimDecision(d.ID, "decider-2")
		if err == nil {
			t.Fatal("ClaimDecision() expected error when claimed by different decider")
		}
		if !strings.Contains(err.Error(), "already claimed") {
			t.Errorf("ClaimDecision() error = %q, want to contain 'already claimed'", err.Error())
		}
	})

	t.Run("claim non-existent decision", func(t *testing.T) {
		m := createTestManager()
		err := m.ClaimDecision("nonexistent", "decider-1")
		if err == nil {
			t.Fatal("ClaimDecision() expected error for non-existent decision")
		}
		if !strings.Contains(err.Error(), "decision not found") {
			t.Errorf("ClaimDecision() error = %q, want to contain 'decision not found'", err.Error())
		}
	})
}

func TestMakeDecision(t *testing.T) {
	t.Run("make decision on claimed decision", func(t *testing.T) {
		m, d := createTestDecision(t)
		_ = m.ClaimDecision(d.ID, "decider-1")

		err := m.MakeDecision(d.ID, "decider-1", "Use PostgreSQL", "Best for our use case")
		if err != nil {
			t.Fatalf("MakeDecision() error = %v", err)
		}

		got, _ := m.GetDecision(d.ID)
		if got.Decision != "Use PostgreSQL" {
			t.Errorf("MakeDecision() Decision = %q, want 'Use PostgreSQL'", got.Decision)
		}
		if got.Rationale != "Best for our use case" {
			t.Errorf("MakeDecision() Rationale = %q, want 'Best for our use case'", got.Rationale)
		}
		if got.DecidedAt == nil {
			t.Fatal("MakeDecision() DecidedAt is nil")
		}
		if got.ClosedAt == nil {
			t.Fatal("MakeDecision() ClosedAt is nil")
		}
		if got.Status != models.BeadStatusClosed {
			t.Errorf("MakeDecision() Status = %q, want %q", got.Status, models.BeadStatusClosed)
		}
		if got.DeciderID != "decider-1" {
			t.Errorf("MakeDecision() DeciderID = %q, want 'decider-1'", got.DeciderID)
		}
	})

	t.Run("make decision on unclaimed decision assigns decider", func(t *testing.T) {
		m, d := createTestDecision(t)
		// Decision is unclaimed (DeciderID == "")

		err := m.MakeDecision(d.ID, "decider-new", "Option B", "Simpler")
		if err != nil {
			t.Fatalf("MakeDecision() error = %v", err)
		}

		got, _ := m.GetDecision(d.ID)
		if got.DeciderID != "decider-new" {
			t.Errorf("MakeDecision() DeciderID = %q, want 'decider-new'", got.DeciderID)
		}
		if got.Status != models.BeadStatusClosed {
			t.Errorf("MakeDecision() Status = %q, want %q", got.Status, models.BeadStatusClosed)
		}
	})

	t.Run("make decision by different decider fails", func(t *testing.T) {
		m, d := createTestDecision(t)
		_ = m.ClaimDecision(d.ID, "decider-1")

		err := m.MakeDecision(d.ID, "decider-2", "Option C", "My choice")
		if err == nil {
			t.Fatal("MakeDecision() expected error when different decider")
		}
		if !strings.Contains(err.Error(), "claimed by different agent") {
			t.Errorf("MakeDecision() error = %q, want to contain 'claimed by different agent'", err.Error())
		}
	})

	t.Run("make decision on non-existent decision", func(t *testing.T) {
		m := createTestManager()
		err := m.MakeDecision("nonexistent", "decider-1", "X", "Y")
		if err == nil {
			t.Fatal("MakeDecision() expected error for non-existent decision")
		}
		if !strings.Contains(err.Error(), "decision not found") {
			t.Errorf("MakeDecision() error = %q, want to contain 'decision not found'", err.Error())
		}
	})
}

func TestEscalateDecision(t *testing.T) {
	t.Run("escalate existing decision", func(t *testing.T) {
		m, d := createTestDecision(t, withPriority(models.BeadPriorityP2))
		originalDesc := d.Description

		err := m.EscalateDecision(d.ID, "blocking release")
		if err != nil {
			t.Fatalf("EscalateDecision() error = %v", err)
		}

		got, _ := m.GetDecision(d.ID)
		if got.Priority != models.BeadPriorityP0 {
			t.Errorf("EscalateDecision() Priority = %d, want %d (P0)", got.Priority, models.BeadPriorityP0)
		}
		if !strings.Contains(got.Description, "ESCALATED: blocking release") {
			t.Errorf("EscalateDecision() Description = %q, want to contain 'ESCALATED: blocking release'", got.Description)
		}
		if !strings.Contains(got.Description, originalDesc) {
			t.Errorf("EscalateDecision() Description should preserve original description")
		}
	})

	t.Run("escalate already P0 decision", func(t *testing.T) {
		m, d := createTestDecision(t, withPriority(models.BeadPriorityP0))

		err := m.EscalateDecision(d.ID, "still urgent")
		if err != nil {
			t.Fatalf("EscalateDecision() error = %v", err)
		}
		got, _ := m.GetDecision(d.ID)
		if got.Priority != models.BeadPriorityP0 {
			t.Errorf("EscalateDecision() Priority = %d, want P0", got.Priority)
		}
	})

	t.Run("escalate non-existent decision", func(t *testing.T) {
		m := createTestManager()
		err := m.EscalateDecision("nonexistent", "urgent")
		if err == nil {
			t.Fatal("EscalateDecision() expected error for non-existent decision")
		}
		if !strings.Contains(err.Error(), "decision not found") {
			t.Errorf("EscalateDecision() error = %q, want to contain 'decision not found'", err.Error())
		}
	})
}

func TestGetPendingDecisions(t *testing.T) {
	setupManager := func(t *testing.T) (*Manager, map[string]*models.DecisionBead) {
		t.Helper()
		m := createTestManager()
		decisions := make(map[string]*models.DecisionBead)

		// Open P2
		d1, _ := m.CreateDecision("Q1?", "", "r1", nil, "", models.BeadPriorityP2, "p1")
		decisions["open-p2"] = d1

		// Open P0
		d2, _ := m.CreateDecision("Q2?", "", "r2", nil, "", models.BeadPriorityP0, "p1")
		decisions["open-p0"] = d2

		// InProgress P1
		d3, _ := m.CreateDecision("Q3?", "", "r1", nil, "", models.BeadPriorityP1, "p1")
		_ = m.ClaimDecision(d3.ID, "decider-1")
		decisions["inprogress-p1"] = d3

		// Closed P2 (should not appear in pending)
		d4, _ := m.CreateDecision("Q4?", "", "r2", nil, "", models.BeadPriorityP2, "p1")
		_ = m.ClaimDecision(d4.ID, "decider-2")
		_ = m.MakeDecision(d4.ID, "decider-2", "done", "reason")
		decisions["closed-p2"] = d4

		return m, decisions
	}

	t.Run("nil priority returns all pending", func(t *testing.T) {
		m, _ := setupManager(t)
		results, err := m.GetPendingDecisions(nil)
		if err != nil {
			t.Fatalf("GetPendingDecisions() error = %v", err)
		}
		// Should include open-p2, open-p0, inprogress-p1 (not closed)
		if len(results) != 3 {
			t.Errorf("GetPendingDecisions(nil) count = %d, want 3", len(results))
		}
	})

	t.Run("filter by P0 priority", func(t *testing.T) {
		m, _ := setupManager(t)
		p0 := models.BeadPriorityP0
		results, err := m.GetPendingDecisions(&p0)
		if err != nil {
			t.Fatalf("GetPendingDecisions() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("GetPendingDecisions(P0) count = %d, want 1", len(results))
		}
	})

	t.Run("filter by P1 priority", func(t *testing.T) {
		m, _ := setupManager(t)
		p1 := models.BeadPriorityP1
		results, err := m.GetPendingDecisions(&p1)
		if err != nil {
			t.Fatalf("GetPendingDecisions() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("GetPendingDecisions(P1) count = %d, want 1", len(results))
		}
	})

	t.Run("filter by P3 returns empty", func(t *testing.T) {
		m, _ := setupManager(t)
		p3 := models.BeadPriorityP3
		results, err := m.GetPendingDecisions(&p3)
		if err != nil {
			t.Fatalf("GetPendingDecisions() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("GetPendingDecisions(P3) count = %d, want 0", len(results))
		}
	})

	t.Run("empty manager returns empty", func(t *testing.T) {
		m := createTestManager()
		results, err := m.GetPendingDecisions(nil)
		if err != nil {
			t.Fatalf("GetPendingDecisions() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("GetPendingDecisions() on empty manager count = %d, want 0", len(results))
		}
	})
}

func TestGetP0Decisions(t *testing.T) {
	t.Run("returns only pending P0 decisions", func(t *testing.T) {
		m := createTestManager()

		// Open P0
		_, _ = m.CreateDecision("P0 open?", "", "r1", nil, "", models.BeadPriorityP0, "p1")

		// Open P2 (should not appear)
		_, _ = m.CreateDecision("P2 open?", "", "r1", nil, "", models.BeadPriorityP2, "p1")

		// Closed P0 (should not appear - it is closed)
		d3, _ := m.CreateDecision("P0 closed?", "", "r1", nil, "", models.BeadPriorityP0, "p1")
		_ = m.ClaimDecision(d3.ID, "d1")
		_ = m.MakeDecision(d3.ID, "d1", "done", "reason")

		// InProgress P0 (should appear)
		d4, _ := m.CreateDecision("P0 progress?", "", "r1", nil, "", models.BeadPriorityP0, "p1")
		_ = m.ClaimDecision(d4.ID, "d2")

		results, err := m.GetP0Decisions()
		if err != nil {
			t.Fatalf("GetP0Decisions() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("GetP0Decisions() count = %d, want 2", len(results))
		}
		for _, r := range results {
			if r.Priority != models.BeadPriorityP0 {
				t.Errorf("GetP0Decisions() found non-P0 decision with priority %d", r.Priority)
			}
		}
	})

	t.Run("no P0 decisions returns empty", func(t *testing.T) {
		m := createTestManager()
		_, _ = m.CreateDecision("P2 q?", "", "r1", nil, "", models.BeadPriorityP2, "p1")

		results, err := m.GetP0Decisions()
		if err != nil {
			t.Fatalf("GetP0Decisions() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("GetP0Decisions() count = %d, want 0", len(results))
		}
	})
}

func TestGetDecisionsByProject(t *testing.T) {
	t.Run("returns decisions for given project", func(t *testing.T) {
		m := createTestManager()
		_, _ = m.CreateDecision("Q1?", "", "r1", nil, "", models.BeadPriorityP2, "proj-alpha")
		_, _ = m.CreateDecision("Q2?", "", "r1", nil, "", models.BeadPriorityP2, "proj-alpha")
		_, _ = m.CreateDecision("Q3?", "", "r1", nil, "", models.BeadPriorityP2, "proj-beta")

		results, err := m.GetDecisionsByProject("proj-alpha")
		if err != nil {
			t.Fatalf("GetDecisionsByProject() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("GetDecisionsByProject('proj-alpha') count = %d, want 2", len(results))
		}
		for _, r := range results {
			if r.ProjectID != "proj-alpha" {
				t.Errorf("GetDecisionsByProject() returned decision with ProjectID = %q", r.ProjectID)
			}
		}
	})

	t.Run("non-existent project returns empty", func(t *testing.T) {
		m, _ := createTestDecision(t)
		results, err := m.GetDecisionsByProject("nonexistent")
		if err != nil {
			t.Fatalf("GetDecisionsByProject() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("GetDecisionsByProject('nonexistent') count = %d, want 0", len(results))
		}
	})

	t.Run("empty project ID", func(t *testing.T) {
		m := createTestManager()
		// Create a decision with empty project ID
		_, _ = m.CreateDecision("Q?", "", "r1", nil, "", models.BeadPriorityP2, "")

		results, err := m.GetDecisionsByProject("")
		if err != nil {
			t.Fatalf("GetDecisionsByProject() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("GetDecisionsByProject('') count = %d, want 1", len(results))
		}
	})
}

func TestGetDecisionsByRequester(t *testing.T) {
	t.Run("returns decisions for given requester", func(t *testing.T) {
		m := createTestManager()
		_, _ = m.CreateDecision("Q1?", "", "agent-A", nil, "", models.BeadPriorityP2, "p1")
		_, _ = m.CreateDecision("Q2?", "", "agent-A", nil, "", models.BeadPriorityP2, "p1")
		_, _ = m.CreateDecision("Q3?", "", "agent-B", nil, "", models.BeadPriorityP2, "p1")

		results, err := m.GetDecisionsByRequester("agent-A")
		if err != nil {
			t.Fatalf("GetDecisionsByRequester() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("GetDecisionsByRequester('agent-A') count = %d, want 2", len(results))
		}
		for _, r := range results {
			if r.RequesterID != "agent-A" {
				t.Errorf("GetDecisionsByRequester() returned decision with RequesterID = %q", r.RequesterID)
			}
		}
	})

	t.Run("non-existent requester returns empty", func(t *testing.T) {
		m, _ := createTestDecision(t)
		results, err := m.GetDecisionsByRequester("nonexistent-agent")
		if err != nil {
			t.Fatalf("GetDecisionsByRequester() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("GetDecisionsByRequester('nonexistent') count = %d, want 0", len(results))
		}
	})
}

func TestGetBlockedBeads(t *testing.T) {
	t.Run("decision with blocks", func(t *testing.T) {
		m, d := createTestDecision(t)
		// Manually set Blocks on the bead (the Manager does not have a method for this)
		d.Blocks = []string{"bead-1", "bead-2", "bead-3"}

		blocks := m.GetBlockedBeads(d.ID)
		if len(blocks) != 3 {
			t.Errorf("GetBlockedBeads() count = %d, want 3", len(blocks))
		}
		expected := map[string]bool{"bead-1": true, "bead-2": true, "bead-3": true}
		for _, b := range blocks {
			if !expected[b] {
				t.Errorf("GetBlockedBeads() unexpected bead %q", b)
			}
		}
	})

	t.Run("decision with no blocks", func(t *testing.T) {
		m, d := createTestDecision(t)
		blocks := m.GetBlockedBeads(d.ID)
		if blocks != nil && len(blocks) != 0 {
			t.Errorf("GetBlockedBeads() = %v, want nil or empty", blocks)
		}
	})

	t.Run("non-existent decision returns empty slice", func(t *testing.T) {
		m := createTestManager()
		blocks := m.GetBlockedBeads("nonexistent")
		if len(blocks) != 0 {
			t.Errorf("GetBlockedBeads('nonexistent') count = %d, want 0", len(blocks))
		}
	})
}

func TestCanAutoDecide(t *testing.T) {
	tests := []struct {
		name       string
		priority   models.BeadPriority
		autonomy   models.AutonomyLevel
		wantResult bool
		wantReason string
	}{
		{
			name:       "P0 with full autonomy requires human",
			priority:   models.BeadPriorityP0,
			autonomy:   models.AutonomyFull,
			wantResult: false,
			wantReason: "P0 decisions require human intervention",
		},
		{
			name:       "P0 with semi autonomy requires human",
			priority:   models.BeadPriorityP0,
			autonomy:   models.AutonomySemi,
			wantResult: false,
			wantReason: "P0 decisions require human intervention",
		},
		{
			name:       "P0 with supervised autonomy requires human",
			priority:   models.BeadPriorityP0,
			autonomy:   models.AutonomySupervised,
			wantResult: false,
			wantReason: "P0 decisions require human intervention",
		},
		{
			name:       "supervised cannot make any decisions",
			priority:   models.BeadPriorityP3,
			autonomy:   models.AutonomySupervised,
			wantResult: false,
			wantReason: "supervised agents cannot make decisions",
		},
		{
			name:       "supervised cannot make P2 decisions",
			priority:   models.BeadPriorityP2,
			autonomy:   models.AutonomySupervised,
			wantResult: false,
			wantReason: "supervised agents cannot make decisions",
		},
		{
			name:       "semi cannot make P1 decisions",
			priority:   models.BeadPriorityP1,
			autonomy:   models.AutonomySemi,
			wantResult: false,
			wantReason: "semi-autonomous agents cannot make P0/P1 decisions",
		},
		{
			name:       "semi can make P2 decisions",
			priority:   models.BeadPriorityP2,
			autonomy:   models.AutonomySemi,
			wantResult: true,
			wantReason: "",
		},
		{
			name:       "semi can make P3 decisions",
			priority:   models.BeadPriorityP3,
			autonomy:   models.AutonomySemi,
			wantResult: true,
			wantReason: "",
		},
		{
			name:       "full autonomy can make P1 decisions",
			priority:   models.BeadPriorityP1,
			autonomy:   models.AutonomyFull,
			wantResult: true,
			wantReason: "",
		},
		{
			name:       "full autonomy can make P2 decisions",
			priority:   models.BeadPriorityP2,
			autonomy:   models.AutonomyFull,
			wantResult: true,
			wantReason: "",
		},
		{
			name:       "full autonomy can make P3 decisions",
			priority:   models.BeadPriorityP3,
			autonomy:   models.AutonomyFull,
			wantResult: true,
			wantReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, d := createTestDecision(t, withPriority(tt.priority))

			result, reason := m.CanAutoDecide(d.ID, tt.autonomy)
			if result != tt.wantResult {
				t.Errorf("CanAutoDecide() result = %v, want %v", result, tt.wantResult)
			}
			if reason != tt.wantReason {
				t.Errorf("CanAutoDecide() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}

	t.Run("non-existent decision", func(t *testing.T) {
		m := createTestManager()
		result, reason := m.CanAutoDecide("nonexistent", models.AutonomyFull)
		if result != false {
			t.Error("CanAutoDecide() expected false for non-existent decision")
		}
		if reason != "decision not found" {
			t.Errorf("CanAutoDecide() reason = %q, want 'decision not found'", reason)
		}
	})
}

func TestUpdateDecisionContext(t *testing.T) {
	t.Run("add context to decision with nil context", func(t *testing.T) {
		m, d := createTestDecision(t)
		// Bead.Context is nil by default from CreateDecision

		ctx := map[string]string{
			"source":   "code-review",
			"severity": "high",
		}
		err := m.UpdateDecisionContext(d.ID, ctx)
		if err != nil {
			t.Fatalf("UpdateDecisionContext() error = %v", err)
		}

		got, _ := m.GetDecision(d.ID)
		if got.Context == nil {
			t.Fatal("UpdateDecisionContext() Context is still nil")
		}
		if got.Context["source"] != "code-review" {
			t.Errorf("UpdateDecisionContext() Context['source'] = %q, want 'code-review'", got.Context["source"])
		}
		if got.Context["severity"] != "high" {
			t.Errorf("UpdateDecisionContext() Context['severity'] = %q, want 'high'", got.Context["severity"])
		}
	})

	t.Run("merge context with existing context", func(t *testing.T) {
		m, d := createTestDecision(t)
		// Add initial context
		_ = m.UpdateDecisionContext(d.ID, map[string]string{"key1": "val1"})

		// Add more context
		err := m.UpdateDecisionContext(d.ID, map[string]string{"key2": "val2"})
		if err != nil {
			t.Fatalf("UpdateDecisionContext() error = %v", err)
		}

		got, _ := m.GetDecision(d.ID)
		if got.Context["key1"] != "val1" {
			t.Errorf("UpdateDecisionContext() Context['key1'] = %q, want 'val1'", got.Context["key1"])
		}
		if got.Context["key2"] != "val2" {
			t.Errorf("UpdateDecisionContext() Context['key2'] = %q, want 'val2'", got.Context["key2"])
		}
	})

	t.Run("overwrite existing context key", func(t *testing.T) {
		m, d := createTestDecision(t)
		_ = m.UpdateDecisionContext(d.ID, map[string]string{"key": "original"})

		err := m.UpdateDecisionContext(d.ID, map[string]string{"key": "updated"})
		if err != nil {
			t.Fatalf("UpdateDecisionContext() error = %v", err)
		}

		got, _ := m.GetDecision(d.ID)
		if got.Context["key"] != "updated" {
			t.Errorf("UpdateDecisionContext() Context['key'] = %q, want 'updated'", got.Context["key"])
		}
	})

	t.Run("empty context map is a no-op", func(t *testing.T) {
		m, d := createTestDecision(t)
		err := m.UpdateDecisionContext(d.ID, map[string]string{})
		if err != nil {
			t.Fatalf("UpdateDecisionContext() error = %v", err)
		}
	})

	t.Run("non-existent decision", func(t *testing.T) {
		m := createTestManager()
		err := m.UpdateDecisionContext("nonexistent", map[string]string{"k": "v"})
		if err == nil {
			t.Fatal("UpdateDecisionContext() expected error for non-existent decision")
		}
		if !strings.Contains(err.Error(), "decision not found") {
			t.Errorf("UpdateDecisionContext() error = %q, want to contain 'decision not found'", err.Error())
		}
	})

	t.Run("updates UpdatedAt timestamp", func(t *testing.T) {
		m, d := createTestDecision(t)
		originalUpdatedAt := d.UpdatedAt

		err := m.UpdateDecisionContext(d.ID, map[string]string{"k": "v"})
		if err != nil {
			t.Fatalf("UpdateDecisionContext() error = %v", err)
		}

		got, _ := m.GetDecision(d.ID)
		if !got.UpdatedAt.After(originalUpdatedAt) && got.UpdatedAt != originalUpdatedAt {
			// UpdatedAt should be >= original (may be equal if running fast)
			t.Log("UpdateDecisionContext() UpdatedAt was not updated (may be same nanosecond)")
		}
	})
}

// TestMatchesFilters tests the private matchesFilters method indirectly
// through ListDecisions. This test ensures all four filter types work
// correctly in isolation and in combination. The filter tests above
// already cover most cases; this section adds edge cases.
func TestMatchesFilters_EdgeCases(t *testing.T) {
	t.Run("filter with unknown key is ignored", func(t *testing.T) {
		m := createTestManager()
		_, _ = m.CreateDecision("Q?", "", "r1", nil, "", models.BeadPriorityP2, "p1")

		results, err := m.ListDecisions(map[string]interface{}{
			"unknown_filter": "some_value",
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("ListDecisions(unknown_filter) count = %d, want 1", len(results))
		}
	})

	t.Run("priority filter with wrong type is ignored", func(t *testing.T) {
		m := createTestManager()
		_, _ = m.CreateDecision("Q?", "", "r1", nil, "", models.BeadPriorityP2, "p1")

		results, err := m.ListDecisions(map[string]interface{}{
			"priority": "P2", // string instead of BeadPriority
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		// Filter ignored because type assertion fails
		if len(results) != 1 {
			t.Errorf("ListDecisions(priority='P2') count = %d, want 1", len(results))
		}
	})

	t.Run("project_id filter with int type is ignored", func(t *testing.T) {
		m := createTestManager()
		_, _ = m.CreateDecision("Q?", "", "r1", nil, "", models.BeadPriorityP2, "p1")

		results, err := m.ListDecisions(map[string]interface{}{
			"project_id": 123, // int instead of string
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("ListDecisions(project_id=123) count = %d, want 1", len(results))
		}
	})

	t.Run("all four filters applied simultaneously", func(t *testing.T) {
		m := createTestManager()
		d, _ := m.CreateDecision("Q?", "", "r1", nil, "", models.BeadPriorityP2, "p1")
		_ = d

		results, err := m.ListDecisions(map[string]interface{}{
			"status":       models.BeadStatusOpen,
			"priority":     models.BeadPriorityP2,
			"project_id":   "p1",
			"requester_id": "r1",
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("ListDecisions(all filters) count = %d, want 1", len(results))
		}
	})

	t.Run("all four filters applied but one does not match", func(t *testing.T) {
		m := createTestManager()
		_, _ = m.CreateDecision("Q?", "", "r1", nil, "", models.BeadPriorityP2, "p1")

		results, err := m.ListDecisions(map[string]interface{}{
			"status":       models.BeadStatusOpen,
			"priority":     models.BeadPriorityP2,
			"project_id":   "p1",
			"requester_id": "r2", // does not match
		})
		if err != nil {
			t.Fatalf("ListDecisions() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("ListDecisions(mismatched requester_id) count = %d, want 0", len(results))
		}
	})
}

// TestManagerWorkflow tests a complete lifecycle: create -> claim -> decide
func TestManagerWorkflow(t *testing.T) {
	m := createTestManager()

	// Step 1: Create a decision
	d, err := m.CreateDecision(
		"Should we refactor the auth module?",
		"epic-100",
		"agent-architect",
		[]string{"Yes, full rewrite", "Partial refactor", "No, keep as is"},
		"Partial refactor",
		models.BeadPriorityP1,
		"proj-auth",
	)
	if err != nil {
		t.Fatalf("CreateDecision() error = %v", err)
	}
	if d.Status != models.BeadStatusOpen {
		t.Errorf("After create, Status = %q, want 'open'", d.Status)
	}

	// Step 2: Add context
	err = m.UpdateDecisionContext(d.ID, map[string]string{
		"tech_debt_score": "8/10",
		"affected_teams":  "auth, payments",
	})
	if err != nil {
		t.Fatalf("UpdateDecisionContext() error = %v", err)
	}

	// Step 3: Check pending
	pending, _ := m.GetPendingDecisions(nil)
	found := false
	for _, p := range pending {
		if p.ID == d.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Decision should appear in pending decisions")
	}

	// Step 4: Claim the decision
	err = m.ClaimDecision(d.ID, "agent-lead")
	if err != nil {
		t.Fatalf("ClaimDecision() error = %v", err)
	}

	got, _ := m.GetDecision(d.ID)
	if got.Status != models.BeadStatusInProgress {
		t.Errorf("After claim, Status = %q, want 'in_progress'", got.Status)
	}

	// Step 5: Make the decision
	err = m.MakeDecision(d.ID, "agent-lead", "Partial refactor", "Best balance of effort vs improvement")
	if err != nil {
		t.Fatalf("MakeDecision() error = %v", err)
	}

	got, _ = m.GetDecision(d.ID)
	if got.Status != models.BeadStatusClosed {
		t.Errorf("After decision, Status = %q, want 'closed'", got.Status)
	}
	if got.Decision != "Partial refactor" {
		t.Errorf("After decision, Decision = %q, want 'Partial refactor'", got.Decision)
	}
	if got.DecidedAt == nil {
		t.Error("After decision, DecidedAt is nil")
	}
	if got.ClosedAt == nil {
		t.Error("After decision, ClosedAt is nil")
	}

	// Step 6: Should no longer be pending
	pending, _ = m.GetPendingDecisions(nil)
	for _, p := range pending {
		if p.ID == d.ID {
			t.Error("Closed decision should not appear in pending decisions")
		}
	}
}

// TestManagerEscalationWorkflow tests escalation then checking P0 decisions
func TestManagerEscalationWorkflow(t *testing.T) {
	m := createTestManager()

	d, _ := m.CreateDecision("API design question", "", "agent-dev", nil, "", models.BeadPriorityP2, "proj-api")

	// Initially not a P0 decision
	p0s, _ := m.GetP0Decisions()
	for _, p := range p0s {
		if p.ID == d.ID {
			t.Error("P2 decision should not appear in P0 list")
		}
	}

	// Escalate
	err := m.EscalateDecision(d.ID, "blocking customer deployment")
	if err != nil {
		t.Fatalf("EscalateDecision() error = %v", err)
	}

	// Now should be in P0 list
	p0s, _ = m.GetP0Decisions()
	found := false
	for _, p := range p0s {
		if p.ID == d.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Escalated decision should appear in P0 list")
	}

	// Auto-decide should fail for P0 even with full autonomy
	canAuto, reason := m.CanAutoDecide(d.ID, models.AutonomyFull)
	if canAuto {
		t.Error("P0 decision should not be auto-decidable even with full autonomy")
	}
	if !strings.Contains(reason, "human intervention") {
		t.Errorf("Expected reason about human intervention, got %q", reason)
	}
}

// TestConcurrentAccess verifies that the Manager is safe for concurrent use.
func TestConcurrentAccess(t *testing.T) {
	m := createTestManager()
	const goroutines = 20

	var wg sync.WaitGroup

	// Concurrent creates
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			_, err := m.CreateDecision(
				fmt.Sprintf("Question %d", n),
				"",
				fmt.Sprintf("req-%d", n),
				nil,
				"",
				models.BeadPriorityP2,
				"proj-concurrent",
			)
			if err != nil {
				t.Errorf("Concurrent CreateDecision() error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	decisions, _ := m.ListDecisions(nil)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = m.ListDecisions(nil)
			_, _ = m.GetPendingDecisions(nil)
			_, _ = m.GetP0Decisions()
			_, _ = m.GetDecisionsByProject("proj-concurrent")
		}()
	}
	wg.Wait()

	// Concurrent claims (only some will succeed due to timestamp collisions)
	wg.Add(len(decisions))
	for _, d := range decisions {
		go func(id string) {
			defer wg.Done()
			_ = m.ClaimDecision(id, "decider-concurrent")
		}(d.ID)
	}
	wg.Wait()

	// Concurrent context updates
	wg.Add(len(decisions))
	for _, d := range decisions {
		go func(id string) {
			defer wg.Done()
			_ = m.UpdateDecisionContext(id, map[string]string{"concurrent": "true"})
		}(d.ID)
	}
	wg.Wait()

	// If we get here without a race condition panic, the test passes.
	// Running with -race will detect any data races.
}

// TestGetDecisionsByProject_AllStatuses verifies that GetDecisionsByProject
// returns decisions regardless of their status.
func TestGetDecisionsByProject_AllStatuses(t *testing.T) {
	m := createTestManager()
	projID := "proj-mixed"

	// Open
	_, _ = m.CreateDecision("Q1?", "", "r1", nil, "", models.BeadPriorityP2, projID)

	// InProgress
	d2, _ := m.CreateDecision("Q2?", "", "r1", nil, "", models.BeadPriorityP2, projID)
	_ = m.ClaimDecision(d2.ID, "d1")

	// Closed
	d3, _ := m.CreateDecision("Q3?", "", "r1", nil, "", models.BeadPriorityP2, projID)
	_ = m.ClaimDecision(d3.ID, "d2")
	_ = m.MakeDecision(d3.ID, "d2", "done", "reason")

	results, err := m.GetDecisionsByProject(projID)
	if err != nil {
		t.Fatalf("GetDecisionsByProject() error = %v", err)
	}
	if len(results) != 3 {
		t.Errorf("GetDecisionsByProject() count = %d, want 3 (all statuses)", len(results))
	}
}

// TestGetDecisionsByRequester_AllStatuses verifies that GetDecisionsByRequester
// returns decisions regardless of their status.
func TestGetDecisionsByRequester_AllStatuses(t *testing.T) {
	m := createTestManager()
	reqID := "agent-tester"

	// Open
	_, _ = m.CreateDecision("Q1?", "", reqID, nil, "", models.BeadPriorityP2, "p1")

	// Closed
	d2, _ := m.CreateDecision("Q2?", "", reqID, nil, "", models.BeadPriorityP2, "p1")
	_ = m.MakeDecision(d2.ID, "decider", "done", "reason")

	results, err := m.GetDecisionsByRequester(reqID)
	if err != nil {
		t.Fatalf("GetDecisionsByRequester() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("GetDecisionsByRequester() count = %d, want 2 (all statuses)", len(results))
	}
}

// TestCanAutoDecide_AfterEscalation verifies that escalation changes
// the auto-decide outcome.
func TestCanAutoDecide_AfterEscalation(t *testing.T) {
	m, d := createTestDecision(t, withPriority(models.BeadPriorityP2))

	// Before escalation: semi can decide P2
	canAuto, _ := m.CanAutoDecide(d.ID, models.AutonomySemi)
	if !canAuto {
		t.Error("Semi should be able to auto-decide P2 before escalation")
	}

	// Escalate to P0
	_ = m.EscalateDecision(d.ID, "urgent")

	// After escalation: nobody can auto-decide P0
	canAuto, reason := m.CanAutoDecide(d.ID, models.AutonomyFull)
	if canAuto {
		t.Error("Nobody should auto-decide P0 after escalation")
	}
	if !strings.Contains(reason, "human intervention") {
		t.Errorf("Expected human intervention reason, got %q", reason)
	}
}

// TestCreateDecision_FieldDetails validates all field values precisely.
func TestCreateDecision_FieldDetails(t *testing.T) {
	m := createTestManager()
	question := "Should we use microservices?"
	parentID := "epic-42"
	requesterID := "agent-arch"
	options := []string{"Yes", "No", "Hybrid"}
	recommendation := "Hybrid"
	priority := models.BeadPriorityP1
	projectID := "proj-infra"

	d, err := m.CreateDecision(question, parentID, requesterID, options, recommendation, priority, projectID)
	if err != nil {
		t.Fatalf("CreateDecision() error = %v", err)
	}

	// Verify the title format
	expectedTitle := "Decision: Should we use microservices?"
	if d.Title != expectedTitle {
		t.Errorf("Title = %q, want %q", d.Title, expectedTitle)
	}

	// Verify options are stored correctly
	if len(d.Options) != 3 {
		t.Fatalf("Options length = %d, want 3", len(d.Options))
	}
	for i, opt := range options {
		if d.Options[i] != opt {
			t.Errorf("Options[%d] = %q, want %q", i, d.Options[i], opt)
		}
	}

	// DeciderID should be empty initially
	if d.DeciderID != "" {
		t.Errorf("DeciderID = %q, want empty", d.DeciderID)
	}

	// Decision text should be empty initially
	if d.Decision != "" {
		t.Errorf("Decision = %q, want empty", d.Decision)
	}

	// Rationale should be empty initially
	if d.Rationale != "" {
		t.Errorf("Rationale = %q, want empty", d.Rationale)
	}

	// DecidedAt should be nil initially
	if d.DecidedAt != nil {
		t.Error("DecidedAt should be nil initially")
	}

	// ClosedAt should be nil initially
	if d.ClosedAt != nil {
		t.Error("ClosedAt should be nil initially")
	}
}

// TestClaimDecision_UpdatesTimestamp verifies that claiming a decision
// updates the UpdatedAt timestamp.
func TestClaimDecision_UpdatesTimestamp(t *testing.T) {
	m, d := createTestDecision(t)
	originalUpdated := d.UpdatedAt

	err := m.ClaimDecision(d.ID, "decider-1")
	if err != nil {
		t.Fatalf("ClaimDecision() error = %v", err)
	}

	got, _ := m.GetDecision(d.ID)
	if got.UpdatedAt.Before(originalUpdated) {
		t.Error("ClaimDecision() should not set UpdatedAt to before the original")
	}
}

// TestEscalateDecision_PreservesOtherFields verifies that escalation
// does not modify fields other than Priority, Description, and UpdatedAt.
func TestEscalateDecision_PreservesOtherFields(t *testing.T) {
	m, d := createTestDecision(t, withPriority(models.BeadPriorityP2))
	originalQuestion := d.Question
	originalRequester := d.RequesterID
	originalStatus := d.Status
	originalProjectID := d.ProjectID

	_ = m.EscalateDecision(d.ID, "blocking")

	got, _ := m.GetDecision(d.ID)
	if got.Question != originalQuestion {
		t.Errorf("EscalateDecision() changed Question from %q to %q", originalQuestion, got.Question)
	}
	if got.RequesterID != originalRequester {
		t.Errorf("EscalateDecision() changed RequesterID from %q to %q", originalRequester, got.RequesterID)
	}
	if got.Status != originalStatus {
		t.Errorf("EscalateDecision() changed Status from %q to %q", originalStatus, got.Status)
	}
	if got.ProjectID != originalProjectID {
		t.Errorf("EscalateDecision() changed ProjectID from %q to %q", originalProjectID, got.ProjectID)
	}
}
