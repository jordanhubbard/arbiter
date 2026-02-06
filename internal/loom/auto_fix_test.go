package loom

import (
	"strings"
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

func TestExtractOriginalBugID(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expected    string
	}{
		{
			name: "standard format with bold",
			description: `## Code Fix Proposal

**Original Bug:** ac-001

### Root Cause
Something went wrong
`,
			expected: "ac-001",
		},
		{
			name: "format without bold",
			description: `## Code Fix Proposal

Original Bug: bd-build-failure-001

### Root Cause
Build failed
`,
			expected: "bd-build-failure-001",
		},
		{
			name: "format without colon space",
			description: `## Code Fix Proposal

**Original Bug:**ac-123

### Root Cause
Error
`,
			expected: "ac-123",
		},
		{
			name: "no bug ID",
			description: `## Code Fix Proposal

Something else entirely
`,
			expected: "",
		},
		{
			name: "bug ID with many dashes",
			description: `## Code Fix Proposal

**Original Bug:** bd-auto-filed-error-20260127001234

### Fix
Apply patch
`,
			expected: "bd-auto-filed-error-20260127001234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractOriginalBugID(tt.description)
			if result != tt.expected {
				t.Errorf("extractOriginalBugID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCreateApplyFixBead(t *testing.T) {
	// This is more of an integration test - we'll just verify the helper function
	// Full integration would require a complete Loom instance with database

	approvalBead := &models.Bead{
		ID:          "dc-approval-001",
		Title:       "[CEO] Code Fix Approval: Fix duplicate API_BASE",
		Description: "## Code Fix Proposal\n\n**Original Bug:** ac-js-error-001\n\n### Root Cause\nDuplicate declaration\n",
		Type:        "decision",
		ProjectID:   "loom-self",
		AssignedTo:  "agent-web-designer-001",
		Context: map[string]string{
			"agent_id": "agent-web-designer-001",
		},
	}

	// Test bug ID extraction
	bugID := extractOriginalBugID(approvalBead.Description)
	if bugID != "ac-js-error-001" {
		t.Errorf("Expected to extract bug ID 'ac-js-error-001', got '%s'", bugID)
	}

	// Test that approval bead detection works
	isCodeFixApproval := strings.Contains(strings.ToLower(approvalBead.Title), "code fix approval") &&
		approvalBead.Type == "decision"

	if !isCodeFixApproval {
		t.Error("Failed to detect code fix approval bead")
	}
}

func TestApplyFixTriggerConditions(t *testing.T) {
	tests := []struct {
		name          string
		beadTitle     string
		beadType      string
		closeReason   string
		shouldTrigger bool
	}{
		{
			name:          "approved code fix",
			beadTitle:     "[CEO] Code Fix Approval: Fix bug",
			beadType:      "decision",
			closeReason:   "Approved. Apply the fix.",
			shouldTrigger: true,
		},
		{
			name:          "approved with lowercase",
			beadTitle:     "[ceo] code fix approval: another fix",
			beadType:      "decision",
			closeReason:   "approved - looks good",
			shouldTrigger: true,
		},
		{
			name:          "rejected code fix",
			beadTitle:     "[CEO] Code Fix Approval: Fix bug",
			beadType:      "decision",
			closeReason:   "Rejected. Needs more work.",
			shouldTrigger: false,
		},
		{
			name:          "not a decision bead",
			beadTitle:     "[CEO] Code Fix Approval: Fix bug",
			beadType:      "task",
			closeReason:   "Approved",
			shouldTrigger: false,
		},
		{
			name:          "not a code fix approval",
			beadTitle:     "[CEO] Strategic Review",
			beadType:      "decision",
			closeReason:   "Approved",
			shouldTrigger: false,
		},
		{
			name:          "closed without approval",
			beadTitle:     "[CEO] Code Fix Approval: Fix bug",
			beadType:      "decision",
			closeReason:   "Closing for other reason",
			shouldTrigger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldTrigger := strings.Contains(strings.ToLower(tt.beadTitle), "code fix approval") &&
				tt.beadType == "decision" &&
				strings.Contains(strings.ToLower(tt.closeReason), "approve")

			if shouldTrigger != tt.shouldTrigger {
				t.Errorf("Expected shouldTrigger=%v, got %v", tt.shouldTrigger, shouldTrigger)
			}
		})
	}
}
