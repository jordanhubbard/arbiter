package activities

import (
	"context"

	"github.com/jordanhubbard/loom/internal/database"
)

// LoomActivities supplies activities for the Loom heartbeat
type LoomActivities struct {
	database *database.Database
}

func NewLoomActivities(db *database.Database) *LoomActivities {
	return &LoomActivities{database: db}
}

// LoomHeartbeatActivity is the master clock activity
// It runs on every heartbeat to check if we should dispatch work or run idle tasks
func (a *LoomActivities) LoomHeartbeatActivity(ctx context.Context, beatCount int) error {
	// This is a placeholder activity that just logs the heartbeat
	// The real work dispatch happens via the dispatcher workflow
	// which is triggered separately during initialization
	if beatCount%10 == 0 {
		// Log every 10 beats (100 seconds at 10s interval)
		_ = ctx // Use ctx to satisfy linter
	}
	return nil
}
