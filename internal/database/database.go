package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/jordanhubbard/arbiter/internal/models"
)

// Database represents the arbiter database
type Database struct {
	db *sql.DB
}

// New creates a new database instance and initializes the schema
func New(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	d := &Database{db: db}

	// Initialize schema
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return d, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// initSchema creates the database tables
func (d *Database) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS providers (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		endpoint TEXT NOT NULL,
		description TEXT,
		requires_key BOOLEAN NOT NULL DEFAULT 0,
		key_id TEXT,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		provider_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		config TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_agents_provider_id ON agents(provider_id);
	CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
	CREATE INDEX IF NOT EXISTS idx_providers_status ON providers(status);
	`

	if _, err := d.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// CreateProvider creates a new provider
func (d *Database) CreateProvider(provider *models.Provider) error {
	provider.CreatedAt = time.Now()
	provider.UpdatedAt = time.Now()

	query := `
		INSERT INTO providers (id, name, type, endpoint, description, requires_key, key_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		provider.ID,
		provider.Name,
		provider.Type,
		provider.Endpoint,
		provider.Description,
		provider.RequiresKey,
		provider.KeyID,
		provider.Status,
		provider.CreatedAt,
		provider.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	return nil
}

// GetProvider retrieves a provider by ID
func (d *Database) GetProvider(id string) (*models.Provider, error) {
	query := `
		SELECT id, name, type, endpoint, description, requires_key, key_id, status, created_at, updated_at
		FROM providers
		WHERE id = ?
	`

	provider := &models.Provider{}
	err := d.db.QueryRow(query, id).Scan(
		&provider.ID,
		&provider.Name,
		&provider.Type,
		&provider.Endpoint,
		&provider.Description,
		&provider.RequiresKey,
		&provider.KeyID,
		&provider.Status,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("provider not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return provider, nil
}

// ListProviders retrieves all providers
func (d *Database) ListProviders() ([]*models.Provider, error) {
	query := `
		SELECT id, name, type, endpoint, description, requires_key, key_id, status, created_at, updated_at
		FROM providers
		ORDER BY created_at DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	defer rows.Close()

	var providers []*models.Provider
	for rows.Next() {
		provider := &models.Provider{}
		err := rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.Type,
			&provider.Endpoint,
			&provider.Description,
			&provider.RequiresKey,
			&provider.KeyID,
			&provider.Status,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		providers = append(providers, provider)
	}

	return providers, nil
}

// UpdateProvider updates a provider
func (d *Database) UpdateProvider(provider *models.Provider) error {
	provider.UpdatedAt = time.Now()

	query := `
		UPDATE providers
		SET name = ?, type = ?, endpoint = ?, description = ?, requires_key = ?, key_id = ?, status = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := d.db.Exec(query,
		provider.Name,
		provider.Type,
		provider.Endpoint,
		provider.Description,
		provider.RequiresKey,
		provider.KeyID,
		provider.Status,
		provider.UpdatedAt,
		provider.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("provider not found: %s", provider.ID)
	}

	return nil
}

// DeleteProvider deletes a provider
func (d *Database) DeleteProvider(id string) error {
	query := `DELETE FROM providers WHERE id = ?`

	result, err := d.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("provider not found: %s", id)
	}

	return nil
}

// CreateAgent creates a new agent
func (d *Database) CreateAgent(agent *models.Agent) error {
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()

	query := `
		INSERT INTO agents (id, name, description, provider_id, status, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		agent.ID,
		agent.Name,
		agent.Description,
		agent.ProviderID,
		agent.Status,
		agent.Config,
		agent.CreatedAt,
		agent.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

// GetAgent retrieves an agent by ID
func (d *Database) GetAgent(id string) (*models.Agent, error) {
	query := `
		SELECT id, name, description, provider_id, status, config, created_at, updated_at
		FROM agents
		WHERE id = ?
	`

	agent := &models.Agent{}
	err := d.db.QueryRow(query, id).Scan(
		&agent.ID,
		&agent.Name,
		&agent.Description,
		&agent.ProviderID,
		&agent.Status,
		&agent.Config,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return agent, nil
}

// ListAgents retrieves all agents
func (d *Database) ListAgents() ([]*models.Agent, error) {
	query := `
		SELECT id, name, description, provider_id, status, config, created_at, updated_at
		FROM agents
		ORDER BY created_at DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		agent := &models.Agent{}
		err := rows.Scan(
			&agent.ID,
			&agent.Name,
			&agent.Description,
			&agent.ProviderID,
			&agent.Status,
			&agent.Config,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// ListAgentsByProvider retrieves all agents for a specific provider
func (d *Database) ListAgentsByProvider(providerID string) ([]*models.Agent, error) {
	query := `
		SELECT id, name, description, provider_id, status, config, created_at, updated_at
		FROM agents
		WHERE provider_id = ?
		ORDER BY created_at DESC
	`

	rows, err := d.db.Query(query, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by provider: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		agent := &models.Agent{}
		err := rows.Scan(
			&agent.ID,
			&agent.Name,
			&agent.Description,
			&agent.ProviderID,
			&agent.Status,
			&agent.Config,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// UpdateAgent updates an agent
func (d *Database) UpdateAgent(agent *models.Agent) error {
	agent.UpdatedAt = time.Now()

	query := `
		UPDATE agents
		SET name = ?, description = ?, provider_id = ?, status = ?, config = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := d.db.Exec(query,
		agent.Name,
		agent.Description,
		agent.ProviderID,
		agent.Status,
		agent.Config,
		agent.UpdatedAt,
		agent.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	return nil
}

// DeleteAgent deletes an agent
func (d *Database) DeleteAgent(id string) error {
	query := `DELETE FROM agents WHERE id = ?`

	result, err := d.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("agent not found: %s", id)
	}

	return nil
}
