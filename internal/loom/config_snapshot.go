package loom

import (
	"context"
	"encoding/json"
	"fmt"

	internalmodels "github.com/jordanhubbard/loom/internal/models"
	"github.com/jordanhubbard/loom/internal/provider"
	"github.com/jordanhubbard/loom/pkg/config"
	"github.com/jordanhubbard/loom/pkg/models"
	"gopkg.in/yaml.v3"
)

// ConfigSnapshot is the full, import/exportable configuration for Loom.
type ConfigSnapshot struct {
	Server           config.ServerConfig        `json:"server" yaml:"server"`
	Database         config.DatabaseConfig      `json:"database" yaml:"database"`
	Beads            config.BeadsConfig         `json:"beads" yaml:"beads"`
	Agents           config.AgentsConfig        `json:"agents" yaml:"agents"`
	Security         config.SecurityConfig      `json:"security" yaml:"security"`
	WebUI            config.WebUIConfig         `json:"web_ui" yaml:"web_ui"`
	Temporal         config.TemporalConfig      `json:"temporal" yaml:"temporal"`
	Projects         []*models.Project          `json:"projects" yaml:"projects"`
	Providers        []*internalmodels.Provider `json:"providers" yaml:"providers"`
	AgentAssignments []*models.Agent            `json:"agent_assignments" yaml:"agent_assignments"`
	ModelCatalog     []internalmodels.ModelSpec `json:"model_catalog" yaml:"model_catalog"`
}

func (a *Loom) GetConfigSnapshot(ctx context.Context) (*ConfigSnapshot, error) {
	// Base config: default to current runtime config.
	cfg := a.config
	if a.database != nil {
		if raw, ok, err := a.database.GetConfigValue(configKVKey); err == nil && ok {
			var stored config.Config
			if err := json.Unmarshal([]byte(raw), &stored); err == nil {
				cfg = &stored
			}
		}
	}

	snap := &ConfigSnapshot{
		Server:   cfg.Server,
		Database: cfg.Database,
		Beads:    cfg.Beads,
		Agents:   cfg.Agents,
		Security: cfg.Security,
		WebUI:    cfg.WebUI,
		Temporal: cfg.Temporal,
	}

	if a.modelCatalog != nil {
		snap.ModelCatalog = a.modelCatalog.List()
	}

	if a.database != nil {
		projects, err := a.database.ListProjects()
		if err != nil {
			return nil, err
		}
		providers, err := a.database.ListProviders()
		if err != nil {
			return nil, err
		}
		agents, err := a.database.ListAgents()
		if err != nil {
			return nil, err
		}
		snap.Projects = projects
		snap.Providers = providers
		snap.AgentAssignments = agents
	} else {
		snap.Projects = a.projectManager.ListProjects()
		snap.AgentAssignments = a.agentManager.ListAgents()
		snap.Providers = []*internalmodels.Provider{}
	}

	return snap, nil
}

func (a *Loom) ExportConfigSnapshotYAML(ctx context.Context) ([]byte, error) {
	snap, err := a.GetConfigSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(snap)
}

