package actions

import (
	"context"
	"fmt"
	"sync"

	"github.com/jordanhubbard/loom/internal/gitops"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const projectIDKey contextKey = "projectID"

// WithProjectID returns a context with the project ID set.
func WithProjectID(ctx context.Context, projectID string) context.Context {
	return context.WithValue(ctx, projectIDKey, projectID)
}

// ProjectIDFromContext extracts the project ID from context.
func ProjectIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(projectIDKey).(string); ok {
		return v
	}
	return ""
}

// ProjectGitRouter implements GitOperator by routing each call through a
// per-project GitServiceAdapter. It uses the gitops.Manager to resolve
// project work directories and SSH key locations.
type ProjectGitRouter struct {
	gitopsMgr *gitops.Manager
	mu        sync.RWMutex
	cache     map[string]*GitServiceAdapter // projectID -> adapter
}

// NewProjectGitRouter creates a project-aware GitOperator.
func NewProjectGitRouter(gitopsMgr *gitops.Manager) *ProjectGitRouter {
	return &ProjectGitRouter{
		gitopsMgr: gitopsMgr,
		cache:     make(map[string]*GitServiceAdapter),
	}
}

// forProject returns a cached or newly-created GitServiceAdapter for the project.
func (r *ProjectGitRouter) forProject(projectID string) (*GitServiceAdapter, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID is required for git operations")
	}

	r.mu.RLock()
	if adapter, ok := r.cache[projectID]; ok {
		r.mu.RUnlock()
		return adapter, nil
	}
	r.mu.RUnlock()

	workDir := r.gitopsMgr.GetProjectWorkDir(projectID)
	keyDir := r.gitopsMgr.GetProjectKeyDir()

	adapter, err := NewGitServiceAdapter(workDir, projectID, keyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create git adapter for project %s: %w", projectID, err)
	}

	r.mu.Lock()
	r.cache[projectID] = adapter
	r.mu.Unlock()

	return adapter, nil
}

// resolve gets the project-scoped adapter from context or returns an error.
func (r *ProjectGitRouter) resolve(ctx context.Context) (*GitServiceAdapter, error) {
	projectID := ProjectIDFromContext(ctx)
	if projectID == "" {
		return nil, fmt.Errorf("no project ID in context — git operations require project context")
	}
	return r.forProject(projectID)
}

// --- GitOperator interface implementation ---

func (r *ProjectGitRouter) Status(ctx context.Context, projectID string) (string, error) {
	return r.gitopsMgr.Status(ctx, projectID)
}

func (r *ProjectGitRouter) Diff(ctx context.Context, projectID string) (string, error) {
	return r.gitopsMgr.Diff(ctx, projectID)
}

func (r *ProjectGitRouter) CreateBranch(ctx context.Context, beadID, description, baseBranch string) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.CreateBranch(ctx, beadID, description, baseBranch)
}

func (r *ProjectGitRouter) Commit(ctx context.Context, beadID, agentID, message string, files []string, allowAll bool) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.Commit(ctx, beadID, agentID, message, files, allowAll)
}

func (r *ProjectGitRouter) Push(ctx context.Context, beadID, branch string, setUpstream bool) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.Push(ctx, beadID, branch, setUpstream)
}

func (r *ProjectGitRouter) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetStatus(ctx)
}

func (r *ProjectGitRouter) GetDiff(ctx context.Context, staged bool) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetDiff(ctx, staged)
}

func (r *ProjectGitRouter) CreatePR(ctx context.Context, beadID, title, body, base, branch string, reviewers []string, draft bool) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.CreatePR(ctx, beadID, title, body, base, branch, reviewers, draft)
}

func (r *ProjectGitRouter) Merge(ctx context.Context, beadID, sourceBranch, message string, noFF bool) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.Merge(ctx, beadID, sourceBranch, message, noFF)
}

func (r *ProjectGitRouter) Revert(ctx context.Context, beadID string, commitSHAs []string, reason string) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.Revert(ctx, beadID, commitSHAs, reason)
}

func (r *ProjectGitRouter) DeleteBranch(ctx context.Context, branch string, deleteRemote bool) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.DeleteBranch(ctx, branch, deleteRemote)
}

func (r *ProjectGitRouter) Checkout(ctx context.Context, branch string) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.Checkout(ctx, branch)
}

func (r *ProjectGitRouter) Log(ctx context.Context, branch string, maxCount int) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.Log(ctx, branch, maxCount)
}

func (r *ProjectGitRouter) Fetch(ctx context.Context) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.Fetch(ctx)
}

func (r *ProjectGitRouter) ListBranches(ctx context.Context) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.ListBranches(ctx)
}

func (r *ProjectGitRouter) DiffBranches(ctx context.Context, branch1, branch2 string) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.DiffBranches(ctx, branch1, branch2)
}

func (r *ProjectGitRouter) GetBeadCommits(ctx context.Context, beadID string) (map[string]interface{}, error) {
	adapter, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetBeadCommits(ctx, beadID)
}

// ForProject returns a project-scoped GitOperator.
func (r *ProjectGitRouter) ForProject(projectID string) (GitOperator, error) {
	return r.forProject(projectID)
}
