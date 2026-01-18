package models

import "time"

// Agent represents an LLM wrapped in glue code that interacts with providers
type Agent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ProviderID  string    `json:"provider_id"` // Reference to the provider this agent uses
	Status      string    `json:"status"`      // active, inactive, etc.
	Config      string    `json:"config"`      // JSON configuration for the agent
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
