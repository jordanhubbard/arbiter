package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DistributedLock represents a distributed lock for coordination.
type DistributedLock struct {
	db         *Database
	lockName   string
	instanceID string
	ttl        time.Duration
	stopCh     chan struct{}
}

// AcquireLock attempts to acquire a distributed lock.
// Returns a lock object if successful, or an error if the lock is held.
func (d *Database) AcquireLock(ctx context.Context, lockName string, ttl time.Duration) (*DistributedLock, error) {
	if !d.supportsHA {
		return nil, fmt.Errorf("distributed locks require PostgreSQL")
	}

	instanceID := uuid.New().String()
	expiresAt := time.Now().Add(ttl)

	// Try to acquire lock
	query := `
		INSERT INTO distributed_locks (lock_name, instance_id, expires_at, heartbeat_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (lock_name) DO NOTHING
	`

	result, err := d.db.ExecContext(ctx, query, lockName, instanceID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check lock acquisition: %w", err)
	}

	if rows == 0 {
		// Lock is held by another instance
		// Check if it's expired
		var currentExpiry time.Time
		err := d.db.QueryRowContext(ctx, "SELECT expires_at FROM distributed_locks WHERE lock_name = $1", lockName).Scan(&currentExpiry)
		if err != nil {
			return nil, fmt.Errorf("lock held by another instance")
		}

		if time.Now().After(currentExpiry) {
			// Lock expired - try to steal it
			query = `
				UPDATE distributed_locks
				SET instance_id = $1, expires_at = $2, heartbeat_at = CURRENT_TIMESTAMP, acquired_at = CURRENT_TIMESTAMP
				WHERE lock_name = $3 AND expires_at < CURRENT_TIMESTAMP
			`
			result, err = d.db.ExecContext(ctx, query, instanceID, expiresAt, lockName)
			if err != nil {
				return nil, fmt.Errorf("failed to steal expired lock: %w", err)
			}

			rows, _ = result.RowsAffected()
			if rows == 0 {
				return nil, fmt.Errorf("lock held by another instance")
			}
		} else {
			return nil, fmt.Errorf("lock held by another instance")
		}
	}

	lock := &DistributedLock{
		db:         d,
		lockName:   lockName,
		instanceID: instanceID,
		ttl:        ttl,
		stopCh:     make(chan struct{}),
	}

	// Start heartbeat goroutine
	go lock.heartbeat()

	return lock, nil
}

// heartbeat periodically refreshes the lock to prevent expiration.
func (dl *DistributedLock) heartbeat() {
	ticker := time.NewTicker(dl.ttl / 3) // Heartbeat at 1/3 of TTL
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			expiresAt := time.Now().Add(dl.ttl)

			query := `
				UPDATE distributed_locks
				SET heartbeat_at = CURRENT_TIMESTAMP, expires_at = $1
				WHERE lock_name = $2 AND instance_id = $3
			`
			_, err := dl.db.db.ExecContext(ctx, query, expiresAt, dl.lockName, dl.instanceID)
			cancel()

			if err != nil {
				// Lost lock - stop heartbeat
				return
			}

		case <-dl.stopCh:
			return
		}
	}
}

// Release releases the distributed lock.
func (dl *DistributedLock) Release(ctx context.Context) error {
	close(dl.stopCh)

	query := `
		DELETE FROM distributed_locks
		WHERE lock_name = $1 AND instance_id = $2
	`

	_, err := dl.db.db.ExecContext(ctx, query, dl.lockName, dl.instanceID)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}

// Instance represents an AgentiCorp instance in the cluster.
type Instance struct {
	InstanceID    string
	Hostname      string
	StartedAt     time.Time
	LastHeartbeat time.Time
	Status        string
	Metadata      map[string]interface{}
}

// RegisterInstance registers this instance in the database.
func (d *Database) RegisterInstance(ctx context.Context, hostname string, metadata map[string]interface{}) (string, error) {
	if !d.supportsHA {
		return "", nil // Non-HA databases don't track instances
	}

	instanceID := uuid.New().String()

	metadataJSON := "{}"
	if metadata != nil {
		// Simple JSON encoding - in production use proper JSON marshaling
		metadataJSON = "{}"
	}

	query := `
		INSERT INTO instances (instance_id, hostname, metadata, started_at, last_heartbeat, status)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'active')
	`

	_, err := d.db.ExecContext(ctx, query, instanceID, hostname, metadataJSON)
	if err != nil {
		return "", fmt.Errorf("failed to register instance: %w", err)
	}

	return instanceID, nil
}

// HeartbeatInstance updates the instance heartbeat.
func (d *Database) HeartbeatInstance(ctx context.Context, instanceID string) error {
	if !d.supportsHA {
		return nil
	}

	query := `
		UPDATE instances
		SET last_heartbeat = CURRENT_TIMESTAMP
		WHERE instance_id = $1
	`

	result, err := d.db.ExecContext(ctx, query, instanceID)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("instance not found")
	}

	return nil
}

// UnregisterInstance removes an instance from the registry.
func (d *Database) UnregisterInstance(ctx context.Context, instanceID string) error {
	if !d.supportsHA {
		return nil
	}

	query := `
		DELETE FROM instances
		WHERE instance_id = $1
	`

	_, err := d.db.ExecContext(ctx, query, instanceID)
	return err
}

// ListActiveInstances returns all active instances.
func (d *Database) ListActiveInstances(ctx context.Context) ([]*Instance, error) {
	if !d.supportsHA {
		return nil, nil
	}

	// Consider instances active if heartbeat within last 60 seconds
	query := `
		SELECT instance_id, hostname, started_at, last_heartbeat, status
		FROM instances
		WHERE last_heartbeat > (CURRENT_TIMESTAMP - INTERVAL '60 seconds')
		ORDER BY started_at
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query instances: %w", err)
	}
	defer rows.Close()

	var instances []*Instance
	for rows.Next() {
		var inst Instance
		if err := rows.Scan(&inst.InstanceID, &inst.Hostname, &inst.StartedAt, &inst.LastHeartbeat, &inst.Status); err != nil {
			return nil, fmt.Errorf("failed to scan instance: %w", err)
		}
		instances = append(instances, &inst)
	}

	return instances, nil
}

// CleanupExpiredLocks removes expired locks from the database.
func (d *Database) CleanupExpiredLocks(ctx context.Context) (int, error) {
	if !d.supportsHA {
		return 0, nil
	}

	query := `
		DELETE FROM distributed_locks
		WHERE expires_at < CURRENT_TIMESTAMP
	`

	result, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup locks: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// CleanupStaleInstances removes stale instances from the registry.
func (d *Database) CleanupStaleInstances(ctx context.Context, timeout time.Duration) (int, error) {
	if !d.supportsHA {
		return 0, nil
	}

	query := `
		DELETE FROM instances
		WHERE last_heartbeat < $1
	`

	cutoff := time.Now().Add(-timeout)
	result, err := d.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup instances: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// WithTransaction executes a function within a database transaction.
func (d *Database) WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
