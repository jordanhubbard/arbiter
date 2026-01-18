package arbiter

import (
	"fmt"
	"sync"
	"time"

	"github.com/jordanhubbard/arbiter/pkg/models"
)

// FileLockManager manages file locks to prevent merge conflicts
type FileLockManager struct {
	locks   map[string]*models.FileLock // key: projectID:filePath
	mu      sync.RWMutex
	timeout time.Duration
}

// NewFileLockManager creates a new file lock manager
func NewFileLockManager(timeout time.Duration) *FileLockManager {
	return &FileLockManager{
		locks:   make(map[string]*models.FileLock),
		timeout: timeout,
	}
}

// lockKey generates a unique key for a file lock
func (m *FileLockManager) lockKey(projectID, filePath string) string {
	return fmt.Sprintf("%s:%s", projectID, filePath)
}

// AcquireLock attempts to acquire a lock on a file
func (m *FileLockManager) AcquireLock(projectID, filePath, agentID, beadID string) (*models.FileLock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.lockKey(projectID, filePath)

	// Check if already locked
	if lock, exists := m.locks[key]; exists {
		// Check if lock has expired
		if lock.ExpiresAt.IsZero() || time.Now().Before(lock.ExpiresAt) {
			return nil, fmt.Errorf("file already locked by agent %s", lock.AgentID)
		}
		// Lock expired, can proceed
		delete(m.locks, key)
	}

	// Create new lock
	lock := &models.FileLock{
		FilePath:  filePath,
		ProjectID: projectID,
		AgentID:   agentID,
		BeadID:    beadID,
		LockedAt:  time.Now(),
		ExpiresAt: time.Now().Add(m.timeout),
	}

	m.locks[key] = lock

	return lock, nil
}

// ReleaseLock releases a file lock
func (m *FileLockManager) ReleaseLock(projectID, filePath, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.lockKey(projectID, filePath)

	lock, exists := m.locks[key]
	if !exists {
		return fmt.Errorf("no lock found for file: %s", filePath)
	}

	// Verify the agent releasing the lock is the one that acquired it
	if lock.AgentID != agentID {
		return fmt.Errorf("agent %s cannot release lock held by agent %s", agentID, lock.AgentID)
	}

	delete(m.locks, key)

	return nil
}

// IsLocked checks if a file is currently locked
func (m *FileLockManager) IsLocked(projectID, filePath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.lockKey(projectID, filePath)
	lock, exists := m.locks[key]

	if !exists {
		return false
	}

	// Check if expired
	if !lock.ExpiresAt.IsZero() && time.Now().After(lock.ExpiresAt) {
		return false
	}

	return true
}

// GetLock retrieves a file lock
func (m *FileLockManager) GetLock(projectID, filePath string) (*models.FileLock, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.lockKey(projectID, filePath)
	lock, exists := m.locks[key]

	if !exists {
		return nil, fmt.Errorf("no lock found for file: %s", filePath)
	}

	// Check if expired
	if !lock.ExpiresAt.IsZero() && time.Now().After(lock.ExpiresAt) {
		return nil, fmt.Errorf("lock expired for file: %s", filePath)
	}

	return lock, nil
}

// ListLocks returns all active locks
func (m *FileLockManager) ListLocks() []*models.FileLock {
	m.mu.RLock()
	defer m.mu.RUnlock()

	locks := make([]*models.FileLock, 0, len(m.locks))
	now := time.Now()

	for _, lock := range m.locks {
		// Skip expired locks
		if !lock.ExpiresAt.IsZero() && now.After(lock.ExpiresAt) {
			continue
		}
		locks = append(locks, lock)
	}

	return locks
}

// ListLocksByProject returns locks for a specific project
func (m *FileLockManager) ListLocksByProject(projectID string) []*models.FileLock {
	m.mu.RLock()
	defer m.mu.RUnlock()

	locks := make([]*models.FileLock, 0)
	now := time.Now()

	for _, lock := range m.locks {
		if lock.ProjectID == projectID {
			// Skip expired locks
			if !lock.ExpiresAt.IsZero() && now.After(lock.ExpiresAt) {
				continue
			}
			locks = append(locks, lock)
		}
	}

	return locks
}

// ListLocksByAgent returns locks held by a specific agent
func (m *FileLockManager) ListLocksByAgent(agentID string) []*models.FileLock {
	m.mu.RLock()
	defer m.mu.RUnlock()

	locks := make([]*models.FileLock, 0)
	now := time.Now()

	for _, lock := range m.locks {
		if lock.AgentID == agentID {
			// Skip expired locks
			if !lock.ExpiresAt.IsZero() && now.After(lock.ExpiresAt) {
				continue
			}
			locks = append(locks, lock)
		}
	}

	return locks
}

// ReleaseAgentLocks releases all locks held by an agent
func (m *FileLockManager) ReleaseAgentLocks(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	keysToDelete := make([]string, 0)

	for key, lock := range m.locks {
		if lock.AgentID == agentID {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(m.locks, key)
	}

	return nil
}

// CleanExpiredLocks removes expired locks
func (m *FileLockManager) CleanExpiredLocks() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	keysToDelete := make([]string, 0)
	now := time.Now()

	for key, lock := range m.locks {
		if !lock.ExpiresAt.IsZero() && now.After(lock.ExpiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(m.locks, key)
	}

	return len(keysToDelete)
}

// ExtendLock extends the expiration time of a lock
func (m *FileLockManager) ExtendLock(projectID, filePath, agentID string, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.lockKey(projectID, filePath)
	lock, exists := m.locks[key]

	if !exists {
		return fmt.Errorf("no lock found for file: %s", filePath)
	}

	if lock.AgentID != agentID {
		return fmt.Errorf("agent %s cannot extend lock held by agent %s", agentID, lock.AgentID)
	}

	lock.ExpiresAt = time.Now().Add(duration)

	return nil
}
