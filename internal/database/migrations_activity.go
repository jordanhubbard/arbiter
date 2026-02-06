package database

import (
	"log"
)

// migrateActivity creates the activity feed and notifications tables
func (d *Database) migrateActivity() error {
	// Users table (persist users to database)
	usersSchema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		email TEXT,
		role TEXT NOT NULL,
		is_active BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
	`

	if _, err := d.db.Exec(usersSchema); err != nil {
		return err
	}

	// Activity feed table
	activityFeedSchema := `
	CREATE TABLE IF NOT EXISTS activity_feed (
		id TEXT PRIMARY KEY,
		event_type TEXT NOT NULL,
		event_id TEXT,
		timestamp DATETIME NOT NULL,
		source TEXT NOT NULL,
		actor_id TEXT,
		actor_type TEXT,
		project_id TEXT,
		agent_id TEXT,
		bead_id TEXT,
		provider_id TEXT,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_id TEXT NOT NULL,
		resource_title TEXT,
		metadata_json TEXT,
		aggregation_key TEXT,
		aggregation_count INTEGER DEFAULT 1,
		is_aggregated BOOLEAN DEFAULT 0,
		visibility TEXT NOT NULL DEFAULT 'project',
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
		FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE SET NULL
	);

	CREATE INDEX IF NOT EXISTS idx_activity_feed_timestamp ON activity_feed(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_activity_feed_project_id ON activity_feed(project_id);
	CREATE INDEX IF NOT EXISTS idx_activity_feed_actor_id ON activity_feed(actor_id);
	CREATE INDEX IF NOT EXISTS idx_activity_feed_event_type ON activity_feed(event_type);
	CREATE INDEX IF NOT EXISTS idx_activity_feed_aggregation ON activity_feed(aggregation_key, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_activity_feed_resource_type ON activity_feed(resource_type);
	`

	if _, err := d.db.Exec(activityFeedSchema); err != nil {
		return err
	}

	// Notifications table
	notificationsSchema := `
	CREATE TABLE IF NOT EXISTS notifications (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		activity_id TEXT,
		event_type TEXT NOT NULL,
		title TEXT NOT NULL,
		message TEXT NOT NULL,
		link TEXT,
		status TEXT NOT NULL DEFAULT 'unread',
		priority TEXT NOT NULL DEFAULT 'normal',
		metadata_json TEXT,
		created_at DATETIME NOT NULL,
		read_at DATETIME,
		archived_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (activity_id) REFERENCES activity_feed(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
	CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
	CREATE INDEX IF NOT EXISTS idx_notifications_user_status ON notifications(user_id, status, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);
	`

	if _, err := d.db.Exec(notificationsSchema); err != nil {
		return err
	}

	// Notification preferences table
	preferencesSchema := `
	CREATE TABLE IF NOT EXISTS notification_preferences (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL UNIQUE,
		enable_in_app BOOLEAN NOT NULL DEFAULT 1,
		enable_email BOOLEAN NOT NULL DEFAULT 0,
		enable_webhook BOOLEAN NOT NULL DEFAULT 0,
		subscribed_events_json TEXT,
		digest_mode TEXT DEFAULT 'realtime',
		quiet_hours_start TIME,
		quiet_hours_end TIME,
		project_filters_json TEXT,
		min_priority TEXT DEFAULT 'normal',
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_notification_preferences_user_id ON notification_preferences(user_id);
	`

	if _, err := d.db.Exec(preferencesSchema); err != nil {
		return err
	}

	// Migrate default admin user if not exists
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err == nil && count == 0 {
		// Create default admin user
		_, _ = d.db.Exec(`
			INSERT INTO users (id, username, email, role, is_active, created_at, updated_at)
			VALUES ('user-admin', 'admin', 'admin@loom.local', 'admin', 1, datetime('now'), datetime('now'))
		`)
		log.Println("Default admin user created in database")
	}

	log.Println("Activity and notification tables migrated successfully")
	return nil
}