func (a *Loom) ImportConfigSnapshotYAML(ctx context.Context, data []byte) (*ConfigSnapshot, error) {
	var snap ConfigSnapshot
	if err := yaml.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}
	if err := a.ApplyConfigSnapshot(ctx, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

func (a *Loom) ApplyConfigSnapshot(ctx context.Context, snap *ConfigSnapshot) error {
	if snap == nil {
		return fmt.Errorf("snapshot cannot be nil")
	}
	if a.database == nil {
		return fmt.Errorf("database not configured")
	}

	// Store global config values.
	stored := config.Config{
		Server:   snap.Server,
		Database: snap.Database,
		Beads:    snap.Beads,
		Agents:   snap.Agents,
		Security: snap.Security,
		WebUI:    snap.WebUI,
		Temporal: snap.Temporal,
	}
	raw, err := json.Marshal(&stored)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := a.database.SetConfigValue(configKVKey, string(raw)); err != nil {
		return err
	}

	// Replace normalized configuration tables.
	if err := a.database.DeleteAllAgents(); err != nil {
		return err
	}
	if err := a.database.DeleteAllProviders(); err != nil {
		return err
	}
	if err := a.database.DeleteAllProjects(); err != nil {
		return err
	}

	for _, p := range snap.Projects {
		if p == nil {
			continue
		}
		if err := a.database.UpsertProject(p); err != nil {
			return err
		}
	}

	for _, p := range snap.Providers {
		if p == nil {
			continue
		}
		p.Endpoint = normalizeProviderEndpoint(p.Endpoint)
		if err := a.database.UpsertProvider(p); err != nil {
			return err
		}
	}

	for _, ag := range snap.AgentAssignments {
		if ag == nil {
			continue
		}
		if err := a.database.UpsertAgent(ag); err != nil {
			return err
		}
	}

	if len(snap.ModelCatalog) > 0 {
		rawModels, err := json.Marshal(snap.ModelCatalog)
		if err != nil {
			return fmt.Errorf("failed to marshal model catalog: %w", err)
		}
		if err := a.database.SetConfigValue(modelCatalogKey, string(rawModels)); err != nil {
			return err
		}
		if a.modelCatalog != nil {
			a.modelCatalog.Replace(snap.ModelCatalog)
		}
	}

	// Apply runtime config in-memory (best-effort).
	a.config.Server = snap.Server
	a.config.Database = snap.Database
	a.config.Beads = snap.Beads
	a.config.Agents = snap.Agents
	a.config.Security = snap.Security
	a.config.WebUI = snap.WebUI
	a.config.Temporal = snap.Temporal

	// Reload in-memory managers from the DB for immediate visibility in UI.
	return a.ReloadFromDatabase(ctx)
}

func (a *Loom) ReloadFromDatabase(ctx context.Context) error {
	if a.database == nil {
		return nil
	}

	a.agentManager.StopAll()
	a.providerRegistry.Clear()
	a.projectManager.Clear()
	a.beadsManager.Reset()

	projects, err := a.database.ListProjects()
	if err != nil {
		return err
	}
	var projectValues []models.Project
	for _, p := range projects {
		if p == nil {
			continue
		}
		projectValues = append(projectValues, *p)
	}
	if err := a.projectManager.LoadProjects(projectValues); err != nil {
		return err
	}

	for _, p := range projectValues {
		if p.BeadsPath == "" {
			continue
		}
		a.beadsManager.SetBeadsPath(p.BeadsPath)
		// Load project prefix from config
		_ = a.beadsManager.LoadProjectPrefixFromConfig(p.ID, p.BeadsPath)
		// Use project's BeadPrefix if set in the model
		if p.BeadPrefix != "" {
			a.beadsManager.SetProjectPrefix(p.ID, p.BeadPrefix)
		}
		_ = a.beadsManager.LoadBeadsFromFilesystem(p.ID, p.BeadsPath)
	}

	providers, err := a.database.ListProviders()
	if err != nil {
		return err
	}
	for _, p := range providers {
		if p == nil {
			continue
		}
		_ = a.providerRegistry.Register(&provider.ProviderConfig{
			ID:       p.ID,
			Name:     p.Name,
			Type:     p.Type,
			Endpoint: normalizeProviderEndpoint(p.Endpoint),
			APIKey:   "",
			Model:    p.Model,
		})
	}

	agents, err := a.database.ListAgents()
	if err != nil {
		return err
	}
	for _, ag := range agents {
		if ag == nil {
			continue
		}
		persona, err := a.personaManager.LoadPersona(ag.PersonaName)
		if err != nil {
			continue
		}
		ag.Persona = persona
		if ag.ProviderID == "" {
			list := a.providerRegistry.List()
			if len(list) == 0 {
				continue
			}
			ag.ProviderID = list[0].Config.ID
		}
		_, _ = a.agentManager.RestoreAgentWorker(ctx, ag)
		_ = a.projectManager.AddAgentToProject(ag.ProjectID, ag.ID)
	}

	return nil
}
